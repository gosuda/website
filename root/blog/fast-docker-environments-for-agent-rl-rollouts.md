---
id: 517bf76b20da9b1bac643f4bac1bfd6b
author: Lemon Mint
title: Fast Isolated Docker Environments for AI Agent RL Rollouts with gvx
description: How to securely run hundreds of parallel, strongly isolated unprivileged Docker and Kubernetes environments for SWE-bench and coding agent reinforcement learning rollouts with 100-millisecond copy-on-write ext4lite volume resets.
language: en
date: 2026-11-06T12:02:55.95126412Z
path: /blog/posts/fast-docker-environments-for-agent-rl-rollouts-z48916d04
go_package: z.gfire.dev/gvx
go_repourl: https://github.com/gfire-sigs/gvx.git
category: ai
---

## The Security and Infrastructure Bottleneck in Coding Agent RL Rollouts

Training and evaluating software engineering agents across benchmarks and reinforcement learning (RL) frameworks such as **SWE-bench**, **SWE-ReX**, **R2E-Gym**, **SWE-Gym**, **Verl**, and **OpenHands** requires executing thousands of concurrent rollouts of **untrusted, LLM-generated code**. In real-world software engineering tasks, an agent does not merely edit text files: it runs `docker build`, spins up databases and microservices via `docker compose`, executes integration test suites, and inspects container logs.

At rollout scale (128 to 1,024 concurrent episodes per cluster), existing container infrastructure hits four systemic security and performance bottlenecks:

1. **Host `dockerd` Lock Contention and State Leakage**: Pointing hundreds of concurrent RL workers at a single host `/var/run/docker.sock` gives every agent root-equivalent access to the host daemon, serializes image layer extraction, exhausts `docker0` subnet allocations, triggers `Another app is currently holding the xtables lock` errors in host `iptables`, and leaks orphan containers across episodes.
2. **Privileged DinD Container Escape Risk**: Running standard `docker:dind` containers requires `--privileged` (`CAP_SYS_ADMIN`), which exposes raw host block devices, cgroup release agents, and kernel modules to untrusted RL rollouts.
3. **Hardware Virtualization (`/dev/kvm`) Constraints**: MicroVM runtimes (`Firecracker`, `QEMU`, `Kata Containers`) isolate inner kernels cleanly, but they require `/dev/kvm`. Standard Kubernetes pods without nested virtualization, bare-metal cloud instances where KVM is disabled, and existing `gVisor` (`runsc`) sandboxes cannot launch KVM microVMs.
4. **Slow Episode Reset (`env.reset()`) Latency**: Starting a fresh `docker:dind` environment and pulling or unpacking base images for every RL episode takes 10 to 60 seconds, leaving GPU training workers idle while waiting for environment resets.

**[gvx](https://github.com/gfire-sigs/gvx)** (`rungv` CLI and `z.gfire.dev/gvx/pkg/gvx` Go SDK) provides a **secure, defense-in-depth rollout runtime** that resolves all four bottlenecks. Each rollout environment runs as **a single unprivileged, self-locked user-space process** (isolated by Google's gVisor `Sentry` kernel, `pkg/hostlockdown`, and an in-memory `netstack`) whose entire `/var/lib/docker` state resides inside **a single copy-on-write `.ext4` file (`ext4lite`)**.

> **Zero-Privilege Security for Untrusted Agent Rollouts**: Every `gvx` episode locks down its own host OS process before running guest code (`PR_SET_NO_NEW_PRIVS` + BPF `seccomp` on Linux, Seatbelt `deny default` on macOS, Restricted Token + Job Object on Windows). Even if an RL agent executes `docker run --privileged` or destructive `rm -rf /` commands inside its episode, those operations remain strictly confined to that episode's user-space `Sentry` kernel and ephemeral `.ext4` volume file.

---

## Rollout Architecture: Parallel Unprivileged Instances + Reflink Snapshots

In `gvx`, every rollout worker is an ordinary unprivileged OS process with its own gVisor `Sentry` kernel, its own user-space `netstack` (`10.0.2.15/24`), and its own `ext4lite` block volume (`ep-N.ext4`), while sharing a read-only content-addressable `erofs` OCI image cache across the host page cache.

![Parallel Agent RL Rollout Architecture with gvx](/assets/images/gvx/gvx-rl-rollout-snapshot.png)

### Why Single-File `ext4lite` Volumes Change RL Reset Economics

Because `gvx`'s `ext4lite` filesystem stores an entire JBD2-journaled `ext4` volume (including Docker's `overlayfs` layer directories, `trusted.overlay.*` extended attributes, whiteout device nodes, and image metadata) inside one regular host file, episode lifecycle management becomes a single file operation:

