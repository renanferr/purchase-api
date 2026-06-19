// @title Purchase API
// @version 1.0.0
// @description RESTful API for managing purchase transactions with multi-currency support
// @termsOfService http://swagger.io/terms/

// @host localhost:8080
// @basePath /
// @schemes http https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/example/purchase-api/internal/adapters/db"
	"github.com/example/purchase-api/internal/api"
	"github.com/example/purchase-api/internal/app"
	"github.com/example/purchase-api/internal/config"
	"github.com/example/purchase-api/internal/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build application config using options pattern
	// Can override defaults with options: config.WithAPIPort("9000"), config.WithRealProvider(), etc.
	cfg, err := config.BuildConfig(ctx)
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Error("failed to build config", "error", err)
		} else {
			fmt.Fprintf(os.Stderr, "failed to build config: %v\n", err)
		}
		os.Exit(1)
	}

	// Initialize database connection
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		cfg.Logger.Error("unable to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Verify database connectivity
	if err := pool.Ping(ctx); err != nil {
		cfg.Logger.Error("unable to ping database", "error", err)
		os.Exit(1)
	}

	// Run database migrations
	migrationRunner := migrations.NewRunner(pool, "db/migrations")
	if err := migrationRunner.Up(); err != nil {
		cfg.Logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize repositories
	purchaseRepo := db.NewPurchaseRepository(pool)
	rateRepo := db.NewExchangeRateRepository(pool)

	// Create service with all dependencies (wired via config)
	service := app.NewPurchaseService(purchaseRepo, rateRepo, cfg.TreasuryProvider)

	// Set up HTTP router directly (already a chi router from NewRouter)
	handler := api.NewRouter(service, api.WithDatabasePool(pool))

	// Create and start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.APIPort),
		Handler:      handler,
		ReadTimeout:  cfg.HTTPTimeout,
		WriteTimeout: cfg.HTTPTimeout,
	}

	cfg.Logger.Info("starting purchase-api", "port", cfg.APIPort, "provider", os.Getenv("TREASURY_PROVIDER"))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		cfg.Logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
