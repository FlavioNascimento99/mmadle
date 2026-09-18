# MMAdle — Daily MMA Fighter Guessing Game

A Wordle-style daily guessing game for MMA/UFC fighters. Each day one hidden
fighter is selected deterministically; players search and submit fighters, and
the backend returns structured per-attribute comparisons (age, division,
height, record, nationality, last UFC event) without ever revealing the target.

This project is a **software-engineering learning project**: clean boundaries,
explicit contracts, and tests are prioritized over cleverness.

## Stack

| Layer    | Tech                                                              |
| -------- | ----------------------------------------------------------------- |
| Frontend | Next.js 14 (App Router), React, TypeScript, Tailwind CSS, Zod     |
| Backend  | Go (stdlib `net/http` only), `pgx` v5, `sqlc`-managed SQL queries |
| Database | PostgreSQL 16, versioned SQL migrations, `pg_trgm` search indexes |
| Infra    | Docker + Docker Compose, GitHub Actions                           |

## Architecture

Modular monolith. The backend is authoritative; the frontend only displays.

```mermaid
flowchart LR
    subgraph Frontend [Next.js]
        UI[GameBoard / SearchBar / GuessTable]
        ZOD[Zod API contracts]
    end
    subgraph Backend [Go stdlib net/http]
        H[httpapi: handlers + validation + CORS]
        D[domain: compare + age + selector]
        S[store: pgx repositories]
    end
    subgraph DB [PostgreSQL]
        PG[(fighters · divisions · events · fights)]
    end
    UI -->|search / guess JSON| H
    H --> D
    H --> S
    S --> PG
    D -.- S
```

Request flow for a guess:

```text
POST /api/game/guess {"fighter_id": 8}
     ↓  validate body (422-style 400s, unknown fields rejected)
HTTP Handler (internal/httpapi)
     ↓  load guess view + resolve today's target id (hash selector)
Service wiring (same package, thin)
     ↓  domain.EvaluateGuess(target, guess, gameDate)
Domain logic (internal/domain, pure, no I/O)
     ↓  store.FighterView (parameterized SQL, derived division/event)
Repository (internal/store) → PostgreSQL
```

## Project structure

```text
backend/
├── cmd/api/main.go            # entrypoint: migrate → serve
├── cmd/importer/main.go       # data-ingestion boundary (seed import)
├── internal/domain/           # pure game logic + tests
├── internal/httpapi/          # routes, validation, CORS + tests
├── internal/store/            # pgx repositories + integration tests
├── migrations/001_*.sql …     # versioned schema (002–006) + seed.sql
├── queries/fighters.sql       # sqlc query definitions
├── sqlc.yaml                  # sqlc codegen config
└── Dockerfile
frontend/
├── app/ (page, layout)        # main game screen
├── components/                # Header, SearchBar, GuessTable, GameBoard
└── lib/ (api.ts, game.ts)     # zod contracts, comparison display + tests
docker-compose.yml
.github/workflows/ci.yml
```

## Local setup (without Docker)

Requirements: Go 1.24+, Node 20+, PostgreSQL 16+.

```bash
cp .env.example .env

# 1. create database
createdb mmadle  # or: psql -c "CREATE DATABASE mmadle;"

# 2. backend: migrations run automatically on startup
cd backend
export DATABASE_URL="postgres://mmadle:mmadle_dev_password@localhost:5432/mmadle?sslmode=disable"
go run ./cmd/api -migrate-only
go run ./cmd/importer --seed ./migrations/seed.sql
go run ./cmd/api            # listens on :8080

# 3. frontend (new terminal)
cd frontend
npm install
export NEXT_PUBLIC_API_URL=http://localhost:8080
npm run dev                 # http://localhost:3000
```

## Docker setup

```bash
cp .env.example .env
docker compose up --build
# frontend → http://localhost:3000, backend → http://localhost:8080
```

The backend container runs migrations on boot. Seed data is applied once:

```bash
docker compose exec backend /app/importer --seed /app/migrations/seed.sql
# (or locally: cd backend && go run ./cmd/importer --seed ./migrations/seed.sql)
```

Reset local database:

```bash
docker compose down -v          # drops the postgres volume
docker compose up --build -d postgres
# then migrate + seed as above
```