1. **100-Millisecond `env.reset()` via `cp --reflink=always`**:
   - You build a `golden-docker.ext4` file once containing pre-pulled base images (`python:3.11`, `postgres:16`, `redis:7`, or task-specific `SWE-bench` base images).
   - At the start of each RL episode, `cp --reflink=always golden-docker.ext4 ep-0042.ext4` (on XFS, Btrfs, ZFS, or macOS APFS) creates an independent Copy-on-Write clone in under 100 milliseconds without copying underlying disk blocks.
   - Only the 4 KiB blocks dirtied during that specific episode consume additional disk space.
2. **Zero-Residue Teardown (`env.close()`)**:
   - Terminating the `rungv` process (`sb.Close()` in Go) and unlinking `ep-0042.ext4` reclaims 100% of the episode's processes, network sockets, mounts, and storage in ~10 milliseconds. Nothing is ever mounted in the host kernel's mount table or registered in host cgroups.

---

## Step-by-Step Workflow: Golden `.ext4` Snapshot and Fast Episode Reset

You can install `rungv` on each rollout worker node either by downloading the SLSA-attested prebuilt binary from [GitHub Releases](https://github.com/gfire-sigs/gvx/releases) (`gh attestation verify rungv-v0.2.0-linux-amd64.tar.gz --repo gfire-sigs/gvx --signer-workflow gfire-sigs/gvx/.github/workflows/release.yml --deny-self-hosted-runners`) or by running `CGO_ENABLED=0 go install z.gfire.dev/gvx/cmd/rungv@latest`.

### Step 1: Create the Golden `docker:dind` Volume Once

```sh
mkdir -p /srv/rl-snapshots /srv/rl-episodes

# 1. Start a seed docker:dind instance backed by golden-docker.ext4
rungv run \
  -p unix:/srv/rl-snapshots/seed.sock:2375 \
  --mount type=volume,src=/srv/rl-snapshots/golden-docker.ext4,dst=/var/lib/docker \
  -e DOCKER_TLS_CERTDIR= \
  docker:dind --tls=false &
SEED_PID=$!

# 2. Pre-populate base images and build cache inside golden-docker.ext4
export DOCKER_HOST="unix:///srv/rl-snapshots/seed.sock"
until docker info >/dev/null 2>&1; do sleep 0.5; done

docker pull python:3.11-slim
docker pull redis:7-alpine
docker pull postgres:16-alpine

# 3. Gracefully stop the seed instance so ext4lite flushes and compacts the image
kill -INT $SEED_PID
wait $SEED_PID
```

### Step 2: Spawn Parallel RL Episodes in 100 Milliseconds

Each parallel rollout worker clones `golden-docker.ext4` via reflink and boots an isolated `docker:dind` instance with all images already present in `/var/lib/docker`:

```sh
EPISODE_ID="ep-0042"
SOCK="/srv/rl-episodes/${EPISODE_ID}.sock"

# Instant Copy-on-Write clone (~100 ms on XFS/Btrfs/APFS)
cp --reflink=always \
  /srv/rl-snapshots/golden-docker.ext4 \
  "/srv/rl-episodes/${EPISODE_ID}.ext4"

# Launch isolated unprivileged episode with 4 GiB RAM and 2 CPUs
rungv run \
  -m 4g --cpus 2 \
  -p "unix:${SOCK}:2375" \
  --mount "type=volume,src=/srv/rl-episodes/${EPISODE_ID}.ext4,dst=/var/lib/docker" \
  -e DOCKER_TLS_CERTDIR= \
  docker:dind --tls=false &
EP_PID=$!

export DOCKER_HOST="unix://${SOCK}"
until docker info >/dev/null 2>&1; do sleep 0.2; done

# Inside the rollout worker, images are immediately available without network pull
docker images

# Episode teardown after reward calculation (~10 ms)
kill -KILL $EP_PID
rm -f "/srv/rl-episodes/${EPISODE_ID}.ext4" "${SOCK}"
```

---

## Programmatic Rollout Pool Using the `z.gfire.dev/gvx/pkg/gvx` Go SDK

If your RL environment server or evaluation harness is written in Go (or calls a Go rollout daemon over gRPC), you can manage parallel container episodes directly through `z.gfire.dev/gvx/pkg/gvx`:

```go
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"z.gfire.dev/gvx/pkg/gvx"
)

func runEpisode(ctx context.Context, goldenExt4, scratchDir string, epID int) error {
	epExt4 := filepath.Join(scratchDir, fmt.Sprintf("ep-%04d.ext4", epID))
	defer os.Remove(epExt4)

	// 1. Reflink clone the golden ext4lite volume
	cpCmd := exec.CommandContext(ctx, "cp", "--reflink=always", goldenExt4, epExt4)
	if err := cpCmd.Run(); err != nil {
		return fmt.Errorf("clone golden volume: %w", err)
	}

	// 2. Apply per-episode wall-clock timeout
	epCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	var stdout, stderr bytes.Buffer
	sb, err := gvx.New(epCtx, gvx.Config{
		Platform:               gvx.PlatformAuto,
		NetworkMode:            gvx.NetModeNAT,
		NetRaw:                 true,
		AllowPacketSocketWrite: true,
		AllowSUID:              true,
		OCIImage:               "docker:dind",
		Entrypoint:             []string{"sh", "-c"},
		EntrypointSet:          true,
		Cmd: []string{
			"dockerd-entrypoint.sh --tls=false >/var/log/dockerd.log 2>&1 & " +
				"until docker info >/dev/null 2>&1; do sleep 0.2; done && " +
				"docker run --rm python:3.11-slim python -c 'print(6 * 7)'",
		},
		Envv: []string{"DOCKER_TLS_CERTDIR="},
		Mounts: []gvx.Mount{
			{
				Kind:      gvx.MountVolume,
				ImagePath: epExt4,
				GuestPath: "/var/lib/docker",
			},
		},
		Resources: gvx.Resources{
			MemoryBytes: 4 << 30, // 4 GiB cap
			CPUMillis:   2000,    // 2.0 CPUs
		},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		return fmt.Errorf("init episode %d: %w", epID, err)
	}
	defer sb.Close()

	exitCode, err := sb.Run(epCtx)
	if err != nil {
		return fmt.Errorf("run episode %d: %w", epID, err)
	}

	log.Printf("episode %d finished: exit=%d stdout=%q", epID, exitCode, stdout.String())
	return nil
}

func main() {
	ctx := context.Background()
	if err := runEpisode(ctx, "/srv/rl-snapshots/golden-docker.ext4", "/srv/rl-episodes", 1); err != nil {
		log.Fatal(err)
	}
}
```

---

## Comparison of Agent RL Sandbox Runtimes

| Metric / Requirement                                      | Shared Host `dockerd`             | Privileged `docker:dind` | Firecracker / KVM MicroVM      | Standard `gVisor` (`runsc`)       | `gvx` (`rungv` / `pkg/gvx`)                 |
| :-------------------------------------------------------- | :-------------------------------- | :----------------------- | :----------------------------- | :-------------------------------- | :------------------------------------------ |
| **Host `root` or `--privileged` required**                | Yes (`docker.sock` = root)        | Yes (`--privileged`)     | Needs `/dev/kvm` + `TAP` setup | Yes (or `userns` + `mount`)       | **No (single unprivileged process)**        |
| **Works inside unprivileged K8s Pods / existing `runsc`** | No                                | No                       | No (no `/dev/kvm`)             | Fails (`EBUSY` on nested seccomp) | **Yes (automatic futex loop fallback)**     |
| **Supports inner `docker build` & `docker compose`**      | Shares host daemon                | Yes                      | Yes                            | Fails without host DinD tuning    | **Yes (`overlayfs` on `ext4lite`)**         |
| **Per-episode network isolation**                         | Shared `docker0` / `xtables` lock | Separate namespace       | Separate VM NIC                | Separate netns                    | **Isolated in-process `netstack`**          |
| **Warm episode reset latency (`env.reset()`)**            | 5s to 30s (container/vol prune)   | 10s to 60s (cold daemon) | ~150 ms (VM snapshot)          | 5s to 20s                         | **~100 ms (`cp --reflink=always` `.ext4`)** |
| **Cross-platform local dev (Windows / macOS / Linux)**    | Requires Linux VM                 | Requires Linux VM        | Linux KVM only                 | Linux only                        | **Native on Windows, macOS, and Linux**     |

---

## Further Reading in This Series

1. **[gvx: Running Linux Containers, Docker, and Kubernetes in One Unprivileged Process](/blog/posts/gvx-cross-platform-linux-containers-z48916d01)**: Complete architectural walkthrough of `gvx` across Windows (`whpx`, `systrap`), macOS (`hvf`), and Linux (`systrap`).
2. **[Running Docker, docker build, and docker compose Without Root or User Namespaces](/blog/posts/run-docker-without-root-or-userns-gvx-z48916d02)**: Quick 3-step bootstrap and troubleshooting guide when `sudo`, `unshare(CLONE_NEWUSER)`, `mount(2)`, and `/dev/fuse` are blocked.
3. **[Running Docker-in-Docker Inside gVisor and Nested Sandboxes with gvx](/blog/posts/docker-in-docker-inside-gvisor-with-gvx-z48916d03)**: Deep dive into why `docker.io` and nested `runsc` fail inside `gVisor` (`runsc`) and how `gvx` falls back to the `_RUN_SYSCALL_LOOP` futex loop on `EBUSY`.
4. **[Fast Isolated Docker Environments for AI Agent RL Rollouts with gvx](/blog/posts/fast-docker-environments-for-agent-rl-rollouts-z48916d04)** (this post)
