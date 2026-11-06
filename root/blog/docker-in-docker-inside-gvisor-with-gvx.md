---
id: 83fb2c108243a7df80db3a6e78290daf
author: Lemon Mint
title: Running Docker-in-Docker Inside gVisor and Nested Sandboxes with gvx
description: Why installing docker.io or nesting runsc inside a gVisor container fails across modprobe, iptables-nft, overlay2, cgroups, and seccomp EBUSY, and how gvx securely runs nested docker:dind inside an isolated user-space kernel without --privileged.
language: en
date: 2026-11-06T12:02:55.931762327Z
path: /blog/posts/docker-in-docker-inside-gvisor-with-gvx-z48916d03
go_package: z.gfire.dev/gvx
go_repourl: https://github.com/gfire-sigs/gvx.git
category: lowlevel
---

## Quick Solution: Securely Running `docker:dind` Inside an Unprivileged `gVisor` (`runsc`) Sandbox

Cloud AI agent platforms, Kubernetes `agent-sandbox` deployments (`RuntimeClass: gvisor`), and reinforcement learning evaluation harnesses isolate untrusted workloads inside **gVisor (`runsc`)** containers specifically to protect the host kernel. When a workload inside a default unprivileged `runsc` container tries to install `docker.io` or `docker-ce` via `apt-get` and start `dockerd`, it fails even if it has `root` (`UID 0`) inside the container.

