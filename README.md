# Purchase API

A RESTful API for managing purchase transactions with multi-currency support and exchange rate conversions.

Built with:
- **Language**: Go 1.21+
- **Database**: PostgreSQL 15+
- **Framework**: Chi v5 (HTTP router)
- **Database Layer**: sqlc (typesafe SQL)
- **Migrations**: golang-migrate v4

## Features

- ✅ **Create purchases** with description, transaction date, and USD amount
- ✅ **Retrieve purchases** with optional currency conversion
- ✅ **Multi-currency support** with Treasury exchange rates
- ✅ **Structured error codes** for client error handling
- ✅ **Request ID tracing** via X-Request-ID headers
- ✅ **Health checks** (liveness and readiness probes)
- ✅ **Comprehensive integration tests** with real PostgreSQL
- ✅ **CI/CD workflow** with GitHub Actions

## Project Structure

```
purchase-api/
├── cmd/
│   ├── purchase-api/          # Main application entry point
│   └── migrate/               # Database migration CLI tool
├── db/
│   ├── migrations/            # SQL migration files (golang-migrate format)
│   └── queries/               # SQLc query definitions
├── internal/
│   ├── app/                   # Business logic layer (services)
│   ├── api/                   # HTTP handlers layer
│   ├── domain/                # Domain entities
│   ├── adapters/
│   │   ├── db/               # PostgreSQL adapter (sqlc-based)
│   │   └── treasury/         # Treasury API adapter
│   ├── ports/                 # Interface definitions
│   ├── config/                # Application configuration
│   └── migrations/            # Migration runner (golang-migrate wrapper)
├── tests/
│   ├── integration/           # Integration tests with real DB
│   └── README.md              # Test documentation
├── .github/workflows/         # GitHub Actions CI/CD
├── Makefile                   # Build and test targets
├── docker-compose.yml         # PostgreSQL Docker setup
├── sqlc.yaml                  # SQLc configuration
└── README.md                  # This file
```

## Architecture Design

### Hexagonal Architecture (Ports & Adapters)

The application follows **hexagonal/ports-and-adapters architecture** with clear separation of concerns:

```
┌─────────────────────────────────────────────────────┐
│                    HTTP Layer (API)                  │
│  - Handlers (request/response, validation)          │
│  - Middleware (logging, request ID tracking)        │
│  - Swagger/OpenAPI documentation                    │
└────────────────────────┬────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────┐
│           Application Layer (Services)              │
│  - Business logic orchestration                     │
│  - Currency conversion workflow                     │
│  - Error handling and validation                    │
└────────┬────────────────────────────────────────┬───┘
         │                                        │
┌────────▼──────────┐                  ┌─────────▼──────────┐
│  Ports (Domain)   │                  │  Ports (External)  │
│  - Repositories   │                  │  - Rate Provider   │
│  - Transactions   │                  │                    │
└────────┬──────────┘                  └─────────┬──────────┘
         │                                        │
┌────────▼──────────────────────┐       ┌────────▼──────────┐
│ PostgreSQL Adapter            │       │ Treasury API      │
│ - Purchase persistence        │       │ - Exchange rates  │
│ - Exchange rate cache         │       │ - Rate lookups    │
│ - Query execution             │       │                   │
└───────────────────────────────┘       └───────────────────┘
```

**Key Design Principles:**

1. **Domain-Driven Design**: Core business logic (Purchase, ExchangeRate entities) lives in `domain/` package with clear validation rules
2. **Port/Adapter Pattern**: Database and external APIs are swappable via interfaces defined in `ports/`
3. **Clean Dependencies**: Dependencies flow inward; infrastructure code never imports business logic
4. **Type Safety**: sqlc generates Go code from SQL, ensuring compile-time correctness

### Data Layer

The data layer uses **PostgreSQL with type-safe SQL generation (sqlc)**:

- **sqlc**: Generates Go functions from hand-written SQL queries (`db/queries/`), avoiding ORM overhead
- **golang-migrate**: Version-controlled schema migrations in `db/migrations/`
- **pgx**: Native PostgreSQL driver with connection pooling

### Exchange Rate System (External Integration)

The application fetches exchange rates from the **US Treasury Rates of Exchange API**:

