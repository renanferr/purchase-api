package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/renanferr/purchase-api/internal/adapters/db"
	"github.com/renanferr/purchase-api/internal/adapters/treasury"
	"github.com/renanferr/purchase-api/internal/api"
	"github.com/renanferr/purchase-api/internal/app"
	"github.com/renanferr/purchase-api/internal/domain"
	"github.com/renanferr/purchase-api/internal/migrations"
	"github.com/renanferr/purchase-api/internal/ports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

// ErrorPathTestSuite tests comprehensive error handling and all error codes
type ErrorPathTestSuite struct {
	suite.Suite
	server           *httptest.Server
	client           *http.Client
	pool             *pgxpool.Pool
	purchaseRepo     ports.PurchaseRepository
	rateRepo         ports.ExchangeRateRepository
	treasuryProvider ports.TreasuryRateProvider
}

// SetupSuite initializes the error test suite
func (suite *ErrorPathTestSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/purchase_api?sslmode=disable"
	}

	// Connect to database
	pool, err := pgxpool.New(ctx, dbURL)
	suite.NoError(err, "Failed to connect to database")

	// Verify connection
	err = pool.Ping(ctx)
	suite.NoError(err, "Failed to ping database")

	suite.pool = pool

	// Get absolute path to project root by going up 2 directories from test file
	_, currentFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(currentFile)
	projectRoot := filepath.Dir(filepath.Dir(testDir))
	migrationsPath := filepath.Join(projectRoot, "db", "migrations")

	runner := migrations.NewRunner(pool, migrationsPath)
	err = runner.Up()
	suite.NoError(err, "Failed to run migrations")

	// Initialize repositories with real database
	suite.purchaseRepo = db.NewPurchaseRepository(pool)
	suite.rateRepo = db.NewExchangeRateRepository(pool)

	// Initialize Treasury provider (sample)
	suite.treasuryProvider = treasury.NewSampleTreasuryRateProvider(
		decimal.NewFromFloat(0.92),
		"EUR",
		time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		nil,
	)

	// Create service with all dependencies
	service := app.NewPurchaseService(suite.purchaseRepo, suite.rateRepo, suite.treasuryProvider)

	// Set up HTTP test server with router
	router := api.NewRouter(service)
	suite.server = httptest.NewServer(router)
	suite.client = &http.Client{
		Timeout: 5 * time.Second,
	}
}

// TearDownSuite runs once after all tests in the suite
func (suite *ErrorPathTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}

	if suite.pool != nil {
		// Run migrations down to clean up
		_, currentFile, _, _ := runtime.Caller(0)
		testDir := filepath.Dir(currentFile)
		projectRoot := filepath.Dir(filepath.Dir(testDir))
		migrationsPath := filepath.Join(projectRoot, "db", "migrations")

		runner := migrations.NewRunner(suite.pool, migrationsPath)
		_ = runner.Down()
		suite.pool.Close()
	}
}

// SetupTest runs before each individual test
func (suite *ErrorPathTestSuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite.pool.Exec(ctx, "TRUNCATE TABLE exchange_rates CASCADE")
	suite.pool.Exec(ctx, "TRUNCATE TABLE purchases CASCADE")
}

