package database

import (
	"bytes"
	"math"
	"math/rand/v2"
)

const (
	_DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT   = 32
	_DATABASE_MEMTABLE_SKIPLIST_MAX_SIZE     = 8 * 1024 * 1024 // 8 MiB
	_DATABASE_MEMTABLE_SKIPLIST_MAX_KEY_SIZE = 1 * 1024 * 1024 // 1 MiB
)

type skipNode struct {
	key   uint64
	value uint64

	level int
	next  [_DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT]uint32
}

type skipList struct {
	_r    rand.Source
	nodes []skipNode

	head    uint32
	tail    uint32
	deleted uint64

	minVersion uint64
	maxVersion uint64

	minKey uint64
	maxKey uint64

	b     []byte
	bHead uint32
}

// _bytes returns a byte slice from the memtable's byte buffer, given an offset and size.
// The offset and size are encoded in the provided uint64 value.
// If the offset or size are out of bounds for the byte buffer, this function returns nil.
func (g *skipList) _bytes(i uint64) []byte {
	offset := uint32(i >> 32)
	size := uint32(i)

	// Index check
	if offset >= uint32(len(g.b)) {
		return nil
	}

	// Size check
	if uint64(offset)+uint64(size) > uint64(len(g.b)) {
		return nil
	}

	return g.b[offset : offset+size]
}

// _allocate allocates a contiguous block of memory from the memtable's byte buffer.
// The size parameter specifies the size of the block to allocate, in bytes.
// The function returns a 64-bit value that encodes the offset and size of the allocated block.
// If the allocation fails due to insufficient space in the byte buffer, the function returns 0.
func (g *skipList) _allocate(size uint32) uint64 {
	offset := g.bHead
	alignedSize := ((size + 7) >> 3) << 3

	if int(offset) >= len(g.b) {
		return 0
	}

	if uint64(offset)+uint64(alignedSize) > uint64(len(g.b)) {
		return 0
	}

	g.bHead += alignedSize
	return uint64(offset)<<32 | uint64(size)
}

// _newNode creates a new skipNode with the given key, value, and level, and appends it to the g.nodes slice.
// It returns the index of the new node within the g.nodes slice.
func (g *skipList) _newNode(key, value uint64, level int) uint32 {
	g.nodes = append(g.nodes, skipNode{
		key:   key,
		value: value,
		level: level,
	})

	return uint32(len(g.nodes) - 1)
}

// _randomLevel generates a random level for a new skiplist node. It uses a geometric
// distribution with a success probability of 0.5 to determine the level, up to a
// maximum of _DATABASE_MEMTABLE_MAX_HEIGHT. This ensures that the skiplist has the
// desired logarithmic height distribution.
func (g *skipList) _randomLevel() int {
	var n int = 1
	for g._r.Uint64()%2 == 1 && n < _DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT {
		n++
	}
	return n
}

// newSkipList creates a new skipList instance. It initializes the skipList with a
// random number generator, sets the min and max version to the appropriate values,
// allocates a byte buffer for the skipList, and creates the head and tail nodes.
// The function returns a pointer to the new skipList instance.
func newSkipList() *skipList {
	g := &skipList{
		_r:         rand.NewPCG(rand.Uint64(), rand.Uint64()),
		minVersion: math.MaxUint64,
		maxVersion: 0,
		b:          make([]byte, _DATABASE_MEMTABLE_SKIPLIST_MAX_SIZE),
		bHead:      0,
	}

	// Allocate Null Memory
	null := g._allocate(8)
	if null == 0 {
		return nil
	}
	null = (null >> 32) << 32
	if null != 0 {
		return nil
	}
	g.deleted = null

	g.minKey = null
	g.maxKey = null

	// Allocate Tail Node: 0
	g.tail = g._newNode(null, null, 0)
	// Allocate Head Node: 1
	g.head = g._newNode(0, 0, _DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT)

	return g
}

