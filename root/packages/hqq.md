---
id: 43974d116e135fd9d3478f658dcf71c6
author: Lemon Mint
title: 'HQQ: Isolating C Libraries from Go Over Cache-Line Shared Memory'
description: Why we built a shared-memory IPC runtime, an IDL compiler with generation-checked handles, and a sealed-binary process launcher to call C libraries from Go with CGO_ENABLED=0.
language: en
date: 2026-10-18T13:56:46.094643498Z
path: /hqq
go_package: gosuda.org/hqq
go_repourl: https://github.com/gosuda/HQQ.git
---

`gosuda.org/hqq` is a zero-cgo inter-process communication runtime and code generation toolchain for Go and C11. It isolates native C libraries inside separate OS worker processes and connects them to Go callers over memory-mapped, cache-line-aligned shared-memory rings. The project combines two transport tiers (`StandardLink` for dedicated single-producer single-consumer sessions and `Advanced` for multi-worker MPMC dispatch), the `hqq-bindgen` IDL compiler, and a tamper-verified process supervisor (`secureexec` and `launch`).

```bash
go get gosuda.org/hqq
```

## The Cost of Sharing an Address Space with C

Every Go systems engineer eventually hits the same fork in the road. You are building a Go service that compiles in seconds with `CGO_ENABLED=0`, ships as a single static binary, cross-compiles from a laptop to Linux or Windows targets without a sysroot, and runs inside a memory-safe garbage-collected heap. Then you need a mature C library: SQLite for embedded relational storage, Zstandard (`libzstd`) for dictionary compression, or a hardware-accelerated media or cryptography library.

Enabling `cgo` to link that C library into the Go process solves the immediate binding problem while introducing three structural liabilities:

1. **Toolchain and Cross-Compilation Coupling**: `CGO_ENABLED=1` requires a C compiler, target libc headers, and a platform linker for every target OS and architecture. Building `linux/arm64` or `windows/arm64` artifacts from a `darwin/arm64` workstation turns a single `go build` command into a matrix of cross-compilers and sysroots.
2. **Runtime Scheduling Overhead**: Every `cgo` call crosses the Go runtime boundary, switches from a growable goroutine stack to a system stack, and pins an OS thread (`M`). For fine-grained calls, the transition tax (~50 to 150 ns per call) and scheduler contention accumulate quickly.
3. **Fault Domain Collapse**: C code runs in the same virtual address space as the Go runtime. A single out-of-bounds write, use-after-free, stack overflow, or `abort()` inside a C function corrupts Go heap metadata or terminates the entire server process immediately, bypassing `recover()` and taking every in-flight HTTP or RPC request down with it.

When teams try to avoid `cgo`, the common alternatives force a different compromise. Pure-Go mechanical translations of C codebases preserve `CGO_ENABLED=0`, but often trail upstream C releases, carry an emulated virtual memory layer, and allocate heavily under load. Moving the C library into a helper process over loopback TCP or Unix domain sockets restores fault isolation, but every call pays for two kernel context switches, socket buffer copies, and wire serialization.

We built HQQ after asking a direct question: what if the C library lived in a separate OS process with its own virtual address space, while Go and C exchanged frames directly through shared CPU cache lines without entering the kernel on the fast path?

| Integration Model | `CGO_ENABLED=0` | Fault Isolation | Fast-Path Syscalls | Per-Call Heap Allocations | Upstream C11 Fidelity |
| :--- | :--- | :--- | ---: | ---: | :--- |
| **`cgo` In-Process Linking** | No | None (shared heap) | 0 | 0 to 1+ | Exact upstream C |
| **Socket / Pipe Subprocess RPC** | Yes | Process boundary | 2 to 4 | Multiple (framing + I/O) | Exact upstream C |
| **Mechanical C-to-Go Translation** | Yes | In-process Go | 0 | High on complex calls | Emulated C runtime |
| **HQQ Shared-Memory IPC** | Yes | Process boundary | 0 (adaptive spin) | 0 (`StandardLink`) | Exact upstream C |

## Architectural Overview: Four Layers Across Process Boundaries

