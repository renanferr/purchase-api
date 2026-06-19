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
	"strings"
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

// PurchaseAPIIntegrationTestSuite tests the Purchase API with real database
type PurchaseAPIIntegrationTestSuite struct {
	suite.Suite
	server             *httptest.Server
	client             *http.Client
	pool               *pgxpool.Pool
	purchaseRepo       ports.PurchaseRepository
	rateRepo           ports.ExchangeRateRepository
	treasuryProvider   ports.TreasuryRateProvider
	testPurchaseID     uuid.UUID
	testPurchaseAmount string
}

// SetupSuite runs once before all tests in the suite
func (suite *PurchaseAPIIntegrationTestSuite) SetupSuite() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get test database URL from environment or use default test database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		// Default to separate test database: purchase_api_test
		dbURL = "postgres://postgres:postgres@localhost:5432/purchase_api_test?sslmode=disable"
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

	// Initialize Treasury provider (real API)
	httpClient := &http.Client{Timeout: 10 * time.Second}
	suite.treasuryProvider = treasury.NewExchangeRateProvider(httpClient)

	// Create service with all dependencies
	service := app.NewPurchaseService(suite.purchaseRepo, suite.rateRepo, suite.treasuryProvider)

	// Set up HTTP test server with router
	router := api.NewRouter(service)
	suite.server = httptest.NewServer(router)
	suite.client = &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create a sample purchase for retrieval tests
	testPurchase := domain.Purchase{
		ID:              uuid.New(),
		Description:     "Test Purchase",
		TransactionDate: time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC),
		AmountUsdCents:  150000, // $1500.00
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	err = suite.purchaseRepo.Create(ctx, testPurchase)
	suite.NoError(err)
	suite.testPurchaseID = testPurchase.ID
	suite.testPurchaseAmount = "1500.00"

	// Add sample exchange rates for conversion tests
	eurRate := domain.ExchangeRate{
		Currency:  "EUR",
		Rate:      decimal.NewFromFloat(0.92),
		RateDate:  time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now().UTC(),
	}
	err = suite.rateRepo.Create(ctx, eurRate)
	suite.NoError(err)

	usdRate := domain.ExchangeRate{
		Currency:  "USD",
		Rate:      decimal.NewFromFloat(1.0),
		RateDate:  time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now().UTC(),
	}
	err = suite.rateRepo.Create(ctx, usdRate)
	suite.NoError(err)
}

