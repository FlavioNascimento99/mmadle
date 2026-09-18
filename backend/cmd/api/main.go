// Command api runs the MMAdle backend: applies SQL migrations, then serves
// the REST API with the standard library's net/http.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"mmadle/backend/internal/domain"
	"mmadle/backend/internal/httpapi"
	"mmadle/backend/internal/store"

	"github.com/jackc/pgx/v5"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// sessionTTL parses SESSION_TTL (e.g. "720h") with a 30-day default.
func sessionTTL() time.Duration {
	if raw := os.Getenv("SESSION_TTL"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			return d
		}
	}
	return 30 * 24 * time.Hour
}

// runMigrations applies versioned *.sql files from dir in lexical order,
// tracking applied files in schema_migrations. Idempotent: safe on every boot.
// seed.sql is skipped here; it is applied separately by cmd/importer.
func runMigrations(ctx context.Context, databaseURL, dir string) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect for migrations: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	names := []string{}
	for _, e := range entries {
		if e.IsDir() || e.Name() == "seed.sql" || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return fmt.Errorf("no migrations found in %s", dir)
	}

	for _, name := range names {
		var applied bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := conn.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
			return err
		}
		fmt.Println("applied migration:", name)
	}
	return nil
}

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply migrations then exit")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := env("DATABASE_URL", "postgres://mmadle:mmadle_dev_password@localhost:5432/mmadle?sslmode=disable")
	port := env("PORT", "8080")
	migrationsDir := env("MIGRATIONS_DIR", "migrations")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := runMigrations(ctx, databaseURL, migrationsDir); err != nil {
		logger.Error("migrations failed", "err", err)
		os.Exit(1)
	}
	if *migrateOnly {
		return
	}

	st, err := store.NewPostgres(context.Background(), databaseURL)
	if err != nil {
		logger.Error("database connection failed", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	tz := env("GAME_TIMEZONE", "UTC")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		logger.Warn("invalid GAME_TIMEZONE, using UTC", "tz", tz)
		loc = time.UTC
	}

	srv := httpapi.New(st, st, domain.HashSelector{}, httpapi.SystemClock{Location: loc}, logger)
	srv.SessionTTL = sessionTTL()
	srv.SessionSecure = env("SESSION_COOKIE_SECURE", "true") != "false"
	addr := ":" + port
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(env("ALLOWED_ORIGINS", "http://localhost:3000")),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
	}
	logger.Info("mmadle backend listening", "addr", addr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}