HQQ separates process isolation into four composable layers so that shared-memory ring mechanics, typed RPC code generation, binary integrity verification, and child process supervision remain independent.

```text
+-------------------------------------------------------------------+
| Go Application Process (CGO_ENABLED=0)                            |
|                                                                   |
|  +-----------------------+     +-------------------------------+  |
|  | Generated Go Client   |     | Process Supervisor            |  |
|  | (gosuda.org/hqq/      |     | (gosuda.org/hqq/launch +      |  |
|  |  bindgen/rt)          |     |  gosuda.org/hqq/secureexec)   |  |
|  +-----------+-----------+     +---------------+---------------+  |
|              |                                 |                  |
|  +-----------v-----------+                     | Anonymous Pipe   |
|  | Transport Tier        |                     | Lifeline         |
|  | (StandardLink /       |                     | (--parent-stdin) |
|  |  Advanced MPMC)       |                     |                  |
+--------------+---------------------------------+------------------+
               |                                 |
===============|========= Shared Memory =========|===================
               |     (mmap / CreateFileMapping)  |
+--------------v---------------------------------v------------------+
| C11 Worker Process (Isolated Address Space)                       |
|                                                                   |
|  +-----------------------+     +-------------------------------+  |
|  | Generated C11 Server  |---->| Generation-Checked Handle     |  |
|  | & Batch Shims         |     | Table (hqq_rpc.c)             |  |
|  +-----------+-----------+     +---------------+---------------+  |
|              |                                 |                  |
|              +----------------+----------------+                  |
|                               |                                   |
|                     +---------v---------+                         |
|                     | Native C Library  |                         |
|                     | (SQLite, Zstd...) |                         |
|                     +-------------------+                         |
+-------------------------------------------------------------------+
```

### 1. Shared-Memory Transport (`gosuda.org/hqq` and `c/`)

The base package `gosuda.org/hqq` and its C11 counterpart in `c/` implement two shared-memory transport modes over POSIX `mmap` (Linux and macOS) and Win32 `CreateFileMappingW` (Windows). Every shared segment begins with a verified `HQQ_STL3` superblock that pins protocol version, buffer geometry, and endpoint PIDs before any ring index is read.

All hot control words and slot headers are aligned to 64-byte cache-line boundaries so producer and consumer cores never contend on the same cache line through false sharing.

```text
StandardLink Slot Layout (64-Byte Cache-Line Aligned)
Byte Offset:  0x00                0x08                0x10                          0x40
              +-------------------+-------------------+-----------------------------+
Header Line:  | seq (u64 atomic)  | len (u32) | flags | reserved (zero-padded)      |
              +-------------------+-------------------+-----------------------------+
Payload Area: | payload bytes [0 .. BufferSize - 1] (directly encoded / decoded)    |
              +---------------------------------------------------------------------+
```

| Field | Offset | Size | Semantics |
| :--- | ---: | ---: | :--- |
| `seq` | `0x00` | 8 B | Monotonic sequence counter. Odd values mark an in-progress write; even values publish a committed frame via store-release. |
| `len` | `0x08` | 4 B | Payload byte length written into the slot (`0 <= len <= BufferSize`). |
| `flags` | `0x0C` | 4 B | Frame control bits (continuation, error status, or shutdown notice). |
| `payload` | `0x40` | `BufferSize` | Contiguous payload region read and written in place without intermediate copies. |

Callers choose between two transport tiers depending on concurrency structure:

- **`StandardLink` (1:1 SPSC Session)**: Pairs a client and a server session over dedicated request and response slots protected by a sequence lock (`SeqLock`). On the fast path, the writer serializes directly into the mapped slot and commits with an atomic store-release; the reader spins briefly in user space before falling back to OS wake primitives (`futex` on Linux, `__ulock_wait` on macOS, `WaitOnAddress` on Windows). A single-frame round trip completes in ~200 ns with `0 allocs/op`.
- **`Advanced` (MPMC Dispatch Ring)**: Multiplexes `M` concurrent Go callers across `N` C worker threads using a lock-free multi-producer multi-consumer submission ring, a shared slot pool, and an indexed completion table. Callers reserve a slot, publish the slot index to the ring, and wait on their completion entry while worker threads dequeue and complete requests out of order.

