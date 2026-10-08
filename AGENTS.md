# DioramaOps (Self-Hosted Community Edition) - Agent Guide

This repository contains the **Self-Hosted Community Edition** of DioramaOps.

---

## 1. Architectural Philosophy & Isolation

- **1 Instance = 1 Project Policy:**
  This edition is strictly designed for self-hosters running isolated containers (e.g., in Docker, Portainer, or single VPS). Each container instance is dedicated to 1 project.
- **Authentication Model:**
  Protected via a single `ADMIN_SECRET` environment variable. There is no user database, no OAuth dependency, and no registration flow.
- **Database Model:**
  Embedded pure-Go SQLite WAL database stored in `DATA_DIR` (`./data/diorama.db`). All tenants belong to the default singleton workspace (`'default'`).
- **Privacy First:**
  Zero cookies, zero local storage tokens, no raw IP logging. Visitor hashes are salted with daily volatile salts in RAM.

---

## 2. Directory Layout

- `cmd/dioramaops/main.go`: Entrypoint for hub server, CLI import, and export.
- `internal/config/`: Configuration parsing (`ADMIN_SECRET`, `PORT`, `DATA_DIR`, etc.).
- `internal/server/`: HTTP routing, admin middleware, static file serving.
- `internal/ingest/`: WebSocket hub (`/ws`), Beacon ingest (`/api/v1/b`), rate limiting.
- `internal/presence/`: Real-time active visitor tracking.
- `internal/rollup/`: Hourly and daily aggregation engine for telemetry.
- `internal/uptime/`: Health check engine and Uptime Kuma synchronization.
- `tracker/m.js`: Embedded JavaScript telemetry script.
- `web/`: React Three Fiber frontend with 2D fallback.

---

## 3. Developer & Agent Workflow

### Backend Testing & Build
```bash
go test -v ./...
go build -o dioramaops ./cmd/dioramaops
```

### Frontend Build (Dynamic NVM)
```bash
export NVM_DIR="$HOME/.nvm" && [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" && cd web && npm run build
```

### Constraints for Agents
- Do **not** add multi-tenant or OAuth complexity to this repo; multi-tenancy belongs strictly to `diorama-cloud`.
- Preserve the 1-project self-host limit and single-secret authentication.
- Ensure the static binary remains under 25 MB and container image under 40 MB.
