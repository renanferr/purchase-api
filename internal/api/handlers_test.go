package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/purchase-api/internal/app"
	"github.com/example/purchase-api/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

// ValidationSuite tests validation functions
type ValidationSuite struct {
	suite.Suite
}

func TestValidationSuite(t *testing.T) {
	suite.Run(t, new(ValidationSuite))
}

// TestValidateCreatePurchaseRequest tests request validation
func (s *ValidationSuite) TestValidateCreatePurchaseRequest() {
	testCases := []struct {
		name          string
		req           CreatePurchaseRequest
		expectedError bool
		expectedCode  string
		description   string
	}{
		{
			name: "valid request",
			req: CreatePurchaseRequest{
				Description:     "Office supplies",
				TransactionDate: "2026-06-15",
				AmountUsd:       "1500.00",
			},
			expectedError: false,
		},
		{
			name: "missing description",
			req: CreatePurchaseRequest{
				Description:     "",
				TransactionDate: "2026-06-15",
				AmountUsd:       "1500.00",
			},
			expectedError: true,
			expectedCode:  ErrorCodeMissingField,
		},
		{
			name: "description too long",
			req: CreatePurchaseRequest{
				Description:     "This is a very long description that exceeds the fifty character limit imposed",
				TransactionDate: "2026-06-15",
				AmountUsd:       "1500.00",
			},
			expectedError: true,
			expectedCode:  ErrorCodeDescriptionTooLong,
		},
		{
			name: "missing transaction date",
			req: CreatePurchaseRequest{
				Description:     "Office supplies",
				TransactionDate: "",
				AmountUsd:       "1500.00",
			},
			expectedError: true,
			expectedCode:  ErrorCodeMissingField,
		},
		{
			name: "invalid date format",
			req: CreatePurchaseRequest{
				Description:     "Office supplies",
				TransactionDate: "06/15/2026",
				AmountUsd:       "1500.00",
			},
			expectedError: true,
			expectedCode:  ErrorCodeInvalidDate,
		},
		{
			name: "future date",
			req: CreatePurchaseRequest{
				Description:     "Office supplies",
				TransactionDate: time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
				AmountUsd:       "1500.00",
			},
			expectedError: true,
			expectedCode:  ErrorCodeInvalidDate,
		},
		{
			name: "missing amount",
			req: CreatePurchaseRequest{
				Description:     "Office supplies",
				TransactionDate: "2026-06-15",
				AmountUsd:       "",
			},
			expectedError: true,
			expectedCode:  ErrorCodeMissingField,
		},
		{
			name: "negative amount",
			req: CreatePurchaseRequest{
				Description:     "Office supplies",
				TransactionDate: "2026-06-15",
				AmountUsd:       "-100.00",
			},
			expectedError: true,
			expectedCode:  ErrorCodeNegativeAmount,
		},
		{
			name: "zero amount",
			req: CreatePurchaseRequest{
				Description:     "Office supplies",
				TransactionDate: "2026-06-15",
				AmountUsd:       "0.00",
			},
			expectedError: true,
			expectedCode:  ErrorCodeNegativeAmount,
		},
		{
			name: "invalid amount format",
			req: CreatePurchaseRequest{
				Description:     "Office supplies",
				TransactionDate: "2026-06-15",
				AmountUsd:       "not-a-number",
			},
			expectedError: true,
			expectedCode:  ErrorCodeValidationError,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := validateCreatePurchaseRequest(tc.req)
			if tc.expectedError {
				s.NotNil(err, "expected validation error")
				s.Equal(tc.expectedCode, err.Code, "error code mismatch")
			} else {
				s.Nil(err, "expected no validation error")
			}
		})
	}
}

