---
id: f5707f0f4f230a276a3428b1eb1e6469
author: Lemon Mint
title: 'Schottky: Zero-Allocation, Order-Preserving Byte-Key Encoding for Go'
description: Schottky is a high-performance Go library designed to encode multi-type composite values into order-preserving byte keys. Featuring zero-allocation execution, independent NULL ordering, PostgreSQL 18 type compatibility, and prefix-scan upper bounding, it is built for modern storage engines and index pipelines.
language: en
date: 2026-08-31T07:47:42.614875839Z
path: /schottky
go_package: gosuda.org/schottky
go_repourl: https://github.com/gosuda/schottky.git
---
When implementing LSM-tree or B-tree based key-value stores and database indexes, composite fields often need to be combined into a single byte key. 

Standard serialization formats like JSON or Protocol Buffers are not designed for this purpose because their serialized byte outputs do not preserve the natural sorting order required by unsigned bytewise comparisons (`bytes.Compare` or `memcmp`).

Schottky is a Go library designed to encode multi-type composite tuples into order-preserving byte keys.

```bash
go get gosuda.org/schottky@latest
```

### Serialization vs. Sort Keys

Guaranteeing correct bytewise sorting requires addressing several low-level data representation details:

- **Integers**: Standard two's complement big-endian encoding breaks natural ordering due to the most significant sign bit. Inverting the sign bit is necessary for correct unsigned byte comparisons.
- **Floating-point numbers**: Requires sign bit adjustments, inverted ordering for negative values, and consistent handling of `NaN` and `-0`.
- **Variable-length strings and byte slices**: Field boundaries must be preserved without breaking prefix sort orders.
- **Composite key requirements**: Support for independent ASC/DESC ordering per field, decoupled NULLS FIRST/LAST rules, strict lexicographical precedence (earlier fields determine order), and compatibility with prefix scanning.

Schottky converts each value into a canonical payload before applying presence tags and directional orientation. For DESC fields, each byte of the ASC payload is bitwise inverted (`^b`). NULL placement is handled via dedicated presence tags and operates independently of sort direction.

### Basic Usage

The following example builds a composite key consisting of an Account ID (`ASC, NULLS LAST`) and a Name (`DESC, NULLS FIRST`):

```go
package main

import (
        "fmt"

        "gosuda.org/schottky"
)

func main() {
        storage := make([]byte, 0, 128)
        builder := schottky.NewBuilder(storage)

        builder.Int64(42, schottky.AscNullsLast)
        accountPrefixLen := builder.Len()

        builder.String("Ada", schottky.DescNullsFirst)
        key, err := builder.Key()

        if err != nil {
                panic(err)
        }
        accountPrefix := key[:accountPrefixLen]
        fmt.Printf("key=%x\nprefix=%x\n", key, accountPrefix)
}
```

Schottky provides four explicit sort order configurations:

- `AscNullsFirst`
- `AscNullsLast`
- `DescNullsFirst`
- `DescNullsLast`

NULL positioning is never implicitly inferred. If an invalid `Order` value is passed, the builder records `ErrInvalidOrder`, which is returned when calling `Key()` or `Err()`.

### Prefix Scanning and Range Bounds

Schottky composite keys contain no global headers, field count metadata, type tags, or trailers. Assuming the schema is known ahead of time, field encodings are simply concatenated.

Because of this layout, the encoded bytes of leading fields form a valid prefix for range scans. In the example above, `accountPrefix` can be directly used as a prefix filter to scan all records where `Account ID == 42`.

To compute the exclusive upper bound for half-open `[prefix, upper)` range scans, use `PrefixUpperBound`:

```go
upperStorage := make([]byte, 0, len(accountPrefix))
upper, finite, err := schottky.PrefixUpperBound(upperStorage, accountPrefix)

if err != nil {
        panic(err)
}

if finite {
        // Half-open [accountPrefix, upper) range scan
} else {
        // Unbounded open range scan
}
```

*Note: `Builder.Len()` must be measured at clean field boundaries. Slicing inside a field's internal byte stream produces an invalid prefix.*

### Zero-Allocation and Buffer Management

Key generation frequently runs on critical database paths. To eliminate heap allocations and buffer resizing overhead, `Builder` works strictly within the capacity of the caller-provided slice and will not reallocate internally.

If the buffer runs out of capacity, `ErrShortBuffer` is recorded without writing partial bytes. Field writes are atomic, and the first error encountered is preserved until checked via `Key()` or `Err()`. Providing sufficient capacity upfront ensures zero-allocation encoding.

Buffer sizes can be calculated in advance using helper functions such as `EncodedBytesSize`, `EncodedStringSize`, and `EncodedDecimalSize`, or via fixed-size constants. The returned key references the provided buffer directly, leaving memory lifecycle management to the caller.

The `Decoder` works symmetrically: it borrows directly from the input key, requires caller-provided destination buffers for variable-length fields, and provides `Remaining() == 0` to detect trailing bytes or schema mismatches.

### Supported Data Types

- **Integers**: Signed and Unsigned (8-bit to 64-bit), `Int128`
- **Floating-Point & Numerics**: `Float32`, `Float64`, Decimal Text
- **Basic Types**: Binary String, Byte Slice, Boolean, Enum Rank
- **Date & Time**: Date, Time, Zoned Time, Timestamp, Duration, Calendar Interval
- **Network & Identifiers**: UUID, MAC, IP, IP Prefix, Canonical Network Prefix, LSN
- **Composite Structures**: Nested Tuples, Ranges, and raw structural encodings
- **Collation**: Unicode Collation Keys and external canonical tokens

SQL type mapping is aligned with PostgreSQL 18 B-tree sorting rules. Types dependent on database catalogs or internal engine state are handled by passing external canonical tokens.

### String Collation

`Builder.String` defaults to raw UTF-8 binary order. For locale-aware sorting, Schottky provides a concurrent-safe, immutable `Collator`:

- **Deterministic Collation**: Encodes the collation key alongside raw UTF-8 bytes to provide a tie-breaker when collation weights are identical.
- **Nondeterministic Collation**: Treats collation-equal strings as identical, omitting the raw byte tie-breaker.

Unicode and profile versions should be tracked in the metadata schema. If collation providers or profile settings change, existing keys must be rebuilt.

### Schema Management

Because Schottky keys are raw, headerless byte sequences, the schema layer must track:

1. Field sequence and data types.
2. Sort directions (`ASC`/`DESC`) and NULL ordering (`NULLS FIRST`/`LAST`).
3. String collation and normalization rules.
4. Schottky and Collation profile versions.

Comparing keys generated with different schemas or decoding against a mismatched schema breaks ordering guarantees.

### Performance and Links

On Go 1.27+, experimental portable SIMD acceleration can be enabled using `GOEXPERIMENT=simd`. Scalar and SIMD paths produce byte-identical keys.

- **GitHub Repository**: https://github.com/gosuda/schottky
- **Key Layout Specification**: https://github.com/gosuda/schottky/blob/main/docs/03-key-layout.md
- **SQL Type Mapping Guide**: https://github.com/gosuda/schottky/blob/main/docs/17-sql-type-map.md
- **Go API Reference**: https://github.com/gosuda/schottky/blob/main/docs/18-api.md

