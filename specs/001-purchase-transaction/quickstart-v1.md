# Quickstart — Local Development

## Prerequisites

- Go 1.25+ (install via [Go installer](https://golang.org/dl) or `gvm`/`asdf`)
- `sqlc` installed (https://sqlc.dev) — install via `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- Docker and Docker Compose (for PostgreSQL in local dev/CI)
- `curl` or Postman (for API testing)

## Environment Variables

- `TREASURY_PROVIDER` — Adapter selection: `real` (default, fetch from Treasury API) or `mock` (testing only, deterministic test data)
- `LOG_LEVEL` — Logging level: `debug`, `info`, `warn`, `error` (default: `info`)
- `DB_URL` — PostgreSQL connection string (default: `postgres://postgres:postgres@localhost:5432/purchases_db`)
- `API_PORT` — HTTP server port (default: `8080`)
- `REQUEST_TIMEOUT` — Request context timeout in seconds (default: `30`)

## Local Development Quickstart

### 1. Start PostgreSQL and Create Database

```powershell
# Start PostgreSQL container
docker compose -f docker-compose.yml up -d

# Create database
docker exec -it postgres_container psql -U postgres -c "CREATE DATABASE purchases_db;"
```

### 2. Generate DB Code and Run Migrations

```powershell
cd src/purchase-api

# Generate type-safe database code from SQL queries
sqlc generate

# Run migrations (if using golang-migrate)
migrate -path db/migrations -database "postgres://postgres:postgres@localhost:5432/purchases_db" up

# Or use custom migration script
./scripts/migrate up
```

### 3. (Optional) Configure Treasury Provider

```powershell
# By default, the application uses the real Treasury API (TREASURY_PROVIDER=real)
# The adapter will fetch and cache rates on demand during GET requests

# For testing with mock data (deterministic rates):
export TREASURY_PROVIDER=mock
go run cmd/purchase-api/main.go
# The mock adapter returns deterministic test data (0.92 EUR/USD on 2026-06-12)
# See internal/adapters/treasury/sample_provider.go (contains the mock implementation)
```

### 4. Build and Run Service

```powershell
# Build
go build ./cmd/purchase-api

# Run with defaults
./purchase-api

# Run with custom environment
$env:LOG_LEVEL = "debug"
$env:TREASURY_PROVIDER = "mock"
./purchase-api
```

### 5. Test the API

#### Example 1: Create a Purchase

```bash
curl -X POST http://localhost:8080/purchases \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: $(uuidgen)" \
  -d '{
    "description": "Office Supplies",
    "transactionDate": "2026-06-15",
    "amountUsd": "1500.00"
  }'

# Response (201 Created):
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Office Supplies",
  "transactionDate": "2026-06-15",
  "amountUsd": "1500.00",
  "createdAt": "2026-06-18T14:30:00Z",
  "updatedAt": "2026-06-18T14:30:00Z"
}
```

#### Example 2: Retrieve Purchase (No Conversion)

```bash
PURCHASE_ID="550e8400-e29b-41d4-a716-446655440000"

curl -X GET "http://localhost:8080/purchases/$PURCHASE_ID" \
  -H "X-Request-ID: $(uuidgen)"

# Response (200 OK):
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Office Supplies",
  "transactionDate": "2026-06-15",
  "amountUsd": "1500.00",
  "createdAt": "2026-06-18T14:30:00Z",
  "updatedAt": "2026-06-18T14:30:00Z"
}
```

#### Example 3: Retrieve Purchase With Currency Conversion

```bash
PURCHASE_ID="550e8400-e29b-41d4-a716-446655440000"

curl -X GET "http://localhost:8080/purchases/$PURCHASE_ID?currency=EUR" \
  -H "X-Request-ID: $(uuidgen)"

# Response (200 OK):
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Office Supplies",
  "transactionDate": "2026-06-15",
  "amountUsd": "1500.00",
  "exchangeRate": {
    "currency": "EUR",
    "rateDate": "2026-06-12",
    "rate": "0.92"
  },
  "convertedAmount": "1380.00",
  "createdAt": "2026-06-18T14:30:00Z",
  "updatedAt": "2026-06-18T14:30:00Z"
}
```

#### Example 4: Error Case — No Rate Available

```bash
PURCHASE_ID="550e8400-e29b-41d4-a716-446655440000"

# Assume purchase date is very old or currency has no rates
curl -X GET "http://localhost:8080/purchases/$PURCHASE_ID?currency=XXX" \
  -H "X-Request-ID: $(uuidgen)"

# Response (400 Bad Request):
{
  "code": "RATE_NOT_FOUND",
  "message": "No exchange rate available for XXX on or before 2026-06-15",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

#### Example 5: Validation Error — Description Too Long

```bash
curl -X POST http://localhost:8080/purchases \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: $(uuidgen)" \
  -d '{
    "description": "This is a very long description that exceeds the maximum of fifty characters allowed",
    "transactionDate": "2026-06-15",
    "amountUsd": "1500.00"
  }'

# Response (400 Bad Request):
{
  "code": "DESCRIPTION_TOO_LONG",
  "message": "Description exceeds 50 character limit",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

### 6. Run Tests

```powershell
cd src/purchase-api

# Run all tests
go test ./... -v

# Run with coverage
go test ./... -v -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run tests in a specific package
go test ./internal/domain -v
go test ./internal/app -v

# Run integration tests (requires PostgreSQL running)
go test ./tests/integration/... -v
```

### 7. View Logs

The service logs to stdout in structured JSON format:

```powershell
# Capture logs from running service
./purchase-api 2>&1 | jq '.'
```

Example log entry:

```json
{
  "timestamp": "2026-06-18T14:30:00Z",
  "level": "info",
  "component": "http_handler",
  "message": "purchase created successfully",
  "context": {
    "purchase_id": "550e8400-e29b-41d4-a716-446655440000",
    "description": "Office Supplies",
    "amount_usd_cents": 150000
  }
}
```

## Troubleshooting

### `sqlc: command not found`
Ensure `sqlc` is installed: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

### PostgreSQL connection refused
Check that Docker container is running: `docker ps | grep postgres`
Verify connection string matches container details.

### Migration errors
Run migrations with verbose output: `migrate -path db/migrations -database "..." up -verbose`

### API returns 500
Check logs for errors: `./purchase-api 2>&1 | jq '.level == "error"'`

## Project Structure

After local setup, the project structure is:

```
src/purchase-api/
├── cmd/purchase-api/
│   └── main.go                 # Application entrypoint
├── internal/
│   ├── api/
│   │   ├── handlers.go         # HTTP request/response handlers
│   │   └── dto.go              # Request/response data transfer objects
│   ├── app/
│   │   └── purchase_service.go # Business logic / use cases
│   ├── domain/
│   │   ├── purchase.go         # Purchase entity
│   │   ├── money.go            # Money/decimal handling
│   │   └── exchange_rate.go    # ExchangeRate entity
│   ├── adapters/
│   │   ├── db/postgres.go      # PostgreSQL repository adapter
│   │   └── treasury/           # Treasury API adapters
│   └── ports/
│       └── ports.go            # Port/interface definitions
├── db/
│   ├── migrations/             # SQL migration files
│   ├── queries/                # SQLC query definitions
│   ├── sqlc/                   # Generated code (after sqlc generate)
│   └── sqlc.yaml               # SQLC configuration
├── tests/
│   ├── integration/            # Integration tests (with PostgreSQL)
│   └── contract/               # Contract/adapter tests
├── docker-compose.yml          # PostgreSQL container definition
├── go.mod / go.sum             # Go module definition
└── Makefile or scripts/        # Build/run convenience scripts
```

## Next Steps

- Review [API contract](contracts/purchases-openapi-v1.yaml) for full OpenAPI specification
- Read [Data Model](data-model-v1.md) for schema and validation details
- Check [Implementation Plan](plan.md) for architecture and design decisions
- Run integration tests to verify full API workflow with PostgreSQL
