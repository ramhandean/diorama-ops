# DioramaOps (Self-Hosted Community Edition)

<p align="center">
  <strong>Interactive 3D Infrastructure & Real-Time Telemetry Hub</strong><br>
  Self-hosted diorama visualizing your website, real-time visitors, and uptime health in a single 3D scene.
</p>

<p align="center">
  <span style="background:#18181b;padding:4px 8px;border-radius:4px;border:1px solid #27272a;font-size:12px;">Edition: <strong>Self-Hosted (1 Instance = 1 Project)</strong></span>
</p>

<p align="center">
  <a href="#quick-start-2-minutes">Quick Start (2 Mins)</a> •
  <a href="#key-features">Key Features</a> •
  <a href="#tracker-installation">Tracker Installation</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#license">License</a>
</p>

---

## Overview

**DioramaOps Self-Hosted Edition** turns your website and web apps into an interactive 3D cyberpunk diorama. Each instance is dedicated to 1 project for lightweight, isolated private hosting on your VPS or home server. Real visitors browsing your sites spawn as low-poly avatars in the central lobby and walk to their destination booth in real-time.

Built from scratch as a single static Go binary (`< 25 MB`) running an embedded pure-Go SQLite WAL database and React Three Fiber frontend, packaged in a lightweight distroless Docker container (`< 40 MB`).

> [!NOTE]
> Looking for multi-tenant team hosting with Google OAuth and workspaces? Check out [DioramaOps Cloud](https://github.com/ramhandean/diorama-cloud).

---

## Key Features

- **Interactive 3D Diorama:** Procedural booths, ambient neon lighting, orbit controls, and smooth visitor avatar pathing.
- **Privacy by Design:** Zero cookies, zero local storage tokens, no raw IP logging. Visitor hashes rotate daily in volatile server memory.
- **Ultra-Lightweight Tracker (`m.js`):** Under 1.5 KB gzip with zero external dependencies. Supports SPAs (`pushState`), tab visibility pausing, and beacon fallbacks.
- **Uptime Monitoring & Hysteresis:** Direct status sync with Uptime Kuma Status Page APIs plus internal HTTP health probes with flap prevention.
- **HUD Analytics Drawer:** Real-time active counters, 24-hour and 7-day traffic charts (Recharts), top pages, and top referrers.
- **Prometheus Exporter (`/metrics`):** Native Prometheus metrics exporter without cardinality explosion, plus ready-to-use Grafana dashboard template.
- **2D Accessible Fallback:** Automatic high-contrast 2D grid view for devices without WebGL acceleration or users preferring standard interfaces.

---

## Quick Start (2 Minutes)

### Option 1: Instant Docker Run (Tanpa Clone Repo)

Jalankan langsung container siap pakai dari GitHub Container Registry:

```bash
docker run -d \
  --name dioramaops-hub \
  -p 7437:7437 \
  -e ADMIN_SECRET="ganti_dengan_token_rahasia_anda" \
  -v $(pwd)/data:/app/data \
  ghcr.io/ramhandean/diorama-ops:latest
```

Buka `http://localhost:7437` di browser.

---

### Option 2: Docker Compose Mandiri (Paket `docker/`)

Unduh template compose tanpa clone seluruh repo:
```bash
curl -O https://raw.githubusercontent.com/ramhandean/diorama-ops/main/docker/docker-compose.yml
curl -O https://raw.githubusercontent.com/ramhandean/diorama-ops/main/docker/.env.example
cp .env.example .env
# Sesuaikan ADMIN_SECRET pada .env lalu jalankan:
docker compose up -d
```

---

### Option 3: VPS Git Clone & `deploy.sh` (Untuk Modifikasi / Kontributor)

```bash
git clone https://github.com/ramhandean/diorama-ops.git
cd diorama-ops
./deploy.sh
```

Buka browser di `http://localhost:7437` untuk mengeksplorasi diorama telemetri 3D.

---

### Option 3: Run from Binary

```bash
# Build binary
go build -o dioramaops ./cmd/dioramaops

# Run in demo mode with sample tenants
DEMO_MODE=true ADMIN_SECRET="secret123" ./dioramaops serve
```

---

## Tracker Installation

Add a single script tag to the `<head>` or `<body>` of your website:

```html
<script defer src="https://diorama.yourdomain.com/m.js" data-key="pk_your_site_key_here"></script>
```

- Tracks pageviews and real-time active presence.
- Automatically handles Single Page Application (SPA) route changes.
- Pauses heartbeats when the user switches browser tabs.
- Silently degrades to POST beacon if WebSockets are blocked.

---

## Configuration

All configuration is provided via environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `7437` | HTTP port for the hub server |
| `ADMIN_SECRET` | *(Required)* | Secret token for admin dashboard and API authentication |
| `PUBLIC_URL` | `http://localhost:7437` | Public base URL used to construct embed snippets |
| `DATA_DIR` | `./data` | Directory for SQLite database (`dioramaops.db`), backups, and cached favicons |
| `STATIC_DIR` | `./web/dist` | Directory containing built frontend assets |
| `PUBLIC_VIEW` | `scene-only` | Visibility level: `scene-only` (3D only), `full` (public analytics), `private` (secret-gated) |
| `TRUSTED_PROXIES` | `127.0.0.1/32,...` | CIDR networks allowed to set `X-Forwarded-For` or `X-Real-IP` |
| `TZ` | `Asia/Jakarta` | Timezone for hourly/daily rollups and daily salt rotations |
| `RETENTION_DAYS` | `395` | Retention period in days before pruning old analytics rollups |
| `DEMO_MODE` | `false` | When `true`, automatically generates demo booths and virtual visitor traffic |
| `METRICS_TOKEN` | *(None)* | Optional Bearer token required to access `/metrics` |

---

## Prometheus & Grafana Monitoring

DioramaOps exports metrics in standard Prometheus exposition format on `GET /metrics`:

- `dioramaops_active_visitors{tenant="slug"}`
- `dioramaops_hits_total{tenant="slug"}`
- `dioramaops_sessions_total{tenant="slug"}`
- `dioramaops_avg_dwell_seconds{tenant="slug"}`
- `dioramaops_tenant_up{tenant="slug"}`
- `dioramaops_ws_connections`
- `dioramaops_ingest_dropped_total{reason="rate_limit"}`

A sample Prometheus scrape config and Grafana dashboard are located in the `deploy/` directory.

---

## CLI Commands

The `dioramaops` binary includes subcommands for operations and container health checks:

```bash
dioramaops serve                     # Start the telemetry server
dioramaops healthcheck               # Probe health endpoint (used by Docker HEALTHCHECK)
dioramaops version                   # Display version information
dioramaops export [output.json]      # Export all registered tenants from SQLite to JSON
dioramaops import <input.json>       # Import tenants into SQLite from JSON
```

---

## License

DioramaOps is licensed under the [MIT License](LICENSE).
