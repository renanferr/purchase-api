package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/renanferr/purchase-api/internal/app"
	"github.com/shopspring/decimal"
)

// Error sentinel for date parsing validation
var errDateInvalid = errors.New("transactionDate must be a valid ISO 8601 date")

func NewRouter(service *app.PurchaseService, opts ...RouterOption) http.Handler {
	r := chi.NewRouter()
	logger := NewLogger()

	// Apply options to router
	var config routerConfig
	for _, opt := range opts {
		opt(&config)
	}

	// T028.5: Add request ID middleware for tracing
	r.Use(RequestIDMiddleware)

	// Documentation endpoints (Swagger UI)
	// Swagger specs are auto-generated from doc comments via swag init
	r.Get("/docs", swaggerHandler())
	r.Get("/docs/swagger.json", swaggerJSONHandler()) // Auto-generated Swagger 2.0 spec
	r.Get("/docs/swagger.yaml", swaggerYAMLHandler()) // Auto-generated Swagger 2.0 spec
	r.Get("/docs/openapi.json", swaggerJSONHandler()) // Alias for swagger.json
	r.Get("/docs/openapi.yaml", swaggerYAMLHandler()) // Alias for swagger.yaml
	// Note: Swagger UI is served via CDN, no need for separate static assets handler

	// T037: Health endpoints
	r.Get("/health", healthHandler(logger))
	r.Get("/health/ready", readinessHandler(config.pool, logger))

	// Purchase endpoints
	r.Post("/purchases", createPurchaseHandler(service, logger))
	r.Get("/purchases/{id}", getPurchaseHandler(service, logger))

	return r
}

// routerConfig holds optional configuration for the router
type routerConfig struct {
	pool *pgxpool.Pool
}

// RouterOption is a functional option for configuring the router
type RouterOption func(*routerConfig)

// WithDatabasePool provides database pool for readiness checks
func WithDatabasePool(pool *pgxpool.Pool) RouterOption {
	return func(c *routerConfig) {
		c.pool = pool
	}
}

// @Summary Create a new purchase
// @Description Create a new purchase transaction with description, date, and USD amount. Validates that description is 1-50 chars, date is in ISO 8601 format and not in the future, and amount is positive.
// @Tags purchases
// @Accept json
// @Produce json
// @Param X-Request-ID header string false "Request ID for tracing"
// @Param purchase body CreatePurchaseRequest true "Purchase details"
// @Success 201 {object} PurchaseResponse
// @Failure 400 {object} ErrorResponse "Validation error"
// @Router /purchases [post]
func createPurchaseHandler(service *app.PurchaseService, logger *Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := getOrGenerateRequestID(r)
		ctx := r.Context()

		var req CreatePurchaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.LogCreateError(ctx, ErrorCodeValidationError, "invalid request payload", map[string]interface{}{
				"request_id": requestID,
				"error":      err.Error(),
			})
			writeErrorResponse(w, http.StatusBadRequest, ErrorCodeValidationError, "invalid request payload", requestID)
			return
		}

		// Validation
		validationErr := validateCreatePurchaseRequest(req)
		if validationErr != nil {
			logger.LogCreateError(ctx, validationErr.Code, validationErr.Message, map[string]interface{}{
				"request_id": requestID,
			})
			writeErrorResponse(w, StatusForErrorCode(validationErr.Code), validationErr.Code, validationErr.Message, requestID)
			return
		}

		// Create purchase
		purchase, err := service.CreatePurchase(ctx, req.Description, req.TransactionDate, req.AmountUsd)
		if err != nil {
			logger.LogCreateError(ctx, ErrorCodeValidationError, err.Error(), map[string]interface{}{
				"request_id": requestID,
			})
			writeErrorResponse(w, http.StatusBadRequest, ErrorCodeValidationError, err.Error(), requestID)
			return
		}

		resp := PurchaseResponse{
			ID:              purchase.ID.String(),
			Description:     purchase.Description,
			TransactionDate: purchase.TransactionDate.Format("2006-01-02"),
			AmountUsd:       purchase.AmountUsd(),
			CreatedAt:       purchase.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       purchase.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)

		logger.LogCreate(ctx, purchase.ID.String(), map[string]interface{}{
			"request_id":       requestID,
			"description":      purchase.Description,
			"transaction_date": purchase.TransactionDate.Format("2006-01-02"),
			"amount_usd":       purchase.AmountUsd(),
		})
	}
}

