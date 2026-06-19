package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// PurchaseSuite tests Purchase domain entity
type PurchaseSuite struct {
	suite.Suite
}

func TestPurchaseSuite(t *testing.T) {
	suite.Run(t, new(PurchaseSuite))
}

// TestNewPurchase creates valid purchase
func (s *PurchaseSuite) TestNewPurchase() {
	desc := "Office supplies"
	date := time.Now().AddDate(0, 0, -1)
	cents := int64(150000)

	purchase, err := NewPurchase(desc, date, cents)

	s.NoError(err)
	s.NotZero(purchase.ID)
	s.Equal(desc, purchase.Description)
	s.Equal(date.Format("2006-01-02"), purchase.TransactionDate.Format("2006-01-02"))
	s.Equal(int64(150000), purchase.AmountUsdCents)
}

// TestNewPurchaseInvalidDescription tests description validation
func (s *PurchaseSuite) TestNewPurchaseInvalidDescription() {
	testCases := []struct {
		name        string
		description string
		shouldError bool
	}{
		{
			name:        "valid short description",
			description: "Test",
			shouldError: false,
		},
		{
			name:        "valid 50 char description",
			description: "12345678901234567890123456789012345678901234567890",
			shouldError: false,
		},
		{
			name:        "empty description",
			description: "",
			shouldError: true,
		},
		{
			name:        "description exceeds 50 chars",
			description: "123456789012345678901234567890123456789012345678901",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			purchase, err := NewPurchase(tc.description, time.Now(), 150000)

			if tc.shouldError {
				s.Error(err)
			} else {
				s.NoError(err)
				s.NotZero(purchase.ID)
			}
		})
	}
}

// TestAmountUsd tests amount formatting
func (s *PurchaseSuite) TestAmountUsd() {
	testCases := []struct {
		name     string
		cents    int64
		expected string
	}{
		{
			name:     "simple amount",
			cents:    150000,
			expected: "1500.00",
		},
		{
			name:     "amount with single digit cents",
			cents:    150005,
			expected: "1500.05",
		},
		{
			name:     "small amount",
			cents:    100,
			expected: "1.00",
		},
		{
			name:     "single cent",
			cents:    1,
			expected: "0.01",
		},
		{
			name:     "zero cents",
			cents:    0,
			expected: "0.00",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			purchase := &Purchase{AmountUsdCents: tc.cents}
			amount := purchase.AmountUsd()
			s.Equal(tc.expected, amount)
		})
	}
}

// ExchangeRateSuite tests ExchangeRate domain entity
type ExchangeRateSuite struct {
	suite.Suite
}

func TestExchangeRateSuite(t *testing.T) {
	suite.Run(t, new(ExchangeRateSuite))
}

// TestExchangeRate creates valid exchange rate
func (s *ExchangeRateSuite) TestExchangeRate() {
	rate := ExchangeRate{
		Currency: "EUR",
		RateDate: time.Now(),
		Rate:     decimal.NewFromFloat(0.85),
	}

	s.NotEmpty(rate.Currency)
	s.NotZero(rate.RateDate)
	s.NotZero(rate.Rate)
}

// TestExchangeRateString tests rate string formatting
func (s *ExchangeRateSuite) TestExchangeRateString() {
	testCases := []struct {
		name     string
		rate     string
		expected string
	}{
		{
			name:     "simple rate",
			rate:     "0.85",
			expected: "0.850000",
		},
		{
			name:     "rate greater than 1",
			rate:     "1.20",
			expected: "1.200000",
		},
		{
			name:     "rate with many decimals",
			rate:     "1.234567",
			expected: "1.234567",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			rateVal, err := decimal.NewFromString(tc.rate)
			s.NoError(err)
			er := ExchangeRate{
				Currency: "EUR",
				RateDate: time.Now(),
				Rate:     rateVal,
			}
			result := er.RateString()
			s.Equal(tc.expected, result)
		})
	}
}

// MoneySuite tests Money value object
type MoneySuite struct {
	suite.Suite
}

func TestMoneySuite(t *testing.T) {
	suite.Run(t, new(MoneySuite))
}