Both tiers track peer liveness continuously. Each endpoint updates an atomic monotonic timestamp via `Heartbeat()` and inspects `PeerAlive(timeout)` while waiting. If a worker process crashes mid-call, the peer detects the stale heartbeat or dead PID immediately, aborts blocked waits with `ErrPeerGone`, and supports atomic session `Takeover` without leaving wedged shared-memory files behind.

### 2. Zero-Allocation RPC and Generation-Checked Handles (`gosuda.org/hqq/bindgen`)

Writing shared-memory framing by hand for a C library with dozens of functions is tedious and error-prone. `hqq-bindgen` is a schema compiler that reads an Interface Definition Language (`.idl`) file (or bootstraps one directly from a C `.h` header via `hqq-bindgen init`) and emits a zero-cgo Go client package alongside a C11 server skeleton.

The most dangerous mistake in any C binding is exposing raw C pointers (`sqlite3*`, `sqlite3_stmt*`, `ZSTD_CCtx*`) across the language boundary. If a caller passes a pointer after freeing it, passes a pointer belonging to another session, or corrupts an integer address, the C library dereferences invalid memory.

`hqq-bindgen` forbids raw pointers on the wire. Instead, the IDL models stateful C resources as typed `handle` declarations:

```text
service sqlite3 {
    handle Db;
    handle Stmt @owned_by(Db);

    fn open_v2(path: string, flags: i32) -> result<Db, i32>;
    fn prepare_one(db: Db, sql: string) -> result<Stmt, i32>;
    fn finalize(stmt: @consume Stmt) -> result<void, i32>;
    fn close(db: @consume Db) -> result<void, i32>;
}
```

On the wire, every handle is a 64-bit integer packing a 32-bit table index and a 32-bit non-zero generation counter: `(uint64_t)generation << 32 | index`. The C runtime (`c/src/hqq_rpc.c`) maintains a per-session handle table with three invariants:

1. **Generation Validation**: Every lookup verifies that the slot is occupied, its stored generation matches the upper 32 bits of the handle, and its handle type tag matches the parameter type (`Db` vs. `Stmt`). Reusing a closed handle fails immediately with `HQQRPC_E_BAD_HANDLE` before calling into the C library.
2. **Hierarchical Ownership (`@owned_by`)**: Because `Stmt` is annotated with `@owned_by(Db)`, closing a `Db` handle automatically walks all live child `Stmt` handles owned by that `Db` and runs their destructors in reverse allocation order (`sqlite3_finalize` before `sqlite3_close_v2`).
3. **Session Teardown Sweep**: If a Go client disconnects or crashes without closing its handles, the C server's session destructor sweeps the entire session handle table in reverse order, guaranteeing zero leaked C allocations across client restarts.

Before a session processes its first RPC call, the client and server exchange a 32-byte SHA-256 interface digest computed over the canonicalized IDL (`HQQ-BINDGEN-IDL-V1\n` prefix). If the Go client and C worker were compiled from incompatible IDL schemas, the handshake rejects the connection at bootstrap.

### 3. Tamper-Free Binary Execution (`gosuda.org/hqq/secureexec`)

Moving C code into a worker binary raises a supply-chain question: how does the Go process know that the worker binary on disk was not modified or replaced between verification and execution?

Calling `crypto/sha256` on a file path followed by `os/exec.Command(path).Start()` leaves a Time-of-Check to Time-of-Use (TOCTOU) window where another process can overwrite or swap the file before the kernel executes it. `gosuda.org/hqq/secureexec` closes this window:

- On Linux, `secureexec` opens the binary, streams its contents into an anonymous memory file descriptor created via `memfd_create(MFD_CLOEXEC | MFD_ALLOW_SEALING)` while computing its SHA-256 hash, verifies the hash against a pinned 32-byte digest or an Ed25519-signed release manifest (`cmd/hqq-manifest`), applies `fcntl(F_ADD_SEALS, F_SEAL_SEAL | F_SEAL_SHRINK | F_SEAL_GROW | F_SEAL_WRITE)` so no process on the system can mutate the pages, and spawns the child from the sealed descriptor.
- On macOS and Windows, `secureexec` materializes verified binaries into permission-locked cache directories keyed by SHA-256 digest and verifies file size and digest integrity prior to launch.

