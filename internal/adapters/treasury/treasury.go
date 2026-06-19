package treasury

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/example/purchase-api/internal/ports"
	"github.com/shopspring/decimal"
)

const defaultTreasuryBaseURL = "https://api.fiscaldata.treasury.gov"

// currencyCodeToName maps ISO 4217 currency codes to Treasury API currency names
var currencyCodeToName = map[string]string{
	"USD": "U.S. Dollar",
	"EUR": "Euro",
	"GBP": "Pound",
	"JPY": "Yen",
	"CAD": "Dollar", // Canadian Dollar
	"AUD": "Dollar", // Australian Dollar
	"BRL": "Real",   // Brazilian Real
	"INR": "Rupee",  // Indian Rupee
	"CHF": "Franc",  // Swiss Franc
	"CNY": "Renminbi",
	"SEK": "Krona",  // Swedish Krona
	"NOK": "Krone",  // Norwegian Krone
	"NZD": "Dollar", // New Zealand Dollar
	"ZAR": "Rand",   // South African Rand
	"HKD": "Dollar", // Hong Kong Dollar
	"SGD": "Dollar", // Singapore Dollar
}

// ExchangeRateProvider fetches the latest Treasury reporting exchange rate for a target currency.
// All rates are expressed as: 1 USD = X target_currency
// For example: 1 USD = 0.87 EUR, 1 USD = 5.254 BRL
type ExchangeRateProvider struct {
	client  *http.Client
	baseURL string
}

func NewExchangeRateProvider(client *http.Client) ports.TreasuryRateProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &ExchangeRateProvider{
		client:  client,
		baseURL: defaultTreasuryBaseURL,
	}
}

type treasuryRateResponse struct {
	Data []struct {
		Currency     string `json:"currency"`
		RecordDate   string `json:"record_date"`
		ExchangeRate string `json:"exchange_rate"`
	} `json:"data"`
}

func (p *ExchangeRateProvider) LatestRateBeforeDate(ctx context.Context, currency string, before time.Time) (decimal.Decimal, string, time.Time, error) {
	// Returns: (rate, currencyCode, rateDate, error)
	// rate = how much target_currency equals 1 USD (e.g., 0.87 for EUR means 1 USD = 0.87 EUR)
	currency = strings.TrimSpace(strings.ToUpper(currency))
	if currency == "" {
		return decimal.Zero, "", time.Time{}, errors.New("currency must be provided")
	}

	// Convert currency code to Treasury API currency name
	treasuryName, ok := currencyCodeToName[currency]
	if !ok {
		return decimal.Zero, "", time.Time{}, fmt.Errorf("unsupported currency code: %s", currency)
	}

	query := url.Values{}
	query.Set("fields", "record_date,currency,exchange_rate")
	query.Set("filter", fmt.Sprintf("currency:eq:%s,record_date:lte:%s", treasuryName, before.Format("2006-01-02")))
	query.Set("sort", "-record_date")
	query.Set("limit", "1")

	endpoint := fmt.Sprintf("%s/services/api/fiscal_service/v1/accounting/od/rates_of_exchange?%s", strings.TrimSuffix(p.baseURL, "/"), query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return decimal.Zero, "", time.Time{}, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return decimal.Zero, "", time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, "", time.Time{}, fmt.Errorf("failed to fetch exchange rate: %s", resp.Status)
	}

	var body treasuryRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return decimal.Zero, "", time.Time{}, err
	}

	if len(body.Data) == 0 {
		return decimal.Zero, "", time.Time{}, fmt.Errorf("no exchange rate available for %s on or before %s", currency, before.Format("2006-01-02"))
	}

	rate, err := decimal.NewFromString(body.Data[0].ExchangeRate)
	if err != nil {
		return decimal.Zero, "", time.Time{}, fmt.Errorf("invalid exchange rate value %q: %w", body.Data[0].ExchangeRate, err)
	}

	rateDate, err := time.Parse("2006-01-02", body.Data[0].RecordDate)
	if err != nil {
		return decimal.Zero, "", time.Time{}, fmt.Errorf("invalid record_date value %q: %w", body.Data[0].RecordDate, err)
	}

	return rate, currency, rateDate, nil
}