// @Summary Get a purchase
// @Description Retrieve a purchase by ID. Optionally provide a currency code to convert the amount using historical exchange rates from the transaction date or earlier (within 6 months).
// @Tags purchases
// @Accept json
// @Produce json
// @Param id path string true "Purchase ID (UUID)"
// @Param currency query string false "ISO 4217 currency code for conversion (e.g., EUR, GBP, JPY)"
// @Param X-Request-ID header string false "Request ID for tracing"
// @Success 200 {object} PurchaseResponse
// @Failure 400 {object} ErrorResponse "Invalid currency, invalid UUID, or rate not found"
// @Failure 404 {object} ErrorResponse "Purchase not found"
// @Router /purchases/{id} [get]
func getPurchaseHandler(service *app.PurchaseService, logger *Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := getOrGenerateRequestID(r)
		ctx := r.Context()

		id := chi.URLParam(r, "id")
		currency := strings.TrimSpace(r.URL.Query().Get("currency"))
		// Normalize currency to uppercase
		currency = strings.ToUpper(currency)

		// Validate UUID
		_, err := uuid.Parse(id)
		if err != nil {
			logger.LogRetrieveError(ctx, id, ErrorCodeValidationError, "invalid purchase ID format", map[string]interface{}{
				"request_id": requestID,
			})
			writeErrorResponse(w, http.StatusBadRequest, ErrorCodeValidationError, "invalid purchase ID format", requestID)
			return
		}

		// If no currency, just retrieve purchase (T020-T022 - US2)
		if currency == "" {
			purchase, found, err := service.GetPurchase(ctx, id)

			if err != nil {
				logger.LogRetrieveError(ctx, id, ErrorCodeValidationError, err.Error(), map[string]interface{}{
					"request_id": requestID,
				})
				writeErrorResponse(w, http.StatusBadRequest, ErrorCodeValidationError, err.Error(), requestID)
				return
			}
			if !found {
				logger.LogRetrieveError(ctx, id, ErrorCodeNotFound, "Purchase with ID "+id+" not found", map[string]interface{}{
					"request_id": requestID,
				})
				writeErrorResponse(w, http.StatusNotFound, ErrorCodeNotFound, "Purchase with ID "+id+" not found", requestID)
				return
			}

			resp := PurchaseResponse{
				ID:              purchase.ID.String(),
				Description:     purchase.Description,
				TransactionDate: purchase.TransactionDate.Format("2006-01-02"),
				AmountUsd:       purchase.AmountUsd(),
				CreatedAt:       purchase.CreatedAt.Format("2006-01-02T15:04:05Z"),
				UpdatedAt:       purchase.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-ID", requestID)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)

			logger.LogRetrieve(ctx, id, map[string]interface{}{
				"request_id": requestID,
			})
			return
		}

		// Validate currency is 3-letter ISO 4217 code (T024)
		if !isValidCurrencyCode(currency) {
			logger.LogConversionError(ctx, id, currency, ErrorCodeValidationError, "Invalid currency code", map[string]interface{}{
				"request_id": requestID,
			})
			writeErrorResponse(w, http.StatusBadRequest, ErrorCodeValidationError, "Invalid currency code: "+currency+". Expected ISO 4217 3-letter code", requestID)
			return
		}

		// With currency conversion (US3) - T023-T027
		purchase, conversion, found, err := service.GetPurchaseWithConversion(ctx, id, currency)
		if err != nil {
			// Check if it's a rate not found error
			if strings.Contains(err.Error(), "no valid rate") || strings.Contains(err.Error(), "no exchange rate") {
				logger.LogConversionError(ctx, id, currency, ErrorCodeRateNotFound, err.Error(), map[string]interface{}{
					"request_id": requestID,
				})
				writeErrorResponse(w, http.StatusBadRequest, ErrorCodeRateNotFound, "No exchange rate available for "+currency+" on or before "+purchase.TransactionDate.Format("2006-01-02"), requestID)
				return
			}
			logger.LogConversionError(ctx, id, currency, ErrorCodeValidationError, err.Error(), map[string]interface{}{
				"request_id": requestID,
			})
			writeErrorResponse(w, http.StatusBadRequest, ErrorCodeValidationError, err.Error(), requestID)
			return
		}

		if !found {
			logger.LogRetrieveError(ctx, id, ErrorCodeNotFound, "Purchase with ID "+id+" not found", map[string]interface{}{
				"request_id": requestID,
			})
			writeErrorResponse(w, http.StatusNotFound, ErrorCodeNotFound, "Purchase with ID "+id+" not found", requestID)
			return
		}

		resp := PurchaseResponse{
			ID:              purchase.ID.String(),
			Description:     purchase.Description,
			TransactionDate: purchase.TransactionDate.Format("2006-01-02"),
			AmountUsd:       purchase.AmountUsd(),
			CreatedAt:       purchase.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       purchase.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}

		if conversion != nil {
			resp.ConvertedCurrency = conversion.Currency
			resp.RateDate = conversion.RateDate
			resp.Rate = conversion.Rate
			resp.ConvertedAmount = conversion.Amount
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)

		logger.LogConversion(ctx, id, currency, conversion.Amount, map[string]interface{}{
			"request_id": requestID,
		})
	}
}

