# Quickstart — Local Development & API

## Prerequisites

- Go 1.21+ (install via Go installer or `gvm`/`asdf`)
- `sqlc` installed (https://sqlc.dev)
- Docker and Docker Compose (for PostgreSQL in local dev/CI)
- `curl` (for API testing)

## Local Development Setup

### 1. Start PostgreSQL and Dependencies

```bash
docker compose -f docker-compose.yml up -d
```

### 2. Generate DB Code and Run Migrations

```bash
sqlc generate
go run ./cmd/migrate -dir up
```

### 3. Build and Run Service

```bash
go build ./cmd/purchase-api
./purchase-api
```

The API will start on `http://localhost:8080`

### 4. Run Tests (Unit + Integration)

```bash
go test ./... -v
```

## API Examples

All monetary amounts are **strings** for precision. Dates are ISO 8601 format.

### Health Check

**Liveness Probe:**
```bash
curl -X GET http://localhost:8080/health
```

Response:
```json
{
  "status": "ok"
}
```

**Readiness Probe:**
```bash
curl -X GET http://localhost:8080/health/ready
```

Response (when ready):
```json
{
  "status": "ok"
}
```

### Create Purchase

**Request:**
```bash
curl -X POST http://localhost:8080/purchases \
  -H "Content-Type: application/json" \
  -H "X-Request-ID: my-request-123" \
  -d '{
    "description": "Flight to Paris",
    "transactionDate": "2026-06-15",
    "amountUsd": "1500.00"
  }'
```

**Response (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Flight to Paris",
  "transactionDate": "2026-06-15",
  "amountUsd": "1500.00",
  "createdAt": "2026-06-18T14:30:00Z",
  "updatedAt": "2026-06-18T14:30:00Z"
}
```

### Retrieve Purchase (No Conversion)

**Request:**
```bash
curl -X GET http://localhost:8080/purchases/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-Request-ID: my-request-123"
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Flight to Paris",
  "transactionDate": "2026-06-15",
  "amountUsd": "1500.00",
  "createdAt": "2026-06-18T14:30:00Z",
  "updatedAt": "2026-06-18T14:30:00Z"
}
```

### Retrieve Purchase with Currency Conversion

**Request:**
```bash
curl -X GET "http://localhost:8080/purchases/550e8400-e29b-41d4-a716-446655440000?currency=EUR" \
  -H "X-Request-ID: my-request-123"
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "description": "Flight to Paris",
  "transactionDate": "2026-06-15",
  "amountUsd": "1500.00",
  "createdAt": "2026-06-18T14:30:00Z",
  "updatedAt": "2026-06-18T14:30:00Z",
  "exchangeRate": "0.92",
  "rateDate": "2026-06-12",
  "convertedAmount": "1380.00"
}
```

## Error Responses

All errors return structured JSON with error codes:

### Error Code: DESCRIPTION_TOO_LONG

**Request:**
```bash
curl -X POST http://localhost:8080/purchases \
  -H "Content-Type: application/json" \
  -d '{
    "description": "This description is way too long and exceeds the fifty character limit",
    "transactionDate": "2026-06-15",
    "amountUsd": "100.00"
  }'
```

**Response (400 Bad Request):**
```json
{
  "code": "DESCRIPTION_TOO_LONG",
  "message": "description must be 50 characters or less",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

### Error Code: INVALID_DATE

**Request:**
```bash
curl -X POST http://localhost:8080/purchases \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Future transaction",
    "transactionDate": "2099-12-31",
    "amountUsd": "100.00"
  }'
```

**Response (400 Bad Request):**
```json
{
  "code": "INVALID_DATE",
  "message": "transactionDate cannot be in the future",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

### Error Code: NEGATIVE_AMOUNT

**Request:**
```bash
curl -X POST http://localhost:8080/purchases \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Invalid amount",
    "transactionDate": "2026-06-15",
    "amountUsd": "-100.00"
  }'
```

**Response (400 Bad Request):**
```json
{
  "code": "NEGATIVE_AMOUNT",
  "message": "amountUsd must be positive",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

### Error Code: MISSING_FIELD

**Request:**
```bash
curl -X POST http://localhost:8080/purchases \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Incomplete",
    "transactionDate": "2026-06-15"
  }'
```

**Response (400 Bad Request):**
```json
{
  "code": "MISSING_FIELD",
  "message": "amountUsd is required",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

### Error Code: VALIDATION_ERROR (Invalid Currency)

**Request:**
```bash
curl -X GET "http://localhost:8080/purchases/550e8400-e29b-41d4-a716-446655440000?currency=INVALID"
```

**Response (400 Bad Request):**
```json
{
  "code": "VALIDATION_ERROR",
  "message": "Invalid currency code: INVALID. Expected ISO 4217 3-letter code",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

### Error Code: RATE_NOT_FOUND

**Request:**
```bash
curl -X GET "http://localhost:8080/purchases/550e8400-e29b-41d4-a716-446655440000?currency=GBP"
```

**Response (400 Bad Request):**
```json
{
  "code": "RATE_NOT_FOUND",
  "message": "No exchange rate available for GBP on or before 2026-06-15",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

### Error Code: NOT_FOUND

**Request:**
```bash
curl -X GET http://localhost:8080/purchases/00000000-0000-0000-0000-000000000000
```

**Response (404 Not Found):**
```json
{
  "code": "NOT_FOUND",
  "message": "Purchase with ID 00000000-0000-0000-0000-000000000000 not found",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

## X-Request-ID Header

All requests and responses include an `X-Request-ID` header for request tracing:

```bash
# Request with X-Request-ID
curl -X GET http://localhost:8080/purchases/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-Request-ID: my-trace-id-123"

# Response includes same X-Request-ID
# X-Request-ID: my-trace-id-123

# If not provided, server generates one automatically
```

## Configuration

Environment variables:

- `DATABASE_URL`: PostgreSQL connection string (default: `postgres://postgres:postgres@localhost:5432/purchase_api?sslmode=disable`)
- `API_PORT`: HTTP server port (default: `8080`)
- `TREASURY_PROVIDER`: Rate provider type (default: `real`)
  - `real`: Live Treasury API (default, production-ready)
  - `mock`: Deterministic test rates (0.92 EUR/USD, testing only)
- `LOG_LEVEL`: Logging level (default: `INFO`)
  - `DEBUG`, `INFO`, `WARN`, `ERROR`

**Example with custom configuration:**
```bash
DATABASE_URL=postgres://user:pass@db:5432/api \
API_PORT=9000 \
TREASURY_PROVIDER=real \ # default, fetch live Treasury API
LOG_LEVEL=DEBUG \
./purchase-api
```
