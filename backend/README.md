# Recent iOS App Store Reviews — Backend (Go)

A tiny Go service that:
- polls App Store **Customer Reviews RSS**,
- stores reviews on disk (append-only **JSONL**),
- serves an HTTP API for reviews from the **last N hours** (default 48h).

## Key Decisions

- **Language**: Go, standard library only.
- **Persistence**: `data/reviews/<appId>-<country>.jsonl` + `data/state.json` (seen IDs + lastPoll).
- **Idempotency**: dedupe by review `id`.
- **Ordering**: API returns *newest first*.
- **Multi-app**: iterate over `{appId,country}` pairs from `config/apps.json`.
- **Resilience**:
  - Per-app **retry** (exponential backoff + jitter) per page.
  - Per-app **circuit breaker** (Closed/Open/Half-Open).
  - **Webhook** on final failure with `{ id, timestamp, errorType }`.
  - **Graceful shutdown** on SIGINT/SIGTERM.

## Project Structure
```
backend/
├─ cmd/server/main.go # entrypoint (HTTP + shutdown)
├─ config/apps.json # config (poll interval, apps, webhook, CB)
├─ data/
│ ├─ reviews/ # JSONL files
│ └─ state.json # seenIds + lastPoll (atomic writes)
└─ internal/ # single package "internal"
├─ api.go # routes & JSON helpers
├─ poller.go # poll manager + per-app workers
├─ store.go # file persistence
├─ apple_feed.go # fetch & parse Apple RSS (with retry)
├─ webhook.go # best-effort POST on failures
├─ circuit_breaker.go # simple CB per app
└─ types.go # data models & config parsing
```


## Requirements

- Go 1.21+
- `curl` or `httpie` (optional) for testing

## Configure

`config/apps.json` example:
```json
{
  "pollIntervalMinutes": 15,
  "webhookUrl": "http://localhost:9000/reviews-webhook",
  "circuitBreaker": {
    "failureThreshold": 3,
    "openCooldownSeconds": 60
  },
  "apps": [
    { "appId": "595068606", "country": "us" },
    { "appId": "447188370", "country": "us" }
  ]
}
```
- Leave webhookUrl empty ("") to disable webhook.
- Add or remove apps as you like.

## Run
From repo root or backend/:
```
cd backend
go mod init backend # run once 
go run ./cmd/server
# Expected: "HTTP server listening on :8080"
```

First run creates:

- `data/reviews/<appId>-<country>.jsonl`
- `data/state.json` (atomic write via `*.tmp` + rename)

## API

Base: `http://localhost:8080`

- **Health**
```
GET /health
-> { "status": "ok" }
```

- **Configured Apps**
```
GET /apps
-> [ { "appId": "595068606", "country": "us" }, ... ]
```

- **Trigger one poll (async)**
```
POST /poll?appId=595068606&country=us
-> 202 Accepted
```

- **Recent reviews (default 48h)**
```
GET /reviews?appId=595068606&country=us&hours=48
-> {
     "appId": "...", "country": "...",
     "from": "ISO", "to": "ISO",
     "count": N,
     "reviews": [
       { "id": "...", "author": "...", "rating": 5, "content": "...", "submittedAt": "ISO" },
       ...
     ]
   }
```


**Notes:**
- hours validated (1…2160).
- Reviews are sorted newest-first.


## Quick Smoke Test (HTTPie)

Install
```
# macOS
brew install httpie jq
# Debian/Ubuntu
sudo apt-get update && sudo apt-get install -y httpie jq
```

Run
```

# Default (localhost:8080, app 595068606/us, 48h)
./scripts/smoke.sh

# App diversa e finestra più ampia (7 giorni)
APP_ID=447188370 COUNTRY=us HOURS=168 ./scripts/smoke.sh

```

(See `scripts/smoke.sh` for details; it checks `/health`, `/apps`, triggers `/poll`, and reads `/reviews` with short retries.)

## Future Improvements

- SQLite/Postgres for richer queries and indexes.
- Prometheus metrics, structured logs, tracing.
- Config hot-reload; per-app rate limits.
- SSE/WebSocket for live push to clients.