package ports

import (
	"context"
	"time"

	"github.com/example/purchase-api/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PurchaseRepository interface {
	Create(ctx context.Context, purchase domain.Purchase) error
	GetByID(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error)
}

type ExchangeRateRepository interface {
	Create(ctx context.Context, rate domain.ExchangeRate) error
	GetLatestBeforeDate(ctx context.Context, currency string, before time.Time) (domain.ExchangeRate, bool, error)
}

type TreasuryRateProvider interface {
	LatestRateBeforeDate(ctx context.Context, currency string, before time.Time) (decimal.Decimal, string, time.Time, error)
}
