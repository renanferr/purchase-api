package domain

import (
	"fmt"
	"time"
)

// ValidationError represents a validation failure with field and message
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// RateNotFoundError indicates an exchange rate could not be found
type RateNotFoundError struct {
	Currency string
	Date     time.Time
}

func (e *RateNotFoundError) Error() string {
	return fmt.Sprintf("no exchange rate found for %s on or before %s", e.Currency, e.Date.Format("2006-01-02"))
}

// IsRateNotFoundError checks if an error is a RateNotFoundError
func IsRateNotFoundError(err error) bool {
	_, ok := err.(*RateNotFoundError)
	return ok
}

// PurchaseNotFoundError indicates a purchase was not found
type PurchaseNotFoundError struct {
	PurchaseID string
}

func (e *PurchaseNotFoundError) Error() string {
	return fmt.Sprintf("purchase with ID %s not found", e.PurchaseID)
}

// IsPurchaseNotFoundError checks if an error is a PurchaseNotFoundError
func IsPurchaseNotFoundError(err error) bool {
	_, ok := err.(*PurchaseNotFoundError)
	return ok
}

// InvalidCurrencyCodeError indicates an invalid currency code
type InvalidCurrencyCodeError struct {
	Code string
}

func (e *InvalidCurrencyCodeError) Error() string {
	return fmt.Sprintf("invalid currency code: %s. Expected ISO 4217 3-letter code", e.Code)
}

// IsInvalidCurrencyCodeError checks if an error is an InvalidCurrencyCodeError
func IsInvalidCurrencyCodeError(err error) bool {
	_, ok := err.(*InvalidCurrencyCodeError)
	return ok
}
