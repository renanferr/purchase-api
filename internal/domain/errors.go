package domain

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel error constants for specific error conditions
var (
	// ErrDatabaseOperation indicates a database operation failed
	ErrDatabaseOperation = errors.New("database operation failed")
	// ErrUniqueConstraint indicates a unique constraint was violated
	ErrUniqueConstraint = errors.New("unique constraint violation")
	// ErrNoRows indicates no rows were found
	ErrNoRows = errors.New("no rows found")
	// ErrRateNotFound indicates an exchange rate was not found
	ErrRateNotFound = errors.New("exchange rate not found")
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
// Wraps the ErrRateNotFound sentinel
type RateNotFoundError struct {
	Currency string
	Date     time.Time
}

func (e *RateNotFoundError) Error() string {
	return fmt.Sprintf("no exchange rate found for %s on or before %s", e.Currency, e.Date.Format("2006-01-02"))
}

func (e *RateNotFoundError) Unwrap() error {
	return ErrRateNotFound
}

// PurchaseNotFoundError indicates a purchase was not found
type PurchaseNotFoundError struct {
	PurchaseID string
}

func (e *PurchaseNotFoundError) Error() string {
	return fmt.Sprintf("purchase with ID %s not found", e.PurchaseID)
}

// InvalidCurrencyCodeError indicates an invalid currency code
type InvalidCurrencyCodeError struct {
	Code string
}

func (e *InvalidCurrencyCodeError) Error() string {
	return fmt.Sprintf("invalid currency code: %s. Expected ISO 4217 3-letter code", e.Code)
}

// DatabaseErrorDetails wraps database operation errors with context
// Returns ErrDatabaseOperation when unwrapped
type DatabaseErrorDetails struct {
	Operation string // "create", "get", "update", etc.
	Table     string // "purchases", "exchange_rates", etc.
	Cause     error  // underlying error from database driver
}

func (e *DatabaseErrorDetails) Error() string {
	return fmt.Sprintf("database %s operation on %s failed: %v", e.Operation, e.Table, e.Cause)
}

func (e *DatabaseErrorDetails) Unwrap() error {
	return ErrDatabaseOperation
}

// UniqueConstraintError wraps a unique constraint violation error
// Returns ErrUniqueConstraint when unwrapped
type UniqueConstraintError struct {
	Cause error // underlying database error
}

func (e *UniqueConstraintError) Error() string {
	return fmt.Sprintf("unique constraint violation: %v", e.Cause)
}

func (e *UniqueConstraintError) Unwrap() error {
	return ErrUniqueConstraint
}

// NoRateError indicates no exchange rate was found for the requested currency
// Wraps the ErrRateNotFound sentinel
type NoRateError struct {
	Currency string
	Date     time.Time
}

func (e *NoRateError) Error() string {
	return fmt.Sprintf("no exchange rate available for %s on or before %s", e.Currency, e.Date.Format("2006-01-02"))
}

func (e *NoRateError) Unwrap() error {
	return ErrRateNotFound
}