### 4. Parent-Lifeline Process Supervision (`gosuda.org/hqq/launch`)

When a Go application spawns a background C worker process, the worker must never outlive its parent as an orphan process, even if the Go parent is terminated with `SIGKILL` (`kill -9`) and cannot run `defer` cleanup handlers.

`gosuda.org/hqq/launch` enforces lifetime coupling through an anonymous OS pipe passed to the worker's standard input with the `--parent-stdin` flag. The Go parent holds the write end of the pipe and never writes to it; the C worker monitors the read end on a dedicated lifeline thread. If the Go process exits normally, panics, or is killed by the OS kernel (`SIGKILL` or OOM killer), the kernel immediately closes the write descriptor. The C worker receives `EOF` on the pipe within microseconds, interrupts any active calls, runs its session handle destructors, unlinks the shared-memory files, and exits cleanly.

## Quickstart: Calling a Generated Service

With `hqq-bindgen` and `launch`, calling an isolated C worker from Go looks like a regular typed Go client while keeping `CGO_ENABLED=0`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"gosuda.org/hqq/bindgen/rt"
	"gosuda.org/hqq/launch"
	"gosuda.org/hqq/secureexec"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	hash, err := secureexec.ParseHash("7649b1b45dacd934cde8149f35f4fcc3937f6486fbac23ce976466719b3752bb")
	if err != nil {
		log.Fatal(err)
	}

	listener, err := launch.NewListenerName("demo")
	if err != nil {
		log.Fatal(err)
	}

	srv, err := launch.Start(ctx, launch.Config{
		Binary:   "/path/to/verified-worker",
		SHA256:   hash,
		Listener: listener,
		Args:     []string{listener},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer srv.Close()

	fmt.Printf("worker running on shared-memory listener %s (opts=%+v)\n", listener, rt.SessionOptions{})
}
```

## Repository Structure

Every subsystem in `gosuda.org/hqq` is organized into focused packages with explicit boundaries:

```text
gosuda.org/hqq
├── standard.go / ring.go      # StandardLink (SPSC) and low-level shared-memory ring primitives
├── advanced.go / slotpool.go  # Advanced (MPMC) dispatcher, slot pool, and completion table
├── shm_*.go / wait_*.go       # OS-specific mmap/CreateFileMapping and futex/ulock/WaitOnAddress
├── c/                         # C11 transport library (hqq.c) and RPC server runtime (hqq_rpc.c)
├── bindgen/                   # hqq-bindgen IDL parser, verifier (V1-V41), and Go/C11 code generator
│   └── rt/                    # Zero-allocation Go client runtime for generated services
├── secureexec/                # SHA-256 / Ed25519 manifest verification and memfd-sealed execution
├── launch/                    # Worker process supervisor with --parent-stdin lifeline management
├── cmd/hqq-manifest/          # CLI tool for generating and signing release binary manifests
└── spec/                      # Executable wire specifications and golden test vectors (BINDGEN.md)
```

To see how HQQ powers drop-in production replacements for two of the most widely used `cgo` libraries in the Go ecosystem, check out the companion packages built on this stack:

- **[`gosuda.org/sqlite3`](https://gosuda.org/sqlite3)**: Drop-in `database/sql` driver compatible with `github.com/mattn/go-sqlite3`, running native C SQLite `3.54.1` over shared memory with pipelined batch prefetching.
- **[`gosuda.org/zstd`](https://gosuda.org/zstd)**: Drop-in replacement for `github.com/DataDog/zstd`, running native C `libzstd 1.5.7` in an isolated shared-memory worker with zero-allocation contexts and bulk batching.
- **GitHub Repository**: [https://github.com/gosuda/HQQ](https://github.com/gosuda/HQQ)
