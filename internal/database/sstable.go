package database

const (
	_DATABASE_SSTABLE_MAGIC        = 0xf3db64e176e9b2f5
	_DATABASE_SSTABLE_FOOTER_MAGIC = 0xcf56bff25a91312a
	_DATABASE_SSTABLE_VERSION      = 10

	_DATABASE_SSTABLE_MAX_SIZE = 20 * 1024 * 1024 // 20MiB
)

// SSTable Format:
//
// An SSTable (Sorted String Table) is a file format used for storing key-value pairs.
// The format is as follows:
//
// 1. Header:
//    - Magic Number (8 bytes): Identifies the file as an SSTable
//    - Version (4 bytes): SSTable format version
//    - Flags (4 bytes): Reserved for future use
//    - Hash Seed (8 bytes): Seed for the WyHash checksum
//    - WyHash Checksum of above (8 bytes)
//
// 2. Data Blocks:
//    - Multiple data blocks, each containing:
//      a. Flags (4 bytes): Flags for the block
//      b. Bloom Filter Size (4 bytes): Optional, 0 if not present
//      c. Bloom Filter (variable length): Optional
//      d. Key-Value Pairs:
//         - Key Length (4 bytes)
//         - Key (variable length)
//         - Version (8 bytes)
//         - Value Length (4 bytes)
//         - Value (variable length)
//      e. WyHash Checksum of a || b || c || d (8 bytes)
//
// 3. Index Block:
//    - Number of Index Entries (4 bytes)
//    - Index Entries:
//      a. Key Length (4 bytes)
//      b. Key (variable length)
//      c. Version (8 bytes)
//      d. Offset (8 bytes): Offset of the data block containing the key
//
// 5. Footer:
//    - Index Block Offset (8 bytes): Offset of the index block
//    - Index Block Size (4 bytes): Size of the index block
//    - Minimum Version (8 bytes): Minimum version of the SSTable
//    - Maximum Version (8 bytes): Maximum version of the SSTable
//    - Minimum Key Length (4 bytes): Minimum length of keys in the SSTable
//    - Maximum Key Length (4 bytes): Maximum length of keys in the SSTable
//    - Minimum Key (variable length): Minimum key in the SSTable
//    - Maximum Key (variable length): Maximum key in the SSTable
//    - Footer Block Offset (8 bytes): Offset of the footer block
//    - WyHash Checksum of above (8 bytes)
//    - Footer Magic (8 bytes): Identifies the file as an SSTable footer
//
// Note: All multi-byte integers are stored in little-endian format.
