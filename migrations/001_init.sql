CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    type VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    balance NUMERIC(20,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE journal_entries (
    id UUID PRIMARY KEY,
    reference_id VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE ledger_entries (
    id UUID PRIMARY KEY,
    journal_id UUID REFERENCES journal_entries(id),
    user_id UUID REFERENCES accounts(id),
    amount NUMERIC(20,2) NOT NULL CHECK (amount > 0),
    entry_type VARCHAR(10) CHECK (entry_type IN ('debit','credit')),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_ledger_account ON ledger_entries(user_id);