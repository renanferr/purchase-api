package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/purchase-api/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// fakePurchaseRepository provides mock implementation for PurchaseRepository
type fakePurchaseRepository struct {
	getByIDFunc func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error)
	lastCreated domain.Purchase
}

func (r *fakePurchaseRepository) Create(ctx context.Context, purchase domain.Purchase) error {
	r.lastCreated = purchase
	return nil
}

func (r *fakePurchaseRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
	if r.getByIDFunc != nil {
		return r.getByIDFunc(ctx, id)
	}
	return domain.Purchase{}, false, errors.New("not implemented")
}

// fakeExchangeRateRepository provides mock implementation for ExchangeRateRepository
type fakeExchangeRateRepository struct {
	getLatestFunc func(ctx context.Context, currency string, before time.Time) (domain.ExchangeRate, bool, error)
	lastCreated   domain.ExchangeRate
}

func (r *fakeExchangeRateRepository) Create(ctx context.Context, rate domain.ExchangeRate) error {
	r.lastCreated = rate
	return nil
}

func (r *fakeExchangeRateRepository) GetLatestBeforeDate(ctx context.Context, currency string, before time.Time) (domain.ExchangeRate, bool, error) {
	if r.getLatestFunc != nil {
		return r.getLatestFunc(ctx, currency, before)
	}
	return domain.ExchangeRate{}, false, nil
}

// fakeExchangeRateProvider provides mock implementation for TreasuryRateProvider
type fakeExchangeRateProvider struct {
	rate      decimal.Decimal
	currency  string
	rateDate  time.Time
	returnErr error
}

func (f *fakeExchangeRateProvider) LatestRateBeforeDate(ctx context.Context, currency string, before time.Time) (decimal.Decimal, string, time.Time, error) {
	if f.returnErr != nil {
		return decimal.Zero, "", time.Time{}, f.returnErr
	}
	rateDate := f.rateDate
	if rateDate.IsZero() {
		rateDate = before
	}
	return f.rate, f.currency, rateDate, nil
}

// CreatePurchaseTestSuite defines the test suite for CreatePurchase
type CreatePurchaseTestSuite struct {
	suite.Suite
	purchaseRepo *fakePurchaseRepository
	rateRepo     *fakeExchangeRateRepository
	service      *PurchaseService
}

func (suite *CreatePurchaseTestSuite) SetupTest() {
	suite.purchaseRepo = &fakePurchaseRepository{}
	suite.rateRepo = &fakeExchangeRateRepository{}
	suite.service = NewPurchaseService(suite.purchaseRepo, suite.rateRepo, nil)
}

// TestCreatePurchase_TableDriven validates CreatePurchase with various input scenarios
func (suite *CreatePurchaseTestSuite) TestCreatePurchase_TableDriven() {
	testCases := []struct {
		name            string
		description     string
		transactionDate string
		amountUsd       string
		expectErr       bool
	}{
		{
			name:            "success",
			description:     "Coffee",
			transactionDate: "2026-06-17",
			amountUsd:       "12.34",
			expectErr:       false,
		},
		{
			name:            "empty_description_error",
			description:     "",
			transactionDate: "2026-06-17",
			amountUsd:       "12.34",
			expectErr:       true,
		},
		{
			name:            "description_too_long_error",
			description:     "This is a very long description that exceeds the maximum allowed length of 50 characters",
			transactionDate: "2026-06-17",
			amountUsd:       "12.34",
			expectErr:       true,
		},
		{
			name:            "invalid_date_format_error",
			description:     "Coffee",
			transactionDate: "2026-13-01",
			amountUsd:       "12.34",
			expectErr:       true,
		},
		{
			name:            "negative_amount_error",
			description:     "Coffee",
			transactionDate: "2026-06-17",
			amountUsd:       "-12.34",
			expectErr:       true,
		},
		{
			name:            "zero_amount_error",
			description:     "Coffee",
			transactionDate: "2026-06-17",
			amountUsd:       "0.00",
			expectErr:       true,
		},
		{
			name:            "invalid_amount_format_error",
			description:     "Coffee",
			transactionDate: "2026-06-17",
			amountUsd:       "not-a-number",
			expectErr:       true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			purchase, err := suite.service.CreatePurchase(context.Background(), tc.description, tc.transactionDate, tc.amountUsd)

			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.NotEmpty(purchase.ID)
				suite.Equal(tc.description, purchase.Description)
			}
		})
	}
}

// GetPurchaseTestSuite defines the test suite for GetPurchase operations
type GetPurchaseTestSuite struct {
	suite.Suite
	purchaseRepo *fakePurchaseRepository
	service      *PurchaseService
}

func (suite *GetPurchaseTestSuite) SetupTest() {
	suite.purchaseRepo = &fakePurchaseRepository{}
	suite.service = NewPurchaseService(suite.purchaseRepo, nil, nil)
}