// TestAllErrorCodes_TableDriven validates all 7 error codes are returned in appropriate scenarios
func (suite *ErrorPathTestSuite) TestAllErrorCodes_TableDriven() {
	testCases := []struct {
		name              string
		method            string
		path              string
		requestBody       map[string]interface{}
		expectedStatus    int
		expectedErrorCode string
		description       string
	}{
		// VALIDATION_ERROR: Invalid currency code
		{
			name:              "error_validation_error_invalid_currency",
			method:            http.MethodGet,
			path:              "/purchases/550e8400-e29b-41d4-a716-446655440001?currency=XYZ123",
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "VALIDATION_ERROR",
			description:       "Currency code with more than 3 characters",
		},
		// INVALID_DATE: Future date
		{
			name:   "error_invalid_date_future_date",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "Future transaction",
				"transactionDate": "2099-12-31",
				"amountUsd":       "100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "INVALID_DATE",
			description:       "Transaction date is in the future",
		},
		// INVALID_DATE: Invalid date format
		{
			name:   "error_invalid_date_bad_format",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "Bad date",
				"transactionDate": "2026-13-01",
				"amountUsd":       "100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "INVALID_DATE",
			description:       "Invalid date format (month 13)",
		},
		// NEGATIVE_AMOUNT: Negative amount
		{
			name:   "error_negative_amount_negative_value",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "Negative amount",
				"transactionDate": "2026-06-15",
				"amountUsd":       "-100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "NEGATIVE_AMOUNT",
			description:       "Amount is negative",
		},
		// NEGATIVE_AMOUNT: Zero amount
		{
			name:   "error_negative_amount_zero_value",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "Zero amount",
				"transactionDate": "2026-06-15",
				"amountUsd":       "0.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "NEGATIVE_AMOUNT",
			description:       "Amount is zero",
		},
		// DESCRIPTION_TOO_LONG: Exceeds 50 characters
		{
			name:   "error_description_too_long_exceeds_limit",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "This description is way too long and exceeds the fifty character maximum limit allowed",
				"transactionDate": "2026-06-15",
				"amountUsd":       "100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "DESCRIPTION_TOO_LONG",
			description:       "Description exceeds 50 character limit",
		},
		// MISSING_FIELD: Missing description
		{
			name:   "error_missing_field_description",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"transactionDate": "2026-06-15",
				"amountUsd":       "100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "MISSING_FIELD",
			description:       "Description field is missing",
		},
		// MISSING_FIELD: Missing transactionDate
		{
			name:   "error_missing_field_date",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description": "Missing date",
				"amountUsd":   "100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "MISSING_FIELD",
			description:       "Transaction date field is missing",
		},
		// MISSING_FIELD: Missing amountUsd
		{
			name:   "error_missing_field_amount",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "Missing amount",
				"transactionDate": "2026-06-15",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "MISSING_FIELD",
			description:       "Amount field is missing",
		},
		// NOT_FOUND: Non-existent purchase
		{
			name:              "error_not_found_nonexistent_id",
			method:            http.MethodGet,
			path:              fmt.Sprintf("/purchases/%s", uuid.New().String()),
			expectedStatus:    http.StatusNotFound,
			expectedErrorCode: "NOT_FOUND",
			description:       "Purchase with specified ID does not exist",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			var req *http.Request
			var err error

			if tc.method == http.MethodPost {
				body, marshalErr := json.Marshal(tc.requestBody)
				suite.NoError(marshalErr)
				req, err = http.NewRequest(tc.method, suite.server.URL+tc.path, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tc.method, suite.server.URL+tc.path, nil)
			}

			suite.NoError(err)

			// Submit request
			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			// Verify status
			suite.Equal(tc.expectedStatus, resp.StatusCode, "Expected status %d for %s, got %d", tc.expectedStatus, tc.description, resp.StatusCode)

			// Parse response
			var respBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			suite.NoError(err)

			// Verify error code
			suite.Equal(tc.expectedErrorCode, respBody["code"], "Error code mismatch for %s", tc.description)
			suite.Contains(respBody, "message", "Response should contain message field")
			suite.Contains(respBody, "timestamp", "Response should contain timestamp field")
		})
	}
}

