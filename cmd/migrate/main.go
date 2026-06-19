package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"context"

	"github.com/example/purchase-api/internal/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var (
		dbURL     = flag.String("db", os.Getenv("DATABASE_URL"), "Database connection URL")
		direction = flag.String("dir", "up", "Migration direction: up or down")
	)
	flag.Parse()

	if *dbURL == "" {
		*dbURL = "postgres://postgres:postgres@localhost:5432/purchase_api?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to database
	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		os.Exit(1)
	}

	// Determine migrations path - use absolute path
	var migrationsPath string
	if wd, err := os.Getwd(); err == nil {
		// Try db/migrations relative to current directory first
		testPath := filepath.Join(wd, "db", "migrations")
		if _, err := os.Stat(testPath); err == nil {
			migrationsPath = testPath
		} else {
			// Try ../db/migrations if not found
			testPath = filepath.Join(wd, "..", "db", "migrations")
			if _, err := os.Stat(testPath); err == nil {
				migrationsPath = testPath
			} else {
				// Default to db/migrations
				migrationsPath = filepath.Join(wd, "db", "migrations")
			}
		}
	}

	// Create and run migration runner
	runner := migrations.NewRunner(pool, migrationsPath)

	switch *direction {
	case "up":
		if err := runner.Up(); err != nil {
			fmt.Fprintf(os.Stderr, "Migration up failed: %v\n", err)
			os.Exit(1)
		}
	case "down":
		if err := runner.Down(); err != nil {
			fmt.Fprintf(os.Stderr, "Migration down failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Invalid direction: %s (use 'up' or 'down')\n", *direction)
		os.Exit(1)
	}
}