## Deploying to Cloudflare Workers

One Worker (`wrangler.jsonc`, `worker/index.ts`) serves the whole app:

- **Frontend** — `next build` emits a static export (`frontend/out`) that is
  served as Workers static assets. `NEXT_PUBLIC_API_URL` is left unset, so the
  UI calls the API same-origin.
- **Backend** — requests to `/api/*` run the Worker first, which forwards them
  to the Go image (`backend/Dockerfile`) running as a Cloudflare Container.
  Migrations run on container boot; the container sleeps after 10 idle
  minutes, so the first request after that takes a few seconds to wake it.

Deploys happen through Workers Builds (Git integration, repo root, default
`npx wrangler deploy`) or locally with `npm install && npx wrangler deploy`
from the repo root (requires Docker to build the image).

The only secret is the Postgres URL (any Postgres 16+ with `pg_trgm`):

```bash
npx wrangler secret put DATABASE_URL
```

Seed a fresh database once by running the importer against it:

```bash
docker build -t mmadle-api backend
docker run --rm -e DATABASE_URL=... mmadle-api /app/importer --seed /app/migrations/seed.sql
```

Docker Compose is unaffected: the frontend image sets `NEXT_OUTPUT=standalone`
to build the `next start` server instead of the static export.

## Environment variables

See `.env.example`. Summary:

| Var                  | Used by | Default                                  |
| -------------------- | ------- | ---------------------------------------- |
| `DATABASE_URL`       | backend | postgres URL for local `mmadle` db       |
| `PORT`               | backend | `8080`                                   |
| `ALLOWED_ORIGINS`    | backend | `http://localhost:3000`                  |
| `GAME_TIMEZONE`      | backend | `UTC`                                    |
| `SESSION_TTL`        | backend | `720h` (Go duration; session cookie lifetime) |
| `SESSION_COOKIE_SECURE` | backend | `true`, except `false` for plain-http local dev |
| `MIGRATIONS_DIR`     | backend | `migrations` (`/app/migrations` in Docker) |
| `NEXT_PUBLIC_API_URL`| frontend| unset (same-origin); `http://localhost:8080` in `next dev` |
| `TEST_DATABASE_URL`  | backend tests | postgres URL for integration tests |

## Migrations

Versioned SQL in `backend/migrations/`, applied in lexical order by the API
on startup and tracked in `schema_migrations`:

- `001_extensions.sql` — `pg_trgm` for search
- `002_create_divisions.sql` — weight classes
- `003_create_fighters.sql` — canonical fighter data (no derived fields)
- `004_create_fighter_divisions.sql` — many-to-many + `is_current` flag
- `005_create_events_fights.sql` — events + fights (last event is derived)
- `006_create_indexes.sql` — trigram/B-tree indexes
- `007_add_fighter_photo_credit.sql` — mandatory attribution whenever a photo is set
- `008_add_division_gender.sql` — `men`/`women` on divisions for pool filtering
- `009_auth.sql` — `users`, `sessions` (hashed opaque tokens), `game_guesses`
- `seed.sql` — demo dataset (applied separately, see below)

## Seed / data import

`backend/cmd/importer` is the **ingestion boundary**: the API never scrapes or
imports data at request time. Today it applies `migrations/seed.sql` (67
fighters across 11 divisions, 36 events, 59 fights; every fighter is eligible as
a daily target). The seed is idempotent, so re-running it on an existing
database updates the data in place. It is designed to be replaced by a pipeline
consuming a real MMA/UFC source; only this command and the SQL files change,
not the runtime.

Fighter photos are free-licensed Wikimedia Commons images (public domain,
CC0, CC BY, CC BY-SA), picked from each fighter's English Wikipedia lead image
and hotlinked from Wikimedia. Each one is stored with its author and licence in
`photo_credit`. The UI shows the credit in the photo tooltip and under the
winner's portrait. Fighters without a free photo get an initials fallback.

## API endpoints

