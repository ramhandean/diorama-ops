# DioramaOps Telemetry & Real-Time Protocol Specification

Version: 1.0.0  
Status: Draft / Standard

This document outlines the wire protocol between tenant website trackers (`m.js`), the DioramaOps backend hub, and connected frontend 3D dashboards.

---

## 1. Protocol Overview & Transports

Telemetry supports two transport channels:
1. **Primary: WebSocket (`/ws`)**
   - Bi-directional full-duplex connection.
   - Low latency heartbeat and navigation events.
   - Used by web clients for real-time live presence and dashboard synchronization.
2. **Fallback: HTTP Beacon (`/api/v1/b`) & SSE (`/events`)**
   - Used when WebSocket connection is blocked by client firewalls or corporate proxies.
   - `navigator.sendBeacon()` or `fetch(..., { keepalive: true })` for reliable `bye` notifications during tab close (`pagehide`).
   - Server-Sent Events (`/events`) stream dashboard updates downstream if WebSocket cannot upgrade.

---

## 2. Limits & Constraints

- **Maximum Message Size:** 1024 bytes (1 KB) per payload. Messages exceeding this limit are discarded.
- **Rate Limit:** Capped per IP address and per site key. Violations increment `dioramaops_ingest_dropped_total{reason="rate_limit"}`.
- **Bot & DNT Filtering:** Requests with `DNT: 1` or standard bot user-agents (`bot`, `crawler`, `spider`) are dropped immediately with zero side-effects.

---

## 3. Client-to-Hub Messages (Ingest)

All incoming messages from tracked sites are JSON objects identified by the `t` (type) discriminator.

### 3.1 `hello` (Initial Handshake)
Sent when a tenant page first loads.
```json
{
  "t": "hello",
  "key": "pk_tenant_abcdef123456",
  "sid": "s_9f83b2a1",
  "path": "/docs/getting-started",
  "ref": "news.ycombinator.com"
}
```
- `key` (string, required): Public tenant site key.
- `sid` (string, required): Session/tab ID generated client-side. Unique per browser tab.
- `path` (string, required): Cleaned URL path (query strings and hash fragments stripped).
- `ref` (string, optional): Cleaned referrer hostname (host only, no paths/query).

### 3.2 `hb` (Heartbeat)
Sent every 15 seconds while the browser tab remains active and visible.
```json
{
  "t": "hb",
  "sid": "s_9f83b2a1",
  "path": "/docs/getting-started"
}
```
- Heartbeats are paused when `document.visibilityState === 'hidden'`.
- Resumed immediately when tab visibility returns to `'visible'`.

### 3.3 `nav` (SPA Route Navigation)
Sent on client-side route changes (`pushState` / `popstate`).
```json
{
  "t": "nav",
  "sid": "s_9f83b2a1",
  "path": "/pricing"
}
```

### 3.4 `bye` (Session Termination)
Sent during `pagehide` / unload via WebSocket or `sendBeacon`.
```json
{
  "t": "bye",
  "sid": "s_9f83b2a1"
}
```

---

## 4. Hub-to-Client Messages (Dashboard Broadcast)

Broadcasts to 3D dioramas and HUD dashboards are batched and throttled at 4 to 10 Hz (100–250 ms ticks).

### 4.1 `snap` (Initial State Snapshot)
Sent immediately upon dashboard connection.
```json
{
  "t": "snap",
  "tenants": [
    {
      "id": "blog",
      "name": "Personal Blog",
      "active": 3,
      "status": "up"
    },
    {
      "id": "app",
      "name": "Cloud Dashboard",
      "active": 12,
      "status": "up"
    }
  ]
}
```

### 4.2 `enter` (Visitor Spawn)
Broadcast when a new unique visitor (`vid`) enters a tenant booth.
```json
{
  "t": "enter",
  "tenant": "blog",
  "vid": "v_e3b0c44298fc1c14"
}
```
- Triggers 3D avatar spawn at lobby and path lerp towards the tenant booth.

### 4.3 `leave` (Visitor Despawn)
Broadcast when a visitor exits or their session TTL expires.
```json
{
  "t": "leave",
  "tenant": "blog",
  "vid": "v_e3b0c44298fc1c14"
}
```
- Triggers 3D avatar wave/dissolve animation followed by unmount.

### 4.4 `status` (Uptime State Change)
Broadcast when a tenant uptime status changes between `up`, `degraded`, or `down`.
```json
{
  "t": "status",
  "tenant": "blog",
  "state": "degraded"
}
```

---

## 5. Presence & Dwell Time Semantics

1. **Session TTL:** In-memory presence records expire after **45 seconds** without an incoming heartbeat.
2. **Deduplication:** Multiple tabs opened by the same visitor share the same `vid` (calculated on the hub from `IP + UA + site_key + daily_salt`). Multiple tabs yield one visual avatar in the diorama booth.
3. **Dwell Time:** Calculated strictly as accumulated heartbeat intervals (15 seconds per heartbeat tick), preventing inflated dwell numbers from stalled tabs.
