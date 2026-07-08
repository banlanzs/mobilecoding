# mobilecoding

[English](./README.md) | [简体中文](./README.zh-CN.md)

**A personal AI CLI remote console** — turn your local Claude Code / Codex / any LLM CLI into a structured chat interface operable from your phone.

Two mobile clients share one Go backend:

- **PWA (browser, `web/`)** — open `https://<your-pc-ip>:8443` on your phone, scan the QR code or enter the token to connect. No app install.
- **Native app (React Native, `mobile/`)** — Android/iOS client with local SQLite persistence, scanning to connect.

---

## Features

### AI Engines

| Feature | Description |
|---|---|
| **ClaudeRunner** | Parses `claude --output-format stream-json`; supports assistant / tool_use / permission_request / plan_mode events |
| **CodexRunner** | `codex app-server` JSON-RPC long connection with initialize handshake, turn/interrupt abort |
| **PtyRunner / PipeRunner** | Generic PTY / Pipe modes for any CLI (aichat / ollama / ...) |
| **Declarative agents** | `agents.json` declares agent metadata; add an agent by editing JSON |
| **Multi-turn** | `--resume` keeps Claude session context |
| **Model switching** | Phone dropdown selects model via `--model`; model list follows the active settings file |
| **Permission answers** | Allow / Deny buttons on the phone, decisions returned to the CLI via stdin / hook |
| **Abort** | Send button becomes Stop while waiting; kills the CLI process but keeps the session |

### Mobile Experience

| Feature | Description |
|---|---|
| **PWA** | Service Worker offline cache, `display: standalone` fullscreen |
| **Streaming** | content_block_delta incremental render |
| **Thinking collapse** | Thinking blocks collapsed by default, one-tap expand |
| **Markdown** | marked with full GFM (tables, lists, code blocks with copy + language tag) |
| **Persistence** | localStorage + SQLite dual persistence; survives refresh |
| **Reconnect** | WebSocket reconnect with `after_seq` catch-up replay |
| **Input history** | Up/Down arrow to recall recent inputs (cross-session, localStorage) |
| **Drafts** | Per-session draft preserved when switching sessions |
| **Offline queue** | Input while offline is queued and auto-sent on reconnect |
| **Unified diff** | DiffView parses git diff output, structured +/ /- rendering |
| **Session management** | Rename sessions, resume history sessions (resume ID persisted) |
| **Permissions** | Banner shows tool details (file path / command); "allow this tool this session" + "allow all this turn" |
| **QR pairing** | Scan QR code to auto-fill token |

### Subcommands

| Command | Behavior |
|---|---|
| `mobilecoding claude` | Start server + Claude (remote-control mode); phone scans to co-exist |
| `mobilecoding codex` | Start server + Codex |
| `mobilecoding relay` | Connect to a relay server as an agent |
| `mobilecoding server` | Start server only (default) |
| `mc` | Alias for `mobilecoding` |

**Smart settings detection**: `mc claude` without `-settings` auto-detects `<CWD>/.claude/settings.local.json`; if absent, falls back to the global `~/.claude/settings.json`. Explicit `-settings <path>` overrides.

**Smart IP selection**: prefers LAN IP (10.x > 172.16-31 > 192.168), skips virtual adapters. Override with `-ip 192.168.1.100` or `MOBILECODING_IP=...`.

### Transport & Security

| Feature | Description |
|---|---|
| **HTTPS** | Self-signed CA + server certificate |
| **mTLS** | Optional client cert auth (`--mtls=required`) |
| **QR pairing** | QR code printed to terminal at startup |
| **WebSocket** | Structured RPC protocol (codec/conn/hub/handler) with exponential backoff |
| **Relay** | Cross-network remote (agent ↔ relay ↔ client) |
| **Auth** | 32B random Bearer token + constant-time compare + log redaction |
| **Cert rotation** | Auto-renew 30 days before expiry |
| **Config hot-reload** | SIGHUP triggers reload |

---

## Quick Start

### Build & Run

```bash
make build
./dist/mobilecoding.exe

# Browser: https://127.0.0.1:8443/
```

### Remote-Control Mode (recommended)

```bash
make build
npm link

# Auto-enters remote-control mode
mobilecoding claude
mobilecoding claude --settings ~/.claude/settings.xxx.json
mobilecoding claude --model claude-opus-4-8

# mc is an alias for mobilecoding
mc claude
mc codex
```

### npm Install

```bash
npm install -g @banlan/mobilecoding
mobilecoding                          # start server (default)
mobilecoding claude                   # remote-control mode
mobilecoding relay -relay wss://...   # relay mode
```

### Custom Token

```bash
MOBILECODING_AUTH_TOKEN=mysecrettoken ./mobilecoding.exe
```

### Native Client (React Native)

`mobile/` is the Android/iOS client, parallel to the PWA, connecting to the same Go backend.

```bash
cd mobile
npm install
npm run android   # or: npm run ios (first run: bundle exec pod install --project-directory=ios)
```

