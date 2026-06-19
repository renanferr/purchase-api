# Research: Decisions and Rationale

## Decision: Language and Framework

- Choice: Go 1.21+. Rationale: You requested Go; Go provides a lightweight runtime, fast compilation, strong standard library for HTTP servers, excellent ecosystem for SQL tooling (sqlc, pgx), and straightforward deployment in containers. Go's static typing plus `sqlc` generated code produces a typesafe, maintainable DB layer without an ORM.

## Decision: Numeric and Rounding Strategy

- Use `int64` to store amounts in cents (recommended) or `shopspring/decimal` if arbitrary precision required. Storing as integer cents avoids floating point errors and simplifies DB schemas (`bigint`). Round on input to the nearest cent using half-away-from-zero (round half away) and validate positivity.

## Decision: Persistence

- Choice: PostgreSQL (ACID, transactional safety, robust time-series querying, and indexes). Justification: The application requires strong consistency for financial data, joins and indexable queries for exchange-rate lookups, and reliable transactional semantics. NoSQL solutions (e.g., document stores) add complexity for relational/time-series queries and do not provide the same strong transactional guarantees without additional engineering.

- Local/CI: Use Dockerized PostgreSQL (via Docker Compose or `ory/dockertest`) to ensure parity with production. Use `golang-migrate` for schema migrations. Use `sqlc` to generate type-safe DB access code from SQL queries for performance, clarity, and testability.

## Decision: Treasury Rates Adapter

- Design: Define a `TreasuryRateProvider` port (Go interface) that returns the latest rate ≤ the requested date within 6 months. Implement an adapter that can pull/bulk-download the Treasury dataset into a local `exchange_rates` table and provide SQL-backed lookups. The adapter will expose `GetRateForDate(ctx, currency, date) (Rate, error)` and return a specific error when no rate is found.

## Decision: Rate selection & caching

- **Selection**: Prefer the most recent rate published on or before the purchase date (i.e., `max(rateDate)` where `rateDate ≤ purchaseDate` and `rateDate ≥ purchaseDate - 6 months`).
- **Caching**: None required for initial modest scale (<100 purchases/day, <10 concurrent users). Direct database lookups via indexed queries are sufficient. Caching can be added later if performance targets change.
- **Failure handling**: If no rate exists within 6-month window, return HTTP 400 error-only response. No fallback to older rates, cached rates, or degraded responses. Conversion is all-or-nothing to maintain financial precision and client clarity.
- **Error signal**: Return structured error: `{ code: "RATE_NOT_FOUND", message: "No exchange rate available for {currency} on or before {purchaseDate}", timestamp: "ISO-8601" }`

## Testing Strategy

- Unit tests: domain invariants, validation, rounding behavior.
- Integration tests: persistence and API flows using `dockertest` or Docker Compose with a real PostgreSQL instance; run the full database-backed workflow in CI.
- Contract tests: mock Treasury adapter (TREASURY_PROVIDER=mock) to assert expected behavior; include a small sample dataset for deterministic tests.

## Migration & Schema

- Use `golang-migrate` for schema evolution. Keep schema small: `purchases` and `exchange_rates` tables with appropriate indexes (index on currency + rate_date).

## Observability & Security

- **Structured logging**: All application events logged to stdout in JSON format with fields: `timestamp`, `level` (info/warn/error), `component`, `message`, `context` (relevant IDs/values). Example:
  ```json
  {
    "timestamp": "2026-06-18T14:30:00Z",
    "level": "error",
    "component": "purchase_service",
    "message": "rate lookup failed",
    "context": {"purchase_id": "550e8400-e29b-41d4-a716-446655440000", "currency": "EUR", "reason": "no_rate_within_window"}
  }
  ```
- **Error responses**: Standardized JSON format: `{ code: "ERROR_CODE", message: "Human-readable description", timestamp: "ISO-8601" }`
- **Request tracing** (optional): Accept `X-Request-ID` header or generate UUID if absent. Include request ID in error responses and log all related events for correlation.
- **Health endpoints**: Expose `/health` (liveness) and `/health/ready` (readiness).
- **TLS and secrets**: TLS and secret management are deployment concerns; document recommended Docker/K8s secrets usage.
- **Data retention**: Indefinite retention of all purchases and exchange rates. Support explicit deletion on user request only. No time-based archival, no soft-delete required initially.
