package config

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/renanferr/purchase-api/internal/ports"
)

// AppConfig holds all application configuration
type AppConfig struct {
	Logger           *slog.Logger
	DatabaseURL      string
	APIPort          string
	HTTPTimeout      time.Duration
	TreasuryProvider ports.TreasuryRateProvider
	HTTPClient       *http.Client
}

// Option is a functional option for configuring AppConfig
type Option func(*AppConfig)

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *AppConfig {
	return &AppConfig{
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/purchase_api?sslmode=disable",
		APIPort:     "8080",
		HTTPTimeout: 15 * time.Second,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// WithLogger sets the logger for the application
func WithLogger(logger *slog.Logger) Option {
	return func(c *AppConfig) {
		c.Logger = logger
	}
}

// WithDatabaseURL sets the PostgreSQL connection string
func WithDatabaseURL(url string) Option {
	return func(c *AppConfig) {
		c.DatabaseURL = url
	}
}

// WithAPIPort sets the HTTP server port
func WithAPIPort(port string) Option {
	return func(c *AppConfig) {
		c.APIPort = port
	}
}

// WithHTTPTimeout sets the HTTP server timeout
func WithHTTPTimeout(timeout time.Duration) Option {
	return func(c *AppConfig) {
		c.HTTPTimeout = timeout
	}
}

// WithTreasuryProvider sets the exchange rate provider
func WithTreasuryProvider(provider ports.TreasuryRateProvider) Option {
	return func(c *AppConfig) {
		c.TreasuryProvider = provider
	}
}

// WithHTTPClient sets the HTTP client for provider requests
func WithHTTPClient(client *http.Client) Option {
	return func(c *AppConfig) {
		c.HTTPClient = client
	}
}

// BuildConfig creates an AppConfig by applying options to defaults
// and loading environment variables where options are not provided
func BuildConfig(ctx context.Context, opts ...Option) (*AppConfig, error) {
	config := DefaultConfig()

	// Load .env file if it exists (doesn't fail if missing)
	_ = godotenv.Load()

	// Apply provided options
	for _, opt := range opts {
		opt(config)
	}

	// Load from environment variables if not set by options
	if config.Logger == nil {
		config.Logger = newDefaultLogger()
	}

	if config.DatabaseURL == DefaultConfig().DatabaseURL {
		if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
			config.DatabaseURL = dbURL
		}
	}

	if config.APIPort == DefaultConfig().APIPort {
		if port := os.Getenv("API_PORT"); port != "" {
			config.APIPort = port
		}
	}

	if config.TreasuryProvider == nil {
		provider, err := newTreasuryProvider(ctx, config.HTTPClient)
		if err != nil {
			return nil, err
		}
		config.TreasuryProvider = provider
	}

	return config, nil
}

func newDefaultLogger() *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: getLogLevel(),
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

func getLogLevel() slog.Level {
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
