# Data Model

## Entities

- Purchase
  - `id` (UUID, PK)
  - `description` (string, max 50)
  - `transaction_date` (date)
  - `amount_usd_cents` (bigint) — stored in USD cents as integer to guarantee exactness

- ExchangeRate (local cache)
  - `currency` (char(3))
  - `rate_date` (date)
  - `rate` (numeric(18,6)) — multiplier to convert USD → target currency
  - (PK: currency + rate_date)

## Relationships

- No direct FK between Purchase and ExchangeRate — ExchangeRate is historical/time-series data queried by date and currency.

## Indexing

- Index `ExchangeRate(currency, rate_date DESC)` to efficiently find latest rate ≤ purchase date.

## Validation Rules

- `description` ≤ 50 characters
- `transaction_date` must be a valid date (ISO 8601)
- `amount_usd_cents` must be a positive integer

## sqlc / SQL considerations

- Provide SQL files under `db/queries/` (e.g., `purchases.sql`, `exchange_rates.sql`) with queries that `sqlc` will compile into type-safe Go functions.
- Example purchase insert query (pseudocode):

  -- name: CreatePurchase :one
  INSERT INTO purchases (id, description, transaction_date, amount_usd_cents)
  VALUES ($1, $2, $3, $4)
  RETURNING id, description, transaction_date, amount_usd_cents;

- Query to find latest exchange rate ≤ date within 6 months:

  -- name: GetRateForDate :one
  SELECT currency, rate_date, rate FROM exchange_rates
  WHERE currency = $1 AND rate_date <= $2 AND rate_date >= ($2 - interval '6 months')
  ORDER BY rate_date DESC LIMIT 1;