// TestGetPurchase_TableDriven validates GetPurchase with various input scenarios
func (suite *GetPurchaseTestSuite) TestGetPurchase_TableDriven() {
	purchaseID := uuid.New()
	now := time.Now().UTC()

	testCases := []struct {
		name        string
		idStr       string
		setup       func()
		expectFound bool
		expectErr   bool
	}{
		{
			name:  "valid_uuid_found",
			idStr: purchaseID.String(),
			setup: func() {
				suite.purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
					return domain.Purchase{
						ID:              purchaseID,
						Description:     "Test",
						TransactionDate: now.AddDate(0, 0, -1),
						AmountUsdCents:  5000,
						CreatedAt:       now,
						UpdatedAt:       now,
					}, true, nil
				}
			},
			expectFound: true,
			expectErr:   false,
		},
		{
			name:  "valid_uuid_not_found",
			idStr: uuid.New().String(),
			setup: func() {
				suite.purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
					return domain.Purchase{}, false, nil
				}
			},
			expectFound: false,
			expectErr:   false,
		},
		{
			name:        "invalid_uuid",
			idStr:       "not-a-uuid",
			setup:       func() {},
			expectFound: false,
			expectErr:   true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setup()
			purchase, found, err := suite.service.GetPurchase(context.Background(), tc.idStr)

			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(tc.expectFound, found)
				if found {
					suite.NotEmpty(purchase.ID)
					suite.NotZero(purchase.CreatedAt)
					suite.NotZero(purchase.UpdatedAt)
				}
			}
		})
	}
}

// GetPurchaseWithConversionTestSuite defines the test suite for GetPurchaseWithConversion operations
type GetPurchaseWithConversionTestSuite struct {
	suite.Suite
	purchaseRepo *fakePurchaseRepository
	rateRepo     *fakeExchangeRateRepository
	service      *PurchaseService
}

func (suite *GetPurchaseWithConversionTestSuite) SetupTest() {
	suite.purchaseRepo = &fakePurchaseRepository{}
	suite.rateRepo = &fakeExchangeRateRepository{}
	suite.service = NewPurchaseService(suite.purchaseRepo, suite.rateRepo, nil)
}

// TestGetPurchaseWithConversion_TableDriven validates GetPurchaseWithConversion with various scenarios
func (suite *GetPurchaseWithConversionTestSuite) TestGetPurchaseWithConversion_TableDriven() {
	purchaseID := uuid.New()
	now := time.Now().UTC()
	txDate := now.AddDate(0, 0, -10)

	// Define purchase date explicitly to ensure 6-month window calculations are correct
	purchaseDateOutsideWindow := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	rateDateOutsideWindow := time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC)

	testCases := []struct {
		name             string
		purchaseExists   bool
		purchaseDate     time.Time
		currency         string
		provider         *fakeExchangeRateProvider
		expectErr        bool
		expectConversion bool
		expectedCurrency string
		expectedAmount   string
	}{
		{
			name:             "no_currency_requested",
			purchaseExists:   true,
			purchaseDate:     txDate,
			currency:         "",
			provider:         nil,
			expectErr:        false,
			expectConversion: false,
		},
		{
			name:             "usd_currency_requested",
			purchaseExists:   true,
			purchaseDate:     txDate,
			currency:         "USD",
			provider:         nil,
			expectErr:        false,
			expectConversion: true,
			expectedCurrency: "USD",
			expectedAmount:   "50.00",
		},
		{
			name:             "purchase_not_found",
			purchaseExists:   false,
			purchaseDate:     txDate,
			currency:         "EUR",
			provider:         nil,
			expectErr:        false,
			expectConversion: false,
		},
		{
			name:             "rate_found_in_cache",
			purchaseExists:   true,
			purchaseDate:     txDate,
			currency:         "EUR",
			provider:         &fakeExchangeRateProvider{rate: decimal.NewFromFloat(0.85), currency: "EUR", rateDate: txDate.AddDate(0, 0, -5)},
			expectErr:        false,
			expectConversion: true,
			expectedCurrency: "EUR",
			expectedAmount:   "42.50",
		},
		{
			name:             "rate_outside_6_month_window",
			purchaseExists:   true,
			purchaseDate:     purchaseDateOutsideWindow,
			currency:         "EUR",
			provider:         &fakeExchangeRateProvider{rate: decimal.NewFromFloat(0.85), currency: "EUR", rateDate: rateDateOutsideWindow},
			expectErr:        true,
			expectConversion: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup purchase repository with the test case's purchase date
			if tc.purchaseExists {
				suite.purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
					return domain.Purchase{
						ID:              purchaseID,
						Description:     "Test",
						TransactionDate: tc.purchaseDate,
						AmountUsdCents:  5000,
						CreatedAt:       now,
						UpdatedAt:       now,
					}, true, nil
				}
			} else {
				suite.purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
					return domain.Purchase{}, false, nil
				}
			}

			service := NewPurchaseService(suite.purchaseRepo, suite.rateRepo, tc.provider)
			_, conversion, found, err := service.GetPurchaseWithConversion(context.Background(), purchaseID.String(), tc.currency)

			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
				if tc.purchaseExists {
					suite.True(found)
					if tc.expectConversion {
						suite.NotNil(conversion)
						suite.Equal(tc.expectedCurrency, conversion.Currency)
						suite.Equal(tc.expectedAmount, conversion.Amount)
					} else {
						suite.Nil(conversion)
					}
				} else {
					suite.False(found)
				}
			}
		})
	}
}