// _seek performs a search operation on the skip list for the given key (bytes + version).
// It returns the index of the node that either contains the key, or is the
// first node whose key is greater than the given key.
//
// Optionally, a log parameter can be provided to track the path taken
// during the search. The log is an array of uint32 values, where each index
// corresponds to a level in the skip list. For each level, the log stores the
// index of the node that was visited at that level.
//
// This function uses a technique known as "level-by-level search," where it
// iterates through the levels of the skip list from highest to lowest.
// At each level, it traverses the linked list of nodes until it encounters
// a node whose key is greater than or equal to the search key.
//
// The log is updated during the search to provide a record of the path taken.
// This can be useful for debugging or for implementing other operations that
// require knowledge of the search path.
func (g *skipList) _seek(key []byte, log *[_DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT]uint32) uint32 {
	// If a log is provided, initialize it with the head node for all levels.
	if log != nil {
		for i := 0; i < _DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT; i++ {
			log[i] = g.head
		}
	}

	// Start at the head node.
	var node uint32 = g.head

	// Iterate through the levels of the skip list from highest to lowest.
	for i := _DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT - 1; i >= 0; i-- {
		// Traverse the linked list at the current level until we encounter a node
		// whose key is greater than or equal to the search key.
		for g.nodes[node].next[i] != g.tail && _CompareKey(g._bytes(g.nodes[g.nodes[node].next[i]].key), key) <= 0 {
			node = g.nodes[node].next[i]
		}

		// If a log is provided, update it with the current node index for the current level.
		if log != nil {
			log[i] = node
		}
	}

	// Return the index of the node that either contains the key, or is the
	// first node whose key is greater than the given key.
	return node
}

func (g *skipList) Lookup(key []byte) ([]byte, bool) {
	if len(key) < _VERSION_LEN {
		return nil, false
	}

	node := g._seek(key, nil)
	if node == g.tail {
		return nil, false
	}

	raw := _RawKey(g._bytes(g.nodes[node].key))
	if !bytes.Equal(raw, _RawKey(key)) {
		return nil, false
	}

	if g.nodes[node].value == g.deleted {
		return nil, false
	}

	return g._bytes(g.nodes[node].value), true
}

func (g *skipList) Insert(key []byte, value []byte) bool {
	if len(key) < _VERSION_LEN {
		return false
	}

	var log [_DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT]uint32

	node := g._seek(key, &log)
	if node == g.tail {
		return false
	}

	keyBuf := g._allocate(uint32(len(key)))
	valueBuf := g._allocate(uint32(len(value)))
	if keyBuf == 0 || valueBuf == 0 {
		return false
	}

	if _CompareKey(g._bytes(g.nodes[node].key), key) == 0 {
		g.nodes[node].key = keyBuf
		g.nodes[node].value = valueBuf
		return true
	}

	newNode := g._newNode(keyBuf, valueBuf, g._randomLevel())
	if newNode == 0 {
		return false // tail node accidentally allocated
	}

	for i := 0; i < g.nodes[newNode].level; i++ {
		g.nodes[newNode].next[i] = g.nodes[log[i]].next[i]
		g.nodes[log[i]].next[i] = newNode
	}

	if g.nodes[newNode].key < g.minKey {
		g.minKey = g.nodes[newNode].key
	}
	if g.nodes[newNode].key > g.maxKey {
		g.maxKey = g.nodes[newNode].key
	}

	if _Version(key) > g.maxVersion {
		g.maxVersion = _Version(key)
	}
	if _Version(key) < g.minVersion {
		g.minVersion = _Version(key)
	}

	return true
}

func (g *skipList) Delete(key []byte) bool {
	if len(key) < _VERSION_LEN {
		return false
	}

	var log [_DATABASE_MEMTABLE_SKIPLIST_MAX_HEIGHT]uint32

	node := g._seek(key, &log)
	if node == g.tail {
		return false
	}

	keyBuf := g._allocate(uint32(len(key)))
	if keyBuf == 0 {
		return false
	}

	if _CompareKey(g._bytes(g.nodes[node].key), key) == 0 {
		g.nodes[node].key = keyBuf
		g.nodes[node].value = g.deleted
		return true
	}

	newNode := g._newNode(keyBuf, g.deleted, g._randomLevel())
	if newNode == 0 {
		return false // tail node accidentally allocated
	}

	for i := 0; i < g.nodes[newNode].level; i++ {
		g.nodes[newNode].next[i] = g.nodes[log[i]].next[i]
		g.nodes[log[i]].next[i] = newNode
	}

	if g.nodes[newNode].key < g.minKey {
		g.minKey = g.nodes[newNode].key
	}
	if g.nodes[newNode].key > g.maxKey {
		g.maxKey = g.nodes[newNode].key
	}

	if _Version(key) > g.maxVersion {
		g.maxVersion = _Version(key)
	}
	if _Version(key) < g.minVersion {
		g.minVersion = _Version(key)
	}

	return true
}