// TestParseDate tests date parsing
func (s *ValidationSuite) TestParseDate() {
	testCases := []struct {
		name        string
		dateStr     string
		shouldError bool
		description string
	}{
		{
			name:        "valid ISO 8601 date",
			dateStr:     "2026-06-15",
			shouldError: false,
		},
		{
			name:        "valid RFC3339 date",
			dateStr:     "2026-06-15T14:30:00Z",
			shouldError: false,
		},
		{
			name:        "invalid date format",
			dateStr:     "06/15/2026",
			shouldError: true,
		},
		{
			name:        "invalid date",
			dateStr:     "2026-13-01",
			shouldError: true,
		},
		{
			name:        "empty string",
			dateStr:     "",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			date, err := parseDate(tc.dateStr)
			if tc.shouldError {
				s.NotNil(err)
			} else {
				s.NoError(err)
				s.NotZero(date)
			}
		})
	}
}

// TestIsInTheFuture tests future date detection
func (s *ValidationSuite) TestIsInTheFuture() {
	testCases := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			name:     "future date",
			date:     time.Now().AddDate(1, 0, 0),
			expected: true,
		},
		{
			name:     "past date",
			date:     time.Now().AddDate(-1, 0, 0),
			expected: false,
		},
		{
			name:     "today",
			date:     time.Now(),
			expected: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := isInTheFuture(tc.date)
			s.Equal(tc.expected, result)
		})
	}
}

// TestValidateAmount tests amount validation
func (s *ValidationSuite) TestValidateAmount() {
	testCases := []struct {
		name          string
		amount        string
		expectedError bool
		expectedCode  string
	}{
		{
			name:          "valid positive amount",
			amount:        "1500.50",
			expectedError: false,
		},
		{
			name:          "valid integer amount",
			amount:        "1500",
			expectedError: false,
		},
		{
			name:          "negative amount",
			amount:        "-100.00",
			expectedError: true,
			expectedCode:  ErrorCodeNegativeAmount,
		},
		{
			name:          "zero amount",
			amount:        "0.00",
			expectedError: true,
			expectedCode:  ErrorCodeNegativeAmount,
		},
		{
			name:          "invalid format",
			amount:        "abc",
			expectedError: true,
			expectedCode:  ErrorCodeValidationError,
		},
		{
			name:          "empty string",
			amount:        "",
			expectedError: true,
			expectedCode:  ErrorCodeValidationError,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			err := validateAmount(tc.amount)
			if tc.expectedError {
				s.NotNil(err)
				s.Equal(tc.expectedCode, err.Code)
			} else {
				s.Nil(err)
			}
		})
	}
}

// TestIsValidCurrencyCode tests currency code validation
func (s *ValidationSuite) TestIsValidCurrencyCode() {
	testCases := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "valid code USD",
			code:     "USD",
			expected: true,
		},
		{
			name:     "valid code EUR",
			code:     "EUR",
			expected: true,
		},
		{
			name:     "valid code JPY",
			code:     "JPY",
			expected: true,
		},
		{
			name:     "lowercase",
			code:     "eur",
			expected: false,
		},
		{
			name:     "too short",
			code:     "EU",
			expected: false,
		},
		{
			name:     "too long",
			code:     "EURO",
			expected: false,
		},
		{
			name:     "with numbers",
			code:     "US1",
			expected: false,
		},
		{
			name:     "empty",
			code:     "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := isValidCurrencyCode(tc.code)
			s.Equal(tc.expected, result)
		})
	}
}

// HandlerSuite tests HTTP handlers
type HandlerSuite struct {
	suite.Suite
	service *MockPurchaseService
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlerSuite))
}

func (s *HandlerSuite) SetupTest() {
	s.service = NewMockPurchaseService()
}

// TestHealthHandler tests the health check endpoint
func (s *HandlerSuite) TestHealthHandler() {
	logger := NewLogger()
	handler := healthHandler(logger)

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-Request-ID", "test-id")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))
	s.Equal("test-id", w.Header().Get("X-Request-ID"))

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	s.NoError(err)
	s.Equal("ok", resp["status"])
}

