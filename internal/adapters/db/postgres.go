package db

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/renanferr/purchase-api/internal/db"
	"github.com/renanferr/purchase-api/internal/domain"
	"github.com/renanferr/purchase-api/internal/ports"
	"github.com/shopspring/decimal"
)

type PurchaseRepository struct {
	queries *db.Queries
}

type ExchangeRateRepositoryAdapter struct {
	queries *db.Queries
}

func NewPurchaseRepository(pool *pgxpool.Pool) ports.PurchaseRepository {
	return &PurchaseRepository{
		queries: db.New(pool),
	}
}

func NewExchangeRateRepository(pool *pgxpool.Pool) ports.ExchangeRateRepository {
	return &ExchangeRateRepositoryAdapter{
		queries: db.New(pool),
	}
}

// PurchaseRepository methods

func (r *PurchaseRepository) Create(ctx context.Context, purchase domain.Purchase) error {
	_, err := r.queries.CreatePurchase(ctx, db.CreatePurchaseParams{
		ID:              pgtype.UUID{Bytes: purchase.ID, Valid: true},
		Description:     purchase.Description,
		TransactionDate: pgtype.Date{Time: purchase.TransactionDate, Valid: true},
		AmountUsdCents:  purchase.AmountUsdCents,
	})
	if err != nil {
		// Check for unique constraint violation (PostgreSQL error code 23505)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &domain.UniqueConstraintError{Cause: err}
		}
		// Wrap other errors with context, preserving the sentinel
		return &domain.DatabaseErrorDetails{
			Operation: "create",
			Table:     "purchases",
			Cause:     err,
		}
	}
	return nil
}

func (r *PurchaseRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
	dbPurchase, err := r.queries.GetPurchaseByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Purchase{}, false, nil
		}
		return domain.Purchase{}, false, err
	}

	return domain.Purchase{
		ID:              dbPurchase.ID.Bytes,
		Description:     dbPurchase.Description,
		TransactionDate: dbPurchase.TransactionDate.Time,
		AmountUsdCents:  dbPurchase.AmountUsdCents,
		CreatedAt:       dbPurchase.CreatedAt.Time,
		UpdatedAt:       dbPurchase.UpdatedAt.Time,
	}, true, nil
}

// ExchangeRateRepository methods

func (r *ExchangeRateRepositoryAdapter) Create(ctx context.Context, rate domain.ExchangeRate) error {
	_, err := r.queries.CreateExchangeRate(ctx, db.CreateExchangeRateParams{
		Currency: rate.Currency,
		RateDate: pgtype.Date{Time: rate.RateDate, Valid: true},
		Rate:     pgtype.Numeric{Int: rate.Rate.Coefficient(), Exp: int32(rate.Rate.Exponent()), Valid: true},
	})
	if err != nil {
		// Check for unique constraint violation (PostgreSQL error code 23505)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return &domain.UniqueConstraintError{Cause: err}
		}
		// Wrap other errors with context, preserving the sentinel
		return &domain.DatabaseErrorDetails{
			Operation: "create",
			Table:     "exchange_rates",
			Cause:     err,
		}
	}
	return nil
}

func (r *ExchangeRateRepositoryAdapter) GetLatestBeforeDate(ctx context.Context, currency string, before time.Time) (domain.ExchangeRate, bool, error) {
	rate, err := r.queries.GetRateForDate(ctx, db.GetRateForDateParams{
		Currency: currency,
		RateDate: pgtype.Date{Time: before, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ExchangeRate{}, false, nil
		}
		return domain.ExchangeRate{}, false, err
	}

	// Convert pgtype.Numeric to decimal.Decimal
	decimalRate := decimal.NewFromBigInt(rate.Rate.Int, rate.Rate.Exp)

	return domain.ExchangeRate{
		Currency:  rate.Currency,
		RateDate:  rate.RateDate.Time,
		Rate:      decimalRate,
		CreatedAt: time.Time{}, // GetRateForDate doesn't return created_at
	}, true, nil
}
