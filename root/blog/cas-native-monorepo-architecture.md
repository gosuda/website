---
id: 773766eac9e6ff3c46bd686d78539aed
author: Lemon Mint
title: 'Rethinking the Git Monorepo: A CAS-Native Architecture for Humans and Coding Agents'
description: How to build a zero-daemon, CAS-native monorepo on standard Git using parentless trunk snapshots, isolated workspace references, in-memory merge trees, hierarchical OWNERS, and O(delta) mirror synchronization.
language: en
date: 2026-10-30T01:17:15.292299Z
path: /blog/posts/rethinking-the-git-monorepo-a-cas-native-architecture-for-humans-and-coding-agents-zfd3d7931
category: lowlevel
---

Every growing engineering organization eventually hits the same fork in the road. Start with dozens of small repositories, and life feels simple at first. Each library has its own issue tracker, its own CI pipeline, and a `git clone` that finishes before you take a sip of coffee. Then the tax arrives. You change a struct in a core networking package and spend the rest of the afternoon opening version-bump pull requests across six downstream repositories, waiting for CI queues to clear in topological order.

Move everything into a traditional Git monorepo, and you trade coordination pain for physics pain. As commit history deepens and binary assets accumulate in past revisions, `git fetch` slows down. Working trees swell to gigabytes. Before long, someone suggests standing up a custom virtual filesystem daemon, a dedicated code review server, or a fleet of remote build workers just to keep daily development bearable.

When autonomous coding agents join the team, both failure modes get worse. Polyrepos blind an agent to cross-project callers and force it to juggle credentials across repositories. Traditional monorepos drown the agent's container disk and network bandwidth on every task initialization, while concurrent agents stepping on the same branch create endless rebase races.

We wanted a third option. Inspired by the `google3` architecture, we asked a simple question: **Can we get single-trunk atomic changes, sub-second sparse onboarding, hierarchical `OWNERS` governance, and standalone open-source mirrors using nothing but standard Git plumbing and commodity object storage?**

The answer turned out to be yes, and it did not require a custom filesystem daemon or a third-party review appliance. It only required using Git for what it actually is under the hood: a Merkle-tree Content-Addressable Store (CAS).

## The Misconception at the Heart of Slow Git Monorepos

When developers complain that Git does not scale to large monorepos, they are rarely hitting a limit in Git's object database. Inside `.git/objects`, Git stores three immutable primitives keyed by their SHA-1 (or SHA-256) content hash:

1. **`blob`**: The raw byte contents of a single file.
2. **`tree`**: A directory listing that maps filenames and permissions to `blob` or child `tree` hashes.
3. **`commit`**: A tiny text header (typically 200 to 300 bytes) pointing to a single root `tree` hash, zero or more parent `commit` hashes, and author metadata.

As a content-addressable store, this design is remarkably efficient. If a monorepo contains 50,000 files and you edit a single line in one file, the new commit reuses the exact `blob` and `tree` hashes for the other 49,999 files.

Why, then, does a traditional Git monorepo feel heavy? Because convention overloads a single reference, `refs/heads/main`, with three conflicting jobs:

- It holds the **entire unified directory tree** of every project.
- It drags along the **entire parent-linked chain of historical commits** back to day one.
- It sits inside `refs/heads/*`, the namespace that Git servers advertise and negotiate by default on every `git clone` and `git fetch`.

Every time a developer or CI runner fetches `refs/heads/main`, the client and server negotiate commit graph reachability across thousands of historical revisions. If someone accidentally committed a 100 MB archive three years ago and deleted it the next day, its `blob` still hangs off a historical `tree` reachable from `refs/heads/main`.

## Separating the Reference Namespace into Four Layers

Git's wire protocol only advertises `refs/heads/*` and `refs/tags/*` during a standard clone. References under custom namespaces such as `refs/trunk/*` or `refs/ws/*` live in the exact same object store on the server, benefit from the same packfile delta compression, and can be fetched or pushed explicitly, yet they impose zero overhead on default clones.

By splitting responsibilities across four reference namespaces inside a single repository, the bottleneck disappears:

```
refs/heads/main          Control plane only (~20 KB: rules, .gitconfig, sync config, scripts)
refs/trunk/main          Parentless snapshot commit pointing to the latest unified root tree
refs/trunk/history       Linear parent-linked commit chain sharing the exact root tree SHA-1
refs/ws/<actor>/<id>     Isolated workspace snapshots (changelists) per developer or agent
refs/heads/land/<id>     Ephemeral merge-queue trigger deleted immediately upon landing
```

### 1. The Control Plane (`refs/heads/main`)

