# SSTable File Format

The SSTable (Sorted String Table) is a file format used for storing key-value pairs. The format is structured as follows:

## 1. Header

| Field | Size (bytes) | Description |
|-------|--------------|-------------|
| Magic Number | 8 | Identifies the file as an SSTable (0xf3db64e176e9b2f5) |
| Version | 4 | SSTable format version (current: 10) |
| Flags | 4 | Reserved for future use and optimizations |
| Hash Seed | 8 | Seed for the WyHash checksum, ensures integrity |
| WyHash Checksum | 8 | Checksum of the above data for validation |

## 2. Data Blocks

Multiple data blocks, each containing:

| Field | Size (bytes) | Description |
|-------|--------------|-------------|
| Flags | 4 | Flags for the block (e.g., compression, encryption) |
| Bloom Filter Size | 4 | Size of the optional Bloom filter (0 if not present) |
| Bloom Filter | Variable | Optional filter for optimizing key lookups |
| Key-Value Pairs | Variable | Core data storage (see below) |
| WyHash Checksum | 8 | Checksum of all above data in the block |

Key-Value Pairs:

| Field | Size (bytes) | Description |
|-------|--------------|-------------|
| Key Length | 4 | Length of the key |
| Key | Variable | The actual key |
| Version | 8 | Version of the key-value pair |
| Flags | 4 | Flags specific to the key-value pair |
| Value Length | 4 | Length of the value |
| Value | Variable | The actual value |

## 3. Index Block

| Field | Size (bytes) | Description |
|-------|--------------|-------------|
| Number of Index Entries | 4 | Count of index entries |
| Index Entries | Variable | Quick lookup entries (see below) |

Index Entries:

| Field | Size (bytes) | Description |
|-------|--------------|-------------|
| Key Length | 4 | Length of the indexed key |
| Key | Variable | The indexed key |
| Version | 8 | Version of the indexed key-value pair |
| Offset | 8 | Offset of the data block containing the key |

## 4. Footer

| Field | Size (bytes) | Description |
|-------|--------------|-------------|
| Index Block Offset | 8 | Offset of the index block for quick access |
| Index Block Size | 4 | Size of the index block |
| Minimum Version | 8 | Lowest version in the SSTable |
| Maximum Version | 8 | Highest version in the SSTable |
| Minimum Key Length | 4 | Shortest key length in the SSTable |
| Maximum Key Length | 4 | Longest key length in the SSTable |
| Minimum Key | Variable | Lexicographically smallest key |
| Maximum Key | Variable | Lexicographically largest key |
| Footer Block Offset | 8 | Offset of the footer block |
| WyHash Checksum | 8 | Checksum of the above footer data |
| Footer Magic | 8 | Identifies the file as an SSTable footer (0xcf56bff25a91312a) |

Note: All multi-byte integers are stored in little-endian format. The maximum size of an SSTable is limited to 20MiB for efficient management.
