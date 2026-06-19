CREATE TABLE IF NOT EXISTS exchange_rates (
  currency CHAR(3) NOT NULL,
  rate_date DATE NOT NULL,
  rate NUMERIC(18,6) NOT NULL CHECK (rate > 0),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (currency, rate_date)
);

CREATE INDEX exchange_rates_currency_date ON exchange_rates(currency, rate_date DESC);
