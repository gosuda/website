package database

import (
	"bytes"
	"encoding/binary"
)

const (
	_VERSION_LEN = 8
)

// _CompareKey compares two byte slices, first by their content up to the last _VERSION_LEN bytes,
// and then by the version encoded in the last _VERSION_LEN bytes. The comparison order for versions
// is reversed, so that higher versions are considered "smaller" than lower versions.
func _CompareKey(a, b []byte) int {
	if len(a) < _VERSION_LEN {
		return -1
	}
	if len(b) < _VERSION_LEN {
		return 1
	}

	cmp := bytes.Compare(a[:len(a)-_VERSION_LEN], b[:len(b)-_VERSION_LEN])
	if cmp != 0 {
		return cmp
	}

	aVersion := binary.BigEndian.Uint64(a[len(a)-_VERSION_LEN:])
	bVersion := binary.BigEndian.Uint64(b[len(b)-_VERSION_LEN:])

	// Reverse the comparison order when comparing versions
	if aVersion > bVersion {
		return -1
	} else if aVersion < bVersion {
		return 1
	}

	return 0
}

// _Version returns the version encoded in the last _VERSION_LEN bytes of the given key.
func _Version(key []byte) uint64 {
	if len(key) < _VERSION_LEN {
		return 0
	}
	return binary.BigEndian.Uint64(key[len(key)-_VERSION_LEN:])
}

// _RawKey returns the raw key without the version encoded in the last _VERSION_LEN bytes.
func _RawKey(key []byte) []byte {
	if len(key) < _VERSION_LEN {
		return key
	}
	return key[:len(key)-_VERSION_LEN]
}

func _KeyAt(key []byte, version uint64) []byte {
	if len(key) < _VERSION_LEN {
		return key
	}
	keyBuf := make([]byte, len(key)+_VERSION_LEN)
	copy(keyBuf, key)
	binary.BigEndian.PutUint64(keyBuf[len(key):], version)
	return keyBuf
}
