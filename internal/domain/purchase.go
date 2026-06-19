package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Purchase struct {
	ID              uuid.UUID `json:"id"`
	Description     string    `json:"description"`
	TransactionDate time.Time `json:"transactionDate"`
	AmountUsdCents  int64     `json:"-"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (p Purchase) AmountUsd() string {
	return formatCents(p.AmountUsdCents)
}

func NewPurchase(description string, transactionDate time.Time, amountUsdCents int64) (Purchase, error) {
	if len(description) == 0 || len(description) > 50 {
		return Purchase{}, errors.New("description must be 1 to 50 characters")
	}
	if amountUsdCents <= 0 {
		return Purchase{}, errors.New("amountUsd must be positive")
	}
	return Purchase{
		ID:              uuid.New(),
		Description:     description,
		TransactionDate: transactionDate,
		AmountUsdCents:  amountUsdCents,
	}, nil
}

func formatCents(cents int64) string {
	dollars := cents / 100
	remainder := cents % 100
	if remainder < 0 {
		remainder = -remainder
	}
	return fmt.Sprintf("%d.%02d", dollars, remainder)
}