// TearDownSuite runs once after all tests in the suite
func (suite *PurchaseAPIIntegrationTestSuite) TearDownSuite() {
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
func (suite *PurchaseAPIIntegrationTestSuite) SetupTest() {
	// Clear tables between tests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite.pool.Exec(ctx, "TRUNCATE TABLE exchange_rates CASCADE")
	suite.pool.Exec(ctx, "TRUNCATE TABLE purchases CASCADE")

	// Re-populate with test data
	eurRate := domain.ExchangeRate{
		Currency:  "EUR",
		Rate:      decimal.NewFromFloat(0.92),
		RateDate:  time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now().UTC(),
	}
	_ = suite.rateRepo.Create(ctx, eurRate)

	usdRate := domain.ExchangeRate{
		Currency:  "USD",
		Rate:      decimal.NewFromFloat(1.0),
		RateDate:  time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now().UTC(),
	}
	_ = suite.rateRepo.Create(ctx, usdRate)

	testPurchase := domain.Purchase{
		ID:              uuid.New(),
		Description:     "Test Purchase",
		TransactionDate: time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC),
		AmountUsdCents:  150000,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	_ = suite.purchaseRepo.Create(ctx, testPurchase)
	suite.testPurchaseID = testPurchase.ID
}

// TestCreatePurchaseEndpoint_TableDriven validates POST /purchases endpoint
func (suite *PurchaseAPIIntegrationTestSuite) TestCreatePurchaseEndpoint_TableDriven() {
	testCases := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedCode   string
		checkFields    bool
	}{
		{
			name: "create_purchase_success",
			requestBody: map[string]interface{}{
				"description":     "Flight to Paris",
				"transactionDate": "2026-06-15",
				"amountUsd":       "1500.00",
			},
			expectedStatus: http.StatusCreated,
			checkFields:    true,
		},
		{
			name: "validation_error_missing_field",
			requestBody: map[string]interface{}{
				"transactionDate": "2026-06-15",
				"amountUsd":       "1500.00",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "MISSING_FIELD",
		},
		{
			name: "validation_error_invalid_date",
			requestBody: map[string]interface{}{
				"description":     "Future transaction",
				"transactionDate": "2099-12-31",
				"amountUsd":       "1500.00",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "INVALID_DATE",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			body, err := json.Marshal(tc.requestBody)
			suite.NoError(err)

			req, err := http.NewRequest(http.MethodPost, suite.server.URL+"/purchases", strings.NewReader(string(body)))
			suite.NoError(err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			suite.Equal(tc.expectedStatus, resp.StatusCode)

			var respBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			suite.NoError(err)

			if tc.checkFields {
				suite.Contains(respBody, "id")
				suite.Contains(respBody, "description")
				suite.Contains(respBody, "transactionDate")
				suite.Contains(respBody, "amountUsd")
				suite.Contains(respBody, "createdAt")
			} else if tc.expectedCode != "" {
				suite.Equal(tc.expectedCode, respBody["code"])
			}
		})
	}
}

// TestGetPurchaseEndpoint_TableDriven validates GET /purchases/{id} endpoint
func (suite *PurchaseAPIIntegrationTestSuite) TestGetPurchaseEndpoint_TableDriven() {
	testCases := []struct {
		name           string
		purchaseID     string
		currency       string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "retrieve_purchase_no_conversion",
			purchaseID:     suite.testPurchaseID.String(),
			currency:       "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "retrieve_purchase_with_conversion",
			purchaseID:     suite.testPurchaseID.String(),
			currency:       "EUR",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "retrieve_nonexistent_purchase",
			purchaseID:     uuid.New().String(),
			currency:       "",
			expectedStatus: http.StatusNotFound,
			expectedCode:   "NOT_FOUND",
		},
		{
			name:           "invalid_currency_code",
			purchaseID:     suite.testPurchaseID.String(),
			currency:       "INVALID",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "VALIDATION_ERROR",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Build GET request to /purchases/{id}?currency=X
			url := fmt.Sprintf("%s/purchases/%s", suite.server.URL, tc.purchaseID)
			if tc.currency != "" {
				url = fmt.Sprintf("%s?currency=%s", url, tc.currency)
			}

			req, err := http.NewRequest(http.MethodGet, url, nil)
			suite.NoError(err)

			// Submit request
			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			// Assert response status code
			suite.Equal(tc.expectedStatus, resp.StatusCode)

			// Parse response body
			var respBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			suite.NoError(err)

			// Check response structure
			if tc.expectedStatus == http.StatusOK {
				suite.Contains(respBody, "id")
				suite.Contains(respBody, "description")
				suite.Contains(respBody, "transactionDate")
				suite.Contains(respBody, "amountUsd")
				suite.Contains(respBody, "createdAt")
				suite.Contains(respBody, "updatedAt")

				// If conversion requested, check for exchange rate info
				if tc.currency != "" && tc.currency != "INVALID" {
					suite.Contains(respBody, "rate")
					suite.Contains(respBody, "convertedAmount")
				}
			} else if tc.expectedCode != "" {
				suite.Equal(tc.expectedCode, respBody["code"])
			}
		})
	}
}

// TestCurrencyConversionFlow_TableDriven validates end-to-end conversion flow
func (suite *PurchaseAPIIntegrationTestSuite) TestCurrencyConversionFlow_TableDriven() {
	ctx := context.Background()

	// Create a fresh purchase for conversion testing with recent date
	conversionPurchase := domain.Purchase{
		ID:              uuid.New(),
		Description:     "Conversion Test",
		TransactionDate: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), // Recent date with available rates
		AmountUsdCents:  10000,                                        // $100.00
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}
	suite.purchaseRepo.Create(ctx, conversionPurchase)
	conversionID := conversionPurchase.ID.String()

	testCases := []struct {
		name             string
		purchaseID       string
		targetCurrency   string
		expectedStatus   int
		shouldHaveRate   bool
		shouldHaveAmount bool
		expectedCode     string
	}{
		{
			name:             "conversion_with_valid_rate_eur",
			purchaseID:       conversionID,
			targetCurrency:   "EUR",
			expectedStatus:   http.StatusOK,
			shouldHaveRate:   true,
			shouldHaveAmount: true,
		},
		{
			name:             "conversion_usd_pass_through",
			purchaseID:       conversionID,
			targetCurrency:   "USD",
			expectedStatus:   http.StatusOK,
			shouldHaveRate:   true,
			shouldHaveAmount: true,
		},
		{
			name:             "conversion_with_valid_rate_gbp",
			purchaseID:       conversionID,
			targetCurrency:   "GBP",
			expectedStatus:   http.StatusOK,
			shouldHaveRate:   true,
			shouldHaveAmount: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Request conversion to targetCurrency
			url := fmt.Sprintf("%s/purchases/%s?currency=%s", suite.server.URL, tc.purchaseID, tc.targetCurrency)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			suite.NoError(err)

			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			// Assert response status
			suite.Equal(tc.expectedStatus, resp.StatusCode)

			// Parse response
			var respBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			suite.NoError(err)

			// Verify response structure
			if tc.expectedStatus == http.StatusOK {
				suite.Contains(respBody, "amountUsd")
				suite.Equal("100.00", respBody["amountUsd"])

				if tc.shouldHaveRate {
					suite.Contains(respBody, "rate")
					suite.Contains(respBody, "convertedAmount")
					suite.NotEmpty(respBody["convertedAmount"])
				}
			} else {
				suite.Equal(tc.expectedCode, respBody["code"])
			}
		})
	}
}

