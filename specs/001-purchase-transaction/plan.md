# Implementation Plan: Store and Retrieve Purchases (Currency Conversion)

**Branch**: `master` | **Date**: 2026-06-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-purchase-transaction/spec.md`

## Summary

Implement a small web service that stores purchase transactions (description, transaction date, USD amount) with strict financial precision, and provides retrieval endpoints that optionally convert stored USD amounts to a target currency using the Treasury Reporting Rates of Exchange dataset. The implementation will follow hexagonal architecture (ports & adapters), domain-driven design for the `Purchase` entity, ACID-compliant persistence with PostgreSQL, and comprehensive automated tests (unit, integration, contract).

## Technical Context

**Language/Version**: Go 1.21+ (or latest stable)  
**Primary Dependencies**: `chi` for HTTP routing, `sqlc` for SQL code generation, `pgx` driver for PostgreSQL, `golang-migrate` for migrations, `testify` for tests, `github.com/shopspring/decimal` or `int64` cents for money, `swaggo` or manual OpenAPI YAML for docs.  
**Storage**: PostgreSQL (ACID) for production; Dockerized PostgreSQL for local development and CI. SQL is the best fit: strong consistency, relational queries, transaction guarantees, and efficient time-series lookups for exchange rates. `sqlc` is preferred over ORMs like GORM.  
**Testing**: `go test ./...` with `testify`; unit tests for domain/use-cases; integration tests using `dockertest` or Docker Compose with a real PostgreSQL instance; contract tests for the Treasury adapter. Structured error response and logging validation included in test coverage.  
**Target Platform**: Linux containers (Docker), Kubernetes-ready  
**Project Type**: Web service / API  
**Performance Goals**: Correctness-first low-latency service targeting <1 second latency per request. Initial scale: <100 purchases/day, <10 concurrent users. Simple database indexing (primary key, composite on currency+rateDate) is sufficient; no distributed caching, connection pooling optimization, or query tuning required initially. Horizontal scaling via multiple instances with shared PostgreSQL backend.  
**Constraints**: Financial precision using integer cents or `decimal`. Explicit rounding rules: round amount input to nearest cent using half-away-from-zero. Treasury rate lookup must select rate with date ≤ purchase date and ≥ purchase date - 6 months. Rate lookup failure is error-only (HTTP 400)—no fallback to cached/older rates, no degraded response. Conversion is all-or-nothing.  
**Scale/Scope**: Small service focused on correctness, observability (structured JSON logging & errors), and secure deployability. Data retention: indefinite (no time-based archival); support explicit deletion on request.

## Constitution Check

GATES (from constitution):
- Hexagonal/DDD: MUST be satisfied — plan uses ports/adapters, domain model separate from infra. ✅
- Financial precision: MUST use `decimal` or integer cents; explicitly documented rounding rules (AwayFromZero). ✅
- ACID persistence: MUST use ACID datastore; plan prescribes PostgreSQL for production and local development to preserve production parity. ✅
- Tests & CI: Unit, integration, and contract tests required; CI pipeline must run tests. ✅
- Observability & Security: Structured JSON logging (stdout) with timestamp/level/component/message/context fields. Error responses standardized: `{ code, message, timestamp }`. Optional request ID (`X-Request-ID`) for tracing. Health endpoint and TLS in deployment notes. ✅

GATE RESULT: PASS — no constitution violations identified. Proceed to Phase 0.

## Project Structure

Documentation for this feature:
```text
specs/001-purchase-transaction/
├── spec.md
├── plan.md         # (this file)
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── purchases-openapi.yaml
└── checklists/
    └── requirements.md
```

Source code layout (single-project API):
```text
purchase-api/
├── cmd/                # main application entrypoint
├── internal/
│   ├── api/            # HTTP transport and handlers
│   ├── app/            # application services/use-cases
│   ├── domain/         # entities, value objects, interfaces (ports)
│   ├── adapters/       # DB and Treasury adapters
│   ├── config/         # configuration and options
│   ├── db/             # SQL schema and sqlc query files
│   └── ports/          # interface definitions (ports)
├── tests/              # unit/integration/contract
├── db/                 # database migrations and queries
├── cmd/                # main application entrypoint
└── Makefile            # build and test automation
```

**Structure Decision**: Single API project simplifies delivery and aligns with constitution principles (clear domain boundary, ports/adapters). Use separate folders to enforce hexagonal boundaries.

## Complexity Tracking

No constitution violations requiring justification. No additional complexity tracking entries.

## Phase 0: Research (Output: research.md)

See `research.md` for details: decisions on numeric types, database choice (SQL vs NoSQL), treasury adapter design (error-only rate lookup failure, 6-month window), caching strategy (none required initially), migrations, structured logging/error handling patterns, and testing patterns. SQLC will be used as the DB access layer generator for PostgreSQL. Database indexing: `purchases(id PRIMARY KEY)`, `exchange_rates(currency, rateDate COMPOSITE)` for efficient 6-month lookups.

## Design Decisions: Error Handling & Observability

**Error Response Format**:
```json
{
  "code": "ERROR_CODE",
  "message": "Human-readable error message",
  "timestamp": "2026-06-18T14:30:00Z"
}
```
Examples:
- `RATE_NOT_FOUND`: "No exchange rate available for EUR on or before 2026-06-18"
- `VALIDATION_ERROR`: "Description exceeds 50 character limit"
- `INVALID_REQUEST`: "Missing or invalid field: amountUsd"

**Structured Logging** (JSON to stdout):
All application events logged with: `timestamp`, `level` (info/warn/error), `component`, `message`, `context` (relevant IDs/values).  
Example: `{"timestamp":"2026-06-18T14:30:00Z","level":"error","component":"purchase_service","message":"rate lookup failed","context":{"purchase_id":"550e8400-e29b-41d4-a716-446655440000","currency":"EUR","reason":"no_rate_within_window"}}`

**Request Tracing** (Optional):
- Accept `X-Request-ID` header from client or generate UUID if absent.
- Include request ID in all response error objects.
- Log request ID with all related events for this request.
- Return request ID in response headers for client tracking.

**Conversion Failure Behavior**:
- Rate lookup succeeds (rate found within 6 months): return 200 with converted amount.
- Rate lookup fails (no rate within 6 months): return 400 with RATE_NOT_FOUND error.
- No fallback to older rates, cached rates, or degraded responses.
- All-or-nothing conversion ensures financial precision and client clarity.

## Phase 1: Design Outputs

- `data-model.md`: Entity definitions (Purchase, ExchangeRate cache) and schema suggestions.
- `contracts/`: OpenAPI contract for the minimal API surface.
- `quickstart.md`: Developer onboarding steps to create, run, and test the service locally using PostgreSQL and sqlc.

---

