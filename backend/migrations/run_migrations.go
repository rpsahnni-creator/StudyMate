//go:build ignore

// Run idempotent SQL migrations against DATABASE_URL:
//
//	go run migrations/run_migrations.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS migrations_applied (
	id          BIGSERIAL PRIMARY KEY,
	filename    TEXT NOT NULL UNIQUE,
	applied_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func main() {
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, createTableSQL); err != nil {
		fmt.Fprintf(os.Stderr, "create migrations_applied: %v\n", err)
		os.Exit(1)
	}

	migrationsDir := "migrations"
	if len(os.Args) > 1 {
		migrationsDir = os.Args[1]
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "glob migrations: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(files)

	applied := 0
	skipped := 0

	for _, path := range files {
		filename := filepath.Base(path)
		var exists bool
		if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM migrations_applied WHERE filename = $1)`, filename).Scan(&exists); err != nil {
			fmt.Fprintf(os.Stderr, "check %s: %v\n", filename, err)
			os.Exit(1)
		}
		if exists {
			fmt.Printf("skip %s (already applied)\n", filename)
			skipped++
			continue
		}

		body, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", filename, err)
			os.Exit(1)
		}
		sql := strings.TrimSpace(string(body))
		if sql == "" {
			fmt.Printf("skip %s (empty file)\n", filename)
			continue
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "begin %s: %v\n", filename, err)
			os.Exit(1)
		}

		if _, err := tx.Exec(ctx, sql); err != nil {
			_ = tx.Rollback(ctx)
			fmt.Fprintf(os.Stderr, "apply %s: %v\n", filename, err)
			os.Exit(1)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO migrations_applied (filename) VALUES ($1)`, filename); err != nil {
			_ = tx.Rollback(ctx)
			fmt.Fprintf(os.Stderr, "record %s: %v\n", filename, err)
			os.Exit(1)
		}
		if err := tx.Commit(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "commit %s: %v\n", filename, err)
			os.Exit(1)
		}

		fmt.Printf("applied %s\n", filename)
		applied++
	}

	fmt.Printf("done: applied=%d skipped=%d\n", applied, skipped)
}
