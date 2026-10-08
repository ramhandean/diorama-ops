# Architecture & Technical Decisions

This document records the foundational and ongoing architectural decisions for DioramaOps.

---

## ADR 001: Backend Stack & Routing
- **Status:** Accepted
- **Decision:** Use Go (>= 1.24) with standard library `net/http` (`http.NewServeMux()` introduced in Go 1.22+).
- **Rationale:** Standard library provides robust HTTP method and wildcard routing (`GET /api/v1/...`) without external framework dependencies, minimizing supply-chain risks and binary size.
- **WebSocket:** Use `github.com/coder/websocket` for standard-compliant, zero-dependency, context-aware WebSocket connections.

---

## ADR 002: Embedded Pure-Go SQLite Engine
- **Status:** Accepted
- **Decision:** Use `modernc.org/sqlite` with `CGO_ENABLED=0` and WAL mode (`_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)`).
- **Rationale:** Eliminates CGO and C toolchain requirements for cross-compiling amd64 and arm64 targets. Yields a single static binary runnable inside `gcr.io/distroless/static-debian12:nonroot`.
- **Concurrency & Batching:** Writes route through a dedicated in-process batch writer (flush interval: 5 seconds or 500 items).

---

## ADR 003: Storage Strategy & Rollup-Only Analytics
- **Status:** Accepted
- **Decision:** Do not persist raw clickstream or event logs. Persist pre-aggregated metrics into:
  - `hits_hourly` (bucket timestamp, pageviews, sessions, dwell sum)
  - `paths_daily` (tenant, day, cleaned path, hits with top-N pruning)
  - `referrers_daily` (tenant, day, domain-only, hits)
- **Retention:** Data older than `RETENTION_DAYS` (default 395 days) is pruned daily. Database backup runs via non-blocking SQLite `VACUUM INTO`.

---

## ADR 004: Privacy & Anonymity Invariants
- **Status:** Accepted
- **Decision:**
  - Zero cookies, zero local storage tracker tokens on client websites.
  - Raw visitor IP addresses are never logged or stored.
  - Visitor identifier (`vid`) = `HMAC-SHA256(IP + UA + site_key, daily_salt)`.
  - `daily_salt` is generated randomly per calendar day in server memory and is never written to persistent disk.
  - URL paths are stripped of query parameters and hashes. Referrers store domain names only.

---

## ADR 005: Frontend 3D Rendering & UI Design
- **Status:** Accepted (Updated)
- **Decision:** Vite + React + TypeScript with React Three Fiber (`@react-three/fiber`) and Drei (`@react-three/drei`).
- **Package Manager:** Bun (1.x) per user directive, replacing pnpm for drastically faster install times and lower memory footprint. Docker build uses `oven/bun:1-alpine` in Stage 1.
- **Styling:** Tailwind CSS following strict 60-30-10 palette (neutral dark canvas `#09090b`, subtle 1px border-first architecture, no generic AI purple/blue gradients).
- **Performance:** Instanced meshes for visitor avatars (capped at 60 rendered models). Fallback 2D canvas/DOM view when WebGL context initialization fails.

---

## ADR 006: Tenant Source of Truth
- **Status:** Accepted
- **Decision:** SQLite is the single source of truth for tenants. `tenants.config.json` serves exclusively for initial seeding, bulk import, and export workflows.

---

## ADR 007: Reverse Proxy Deployment
- **Status:** Accepted
- **Decision:** Do not bundle a separate Caddy container inside `docker-compose.yml`. Caddy runs directly on the VPS host system.
- **Port Mapping:** `dioramaops` exposes `127.0.0.1:7437:7437` to localhost only. Host Caddy proxies external HTTPS/WSS traffic to `127.0.0.1:7437`.
- **Sample Configuration:** Provided in `deploy/Caddyfile` and root `Caddyfile`.

---

## ADR 008: VPS Deployment via deploy.sh
- **Status:** Accepted
- **Decision:** Deployments on user VPS operate via direct `git pull` and execution of `./deploy.sh`, bypassing GitHub Actions / external registries.
- **Compose Build:** `docker-compose.yml` specifies `build: .` and names local image `dioramaops:local`.
- **Automation & Sequential Build:** `deploy.sh` verifies `.env` existence (generating random `ADMIN_SECRET` if initial), prepares volume permissions, performs sequential stage build (frontend first, then server) to prevent VPS RAM/CPU exhaustion, runs `docker compose up -d`, and checks container status.
