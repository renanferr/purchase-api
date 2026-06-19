-- name: CreateExchangeRate :one
INSERT INTO exchange_rates (currency, rate_date, rate, created_at)
VALUES ($1, $2, $3, NOW())
RETURNING currency, rate_date, rate, created_at;

-- name: GetLatestExchangeRateBeforeDate :one
SELECT currency, rate_date, rate
FROM exchange_rates
WHERE currency = $1
  AND rate_date <= $2
ORDER BY rate_date DESC
LIMIT 1;
