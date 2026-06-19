package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/renanferr/purchase-api/internal/domain"
	"github.com/renanferr/purchase-api/internal/ports"
	"github.com/shopspring/decimal"
)

type CurrencyConversion struct {
	Currency string
	RateDate string
	Rate     string
	Amount   string
}

type PurchaseService struct {
	purchaseRepo ports.PurchaseRepository
	rateRepo     ports.ExchangeRateRepository
	treasury     ports.TreasuryRateProvider
	logger       ports.Logger
}

func NewPurchaseService(purchaseRepo ports.PurchaseRepository, rateRepo ports.ExchangeRateRepository, treasury ports.TreasuryRateProvider) *PurchaseService {
	return &PurchaseService{purchaseRepo: purchaseRepo, rateRepo: rateRepo, treasury: treasury, logger: nil}
}

func (s *PurchaseService) WithLogger(logger ports.Logger) *PurchaseService {
	s.logger = logger
	return s
}

func (s *PurchaseService) CreatePurchase(ctx context.Context, description, transactionDate, amountUsd string) (domain.Purchase, error) {
	if len(description) == 0 || len(description) > 50 {
		return domain.Purchase{}, errors.New("description must be 1 to 50 characters")
	}

	date, err := parseDate(transactionDate)
	if err != nil {
		return domain.Purchase{}, err
	}

	amount, err := domain.ParseMoney(amountUsd)
	if err != nil {
		return domain.Purchase{}, err
	}

	purchase, err := domain.NewPurchase(description, date, amount.Cents)
	if err != nil {
		return domain.Purchase{}, err
	}

	if err := s.purchaseRepo.Create(ctx, purchase); err != nil {
		return domain.Purchase{}, err
	}

	return purchase, nil
}

func (s *PurchaseService) GetPurchase(ctx context.Context, id string) (domain.Purchase, bool, error) {
	uuidID, err := uuid.Parse(id)
	if err != nil {
		return domain.Purchase{}, false, errors.New("invalid purchase id")
	}
	return s.purchaseRepo.GetByID(ctx, uuidID)
}

func (s *PurchaseService) GetPurchaseWithConversion(ctx context.Context, id, targetCurrency string) (domain.Purchase, *CurrencyConversion, bool, error) {
	purchase, found, err := s.GetPurchase(ctx, id)
	if err != nil || !found {
		return domain.Purchase{}, nil, false, err
	}

	// No currency conversion requested
	if strings.TrimSpace(targetCurrency) == "" {
		return purchase, nil, true, nil
	}

	normalized := strings.TrimSpace(targetCurrency)

	// Handle USD pass-through (no conversion needed)
	// USD is always the base currency with rate = 1.0
	if s.isUSD(normalized) {
		return purchase, &CurrencyConversion{
			Currency: "USD",
			RateDate: purchase.TransactionDate.Format("2006-01-02"),
			Rate:     "1.000000",
			Amount:   purchase.AmountUsd(),
		}, true, nil
	}

	// Get exchange rate (from cache or treasury provider)
	conversion, err := s.getConversionRate(ctx, purchase, normalized)
	if err != nil {
		return domain.Purchase{}, nil, false, err
	}

	return purchase, conversion, true, nil
}

// isUSD checks if the currency string is USD (ISO 4217 code)
// Only accepts 3-letter uppercase ISO currency codes
func (s *PurchaseService) isUSD(currency string) bool {
	return strings.EqualFold(currency, "USD")
}

// getConversionRate retrieves the exchange rate from cache or treasury provider
// Assumes currency is already normalized to uppercase ISO code
func (s *PurchaseService) getConversionRate(ctx context.Context, purchase domain.Purchase, currency string) (*CurrencyConversion, error) {
	if s.treasury == nil {
		return nil, errors.New("exchange rate provider is not configured")
	}

	// Try to get rate from cache first
	if s.rateRepo != nil {
		rate, found, err := s.rateRepo.GetLatestBeforeDate(ctx, currency, purchase.TransactionDate)
		if err != nil {
			return nil, err
		}

		if found && withinSixMonths(rate.RateDate, purchase.TransactionDate) {
			return s.buildConversion(purchase, rate.Currency, rate.Rate, rate.RateDate), nil
		}
	}

	// Fall back to treasury provider
	return s.getRateFromTreasuryAndCache(ctx, purchase, currency)
}

// getRateFromTreasuryAndCache fetches rate from treasury provider and caches it
func (s *PurchaseService) getRateFromTreasuryAndCache(ctx context.Context, purchase domain.Purchase, currency string) (*CurrencyConversion, error) {
	// Log the Treasury API call for cache validation during testing
	if s.logger != nil {
		s.logger.LogTreasuryAPIQuery(ctx, currency, purchase.TransactionDate.Format("2006-01-02"), purchase.ID.String())
	}

	rateValue, currencyLabel, rateDate, err := s.treasury.LatestRateBeforeDate(ctx, currency, purchase.TransactionDate)
	if err != nil {
		return nil, err
	}

	if !withinSixMonths(rateDate, purchase.TransactionDate) {
		return nil, &domain.RateNotFoundError{
			Currency: currency,
			Date:     purchase.TransactionDate,
		}
	}

	// Try to cache the rate (ignore duplicate key errors)
	if s.rateRepo != nil {
		storedRate := domain.ExchangeRate{
			Currency: strings.ToUpper(currencyLabel),
			RateDate: rateDate,
			Rate:     rateValue,
		}
		if err := s.rateRepo.Create(ctx, storedRate); err != nil {
			// Ignore duplicate key errors - the rate already exists in cache
			if !strings.Contains(err.Error(), "duplicate key") {
				return nil, err
			}
		}
	}

	return s.buildConversion(purchase, currencyLabel, rateValue, rateDate), nil
}

// buildConversion creates a CurrencyConversion struct with calculated amount
// Calculation: USD amount × exchange rate = target currency amount
// Example: 100 USD × 0.87 (1 USD = 0.87 EUR) = 87 EUR
func (s *PurchaseService) buildConversion(purchase domain.Purchase, currency string, rate decimal.Decimal, rateDate time.Time) *CurrencyConversion {
	// Convert cents to dollars, then multiply by the USD-to-target-currency rate
	converted := decimal.NewFromInt(purchase.AmountUsdCents).
		Div(decimal.NewFromInt(100)).
		Mul(rate).
		Round(2)

	return &CurrencyConversion{
		Currency: currency,
		RateDate: rateDate.Format("2006-01-02"),
		Rate:     rate.StringFixed(6),
		Amount:   converted.StringFixed(2),
	}
}

func withinSixMonths(rateDate, purchaseDate time.Time) bool {
	if rateDate.IsZero() {
		return false
	}
	limit := purchaseDate.AddDate(0, -6, 0)
	return !rateDate.Before(limit) && !rateDate.After(purchaseDate)
}

func parseDate(value string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errors.New("transactionDate must be a valid ISO 8601 date")
}
