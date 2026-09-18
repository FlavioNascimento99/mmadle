// Command importer applies the seed dataset (data ingestion boundary).
//
// The application runtime never scrapes or imports data; all ingestion goes
// through this process so it can later be replaced by a proper pipeline
// consuming a reliable MMA/UFC data source.
//
// Usage:
//
//	go run ./cmd/importer --seed ./migrations/seed.sql
//	go run ./cmd/importer --seed ./migrations/seed.sql --file ./data/custom.sql
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	seed := flag.String("seed", "migrations/seed.sql", "path to seed SQL file")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://mmadle:mmadle_dev_password@localhost:5432/mmadle?sslmode=disable"
	}
	sql, err := os.ReadFile(*seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read seed file:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, string(sql)); err != nil {
		fmt.Fprintln(os.Stderr, "apply seed:", err)
		os.Exit(1)
	}
	fmt.Println("seed applied from", *seed)
}