**API Endpoint**: `https://api.fiscaldata.treasury.gov/services/api/fiscal_service/v1/accounting/od/rates_of_exchange`

**Rate Lookup Strategy**:

1. **On-Demand Fetching**: When a purchase is retrieved with a target currency, the system:
   - First checks the local `exchange_rates` cache table
   - If rate exists and was published within 6 months of purchase date, uses cached rate
   - Otherwise, queries the Treasury API for the most recent rate ≤ purchase date

2. **Automatic Caching**: Rates fetched from Treasury API are stored in the database for future requests

3. **Lazy Loading**: No aggressive pre-fetching; rates are cached only when needed (during reads)

4. **Supported Currencies**: 16 currencies with automatic code→name mapping (EUR, GBP, JPY, etc.)

**Example Flow**:
```
GET /purchases/{id}?currency=EUR
  ↓
Check exchange_rates table for (EUR, ≤ purchase_date)
  ↓
If found within 6 months → return cached rate (instant)
  ↓
If not found → query Treasury API → store in cache → return rate
  ↓
If API fails or no rate within 6 months → return 400 error
```

**Rate Precision**: All rates are stored as `NUMERIC(18,6)` and represent the conversion from USD to target currency (e.g., 1 USD = 0.87 EUR)

## Data Model

### Database Schema

The application stores two primary entities:

#### Purchases Table
```sql
CREATE TABLE purchases (
  id UUID PRIMARY KEY,
  description VARCHAR(50) NOT NULL,
  transaction_date DATE NOT NULL,
  amount_usd_cents BIGINT NOT NULL CHECK (amount_usd_cents > 0),
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_purchases_created_at ON purchases(created_at);
```

**Design Notes**:
- Amount stored as **cents (integer)** to guarantee financial precision (no floating-point errors)
- UUID primary key for globally unique identifiers
- Timestamps for audit trail

#### Exchange Rates Table (Cache)
```sql
CREATE TABLE exchange_rates (
  currency CHAR(3) NOT NULL,
  rate_date DATE NOT NULL,
  rate NUMERIC(18,6) NOT NULL CHECK (rate > 0),
  created_at TIMESTAMP NOT NULL,
  PRIMARY KEY (currency, rate_date),
  UNIQUE (currency, rate_date)
);
CREATE INDEX idx_exchange_rates_lookup ON exchange_rates(currency, rate_date DESC);
```

**Design Notes**:
- Composite primary key on `(currency, rate_date)` to prevent duplicates
- Rates indexed in DESC order for efficient lookups of latest rate ≤ target date
- Rate values are **1 USD = X target_currency** (e.g., 1 USD = 0.87 EUR)
- Historical data never deleted; provides audit trail of rates over time

### Entity Relationships

```
Purchase ──(via currency & date lookup)──> ExchangeRate

Purchase (one) ────many──── ExchangeRate (zero or more rates for lookups)
  - No foreign key constraint
  - ExchangeRate is historical/time-series data
  - Queries by (currency, date ≤ purchase_date) within 6-month window
```

### Data Validation Rules

| Entity | Field | Rules |
|--------|-------|-------|
| Purchase | description | Required, 1-50 characters |
| Purchase | transaction_date | Required, ISO 8601 (YYYY-MM-DD), cannot be in future |
| Purchase | amount_usd_cents | Required, positive integer (>0) |
| ExchangeRate | currency | ISO 4217 3-letter code (e.g., EUR, GBP) |
| ExchangeRate | rate_date | Date of rate publication |
| ExchangeRate | rate | Positive decimal (>0), 1 USD = rate × target_currency |

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+ (or Docker & Docker Compose)
- `sqlc` (for database code generation)
- `golang-migrate` (for migrations)

### Setup Steps

#### 1. Install Dependencies

```bash
go mod download
```

#### 2. Configure Environment Variables

Copy the example environment file and update with your local configuration:

```bash
cp .env.example .env
```

Edit `.env` with your configuration:
```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/purchase_api?sslmode=disable
API_PORT=8080
LOG_LEVEL=INFO
```

**Important:** 
- `.env` is in `.gitignore` - never commit it
- `.env.example` is a template committed to the repository
- Always keep `.env.example` up-to-date with new configuration options

See [.env.example](.env.example) for all available configuration options.

#### 3. Start PostgreSQL (Docker Compose)

