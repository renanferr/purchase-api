# Data Model (Updated: 2026-06-18)

## Entities

- **Purchase**
  - `id` (UUID, PK) — Unique identifier
  - `description` (VARCHAR(50), NOT NULL) — Purchase description; max 50 characters
  - `transaction_date` (DATE, NOT NULL) — ISO 8601 date only (YYYY-MM-DD); must not be in the future
  - `amount_usd_cents` (BIGINT, NOT NULL) — Amount in USD cents (integer) for exact precision; must be positive (> 0)
  - `created_at` (TIMESTAMP, DEFAULT NOW(), NOT NULL) — Audit timestamp: when purchase was created
  - `updated_at` (TIMESTAMP, DEFAULT NOW() ON UPDATE, NOT NULL) — Audit timestamp: last modification time

- **ExchangeRate** (local cache)
  - `currency` (CHAR(3), NOT NULL) — ISO 4217 currency code (uppercase); e.g., EUR, GBP, JPY, CNY
  - `rate_date` (DATE, NOT NULL) — Date the rate was published (ISO 8601)
  - `rate` (NUMERIC(18,6), NOT NULL) — Exchange rate multiplier (convert USD → target currency); e.g., 0.920000
  - `created_at` (TIMESTAMP, DEFAULT NOW(), NOT NULL) — When this rate record was inserted
  - (PK: `currency + rate_date`) — Composite primary key ensures one rate per currency per date

## Relationships

- No foreign key between Purchase and ExchangeRate — ExchangeRate is historical/reference data queried by date and currency at lookup time.

## Indexing Strategy

- **purchases**:
  - PK Index: `purchases_pk ON purchases(id)`
  - Optional: Index on `created_at` for time-series queries
  
- **exchange_rates**:
  - Composite PK Index: `exchange_rates_pk ON exchange_rates(currency, rate_date DESC)` — Enables efficient latest-rate-before-date queries within 6-month window
  - Optional: Index on `currency` for single-currency lookups

## Validation Rules

- `description`: ≤ 50 characters (UTF-8 byte length)
- `transaction_date`: Valid ISO 8601 date (YYYY-MM-DD); cannot be in the future (compared to current UTC date)
- `amount_usd_cents`: Positive integer (> 0); arbitrary precision in storage (BIGINT supports ±9,223,372,036,854,775,807 cents ≈ ±92 billion USD)
- `currency`: ISO 4217 3-letter code (case-insensitive on input, stored uppercase); examples: EUR, GBP, USD, JPY, CHF
- `rate`: Positive decimal (numeric with 6 fractional digits); typically 0.001 to 99.999999

## JSON Representation (API)

**Monetary amounts in JSON are STRINGS** (not numbers) to preserve financial precision and avoid floating-point errors:
- Request: `{ "amountUsd": "1234.56" }`
- Response: `{ "amountUsd": "1234.56", "convertedAmount": "1136.79" }`

**Dates in JSON are ISO 8601 dates** (no time component):
- Request: `{ "transactionDate": "2026-06-15" }`
- Response: `{ "transactionDate": "2026-06-15", "rateDate": "2026-06-12" }`

**Currency codes are 3-letter uppercase**:
- Query parameter: `?currency=eur` → validated/normalized to `EUR`
- Response: `{ "currency": "EUR", ... }`

## sqlc / SQL Considerations

Provide SQL files under `db/queries/` with queries that `sqlc` will compile into type-safe Go functions.

### Example: Create Purchase

```sql
-- name: CreatePurchase :one
INSERT INTO purchases (id, description, transaction_date, amount_usd_cents, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
RETURNING id, description, transaction_date, amount_usd_cents, created_at, updated_at;
```

### Example: Get Purchase by ID

```sql
-- name: GetPurchaseByID :one
SELECT id, description, transaction_date, amount_usd_cents, created_at, updated_at
FROM purchases
WHERE id = $1;
```

### Example: Find Latest Rate Within 6 Months

```sql
-- name: GetRateForDate :one
SELECT currency, rate_date, rate
FROM exchange_rates
WHERE currency = $1 
  AND rate_date <= $2
  AND rate_date >= ($2::date - interval '6 months')
ORDER BY rate_date DESC
LIMIT 1;
```

## Migration Schema

### `000001_create_purchases_table.up.sql`

```sql
CREATE TABLE purchases (
  id UUID PRIMARY KEY,
  description VARCHAR(50) NOT NULL,
  transaction_date DATE NOT NULL,
  amount_usd_cents BIGINT NOT NULL CHECK (amount_usd_cents > 0),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX purchases_created_at ON purchases(created_at);
```

### `000002_create_exchange_rates_table.up.sql`

```sql
CREATE TABLE exchange_rates (
  currency CHAR(3) NOT NULL,
  rate_date DATE NOT NULL,
  rate NUMERIC(18,6) NOT NULL CHECK (rate > 0),
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (currency, rate_date)
);

CREATE INDEX exchange_rates_currency_date ON exchange_rates(currency, rate_date DESC);
```

## Amount Precision & Rounding

- **Input rounding**: Client-provided `amountUsd` (e.g., "1234.567") is rounded to nearest cent using half-away-from-zero (e.g., "1234.57") and stored as integer cents.
- **Conversion rounding**: `convertedAmount = amountUsd * rate`, then rounded to 2 decimals using half-away-from-zero (same algorithm).
- **Storage**: All amounts stored as integer cents (BIGINT) in database; no floating-point types used.
- **JSON transport**: Amounts returned as strings (e.g., "1234.57") to preserve precision and avoid JS/JSON number precision loss.

## Audit & Compliance

- **created_at**: Set once at insert time; never modified.
- **updated_at**: Set at insert time; updated on any row modification (currently only insert is specified, but field supports future updates if needed).
- **Hard delete**: Purchases are hard-deleted on user request (no soft-delete/logical deletion).
- **Indefinite retention**: No automatic archival or time-based deletion of historical data.