// TestRequestIDPropagation_TableDriven validates X-Request-ID header handling
func (suite *PurchaseAPIIntegrationTestSuite) TestRequestIDPropagation_TableDriven() {
	testCases := []struct {
		name              string
		providedRequestID string
		shouldPropagate   bool
	}{
		{
			name:              "request_id_provided_in_header",
			providedRequestID: "test-request-123",
			shouldPropagate:   true,
		},
		{
			name:              "request_id_auto_generated",
			providedRequestID: "",
			shouldPropagate:   true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create request with optional request ID
			url := fmt.Sprintf("%s/purchases/%s", suite.server.URL, suite.testPurchaseID)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			suite.NoError(err)

			if tc.providedRequestID != "" {
				req.Header.Set("X-Request-ID", tc.providedRequestID)
			}

			// Submit request
			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			// Check for X-Request-ID in response
			if tc.shouldPropagate {
				requestID := resp.Header.Get("X-Request-ID")
				if tc.providedRequestID != "" {
					suite.Equal(tc.providedRequestID, requestID)
				} else {
					// Should have auto-generated UUID format
					suite.NotEmpty(requestID)
				}
			}

			// Verify success response
			suite.Equal(http.StatusOK, resp.StatusCode)
		})
	}
}

// TestErrorHandling_TableDriven validates error code mapping
func (suite *PurchaseAPIIntegrationTestSuite) TestErrorHandling_TableDriven() {
	testCases := []struct {
		name              string
		method            string
		path              string
		requestBody       map[string]interface{}
		expectedStatus    int
		expectedErrorCode string
	}{
		{
			name:   "validation_error_missing_description",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"transactionDate": "2026-06-17",
				"amountUsd":       "100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "MISSING_FIELD",
		},
		{
			name:   "validation_error_negative_amount",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "Test",
				"transactionDate": "2026-06-17",
				"amountUsd":       "-50.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "NEGATIVE_AMOUNT",
		},
		{
			name:   "validation_error_description_too_long",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "This is a very long description that exceeds the 50 character limit for purchase descriptions",
				"transactionDate": "2026-06-17",
				"amountUsd":       "100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "DESCRIPTION_TOO_LONG",
		},
		{
			name:   "validation_error_invalid_date",
			method: http.MethodPost,
			path:   "/purchases",
			requestBody: map[string]interface{}{
				"description":     "Test",
				"transactionDate": "2099-12-31",
				"amountUsd":       "100.00",
			},
			expectedStatus:    http.StatusBadRequest,
			expectedErrorCode: "INVALID_DATE",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Marshal request body
			body, err := json.Marshal(tc.requestBody)
			suite.NoError(err)

			// Create request
			url := suite.server.URL + tc.path
			req, err := http.NewRequest(tc.method, url, bytes.NewReader(body))
			suite.NoError(err)
			req.Header.Set("Content-Type", "application/json")

			// Submit request
			resp, err := suite.client.Do(req)
			suite.NoError(err)
			defer resp.Body.Close()

			// Verify status
			suite.Equal(tc.expectedStatus, resp.StatusCode)

			// Parse response
			var respBody map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&respBody)
			suite.NoError(err)

			// Verify error code
			suite.Equal(tc.expectedErrorCode, respBody["code"])
			suite.Contains(respBody, "message")
			suite.Contains(respBody, "timestamp")
		})
	}
}

// TestPurchaseAPIIntegrationSuite entry point
func TestPurchaseAPIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PurchaseAPIIntegrationTestSuite))
}
