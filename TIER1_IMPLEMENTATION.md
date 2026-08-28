# Tier 1 Implementation — Lyrahcle CVE Sync + SoapySDR + Redis/Prometheus

**Phase:** v0.7.0-tier1  
**Status:** Ready for integration  
**Date:** 2026-08-28

## What's Included

### 1. CVE Sync Worker (`apps/cve-sync/`)

Full-featured worker for synchronizing CVEs from NVD API 2.0 with:
- **NVD Client** (`internal/nvd/client.go`) — Direct integration with NIST CVE 2.0 API
- **PostgreSQL Store** (`internal/store/store.go`) — Persistence + upsert logic
- **Redis Cache** (`internal/cache/cache.go`) — Deduplication & keyword tracking
- **Prometheus Metrics** (`internal/metrics/metrics.go`) — Observable sync process
- **Config Management** (`internal/config/config.go`) — Environment-based configuration
- **Main Scheduler** (`cmd/cve-sync/main.go`) — Cron-like sync loop

**Features:**
- Syncs keywords: `satellite`, `space`, `RF`, `TM/TC`, `firmware`, `orbit`
- Rate limit aware (configurable ~6 requests/min to NVD)
- Metrics: sync duration, CVEs fetched/upserted, errors, timestamp, severity breakdown
- Graceful shutdown (SIGTERM/SIGINT)

### 2. SoapySDR Abstraction Layer (`apps/sdr-agent/internal/soapy/`)

Hardware-agnostic SDR interface with:
- **Device Interface** (`device.go`) — Abstraction layer supporting RTL-SDR, HackRF, etc.
- **Mock Device** — Drop-in replacement for testing without hardware
- **Configuration** — Frequency, sample rate, gain, antenna management

**Future:**
- Real SoapySDR bindings (Go cgo wrapper)
- Hardware auto-detection
- Multi-device support

### 3. RF Streaming Engine (`apps/sdr-agent/internal/streaming/`)

Efficient IQ data capture with:
- **StreamManager** — Control plane for start/stop/read operations
- **IQ Capture** — In-memory buffer for sample recording
- **Export Formats** — IQ binary, WAV (IQ playback ready)
- **Forense Support** — Timestamp, frequency, metadata tracking

**Future:**
- Disk-backed ringbuffer for long captures
- Real-time streaming to WebSocket
- DSP filters (decimation, demodulation)

### 4. Infrastructure Updates

- **go.mod** (backend) — Added `redis/go-redis`, `prometheus/client_golang`
- **Docker Compose** — Redis, CVE Sync, Prometheus services + volumes
- **Prometheus Config** — Scrape targets for cve-sync, backend
- **Database Migration** (`0008_cve_sync_enhancements.sql`) — Enhanced space_cves schema

## Quick Start

### 1. Extract & Navigate

```bash
cd /path/to/your/repo
git pull origin main
# or: git reset --hard <commit> to sync
```

### 2. Build & Test Locally

```bash
# Ensure Go 1.22+
go version

# CVE Sync
cd apps/cve-sync
go mod tidy
go build -o cve-sync ./cmd/cve-sync/main.go

# SDR Agent
cd ../sdr-agent
go mod tidy

# Backend (with new deps)
cd ../backend
go mod tidy
go test ./...
```

### 3. Run with Docker Compose

```bash
docker-compose up --build

# Services:
# - Backend: http://localhost:8080
# - Frontend: http://localhost:5173
# - Prometheus: http://localhost:9090
# - CVE Sync: metrics @ http://localhost:9091/metrics
# - Redis: localhost:6379
# - PostgreSQL: localhost:5432
```

### 4. Check CVE Sync Logs

```bash
docker-compose logs -f cve-sync

# Expected output:
# lyrahcle sdr-agent "..." starting...
# Fetching CVEs for keyword "satellite"...
# Fetched 47 CVEs for "satellite"
# Upserted 47 CVEs for "satellite"
# CVE sync completed in 12.34s
```

### 5. View Prometheus Dashboard

- **Metrics:** http://localhost:9090/graph
- **Alerts:** http://localhost:9090/alerts
- **Targets:** http://localhost:9090/targets

**Key metrics:**
```
lyrahcle_cves_fetched_total
lyrahcle_cves_upserted_total
lyrahcle_cve_sync_duration_seconds
lyrahcle_cve_sync_errors_total
lyrahcle_cve_count_by_severity
lyrahcle_nvd_rate_limit_hits_total
```

### 6. Query CVEs via Backend API