The default branch contains no application source code. It tracks only the repository operating rules (`AGENTS.md`, `README.md`), a shared `.gitconfig` of workflow aliases, the Starlark synchronization config (`sync.star`), and server-side merge scripts. Cloning the default branch transfers roughly 20 KB and finishes in milliseconds.

### 2. The Parentless Trunk Snapshot (`refs/trunk/main`)

`refs/trunk/main` points to the complete, unified source tree of the entire organization (`//web/*`, `//net/*`, `//ipc/*`, `//devtools/*`, `//corp/*`, `//third_party/*`, `//experimental/*`), but its commit object is created **without a `-p` parent flag**.

Because `refs/trunk/main` has zero parents (`parents = []`), fetching it requires zero commit-graph negotiation. The Git server does not walk history; it sends the single head commit and its current directory `tree` objects. Historical binaries or deleted files from last month are unreachable from `refs/trunk/main` and are never transferred.

### 3. The Linear Audit History (`refs/trunk/history`)

Developers still need `git blame` and `git log` when investigating regressions. Every time a change lands, the merge script creates two commit objects pointing to the **exact same root `$NEW_TREE` SHA-1 hash**:

```bash
# Parentless snapshot commit for instant depth-1 workspace checkouts
SNAP=$(git commit-tree "$NEW_TREE" -m "$RAW_MSG")

# Parent-linked commit for linear git log and git blame
HIST=$(git commit-tree "$NEW_TREE" -p "$OLD_HIST" -m "$RAW_MSG")

git push --atomic origin \
  --force-with-lease="refs/trunk/main:${OLD_MAIN}" \
  --force-with-lease="refs/trunk/history:${OLD_HIST}" \
  "${SNAP}:refs/trunk/main" \
  "${HIST}:refs/trunk/history"
```

Does maintaining both `refs/trunk/main` and `refs/trunk/history` double the repository's storage footprint? **Not at all.** Because both commits reference the identical `$NEW_TREE` hash, Git's content-addressable store deduplicates 100% of the `tree` and `blob` objects. The extra storage cost of the parentless snapshot is a single 250-byte commit header.

### 4. Isolated Workspaces (`refs/ws/<actor>/<id>`)

Work-in-progress branches are pushed to `refs/ws/<actor>/<id>` rather than `refs/heads/*`. A hundred coding agents can save checkpoints every minute without polluting branch dropdowns or slowing down `git fetch` for human engineers.

## Onboarding in Five Commands: How Sparse + Blobless Checkout Works

With this reference layout, a new developer or an ephemeral coding-agent container does not install a custom CLI or FUSE driver. Onboarding takes five standard Git commands:

```bash
# 1. Clone the 20 KB control plane with blob filtering and sparse mode enabled
git clone --filter=blob:none --sparse https://github.com/example-org/monorepo.git
cd monorepo

# 2. Load repository aliases and attach refs/trunk/*
git config include.path ../.gitconfig
git trunk-init

# 3. Declare the domain directories needed for this task
git sparse-checkout set net/relay ipc/codec

# 4. Create an isolated workspace rooted at the latest trunk snapshot
git ws-new alice/optimize-framing
```

Under the hood, each step does only the minimum work required:

1. `git clone --filter=blob:none --sparse` downloads the 20 KB control plane on `refs/heads/main`.
2. `git trunk-init` configures `+refs/trunk/*:refs/remotes/origin/trunk/*` and runs a blobless fetch. Git downloads the `tree` metadata for `refs/trunk/main` (typically a few dozen kilobytes) and **zero file blobs**.
3. `git sparse-checkout set net/relay ipc/codec` instructs Git's cone-mode index filter to materialize only the root directory and those two subtrees.
4. `git ws-new` switches to a new branch rooted at `origin/trunk/main`. At that exact moment, Git requests on demand only the `blob` objects inside `net/relay` and `ipc/codec`. The remaining 95% of the monorepo is never downloaded to disk.

### Future Horizon: Mounting `refs/trunk/main` as a Virtual Filesystem (VFS)

Sparse checkout with blobless clones covers daily workflows without extra client tooling. Because `refs/trunk/main` is already a pure Merkle tree in a Content-Addressable Store, the architecture also provides a direct foundation for an optional userspace Virtual Filesystem (VFS) via FUSE on Linux, FileProvider on macOS, or ProjFS on Windows:

