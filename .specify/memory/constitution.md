<!--
Sync Impact Report

Version change: none → 1.0.0
Modified principles: placeholder tokens replaced with concrete principles focused on DDD, Financial Precision, Testing, ACID, Observability
Added sections: Additional Constraints & Security Requirements; Development Workflow & Quality Gates
Removed sections: none
Templates requiring updates: ⚠ .specify/templates/plan-template.md (pending review)
					   ⚠ .specify/templates/spec-template.md (pending review)
					   ⚠ .specify/templates/tasks-template.md (pending review)
Follow-up TODOs: none
-->

# Purchase API Constitution

## Core Principles

### I. Domain-Driven Design & Hexagonal Architecture (NON-NEGOTIABLE)
The system MUST be organized around a clear domain model for purchases. Implement a hexagonal (ports-and-adapters)
architecture: core domain and application logic must have no framework or infrastructure dependencies. All
external interactions (HTTP, DB, currency provider) MUST be via well-defined ports and adapters so the domain
is independently testable and replaceable.

### II. Financial Precision & Safety (NON-NEGOTIABLE)
All monetary values MUST use exact, fixed-point arithmetic. Use integer cents or a language native arbitrary-precision
decimal type (e.g., `decimal` in C#) for storage and calculations. Rounding rules MUST be explicit: round to the
nearest cent using bankers rounding only where domain analysis requires; otherwise use half-away-from-zero.
Validation: descriptions ≤ 50 characters; transaction dates must be valid ISO 8601 dates; purchase amounts must be
positive and stored with cent precision.

### III. Test-First and Automated Quality Gates (NON-NEGOTIABLE)
Development MUST be driven by automated tests. Unit tests MUST cover domain invariants and validation. Integration
tests MUST cover persistence, currency conversion flows, and API contracts. Contract tests are REQUIRED for
external API expectations (e.g., exchange-rate provider). CI pipelines MUST block merges on failing tests.

### IV. Data Integrity & ACID Guarantees
Persistence MUST be implemented with an ACID-compliant datastore. Transactions MUST be used to maintain
consistency for multi-step operations. The default production recommendation is PostgreSQL; lightweight
embedded databases (SQLite) are acceptable for local development if transactional semantics are preserved.
Migrations, schema versioning, and durable backups are REQUIRED for production readiness.

### V. Observability, Simplicity & Security
Prefer simple, auditable designs. Provide structured logs for domain events and errors, and expose health and
metrics endpoints for operational observability. Secure all external endpoints with TLS, validate inputs, and
minimize sensitive data storage. Secrets management and least-privilege access are REQUIRED for production.

## Additional Constraints & Security Requirements

- Encryption in transit (TLS) is REQUIRED for all external traffic.
- Store only the minimum necessary personal data; avoid storing raw payment instrument data.
- Use parameterized queries or an ORM that prevents injection vulnerabilities.
- Protect secrets (API keys, DB credentials) via environment-based secret stores.
- Ensure GDPR/PII controls are considered if applicable; otherwise document why not applicable.

## Development Workflow, Review Process & Quality Gates

- Branching: feature branches per work item; PRs for all changes; require at least one approving reviewer.
- Tests: unit + integration + contract tests required for merged changes. CI must run tests and linters.
- Code style: follow language/community conventions; run automatic formatters in CI.
- Deployments: require passing CI, migration plan, and an approved rollout strategy for production releases.

## Governance

The constitution defines mandatory engineering practices for the Purchase API. Amendments MUST be documented
and include a rationale and migration plan. Non-normative guidance (examples, templates) may be updated freely,
but changes to core principles require a MINOR or MAJOR version bump depending on impact.

**Version**: 1.0.0 | **Ratified**: 2026-06-17 | **Last Amended**: 2026-06-17

