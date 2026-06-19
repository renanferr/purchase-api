package config

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/example/purchase-api/internal/adapters/treasury"
	"github.com/example/purchase-api/internal/ports"
	"github.com/shopspring/decimal"
)

// newTreasuryProvider creates a provider based on TREASURY_PROVIDER env var
// Default: "real" (live Treasury API)
// Options: "real" (production) | "mock" (testing only)
func newTreasuryProvider(ctx context.Context, httpClient *http.Client) (ports.TreasuryRateProvider, error) {
	providerType := strings.TrimSpace(os.Getenv("TREASURY_PROVIDER"))
	if providerType == "" {
		providerType = "real"
	}

	switch strings.ToLower(providerType) {
	case "mock":
		// Mock provider for testing/development only
		// Returns a fixed rate of 0.92 EUR/USD on a fixed date
		rate, _ := decimal.NewFromString("0.92")
		return treasury.NewSampleTreasuryRateProvider(rate, "EUR", time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC), nil), nil
	case "real":
		fallthrough
	default:
		// Real Treasury API provider (default, production-ready)
		if httpClient == nil {
			httpClient = &http.Client{Timeout: 10 * time.Second}
		}
		return treasury.NewExchangeRateProvider(httpClient), nil
	}
}

// ProviderOption is a functional option for configuring TreasuryRateProvider
type ProviderOption func(*providerConfig)

type providerConfig struct {
	providerType string
	httpClient   *http.Client
}

// WithRealProvider configures the application to use the real Treasury API
func WithRealProvider() Option {
	return func(c *AppConfig) {
		provider := treasury.NewExchangeRateProvider(c.HTTPClient)
		c.TreasuryProvider = provider
	}
}

// WithSampleProvider configures the application to use the mock provider for testing
func WithSampleProvider(rate decimal.Decimal, currency string, recordDate time.Time) Option {
	return func(c *AppConfig) {
		provider := treasury.NewSampleTreasuryRateProvider(rate, currency, recordDate, nil)
		c.TreasuryProvider = provider
	}
}

// WithMockProvider is an alias for WithSampleProvider, preferred for testing scenarios
func WithMockProvider(rate decimal.Decimal, currency string, recordDate time.Time) Option {
	return WithSampleProvider(rate, currency, recordDate)
}

// WithSampleProviderDefaults configures the application to use the mock provider with default test data
func WithSampleProviderDefaults() Option {
	rate, _ := decimal.NewFromString("0.92")
	return WithSampleProvider(rate, "EUR", time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC))
}
