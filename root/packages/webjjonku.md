---
id: 7d4e9a2c81f53b60e3a7c4190f28b5d1
author: Lemon Mint
title: 'WebJjonku: Pure Go Multi-Attach Web MCP Server for Local Workspaces'
description: Pure Go Web MCP server (gosuda.org/webjjonku) connecting multiple local developer directories to Gemini Web, ChatGPT Web, and HTTP MCP clients with persistent URLs, profile isolation, and a built-in Yamux relay.
language: en
date: 2026-11-28T02:15:00Z
path: /webjjonku
go_package: gosuda.org/webjjonku
go_repourl: https://github.com/gosuda/webjjonku.git
category: ai
---

`gosuda.org/webjjonku` is a Pure Go (`CGO_ENABLED=0`) Web MCP server built on `github.com/modelcontextprotocol/go-sdk` (`v1.8.0`). It connects multiple local developer directories to Google Gemini Web, ChatGPT Web, and HTTP MCP clients with zero browser automation, persistent connector URLs, strict profile isolation, and per-directory shell environments.

```bash
CGO_ENABLED=0 go install gosuda.org/webjjonku/cmd/webjjonku@latest
```

## Why a Multi-Attach Local Hub?

Connecting a web-based AI client to local repositories usually breaks down in two ways. Either you register and tear down a separate connector URL every time you switch projects, or you run one coarse-grained server that mixes work and personal credentials, Git identities, and shell environments together.

WebJjonku solves both problems by running one lightweight local Hub (`webjjonku _hub`) per profile. You register your Web MCP connector once, then attach or detach any number of local directories on the fly from separate terminal tabs:

```bash
# Terminal 1: attach backend/ under the "work" profile (spawns profile Hub automatically)
cd ~/work/backend && webjjonku up --profile work --relay https://relay.example.com

# Terminal 2: attach frontend/ to the same Hub without re-registering the connector
cd ~/work/frontend && webjjonku up --profile work
```

Each `webjjonku up` terminal connects to the profile Hub over an authenticated local IPC socket (`0600` Unix domain socket on macOS/Linux, authenticated loopback TCP on Windows) and sends its live shell environment (`direnv`, `mise`, `nvm`, `GH_CONFIG_DIR`, `GIT_CONFIG_GLOBAL`, `AWS_PROFILE`, etc.) after scrubbing internal `WEBJJONKU_*` secrets. That environment lives strictly in Hub memory for that mount and is zeroed immediately when you press `Ctrl-C` to detach. Closing the last attached terminal shuts the Hub down cleanly after a short idle grace period (unless `--keep-hub` is set).

## Architecture Overview

```text
+------------------------------------------+
|  Gemini Web / ChatGPT Web / HTTP MCP     |
+------------------------------------------+
                     |
  HTTPS /<base>/<profile>/m/<key>/mcp  (Magic Link)
  HTTPS /<base>/<profile>/mcp          (Bearer / OAuth 2.1)
                     v
+------------------------------------------+
| Reverse Proxy (Caddy/Nginx/Tailscale)    |
|   OR built-in `webjjonku relay`          |
|   (Yamux over WebSocket, multi-profile)  |
+------------------------------------------+
                     |
           HTTP or net.Listener
                     v
+------------------------------------------+
| Profile Local Hub (`webjjonku _hub`)     |
| - Auth Engine (Magic / Bearer / OAuth)   |
| - Stateless Streamable HTTP Transport    |
| - 32 Native Tools + Hashline v2 + LSP    |
| - SnapCompact + SecretShield + Audit     |
+------------------------------------------+
           ^                     ^
    Unix Socket / IPC     Unix Socket / IPC
    (Dir A + Shell Env)   (Dir B + Shell Env)
           |                     |
   Terminal 1: `up`      Terminal 2: `up`
```

### Profile Isolation and Git Guardrails

Profiles (`--profile`, `WEBJJONKU_PROFILE`, or an upward `.webjjonku-profile` file) separate every piece of state under `<baseConfigDir>/profiles/<profile>/`: config, ports, URL prefixes (`/<profile>/...`), auth keys, persistent memory notes, rotated `0600` JSONL audit logs, and Git commit email rules.

```bash
echo work > ~/work/.webjjonku-profile
webjjonku --profile work config allowed-emails "alice@company.com,*@company.com"
webjjonku --profile personal config allowed-emails "alice@gmail.com"
```

Before `git_commit` stages or commits anything, it resolves the effective Git author email for that mount and rejects the commit immediately if it does not match `git.allowedGitEmails`.