// TestReadinessHandler tests the readiness check endpoint
func (s *HandlerSuite) TestReadinessHandler() {
	logger := NewLogger()
	handler := readinessHandler(nil, logger) // nil pool means no DB check

	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)
	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	s.NoError(err)
	s.Equal("ok", resp["status"])
}

// TestGetOrGenerateRequestID tests request ID generation/extraction
func (s *HandlerSuite) TestGetOrGenerateRequestID() {
	req := httptest.NewRequest("GET", "/", nil)

	// Test generation when header is missing
	id1 := getOrGenerateRequestID(req)
	s.NotEmpty(id1)
	_, err := uuid.Parse(id1)
	s.NoError(err)

	// Test extraction when header is present
	expectedID := "custom-request-id"
	req.Header.Set("X-Request-ID", expectedID)
	id2 := getOrGenerateRequestID(req)
	s.Equal(expectedID, id2)
}

// MockPurchaseService is a mock implementation for testing
type MockPurchaseService struct {
	createFunc func(ctx context.Context, desc, date, amount, ref string) (domain.Purchase, string, error)
	getFunc    func(ctx context.Context, id string) (domain.Purchase, bool, error)
}

func NewMockPurchaseService() *MockPurchaseService {
	return &MockPurchaseService{}
}

func (m *MockPurchaseService) CreatePurchase(ctx context.Context, desc, date, amount, ref string) (domain.Purchase, string, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, desc, date, amount, ref)
	}
	return domain.Purchase{}, "", nil
}

func (m *MockPurchaseService) GetPurchase(ctx context.Context, id string) (domain.Purchase, bool, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return domain.Purchase{}, false, nil
}

func (m *MockPurchaseService) GetPurchaseWithConversion(ctx context.Context, id, currency string) (domain.Purchase, *app.CurrencyConversion, bool, error) {
	return domain.Purchase{}, nil, false, nil
}

// UtilityFunctionSuite tests utility functions
type UtilityFunctionSuite struct {
	suite.Suite
}

func TestUtilityFunctionSuite(t *testing.T) {
	suite.Run(t, new(UtilityFunctionSuite))
}

// TestGetCurrentTimestamp tests timestamp generation
func (s *UtilityFunctionSuite) TestGetCurrentTimestamp() {
	ts := getCurrentTimestamp()
	s.NotEmpty(ts)

	// Should be parseable as RFC3339
	_, err := time.Parse("2006-01-02T15:04:05Z", ts)
	s.NoError(err)
}

// TestStatusForErrorCode tests HTTP status mapping
func (s *UtilityFunctionSuite) TestStatusForErrorCode() {
	testCases := []struct {
		code           string
		expectedStatus int
	}{
		{ErrorCodeValidationError, http.StatusBadRequest},
		{ErrorCodeInvalidDate, http.StatusBadRequest},
		{ErrorCodeNegativeAmount, http.StatusBadRequest},
		{ErrorCodeDescriptionTooLong, http.StatusBadRequest},
		{ErrorCodeMissingField, http.StatusBadRequest},
		{ErrorCodeRateNotFound, http.StatusBadRequest},
		{ErrorCodeNotFound, http.StatusNotFound},
	}

	for _, tc := range testCases {
		s.Run(tc.code, func() {
			status := StatusForErrorCode(tc.code)
			s.Equal(tc.expectedStatus, status)
		})
	}
}

// TestWriteErrorResponse tests error response writing
func (s *UtilityFunctionSuite) TestWriteErrorResponse() {
	w := httptest.NewRecorder()
	writeErrorResponse(w, http.StatusBadRequest, ErrorCodeInvalidDate, "Date is invalid", "req-123")

	s.Equal(http.StatusBadRequest, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))
	s.Equal("req-123", w.Header().Get("X-Request-ID"))

	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	s.NoError(err)
	s.Equal(ErrorCodeInvalidDate, errResp["code"])
	s.Equal("Date is invalid", errResp["message"])
	s.NotEmpty(errResp["timestamp"])
}