// TestParseMoney parses money string
func (s *MoneySuite) TestParseMoney() {
	testCases := []struct {
		name      string
		value     string
		shouldErr bool
		expected  int64
	}{
		{
			name:      "simple amount",
			value:     "1500.00",
			shouldErr: false,
			expected:  150000,
		},
		{
			name:      "amount without decimals",
			value:     "100",
			shouldErr: false,
			expected:  10000,
		},
		{
			name:      "small amount",
			value:     "0.50",
			shouldErr: false,
			expected:  50,
		},
		{
			name:      "zero amount",
			value:     "0.00",
			shouldErr: true,
		},
		{
			name:      "negative amount",
			value:     "-100.00",
			shouldErr: true,
		},
		{
			name:      "invalid format",
			value:     "not-a-number",
			shouldErr: true,
		},
		{
			name:      "empty string",
			value:     "",
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			money, err := ParseMoney(tc.value)
			if tc.shouldErr {
				s.Error(err)
			} else {
				s.NoError(err)
				s.Equal(tc.expected, money.Cents)
			}
		})
	}
}

// TestMoneyString tests money string formatting
func (s *MoneySuite) TestMoneyString() {
	testCases := []struct {
		name     string
		cents    int64
		expected string
	}{
		{
			name:     "simple amount",
			cents:    150000,
			expected: "1500.00",
		},
		{
			name:     "single digit cents",
			cents:    150005,
			expected: "1500.05",
		},
		{
			name:     "small amount",
			cents:    100,
			expected: "1.00",
		},
		{
			name:     "single cent",
			cents:    1,
			expected: "0.01",
		},
		{
			name:     "zero cents",
			cents:    0,
			expected: "0.00",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			money := Money{Cents: tc.cents}
			result := money.String()
			s.Equal(tc.expected, result)
		})
	}
}

// UUIDSuite tests UUID generation in entities
type UUIDSuite struct {
	suite.Suite
}

func TestUUIDSuite(t *testing.T) {
	suite.Run(t, new(UUIDSuite))
}

// TestPurchaseIDIsUUID validates purchase ID is proper UUID
func (s *UUIDSuite) TestPurchaseIDIsUUID() {
	purchase, err := NewPurchase("Test", time.Now(), 10000)
	s.NoError(err)

	// Should be parseable as UUID
	_, err = uuid.Parse(purchase.ID.String())
	s.NoError(err)
}

// TestPurchaseIDUniqueness tests that each purchase gets unique ID
func (s *UUIDSuite) TestPurchaseIDUniqueness() {
	purchase1, err1 := NewPurchase("Test 1", time.Now(), 10000)
	purchase2, err2 := NewPurchase("Test 2", time.Now(), 20000)

	s.NoError(err1)
	s.NoError(err2)
	s.NotEqual(purchase1.ID, purchase2.ID)
}

// TimestampSuite tests timestamp handling
type TimestampSuite struct {
	suite.Suite
}

func TestTimestampSuite(t *testing.T) {
	suite.Run(t, new(TimestampSuite))
}

// TestPurchaseTimestamps tests purchase timestamp fields
func (s *TimestampSuite) TestPurchaseTimestamps() {
	now := time.Now().UTC()
	purchase, err := NewPurchase("Test", now, 10000)
	s.NoError(err)

	// Timestamps may be set by service/database layer, not domain entity
	// Just verify they exist as fields on the struct
	s.Zero(purchase.CreatedAt) // Domain entity doesn't set these
	s.Zero(purchase.UpdatedAt) // Domain entity doesn't set these
	// But the fields should be of the correct type (time.Time)
	s.IsType(time.Time{}, purchase.CreatedAt)
	s.IsType(time.Time{}, purchase.UpdatedAt)
}

// ValidationSuite tests domain validation
type ValidationSuite struct {
	suite.Suite
}

func TestValidationSuite(t *testing.T) {
	suite.Run(t, new(ValidationSuite))
}

// TestPurchaseValidation tests purchase domain validation
func (s *ValidationSuite) TestPurchaseValidation() {
	testCases := []struct {
		name        string
		description string
		cents       int64
		shouldPass  bool
	}{
		{
			name:        "valid purchase",
			description: "Office supplies",
			cents:       150000,
			shouldPass:  true,
		},
		{
			name:        "invalid empty description",
			description: "",
			cents:       150000,
			shouldPass:  false,
		},
		{
			name:        "invalid zero cents",
			description: "Test",
			cents:       0,
			shouldPass:  false,
		},
		{
			name:        "invalid negative cents",
			description: "Test",
			cents:       -100,
			shouldPass:  false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			purchase, err := NewPurchase(tc.description, time.Now(), tc.cents)
			if tc.shouldPass {
				s.NoError(err)
				s.NotZero(purchase.ID)
			} else {
				s.Error(err)
			}
		})
	}
}
