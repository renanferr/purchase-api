package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// ErrorResponseTestSuite defines the test suite for error handling
type ErrorResponseTestSuite struct {
	suite.Suite
}

// TestErrorResponse_TableDriven validates error codes and HTTP status mappings
func (suite *ErrorResponseTestSuite) TestErrorResponse_TableDriven() {
	testCases := []struct {
		name           string
		errorCode      string
		expectedStatus int
		expectedExists bool
	}{
		{
			name:           "validation_error_400",
			errorCode:      ErrorCodeValidationError,
			expectedStatus: http.StatusBadRequest,
			expectedExists: true,
		},
		{
			name:           "invalid_date_400",
			errorCode:      ErrorCodeInvalidDate,
			expectedStatus: http.StatusBadRequest,
			expectedExists: true,
		},
		{
			name:           "negative_amount_400",
			errorCode:      ErrorCodeNegativeAmount,
			expectedStatus: http.StatusBadRequest,
			expectedExists: true,
		},
		{
			name:           "description_too_long_400",
			errorCode:      ErrorCodeDescriptionTooLong,
			expectedStatus: http.StatusBadRequest,
			expectedExists: true,
		},
		{
			name:           "missing_field_400",
			errorCode:      ErrorCodeMissingField,
			expectedStatus: http.StatusBadRequest,
			expectedExists: true,
		},
		{
			name:           "rate_not_found_400",
			errorCode:      ErrorCodeRateNotFound,
			expectedStatus: http.StatusBadRequest,
			expectedExists: true,
		},
		{
			name:           "not_found_404",
			errorCode:      ErrorCodeNotFound,
			expectedStatus: http.StatusNotFound,
			expectedExists: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.NotEmpty(tc.errorCode)
			status := StatusForErrorCode(tc.errorCode)
			suite.Equal(tc.expectedStatus, status)
		})
	}
}

// TestErrorResponse_FormatValidation validates response structure
func (suite *ErrorResponseTestSuite) TestErrorResponse_FormatValidation() {
	testCases := []struct {
		name    string
		code    string
		message string
	}{
		{"validation_error", ErrorCodeValidationError, "invalid input"},
		{"invalid_date", ErrorCodeInvalidDate, "date in future"},
		{"negative_amount", ErrorCodeNegativeAmount, "amount negative"},
		{"too_long", ErrorCodeDescriptionTooLong, "description exceeded"},
		{"missing_field", ErrorCodeMissingField, "required field missing"},
		{"rate_not_found", ErrorCodeRateNotFound, "no rate available"},
		{"not_found", ErrorCodeNotFound, "purchase not found"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			resp := NewErrorResponse(tc.code, tc.message)

			suite.NotNil(resp)
			suite.Equal(tc.code, resp.Code)
			suite.Equal(tc.message, resp.Message)
			suite.NotEmpty(resp.Timestamp)

			// Verify ISO 8601 format
			_, err := time.Parse("2006-01-02T15:04:05Z", resp.Timestamp)
			suite.NoError(err)
		})
	}
}

// TestErrorCode_Constants validates all error codes are defined
func (suite *ErrorResponseTestSuite) TestErrorCode_Constants() {
	allCodes := []string{
		ErrorCodeValidationError,
		ErrorCodeInvalidDate,
		ErrorCodeNegativeAmount,
		ErrorCodeDescriptionTooLong,
		ErrorCodeMissingField,
		ErrorCodeRateNotFound,
		ErrorCodeNotFound,
	}

	for _, code := range allCodes {
		suite.Run(code, func() {
			suite.NotEmpty(code, "error code should not be empty")
		})
	}
}

// TestHTTPStatusMapping_Completeness validates all error codes are mapped
func (suite *ErrorResponseTestSuite) TestHTTPStatusMapping_Completeness() {
	statusMap := make(map[int][]string)
	allCodes := []string{
		ErrorCodeValidationError,
		ErrorCodeInvalidDate,
		ErrorCodeNegativeAmount,
		ErrorCodeDescriptionTooLong,
		ErrorCodeMissingField,
		ErrorCodeRateNotFound,
		ErrorCodeNotFound,
	}

	for _, code := range allCodes {
		status := StatusForErrorCode(code)
		statusMap[status] = append(statusMap[status], code)
	}

	// Should have at least 2 different status codes
	suite.GreaterOrEqual(len(statusMap), 2)

	// All 400 errors
	suite.NotEmpty(statusMap[http.StatusBadRequest])
	// At least one 404 error
	suite.NotEmpty(statusMap[http.StatusNotFound])

	// 400 errors should include validation-related codes
	badRequestCodes := statusMap[http.StatusBadRequest]
	suite.Contains(badRequestCodes, ErrorCodeValidationError)
	suite.Contains(badRequestCodes, ErrorCodeRateNotFound)

	// 404 errors should include not found
	notFoundCodes := statusMap[http.StatusNotFound]
	suite.Contains(notFoundCodes, ErrorCodeNotFound)
}

// Run the test suite
func TestErrorResponseSuite(t *testing.T) {
	suite.Run(t, new(ErrorResponseTestSuite))
}