// Run all test suites
func TestCreatePurchaseSuite(t *testing.T) {
	suite.Run(t, new(CreatePurchaseTestSuite))
}

func TestGetPurchaseSuite(t *testing.T) {
	suite.Run(t, new(GetPurchaseTestSuite))
}

func TestGetPurchaseWithConversionSuite(t *testing.T) {
	suite.Run(t, new(GetPurchaseWithConversionTestSuite))
}

func TestCreatePurchase_Success(t *testing.T) {
	purchaseRepo := &fakePurchaseRepository{}
	rateRepo := &fakeExchangeRateRepository{}
	service := NewPurchaseService(purchaseRepo, rateRepo, nil)

	purchase, err := service.CreatePurchase(context.Background(), "Coffee", "2026-06-17", "12.34")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if purchase.Description != "Coffee" {
		t.Fatalf("expected description Coffee, got %q", purchase.Description)
	}
	if purchase.AmountUsdCents != 1234 {
		t.Fatalf("expected 1234 cents, got %d", purchase.AmountUsdCents)
	}
}

func TestCreatePurchase_InvalidInput(t *testing.T) {
	purchaseRepo := &fakePurchaseRepository{}
	rateRepo := &fakeExchangeRateRepository{}
	service := NewPurchaseService(purchaseRepo, rateRepo, nil)

	_, err := service.CreatePurchase(context.Background(), "", "2026-06-17", "12.34")
	if err == nil {
		t.Fatal("expected error for empty description")
	}

	_, err = service.CreatePurchase(context.Background(), "Coffee", "invalid-date", "12.34")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}

	_, err = service.CreatePurchase(context.Background(), "Coffee", "2026-06-17", "-1.00")
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
}

// T030: GetPurchase tests
func TestGetPurchase_Found(t *testing.T) {
	purchaseID := uuid.New()
	now := time.Now().UTC()
	txDate := now.AddDate(0, 0, -1) // Yesterday

	expectedPurchase := domain.Purchase{
		ID:              purchaseID,
		Description:     "Test",
		TransactionDate: txDate,
		AmountUsdCents:  5000,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	purchaseRepo := &fakePurchaseRepository{}
	purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
		if id == purchaseID {
			return expectedPurchase, true, nil
		}
		return domain.Purchase{}, false, nil
	}

	service := NewPurchaseService(purchaseRepo, nil, nil)
	purchase, found, err := service.GetPurchase(context.Background(), purchaseID.String())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !found {
		t.Fatal("expected purchase to be found")
	}
	if purchase.ID != purchaseID {
		t.Fatalf("expected ID %s, got %s", purchaseID, purchase.ID)
	}
	if purchase.CreatedAt != now {
		t.Fatalf("expected createdAt %v, got %v", now, purchase.CreatedAt)
	}
	if purchase.UpdatedAt != now {
		t.Fatalf("expected updatedAt %v, got %v", now, purchase.UpdatedAt)
	}
}

func TestGetPurchase_NotFound(t *testing.T) {
	purchaseRepo := &fakePurchaseRepository{}
	purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
		return domain.Purchase{}, false, nil
	}

	service := NewPurchaseService(purchaseRepo, nil, nil)
	_, found, err := service.GetPurchase(context.Background(), uuid.New().String())

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found {
		t.Fatal("expected purchase not to be found")
	}
}