// TestErrorResponseSchema validates error response format
func (suite *ErrorPathTestSuite) TestErrorResponseSchema() {
	testCases := []struct {
		name            string
		expectedCode    string
		requestBodyFunc func() map[string]interface{}
	}{
		{
			name:         "schema_validation_error",
			expectedCode: "VALIDATION_ERROR",
			requestBodyFunc: func() map[string]interface{} {
				return map[string]interface{}{
					"description":     "Test",
					"transactionDate": "2026-06-15",
					"amountUsd":       "ABC",
				}
			},
		},
		{
			name:         "schema_invalid_date",
			expectedCode: "INVALID_DATE",
			requestBodyFunc: func() map[string]interface{} {
				return map[string]interface{}{
					"description":     "Test",
					"transactionDate": "not-a-date",
					"amountUsd":       "100.00",
				}
			},
		},
		{
			name:         "schema_missing_field",
			expectedCode: "MISSING_FIELD",
			requestBodyFunc: func() map[string]interface{} {
				return map[string]interface{}{
					"description": "Test",
					"amountUsd":   "100.00",
				}
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			body, err := json.Marshal(tc.requestBodyFunc())
			suite.NoError(err)

			req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/purchases", bytes.NewReader(body))
			suite.NoError(err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			var respBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			suite.NoError(err)

			// Validate error response schema
			suite.Contains(respBody, "code", "Response must contain 'code' field")
			suite.Contains(respBody, "message", "Response must contain 'message' field")
			suite.Contains(respBody, "timestamp", "Response must contain 'timestamp' field")

			// Validate field types
			_, ok := respBody["code"].(string)
			suite.True(ok, "Code must be a string")

			_, ok = respBody["message"].(string)
			suite.True(ok, "Message must be a string")

			_, ok = respBody["timestamp"].(string)
			suite.True(ok, "Timestamp must be a string")

			// Validate non-empty values
			code := respBody["code"].(string)
			suite.NotEmpty(code, "Code must not be empty")

			message := respBody["message"].(string)
			suite.NotEmpty(message, "Message must not be empty")

			timestamp := respBody["timestamp"].(string)
			suite.NotEmpty(timestamp, "Timestamp must not be empty")
		})
	}
}

// TestErrorPathWithRequestID validates X-Request-ID propagation in error responses
func (suite *ErrorPathTestSuite) TestErrorPathWithRequestID() {
	testCases := []struct {
		name         string
		requestID    string
		hasRequestID bool
	}{
		{
			name:         "error_with_provided_request_id",
			requestID:    "test-error-request-123",
			hasRequestID: true,
		},
		{
			name:         "error_with_auto_generated_request_id",
			requestID:    "",
			hasRequestID: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create request that will cause error
			requestBody := map[string]interface{}{
				"transactionDate": "2099-12-31", // Future date = error
				"amountUsd":       "100.00",
				// missing description
			}

			body, err := json.Marshal(requestBody)
			suite.NoError(err)

			req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/purchases", bytes.NewReader(body))
			suite.NoError(err)
			req.Header.Set("Content-Type", "application/json")

			if tc.requestID != "" {
				req.Header.Set("X-Request-ID", tc.requestID)
			}

			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			// Check status is error
			suite.Equal(http.StatusBadRequest, resp.StatusCode)

			// Check X-Request-ID in response header
			if tc.hasRequestID {
				respRequestID := resp.Header.Get("X-Request-ID")
				if tc.requestID != "" {
					suite.Equal(tc.requestID, respRequestID)
				} else {
					suite.NotEmpty(respRequestID)
				}
			}
		})
	}
}

// TestErrorPathNoPlainTextErrors validates no plain text errors (all use structured format)
func (suite *ErrorPathTestSuite) TestErrorPathNoPlainTextErrors() {
	ctx := context.Background()

	// Create a purchase for conversion rate not found test
	testPurchase := domain.Purchase{
		ID:              uuid.New(),
		Description:     "Rate test",
		TransactionDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), // Very old date
		AmountUsdCents:  10000,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	_ = suite.purchaseRepo.Create(ctx, testPurchase)

	testCases := []struct {
		name        string
		method      string
		url         string
		requestBody map[string]interface{}
	}{
		{
			name:   "no_plain_text_validation_error",
			method: http.MethodPost,
			url:    "/purchases",
			requestBody: map[string]interface{}{
				"description": "Test",
			},
		},
		{
			name:   "no_plain_text_not_found_error",
			method: http.MethodGet,
			url:    fmt.Sprintf("/purchases/%s", uuid.New().String()),
		},
		{
			name:   "no_plain_text_rate_not_found_error",
			method: http.MethodGet,
			url:    fmt.Sprintf("/purchases/%s?currency=GBP", testPurchase.ID),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			var req *http.Request
			var err error

			if tc.method == http.MethodPost {
				body, marshalErr := json.Marshal(tc.requestBody)
				suite.NoError(marshalErr)
				req, err = http.NewRequest(tc.method, suite.server.URL+tc.url, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tc.method, suite.server.URL+tc.url, nil)
			}

			suite.NoError(err)

			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			// Should be error status
			suite.True(resp.StatusCode >= 400, "Expected error status code")

			// Should be JSON structured error, not plain text
			contentType := resp.Header.Get("Content-Type")
			suite.Contains(contentType, "application/json", "Error response must be JSON")

			// Parse and validate structure
			var respBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			suite.NoError(err, "Error response must be valid JSON")

			// Must have structured error fields
			suite.Contains(respBody, "code", "Error response must have code field")
			suite.Contains(respBody, "message", "Error response must have message field")
			suite.Contains(respBody, "timestamp", "Error response must have timestamp field")

			// Verify rate_not_found returns RATE_NOT_FOUND error code
			if tc.name == "no_plain_text_rate_not_found_error" {
				suite.Equal("RATE_NOT_FOUND", respBody["code"], "Expected RATE_NOT_FOUND error code for missing rate")
			}
		})
	}
}

// TestErrorPathTestSuite entry point
func TestErrorPathTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorPathTestSuite))
}
