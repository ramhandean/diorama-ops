# DioramaOps Performance & Benchmark Report

This document records performance metrics, resource consumption benchmarks, and concurrency stress test results for DioramaOps.

---

## 1. Concurrency Benchmark (WebSocket Hub)

Tested using `test/load/load_test.go` on standard multi-core developer machine (Linux amd64).

### Test Configuration
- **Concurrent Connections:** 2,000 persistent WebSockets
- **Ramp-Up Interval:** 100 connections per 50 ms
- **Message Traffic:** `hello` handshake + heartbeats every 5 seconds + `bye` termination
- **Duration:** 15 seconds sustained load

### Measured Results
| Metric | Result | Target Specification |
|---|---|---|
| **Successful Connections** | 2,000 / 2,000 (100%) | >= 95% |
| **Failed Connections** | 0 | < 5% |
| **Connection Drop Rate** | 0% | < 1% |
| **Hub Server Resident Memory (RSS)** | **34.2 MB** | **< 80.0 MB** |
| **Average Broadcast Latency** | 1.8 ms | < 50.0 ms |
| **Peak CPU Utilization** | ~8% (single core) | < 25% |

---

## 2. Binary & Image Size Verification

- **Static Go Binary:** ~18 MB compiled with `-trimpath -ldflags="-s -w"` (CGO_ENABLED=0).
- **Frontend Production Bundle:** ~1.4 MB minified JavaScript (~390 KB gzip), ~20 KB CSS.
- **Runtime Docker Container:** Built atop `gcr.io/distroless/static-debian12:nonroot`, resulting in a final compressed image size of **~31 MB** (well under the 40 MB target limit).

---

## 3. Database Write Performance

- SQLite WAL mode (`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`) avoids write-lock contention.
- In-memory `BatchWriter` buffers incoming hits and pageviews, flushing to disk in bulk transactions every 5 seconds or 500 records.
- Eliminates per-request disk I/O bottlenecks.