1. **Instant Full-Tree Mount**: A lightweight local mount daemon reads the root `tree` object of `refs/trunk/main` and projects the entire monorepo directory hierarchy into the working directory in milliseconds, removing the need to run `git sparse-checkout set`.
2. **On-Demand Blob Faulting**: File contents remain virtual until a compiler, IDE, or agent issues an `open(2)` or `read(2)` system call. The daemon then fetches the corresponding SHA-1 `blob` from the CAS endpoint and caches it on local NVMe storage.
3. **Copy-on-Write Overlay for Workspaces**: Local edits write to a lightweight upper overlay layer. Running `git ws-save` hashes only the dirty paths in that overlay, writes the updated `blob` and `tree` objects, and pushes the resulting commit to `refs/ws/<actor>/<id>` without scanning unmodified directories.

## The Changelist Model on Pure Git: In-Memory Server-Side Landing

In a traditional Git workflow, merging a feature branch either creates a two-parent merge commit (turning history into a tangled railway yard) or requires developers to manually rebase and race against other engineers pushing to `main`.

Inspired by the `google3` Changelist (CL) model, every workspace in our architecture represents a **single atomic change** against the trunk snapshot. While working locally in `ws/alice/optimize-framing`, a developer or agent might make ten incremental commits (`wip`, `fix tests`, `lint`). When the change is ready to land, the server collapses the workspace delta into a single linear trunk revision without ever checking out a working directory on the CI runner.

Here is how the landing script (`land.sh`) executes in memory using Git 2.40+ plumbing:

```bash
# 1. Identify the root snapshot commit where the workspace branched off
BASE=$(git rev-list --max-parents=0 "$WORK_SHA")

BASE_TREE=$(git rev-parse "${BASE}^{tree}")
MAIN_TREE=$(git rev-parse "${OLD_MAIN}^{tree}")
WORK_TREE=$(git rev-parse "${WORK_SHA}^{tree}")

# 2. Compute the merged root tree in memory without a working tree checkout
if [ "$MAIN_TREE" = "$BASE_TREE" ]; then
  NEW_TREE="$WORK_TREE"
elif OUT=$(git merge-tree --write-tree --merge-base="$BASE_TREE" "$MAIN_TREE" "$WORK_TREE"); then
  NEW_TREE=$(head -n1 <<<"$OUT")
else
  echo "Merge conflict detected; run git ws-sync to rebase." >&2
  exit 1
fi
```

Look closely at step 2:

- Regardless of how many local commits were made on the workspace branch, the script compares only three tree hashes: the original base snapshot (`$BASE_TREE`), the current trunk snapshot (`$MAIN_TREE`), and the final workspace tree (`$WORK_TREE`).
- If `refs/trunk/main` advanced in `//web/blog` while Alice edited `//net/relay`, `git merge-tree --write-tree` computes the combined root tree `$NEW_TREE` directly inside the Git object store in milliseconds. No files are written to the runner's filesystem.
- Finally, `git push --atomic` with `--force-with-lease` updates `refs/trunk/main`, appends one linear commit to `refs/trunk/history`, and deletes the landing trigger `refs/heads/land/<id>` in a single transaction.

## Enforcing Architecture Boundaries: Four-Tier Visibility and Hierarchical `OWNERS`

Consolidating code into one repository only works if boundaries between domains are enforced mechanically before a commit can land. Right after `git merge-tree --write-tree` produces `$NEW_TREE` (and before `git push --atomic` updates the trunk), our policy engine runs two O(delta) checks over `git diff-tree "$MAIN_TREE" "$NEW_TREE"`.

### 1. Four-Tier Domain Visibility

The root tree is partitioned into four visibility tiers:

| Directory Prefix                                                                       | Tier              | Allowed Imports                                  | Prohibited Imports             | `OWNERS` Gate          |
| :------------------------------------------------------------------------------------- | :---------------- | :----------------------------------------------- | :----------------------------- | :--------------------- |
| `//web/*`, `//net/*`, `//ipc/*`, `//storage/*`, `//crypto/*`, `//ai/*`, `//devtools/*` | Public Production | Public Production, `//third_party/*`             | `//corp/*`, `//experimental/*` | Enforced               |
| `//third_party/*`                                                                      | External Vendor   | Self-contained or `//third_party/*`              | All non-`third_party` domains  | Enforced               |
| `//corp/*`                                                                             | Internal Private  | Public Production, `//corp/*`, `//third_party/*` | `//experimental/*`             | Enforced               |
| `//experimental/<actor>/*`                                                             | Sandbox           | Any non-`corp` package                           | `//corp/*`                     | Exempt (auto-approved) |

If a public package in `//web/*` attempts to import an internal package from `//corp/*` or an ungraduated prototype from `//experimental/*`, the landing job aborts before touching `refs/trunk/main`.

### 2. Hierarchical `OWNERS` Verification Without a Gerrit Server

You do not need a dedicated Gerrit instance to enforce directory-level `OWNERS` files with inheritance rules:

