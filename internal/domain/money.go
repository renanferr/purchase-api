package domain

import (
	"errors"

	"github.com/shopspring/decimal"
)

type Money struct {
	Cents int64
}

func ParseMoney(value string) (Money, error) {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return Money{}, errors.New("amountUsd must be a numeric value")
	}
	d = d.Round(2)
	cents := d.Mul(decimal.NewFromInt(100))
	if !cents.IsInteger() {
		return Money{}, errors.New("amountUsd must round cleanly to cents")
	}
	if cents.Sign() <= 0 {
		return Money{}, errors.New("amountUsd must be positive")
	}
	return Money{Cents: cents.IntPart()}, nil
}

func (m Money) String() string {
	d := decimal.NewFromInt(m.Cents).Div(decimal.NewFromInt(100))
	return d.StringFixed(2)
}
