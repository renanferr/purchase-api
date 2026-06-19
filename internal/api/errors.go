package api

import (
	"net/http"
	"time"
)

// Error code constants (C7)
const (
	ErrorCodeValidationError    = "VALIDATION_ERROR"
	ErrorCodeInvalidDate        = "INVALID_DATE"
	ErrorCodeNegativeAmount     = "NEGATIVE_AMOUNT"
	ErrorCodeDescriptionTooLong = "DESCRIPTION_TOO_LONG"
	ErrorCodeMissingField       = "MISSING_FIELD"
	ErrorCodeRateNotFound       = "RATE_NOT_FOUND"
	ErrorCodeNotFound           = "NOT_FOUND"
)

// ErrorResponse represents the standard error response format
// @Description API error with code and message
type ErrorResponse struct {
	Code      string `json:"code" example:"VALIDATION_ERROR"`
	Message   string `json:"message" example:"Amount must be a positive number"`
	Timestamp string `json:"timestamp" example:"2026-06-18T14:30:00Z"`
}

// NewErrorResponse creates a new ErrorResponse with current timestamp
func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// StatusForErrorCode returns the appropriate HTTP status code for an error code
func StatusForErrorCode(code string) int {
	switch code {
	case ErrorCodeValidationError:
		return http.StatusBadRequest
	case ErrorCodeInvalidDate:
		return http.StatusBadRequest
	case ErrorCodeNegativeAmount:
		return http.StatusBadRequest
	case ErrorCodeDescriptionTooLong:
		return http.StatusBadRequest
	case ErrorCodeMissingField:
		return http.StatusBadRequest
	case ErrorCodeRateNotFound:
		return http.StatusBadRequest
	case ErrorCodeNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
