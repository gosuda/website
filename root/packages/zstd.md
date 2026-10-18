---
id: e52d68870be3de9f644fcfc584bd8aec
author: Lemon Mint
title: 'gosuda.org/zstd: Drop-In DataDog/zstd Compression with Zero CGO and Crash Isolation'
description: Running upstream C libzstd 1.5.7 in a sandboxed shared-memory worker with a 100% DataDog/zstd-compatible Go API, zero-allocation contexts, and bulk batching.
language: en
date: 2026-10-18T14:07:11.281986997Z
path: /zstd
go_package: gosuda.org/zstd
go_repourl: https://github.com/gosuda/hqq-zstd.git
---

`gosuda.org/zstd` is a `CGO_ENABLED=0` Go compression library that runs the official C Zstandard (`libzstd 1.5.7`) engine inside an isolated worker process over [HQQ](https://gosuda.org/hqq) shared memory. It provides a drop-in replacement API for `github.com/DataDog/zstd` (including one-shot `Compress`/`Decompress`, streaming `io.Writer`/`io.Reader`, pre-digested dictionaries, and reusable zero-allocation contexts) while keeping C execution completely outside the Go runtime address space.

```bash
go get gosuda.org/zstd@v0.3.10507
```

## Why Decompression Belongs Outside Your Go Heap

In production backends, compression and decompression sit directly on the untrusted input boundary. Log collectors, message brokers, RPC gateways, and storage engines decompress gigabytes of externally supplied frames every minute.

Many Go teams choose `github.com/DataDog/zstd` because it binds directly to upstream C `libzstd`: the C implementation delivers high single-core throughput, tight dictionary compression ratios on small records, and off-heap C arenas (`ZSTD_CCtx` and `ZSTD_DCtx`) that do not inflate Go garbage-collection scan times.

Linking `libzstd` into a Go service via `cgo`, however, brings two operational pain points:

1. **Build and Deployment Friction**: Just like `go-sqlite3`, `DataDog/zstd` requires `CGO_ENABLED=1` and a target C compiler toolchain. Cross-compiling services or building static binaries for `linux/arm64` or `windows/arm64` requires external C sysroots.
2. **Shared Address Space on Untrusted Inputs**: Decompressing corrupted or adversarial payloads inside the same virtual address space as your Go server means any bug in native C code or custom dictionary handling can terminate the entire Go process.

`gosuda.org/zstd` keeps the exact upstream C `libzstd 1.5.7` engine and the familiar `github.com/DataDog/zstd` API, but moves the C execution into a verified child process (`zstd_server`) connected to your Go program over cache-line shared memory.

## Drop-In Replacement for `github.com/DataDog/zstd`

Migrating from `github.com/DataDog/zstd` requires changing only the import path to `"gosuda.org/zstd"`. No external binary installation or setup code is needed: on the first call, `gosuda.org/zstd` materializes an embedded SLSA-attested `zstd_server` binary for your platform, verifies its SHA-256 digest via `gosuda.org/hqq/secureexec`, and opens a shared-memory session pool.

```go
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"gosuda.org/zstd"
)

func main() {
	// 1. One-shot Compress / Decompress (mirrors github.com/DataDog/zstd)
	src := []byte("shared-memory zstd payload without cgo")
	compressed, err := zstd.CompressLevel(nil, src, zstd.BestSpeed)
	if err != nil {
		log.Fatal(err)
	}
	plain, err := zstd.Decompress(nil, compressed)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("one-shot: %d bytes -> %d bytes (%s)\n", len(src), len(compressed), plain)

	// 2. Streaming Writer and Reader with Dictionary Support
	var buf bytes.Buffer
	w := zstd.NewWriterLevel(&buf, zstd.DefaultCompression)
	if _, err := w.Write(src); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}

	r := zstd.NewReader(&buf)
	defer r.Close()
	streamed, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("streamed: %s\n", streamed)
}
```

### Supported Platforms and Embedded Binaries

All six target platforms ship with prebuilt, SHA-256-pinned worker binaries embedded via platform-specific build tags (`internal/embedded/embed_<goos>_<goarch>.go`):

| Target OS | Target Arch (`GOARCH`) | Embedded Binary | Binary Size | Build Toolchain & Provenance |
| :--- | :--- | :--- | ---: | :--- |
| **Linux** | `amd64`, `arm64` | `zstd_server_linux_<arch>` | ~770 - 880 KiB | GCC `-O3`, SLSA Build L3 Provenance |
| **macOS** | `arm64`, `amd64` | `zstd_server_darwin_<arch>` | ~660 - 795 KiB | AppleClang `-O3`, SLSA Build L3 Provenance |
| **Windows** | `amd64` | `zstd_server_windows_amd64.exe` | ~818 KiB | MSVC `-O2`, SLSA Build L3 Provenance |
| **Windows** | `arm64` | `zstd_server_windows_arm64.exe` | ~551 KiB | `zig cc -target aarch64-windows-gnu -O3`, SLSA L3 |

## Inside the Shared-Memory Data Path

To keep throughput competitive with in-process `cgo`, `gosuda.org/zstd` avoids socket serialization and heap allocations on the hot path.

### 1. Zero-Allocation Slot Encoding (`zstd_ctx.go` and `c/zstd_hooks.c`)

When you use a reusable `zstd.Ctx` (or when the package-level `Compress` and `Decompress` functions borrow a `Ctx` from the internal session pool), data flows directly through the mapped `StandardLink` slot:

```text
Go Caller (zstd.Ctx)                    Shared-Memory Slot                    C11 Worker (zstd_hooks.c)
--------------------                    ------------------                    -------------------------
src []byte  ------------------------->  Request Slot Payload  ------------->  ZSTD_compressCCtx(
                                                                                cctx, resp_slot, src, level)
dst []byte  <-------------------------  Response Slot Payload <-------------  Writes compressed frame directly
                                                                              into shared response slot
```

1. The generated `hqq-bindgen` client copies `src` straight into the memory-mapped request slot and flips the `SeqLock` sequence word.
2. The C worker reads `src` in place from the request slot and passes the response slot's payload pointer directly as the destination buffer to `ZSTD_compressCCtx` or `ZSTD_decompressDCtx`.
3. The Go client copies the resulting slice from the response slot into the caller-supplied `dst` slice. In steady state, `ctx.CompressTo(dst, src, level)` and `ctx.DecompressInto(dst, src)` perform `0 allocs/op`.

Pre-digested dictionaries (`CDict` and `DDict`) live on the C worker behind generation-checked 64-bit handles (`handle CDict`, `handle DDict` in `zstd.idl`). Creating a dictionary once via `ctx.NewCDict(rawDict, level)` parses the dictionary tables inside the C worker and returns a lightweight handle that can be reused across thousands of calls without re-sending the dictionary bytes.

### 2. Amortizing IPC Overhead with Bulk Framing (`zstd_bulk.go`)

For large payloads (16 KiB and above), the ~200 ns shared-memory round trip is negligible compared to the CPU time spent inside `libzstd`. For workloads that compress thousands of small 100-byte to 1-KiB messages at once, however, paying even 200 ns per item adds up.

`gosuda.org/zstd` provides `CompressBulk` and `DecompressBulk` on `BulkCtx` (`zstd_bulk.go`) to eliminate per-item round trips:

```go
bctx, err := client.NewBulkCtx(1 << 20)
if err != nil {
	log.Fatal(err)
}
defer bctx.Close()

items := [][]byte{
	[]byte(`{"event":"login","user":101}`),
	[]byte(`{"event":"purchase","user":101,"amount":4200}`),
	[]byte(`{"event":"logout","user":101}`),
}
compressedItems, err := bctx.CompressBulk(nil, items, zstd.DefaultCompression)
```

`CompressBulk` packs the entire `[][]byte` batch into a single length-prefixed request frame in shared memory. The C worker loops over every item in C (`compress_bulk` in `c/zstd_hooks.c`), writes all compressed frames contiguously into the response slot, and returns the entire batch in a single shared-memory round trip.

### 3. Surviving Worker Termination Without Crashing the Caller

Because `zstd_server` runs in a separate OS process, a fatal failure inside the C worker never brings down your Go application:

- If `zstd_server` is killed externally (for example, by `SIGKILL` or an OOM event), the HQQ transport layer detects the broken peer immediately and returns a typed `*rt.TransportError` from the in-flight `Compress` or `Decompress` call.
- Package-level functions (`zstd.Compress`, `zstd.Decompress`, `zstd.NewWriter`, `zstd.NewReader`) automatically detect that the shared default worker is gone, spawn a fresh verified `zstd_server` instance on the next call, and resume normal operation.

## Repository Layout

The codebase separates the drop-in `DataDog/zstd` facade from the low-level `hqq-bindgen` session and bulk primitives:

```text
gosuda.org/zstd
├── autolaunch.go / hashes.go  # Drop-in Compress/Decompress API, auto-recovery pool, and SHA-256 hashes
├── zstd_stream.go             # Streaming Writer/Reader (NewWriterLevelDict, NewReaderDict)
├── zstd_ctx.go                # Zero-allocation Ctx, CDict, and DDict wrappers over hqq-bindgen
├── zstd_bulk.go               # Single-round-trip CompressBulk and DecompressBulk framing
├── zstd.go                    # Generated hqq-bindgen Go client for the zstd IDL service
├── zstd.idl                   # HQQ IDL schema defining Context, CDict, DDict, and RPC methods
├── internal/embedded/         # Platform-tagged //go:embed zstd_server binaries (6 OS/Arch targets)
└── c/                         # C11 server implementation (zstd_server.c, zstd_hooks.c, main.c)
```

## Getting Started

Install `gosuda.org/zstd` with `go get` and use it with `CGO_ENABLED=0`:

```bash
go get gosuda.org/zstd@v0.3.10507
```

- **GitHub Repository**: [https://github.com/gosuda/hqq-zstd](https://github.com/gosuda/hqq-zstd)
- **Architecture & Benchmarks**: [docs/ARCHITECTURE.md](https://github.com/gosuda/hqq-zstd/blob/main/docs/ARCHITECTURE.md)
- **HQQ Core Runtime**: [https://gosuda.org/hqq](https://gosuda.org/hqq)
