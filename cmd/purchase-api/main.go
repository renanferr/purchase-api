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
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/renanferr/purchase-api/internal/adapters/db"
	"github.com/renanferr/purchase-api/internal/api"
	"github.com/renanferr/purchase-api/internal/app"
	"github.com/renanferr/purchase-api/internal/config"
	"github.com/renanferr/purchase-api/internal/migrations"
)

// Startup timeouts and durations
const (
	startupTimeout   = 30 * time.Second
	migrationTimeout = 30 * time.Second
	shutdownTimeout  = 30 * time.Second
)

func main() {
	// Create startup context with adequate timeout for initialization
	startupCtx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	// Build application config using options pattern
	// Can override defaults with options: config.WithAPIPort("9000"), config.WithRealProvider(), etc.
	cfg, err := config.BuildConfig(startupCtx)
	if err != nil {
		if cfg.Logger != nil {
			cfg.Logger.Error("failed to build config", "error", err)
		} else {
			fmt.Fprintf(os.Stderr, "failed to build config: %v\n", err)
		}
		os.Exit(1)
	}

	// Initialize database connection
	pool, err := pgxpool.New(startupCtx, cfg.DatabaseURL)
	if err != nil {
		cfg.Logger.Error("unable to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Verify database connectivity
	if err := pool.Ping(startupCtx); err != nil {
		cfg.Logger.Error("unable to ping database", "error", err)
		os.Exit(1)
	}

	// Run database migrations
	// Note: migrations should complete within migrationTimeout threshold
	migrationRunner := migrations.NewRunner(pool, "db/migrations")
	if err := migrationRunner.Up(); err != nil {
		cfg.Logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Initialize repositories
	purchaseRepo := db.NewPurchaseRepository(pool)
	rateRepo := db.NewExchangeRateRepository(pool)

	// Create service with all dependencies (wired via config)
	logger := api.NewLogger()
	service := app.NewPurchaseService(purchaseRepo, rateRepo, cfg.TreasuryProvider).WithLogger(logger)

	// Set up HTTP router directly (already a chi router from NewRouter)
	handler := api.NewRouter(service, api.WithDatabasePool(pool))

	// Create and start HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.APIPort),
		Handler:      handler,
		ReadTimeout:  cfg.HTTPTimeout,
		WriteTimeout: cfg.HTTPTimeout,
	}

	// Channel to receive shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine to handle shutdown signals
	go func() {
		cfg.Logger.Info("starting purchase-api", "port", cfg.APIPort, "provider", os.Getenv("TREASURY_PROVIDER"))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.Logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	cfg.Logger.Info("shutdown signal received", "signal", sig.String())

	// Gracefully shutdown the server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		cfg.Logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	// Clean up database connection
	pool.Close()
	cfg.Logger.Info("server shutdown complete")
}
