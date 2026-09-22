.PHONY: help dev-up dev-down backend-test backend-lint backend-build frontend-install frontend-dev frontend-test frontend-lint frontend-build migrate seed seed-prod

help:
	@echo "Targets: dev-up dev-down backend-test frontend-test ... (see README)"

dev-up:
	docker compose up --build

dev-down:
	docker compose down

# ---- Backend ----
backend-test:
	cd backend && go test ./... && go vet ./...

backend-lint:
	cd backend && golangci-lint run ./... || go vet ./...

backend-build:
	cd backend && go build ./...

# DB helpers (require psql + DATABASE_URL)
migrate:
	cd backend && go run ./cmd/api -migrate-only 2>&1 | head -20; echo "migrations run on startup; see README"

seed:
	cd backend && go run ./cmd/importer --seed ./migrations/seed.sql

# Re-seed an existing prod/dev database with the built image. Idempotent.
# Usage: DATABASE_URL=postgres://... make seed-prod
seed-prod:
	docker build -t mmadle-api backend
	docker run --rm -e DATABASE_URL="$(DATABASE_URL)" mmadle-api /app/importer --seed /app/migrations/seed.sql

# ---- Frontend ----
frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

frontend-test:
	cd frontend && npm test -- --run 2>&1 | tail -20 || npm test

frontend-lint:
	cd frontend && npm run lint && npx tsc --noEmit

frontend-build:
	cd frontend && npm run build
