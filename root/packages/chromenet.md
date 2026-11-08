---
author: Lemon Mint
title: 'gosuda.org/chromenet: Chromium''s Network Stack for Go, Built Reproducibly from a Pinned Release'
description: CGO_ENABLED=0 Go http.RoundTripper that sends requests through Chromium's own //net stack (HTTP/1.1, HTTP/2, QUIC/HTTP/3, BoringSSL) in an isolated HQQ worker, with source, toolchains, and six embedded binaries pinned by SHA-256 and built in gvx sandboxes.
language: en
date: 2026-11-09T00:00:00.000000000Z
path: /chromenet
go_package: gosuda.org/chromenet
go_repourl: https://github.com/gosuda/chromenet.git
category: network
---

`gosuda.org/chromenet` is a Go `http.RoundTripper` that sends requests through Chromium's own network stack. The stack is `//net` from an official Chromium release: its HTTP/1.1, HTTP/2, and QUIC/HTTP/3 implementations, BoringSSL, Chrome's TLS configuration, and the Chrome Root Store. It runs in a separate worker process, `chrome_net_server`, and the Go side stays pure Go (`CGO_ENABLED=0`), talking to the worker over [HQQ](https://gosuda.org/hqq) shared-memory RPC.

```bash
go get gosuda.org/chromenet@v0.1.10157081033800
```

## Why Run Chromium's Network Stack

A Go program that talks to the web with `net/http` presents a Go client to the server. That is the right default for service-to-service traffic. It is the wrong one when the server's behavior depends on what kind of client it thinks it is talking to: browser-gated endpoints, anti-bot middleware, CDNs that serve different protocol versions to different TLS and HTTP/2 fingerprints, and tests that must reproduce what a real browser experiences.

Go's standard library cannot give you a browser's wire behavior. `crypto/tls` has its own cipher-suite order and extension set, and `net/http` has its own HTTP/2 settings. Libraries that imitate a browser by copying its ClientHello bytes keep copying, release after release. `chromenet` takes the other route: it does not imitate the Chromium network stack, it runs it.

Measured on the `darwin/arm64` engine against a public fingerprint service (`tls.peet.ws`), the default configuration reports:

| Property                           | Observed value                                                        |
| :--------------------------------- | :-------------------------------------------------------------------- |
| Negotiated protocol                | `h2`                                                                  |
| JA4                                | `t13d1516h2_8daaf6152771_806a8c22fdea`                                |
| HTTP/2 fingerprint (Akamai format) | `1:65536;2:0;4:6291456;6:262144\|15663105\|0\|m,a,s,p`                |
| User-Agent                         | desktop Chrome of the engine's major version, e.g. `Chrome/157.0.0.0` |

With `ForceHTTP3`, requests to `cloudflare-quic.com` and `www.google.com` were answered over HTTP/3. These numbers come from one platform on one day. The engine is the stack, so they change when the pinned Chromium release changes.

## Using It

`NewTransport` returns an `http.RoundTripper`. Use it wherever you would use `http.Transport`:

```go
package main

import (
	"fmt"
	"io"
	"net/http"

	"gosuda.org/chromenet"
)

func main() {
	client := &http.Client{Transport: chromenet.NewTransport(chromenet.Options{})}

	resp, err := client.Get("https://example.com/")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.Proto, resp.StatusCode, len(body))
}
```

On the first request, the module writes the embedded worker binary for your platform to the user cache directory, verifies its SHA-256 digest with `gosuda.org/hqq/secureexec`, starts it with a parent lifeline (`--parent-stdin`, so it exits when your process exits), and opens a session pool. Nothing needs to be installed.

A few behaviors follow `http.Transport` so that `http.Client` works unchanged:

- The Transport does not follow redirects and keeps no cookies. `http.Client` does both, as it does with `http.Transport`.
- Responses compressed with gzip, deflate, Brotli, or zstd arrive decoded, and `resp.Uncompressed` is true.
- A request holds one engine session until its response body is closed. Close bodies.
- Cancelling the request context cancels the request inside Chromium.
- Network failures are `*client.NetError` values carrying the Chromium error code and name:

```go
var ne *client.NetError
if errors.As(err, &ne) {
	fmt.Println(ne.Code, ne.Name) // -105 ERR_NAME_NOT_RESOLVED
}
```

### Options

| Field                         | Effect                                                                                                 |
| :---------------------------- | :----------------------------------------------------------------------------------------------------- |
| `UserAgent`                   | Replaces the default Chrome User-Agent. A `User-Agent` request header still wins.                      |
| `AcceptLanguage`              | Replaces `en-US,en;q=0.9`.                                                                             |
| `Proxy`                       | Chromium proxy rules such as `http://127.0.0.1:8080` or `socks5://127.0.0.1:1080`. Empty means direct. |
| `DisableHTTP2`, `DisableQUIC` | Turn the protocols off.                                                                                |
| `ForceHTTP3`                  | Use QUIC for every origin without waiting for an `Alt-Svc` advertisement.                              |
| `InsecureSkipVerify`          | Accept any server certificate.                                                                         |
| `Pool`                        | Engine sessions to use. Nil uses `client.DefaultPool()`, which allows 64 concurrent requests.          |

Transports with equal options share one Chromium `URLRequestContext`, so connection pools, HTTP/2 sessions, and QUIC sessions are reused across requests. Setting `HQQ_CHROME_NET_SERVER=/path/to/chrome_net_server` replaces the embedded worker, for example with a local build.

## How It Works

| Layer               | What runs there                                                                                                                                                        |
| :------------------ | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Your program        | `http.Client` and `chromenet.Transport`, pure Go.                                                                                                                      |
| Process boundary    | HQQ shared-memory RPC. The interface is `chrome_net.idl`, compiled by `hqq-bindgen` into the Go client and the C server glue.                                          |
| `chrome_net_server` | A small C++ layer on top of Chromium: one network thread that owns every `//net` object, and session threads that block on a request until the network thread answers. |
| Chromium `//net`    | `URLRequestContext`, `URLRequest`, HTTP/2 and QUIC sessions, BoringSSL.                                                                                                |

The IDL has six operations that matter: create an engine from `key=value` options, start a request with its head (`METHOD URL` plus header lines) and body length, write request body chunks, wait for the response head, read the decoded body, and destroy the handles. A request handle declares `@cancel`, so an abandoned call interrupts the request inside Chromium instead of leaving it to run to completion.

Isolating the stack in a worker process is a safety property as much as a packaging one. Chromium's network code parses hostile input from the open internet. If it crashes, your Go service does not crash with it: the call fails with an error, the session is discarded, and the client starts a new worker the next time it needs one.

## Built Reproducibly, Pinned by SHA-256

A 25 MB binary that contains Chromium is only trustworthy if you can say exactly what went into it. `chromenet` follows Chromium Stable the way `gosuda.org/sqlite3` follows SQLite: the source is never vendored, only pinned.

| Pin                                                                                                                      | Where                          |
| :----------------------------------------------------------------------------------------------------------------------- | :----------------------------- |
| Chromium release and its official `-lite` source archive                                                                 | `third_party/chromium/VERSION` |
| SHA-256 of the archive and of every toolchain that release pins: clang, Rust, GN, ninja, the Windows clang runtime, xwin | `third_party/chromium/SHA256`  |
| MSVC CRT and Windows SDK versions fetched through xwin                                                                   | `third_party/chromium/WINSDK`  |
| The sandbox runtime (`rungv`) release, the guest image digest, the Debian snapshot date                                  | `scripts/engine/BUILDENV`      |
| The HQQ C runtime                                                                                                        | `go.mod` and `go.sum`          |

The current pin is Chromium `157.0.8103.38`. A download that does not match its pin stops the build. The release applies exactly one patch to Chromium: on Linux, `chrome_net_server` is built without NSS and verifies server certificates against the Chrome Root Store only, so the binary needs nothing beyond libc.

The Linux and Windows engines are built inside [gvx](https://gosuda.org/blog/posts/gvx-cross-platform-linux-containers-z48916d01) sandboxes: unprivileged processes with a pinned guest image and no host daemon. The Windows engines are cross-compiled on Linux with Chromium's clang-cl and lld-link against the Windows SDK that xwin fetches from Microsoft. The macOS engines build natively on a macOS runner, because Chromium's macOS build needs Xcode.

Rebuilding the same commit produced byte-identical binaries across separate CI runs. The release binaries are byte-identical to the ones an earlier CI run built for the commit that landed the engine in the monorepo, on all six platforms.

| Platform        | Binary size |
| :-------------- | ----------: |
| `linux/amd64`   |     27.9 MB |
| `linux/arm64`   |     27.4 MB |
| `darwin/amd64`  |     26.2 MB |
| `darwin/arm64`  |     23.9 MB |
| `windows/amd64` |     27.3 MB |
| `windows/arm64` |     25.8 MB |

The Linux engines depend only on `libc`, `libm`, `libdl`, `libpthread`, and `libgcc_s`, and need glibc 2.31 or newer.

### Tracking Chromium and Releases

A scheduled workflow checks Chromium Stable every six hours. For a new release it rewrites the pins, builds and tests all six engines on a `chromium/<version>` branch, attests the binaries, and commits the pins together with the embedded binaries, their digests, and the license texts to `main` in one commit. In the monorepo, per-file `OWNERS` grant the workflow's bot exactly those files, so the update lands without a human step, while changes to the build recipe itself (the Windows SDK pin and the patch) still need an owner.

Every release binary carries a build provenance attestation from the workflow that built it:

```bash
gh attestation verify chrome_net_server_linux_amd64 --repo gosuda/chromenet
```

The license texts of what each binary contains (Chromium, BoringSSL, QUICHE, and the rest of the build graph) are generated by Chromium's own `tools/licenses/licenses.py` and shipped with the release as `NOTICE_<os>_<arch>.txt`.

## Limits

- `DialWebSocket` is a pure-Go RFC 6455 client and does not use the Chromium stack.
- `DialWebTransport` is an in-process implementation. It does not reach a server.
- The default JA4 and HTTP/2 fingerprints are those of the pinned Chromium release. Chromium changes them from release to release, and `chromenet` follows.
- The macOS binaries are ad-hoc signed and not notarized.
- A worker process costs memory. Each engine binary is 24 to 28 MB on disk, and Chromium's runtime starts a thread pool and a network thread.
- An engine session is held until the response body is closed. A program that leaks response bodies will exhaust the pool.

## Source

The repository is [github.com/gosuda/chromenet](https://github.com/gosuda/chromenet), mirrored from `//net/chromenet` in the gosuda monorepo. The build recipe lives in `scripts/engine`, the engine glue in `engine/`, and the interface in `chrome_net.idl`.