| Method | Path                    | Notes                                              |
| ------ | ----------------------- | -------------------------------------------------- |
| GET    | `/api/health`           | `{"status":"ok"}` (503 if DB unreachable)          |
| GET    | `/api/game/today`       | `{"date":"2026-09-18","status":"playing"}` — never reveals the target |
| GET    | `/api/fighters`         | full roster, alphabetical, same minimal fields as search |
| GET    | `/api/fighters/search?q=` | case-insensitive partial match, max 8 results, minimal fields |
| POST   | `/api/game/guess`       | `{"fighter_id": 8}` → structured comparison (see below) |
| POST   | `/api/auth/register`    | `{"email","password","display_name?"}` → account + session cookie |
| POST   | `/api/auth/login`       | `{"email","password"}` → session cookie (errors never reveal whether the email exists) |
| POST   | `/api/auth/logout`      | clears the session cookie (idempotent)   |
| GET    | `/api/auth/me`          | signed-in account, or 401 for guests     |
| GET    | `/api/me/guesses`       | `?pool=&date=` → signed-in history, re-evaluated server-side |
| POST   | `/api/me/import`        | `{"pool","date","fighter_ids":[]}` → import local guesses after sign-in |

Guess response (values are the **guessed** fighter's; target stays hidden):

```json
{
  "fighter_id": 8,
  "fighter_name": "Ilia Topuria",
  "results": {
    "age": { "value": 29, "comparison": "higher" },
    "division": { "value": "Lightweight", "comparison": "incorrect" },
    "height": { "value": 170, "comparison": "lower" },
    "record": { "value": "17-0-0", "comparison": "incorrect" },
    "nationality": { "value": "Georgia", "comparison": "incorrect" },
    "last_event": { "value": "UFC 320", "comparison": "incorrect" }
  },
  "correct": false
}
```

Comparison semantics: `correct` / `incorrect` for exact attributes;
`higher` / `lower` for age & height, meaning the **target** is higher/lower
than the guess (frontend shows ↑ / ↓).

## Testing

```bash
cd backend
go test ./...            # unit + (with TEST_DATABASE_URL) postgres integration
go vet ./...
golangci-lint run ./...  # or: go vet (CI runs golangci-lint)

cd frontend
npm test                 # vitest: contracts + comparison display
npm run lint && npm run typecheck && npm run build
```

## Important decisions (explicit)

1. **No Go web framework** — stdlib `net/http` + `ServeMux` is enough for four
   routes and keeps the dependency surface minimal.
2. **Displayed division = `is_current` flag** on `fighter_divisions` (exactly
   one per fighter, enforced by partial unique index). It represents the
   fighter's most recent recorded UFC division. Alternative considered:
   deriving division from the latest fight — rejected because fights don't
   carry a division and cross-division bouts would make it ambiguous.
3. **Last event derived** via `ORDER BY event.date DESC LIMIT 1` over the
   fighter's fights — never stored on the fighter row.
4. **Age from `date_of_birth` relative to the game date** (injectable `Clock`),
   so daily attributes are deterministic and testable.
5. **Daily target = FNV-1a(date) mod N** over `ORDER BY id` eligible fighters
   (must have current division + ≥1 fight). Behind the `Selector` interface so
   a curated schedule can replace it without touching handlers.
6. **Search = `ILIKE` + `pg_trgm` GIN index.** No Elasticsearch; sufficient for
   roster scale, documented upgrade path in migration 006.
7. **Record `23-5-0` / `23-5-0-1 NC`** is formatted in the domain from four
   integer columns; comparison requires all four to match.
8. Frontend persists guesses in `localStorage` per game date; the backend
   remains the authority on correctness (no auth in MVP).
9. **Accounts are first-class but optional.** Registration is email + password
   (argon2id) with an optional display name; sessions are opaque tokens in an
   `HttpOnly; Secure; SameSite=Lax` cookie, stored hashed server-side with
   rotation on login. Signed-in guesses are recorded in `game_guesses` and
   re-evaluated on read, so another device restores the same board and
   client-supplied outcomes are never trusted. Login errors are identical for
   unknown emails and wrong passwords; state-changing routes require
   `Content-Type: application/json` as a CSRF defense alongside SameSite=Lax.

## Future extension points

Stance is already stored; reach, last-fight result, UFC fight count,
streaks/leaderboards, shareable results, auth, and non-UFC organizations can be
added as new `domain` result fields + view columns without rewriting the
comparison core.
