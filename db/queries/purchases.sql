-- name: CreatePurchase :one
INSERT INTO purchases (id, description, transaction_date, amount_usd_cents, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
RETURNING id, description, transaction_date, amount_usd_cents, created_at, updated_at;

-- name: GetPurchaseByID :one
SELECT id, description, transaction_date, amount_usd_cents, created_at, updated_at
FROM purchases
WHERE id = $1;

-- name: GetRateForDate :one
SELECT currency, rate_date, rate
FROM exchange_rates
WHERE currency = $1 
  AND rate_date <= $2
  AND rate_date >= ($2::date - interval '6 months')
ORDER BY rate_date DESC
LIMIT 1;
