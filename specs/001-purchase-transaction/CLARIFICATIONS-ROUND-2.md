# Clarifications: Round 2 (2026-06-18)

## Summary

9 additional clarifications have been gathered to resolve remaining ambiguities in specification, data model, API contract, and implementation patterns. These clarifications address response schemas, monetary representation, error handling, validation rules, and integration patterns.

---

## Clarifications with Integration Targets

### C5: Response Body Schemas in OpenAPI Contract
**Status**: Implemented in spec.md; **Pending**: contracts/purchases-openapi.yaml  
**Answer**: Full OpenAPI schemas with examples

**Action**: Update `contracts/purchases-openapi.yaml` to include:
- POST `/purchases` 201 response with Purchase schema (id, description, transactionDate, amountUsd)
- GET `/purchases/{id}` 200 response (both variants: without currency, with currency conversion)
- Error responses (400, 404) with ErrorResponse schema (code, message, timestamp)
- Clarify parameter types and constraints

---

### C6: Monetary Amount Representation in JSON
**Status**: Pending update to spec.md Assumptions; **Pending**: contracts/purchases-openapi.yaml, quickstart.md  
**Answer**: String format (e.g., "1234.56")

**Rationale**: Avoid floating-point precision loss; standard for financial APIs.

**Action**: Update documents:
1. **spec.md** Assumptions: Add "Monetary amounts (amountUsd, convertedAmount) are **strings in JSON** (e.g., "1234.56")"
2. **contracts/purchases-openapi.yaml**: Update schema `type: string` (not `type: number`) for amountUsd, convertedAmount
3. **quickstart.md**: Add example curl commands showing string amount format

---

### C7: Error Code Set for Validation Failures
**Status**: Partially in spec.md (RATE_NOT_FOUND documented); **Pending**: spec.md Edge Cases, contracts/purchases-openapi.yaml  
**Answer**: Full set:
- `VALIDATION_ERROR` (generic validation)
- `INVALID_DATE` (invalid or future date)
- `NEGATIVE_AMOUNT` (≤ 0 or non-numeric)
- `DESCRIPTION_TOO_LONG` (> 50 chars)
- `MISSING_FIELD` (required field null/absent)
- `RATE_NOT_FOUND` (no rate within 6 months)
- `NOT_FOUND` (purchase ID doesn't exist → 404)

**Action**: Update documents:
1. **spec.md** Edge Cases: Add detailed error code mapping with examples
2. **contracts/purchases-openapi.yaml**: Add error response examples for each code

---

### C8: Date Format for transactionDate
**Status**: Partially documented; **Pending**: spec.md, contracts/purchases-openapi.yaml, quickstart.md  
**Answer**: ISO 8601 date only (YYYY-MM-DD); no time component

**Action**: Update documents:
1. **spec.md** Assumptions: Add "Date format: ISO 8601 date only (YYYY-MM-DD); no time component"
2. **spec.md** Edge Cases: Add "Future dates rejected with INVALID_DATE error"
3. **contracts/purchases-openapi.yaml**: Update `transactionDate` schema to `type: string, format: date`
4. **quickstart.md**: Add example dates in curl commands

---

### C9: Currency Validation Requirements
**Status**: Not documented; **Pending**: spec.md, data-model.md  
**Answer**: ISO 4217 3-letter codes; case-insensitive input validation; stored as uppercase

**Action**: Update documents:
1. **spec.md** Assumptions: Add "Currency codes: ISO 4217 3-letter codes (EUR, GBP, USD, JPY, etc.); case-insensitive on input, stored as uppercase"
2. **spec.md** Edge Cases: Add validation error for invalid currency codes
3. **data-model.md**: Update ExchangeRate.currency type documentation
4. **contracts/purchases-openapi.yaml**: Add currency parameter validation schema (3-letter uppercase)

---

### C10: Amount Value Limits
**Status**: Not documented; **Pending**: spec.md Assumptions, data-model.md  
**Answer**: Use Go math/big for arbitrary precision; no hard upper limit specified

**Action**: Update documents:
1. **spec.md** Assumptions: Add "Monetary precision: Use arbitrary precision (Go math/big) for cents representation. No hard upper limit specified."
2. **data-model.md** Validation Rules: Update to reflect arbitrary precision instead of "must be a positive integer within storage limits"

---

### C11: Audit Timestamps in Purchases Table
**Status**: Not documented; **Pending**: data-model.md  
**Answer**: Add created_at, updated_at to purchases table for audit trail

**Action**: Update documents:
1. **data-model.md** Entities: Add to Purchase entity:
   - `created_at` (timestamp, auto-set on insert)
   - `updated_at` (timestamp, auto-set on insert & update)
2. **spec.md** Assumptions: Document audit trail requirement

---

### C12: Treasury Adapter Integration Pattern
**Status**: Mentioned but not detailed; **Pending**: plan.md, research.md  
**Answer**: Document adapter factory/config pattern for swapping mock provider (testing) ↔ real API (production)

**Action**: Update documents:
1. **plan.md**: Add new section "Adapter Integration Pattern" describing:
   - Factory interface for selecting provider (mock for testing vs. real Treasury for production)
   - Configuration via environment variable (TREASURY_PROVIDER=real by default, use mock for testing only)
   - Example: `internal/ports/provider_factory.go`
2. **research.md**: Update Treasury Rates Adapter decision section to include factory pattern

---

### C13: Exchange Rate Bulk Load Strategy
**Status**: Not documented; **Pending**: research.md, quickstart.md  
**Answer**: Adapter is responsible; initialization strategy not specified

**Action**: Update documents:
1. **research.md**: Add note: "Adapter responsible for populating exchange_rates table; initialization/sync strategy not specified in this spec."
2. **quickstart.md**: Add note: "Exchange rates must be pre-populated in database or provided by adapter before API can perform conversions"

---

### C14: X-Request-ID Response Header
**Status**: Partially mentioned; **Pending**: spec.md, plan.md, contracts/purchases-openapi.yaml  
**Answer**: Include X-Request-ID in response headers if client provided it in request

**Action**: Update documents:
1. **spec.md** Assumptions: Update observability section to specify "X-Request-ID included in response headers if provided by client"
2. **spec.md** Edge Cases: Add request header requirements to header section
3. **plan.md**: Update Design Decisions to include response header handling
4. **contracts/purchases-openapi.yaml**: Add response headers documentation (X-Request-ID if applicable)

---

## Integration Timeline

1. **Immediate** (blocking spec readiness): C5, C6, C7, C8, C9, C10 (response/validation clarity)
2. **Pre-implementation** (blocking task generation): C11, C12, C13, C14 (schema/integration/observability)

## Document Update Checklist

- [ ] spec.md: Add C8 & C9 to Assumptions section; Add C7 & C9 details to Edge Cases
- [ ] data-model.md: Update Purchase entity (add created_at, updated_at); Update validation rules (C10)
- [ ] contracts/purchases-openapi.yaml: Add full response schemas (C5); Update types to string for amounts (C6); Add error code examples (C7); Add parameter validation (C9); Add X-Request-ID headers
- [ ] plan.md: Add "Adapter Integration Pattern" section (C12) with factory/config pattern
- [ ] research.md: Update Treasury Adapter decision section (C12, C13)
- [ ] quickstart.md: Add curl examples with string amounts (C6); Add environment variable section; Add adapter initialization note (C13)

---

**Status**: All 9 clarifications ready for implementation. Awaiting confirmation to apply changes across all documents.