Scan the QR code or enter the backend address (`https://<pc-ip>:8443`) + token to connect.

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `MOBILECODING_PORT` | `8443` | Listen port |
| `MOBILECODING_IP` | auto-detect | Local IP (overrides auto-detect) |
| `MOBILECODING_AUTH_TOKEN` | auto-generated | Auth token |
| `MOBILECODING_WORKSPACE` | `~/mobilecoding-workspace` | Workspace path |
| `MOBILECODING_MTLS` | `optional` | mTLS mode (optional / required) |
| `MOBILECODING_LOG_LEVEL` | `info` | Log level |
| `MOBILECODING_DEFAULT_COMMAND` | `claude` | Default AI command |
| `MOBILECODING_DEFAULT_ARGS` | — | Default CLI args |
| `MOBILECODING_MODELS` | built-in list | Custom model list (`label:value,label:value`) |
| `MOBILECODING_LAUNCH_MODE` | — | `managed` / `remote-control` |

### Model Configuration

The phone model dropdown lists models from `/api/v1/models`. Customize via:

1. **Env var**: `MOBILECODING_MODELS=Haiku:claude-haiku-4-5,Sonnet:claude-sonnet-4-6`
2. **Settings file**: set `ANTHROPIC_*_MODEL` in the `env` block of `~/.claude/settings.*.json` or `<project>/.claude/settings.local.json`. The dropdown shows the real model names (e.g. `minimax-m3[1m]`), not tier labels. Switching settings refreshes the list.

---

## Architecture

```
Phone browser (PWA) / React Native app
    │
    ▼  WebSocket (wss://)
mobilecoding Go backend
    │
    ├─ Claude CLI (--output-format stream-json, stdin duplex)
    ├─ Codex CLI (app-server JSON-RPC, initialize handshake)
    └─ Generic PTY / Pipe (aichat / ollama ...)
```

```
Internal packages:
  auth/       — Token + CA + mTLS + device cert + QR + cert rotation
  config/     — Config + env vars + SIGHUP hot-reload
  engine/     — ClaudeRunner + CodexRunner + PtyRunner + PipeRunner + declarative agent registry
  files/      — Git status/diff + file tree + file read (path-traversal guarded)
  gateway/    — HTTP routes + SPA + WS upgrade + REST API
  hook/       — Claude HTTP hook endpoint (PermissionRequest)
  logx/       — Logging + redaction + rotation
  projection/ — Event projection: stream-json / Codex JSON-RPC → structured events
  protocol/   — Wire protocol constants (event types / RPC methods / error codes)
  relay/      — WebSocket relay (agent ↔ relay ↔ client)
  session/    — Session lifecycle + metadata persistence + resume ID
  store/      — SQLite message persistence + seq + search + cleanup
  ws/         — WebSocket protocol (codec/conn/hub/handler + replay buffer)
```

---

## Tech Stack

| Layer | Tech |
|---|---|
| Backend | Go 1.22+ |
| Router | chi/v5 |
| WebSocket | gorilla/websocket |
| PTY | creack/pty |
| Crypto | stdlib crypto (TLS / mTLS / token / cert) |
| Storage | modernc.org/sqlite (WAL mode) |
| Web | React 18 + TypeScript + Vite |
| Markdown | marked (GFM) |
| PWA | Workbox (Service Worker) |
| Mobile | React Native 0.81 (`mobile/`, Android/iOS, op-sqlite + Zustand) |
| Distribution | npm + GitHub Actions |

---

## Development

```bash
# Backend
go test ./...
go build -o dist/mobilecoding.exe ./cmd/server

# Frontend
cd web && npm run build

# One-shot build
make build
```

---

## Reference Projects

mobilecoding drew design ideas from these open-source projects:

- **[Happy](https://github.com/slopus/happy)** — Mobile/Web client for Claude Code & Codex. E2E encryption, Local/Remote modes, Session Scanner. → message persistence + seq, V3 message API, wire protocol unification.
- **[VibeAround](https://github.com/jazzenchen/VibeAround)** — AI agent management platform. Multi-agent, Local API Bridge, onboarding. → declarative agent config, first-use onboarding.
- **[MindFS](https://github.com/a9gent/mindfs)** — AI agent remote gateway + result visualization. StreamHub replay, session search, single binary. → StreamHub replay, session search.
- **[EasyCodex](https://github.com/Ryan-Laws/easycodex)** — Codex remote control. JSON-RPC, message normalization, streaming batching. → Codex JSON-RPC protocol, message normalization, projection.

---

## Security

- ✅ Forced HTTPS (self-signed CA + server cert)
- ✅ Optional mTLS + device cert
- ✅ 32B random token + constant-time compare
- ✅ Bearer auth (query + Authorization header)
- ✅ Log redaction (Authorization / api_key / token)
- ✅ CheckOrigin validation (CSRF / cross-origin hijack defense)
- ✅ Cert auto-rotation 30 days before expiry

---

## License

MIT