```bash
docker compose up -d postgres
```

Or connect to an existing PostgreSQL instance by updating `DATABASE_URL` in `.env`:
```env
DATABASE_URL=postgres://user:password@localhost:5432/purchase_api?sslmode=disable
```

#### 4. Generate Database Code

```bash
sqlc generate
```

#### 5. Run Migrations

```bash
go run ./cmd/migrate -dir up
```

Or use the Makefile:
```bash
make migrate-up
```

#### 6. Build the Application

```bash
go build -o bin/purchase-api ./cmd/purchase-api
```

#### 7. Run the Application

```bash
./bin/purchase-api
```

The API will start on `http://localhost:8080`

## Make Commands

The project includes a comprehensive Makefile with convenient commands for common tasks. Use `make help` to see all available commands.

### Quick Start with Make

**Recommended: Full Docker Stack (One Command)**
```bash
make run-docker
# Builds containers, starts postgres + app, output shows API endpoints
```

**Local Development with Docker Postgres**
```bash
# Terminal 1: Start postgres in Docker
make docker-up

# Terminal 2: Run app with live code reloading
make run-dev
# API available at http://localhost:8080

# Stop containers when done
make docker-down
```

**Local Development with Compiled Binary**
```bash
# Terminal 1: Start postgres in Docker
make docker-up

# Terminal 2: Build and run the binary
make build
make run-binary
# API available at http://localhost:8080

# Stop containers when done
make docker-down
```

### Make Command Reference

**Building**

| Command | Purpose |
|---------|---------|
| `make build` | Build app binary (with cache) |
| `make build-no-cache` | Build app binary (clears build cache first) |
| `make install-tools` | Install development tools (sqlc, swag) |

**Running the App**

| Command | Purpose |
|---------|---------|
| `make run-dev` | Run with `go run` (live reloading, postgres required) |
| `make run-binary` | Run compiled binary (postgres required) |
| `make run-docker` | Run full stack in Docker (postgres + app) |

**Docker Operations**

| Command | Purpose |
|---------|---------|
| `make docker-up` | Start postgres + app containers |
| `make docker-down` | Stop all containers |
| `make docker-build` | Build only the app Docker image |
| `make docker-compose-build` | Build all docker-compose images (postgres + app) |

**Testing & Quality**

| Command | Purpose |
|---------|---------|
| `make test` | Run all unit + integration tests |
| `make unit-test` | Run unit tests only (fast) |
| `make integration-test` | Run integration tests (requires database) |
| `make coverage` | Generate HTML coverage report |
| `make lint` | Run code linters |
| `make fmt` | Format code with gofmt |
| `make vet` | Run go vet analysis |

**Database**

| Command | Purpose |
|---------|---------|
| `make migrate-up` | Run pending migrations |
| `make migrate-down` | Rollback migrations |

**Code Generation**

| Command | Purpose |
|---------|---------|
| `make sqlc-generate` | Generate type-safe database code from SQL |
| `make swagger-generate` | Generate Swagger/OpenAPI docs |

**Utilities**

| Command | Purpose |
|---------|---------|
| `make deps` | Download and verify dependencies |
| `make tidy` | Tidy go.mod and go.sum |
| `make clean` | Clean build artifacts |
| `make help` | Display all available commands |

## Configuration

### Environment Variables

All configuration is loaded from `.env` file or environment variables. Copy `.env.example` to `.env` and customize:

```bash
cp .env.example .env
```

**Available configuration options:**

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/purchase_api?sslmode=disable` | Yes | PostgreSQL connection string |
| `API_PORT` | `8080` | No | HTTP server port |
| `LOG_LEVEL` | `INFO` | No | Logging level: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `TREASURY_API_URL` | `https://www.treasurydirect.gov` | No | Treasury service API endpoint |
| `HTTP_TIMEOUT_SECONDS` | `10` | No | HTTP timeout for external service calls |
| `SHUTDOWN_TIMEOUT_SECONDS` | `30` | No | Server shutdown timeout |

### Priority Order

Configuration is loaded in this order (later sources override earlier ones):
1. Default hardcoded values in code
2. Environment variables
3. `.env` file values

**Example `.env` for development:**
```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/purchase_api?sslmode=disable
API_PORT=8080
LOG_LEVEL=DEBUG
```

