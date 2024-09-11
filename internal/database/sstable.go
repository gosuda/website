package database

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"

	"gosuda.org/website/internal/wyhash"
)

const (
	_DATABASE_SSTABLE_MAGIC        = "f3db64e176e9b2f5"
	_DATABASE_SSTABLE_FOOTER_MAGIC = "cf56bff25a91312a"
	_DATABASE_SSTABLE_VERSION      = 10

	_DATABASE_SSTABLE_MAX_SIZE = 20 * 1024 * 1024 // 20MiB

)
const (
	_DATABASE_SSTABLE_HEADER_FLAG_RESERVED uint32 = 1 << iota
)

var (
	_MAGIC_BYTES, _        = hex.DecodeString(_DATABASE_SSTABLE_MAGIC)
	_FOOTER_MAGIC_BYTES, _ = hex.DecodeString(_DATABASE_SSTABLE_FOOTER_MAGIC)
)

// SSTable File Format
//
// The SSTable (Sorted String Table) is a file format used for storing key-value pairs. The format is structured as follows:
//
// 1. Header:
//    - Magic Number (8 bytes): Identifies the file as an SSTable
//    - Version (4 bytes): SSTable format version
//    - Flags (4 bytes): Reserved for future use
//    - Hash Seed (8 bytes): Seed for the WyHash checksum
//    - WyHash Checksum (8 bytes): Checksum of the above data
//
// 2. Data Blocks:
//    Multiple data blocks, each containing:
//    a. Flags (4 bytes): Flags for the block
//    b. Bloom Filter Size (4 bytes): Optional, 0 if not present
//    c. Bloom Filter (variable length): Optional
//    d. Key-Value Pairs:
//       - Key Length (4 bytes)
//       - Key (variable length)
//       - Version (8 bytes)
//       - Flags (4 bytes): Flags for the key-value pair
//       - Value Length (4 bytes)
//       - Value (variable length)
//    e. WyHash Checksum (8 bytes): Checksum of a || b || c || d
//
// 3. Index Block:
//    - Number of Index Entries (4 bytes)
//    - Index Entries:
//      a. Key Length (4 bytes)
//      b. Key (variable length)
//      c. Version (8 bytes)
//      d. Offset (8 bytes): Offset of the data block containing the key
//
// 4. Footer:
//    - Index Block Offset (8 bytes): Offset of the index block
//    - Index Block Size (4 bytes): Size of the index block
//    - Minimum Version (8 bytes): Minimum version of the SSTable
//    - Maximum Version (8 bytes): Maximum version of the SSTable
//    - Minimum Key Length (4 bytes): Minimum length of keys in the SSTable
//    - Maximum Key Length (4 bytes): Maximum length of keys in the SSTable
//    - Minimum Key (variable length): Minimum key in the SSTable
//    - Maximum Key (variable length): Maximum key in the SSTable
//    - Footer Block Offset (8 bytes): Offset of the footer block
//    - WyHash Checksum (8 bytes): Checksum of the above data
//    - Footer Magic (8 bytes): Identifies the file as an SSTable footer
//
// Note: All multi-byte integers are stored in little-endian format.

type SStableWriter struct {
	w io.WriteCloser

	hashSeed uint64

	currentBlockSize int
	currentBlockData []byte

	minKey []byte
	maxKey []byte

	minVersion uint64
	maxVersion uint64

	currentBlockKeys [][]byte
}

func NewSStableWriter(w io.WriteCloser) *SStableWriter {
	g := &SStableWriter{
		w: w,
	}

	var b [8]byte
	rand.Read(b[:])
	g.hashSeed = binary.LittleEndian.Uint64(b[:])
	return g
}

func (g *SStableWriter) WriteHeader() error {
	g.currentBlockData = g.currentBlockData[:0]

	g.currentBlockData = append(g.currentBlockData, _MAGIC_BYTES...)
	var b [8]byte
	binary.LittleEndian.PutUint32(b[:4], _DATABASE_SSTABLE_VERSION)
	g.currentBlockData = append(g.currentBlockData, b[:4]...)
	flag := _DATABASE_SSTABLE_HEADER_FLAG_RESERVED
	binary.LittleEndian.PutUint32(b[:4], flag)
	g.currentBlockData = append(g.currentBlockData, b[:4]...)
	binary.LittleEndian.PutUint64(b[:], g.hashSeed)
	g.currentBlockData = append(g.currentBlockData, b[:8]...)

	// Calculate and append the WyHash checksum
	checksum := wyhash.Hash(g.currentBlockData, g.hashSeed)
	binary.LittleEndian.PutUint64(b[:], checksum)
	g.currentBlockData = append(g.currentBlockData, b[:8]...)

	_, err := g.w.Write(g.currentBlockData)
	if err != nil {
		return err
	}
	return nil
}
