# Recent iOS App Store Reviews — Frontend (React + Vite + TypeScript)

Small React app that calls the backend and shows App Store reviews from the **last N hours** (default 48h).  
UI shows author, stars, relative time + ISO timestamp, and the content.

## Key Decisions

- **Vite + TS + React** for fast dev & simple build.
- **API** calls `/apps` and `/reviews` (optionally `/poll` if you want a “Fetch now” button).
- **Time window** selectable (presets + custom hours with debounce).
- **Type-only imports** to satisfy modern TS (`verbatimModuleSyntax`).

## Requirements

- Node 18+ and npm

## Setup

```bash
cd frontend
npm install
# Option A: use API base env var
echo "VITE_API_BASE=http://localhost:8080" > .env.local
npm run dev
# open http://localhost:5173
```

## Scripts
```bash
npm run dev      # start dev server
npm run build    # production build to dist/
npm run preview  # preview production build locally
```

## UI Overview
- *App selector* from `/apps` endpoint.
- *Time window* presets (12h, 24h, 48h, …) + custom hours with debounce (to avoid spamming the API).
- *Refresh* button to re-fetch.
- *Review Card* with:
  - Avatar initial,
  - Stars + numeric badge (e.g., `4/5`),
  - Relative time and ISO timestamp (e.g., `2 hours ago · 2025-10-14T07:12:33Z`),
  - Content with preserved line breaks.

## Future Improvements
- Filters (min rating), search.
- Auto-refresh toggle.
- Pagination or infinite scroll for large windows.
- Theming toggle; extract components to a UI library.