**Example for Docker Compose:**
```env
DATABASE_URL=postgres://postgres:postgres@postgres:5432/purchase_api?sslmode=disable
API_PORT=8080
LOG_LEVEL=INFO
```

## Testing

The project includes comprehensive unit and integration tests with strong coverage metrics:

**Test Coverage Summary:**
- **Domain Layer**: 91.7% coverage (Purchase entities, Money value objects, business rules)
- **Application Layer**: 81.3% coverage (Service business logic, currency conversion)
- **API Layer**: 26.3% coverage (Request validation, error handling)
- **Total**: 159 test cases, all passing

**Test Structure:**
- `internal/domain/domain_test.go` - Domain entity and value object tests
- `internal/app/app_test.go` - Business logic tests
- `internal/api/handlers_test.go` - HTTP handler and validation tests
- `tests/integration/` - Integration tests with real database

### Running Tests

```bash
# Run all unit tests (fast)
go test ./internal/...

# Run all tests with coverage
go test ./... -cover

# Run integration tests (requires Docker and database)
go test ./tests/integration -v

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Testing

Integration tests use a **separate PostgreSQL database** (`purchase_api_test`) to keep test data isolated from your development database (`purchase_api`).

**Prerequisites:**
- PostgreSQL 15+ installed and running (or Docker Desktop for `docker compose up`)
- Project dependencies installed (`go mod download`)

**One-Time Setup - Create Test Database:**

The project includes setup scripts to automatically create the test database:

**Windows (PowerShell):**
```powershell
.\scripts\setup-test-db.ps1
```

**Linux/macOS (Bash):**
```bash
bash scripts/setup-test-db.sh
```

**Manual Setup (if scripts unavailable):**
```sql
-- Connect to PostgreSQL as administrator
psql -h localhost -p 5432 -U postgres -d postgres

-- Create test database
DROP DATABASE IF EXISTS purchase_api_test;
CREATE DATABASE purchase_api_test 
  OWNER postgres
  ENCODING 'UTF8';
```

**Running Integration Tests:**

```bash
# Run all integration tests (uses purchase_api_test database by default)
go test ./tests/integration -v

# Run integration tests with custom test database
export TEST_DATABASE_URL="postgres://user:password@host:5432/custom_test_db?sslmode=disable"
go test ./tests/integration -v
```

PowerShell:
```powershell
$env:TEST_DATABASE_URL="postgres://user:password@host:5432/custom_test_db?sslmode=disable"
go test ./tests/integration -v
```

**Test Database Configuration:**

| Setting | Default | Environment Variable |
|---------|---------|---------------------|
| Database Name | `purchase_api_test` | `TEST_DATABASE_URL` (full connection string) |
| Host | localhost | (in connection string) |
| Port | 5432 | (in connection string) |
| User | postgres | (in connection string) |
| Password | postgres | (in connection string) |

**Test Lifecycle:**
- Before each test: Database schema is created via migrations
- During tests: Tables are cleaned between test runs
- After tests: Test database remains for inspection (re-run setup script to reset)
- Development database: Completely unaffected by integration tests

**View Coverage:**
```bash
go test ./tests/integration -v -cover
```

## Development

### Code Quality

```bash
# Format code
make fmt

# Run linters
make lint

# Run go vet
make vet
```

### Database Migrations

```bash
# Apply pending migrations
make migrate-up

# Rollback all migrations
make migrate-down

