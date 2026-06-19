package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/renanferr/purchase-api/internal/domain"
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

type Logger interface {
	LogCreate(ctx context.Context, purchaseID string, metadata map[string]interface{})
	LogCreateError(ctx context.Context, errorCode, message string, metadata map[string]interface{})
	LogRetrieve(ctx context.Context, purchaseID string, metadata map[string]interface{})
	LogRetrieveError(ctx context.Context, purchaseID, errorCode, message string, metadata map[string]interface{})
	LogConversion(ctx context.Context, purchaseID, currency, amount string, metadata map[string]interface{})
	LogConversionError(ctx context.Context, purchaseID, currency, errorCode, message string, metadata map[string]interface{})
	LogTreasuryAPIQuery(ctx context.Context, currency, purchaseDate, purchaseID string)
	Error(message string, attrs map[string]interface{})
}