With **[gvx](https://github.com/gfire-sigs/gvx)** (`rungv`), you can **securely** run `docker:dind`, `docker build`, and `docker compose` directly inside an unprivileged `gVisor` (`runsc`) container (or nest `gvx` inside `gvx`) **without weakening the outer `RuntimeClass`, exposing host sockets, or granting `--privileged`**.

> **Preserving and Strengthening Outer Sandbox Isolation**: Running `rungv` inside `runsc` does not bypass a single restriction enforced by the outer `gVisor` sandbox. Instead, `gvx` creates a second, nested user-space gVisor `Sentry` kernel with its own `hostlockdown` (`PR_SET_NO_NEW_PRIVS` + `seccomp`), its own in-memory `netstack`, and a single-file `ext4lite` volume. Inner `dockerd` and `runc` containers are double-sandboxed: their system calls, mounts, and raw packets never leave the inner `gvx` user-space kernel.

```sh
# Inside an existing gVisor (runsc) container or Kubernetes Pod:
# Install rungv from a SLSA-attested GitHub release archive or via go install:
CGO_ENABLED=0 go install z.gfire.dev/gvx/cmd/rungv@latest

mkdir -p ~/.gvx
rungv run \
  -p unix:~/.gvx/docker.sock:2375 \
  -v gvx-docker:/var/lib/docker \
  -e DOCKER_TLS_CERTDIR= \
  docker:dind --tls=false &

# Run standard Docker builds and containers against the inner gvx socket
export DOCKER_HOST="unix://$HOME/.gvx/docker.sock"
until docker info >/dev/null 2>&1; do sleep 0.5; done
docker info --format '{{.Driver}} {{.OperatingSystem}}'
# Output: overlayfs Alpine Linux ... (containerized)

docker build -t nested-app:test .
docker run --rm nested-app:test
```

---

## Five Reasons Why Direct Docker Installation Fails Inside `gVisor` (`runsc`)

When a user or AI coding agent runs `apt-get install -y docker.io` (or `docker-ce`) inside a standard `runsc` container and attempts to start `dockerd`, it hits five sequential technical barriers.

### 1. Kernel Module Loading (`modprobe`) Against Virtual Kernel `4.4.0`

During package installation and `dockerd` startup, initialization scripts run `modprobe` to load `overlay`, `br_netfilter`, `ip_tables`, `nf_nat`, and `xt_conntrack`. The outer gVisor Sentry blocks `init_module(2)` and `finit_module(2)` and reports a virtual kernel version (`4.4.0`) that has no `/lib/modules/4.4.0` directory:

```text
modprobe: FATAL: Module overlay not found in directory /lib/modules/4.4.0
modprobe: FATAL: Module br_netfilter not found in directory /lib/modules/4.4.0
modprobe: FATAL: Module ip_tables not found in directory /lib/modules/4.4.0
modprobe: FATAL: Module nf_nat not found in directory /lib/modules/4.4.0
```

### 2. `iptables-nft` (`nf_tables` Netlink) and `docker0` Bridge Failures

On Ubuntu 22.04, Ubuntu 24.04, and Debian 11/12, `apt-get install docker.io` selects `iptables-nft` (backed by the `nf_tables` netlink subsystem) as the default `/usr/sbin/iptables` alternative. In a standard gVisor container without network admin privileges or `nf_tables` netlink support, `dockerd` crashes while initializing the `DOCKER` NAT chain and the `docker0` bridge:

```text
failed to start daemon: Error initializing network controller: error obtaining controller instance: failed to create NAT chain DOCKER: iptables failed: iptables -t nat -N DOCKER
iptables v1.8.10 (nf_tables): Could not fetch rule set generation id: Invalid argument
iptables v1.8.10 (legacy): can't initialize iptables table 'nat': Table does not exist (do you need to insmod?)
iptables: No chain/target/match by that name.
Unable to enable SKIP DNAT rule: (iptables failed: iptables --wait -t nat -I DOCKER -i docker0 -j RETURN: iptables: No chain/target/match by that name.)
Error initializing network controller: error obtaining controller instance: failed to create bridge "docker0": operation not permitted
```

### 3. `overlay2` Failure on `gofer` / `9p` Rootfs and `vfs` Disk Exhaustion

Unless the outer `runsc` administrator explicitly pre-configured a dedicated `tmpfs` or filestore mount at `/var/lib/docker`, the container rootfs is backed by gVisor's `gofer` (`9p`/`lisafs`) or a nested overlay mount. Mounting `overlay2` on top of `/var/lib/docker` fails:

```text
level=error msg="[graphdriver] prior storage driver overlay2 failed: driver not supported"
failed to mount overlay: invalid argument
Error starting daemon: error initializing graphdriver: driver not supported: overlay2
```

When users or agents attempt to bypass this error by starting `dockerd --storage-driver=vfs`, Docker disables Copy-on-Write layer sharing completely. Every `RUN` instruction in a `Dockerfile` performs a full recursive copy (`dirCopy`) of the entire root filesystem, multiplying a 500 MB image across 10 build steps into 5 GB of writes and terminating the sandbox with `no space left on device` or an I/O timeout.

### 4. Nested `runc` Cgroup and `/proc` Mount Rejections

Even if `dockerd` is forced to start with `dockerd --iptables=false --bridge=none --storage-driver=vfs`, running `docker run` or `docker build` invokes `runc` to create the inner container. In a default unprivileged gVisor container, `runc` fails when mounting cgroups, remounting `/proc`, or loading inner seccomp filters:

```text
docker: Error response from daemon: failed to create task for container: failed to create shim task: OCI runtime create failed: runc create failed: unable to start container process:
error during container init: error mounting "cgroup" to rootfs at "/sys/fs/cgroup": mount cgroup:/sys/fs/cgroup/cpuset (via /proc/self/fd/6), flags: 0xf, data: cpuset: operation not permitted
cgroups: cgroup mountpoint does not exist: unknown
runc create failed: systemd not running on this host: cannot use systemd cgroup manager
error during container init: error mounting "proc" to rootfs at "/proc": mount proc:/proc (via /proc/self/fd/6), flags: 0xe: operation not permitted
error during container init: error loading seccomp filter into kernel: operation not permitted
```

### 5. Nested `runsc` Fails with `SECCOMP_FILTER_FLAG_NEW_LISTENER` (`EBUSY`)

If a user tries to run another `runsc` instance inside `runsc` using the `systrap` platform, initialization aborts immediately:

```text
creating subprocess: sysmsg: failed to initialize thread: failed to install seccomp notify rules: device or resource busy
```

Both the Linux kernel and the gVisor Sentry allow **at most one** `SECCOMP_FILTER_FLAG_NEW_LISTENER` (`SECCOMP_RET_USER_NOTIF`) listener per task hierarchy. Because the outer sandbox already holds that listener slot, the inner `seccomp(2)` call returns `EBUSY` (`device or resource busy`).

---

## How `gvx` Solves Nested Execution Inside `gVisor`

`gvx` addresses all five barriers inside its own user-space kernel and platform layer, enabling `rungv` to run inside an outer `gVisor` (`runsc`) container or even recursively inside another `gvx` container (`gvx` in `gvx`).

![Nested gvx Execution Inside Outer gVisor](/assets/images/gvx/gvx-nested-seccomp-futex.png)

### 1. Automatic `systrap` Fallback from `EBUSY` to the `_RUN_SYSCALL_LOOP` Futex Loop

In Linux `systrap`, each stub subprocess runs two kinds of threads:

- **Guest execution threads (`sysmsg`)**: Trap guest system calls via `SECCOMP_RET_TRAP` (`SIGSYS`) and binary-patched `jmp rel32` trampolines (`syshandler`), exchanging register state with the Sentry over shared memory and `SYS_FUTEX`.
- **A stub `syscallThread` (`syscall_thread.go`)**: Executes address-space system calls (`mmap`, `munmap`, `clone`) inside the stub subprocess on behalf of the Sentry.

In upstream gVisor, `syscallThread` supports two wake-up modes in `stub_amd64.s` and `stub_arm64.s`:

1. **Seccomp User-Notify Loop (`_RUN_SECCOMP_LOOP = 6`, when `t.seccompNotify != nil`)**: During `syscallThread.init()`, `t.installSeccompNotify()` installs a `SECCOMP_RET_USER_NOTIF` listener (`seccomp(SECCOMP_SET_MODE_FILTER, SECCOMP_FILTER_FLAG_NEW_LISTENER, stubSyscallRules)`) so the Sentry can dispatch requests via `ioctl(SECCOMP_IOCTL_NOTIF_SEND)` and `ioctl(SECCOMP_IOCTL_NOTIF_RECV)`.
2. **Futex Syscall Loop (`_RUN_SYSCALL_LOOP = 5`, when `t.seccompNotify == nil`)**: The `syscallThread` runs `syscall_loop` (`stub_amd64.s` / `stub_arm64.s`), sleeping on `SYS_FUTEX` (`FUTEX_WAIT`) over the shared 2-page `syscallSentryMessage` (`PROT_READ` in the stub) and `syscallStubMessage` (`PROT_READ | PROT_WRITE` in the stub) region.

Upstream gVisor unconditionally calls `t.installSeccompNotify()` whenever `seccompNotifyIsSupported` is true, and aborts stub creation if `seccomp(SECCOMP_SET_MODE_FILTER, SECCOMP_FILTER_FLAG_NEW_LISTENER, ...)` returns `EBUSY`.

In `gvx` (`pkg/sentry/platform/systrap/seccomp_notify_xplat.go` and `syscall_thread.go`), `syscallThread.init()` calls `t.installSeccompNotifyIfFree()`:

```go
// pkg/sentry/platform/systrap/seccomp_notify_xplat.go
var seccompNotifyBusy atomic.Bool

func (t *syscallThread) installSeccompNotifyIfFree() (*os.File, error) {
	if seccompNotifyBusy.Load() {
		return nil, nil
	}
	f, err := t.installSeccompNotify()
	if errors.Is(err, unix.EBUSY) {
		if seccompNotifyBusy.CompareAndSwap(false, true) {
			log.Warningf("Another seccomp listener exists in the filter chain of the host process. " +
				"The stub threads use the slower futex loop instead of seccomp-unotify.")
		}
		return nil, nil
	}
	return f, err
}
```

When `installSeccompNotify()` returns `unix.EBUSY` because an outer `gVisor` (`runsc`) or outer `gvx` sandbox already holds the process filter chain's single listener slot:

- `installSeccompNotifyIfFree()` sets `seccompNotifyBusy` (`atomic.Bool`), logs one warning for the process, and leaves `t.seccompNotify == nil` for that stub and all subsequent stubs.
- In `syscallThread.detach()` (`syscall_thread_amd64.go` and `syscall_thread_arm64.go`), `t.seccompNotify == nil` sets `regs.Rbx = _RUN_SYSCALL_LOOP` (`5`) on `amd64` (or `regs.Regs[9] = _RUN_SYSCALL_LOOP` on `arm64`) instead of `_RUN_SECCOMP_LOOP` (`6`).
- In `syscallThread.syscall()` (`syscall_thread.go`), the inner Sentry dispatches stub syscalls purely through shared memory and `SYS_FUTEX`:

```go
// pkg/sentry/platform/systrap/syscall_thread.go
// Notify the syscall thread about a new syscall request.
atomic.AddUint32(&sentryMsg.state, 1)
futexWakeUint32(&sentryMsg.state)

// Wait for reply.
futexWaitWake(&sentryMsg.state, atomic.LoadUint32(&sentryMsg.state))
```

Because `SECCOMP_RET_TRAP` filters on guest threads stack arbitrarily deep and `syscallThread` falls back to `_RUN_SYSCALL_LOOP` without requesting a second user-notification listener, `gvx` runs cleanly inside `runsc` and supports recursive nesting (`gvx` inside `gvx`).

### 2. Native `overlayfs` on Single-File `ext4lite` Volumes

To avoid mounting `overlayfs` on top of the outer gVisor container's `gofer` (`9p`/`lisafs`) root filesystem, `-v gvx-docker:/var/lib/docker` backs `/var/lib/docker` with `gvx`'s internal **`ext4lite`** filesystem stored in a single regular file (`gvx-docker.ext4`).

To outer `runsc`, `gvx-docker.ext4` is just one regular file accessed via `pread64(2)` and `pwrite64(2)`. Inside `gvx`, `ext4lite` provides a JBD2-journaled `ext4` filesystem with `trusted.overlay.*` and `security.capability` extended attributes, character device whiteout nodes (`0:0`), `d_type` directory entries, and `RENAME_EXCHANGE`, allowing the inner `dockerd` to use its native `overlayfs` storage driver.

### 3. Self-Contained `netstack` with `docker0` and `iptables-legacy`

Inside `gvx`, `dockerd` does not interact with the outer `runsc` container's network interfaces or netfilter tables. Instead:

- `dockerd` creates its `docker0` bridge and `veth` pairs inside `gvx`'s own user-space `netstack`.
- All `iptables-legacy` rules (`-t nat -N DOCKER`, `MASQUERADE`, `conntrack`, `addrtype`) execute against `gvx`'s internal netfilter implementation.
- Nested `runc` containers mount their `/proc`, `/sys`, and `/sys/fs/cgroup` hierarchies inside `gvx`'s in-memory `VFS2` mount table, never calling the outer `runsc` `mount(2)` syscall.

---

## Verifying Nested `docker:dind` Inside `gVisor` (or `gvx` in `gvx`)

You can verify both scenarios (`rungv` inside `runsc`, or `rungv` inside `rungv`) with the following commands:

```sh
# Inside an outer gVisor (runsc) container or an outer rungv container:
mkdir -p ~/.gvx
rungv run \
  -p unix:~/.gvx/docker.sock:2375 \
  -v gvx-docker:/var/lib/docker \
  -e DOCKER_TLS_CERTDIR= \
  docker:dind --tls=false &
DIND_PID=$!

export DOCKER_HOST="unix://$HOME/.gvx/docker.sock"
until docker info >/dev/null 2>&1; do sleep 0.5; done

# Verify that the inner Docker daemon is using native overlayfs (not vfs)
docker info --format 'Storage Driver: {{.Driver}}'
# Output: Storage Driver: overlayfs

# Run a multi-layer build and execute a nested container
mkdir -p /tmp/nested-test && cd /tmp/nested-test
cat <<'EOF' > Dockerfile
FROM alpine:3.20
RUN adduser -D -u 1001 appuser && mkdir -p /app && chown appuser:appuser /app
USER appuser
WORKDIR /app
RUN echo "nested-build-ok" > status.txt
CMD ["cat", "/app/status.txt"]
EOF

docker build -t nested-test:latest .
docker run --rm nested-test:latest
# Output: nested-build-ok

kill -INT $DIND_PID
```

---

## Further Reading in This Series

1. **[gvx: Running Linux Containers, Docker, and Kubernetes in One Unprivileged Process](/blog/posts/gvx-cross-platform-linux-containers-z48916d01)**: Overview of `gvx` across Windows (`whpx`, `systrap`), macOS (`hvf`), and Linux (`systrap`).
2. **[Running Docker, docker build, and docker compose Without Root or User Namespaces](/blog/posts/run-docker-without-root-or-userns-gvx-z48916d02)**: How `gvx` runs `docker:dind` when `root`, `sudo`, `unshare(CLONE_NEWUSER)`, `mount(2)`, and `/dev/fuse` are blocked.
3. **[Running Docker-in-Docker Inside gVisor and Nested Sandboxes with gvx](/blog/posts/docker-in-docker-inside-gvisor-with-gvx-z48916d03)** (this post)
4. **[Fast Isolated Docker Environments for AI Agent RL Rollouts with gvx](/blog/posts/fast-docker-environments-for-agent-rl-rollouts-z48916d04)**: How to use `gvx` and single-file `ext4lite` copy-on-write snapshots to run hundreds of parallel `SWE-bench` and RL rollout environments with 100-millisecond resets.