# Check migration status
go run ./cmd/migrate -dir status
```

### Swagger/OpenAPI Documentation

The API documentation is **auto-generated** from doc comments in the code using [swaggo](https://github.com/swaggo/swag).

#### Regenerating the Swagger Specs

After modifying handler functions or request/response types, regenerate the Swagger specification:

```bash
# Regenerate swagger.json and swagger.yaml in docs/
swag init -g cmd/purchase-api/main.go
```

The generated files are:
- `docs/swagger.json` - Swagger 2.0 specification (JSON)
- `docs/swagger.yaml` - Swagger 2.0 specification (YAML)
- `docs/docs.go` - Go code for embedding (auto-generated, not committed to git)

These files should be committed to version control so the API spec stays in sync with the code.

## API Documentation

### Interactive Documentation (Swagger UI)

Once the API is running, open your browser to view interactive API documentation:

- **Swagger UI**: [http://localhost:8080/api/docs](http://localhost:8080/api/docs)
- **Swagger JSON**: [http://localhost:8080/api/docs/swagger.json](http://localhost:8080/api/docs/swagger.json)
- **Swagger YAML**: [http://localhost:8080/api/docs/swagger.yaml](http://localhost:8080/api/docs/swagger.yaml)
- **OpenAPI JSON (alias)**: [http://localhost:8080/api/docs/openapi.json](http://localhost:8080/api/docs/openapi.json)
- **OpenAPI YAML (alias)**: [http://localhost:8080/api/docs/openapi.yaml](http://localhost:8080/api/docs/openapi.yaml)

The Swagger UI allows you to:
- Browse all endpoints and their parameters
- View request/response schemas with examples
- Try out API calls directly from the browser
- Download the Swagger specification

**Note**: The documentation is automatically generated from Swagger comments in the handler code. See [internal/api/handlers.go](internal/api/handlers.go) and [cmd/purchase-api/main.go](cmd/purchase-api/main.go) for the documentation source.

### API Reference

See [quickstart.md](specs/001-purchase-transaction/quickstart.md) for comprehensive API documentation with curl examples.

### Quick Examples

**Health Check (Liveness):**
```bash
curl http://localhost:8080/health
# { "status": "ok" }
```

**Readiness Check:**
```bash
curl http://localhost:8080/health/ready
# { "status": "ok" }
```

**Create Purchase:**
```bash
curl -X POST http://localhost:8080/purchases \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Flight to Paris",
    "transactionDate": "2026-06-15",
    "amountUsd": "1500.00"
  }'
