# DioramaOps Security & Threat Model

This document outlines the security architecture, trust boundaries, and threat model of DioramaOps.

---

## 1. Trust Boundaries & Credentials

DioramaOps separates privileges across two distinct credential tiers:

1. **Public Site Key (`pk_*`)**
   - Embedded publicly in client websites via `<script data-key="...">`.
   - Bound strictly to the tenant's configured `origins` allowlist.
   - Incoming WebSocket and Beacon requests from unlisted origins are immediately rejected.
2. **Administrative Secret (`ADMIN_SECRET`)**
   - Kept strictly private on the host machine.
   - Evaluated using constant-time cryptographic hash comparison (`subtle.ConstantTimeCompare`) to prevent timing side-channel attacks.
   - Required for tenant mutations, logo uploads, and JSON sync.

---

## 2. Privacy & Anonymity Guarantees

- **Zero Cookies & Local Storage Tracking:** The `m.js` tracker stores no identifiers in client cookies or browser persistent storage.
- **Ephemeral Visitor Hashing:**
  $$\text{vid} = \text{Truncate}_{16}(\text{HMAC-SHA256}(\text{IP} + \text{UA} + \text{SiteKey}, \text{DailySalt}))$$
  - `DailySalt` is generated randomly in volatile server RAM and rotated every midnight (server timezone).
  - The salt is **never written to persistent disk**, guaranteeing forward anonymity across days.
- **No Raw IP Storage:** Client IP addresses are discarded immediately after computing the ephemeral hash.
- **Query & Referrer Sanitization:** URL query strings and hash anchors are stripped. Referrers are truncated to domain names only.

---

## 3. Server-Side Request Forgery (SSRF) Defense

When fetching remote website favicons server-side:
1. **IP Range Blacklisting:** Hostnames are resolved to IP addresses before dialing. Any IP matching RFC 1918 private subnets, loopback (`127.0.0.0/8`), link-local (`169.254.0.0/16`), AWS metadata services, IPv6 loopback (`::1`), or ULA addresses are rejected with `ErrPrivateIP`.
2. **Redirect Validation:** Redirects are limited to 3 hops and each hop is re-validated against the IP blacklist.
3. **Resource Capping:** Responses are bounded by a 5-second connection timeout and a 128 KB maximum body read limit.

---

## 4. Logo Upload & SVG Sanitization

- **Size Constraint:** File uploads are strictly capped at 512 KB.
- **SVG Disarming:**
  - `<script>` tags, inline event attributes (`onload`, `onclick`, `onerror`), and `<foreignObject>` elements are stripped.
  - External resource links (`href`, `xlink:href`) are discarded.
  - Uploaded SVGs are served strictly via `<img>` tags in the browser, preventing DOM script execution context.

---

## 5. Network Hardening

- The production container runs with non-root user (`UID 65532`), a read-only root filesystem (`read_only: true`), all Linux capabilities dropped (`cap_drop: ALL`), and `no-new-privileges: true`.
- Internal port `7437` is bound only to `127.0.0.1` when using host reverse proxy, preventing direct public access bypassing Caddy.
- The `/metrics` endpoint is blocked by default in Caddy from external public traffic.