### 3-Way Scrypt-Free Authentication

`internal/auth` uses only Go standard library cryptography (`crypto/sha256`, `crypto/rand`, `crypto/subtle`) and supports three authentication modes on the same Hub:

1. **Magic Link (`/m/<key>/mcp`, default)**: Generates a persistent 256-bit `wjm_...` key stored in `0600` `auth.json` (or a user-pinned static key of at least 24 bytes via `--key` / `WEBJJONKU_KEY`) so your connector URL never changes across restarts. Paste it into Gemini Web or ChatGPT Web Custom MCP Connector settings with **No Auth**. Requests are strictly `POST`-only JSON-RPC (`GET` returns `404` with `Referrer-Policy: no-referrer` and `Cache-Control: no-store`), and relay/server logs automatically redact `/m/<secret>/` to `/m/[REDACTED]/`.
2. **Static Bearer Token (`/mcp`)**: Verifies `Authorization: Bearer wjb_...` (or your pinned static key) in constant time for HTTP MCP clients and API gateways.
3. **Lightweight OAuth 2.1 (`/mcp`, `/oauth/*`, `/.well-known/*`)**: Implements RFC 9728 (including root-level sub-path forwarding), RFC 8414, RFC 7591 (Dynamic Client Registration), RFC 7636 (PKCE `S256`), RFC 8707 (resource binding), RFC 7009 (token revocation), and RFC 9700 (refresh-family replay revocation), with redirect allowlists for Google Gemini Web and ChatGPT Web plus a persistent `wjk_...` approval key.

### Reverse-Proxy Sub-Paths and Built-In Relay

WebJjonku supports sub-path mounting natively behind Caddy, Nginx, or Tailscale Serve without needing a dedicated subdomain:

```bash
webjjonku --profile work config set-base-path /jjonku
webjjonku --profile work config set-public-url https://my.domain/jjonku/work
webjjonku up --profile work
```

If you do not want to expose your local machine directly, WebJjonku ships with an embedded multi-profile WebSocket+Yamux relay server (`webjjonku relay` and `Dockerfile.relay` built on `gcr.io/distroless/static-debian12:nonroot`):

```bash
# Public server
WEBJJONKU_RELAY_TOKEN="replace-with-strong-secret" \
  webjjonku relay --listen :8080 --base-path /jjonku

# Local machine
WEBJJONKU_RELAY_TOKEN="replace-with-strong-secret" \
  webjjonku up --profile work --relay https://relay.example.com/jjonku
```

A single relay instance multiplexes multiple isolated profiles (`work`, `personal`, `default`) over persistent Yamux-over-WebSocket tunnels with 15-second keepalives and exponential backoff reconnects (`1s` to `15s`).

## 32 Native Agentic Tools

WebJjonku exposes a fixed catalog of 32 lowercase `snake_case` tools over Stateless Streamable HTTP. When multiple directories are attached, file tools address other mounts with alias prefixes (such as `frontend/src/App.tsx`), while shell and Git tools accept an optional `workspace` parameter and run with that mount's captured shell environment.

| Category                               | Count | Tools                                                                                              |
| :------------------------------------- | ----: | :------------------------------------------------------------------------------------------------- |
| **Workspace and context**              |     5 | `get_agent_brief`, `list_projects`, `set_project_root`, `get_environment`, `get_project_doc`       |
| **Filesystem, edit, and search**       |     8 | `read_file`, `batch_read`, `write_file`, `edit`, `list_directory`, `tree`, `glob`, `grep`          |
| **Semantic code intelligence**         |     1 | `lsp`                                                                                              |
| **Shell execution**                    |     2 | `exec_command`, `write_stdin`                                                                      |
| **Git and diff checkpoints**           |     5 | `git_status`, `git_log`, `show_diff`, `git_commit`, `git_push`                                     |
| **Memory, plan, and handoff**          |     6 | `recall`, `remember`, `update_memory_note`, `forget_memory_note`, `update_plan`, `session_handoff` |
| **Progressive skills**                 |     2 | `skills_list`, `skills_read`                                                                       |
| **Media, visual compaction, and time** |     3 | `view_image`, `pack_to_frames`, `clock_curr_time`                                                  |

Every tool call runs through a 10-stage middleware pipeline with engineering optimizations designed for web chat context limits:

- **Hashline v2 Editing (`edit`)**: Tracks line anchors with `xxh32` tags (`[PATH#4HEX]`), supports `PUT`, `CUT`, `@name` registers, `REM`, and `MV`, recovers automatically from stale line numbers using Myers diffing, and appends inline post-edit LSP diagnostics (with a 0ms Go `go/parser` fallback).
- **Spill-Before-Minimize Artifacts (`artifact://N`)**: Large command logs and tool outputs are spilled to inspectable artifacts before minimization and UTF-8 rune-safe token budgeting, letting the model drill into specific line ranges via `read_file` selectors (`:N`, `:A-B`, `:A+N`, `:-N`).
- **Pure Go SnapCompact (`pack_to_frames`)**: Renders large files, artifacts, diffs, or session handoffs into compact monochrome PNG frames (`8on22-bw`) directly in Pure Go for vision-capable web models.
- **Loop Guardrails and Built-In Skills**: Detects repetitive tool-call loops across a 32-entry window and bundles 6 starter skills (`codebase-explore`, `hashline-edit-verify`, `tdd-verify`, `systematic-debug`, `code-review`, `session-handoff`).

## Confinement and Security Guardrails

- **Kernel `*os.Root` Confinement**: All file reads, atomic writes, Hashline edits, and LSP `rename`/`code_actions` edits execute strictly inside Go `*os.Root` with 16-hop symlink guards.
- **OS Sensitive Path Denylist**: Blocks `.ssh`, `.gnupg`, `.aws`, `.kube`, `.docker`, `.npmrc`, `.netrc`, `.git-credentials`, OS keychains, browser credential stores, `/etc/shadow`, `/etc/sudoers`, and WebJjonku's own config directory across both direct reads and `grep` (`rg` and Pure Go RE2 fallback).
- **`--no-shell` Git Protection**: Blocks writing `.git`, `.git/hooks/*`, `.git/config`, and `.gitmodules` whenever shell execution is disabled.
- **Secret Shield**: Scans tool outputs with RE2 patterns, truncated-PEM detection, WebJjonku key prefixes (`wjm_`, `wjb_`, `wjk_`, `wja_`, `wjr_`, `wjsec_`), and dynamically registered custom static keys before returning responses.

## CLI Reference

| Subcommand                  | Description                                                                                                                                  |
| :-------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------- |
| `webjjonku up [dir]`        | Attach `dir` (default: current directory) and its live shell environment to the profile Hub, spawning the Hub if offline                     |
| `webjjonku serve`           | Run a persistent headless profile Hub in the foreground (`--root`, `--port`, `--relay`)                                                      |
| `webjjonku status`          | Display live Hub state, attached mounts, and Magic Link / MCP URLs (or persisted offline URLs)                                               |
| `webjjonku down`            | Gracefully stop the active profile Hub                                                                                                       |
| `webjjonku rotate-link`     | Rotate persisted Magic Link, Bearer, and OAuth approval keys                                                                                 |
| `webjjonku revoke`          | Revoke all issued OAuth 2.1 access and refresh tokens                                                                                        |
| `webjjonku config <subcmd>` | View or mutate profile settings (`show`, `set-base-path`, `set-public-url`, `set-key`, `allowed-emails`, `relay`, `snap-mode`, bool toggles) |
| `webjjonku doctor`          | Verify runtime dependencies, config validity, profile resolution, and Hub health                                                             |
| `webjjonku relay`           | Run the lightweight multi-profile WebSocket+Yamux relay server                                                                               |

## Links and Documentation

- **GitHub Repository**: [https://github.com/gosuda/webjjonku](https://github.com/gosuda/webjjonku)
- **Getting Started**: [English](https://github.com/gosuda/webjjonku/blob/main/docs/en/getting-started.md) / [한국어](https://github.com/gosuda/webjjonku/blob/main/docs/ko/getting-started.md)
- **Architecture and Security**: [English](https://github.com/gosuda/webjjonku/blob/main/docs/en/architecture-and-security.md) / [한국어](https://github.com/gosuda/webjjonku/blob/main/docs/ko/architecture-and-security.md)
- **Tools and Hashline v2 Reference**: [English](https://github.com/gosuda/webjjonku/blob/main/docs/en/tools-and-hashline.md) / [한국어](https://github.com/gosuda/webjjonku/blob/main/docs/ko/tools-and-hashline.md)
- **CLI and Relay Reference**: [English](https://github.com/gosuda/webjjonku/blob/main/docs/en/cli-and-relay.md) / [한국어](https://github.com/gosuda/webjjonku/blob/main/docs/ko/cli-and-relay.md)
