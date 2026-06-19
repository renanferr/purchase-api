package treasury

import (
	"context"
	"strings"
	"time"

	"github.com/renanferr/purchase-api/internal/domain"
	"github.com/renanferr/purchase-api/internal/ports"
	"github.com/shopspring/decimal"
)

type SampleTreasuryRateProvider struct {
	Rate       decimal.Decimal
	Currency   string
	RecordDate time.Time
	Err        error
}

func NewSampleTreasuryRateProvider(rate decimal.Decimal, currency string, recordDate time.Time, err error) ports.TreasuryRateProvider {
	return &SampleTreasuryRateProvider{Rate: rate, Currency: currency, RecordDate: recordDate, Err: err}
}

func (s *SampleTreasuryRateProvider) LatestRateBeforeDate(ctx context.Context, currency string, before time.Time) (decimal.Decimal, string, time.Time, error) {
	if s.Err != nil {
		return decimal.Zero, "", time.Time{}, s.Err
	}

	// USD always passes through at 1.0 rate (case-insensitive)
	if strings.EqualFold(currency, "USD") {
		return decimal.NewFromFloat(1.0), "USD", before, nil
	}

	// For configured currency, return the mock rate (case-insensitive)
	if strings.EqualFold(currency, s.Currency) {
		return s.Rate, s.Currency, s.RecordDate, nil
	}

	// For any other currency, return a typed error (no rate available)
	return decimal.Zero, "", time.Time{}, &domain.NoRateError{
		Currency: currency,
		Date:     before,
	}
}