func TestGetPurchase_InvalidID(t *testing.T) {
	purchaseRepo := &fakePurchaseRepository{}
	service := NewPurchaseService(purchaseRepo, nil, nil)

	_, _, err := service.GetPurchase(context.Background(), "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

// T030: GetPurchaseWithConversion tests
func TestGetPurchaseWithConversion_NoConversion(t *testing.T) {
	purchaseID := uuid.New()
	now := time.Now().UTC()
	expectedPurchase := domain.Purchase{
		ID:              purchaseID,
		Description:     "Test",
		TransactionDate: now.AddDate(0, 0, -1),
		AmountUsdCents:  5000,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	purchaseRepo := &fakePurchaseRepository{}
	purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
		return expectedPurchase, true, nil
	}

	service := NewPurchaseService(purchaseRepo, nil, nil)
	purchase, conversion, found, err := service.GetPurchaseWithConversion(context.Background(), purchaseID.String(), "")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !found {
		t.Fatal("expected purchase to be found")
	}
	if conversion != nil {
		t.Fatal("expected no conversion when currency is empty")
	}
	if purchase.ID != purchaseID {
		t.Fatalf("expected ID %s, got %s", purchaseID, purchase.ID)
	}
}

func TestGetPurchaseWithConversion_USDPass(t *testing.T) {
	purchaseID := uuid.New()
	now := time.Now().UTC()
	txDate := now.AddDate(0, 0, -1)
	expectedPurchase := domain.Purchase{
		ID:              purchaseID,
		Description:     "Test",
		TransactionDate: txDate,
		AmountUsdCents:  5000,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	purchaseRepo := &fakePurchaseRepository{}
	purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
		return expectedPurchase, true, nil
	}

	service := NewPurchaseService(purchaseRepo, nil, nil)
	purchase, conversion, found, err := service.GetPurchaseWithConversion(context.Background(), purchaseID.String(), "USD")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !found {
		t.Fatal("expected purchase to be found")
	}
	if conversion == nil {
		t.Fatal("expected conversion for USD")
	}
	if conversion.Currency != "USD" {
		t.Fatalf("expected currency USD, got %s", conversion.Currency)
	}
	if conversion.Rate != "1.000000" {
		t.Fatalf("expected rate 1.000000, got %s", conversion.Rate)
	}
	if conversion.Amount != purchase.AmountUsd() {
		t.Fatalf("expected amount %s, got %s", purchase.AmountUsd(), conversion.Amount)
	}
}

func TestGetPurchaseWithConversion_WithRate(t *testing.T) {
	purchaseID := uuid.New()
	now := time.Now().UTC()
	txDate := now.AddDate(0, 0, -10)
	expectedPurchase := domain.Purchase{
		ID:              purchaseID,
		Description:     "Test",
		TransactionDate: txDate,
		AmountUsdCents:  5000,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	rateDate := txDate.AddDate(0, 0, -5)
	rateDecimal := decimal.NewFromFloat(0.85)

	purchaseRepo := &fakePurchaseRepository{}
	purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
		return expectedPurchase, true, nil
	}

	provider := &fakeExchangeRateProvider{
		rate:     rateDecimal,
		currency: "EUR",
		rateDate: rateDate,
	}

	service := NewPurchaseService(purchaseRepo, nil, provider)
	_, conversion, found, err := service.GetPurchaseWithConversion(context.Background(), purchaseID.String(), "EUR")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !found {
		t.Fatal("expected operation to succeed")
	}
	if conversion == nil {
		t.Fatal("expected conversion details")
	}
	if conversion.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %s", conversion.Currency)
	}
	// 50.00 USD * 0.85 = 42.50 EUR
	if conversion.Amount != "42.50" {
		t.Fatalf("expected converted amount 42.50, got %s", conversion.Amount)
	}
}

func TestGetPurchaseWithConversion_6MonthWindow(t *testing.T) {
	purchaseID := uuid.New()
	now := time.Now().UTC()
	txDate := time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC)
	expectedPurchase := domain.Purchase{
		ID:              purchaseID,
		Description:     "Test",
		TransactionDate: txDate,
		AmountUsdCents:  5000,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Rate from more than 6 months ago
	rateDate := time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC)
	rateDecimal := decimal.NewFromFloat(0.92)

	purchaseRepo := &fakePurchaseRepository{}
	purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
		return expectedPurchase, true, nil
	}

	provider := &fakeExchangeRateProvider{
		rate:     rateDecimal,
		currency: "EUR",
		rateDate: rateDate,
	}

	service := NewPurchaseService(purchaseRepo, nil, provider)
	_, conversion, found, err := service.GetPurchaseWithConversion(context.Background(), purchaseID.String(), "EUR")

	if err == nil {
		t.Fatal("expected error for rate outside 6-month window")
	}
	if found {
		t.Fatal("expected operation to fail (found should be false)")
	}
	if conversion != nil {
		t.Fatal("expected no conversion for rate outside window")
	}
}

func TestGetPurchaseWithConversion_NotFound(t *testing.T) {
	purchaseRepo := &fakePurchaseRepository{}
	purchaseRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.Purchase, bool, error) {
		return domain.Purchase{}, false, nil
	}

	service := NewPurchaseService(purchaseRepo, nil, nil)
	_, _, found, err := service.GetPurchaseWithConversion(context.Background(), uuid.New().String(), "EUR")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found {
		t.Fatal("expected purchase not to be found")
	}
}
