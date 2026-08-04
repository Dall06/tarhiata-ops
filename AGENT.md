# AGENT.md — Tarhiata-Ops Codebase Guide for AI Agents

> This file provides context, architecture, and rules for AI coding agents (Cursor, Copilot, Antigravity, etc.) working on this repository.

---

## 1. Project Overview

**Tarhiata-Ops** is a zero-overhead, local-first private PaaS (Platform as a Service) written in **Go 1.25+**.

- Runs **100% locally** on the developer's workstation (0 MB RAM on the VPS).
- Controls a remote VPS via **encrypted SSH** — no webhooks, no SaaS intermediaries.
- Provides a **Vercel/Railway-aesthetic Web Dashboard** at `http://localhost:8080` and a full **CLI**.
- Manages Docker Swarm, Traefik v3, databases, SSL, env vars, SSH keys, backups, and multi-node scaling.
- Made in México 🇲🇽 — primary region: Vultr `mex` (Mexico City).

---

## 2. Key Architecture Rules

1. **A package inside `pkg/` CANNOT import another package from `pkg/`** — they are isolated utilities.
2. **`srv/sys/repositories/`** implements infra adapters (SSH, SQLite, cloud APIs). Never import from `srv/ui/` or `srv/cli/`.
3. **`srv/sys/usecases/`** contains business logic. Depends only on ports (interfaces) defined in `srv/sys/ports/`.
4. **`srv/ui/`** is the HTTP layer (controllers, DTOs, embedded static assets). Never import usecases directly — use port interfaces.
5. **`srv/cli/`** is the CLI layer (Cobra commands). Same rule as `srv/ui/`.
6. **`pkg/auditlog`** is the only global singleton allowed. Access via `auditlog.GetDefaultLogger()`.
7. **Always run `go test ./...` to validate any change** before committing.

---

## 3. Directory Structure

```
tarhiata-ops/
├── cmd/tarhiata/main.go        # Entrypoint: starts CLI + Web server
├── pkg/
│   ├── auditlog/               # NDJSON async audit logger (singleton)
│   └── sshclient/              # SSH client wrapper (golang.org/x/crypto/ssh)
├── srv/
│   ├── sys/
│   │   ├── domain/             # Core domain types (Server, Service, Database, etc.)
│   │   ├── ports/              # Interfaces: UseCases, Repositories, SSH
│   │   ├── repositories/       # Implementations: SQLite, SSH exec, Vultr API, DigitalOcean
│   │   ├── usecases/           # Business logic (one file per use case)
│   │   └── tests/mocks/        # Mock implementations for testing
│   ├── ui/
│   │   ├── controllers/        # HTTP handlers (net/http), WebSocket SSH streaming
│   │   ├── dto/                # Request/Response DTOs with JSON tags
│   │   └── views/public/       # Embedded frontend: index.html, app.js, style.css
│   └── cli/
│       ├── app/                # CLI commands: dashboard, deploy, list, status
│       ├── cluster/            # CLI commands: init, bootstrap, observability
│       ├── db/                 # CLI commands: db create
│       └── sys/                # CLI commands: config, shell, tools
├── docs/
│   ├── landing/index.html      # GitHub Pages landing page
│   ├── logs/audit.log          # Local audit log (development mode)
│   ├── en/                     # (reserved for English docs expansion)
│   └── es/README.md            # Full Spanish documentation
├── .github/
│   └── workflows/release.yml   # Multi-platform binary release CI
├── action.yml                  # Reusable GitHub Action to install tarhiata CLI in CI
└── Taskfile.yml                # Task runner shortcuts
```

---

## 4. Frontend (Web Dashboard)

- **Single-file app**: `srv/ui/views/public/index.html` + `app.js` + `style.css`
- **Design system**: BlackOps dark theme (`#09090b`, `#121215`, `#27272a`)
- **I18n**: Full ES/EN via `data-i18n` attributes on DOM elements + dictionary in `app.js`. Change language via `applyLanguage(lang)`. State persisted in `localStorage('tarhiata_lang')`.
- **Command Palette**: `⌘K` / `Ctrl+K` — 13 system tools accessible by keyword search.
- **No frameworks**: Vanilla HTML + CSS + JS only. No React, no Tailwind.

---

## 5. Audit Log

| Environment | Path |
|---|---|
| Production (VPS) | `/opt/tarhiata/audit.log` |
| Development (local) | `docs/logs/audit.log` |

Format: **NDJSON** — one JSON object per line. Written async via a buffered channel worker. Access via `auditlog.GetDefaultLogger()`.

---

## 6. Database / State

- **Local state**: SQLite at `~/.config/tarhiata/config.db` — stores server config, services, databases, links.
- **Remote state reference**: `/opt/tarhiata/state.json` on the VPS — non-sensitive topology snapshot for multi-PC sync.
- **No ORM** — raw `database/sql` with `modernc.org/sqlite`.

---

## 7. What `action.yml` Is

`action.yml` makes this repo a **reusable GitHub Action**. Any other repo's CI workflow can use it:

```yaml
- uses: Dall06/tarhiata-ops@main
```

This downloads the `tarhiata-linux-amd64` binary and makes the `tarhiata` command available in that GitHub Actions runner. Useful for CD pipelines that deploy via `tarhiata deploy` from CI.

---

## 8. Testing Rules

- All test files live next to the source file they test (`*_test.go` in the same directory).
- Use `go test ./...` — always, no exceptions.
- Mocks live in `srv/sys/tests/mocks/`.
- Do **not** use `t.Parallel()` in SSH-dependent tests (shared SSH mock state).

---

## 9. Go Version & Dependencies

- **Go**: `1.25+`
- **SSH**: `golang.org/x/crypto/ssh`
- **SQLite**: `modernc.org/sqlite` (CGO-free)
- **HTTP**: stdlib `net/http` only — no Gin, no Echo, no Fiber
- **CLI**: `github.com/spf13/cobra`
- **Cloud**: Vultr API v2 via raw HTTP (`net/http` + `encoding/json`)

---

## 10. Do Not

- ❌ Add agent daemons or background processes to the VPS host
- ❌ Import `pkg/X` from `pkg/Y`
- ❌ Add frontend frameworks (React, Vue, Tailwind) without explicit user request
- ❌ Commit `audit.log`, `*.db`, or any SSH private key
- ❌ Use `os.Exit` inside library packages (only in `cmd/`)
- ❌ Break the 0-dependency-on-VPS-RAM principle
