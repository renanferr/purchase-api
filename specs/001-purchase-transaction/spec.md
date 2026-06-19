# Feature Specification: Store and Retrieve Purchases (Currency Conversion)

**Feature Branch**: `001-purchase-transaction`  
**Created**: 2026-06-17  
**Status**: Draft  
**Input**: User description: "Store a purchase transaction and retrieve it converted to a specified country's currency using Treasury Reporting Rates of Exchange. Financial precision and hexagonal architecture required."

## Clarifications

### Session 2026-06-17

- Q: How should the Treasury exchange rate be interpreted for conversion? → A: Use the rate as target currency per USD, so convertedAmount = amountUsd * rate.

### Session 2026-06-18 (Round 1)

- Q: What are the performance/scale expectations? → A: Modest scale (<100 purchases/day, <10 concurrent users, <1s latency). Simple indexing sufficient; no advanced caching required initially.
- Q: How should errors and observability be handled? → A: Structured JSON error responses (code, message, timestamp). Structured logging (JSON) to stdout. Optional request ID support.
- Q: What should happen when no Treasury rate exists within 6 months? → A: Return 400 Bad Request with clear error message. No fallback, no degraded response—conversion is all-or-nothing.
- Q: What is the data retention policy? → A: Indefinite retention. Keep all purchases and rates forever. Support explicit deletion on request only. No time-based archival needed initially.

### Session 2026-06-18 (Round 2)

- Q: Response schemas for POST/GET/errors? → A: Full OpenAPI schemas with all fields and examples (POST 201, GET 200 both variants, 400/404 errors).
- Q: How represent monetary amounts in JSON? → A: String format (e.g., "12.34") for financial precision; avoids floating-point errors.
- Q: Error code set for validation failures? → A: Full set: `VALIDATION_ERROR`, `INVALID_DATE`, `NEGATIVE_AMOUNT`, `DESCRIPTION_TOO_LONG`, `MISSING_FIELD`, `RATE_NOT_FOUND`, `NOT_FOUND`.
- Q: Date format for transactionDate? → A: ISO 8601 date only (YYYY-MM-DD); no time component.
- Q: Currency validation requirements? → A: ISO 4217 3-letter codes; case-insensitive validation against standard list.
- Q: Maximum amount value? → A: Use Go math/big for arbitrary precision; no hard limit specified (suitable for cents representation).
- Q: Should timestamps be added to purchases table? → A: Yes, add `created_at` and `updated_at` for audit trail.
- Q: Treasury adapter integration? → A: Document adapter factory/config pattern for swapping mock provider (testing) ↔ real API (production).
- Q: X-Request-ID in responses? → A: Include in response header if client provided it in request.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create Purchase (Priority: P1)

An API client submits a purchase with a description, transaction date, and USD amount. The system validates the input, persists the purchase with a unique identifier, and returns the stored record.

Why this priority: This is the core data capture flow and required before any retrieval/conversion can occur.

Independent Test: POST `/purchases` with valid input → 201 Created and body contains `id`, `description`, `transactionDate`, and `amountUsd` rounded to cents.

Acceptance Scenarios:
1. Given valid inputs, When POST `/purchases`, Then respond 201 with the stored purchase including an assigned UUID.
2. Given `description` > 50 chars, When POST `/purchases`, Then respond 400 with validation error.
3. Given `amountUsd` ≤ 0 or invalid, When POST `/purchases`, Then respond 400 with validation error.
4. Given invalid date, When POST `/purchases`, Then respond 400 with validation error.

---

### User Story 2 - Retrieve Purchase Without Conversion (Priority: P1)

An API client requests a stored purchase by id without specifying a target currency and receives the original USD amount and metadata.

Independent Test: GET `/purchases/{id}` → 200 with original USD amount and metadata.

Acceptance Scenario:
1. Given an existing purchase id, When GET `/purchases/{id}`, Then return id, description, transactionDate, amountUsd.

---

### User Story 3 - Retrieve Purchase With Currency Conversion (Priority: P1)

An API client requests a stored purchase by id with `?currency=XXX`. The system finds a treasury exchange rate for the requested currency on or before the purchase date within the prior 6 months, applies it, and returns the exchange rate and converted amount rounded to two decimals. If no rate exists within 6 months prior, return an error.

Independent Test: GET `/purchases/{id}?currency=EUR` → 200 with exchangeRate and convertedAmount, or 400 with a clear error if no valid rate.

Acceptance Scenarios:
1. Given purchase date and a treasury rate ≤ date within last 6 months, When GET with currency, Then return exchangeRate (currency, rateDate, rate) and convertedAmount (rounded to 2 decimals).
2. Given no treasury rate within 6 months ≤ date, When GET with currency, Then return 400 with error stating conversion not possible.

---

### Edge Cases

