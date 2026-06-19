CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS purchases (
    id UUID PRIMARY KEY,
    description VARCHAR(50) NOT NULL,
    transaction_date DATE NOT NULL,
    amount_usd_cents BIGINT NOT NULL CHECK (amount_usd_cents > 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX purchases_created_at ON purchases(created_at);
