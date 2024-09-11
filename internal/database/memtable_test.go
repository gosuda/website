package database

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func createTestKey(key string, version uint64) []byte {
	versionBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(versionBytes, version)
	return append([]byte(key), versionBytes...)
}

func TestNewSkipList(t *testing.T) {
	sl := newSkipList()
	if sl == nil {
		t.Fatal("newSkipList returned nil")
	}
	if sl.minVersion != ^uint64(0) {
		t.Errorf("Expected minVersion to be MaxUint64, got %d", sl.minVersion)
	}
	if sl.maxVersion != 0 {
		t.Errorf("Expected maxVersion to be 0, got %d", sl.maxVersion)
	}
	if len(sl.b) != _DATABASE_MEMTABLE_SKIPLIST_MAX_SIZE {
		t.Errorf("Expected byte buffer size to be %d, got %d", _DATABASE_MEMTABLE_SKIPLIST_MAX_SIZE, len(sl.b))
	}
	if sl.bHead == 0 {
		t.Error("Expected bHead to be non-zero")
	}
}

func TestSkipListInsertLookupDelete(t *testing.T) {
	sl := newSkipList()

	// Test Insert
	key1 := createTestKey("key1", 1)
	value1 := []byte("value1")
	if !sl.Insert(key1, value1) {
		t.Fatal("Failed to insert key1")
	}

	// Test Lookup
	lookupValue, found := sl.Lookup(key1)
	if !found {
		t.Fatal("Failed to find key1")
	}
	if !bytes.Equal(lookupValue, value1) {
		t.Errorf("Expected value %s, got %s", value1, lookupValue)
	}

	// Test Delete
	if !sl.Delete(key1) {
		t.Fatal("Failed to delete key1")
	}

	// Verify deletion
	_, found = sl.Lookup(key1)
	if found {
		t.Fatal("key1 still exists after deletion")
	}
}

func TestSkipListMVCC(t *testing.T) {
	sl := newSkipList()

	baseKey := "mvccKey"
	value1 := []byte("value1")
	value2 := []byte("value2")

	// Insert version 1
	key1 := createTestKey(baseKey, 1)
	if !sl.Insert(key1, value1) {
		t.Fatal("Failed to insert key version 1")
	}

	// Insert version 2
	key2 := createTestKey(baseKey, 2)
	if !sl.Insert(key2, value2) {
		t.Fatal("Failed to insert key version 2")
	}

	// Lookup version 1
	lookupValue1, found := sl.Lookup(key1)
	if !found {
		t.Fatal("Failed to find key version 1")
	}
	if !bytes.Equal(lookupValue1, value1) {
		t.Errorf("Expected value %s for version 1, got %s", value1, lookupValue1)
	}

	// Lookup version 2
	lookupValue2, found := sl.Lookup(key2)
	if !found {
		t.Fatal("Failed to find key version 2")
	}
	if !bytes.Equal(lookupValue2, value2) {
		t.Errorf("Expected value %s for version 2, got %s", value2, lookupValue2)
	}

	// Lookup version 3
	key3 := createTestKey(baseKey, 3)
	lookupValue3, found := sl.Lookup(key3)
	if !found {
		t.Fatal("Failed to find key version 3")
	}
	if !bytes.Equal(lookupValue3, value2) {
		t.Errorf("Expected value %s for version 3, got %s", value2, lookupValue3)
	}

	// Verify minVersion and maxVersion
	if sl.minVersion != 1 {
		t.Errorf("Expected minVersion to be 1, got %d", sl.minVersion)
	}
	if sl.maxVersion != 2 {
		t.Errorf("Expected maxVersion to be 2, got %d", sl.maxVersion)
	}
}

func TestSkipListEdgeCases(t *testing.T) {
	sl := newSkipList()

	// Test inserting a key with insufficient version length
	shortKey := []byte("short")
	if sl.Insert(shortKey, []byte("value")) {
		t.Error("Inserted a key with insufficient version length")
	}

	// Test looking up a non-existent key
	nonExistentKey := createTestKey("nonexistent", 1)
	_, found := sl.Lookup(nonExistentKey)
	if found {
		t.Error("Found a non-existent key")
	}

	sl.Delete(nonExistentKey)

	// Test inserting maximum allowed key size
	maxKey := createTestKey(string(make([]byte, _DATABASE_MEMTABLE_SKIPLIST_MAX_KEY_SIZE-_VERSION_LEN)), 1)
	if !sl.Insert(maxKey, []byte("max_value")) {
		t.Error("Failed to insert maximum allowed key size")
	}

	// Test inserting key larger than maximum allowed size
	oversizeKey := createTestKey(string(make([]byte, _DATABASE_MEMTABLE_SKIPLIST_MAX_KEY_SIZE-_VERSION_LEN+1)), 1)
	if sl.Insert(oversizeKey, []byte("oversize_value")) {
		t.Error("Inserted a key larger than maximum allowed size")
	}
}

func TestSkipListMultipleInsertions(t *testing.T) {
	sl := newSkipList()

	for i := 0; i < 100; i++ {
		key := createTestKey(string([]byte{byte(i)}), uint64(i))
		value := []byte{byte(i)}
		if !sl.Insert(key, value) {
			t.Fatalf("Failed to insert key %d", i)
		}
	}

	for i := 0; i < 100; i++ {
		key := createTestKey(string([]byte{byte(i)}), uint64(i))
		value, found := sl.Lookup(key)
		if !found {
			t.Fatalf("Failed to find key %d", i)
		}
		if !bytes.Equal(value, []byte{byte(i)}) {
			t.Errorf("Incorrect value for key %d. Expected %v, got %v", i, []byte{byte(i)}, value)
		}
	}
}

func TestSkipListOverwrite(t *testing.T) {
	sl := newSkipList()

	key := createTestKey("overwrite", 1)
	value1 := []byte("value1")
	value2 := []byte("value2")

	if !sl.Insert(key, value1) {
		t.Fatal("Failed to insert initial value")
	}

	if !sl.Insert(key, value2) {
		t.Fatal("Failed to overwrite value")
	}

	lookupValue, found := sl.Lookup(key)
	if !found {
		t.Fatal("Failed to find overwritten key")
	}
	if !bytes.Equal(lookupValue, value2) {
		t.Errorf("Expected overwritten value %s, got %s", value2, lookupValue)
	}
}
