---
id: e62d78ef5850c982dbddc9d3ff9134ae
author: Lemon Mint
title: 'gvx: Running Linux Containers, Docker, and Kubernetes in One Unprivileged Process'
description: Technical overview of gvx (rungvx CLI and Go SDK), a security-hardened, cross-platform container runtime built on the gVisor user-space kernel that safely runs Linux containers, Docker-in-Docker, and multi-node HA Kubernetes in a single unprivileged process across Windows, macOS, Linux, and Android.
language: en
date: 2026-11-06T12:02:56.008516536Z
path: /blog/posts/gvx-cross-platform-linux-containers-z48916d01
go_package: z.gfire.dev/gvx
go_repourl: https://github.com/gfire-sigs/gvx.git
category: lowlevel
---

## Introduction: Security-Hardened Containers Without Root or Host Daemons

Running Linux containers, `docker build`, or Kubernetes inside a development machine, CI runner, or multi-tenant cloud node usually forces an uncomfortable security trade-off:

- On **Linux**, standard OCI runtimes (`runc`, `crun`, `runsc`) require either `root` privileges, host `/var/run/docker.sock` exposure, `--privileged` (`CAP_SYS_ADMIN`), or user namespaces (`unshare(CLONE_NEWUSER)`) that expand the host kernel attack surface.
- On **Windows** and **macOS**, container engines (`Docker Desktop`, `Podman Machine`, `Lima`, `OrbStack`) boot a full Linux virtual machine kernel inside WSL2, Hyper-V, or `Virtualization.framework` and run a persistent background daemon inside that VM.

