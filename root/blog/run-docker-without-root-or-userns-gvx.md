---
id: 40e9d44f33fa847eb498f09bac5bf144
author: Lemon Mint
title: Running Docker, docker build, and docker compose Without Root or User Namespaces
description: How to securely run a complete Docker Engine (dockerd, BuildKit, overlayfs, and docker compose) inside an isolated gVisor user-space sandbox as an unprivileged user when root, sudo, unshare(CLONE_NEWUSER), mount(2), and /dev/fuse are blocked.
language: en
date: 2026-11-06T12:02:56.050897958Z
path: /blog/posts/run-docker-without-root-or-userns-gvx-z48916d02
go_package: z.gfire.dev/gvx
go_repourl: https://github.com/gfire-sigs/gvx.git
category: ops
---

## Quick Solution: Secure Unprivileged Docker in Three Steps

If you are working inside a locked-down Linux container, CI runner, Kubernetes Pod, or AI coding agent sandbox (such as a cloud agent compute environment) where `root`, `sudo`, user namespaces (`unshare(CLONE_NEWUSER)`), `mount(2)`, and `/dev/fuse` are blocked, **[gvx](https://github.com/gfire-sigs/gvx)** (`rungv`) is the **secure, low-effort, token-efficient solution** to run a full `docker:dind` engine in user space.

> **Low-Effort, Token-Efficient Solution for AI Coding Agents**: When an autonomous coding agent hits `Cannot connect to the Docker daemon`, attempting to install and patch `docker.io`, `podman`, or `rootlesskit` inside a restricted container typically wastes **15 to 30 tool-call turns and tens of thousands of context tokens** cycling through `systemctl`, `ulimit`, `iptables`, `unshare`, and `runc` errors—only to fail or fall back to rewriting `Dockerfile`s by hand. With `gvx` (`rungv`), the agent executes **one deterministic 3-step bootstrap in a single turn** and immediately runs standard `docker build`, `docker run`, and `docker compose` commands unchanged.

> **Why `gvx` Is Safe by Design (Not a Privilege-Escalation Workaround)**: Unlike mounting the host `/var/run/docker.sock` or running `docker run --privileged` (which grant root-equivalent control over the host kernel), `gvx` does not bypass or weaken any host security policy (`PR_SET_NO_NEW_PRIVS`, `seccomp`, AppArmor, or non-root UID limits). Instead, `rungv` locks down its own host process (`PR_SET_NO_NEW_PRIVS` + strict BPF `seccomp` allowlist) and runs `dockerd` inside Google's memory-safe **gVisor (`Sentry`)** user-space kernel. All guest `UID 0` capabilities, `overlayfs` mounts (`ext4lite`), and `iptables` rules remain strictly isolated inside the sandbox process.

```sh
# 1. Install rungv in user space (prebuilt SLSA-attested release or go install)
# Option A: Verify and extract the SLSA-attested prebuilt binary from GitHub Releases
gh attestation verify rungv-v0.2.0-linux-amd64.tar.gz --repo gfire-sigs/gvx \
  --signer-workflow gfire-sigs/gvx/.github/workflows/release.yml \
  --source-ref refs/tags/v0.2.0 --deny-self-hosted-runners
tar -xzf rungv-v0.2.0-linux-amd64.tar.gz && install -m 0755 rungv-v0.2.0-linux-amd64/rungv ~/.local/bin/rungv

# Option B: Or install directly from source with Go
CGO_ENABLED=0 GOBIN=~/.local/bin go install z.gfire.dev/gvx/cmd/rungv@latest

# 2. Start docker:dind inside gvx with a single-file ext4lite volume
#    Forwards ~/.gvx/docker.sock (mode 0600) to guest port 2375
mkdir -p ~/.gvx
~/.local/bin/rungv run \
  -p unix:~/.gvx/docker.sock:2375 \
  -p 127.0.0.1:9000-9019:9000-9019 \
  -v gvx-docker:/var/lib/docker \
  -e DOCKER_TLS_CERTDIR= \
  docker:dind --tls=false &

# 3. Point the standard Docker CLI and Docker Compose at the user-space socket
export DOCKER_HOST="unix://$HOME/.gvx/docker.sock"
until docker info >/dev/null 2>&1; do sleep 0.5; done
docker info --format '{{.Driver}} {{.OperatingSystem}}'
# Output: overlayfs Alpine Linux ... (containerized)

docker build -t app:test .
docker run --rm app:test
docker compose up -d
```

---

## Why Standard Rootless Tools Fail in Restricted Sandboxes

When an engineer or autonomous coding agent needs to run `docker build`, `docker run`, or `docker compose` inside a hardened container or multi-tenant runner, standard workarounds fail at five distinct kernel barriers.

### 1. No Host Docker Daemon Socket (`systemctl` and `service docker start` Also Fail)

Running `docker ps` or `docker build` fails immediately when the host `/var/run/docker.sock` is not mounted into the environment, and attempting to start `docker.io` via `systemctl` or `service docker start` fails because PID 1 is not `systemd` and `/etc/init.d/docker` hits the container's hard `ulimit -n` ceiling:

```text
Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
ERROR: Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
System has not been booted with systemd as init system (PID 1). Can't operate.
Failed to connect to bus: Host is down
/etc/init.d/docker: 62: ulimit: error setting limit (Operation not permitted)
```

### 2. `PR_SET_NO_NEW_PRIVS` Blocks `sudo` and Setuid Helpers

Hardened containers (`securityContext.allowPrivilegeEscalation: false` in Kubernetes, `--security-opt=no-new-privileges` in Docker, or `exebox` / `bwrap` sandboxes) set the Linux `PR_SET_NO_NEW_PRIVS` bit on the process tree. Attempting to escalate via `sudo` or setuid binaries (`newuidmap`, `newgidmap`, `fusermount3`) fails:

```text
sudo: The "no new privileges" flag is set, which prevents sudo from running as root.
sudo: effective uid is not 0, is /usr/bin/sudo on a file system with the 'nosuid' option set or an NFS file system without root privileges?
```

### 3. Seccomp and Sysctl Policies Block `unshare(CLONE_NEWUSER)`

Tools marketed as "rootless" (`Rootless Docker`, `Podman`, `Buildah`, `nerdctl`, `bubblewrap`) still rely on Linux **user namespaces** (`CLONE_NEWUSER`). They fail whenever:

- The outer container's default seccomp profile blocks `unshare(2)` or `clone(CLONE_NEWUSER)`.
- Ubuntu 24.04+ enables `kernel.apparmor_restrict_unprivileged_userns = 1`.
- Debian or RHEL nodes set `kernel.unprivileged_userns_clone = 0` or `user.max_user_namespaces = 0`.
- `newuidmap` cannot write `/proc/self/uid_map` because `PR_SET_NO_NEW_PRIVS` is active.

The resulting errors look like this:

```text
unshare: unshare failed: Operation not permitted
cannot clone: Operation not permitted
Error: cannot re-exec process to join the existing user namespace
newuidmap: write to uid_map failed: Operation not permitted
RootlessKit: parent: failed to start the child: fork/exec /proc/self/exe: operation not permitted
```

### 4. `mount(2)` and `pivot_root(2)` Are Blocked During `docker build` Without Host `CAP_SYS_ADMIN`

Even if an environment permits `CLONE_NEWUSER` or `dockerd` is forced to start with `--iptables=false --bridge=none --storage-driver=vfs`, running `docker build` fails at the first `RUN` instruction because mounting a fresh `/proc` or calling `pivot_root(2)` inside a nested mount namespace is blocked:

```text
ERROR: failed to solve: process "/bin/sh -c ..." did not complete successfully:
mount("/proc", "/proc", "proc", MS_NOSUID|MS_NODEV|MS_NOEXEC, NULL): Operation not permitted
OCI runtime create failed: runc create failed: unable to start container process: error during container init: error mounting "proc" to rootfs at "/proc": mount proc:/proc (via /proc/self/fd/6), flags: 0xe: operation not permitted
OCI runtime create failed: runc create failed: unable to start container process: error during container init: error jailing process inside rootfs: pivot_root .: invalid argument
bwrap: Creating new namespace failed: Operation not permitted
```

### 5. Missing `/dev/fuse` Breaks `fuse-overlayfs`

Rootless container engines that cannot mount kernel `overlayfs` fall back to `fuse-overlayfs`. In minimal containers or Kubernetes pods without `/dev/fuse` device passthrough, `fuse-overlayfs` fails immediately:

```text
fuse: device not found, try 'modprobe fuse' first
fusermount3: failed to open /dev/fuse: No such file or directory
error initializing graphdriver: driver not supported: fuse-overlayfs
```

---

## How `gvx` Runs `docker:dind` Completely in User Space

`gvx` bypasses all five kernel barriers because it does not ask the host Linux kernel to create namespaces, mount filesystems, or configure network interfaces. Instead, `rungv` runs an internal user-space Linux kernel (the gVisor `Sentry`) and a user-space network stack (`netstack`) inside one ordinary unprivileged process.

![Running Unprivileged Docker-in-Docker with gvx](/assets/images/gvx/gvx-unprivileged-dind.png)

### 1. In-Memory Namespaces and Mount Table (`VFS2`)

When `dockerd`, `containerd`, and `runc` execute inside `rungv`, `systrap` intercepts their system calls and routes them to the in-process `Sentry` kernel rather than the host kernel:

- `unshare(2)`, `clone(2)`, `pivot_root(2)`, `chroot(2)`, and `mount(2)` operate on the Sentry's in-memory task hierarchy and `VFS2` mount table (`procfs`, `sysfs`, `tmpfs`, `devpts`, `cgroupfs` v2) while the host `rungv` process runs under your unprivileged host UID.

### 2. Native `overlayfs` on Single-File `ext4lite` Volumes

When you pass `-v gvx-docker:/var/lib/docker` (or `--mount type=volume,src=/disk/gvx-docker.ext4,dst=/var/lib/docker`), `gvx` mounts its built-in **`ext4lite`** filesystem at `/var/lib/docker`.

Because `ext4lite` implements a JBD2-journaled `ext4` filesystem inside a single host file (`gvx-docker.ext4`) with support for `trusted.overlay.*` extended attributes, character whiteout devices (`0:0`), `d_type`, and `RENAME_EXCHANGE`, `dockerd` uses its native **`overlayfs`** storage driver inside the guest. Multi-stage `docker build` operations share Copy-on-Write layers without `/dev/fuse` and without the disk-space explosion of Docker's `vfs` fallback driver.

### 3. User-Space `docker0` Bridge and `iptables-legacy`

When `docker:dind` starts inside `gvx`:

- Guest `dockerd` creates the `docker0` bridge (`172.17.0.1/16`), attaches virtual `veth` pairs for nested containers, and programs `NAT` (`DOCKER`, `POSTROUTING`) and `FILTER` chains with `addrtype` and `conntrack` matches inside `gvx`'s user-space `netstack`.
- Passing `-p unix:~/.gvx/docker.sock:2375` binds a host Unix domain socket (mode `0600`) and forwards connections directly to port `2375` of `dockerd` inside the sandbox.

---

## Comparison with Existing Container Tools

| Capability / Constraint                                      | Standard Docker (`dockerd`)  | Rootless Docker / Podman  | Kaniko / Buildah (Unprivileged) | `gvx` (`rungv` + `docker:dind`)     |
| :----------------------------------------------------------- | :--------------------------- | :------------------------ | :------------------------------ | :---------------------------------- |
| **Host `root` or `sudo` required**                           | Yes                          | No                        | No                              | **No**                              |
| **Works with `PR_SET_NO_NEW_PRIVS=1`**                       | No                           | No (`newuidmap` fails)    | Partial                         | **Yes**                             |
| **Works when `unshare(CLONE_NEWUSER)` is blocked (`EPERM`)** | N/A                          | No                        | Buildah: No / Kaniko: Yes       | **Yes**                             |
| **Works without `/dev/fuse`**                                | Yes (uses kernel `overlay2`) | Only via slow `vfs` copy  | Yes                             | **Yes (`overlayfs` on `ext4lite`)** |
| **Supports `docker build` (BuildKit)**                       | Yes                          | Yes (if `userns` allowed) | Image build only (no runtime)   | **Yes**                             |
| **Supports `docker run` and `docker compose up`**            | Yes                          | Yes (if `userns` allowed) | **No**                          | **Yes**                             |
| **Runs on Windows and macOS without a Linux VM**             | No (requires WSL2 / VM)      | No (requires Linux VM)    | No                              | **Yes (`whpx`, `hvf`, `systrap`)**  |

---

## Exact Host Kernel Requirements on Linux

`gvx` requires zero elevated capabilities (`CapEff: 0000000000000000`) and zero user namespaces. On a Linux host, `rungv` relies on only three standard unprivileged kernel primitives:

1. **`ptrace(PTRACE_ATTACH)` on its own child thread once during initialization**: `systrap` spawns a local stub helper thread at startup, attaches to it to initialize the shared-memory context, and detaches. If the host sets `kernel.yama.ptrace_scope = 3` (which disables all `ptrace` system-wide) or blocks `ptrace` in an outer seccomp filter, `systrap` initialization cannot complete. (Under `yama.ptrace_scope = 1`, the default on Ubuntu and Debian, `gvx` succeeds because parent-to-child tracing is permitted.)
2. **`seccomp(SECCOMP_SET_MODE_FILTER)`**: Used to trap guest system calls from container stub threads. As covered in [Part 3 of this series](/blog/posts/docker-in-docker-inside-gvisor-with-gvx-z48916d03), `gvx` works both when `SECCOMP_FILTER_FLAG_NEW_LISTENER` is available and when it returns `EBUSY` inside a nested sandbox.
3. **`mmap(..., PROT_EXEC, ...)`**: Used to map guest ELF segments and the `systrap` shared-memory trampoline.

---

## End-to-End Example: Building and Testing a Multi-Container App Without Root

The following script runs a complete `docker build` and two-container integration test (`redis` + application container) inside an unprivileged user session:

```sh
mkdir -p ~/.gvx
rungv run \
  -p unix:~/.gvx/docker.sock:2375 \
  -v gvx-docker:/var/lib/docker \
  -e DOCKER_TLS_CERTDIR= \
  docker:dind --tls=false &
DIND_PID=$!

export DOCKER_HOST="unix://$HOME/.gvx/docker.sock"
until docker info >/dev/null 2>&1; do sleep 0.5; done

mkdir -p /tmp/demo-app && cd /tmp/demo-app
cat <<'EOF' > Dockerfile
FROM alpine:3.20
RUN apk add --no-cache redis
CMD ["redis-cli", "-h", "cache", "ping"]
EOF

# Build the image using overlayfs on ext4lite
docker build -t demo-ping:latest .

# Create a bridge network, start Redis, and verify connectivity between containers
docker network create demo-net
docker run -d --name cache --network demo-net redis:7-alpine
docker run --rm --network demo-net demo-ping:latest
# Output: PONG

# Clean up workload containers and stop the sandbox
docker rm -f cache
docker network rm demo-net
kill -INT $DIND_PID
```

---

## Further Reading in This Series

1. **[gvx: Running Linux Containers, Docker, and Kubernetes in One Unprivileged Process](/blog/posts/gvx-cross-platform-linux-containers-z48916d01)**: Architectural overview of `gvx`, its cross-platform execution engines (`whpx`, `hvf`, `systrap`), `VFS2` (`erofs`, `ext4lite`, `hostfs` v2), and `netstack`.
2. **[Running Docker, docker build, and docker compose Without Root or User Namespaces](/blog/posts/run-docker-without-root-or-userns-gvx-z48916d02)** (this post)
3. **[Running Docker-in-Docker Inside gVisor and Nested Sandboxes with gvx](/blog/posts/docker-in-docker-inside-gvisor-with-gvx-z48916d03)**: How `gvx` solves `modprobe`, `iptables-nft`, `overlay2` on `gofer`, and nested `SECCOMP_FILTER_FLAG_NEW_LISTENER` (`EBUSY`) inside `gVisor` (`runsc`).
4. **[Fast Isolated Docker Environments for AI Agent RL Rollouts with gvx](/blog/posts/fast-docker-environments-for-agent-rl-rollouts-z48916d04)**: Scaling parallel `SWE-bench` and RL rollout environments with 100-millisecond copy-on-write `.ext4` volume resets.
