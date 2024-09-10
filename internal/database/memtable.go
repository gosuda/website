package database

import (
	"math"
	"math/rand/v2"
)

const (
	_DATABASE_MEMTABLE_MAX_HEIGHT   = 32
	_DATABASE_MEMTABLE_MAX_SIZE     = 8 * 1024 * 1024 // 8 MiB
	_DATABASE_MEMTABLE_MAX_KEY_SIZE = 1 * 1024 * 1024 // 1 MiB
)

type skipNode struct {
	key   uint64
	value uint64

	level int
	next  [_DATABASE_MEMTABLE_MAX_HEIGHT]uint32
}

type skipList struct {
	_r    rand.Source
	nodes []skipNode

	head uint32
	tail uint32

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
	size = ((size + 7) >> 3) << 3

	if int(offset) >= len(g.b) {
		return 0
	}

	if uint64(offset)+uint64(size) > uint64(len(g.b)) {
		return 0
	}

	g.bHead += size
	return uint64(offset)<<32 | uint64(size)
}

func (g *skipList) _newNode(key, value uint64, level int) uint32 {
	g.nodes = append(g.nodes, skipNode{
		key:   key,
		value: value,
		level: level,
	})

	return uint32(len(g.nodes) - 1)
}

func (g *skipList) _randomLevel() int {
	var n int = 1
	for g._r.Uint64()%2 == 1 && n < _DATABASE_MEMTABLE_MAX_HEIGHT {
		n++
	}
	return n
}

func newSkipList() *skipList {
	g := &skipList{
		_r:         rand.NewPCG(rand.Uint64(), rand.Uint64()),
		minVersion: math.MaxUint64,
		maxVersion: 0,
		b:          make([]byte, _DATABASE_MEMTABLE_MAX_SIZE),
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

	g.minKey = null
	g.maxKey = null

	// Allocate Tail Node: 0
	g.tail = g._newNode(null, null, 0)
	// Allocate Head Node: 1
	g.head = g._newNode(0, 0, _DATABASE_MEMTABLE_MAX_HEIGHT)

	return g
}