func getOrGenerateRequestID(r *http.Request) string {
	if rid := r.Header.Get("X-Request-ID"); rid != "" {
		return rid
	}
	return uuid.New().String()
}

func validateCreatePurchaseRequest(req CreatePurchaseRequest) *ValidationError {
	if req.Description == "" {
		return &ValidationError{Code: ErrorCodeMissingField, Message: "Missing required field: description"}
	}
	if len(req.Description) > 50 {
		return &ValidationError{Code: ErrorCodeDescriptionTooLong, Message: "Description exceeds 50 character limit"}
	}
	if req.TransactionDate == "" {
		return &ValidationError{Code: ErrorCodeMissingField, Message: "Missing required field: transactionDate"}
	}

	// Validate date format and check if it's in the future (FR-003, Business Rule)
	date, err := parseDate(req.TransactionDate)
	if err != nil {
		return &ValidationError{Code: ErrorCodeInvalidDate, Message: "Invalid date: " + err.Error()}
	}
	if isInTheFuture(date) {
		return &ValidationError{Code: ErrorCodeInvalidDate, Message: "Purchase date cannot be in the future"}
	}

	if req.AmountUsd == "" {
		return &ValidationError{Code: ErrorCodeMissingField, Message: "Missing required field: amountUsd"}
	}

	// Validate amount format and positivity (FR-004, Business Rule)
	if err := validateAmount(req.AmountUsd); err != nil {
		return err
	}

	return nil
}

// isInTheFuture checks if a date is in the future (requirement from spec)
func isInTheFuture(date time.Time) bool {
	return date.After(time.Now())
}

// validateAmount checks if amount is valid positive number (FR-004, Business Rule)
func validateAmount(amountStr string) *ValidationError {
	d, err := decimal.NewFromString(amountStr)
	if err != nil {
		return &ValidationError{Code: ErrorCodeValidationError, Message: "Amount must be a valid numeric value"}
	}
	if d.Sign() <= 0 {
		return &ValidationError{Code: ErrorCodeNegativeAmount, Message: "Amount must be a positive number"}
	}
	return nil
}

type ValidationError struct {
	Code    string
	Message string
}

func writeErrorResponse(w http.ResponseWriter, status int, code, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(status)

	errorResp := map[string]interface{}{
		"code":      code,
		"message":   message,
		"timestamp": getCurrentTimestamp(),
	}
	json.NewEncoder(w).Encode(errorResp)
}

func getCurrentTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func isValidCurrencyCode(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, ch := range currency {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

// parseDate parses a date string in ISO 8601 format (YYYY-MM-DD)
func parseDate(value string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, errDateInvalid
}

// @Summary Liveness probe
// @Description Check if the API is running and responding to requests
// @Tags health
// @Produce json
// @Param X-Request-ID header string false "Request ID for tracing"
// @Success 200 {object} HealthResponse
// @Router /health [get]
// healthHandler returns liveness probe response (T037)
// GET /health - returns { status: "ok" } with X-Request-ID if provided
func healthHandler(logger *Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := getOrGenerateRequestID(r)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}

// @Summary Readiness probe
// @Description Check if the API is ready to handle requests. Verifies database connectivity.
// @Tags health
// @Produce json
// @Param X-Request-ID header string false "Request ID for tracing"
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse "Service not ready - database unavailable"
// @Router /health/ready [get]
// readinessHandler returns readiness probe response (T037)
// GET /health/ready - checks database connectivity and returns { status: "ok" }
func readinessHandler(pool *pgxpool.Pool, logger *Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := getOrGenerateRequestID(r)
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// If pool is provided, check database connectivity
		if pool != nil {
			if err := pool.Ping(ctx); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Request-ID", requestID)
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{
					"status": "not ready",
					"error":  "database not available",
				})
				logger.Error("readiness check failed - database unavailable", map[string]interface{}{
					"request_id": requestID,
					"error":      err.Error(),
				})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}
