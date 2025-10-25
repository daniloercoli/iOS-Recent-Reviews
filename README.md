# Recent iOS App Store Reviews — Monorepo

A tool that automates the collection and analysis of iOS app reviews, making it easier for developers to stay updated on user feedback. 

This repo contains:

- **Backend (Go)** — polls App Store Customer Reviews RSS, stores reviews on disk, exposes an API to read the last *N* hours (default 48h).  
  → See **[backend/README.md](backend/README.md)**

- **Frontend (React + Vite + TypeScript)** — calls the backend and displays reviews with a selectable time window.  
  → See **[frontend/README.md](frontend/README.md)**

---

## Quick Start

### 1) Backend

```bash
cd backend
go mod init backend   # run once (use your module name if different)
go run ./cmd/server
# => HTTP server listening on :8080
```

Check:
```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/apps | jq
curl -s "http://localhost:8080/reviews?appId=595068606&country=us&hours=48" | jq
```

More: [backend/README.md](backend/README.md)


### 2) Frontend

```bash
cd frontend
npm install
# either set API base:
echo "VITE_API_BASE=http://localhost:8080" > .env.local
npm run dev
# open http://localhost:5173
```
More: [frontend/README.md](frontend/README.md)

## Repository Structure
```
/
├─ backend/
│  ├─ cmd/server/main.go
│  ├─ config/apps.json
│  ├─ data/            # created at runtime (reviews JSONL + state.json)
│  └─ internal/        # API, poller, store, feed, circuit breaker, etc.
└─ frontend/
   ├─ src/
   ├─ index.html
   ├─ vite.config.ts (optional proxy)
   └─ .env.local (VITE_API_BASE)
```

## Design at a Glance

- **Backend**
  - Go stdlib; JSONL (`data/reviews/*.jsonl`) + `state.json`(seen IDs, lastPoll).
  - Per-app *retry* (exponential + jitter) per feed page.
  - Per-app *circuit breaker* (Closed/Open/Half-Open).
  - Webhook on final failures: `{ id, timestamp, errorType }`.
  - Graceful shutdown (pollers + HTTP).

- **Frontend**
   - React + Vite + TypeScript.
   - Selectable time window (default 48h).
   - Shows relative time *and* exact ISO timestamp for debugging.

## License
MIT