- **Timezone-normalization**: Transaction dates are normalized to ISO 8601 dates (YYYY-MM-DD format; no time component). UTC date portion used for rate lookup.
- **Multiple rates on the same date**: Use the latest published rate for that date (domain rule).
- **Large amounts**: Amounts use arbitrary precision (via Go math/big); precise to cents.
- **Rate lookup failure**: If no Treasury rate exists for the requested currency on or before the purchase date within the prior 6 months, the conversion request fails with HTTP 400. Error response: `{ code: "RATE_NOT_FOUND", message: "No exchange rate available for {currency} on or before {purchaseDate}", timestamp: "ISO-8601" }`. No fallback to older rates, cached rates, or degraded responses—conversion is all-or-nothing to maintain financial precision.
- **Future purchase dates**: If `transactionDate` is in the future, return HTTP 400 with error code `INVALID_DATE` and message "Purchase date cannot be in the future".
- **Validation error codes**: 
  - Missing/null required field → `MISSING_FIELD`: "Missing required field: {fieldName}"
  - `description` > 50 characters → `DESCRIPTION_TOO_LONG`: "Description exceeds 50 character limit"
  - `amountUsd` ≤ 0 or non-numeric → `NEGATIVE_AMOUNT`: "Amount must be a positive number"
  - `transactionDate` invalid or future → `INVALID_DATE`: "Invalid date: {details}"
  - `currency` parameter invalid (not ISO 4217) → `VALIDATION_ERROR`: "Invalid currency code: {currency}. Expected ISO 4217 3-letter code (case-insensitive)"
- **Monetary amount format**: JSON amounts (amountUsd, convertedAmount) are **strings** (e.g., "1234.56") to preserve financial precision and avoid floating-point errors. Client must parse as decimal.
- **Currency codes**: ISO 4217 3-letter codes (EUR, GBP, JPY, etc.); case-insensitive input validation; stored as uppercase.
- **Request headers**: 
  - Required: `Content-Type: application/json`
  - Optional: `X-Request-ID` (UUID or correlation ID); returned in response headers if provided.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Accept POST `/purchases` with payload `{ description, transactionDate, amountUsd }` and persist a purchase with unique identifier (UUID).
- **FR-002**: Validate `description` length ≤ 50 characters.
- **FR-003**: Validate `transactionDate` is a valid date (ISO 8601). Store the date component for rate lookups.
- **FR-004**: Validate `amountUsd` is a positive number; round to nearest cent on entry and store with exact precision.
- **FR-005**: Provide GET `/purchases/{id}` returning id, description, transactionDate, amountUsd.
- **FR-006**: Provide GET `/purchases/{id}?currency=XXX` returning id, description, transactionDate, amountUsd, exchangeRate (currency, rateDate, rate), and convertedAmount (rounded to 2 decimals) when a rate exists per currency conversion rules. Treat `rate` as the multiplier from USD to target currency: `convertedAmount = amountUsd * rate`.
- **FR-007**: When converting, use a rate with date ≤ purchase date and ≥ (purchase date - 6 months). If none, return 400 with a clear error message.

### Non-functional Requirements

- **NFR-001**: Use hexagonal architecture; domain logic must not depend on infrastructure.
- **NFR-002**: Monetary calculations must use fixed-point arithmetic (decimal or integer cents) and preserve ACID guarantees in persistence.
- **NFR-003**: Provide automated unit and integration tests covering validation, persistence, and conversion rules.
- **NFR-004**: Document how to swap the Treasury Rates adapter with the real API.
- **NFR-005**: **Performance**: Target <1 second latency for individual requests (modest scale: <100 purchases/day, <10 concurrent users). Simple database indexing on `purchase.id` and `exchange_rate(currency, rateDate)` is sufficient; no distributed caching or query optimization required initially.
- **NFR-006**: **Observability**: Return structured JSON error responses with fields `{ code, message, timestamp }`. Use structured logging (JSON format) for all application events and errors. Support optional request ID header (`X-Request-ID`) for request tracking.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of mandatory acceptance scenarios for the three P1 stories are implemented and covered by automated tests.
- **SC-002**: Monetary amounts stored and returned must match expected cent-precision values in 100% of unit tests.
- **SC-003**: For conversions where a rate exists within rules, 100% of retrievals return a convertedAmount rounded to two decimals matching the rate application.

## Assumptions

- **Monetary precision**: Persisted amounts use a decimal with two fractional digits (cents) or integer cents; the implementation will choose the language-appropriate exact numeric type.
- **Treasury data model**: The Treasury dataset will be consumed via a pull adapter providing rates keyed by currency and publication date. If multiple rates exist for a date, the adapter picks the official published rate.
- **Local development parity**: For local development and CI, use Dockerized PostgreSQL to preserve parity with production; production should also use PostgreSQL or equivalent ACID RDBMS.
- **Authentication/Authorization**: Authentication/authorization is out of scope for this specification; API must be designed so it can be added later without touching domain logic.
- **Data retention**: All purchases and exchange rates are retained indefinitely with no time-based archival. Support explicit deletion on user request only. No soft-delete, no retention timestamps needed in the schema.
- **Observability**: All errors logged in structured JSON format (stdout) with fields: timestamp, level (error/warn/info), component, message, context. Request IDs optional but recommended for request tracing.

## Key Entities

- **Purchase**: id (UUID), description (string ≤50), transactionDate (date), amountUsd (decimal cents)
- **ExchangeRate**: currency (ISO 3-letter), rateDate (date), rate (decimal multiplier such that targetAmount = usdAmount * rate)

## Deliverables

- `spec.md` (this file) in `specs/001-purchase-transaction/`
- `checklists/requirements.md` — spec quality checklist for this feature
- A short implementation plan will be produced by `/speckit.plan` when ready
