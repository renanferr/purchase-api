package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type ExchangeRate struct {
	Currency  string
	RateDate  time.Time
	Rate      decimal.Decimal
	CreatedAt time.Time
}

func (r ExchangeRate) RateString() string {
	return r.Rate.StringFixed(6)
}
