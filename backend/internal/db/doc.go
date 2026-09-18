// Package db is the sqlc code-generation target for queries/*.sql
// (see sqlc.yaml). Run `sqlc generate` from backend/ to (re)generate
// type-safe query code here.
//
// The repository in internal/store currently executes the same SQL in
// queries/fighters.sql directly through pgx; generated code in this package
// can progressively replace those hand-written calls without changing the
// store's public interface.
package db