```

**Get Purchase with Conversion:**
```bash
curl http://localhost:8080/purchases/{id}?currency=EUR
```

---

## Business Rules & Validation

The API enforces strict business rules to ensure financial precision and data integrity:

### Purchase Creation

- **Description**: Required, 1-50 characters
- **Transaction Date**: Required, ISO 8601 format (YYYY-MM-DD), **cannot be in the future**
- **Amount USD**: Required, positive decimal number, rounded to nearest cent on storage

### Currency Conversion

- **Currency Code**: Must be valid ISO 4217 3-letter code (e.g., EUR, GBP, JPY)
- **Exchange Rate Lookup**: Searches for rate on or before purchase date within 6 months
- **Precision**: All monetary amounts are stored and returned as strings to preserve decimal precision
- **Rate Not Found**: Returns 400 error if no valid rate exists (conversion is all-or-nothing)

### Error Handling

All validation errors return HTTP 400 with structured JSON error response containing:
- `code`: Error category (see table below)
- `message`: Human-readable error message  
- `timestamp`: ISO 8601 UTC timestamp

| Error Code | HTTP Status | Description |
|-----------|------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid input or currency code format |
| `INVALID_DATE` | 400 | Transaction date format invalid or date is in the future |
| `NEGATIVE_AMOUNT` | 400 | Amount is zero, negative, or invalid |
| `DESCRIPTION_TOO_LONG` | 400 | Description exceeds 50 character limit |
| `MISSING_FIELD` | 400 | Required field is missing |
| `RATE_NOT_FOUND` | 400 | No exchange rate available for the requested conversion |
| `NOT_FOUND` | 404 | Purchase not found |

**Example Error Response:**
```json
{
  "code": "INVALID_DATE",
  "message": "Purchase date cannot be in the future",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

---

## Error Handling

All errors return structured JSON with error codes:

| Error Code | HTTP Status | Description |
|-----------|------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid input or currency code |
| `INVALID_DATE` | 400 | Transaction date is in the future |
| `NEGATIVE_AMOUNT` | 400 | Amount is zero or negative |
| `DESCRIPTION_TOO_LONG` | 400 | Description exceeds 50 characters |
| `MISSING_FIELD` | 400 | Required field missing |
| `RATE_NOT_FOUND` | 400 | No exchange rate available |
| `NOT_FOUND` | 404 | Purchase not found |

**Example Error Response:**
```json
{
  "code": "DESCRIPTION_TOO_LONG",
  "message": "description must be 50 characters or less",
  "timestamp": "2026-06-18T14:30:00Z"
}
```

## Request Tracing

All requests and responses include an `X-Request-ID` header for tracing:

```bash
# Client provides request ID
curl -H "X-Request-ID: my-trace-123" http://localhost:8080/purchases

# Response includes same ID
X-Request-ID: my-trace-123
```

If not provided, the server generates a UUID automatically.

## Troubleshooting

### PostgreSQL Connection Issues

**Problem**: `dial error: dial tcp ... connection refused`

**Solution**: 
1. Verify PostgreSQL is running: `docker ps` should show postgres container
2. Check connection string: `echo $DATABASE_URL`
3. Restart PostgreSQL: `docker compose down && docker compose up -d postgres`

### Database Schema Not Found

**Problem**: `database "purchase_api" does not exist`

**Solution**:
1. Create database: `psql -U postgres -c "CREATE DATABASE purchase_api"`
2. Or update connection string to existing database
3. Run migrations: `go run ./cmd/migrate -dir up`

### Migrations Won't Apply

**Problem**: `no migration files found` or `error: migration failed`

**Solution**:
1. Verify migration files exist: `ls db/migrations/`
2. Check permissions: Files should be readable
3. Run with verbose output: `go run ./cmd/migrate -dir up -v`

### Tests Fail

**Problem**: Integration tests fail with database errors

**Solution**:
1. Set DATABASE_URL: `export DATABASE_URL="postgres://postgres:postgres@localhost:5432/purchase_api?sslmode=disable"`
2. Start PostgreSQL: `docker compose up -d postgres`
3. Run migrations: `go run ./cmd/migrate -dir up`
4. Run tests: `go test ./tests/integration/... -v`

### Port Already in Use

**Problem**: `listen tcp :8080: bind: address already in use`

**Solution**:
1. Use different port: `API_PORT=9000 ./bin/purchase-api`
2. Or kill process using port: `lsof -i :8080` then `kill -9 <PID>`

### Log Output Not JSON

**Problem**: Logs appear as text instead of JSON

**Solution**:
1. Logs are formatted as JSON by default
2. Set LOG_LEVEL: `LOG_LEVEL=DEBUG ./bin/purchase-api`
3. For human-readable output, pipe through `jq`: `./bin/purchase-api | jq '.'`

## CI/CD

GitHub Actions workflow runs on every push/PR:
- Generates sqlc code and verifies no drift
- Applies migrations to test database
- Runs unit and integration tests with coverage
- Checks code formatting (`gofmt`)
- Runs linters (`golangci-lint`)
- Verifies all error codes are tested

See [.github/workflows/ci.yml](.github/workflows/ci.yml) for details.

## Contributing

1. Create feature branch: `git checkout -b feature/your-feature`
2. Make changes and run tests: `make test`
3. Format code: `make fmt`
4. Push and create pull request

## License

MIT
- **Logging**: log/slog (structured JSON logging)
- **Testing**: testify/suite (test suites and assertions)
- **Utilities**: 
  - decimal (precise money arithmetic)
  - uuid (UUID generation)

## Architecture

### Layers

1. **HTTP Layer** (`internal/api/`)
   - Request/response handling
   - Input validation
   - Error formatting
   - Middleware (logging, request ID)

2. **Application Layer** (`internal/app/`)
   - Business logic
   - Use cases
   - Service orchestration

3. **Domain Layer** (`internal/domain/`)
   - Entities (Purchase, ExchangeRate)
   - Value objects (Money)
   - Business rules

4. **Adapter Layer** (`internal/adapters/`)
   - Database (PostgreSQL via pgx)
   - External APIs (Treasury exchange rates)

5. **Ports Layer** (`internal/ports/`)
   - Interface definitions
   - Dependency contracts

## Features

- ✅ Purchase transaction creation with validation
- ✅ Purchase retrieval with audit timestamps
- ✅ Multi-currency conversion with Treasury API integration
- ✅ 6-month exchange rate window enforcement
- ✅ Structured JSON logging with request tracing
- ✅ Comprehensive error handling with error codes
- ✅ Options pattern for dependency injection
- ✅ Type-safe SQL queries with sqlc

## Contributing

1. Write unit tests first (TDD)
2. Format code: `make fmt`
3. Run linters: `make lint`
4. Run all tests: `make test`
5. Update documentation

## License

MIT
