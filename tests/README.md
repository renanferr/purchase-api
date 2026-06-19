# Integration Tests Directory

This directory contains integration tests that verify the purchase API end-to-end functionality.

## Structure

```
tests/
├── integration/
│   └── purchase_api_test.go  - API integration test suites
└── README.md                 - This file
```

## Running Integration Tests

### Requirements

- PostgreSQL database running and accessible
- Environment variables configured:
  - `DATABASE_URL` - PostgreSQL connection string (e.g., `postgres://user:pass@localhost/purchase_api_test`)
  - `TREASURY_PROVIDER` - Set to `sample` for tests (to avoid external API calls)
  - `API_PORT` - Port for test server (default: 8080)
  - `LOG_LEVEL` - Log level (default: INFO)

### Run Tests

**Run integration tests only**:
```bash
make integration-test
# or
go test -v ./tests/...
```

**Run unit tests only** (from repo root):
```bash
make unit-test
# or
go test -v ./internal/...
```

**Run all tests** (unit + integration):
```bash
make test
# or
go test -v ./...
```

**Run specific integration test**:
```bash
go test -v ./tests/integration -run TestPurchaseAPIIntegrationSuite/TestCreatePurchaseEndpoint_TableDriven
```

**Run with coverage**:
```bash
go test -cover ./tests/...
```

## Test Suites

### PurchaseAPIIntegrationTestSuite

Main integration test suite testing the full API stack:

- **TestCreatePurchaseEndpoint_TableDriven** - POST /purchases endpoint
  - ✓ Create purchase successfully
  - ✓ Validation error handling
  - ✓ Invalid date format rejection

- **TestGetPurchaseEndpoint_TableDriven** - GET /purchases/{id} endpoint
  - ✓ Retrieve purchase without conversion
  - ✓ Retrieve purchase with currency conversion
  - ✓ 404 for nonexistent purchase
  - ✓ 400 for invalid currency code

- **TestCurrencyConversionFlow_TableDriven** - End-to-end conversion
  - ✓ Valid rate conversion
  - ✓ USD passthrough (1:1 rate)
  - ✓ Rate outside 6-month window rejection

- **TestRequestIDPropagation_TableDriven** - X-Request-ID header
  - ✓ Custom request ID propagation
  - ✓ Auto-generated request ID

- **TestErrorHandling_TableDriven** - Error responses
  - ✓ Missing field validation
  - ✓ Negative amount rejection
  - ✓ Description length validation
  - ✓ Invalid date format

## Test Setup & Teardown

- **SetupSuite()**: Runs once before all tests
  - Initialize database connection
  - Start HTTP server
  - Run migrations
  - Seed test data

- **TearDownSuite()**: Runs once after all tests
  - Clean up database
  - Close connections
  - Stop server

- **SetupTest()**: Runs before each individual test
  - Clear tables between tests
  - Reset test state

## Difference from Unit Tests

| Aspect | Unit Tests | Integration Tests |
|--------|-----------|------------------|
| Location | `internal/app`, `internal/api` | `tests/integration` |
| Scope | Individual functions/methods | Full system interactions |
| Dependencies | Mocked (fakes) | Real database, HTTP handlers |
| Speed | Fast (< 1ms each) | Slower (network I/O) |
| Environment | No external dependencies | Requires database + server |
| CI/CD | Run on every commit | Run separately or on PR |

## CI/CD Integration

In your CI/CD pipeline, you can run tests separately:

```yaml
# GitHub Actions example
jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make unit-test

  integration-tests:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_DB: purchase_api_test
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make integration-test
```

## Notes

- Integration tests use testify suites for consistent structure
- Table-driven tests allow easy addition of new test cases
- All tests follow the same naming and assertion patterns as unit tests
- Database is expected to be clean before each test run
- Tests use deterministic test data for reproducibility