**[gvx](https://github.com/gfire-sigs/gvx)** (`rungvx` CLI and the `z.gfire.dev/gvx/pkg/gvx` Go SDK) provides a **secure, zero-privilege alternative**. Built as an upstream-trackable fork of Google's **gVisor** user-space Linux kernel (`Sentry`) and user-space network stack (`netstack`) with a `< 5.0%` audited diff against weekly upstream releases, `gvx` executes unmodified `linux/amd64` and `linux/arm64` ELF64 binaries and OCI containers directly inside **a single unprivileged, self-locked host process** on Windows, macOS, Linux, and Android.

> **Security Guarantee (Three-Ring Defense-in-Depth)**: `gvx` never weakens host OS security boundaries, never installs kernel drivers, and never requires `root`, `sudo`, `--privileged` on the host, or `CLONE_NEWUSER`. Before executing a single guest instruction, `gvx` locks down its own host process using Windows Untrusted/Restricted Tokens and Job Objects, macOS Seatbelt (`deny default`), or Linux Landlock + `PR_SET_NO_NEW_PRIVS` + BPF `seccomp`. All guest system calls, virtual `UID 0` privileges, mounts (`VFS2`), and network tables (`netstack`) are strictly confined inside the memory-safe Go `Sentry` kernel.

| Runtime                    | Host OS                                                            |        Cold Start | Memory Footprint | Admin or `sudo` Needed | Background VM Daemon | Direct OCI Image Run |
| :------------------------- | :----------------------------------------------------------------- | ----------------: | ---------------: | :--------------------- | :------------------- | :------------------- |
| **`gvx` (`whpx` / `hvf`)** | Windows (`amd64`, `arm64`) & macOS (`arm64`)                       |    `~12 to 20 ms` |        `~30 MiB` | No                     | No                   | Yes                  |
| **`gvx` (`systrap`)**      | Linux (`amd64`, `arm64`), Android (`arm64`), Windows (opt-in only) |     `~2 to 50 ms` |        `~31 MiB` | No                     | No                   | Yes                  |
| **WSL2 / Docker Desktop**  | Windows & macOS                                                    |  `800 to 2500 ms` |      `> 512 MiB` | Yes (Initial Setup)    | Yes                  | Requires Daemon      |
| **QEMU (Full System)**     | Windows & macOS                                                    | `1500 to 4000 ms` |      `> 256 MiB` | No                     | No                   | Requires Disk Image  |

No Linux VM kernel boots on Windows or macOS. No background daemon runs on the host. No `root`, `sudo`, user namespaces (`CLONE_NEWUSER`), `mount(2)`, `TUN`/`TAP` drivers, or `CAP_NET_ADMIN` privileges are required on the host OS. Inside that unprivileged process, `gvx` runs **Docker-in-Docker (`docker:dind`)**, `docker build`, `docker compose`, and **multi-node High-Availability Kubernetes (`k3s` with embedded `etcd` and `Calico` CNI)**.

---

## Container Lifecycle and Three-Ring Host Lockdown

Every `gvx` sandbox runs inside one unprivileged host process and enforces a three-phase lifecycle: pre-opening all required host file descriptors and sockets, locking down the host OS process via `pkg/hostlockdown`, and entering the in-process `Sentry` execution loop.

![gvx Container Lifecycle and Host Self-Lockdown](/assets/images/gvx/gvx-lifecycle-lockdown.png)

Before transferring control to the first guest instruction, `gvx` pre-opens its cached `.erofs` image layer descriptors, `.ext4` volume file descriptors, `/workspace` `*os.Root` handle, control/checkpoint descriptors (`-S`, `-checkpoint-dir`), and host network sockets, then invokes `hostlockdown.Apply()`:

1. **Windows NT (`lockdown_windows.go`, `broker_windows.go`, `netbroker_windows.go`)**: Calls `AdjustTokenPrivileges` with `SE_PRIVILEGE_REMOVED` to strip privileges in-place, starts locked `winMountBroker` and `winNetBroker` helper threads, binds `NULL_SID` restricted thread tokens via a background sweeper, lowers the **primary token itself to Untrusted integrity (`S-1-16-0`)**, assigns the process to an anonymous Job Object with `JOB_OBJECT_LIMIT_ACTIVE_PROCESS = 1` and `JOB_OBJECT_UILIMIT_ALL`, and enables `DisallowWin32kSystemCalls = 1` via `SetProcessMitigationPolicy`.
2. **macOS XNU (`lockdown_darwin.go`)**: Pins `GOMAXPROCS`, sets `RLIMIT_NPROC = 0`, and applies a Seatbelt profile starting from `(version 1)(deny default)` that blocks `process-fork`, `process-exec*`, unbound `file-*` opens, and unauthorized binds (permitting only explicit mount paths, plus `localhost:*` binds when `--auto-publish` or `--dynamic-ports` is active).
3. **Linux and Android (`landlock_linux.go`, `lockdown_linux.go`, `pkg/gvx/syscallfilter_linux.go`)**: Applies a **Landlock ruleset (ABI v1–v5)** scoped to mounted host paths across all threads, sets `PR_SET_NO_NEW_PRIVS`, and installs the Sentry BPF `seccomp` allowlist across all threads (`SECCOMP_FILTER_FLAG_TSYNC`) with explicit `EPERM` rules for `execve`, `execveat`, `clone`, `openat`, and `socket`. (On Android, `pkg/hostlockdown` detects Zygote's `seccomp` filter, skips Landlock, and relies on per-UID SELinux domains.)

---

## Cross-Platform Syscall Interception and 4 KB Paging (`pkg/sentry/platform`)

Linux ELF64 binaries assume 4 KB (`0x1000`) memory page granularity for `mmap(MAP_FIXED)` segment mappings. However, Windows `VirtualAlloc` and `MapViewOfFile3` enforce 64 KB (`0x10000`) alignment, and Apple Silicon macOS (`darwin/arm64`), 16 KB Linux `arm64` kernels, and Android 16 use 16 KB (`0x4000`) host pages. Furthermore, each OS requires a different mechanism to intercept Linux `SYSCALL` (`amd64`) and `SVC #0` (`arm64`) instructions without booting a Linux VM.

`gvx` solves both problems across execution backends that implement `pkg/sentry/platform.Context` (selected via `--backend=auto|whpx|hvf|systrap`, while `--platform linux/ARCH` selects the OCI image architecture):

![gvx Platform Syscall Interception and 4KB Paging Mechanics](/assets/images/gvx/gvx-platform-syscalls.png)

### 1. Windows Hardware Engine: `platform/whpx` (`WinHvPlatform.dll`)

On `windows/amd64` and `windows/arm64`, `platform/whpx` (sharing `platform/hvcommon` with `hvf`) creates a lightweight hypervisor partition via `WHvCreatePartition` without booting a guest OS kernel:

- **64 KB-to-4 KB Paging Decoupler**: `whpx` registers each arena of `pgalloc.MemoryFile` with the partition once through `WHvMapGpaRange`. Subsequent `MapFile` and `Unmap` calls modify only the in-partition guest page tables, enforcing exact 4 KB `R/W/X` permissions despite Windows' 64 KB allocation granularity.
- **Zero-API Register Exchange via `CommPage`**:
  - On `amd64`, guest Ring-3 `SYSCALL` enters an in-partition Ring-0 `MSR_LSTAR` stub (`stubgen_x86.go`), which spills the fast-path syscall GPRs (`RAX, RDI, RSI, RDX, R10, R8, R9, RIP, RSP`) into a shared 4 KB `CommPage` and executes `HLT` to exit `WHvRunVirtualProcessor`.
  - On `arm64`, guest EL0 `SVC #0` enters EL1 vectors (`stubgen_a64.go`), spills `X0..X8, SP_EL0, ELR_EL1, PSTATE` into `CommPage`, and loads from unbacked GPA zero to trigger a memory exit back to the Sentry.

### 2. macOS Hardware Engine: `platform/hvf` (`Hypervisor.framework`)

On Apple Silicon (`darwin/arm64`; Intel Macs are unsupported), `platform/hvf` runs container threads inside `Hypervisor.framework` vCPUs (`hv_vm_create`, `hv_vcpu_run`):

- **Two-Stage 4 KB over 16 KB MMU Translation**: Stage-2 (`hv_vm_map`) maps 1 GiB memory-file chunks into Intermediate Physical Address (IPA) space once at 16 KB host page granularity with `HV_MEMORY_READ | HV_MEMORY_WRITE | HV_MEMORY_EXEC`. Inside the vCPU, the EL1 `bluepill` stub (`bluepill_arm64.go`) configures `TCR_EL1.TG0 = 0b00` (4 KB granule) in `TTBR0_EL1` and populates 4 KB Stage-1 page table entries with exact `R-X` or `RW-` (`UXN=1`) permissions. Because the ARM64 MMU enforces the intersection of Stage-1 and Stage-2 permissions, the hardware traps permission faults at exact 4 KB boundaries.
- **Syscall Trap**: Guest EL0 `SVC #0` traps to EL1 `VBAR_EL1`, spills registers to the shared 4 KB `CommPage`, and executes `HVC #0` to return from `hv_vcpu_run` to the Go Sentry thread.

### 3. Windows Opt-In User-Mode Engine: `platform/systrap` (`winsystrap`)

When hardware virtualization is unavailable (such as inside nested cloud VMs), `winsystrap` runs **trusted** guest workloads in user mode on `windows/amd64` and `windows/arm64`. Because Windows offers no kernel syscall trap comparable to Linux `seccomp`, `--backend=auto` fails closed when WHP is absent unless you explicitly opt in with `--backend=systrap` or `-allow-unconfined-systrap`:

- **Hardware `CR3` / `TTBR0_EL1` Isolation**: Guest threads run inside isolated stub subprocesses booted from a generated 1 KB executable (`stubimage_xplat.go`) confined by a Low-Integrity token and a Job Object.
- **In-Place Binary Patching & `winSysmsg` IPC**: On `amd64`, the patcher (`patcher_amd64_xplat.go`) replaces 2-byte `syscall` (`0f 05`) instructions with 5-byte `jmp rel32` jumps (or `CC 90` breakpoint traps when a 5-byte window is unavailable) into a fast-path island. On `arm64`, `svc #0` is patched with a `B` branch to an island slot. The island spills registers to the shared `winSysmsg` page and synchronizes with the Sentry via `WaitOnAddress` and `WakeByAddressSingle`, while unpatched traps and page faults stop on a Windows Debug Object (`WaitForDebugEventEx`).

### 4. Linux and Android User-Mode Engine: `platform/systrap`

On `linux/amd64`, `linux/arm64` (both 4 KiB and 16 KiB host pages, via `subblock_xplat.go`), and `android/arm64` (packaged as `librungvx.so` inside `gvx-android.aar`), `systrap` runs guest threads (`sysmsg`) that trap guest syscalls via `SECCOMP_RET_TRAP` (`SIGSYS`) and binary-patch `syscall` sites into a shared-memory trampoline (`sysmsg`) synchronized via `SYS_FUTEX`, while a stub `syscallThread` (`syscall_thread.go`) executes address-space operations (`mmap`, `munmap`, `clone`) via `SECCOMP_RET_USER_NOTIF` (`_RUN_SECCOMP_LOOP`). When `gvx` runs inside an outer `gVisor` (`runsc`) or `rungvx` sandbox where the single `SECCOMP_FILTER_FLAG_NEW_LISTENER` slot returns `EBUSY`, `gvx` automatically switches `syscallThread` to a shared-memory futex loop (`_RUN_SYSCALL_LOOP`), as detailed in [Part 3 of this series](/blog/posts/docker-in-docker-inside-gvisor-with-gvx-z48916d03).

---

## How `gvx` Maps Guest Linux System Calls to Virtual Subsystems and Host Handles

Once a platform engine traps a guest `SYSCALL` (`amd64`) or `SVC #0` (`arm64`) instruction and writes the guest registers into the 4 KB shared page (`CommPage`, `sysmsg`, or `winSysmsg`), control returns to the Go `Sentry` task goroutine (`pkg/sentry/kernel/task_syscall.go`). Rather than forwarding Linux system calls to the underlying host OS kernel, `Task.doSyscall()` saves the original register state (`SyscallSaveOrig()`), evaluates any in-guest BPF `seccomp` filters and `ptrace` stops, and dispatches the system call number through `linux.AMD64` or `linux.ARM64` (`pkg/sentry/syscalls/linux/linux64.go`, covering ~370 Linux system calls) into four internal domains:

![How gvx Maps Guest Linux Syscalls to Subsystems and Host OS](/assets/images/gvx/gvx-syscall-mapping.png)

| Syscall Domain                          | Representative Guest Linux Syscalls                                                                                         | Internal `gvx` Subsystem (`pkg/sentry`)                                                                                        | Host OS Boundary Interaction (After `hostlockdown`)                                                                                                                                         |
| :-------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **A. Process, Namespace, Mount & Sync** | `clone`, `execve`, `unshare`, `setns`, `mount`, `umount2`, `pivot_root`, `setuid`, `capset`, `futex`, `epoll_wait`, `pipe2` | `kernel.Task`, virtual credentials, futex table, and in-memory `VFS2` (`procfs`, `sysfs`, `cgroupfs` v2, `tmpfs`, `overlayfs`) | **Zero host syscalls** (100% in-memory Go state; completely invisible to the host OS)                                                                                                       |
| **B. Virtual Memory (`MM`)**            | `mmap`, `munmap`, `mremap`, `mprotect`, `brk`, `madvise`, `memfd_create`                                                    | `mm.MemoryManager` (4 KB VMA tree) & `pgalloc.MemoryFile`                                                                      | Bidirectional 4 KB page-table updates via `WHvMapGpaRange`, `hv_vm_map` + `TTBR0_EL1`, or stub `mmap`/`mprotect`                                                                            |
| **C. File & Storage (`VFS2`)**          | `openat`, `pread64`, `pwrite64`, `getdents64`, `statx`, `renameat2`, `setxattr`, `mknod`, `inotify_add_watch`               | `vfs.VirtualFilesystem` mount table router (`fsimpl/erofs`, `fsimpl/ext4lite`, `fsimpl/hostfs`, `fsimpl/rpcfs`)                | Bidirectional `pread`/`pwrite` on 1 pinned `.erofs` FD or 1 `.ext4` FD, scoped relative walks on `*os.Root`, or `gvx-ctrl/2` RPC                                                            |
| **D. Networking & Netfilter**           | `socket`, `bind`, `listen`, `connect`, `sendmsg`, `recvmsg`, `netlink`, `setsockopt(IPT_SO_SET_REPLACE)`                    | User-space `pkg/tcpip` (`lo`, `eth0`, `docker0`, `veth`, `vxlan`, `wireguard`, `ipset`, `iptables-legacy`)                     | Intra-sandbox traffic uses **zero host syscalls**; external traffic maps to `net.Dial`/`net.Listen` (`-net=nat`, optional `--nat-policy-file` & bandwidth caps) or 1 UDP socket (`-net=wg`) |

When the handler returns `(rval, ctrl, err)`, `Task.doSyscallInvoke()` converts any error into a negative Linux `errno` (`-ExtractErrno(err, sysno)`), writes the return value back to the shared register page (`RAX` on `amd64`, `X0` on `arm64`), and resumes guest execution.

---

## Storage (`VFS2`), Dual-Track Networking (`netstack`), and Live Control

All guest file and network operations are serviced inside the Go `Sentry` through `VFS2` filesystems and two `pkg/netbackend` tracks:

![gvx VFS2 Storage Engines and Dual-Track Netstack](/assets/images/gvx/gvx-storage-netstack.png)

| Storage Engine        | Mount Role in `gvx`                                                      | Backing Host Resource                  | Host `openat` / `NtCreateFile` Calls at Run Time | Windows Defender (`WdFilter.sys`) Per-File Scans |
| :-------------------- | :----------------------------------------------------------------------- | :------------------------------------- | -----------------------------------------------: | -----------------------------------------------: |
| **`fsimpl/erofs`**    | Read-only OCI image lower layers                                         | Single `<sha256>.erofs` file per layer |                                              `0` |                                              `0` |
| **`fsimpl/ext4lite`** | Named volumes (`-v name:/path`), `-rootfs-volume`, `type=overlay` layers | Single `<name>.ext4` file per volume   |                                              `0` |                                              `0` |
| **`fsimpl/hostfs`**   | Host bind mounts (`-workspace`, `-v /host:/guest`, single-file mounts)   | Pinned `*os.Root` host handle          |                     `O(depth)` per uncached open |                          `1` per host file close |

### Why `ext4lite` Matters for Docker, Kubernetes, and Package Managers

Extracting or writing tens of thousands of container files directly onto Windows NTFS or macOS APFS triggers antivirus minifilter scans (`WdFilter.sys`), case-sensitivity collisions, and missing POSIX extended attributes.

`pkg/sentry/fsimpl/ext4lite` implements a pure-Go `ext4` filesystem inside one pre-opened host file (`%LOCALAPPDATA%\gvx\cache\volumes\<name>.ext4` on Windows, `~/Library/Caches/gvx/volumes/<name>.ext4` on macOS, or `~/.cache/gvx/volumes/<name>.ext4` on Linux) with:

- **4 KiB blocks and 16 MiB dynamic block groups (`meta_bg`, up to 64 TiB)**: Grows dynamically without sparse-file allocation stalls (`fallocate` on Linux, `fcntl(F_PREALLOCATE)` on macOS, `FileAllocationInfo` on Windows), supports `size=` growth caps and `ephemeral` scratch volumes, punches freed holes online (`FITRIM`), and compacts unused trailing groups on clean unmount or via `rungvx volume compact`.
- **JBD2 ordered-mode journal (inode 8) and shared locks**: Atomic transaction commits with CRC-32 checksums and revoke blocks, `fsckLite` recovery, `cgroup2` `io.stat` / `/proc/diskstats` counters, and `LOCK_SH` read-only locks enabling copy-on-write volume branching (`--mount type=overlay,src=base,upper=branch,dst=/path`) as well as persistent root filesystems (`-rootfs-volume`).
- **`overlayfs` upper-layer support**: Implements `trusted.overlay.*` and `security.capability` extended attributes, character device whiteouts (`0:0`), `d_type` directory entry reporting, and `RENAME_EXCHANGE`.

### Live Control (`gvx-ctrl/2`), Checkpoint/Restore, and Extensions

Recent `gvx` releases expand the single-process model with lockdown-safe lifecycle tooling:

- **Live control (`gvx-ctrl/2`)**: Binding `-S`/`--control-socket` (or `--name NAME --control`) before lockdown enables `rungvx exec`, `cp`, `export`, `stats`, `top`, `dmesg`, `events`, `pause`/`unpause`, live `update` (memory, CPU quota, PIDs, LAN isolation, NAT bandwidth, tunables), `port` / `port-forward`, `unix-forward`, `net-policy`, and client-served `mount serve` (`fsimpl/rpcfs`) without weakening host confinement.
- **Lockdown-safe Checkpoint & Restore**: Pre-opening `-checkpoint-dir` and `-S` allows `rungvx checkpoint` to snapshot full guest memory, task state, mounts, and network namespaces into a 4-file bundle (`manifest.json`, `state.img`, `pages.meta`, `pages.img`) and resume it in a new locked-down process via `rungvx restore`.
- **Sandbox Extension SPI (`ext/` submodule)**: Keeps the core `rungvx` binary pure Go (`CGO_ENABLED=0`) while allowing compile-time opt-in extensions such as headless macOS GPU rendering (`ext/cmd/rungvx` built with `-tags ext_gpu`, supporting `-gpu=venus` and `-gpu=virgl` via `gvx-gpu-renderer`). Run `rungvx version -V` to inspect compiled build options and extensions.

---

## Practical Workloads: Docker Engine, Kubernetes (`k3s` + `Calico`), and Go SDK

### Installing `rungvx`: SLSA-Attested Prebuilt Releases or `go install`

You can download a prebuilt `rungvx` archive for `linux/amd64`, `linux/arm64`, `linux/arm64v8.2`, `darwin/arm64`, `windows/amd64`, `windows/arm64`, or `android/arm64` (`gvx-android.aar`) from [GitHub Releases](https://github.com/gfire-sigs/gvx/releases) (each built by the SLSA release pipeline with a Sigstore provenance bundle and `SHA256SUMS`), or install directly from source with Go `1.27.4+` (`go install`). For a graphical interface on macOS or Windows, see the community-maintained [GVX Desktop](https://github.com/ironpark/gvx-desktop) app.

```sh
# Option A: Download the latest prebuilt release and verify SHA-256 + SLSA provenance
VERSION=$(curl -fsSL https://api.github.com/repos/gfire-sigs/gvx/releases/latest | sed -n 's/.*"tag_name": "\([^"]*\)".*/\1/p')
TARGET=linux-amd64 # or darwin-arm64, linux-arm64, linux-arm64v8.2
ARCHIVE="rungvx-${VERSION}-${TARGET}.tar.gz"
BASE="https://github.com/gfire-sigs/gvx/releases/download/${VERSION}"

curl -fsSLO "${BASE}/${ARCHIVE}" "${BASE}/SHA256SUMS" "${BASE}/rungvx-${VERSION}.sigstore.json"
grep " ${ARCHIVE}\$" SHA256SUMS | shasum -a 256 -c -
gh attestation verify "${ARCHIVE}" \
  --bundle "rungvx-${VERSION}.sigstore.json" \
  --repo gfire-sigs/gvx \
  --signer-workflow gfire-sigs/gvx/.github/workflows/release.yml \
  --source-ref "refs/tags/${VERSION}" --deny-self-hosted-runners
tar -xzf "${ARCHIVE}"
mv "rungvx-${VERSION}-${TARGET}/rungvx" ./rungvx

# Option B: Install directly via Go 1.27.4+ (Linux / Windows / macOS)
CGO_ENABLED=0 go install z.gfire.dev/gvx/cmd/rungvx@latest
```

> On macOS (`darwin/arm64`), the prebuilt release archive is already signed with the `com.apple.security.hypervisor` entitlement (`xattr -d com.apple.quarantine ./rungvx` clears browser quarantine). If you build locally with `go install` or `go build` on macOS, sign the binary with `curl -fsSLO https://raw.githubusercontent.com/gfire-sigs/gvx/main/entitlements.plist && codesign --sign - --entitlements entitlements.plist --force "$(go env GOPATH)/bin/rungvx"`.

### 1. Running an Unprivileged Docker Engine (`docker:dind`)

Following [`docs/recipes/docker-engine.md`](https://github.com/gfire-sigs/gvx/blob/main/docs/recipes/docker-engine.md), you can run an unmodified `docker:dind` container inside `rungvx` on Windows, macOS, or Linux and control it with your host `docker` CLI:

```sh
# 1. Create the host directory for the forwarded Unix socket
mkdir -p ~/.gvx

# 2. Start docker:dind inside a gvx sandbox with an ext4lite volume
#    --auto-publish binds `docker run -p` / Compose ports on 127.0.0.1 automatically
#    (or replace --auto-publish with static `-p 127.0.0.1:9000-9019:9000-9019`)
rungvx run \
  --auto-publish \
  -p unix:~/.gvx/docker.sock:2375 \
  -v gvx-docker:/var/lib/docker \
  -e DOCKER_TLS_CERTDIR= \
  docker:dind --tls=false
```

In a second terminal, point your host `docker` CLI at the sandbox socket (add `--host-ca-certs` to `rungvx run` when operating behind a corporate TLS-inspecting proxy):

```sh
docker context create gvx --docker host=unix://$HOME/.gvx/docker.sock
docker context use gvx

docker info --format '{{.Driver}} {{.OperatingSystem}}'
# Output: overlayfs Alpine Linux ... (containerized)

docker build -t myapp ./app
docker compose up -d
```

### 2. Running Kubernetes (`k3s` + `Calico`) in Unprivileged Sandboxes

Following [`docs/recipes/kubernetes.md`](https://github.com/gfire-sigs/gvx/blob/main/docs/recipes/kubernetes.md), because `gvx` implements `/dev/kmsg`, `cgroup` v2 (`memory.max` OOM killer, `cpu.max` bandwidth throttling, `pids.max`, and `io.stat`), `/proc/diskstats`, `wireguard` and `vxlan` netlink devices, `ipset`, and `iptables-legacy` (`addrtype`, `conntrack`, `statistic`, `recent`, `rpfilter`), an unmodified `rancher/k3s` image reaches `Ready` inside a single `rungvx` process:

```sh
mkdir -p ~/.gvx/k3s

rungvx run \
  -e IPTABLES_MODE=legacy \
  -p 127.0.0.1:6443:6443 \
  -p 127.0.0.1:30000-30019:30000-30019 \
  -v k3s:/var/lib/rancher \
  --mount type=bind,src=$HOME/.gvx/k3s,dst=/output \
  rancher/k3s:v1.31.4-k3s1 server \
  --disable=traefik,servicelb \
  --write-kubeconfig=/output/kubeconfig.yaml \
  --write-kubeconfig-mode=644
```

You can also join multiple `gvx` nodes over WireGuard (`flannel-backend=wireguard-native` or a `wg0` mesh) with embedded `etcd` high availability, or run **Calico** (`calico-vxlan.yaml` with `FELIX_IPTABLESBACKEND=Legacy`) to enforce Kubernetes `NetworkPolicy` completely inside user space.

### 3. Embedding Linux Containers in Go via `z.gfire.dev/gvx/pkg/gvx`

Go applications can embed Linux containers directly without an external container daemon:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"z.gfire.dev/gvx/pkg/gvx"
)

func main() {
	ctx := context.Background()
	sb, err := gvx.New(ctx, gvx.Config{
		Platform:      gvx.PlatformAuto,
		NetworkMode:   gvx.NetModeNAT,
		OCIImage:      "alpine/git:latest",
		WorkspacePath: "./repo",
		Argv:          []string{"sh", "-c", "git init /workspace && git -C /workspace status"},
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "init sandbox: %v\n", err)
		os.Exit(1)
	}
	defer sb.Close()

	exitCode, err := sb.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run sandbox: %v\n", err)
		os.Exit(1)
	}
	os.Exit(exitCode)
}
```

---

## Further Reading in This Series

1. **[gvx: Running Linux Containers, Docker, and Kubernetes in One Unprivileged Process](/blog/posts/gvx-cross-platform-linux-containers-z48916d01)** (this post)
2. **[Running Docker, docker build, and docker compose Without Root or User Namespaces](/blog/posts/run-docker-without-root-or-userns-gvx-z48916d02)**: How to bootstrap `docker:dind` in locked-down sandboxes where `sudo`, `unshare(CLONE_NEWUSER)`, `mount(2)`, and `/dev/fuse` are blocked.
3. **[Running Docker-in-Docker Inside gVisor and Nested Sandboxes with gvx](/blog/posts/docker-in-docker-inside-gvisor-with-gvx-z48916d03)**: Why `apt-get install docker.io` and nested `runsc` fail inside `gVisor` across five kernel barriers, and how `gvx` runs nested `docker:dind` using a shared-memory futex loop.
4. **[Fast Isolated Docker Environments for AI Agent RL Rollouts with gvx](/blog/posts/fast-docker-environments-for-agent-rl-rollouts-z48916d04)**: Scaling `SWE-bench` and agent reinforcement learning rollouts with hundreds of parallel unprivileged `docker:dind` instances and 100-millisecond copy-on-write `.ext4` volume resets.
