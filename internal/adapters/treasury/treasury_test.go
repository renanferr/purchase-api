package treasury

import (
	"context"
	"testing"
	"time"

	"github.com/example/purchase-api/internal/ports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// SampleProviderContractTestSuite tests the Treasury Rate Provider contract
type SampleProviderContractTestSuite struct {
	suite.Suite
	provider ports.TreasuryRateProvider
}

// SetupSuite initializes test fixtures
func (suite *SampleProviderContractTestSuite) SetupSuite() {
	// Initialize sample provider for contract testing
	suite.provider = NewSampleTreasuryRateProvider(
		decimal.NewFromFloat(0.92),
		"EUR",
		time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		nil,
	)
	suite.NotNil(suite.provider)
}

// TestSampleProvider_ReturnsDeterministicRates validates deterministic rate responses
func (suite *SampleProviderContractTestSuite) TestSampleProvider_ReturnsDeterministicRates() {
	ctx := context.Background()

	// The sample provider is configured for EUR with rate 0.92
	rate, currency, rateDate, err := suite.provider.LatestRateBeforeDate(ctx, "EUR", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))

	suite.NoError(err)
	suite.Equal("EUR", currency)
	suite.Equal(decimal.NewFromFloat(0.92), rate)
	suite.Equal(time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC), rateDate)
}

// TestSampleProvider_RatesAreDeterministic validates same input returns same rate
func (suite *SampleProviderContractTestSuite) TestSampleProvider_RatesAreDeterministic() {
	ctx := context.Background()

	// Call same request twice
	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	rate1, currency1, date1, err1 := suite.provider.LatestRateBeforeDate(ctx, "EUR", date)
	rate2, currency2, date2, err2 := suite.provider.LatestRateBeforeDate(ctx, "EUR", date)

	// Both calls should succeed
	suite.NoError(err1)
	suite.NoError(err2)

	// Rates should be identical
	suite.Equal(rate1, rate2)
	suite.Equal(currency1, currency2)
	suite.Equal(date1, date2)
}

// TestSampleProvider_USDPassThrough validates USD currency returns 1.0 rate
func (suite *SampleProviderContractTestSuite) TestSampleProvider_USDPassThrough() {
	ctx := context.Background()

	rate, currency, _, err := suite.provider.LatestRateBeforeDate(ctx, "USD", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))

	suite.NoError(err)
	suite.Equal("USD", currency)
	suite.Equal(decimal.NewFromFloat(1.0), rate)
}

// TestSampleProvider_RateStructure validates rate object structure
func (suite *SampleProviderContractTestSuite) TestSampleProvider_RateStructure() {
	ctx := context.Background()

	rate, currency, rateDate, err := suite.provider.LatestRateBeforeDate(ctx, "EUR", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))

	suite.NoError(err)

	// Validate structure
	suite.NotEmpty(currency)
	suite.True(rate.IsPositive())
	suite.False(rateDate.IsZero())
}

// TestSampleProvider_CurrencyCodeValidation validates currency format handling
func (suite *SampleProviderContractTestSuite) TestSampleProvider_CurrencyCodeValidation() {
	ctx := context.Background()

	testCases := []struct {
		name       string
		currency   string
		shouldWork bool
	}{
		{
			name:       "sample_provider_configured_currency",
			currency:   "EUR",
			shouldWork: true,
		},
		{
			name:       "different_currency_not_configured",
			currency:   "GBP",
			shouldWork: false, // Sample provider returns error for unknown currencies
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			rate, currency, _, err := suite.provider.LatestRateBeforeDate(ctx, tc.currency, time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC))

			if tc.shouldWork {
				suite.NoError(err)
				suite.NotEmpty(currency)
				suite.True(rate.IsPositive())
			} else {
				// Sample provider returns error for unknown currencies
				suite.Error(err)
			}
		})
	}
}

// TestSampleProvider_DateHandling validates date parameter handling
func (suite *SampleProviderContractTestSuite) TestSampleProvider_DateHandling() {
	ctx := context.Background()

	testCases := []struct {
		name  string
		date  time.Time
		found bool
	}{
		{
			name:  "current_date",
			date:  time.Now().UTC(),
			found: true,
		},
		{
			name:  "past_date",
			date:  time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			found: true,
		},
		{
			name:  "future_date_30_days",
			date:  time.Now().UTC().AddDate(0, 0, 30),
			found: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			rate, currency, rateDate, err := suite.provider.LatestRateBeforeDate(ctx, "EUR", tc.date)

			if tc.found {
				suite.NoError(err)
				suite.NotEmpty(currency)
				suite.True(rate.IsPositive())
				suite.False(rateDate.IsZero())
			} else {
				suite.Error(err)
			}
		})
	}
}

// TestSampleProviderContractTestSuite entry point
func TestSampleProviderContractTestSuite(t *testing.T) {
	suite.Run(t, new(SampleProviderContractTestSuite))
}
