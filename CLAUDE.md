# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Backend (Go)
```bash
go build -o ./bin/server ./cmd/server
./bin/server
```

### Frontend (Next.js)
```bash
cd web
npm run dev        # Development server on :3000
npm run build      # Production build (standalone output)
npm run lint       # ESLint
```

### E2E Tests (Playwright)
```bash
cd web
npm run test:e2e           # Headless
npm run test:e2e:ui        # Playwright UI mode
npm run test:e2e:headed    # With browser visible
```

### Local Development Infrastructure
```bash
docker compose up -d postgres redis   # Start PostgreSQL + Redis
```

### Production Docker Build
```bash
docker build -t series-tracker --build-arg NEXT_PUBLIC_API_URL=https://your-domain.com .
```

## Architecture

**SeriesChecker** is a series/movie tracking app. Users search via the Kinopoisk API, add series to a watchlist, and track progress (season/episode).

### Request Routing
The Go server listens on `:8080` and reverse-proxies all non-API traffic to the Next.js app on `:3000`. API routes are handled directly by Go:
- `/api/auth/*` — authentication
- `/api/series/*` — search, details, similar/related
- `/api/progress/*` — user tracking (protected)
- `/health` — health check

### Backend (Go) — Layered Architecture
```
HTTP → Middleware (CORS → Recovery → Logger → Auth) → Handler → Service → Repository → DB/Redis
```
- `cmd/server/main.go` — wires all dependencies and defines routes
- `internal/handler/` — HTTP handlers (no business logic)
- `internal/service/` — business logic; `SeriesService` checks PostgreSQL cache before hitting Kinopoisk API
- `internal/repository/` — database abstraction wrapping SQLC-generated code
- `internal/database/` — SQLC-generated type-safe SQL (source queries in `sqlc/queries/`)
- `internal/kinopoisk/` — Kinopoisk API client

**Auth:** Session-based with HTTP-only cookies. Sessions stored in Redis (`session:{id}` → JSON user data, 7-day TTL). No JWT.

**SQLC pattern:** Edit `.sql` files in `sqlc/queries/`, run `sqlc generate` to regenerate `internal/database/sqlc/`.

### Frontend (Next.js App Router)
All pages are client components. State management is split between:
- **`AuthContext`** (`web/components/auth-context.tsx`) — global auth state, calls `/api/auth/me` on mount
- **TanStack Query** — server state for series search and progress data
- **`web/lib/api.ts`** — single typed API client used everywhere; all fetches use `credentials: 'include'`

`web/app/page.tsx` is the main page handling both the search/autocomplete flow and the progress list.

### Database Schema
Three core tables: `users`, `series` (Kinopoisk data cached in Postgres), `user_progress` (user's watch tracking with season/episode/status).

Migrations are plain SQL files in `migrations/` and are applied automatically by `scripts/entrypoint.sh` on container start.

## Workflow

After editing any file `A`, create or update a sibling `A.md` in the same directory describing:
1. What the file does (brief overview)
2. What was changed and why

This allows quickly recovering context when returning to a file in a future conversation.

### Environment Variables
See `.env.example`. Key variables:
- `KINOPOISK_API_KEY` — required for series search
- `NEXT_PUBLIC_API_URL` — injected at Next.js build time (Docker build arg)
- `SESSION_COOKIE_NAME`, `COOKIE_DOMAIN`, `ALLOWED_ORIGINS`