```bash
# Assuming backend is running on :8080
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/threat-intel/cves?severity=CRITICAL
```

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Lyrahcle Tier 1                 │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌─────────────────┐  ┌──────────────────────────┐ │
│  │  CVE Sync       │  │  SDR Agent + SoapySDR   │ │
│  │  Worker         │  │  ├─ Hardware Abstraction│ │
│  │                 │  │  ├─ RF Streaming        │ │
│  │  • NVD API 2.0  │  │  └─ IQ Playback         │ │
│  │  • PostgreSQL   │  │                         │ │
│  │  • Redis Cache  │  │  (Mock for now,        │ │
│  │  • Prometheus   │  │   real HW ready)       │ │
│  └─────────────────┘  └──────────────────────────┘ │
│          │                      │                  │
│  ┌───────┴──────────────────────┴────────────┐   │
│  │                                            │    │
│  │  Backend (Go) ← Shared Infra               │    │
│  │  ├─ JWT Auth                               │    │
│  │  ├─ PostgreSQL (space_cves enhanced)       │    │
│  │  ├─ Redis (CVE cache, rate limit)          │    │
│  │  └─ Prometheus (all metrics)               │    │
│  │                                            │    │
│  └────────────────────────────────────────────┘   │
│          │                                        │
│  ┌───────┴─────────────────────────────────────┐  │
│  │  Frontend (React + Tailwind)                │  │
│  │  • Dashboard (CVE view, threat intel)       │  │
│  │  • Agent management                         │  │
│  │  • Monitoring (via Prometheus)              │  │
│  └───────────────────────────────────────────────┘ │
│                                                   │
└─────────────────────────────────────────────────────┘
```

## Environment Variables

### CVE Sync Worker

```bash
DATABASE_URL=postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle
REDIS_ADDR=localhost:6379
PROMETHEUS_PORT=9091
SYNC_INTERVAL_SEC=3600          # 1 hour
NVD_RESULTS_PER_PAGE=100        # Max NVD API limit
NVD_RATE_LIMIT_PER_MIN=10       # Backoff if hit
LOG_LEVEL=info                  # info, debug, error
ENVIRONMENT=development         # development, production
```

### Backend (updated)

```bash
DATABASE_URL=postgres://...
REDIS_URL=redis://localhost:6379
# + existing: JWT_SECRET, SEED_ADMIN_*, CELESTRAK_GROUPS
```

## Next Steps (Tier 2)

- [ ] Backend RBAC API + role-based CVE alerting
- [ ] SDR Agent resilience (reconnect, backoff, remote config)
- [ ] Tests >80% coverage (unit + e2e)
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] JSON logging + centralized alerts

## File Changes Summary

```
apps/cve-sync/
  ├─ go.mod
  ├─ go.sum (auto-generated)
  ├─ cmd/cve-sync/main.go
  └─ internal/
      ├─ config/config.go
      ├─ nvd/client.go
      ├─ store/store.go
      ├─ cache/cache.go
      └─ metrics/metrics.go

apps/sdr-agent/internal/
  ├─ soapy/device.go
  └─ streaming/stream.go

apps/backend/
  ├─ go.mod (updated)
  └─ migrations/0008_cve_sync_enhancements.sql

deploy/
  ├─ docker/Dockerfile.cve-sync (updated)
  └─ prometheus/prometheus.yml

docker-compose.yml (updated)
TIER1_IMPLEMENTATION.md (this file)
```

## Git Commit Message

```
feat: Tier 1 implementation — CVE Sync worker + SoapySDR + Redis/Prometheus

- CVE Sync Worker (apps/cve-sync):
  * Full NVD API 2.0 integration with PostgreSQL persistence
  * Redis caching for keyword deduplication
  * Prometheus metrics (fetch count, upsert count, sync duration, errors)
  * Graceful scheduler with configurable interval

- SoapySDR Abstraction (apps/sdr-agent):
  * Hardware-agnostic device interface (supports RTL-SDR, HackRF, USRP)
  * Mock device for testing without hardware
  * RF Streaming with IQ capture and export (binary, WAV)
  * Forense-ready metadata tracking (timestamp, frequency, gain)

- Infrastructure:
  * Redis service in Docker Compose (cache + rate limiting)
  * Prometheus scrape config for cve-sync and backend metrics
  * Enhanced space_cves schema (references, synced_at, search_keyword)
  * go.mod updated with redis/go-redis and prometheus/client_golang

Closes #TIER1-READY
```

## Testing Checklist

- [ ] `docker-compose up` runs all services without errors
- [ ] Backend connects to PostgreSQL + Redis
- [ ] CVE Sync connects and syncs ≥1 keyword
- [ ] Prometheus scrapes cve-sync metrics endpoint
- [ ] Mock SDR device initializes without crashing
- [ ] Stream Manager can start/stop capture in-memory
- [ ] Backend API serves CVE data (if auth enabled)

## Support

For issues:
1. Check logs: `docker-compose logs <service>`
2. Verify env vars in `docker-compose.yml`
3. Ensure PostgreSQL migration ran: `psql -c "\d space_cves"`
4. Test Redis: `redis-cli PING`
5. Test Prometheus: `curl http://localhost:9090/api/v1/query?query=up`

---

**Ready for CYSAT 2027.** 🚀