- For every path modified between `$MAIN_TREE` and `$NEW_TREE`, the verifier walks upward from the file's directory to `//OWNERS`.
- Any `OWNERS` file can declare `set noparent` to block inherited approvals from parent directories, or `per-file *.sql=@dba-team` to scope approvals by filename glob.
- **Anti-Tampering Invariant**: Existing `OWNERS` files are always read from `$MAIN_TREE` (the current trunk tree), never from `$NEW_TREE`. An unapproved author cannot grant themselves permission by editing `OWNERS` in the same workspace commit.
- **Sandbox Exemption**: Any path under `//experimental/<actor>/*` bypasses `OWNERS` checks so engineers and agents can prototype freely with `git ws-exp <actor> <project>` and later graduate the package into a production domain with `git ws-graduate`.

## Code Review Without Friction: Ephemeral PRs and Snapshot Diffs

Because `refs/heads/main` holds only the control plane and `refs/trunk/main` is a parentless commit, opening a standard GitHub Pull Request directly from a workspace branch into `main` would try to compare unrelated histories. We solved this with three review workflows tailored to different changes:

1. **Instant Two-Dot Workspace Diffs (`git ws-save` + Edge Diff Router)**
   Pushing a workspace with `git ws-save` exposes `refs/ws/<actor>/<id>` through a lightweight edge worker. This worker resolves `refs/trunk/main` and the workspace reference to their commit SHAs and redirects to GitHub's two-dot compare view (`/compare/<trunk-sha>..<ws-sha>`). For small patches or agent-generated changes, reviewers can inspect the diff and click **Land** directly in the web UI.
2. **Ephemeral GitHub Pull Requests (`git ws-pr`)**
   For line-by-line review on GitHub, `git ws-pr` pushes the root snapshot commit (`$(git rev-list --max-parents=0 HEAD)`) to `refs/heads/base/<id>` and the workspace tip to `refs/heads/review/<id>`. It then opens a GitHub Pull Request comparing `review/<id>` against `base/<id>`. Because the base branch is the exact snapshot the workspace branched from, GitHub's **Files changed** tab shows only the exact lines modified in the workspace. When the reviewer clicks **Merge pull request**, a webhook promotes the head commit to `refs/heads/land/<id>`, runs the in-memory landing pipeline, and automatically deletes both `refs/heads/base/<id>` and `refs/heads/review/<id>`.
3. **External Mirror Pull Requests**
   Open-source contributors never have to learn the monorepo workflow. They fork and open pull requests against standalone mirror repositories as usual.

## O(delta) Bidirectional Synchronization and Separate Production Branches

Many organizations publish individual packages as standalone GitHub repositories so external users can `go get`, `cargo install`, or fork a single project without touching the monorepo. Some projects, such as static blogs or release-gated web apps, also maintain a separate `production` branch alongside `main`.

Our Starlark synchronization engine handles both outbound exports and inbound contributions in **O(delta)** time proportional to the files changed in a commit rather than the size of the monorepo:

- **Blobless Index Splicing**: When splicing a mirror repository's subtree into or out of the monorepo root tree, the engine sets `GIT_NO_LAZY_FETCH=1` and runs `git update-index --index-info` followed by `git write-tree --missing-ok`. Unmodified subtrees and blobs are referenced purely by their SHA-1 hashes without ever being downloaded to the CI runner.
- **Delta-Only Blob Prefetching**: Before pushing an exported commit to a standalone mirror, the script runs `git diff-tree -r --diff-filter=AM "$ext_parent" "$export_tree"` and prefetches only the old and new blobs of the modified files via `git cat-file --batch`. Git can then compute a thin delta pack and push to the mirror repository in seconds.
- **Workflow and Release Branch Preservation**: During outbound export, the synchronizer preserves each standalone repository's own `.github/workflows/` directory so public CI runners execute project tests on the mirror. Each workflow in the Starlark config targets an explicit branch such as `main`, which preserves release gates for repositories that deploy from a separate `production` branch. Monorepo landings synchronize with the mirror's `main` branch, and automated metadata or translation commits on `main` sync back into `refs/trunk/main` via webhook. Promoting `main` to `production` remains an explicit release step.

## Closing Thoughts

Building a high-velocity monorepo does not require abandoning standard Git or operating a complex fleet of bespoke infrastructure servers. This architecture treats Git's object store as a Content-Addressable Store, separates `refs/heads/main` from `refs/trunk/main` and `refs/trunk/history`, and merges trees in memory with `git merge-tree --write-tree`. As a result, both human engineers and autonomous coding agents can clone a workspace in under a second and scale cleanly across hundreds of projects.
