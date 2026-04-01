-- =============================================================================
-- alx-wallet: Full Schema
-- Double-entry ledger with idempotency, account_id references, and status tracking
-- =============================================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- accounts
-- One user can own multiple accounts of different types (wallet, escrow, system).
-- =============================================================================
CREATE TABLE IF NOT EXISTS accounts (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username   VARCHAR(20) NOT NULL,
    type       VARCHAR(20) NOT NULL CHECK (type IN ('wallet', 'escrow', 'system')),
    balance     BIGINT      NOT NULL CHECK (balance >= 0),
    password TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One account of each type per user
    CONSTRAINT uq_accounts_username_type UNIQUE (username, type)
);

CREATE INDEX IF NOT EXISTS idx_accounts_username ON accounts(username);

-- =============================================================================
-- journal_entries
-- One logical transaction per row. The reference_id is the idempotency key.
-- =============================================================================
CREATE TABLE IF NOT EXISTS journal_entries (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id VARCHAR(128) NOT NULL,
    description  TEXT,
    status       VARCHAR(20) NOT NULL DEFAULT 'completed'
                             CHECK (status IN ('pending', 'completed', 'failed')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Idempotency: the same reference_id can never be inserted twice.
    CONSTRAINT uq_journal_reference_id UNIQUE (reference_id)
);

CREATE INDEX IF NOT EXISTS idx_journal_created_at ON journal_entries(created_at DESC);

-- =============================================================================
-- ledger_entries
-- Two rows per journal_entry: one debit, one credit.
-- References account_id (NOT user_id) to correctly support multi-account users.
-- =============================================================================
CREATE TABLE IF NOT EXISTS ledger_entries (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id UUID        NOT NULL REFERENCES journal_entries(id) ON DELETE RESTRICT,
    account_id UUID        NOT NULL REFERENCES accounts(id)        ON DELETE RESTRICT,
    amount     BIGINT      NOT NULL CHECK (amount > 0),
    entry_type VARCHAR(10) NOT NULL CHECK (entry_type IN ('debit', 'credit')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_account_id  ON ledger_entries(account_id);
CREATE INDEX IF NOT EXISTS idx_ledger_journal_id  ON ledger_entries(journal_id);
CREATE INDEX IF NOT EXISTS idx_ledger_created_at  ON ledger_entries(created_at DESC);

-- =============================================================================
-- Integrity view: verify debit = credit for every journal entry.
-- Run this in monitoring/reconciliation jobs:
--   SELECT * FROM v_journal_integrity WHERE is_balanced = false;
-- =============================================================================
CREATE OR REPLACE VIEW v_journal_integrity AS
SELECT
    j.id            AS journal_id,
    j.reference_id,
    j.status,
    SUM(CASE WHEN l.entry_type = 'debit'  THEN l.amount ELSE 0 END) AS total_debits,
    SUM(CASE WHEN l.entry_type = 'credit' THEN l.amount ELSE 0 END) AS total_credits,
    SUM(CASE WHEN l.entry_type = 'debit'  THEN l.amount ELSE 0 END) =
    SUM(CASE WHEN l.entry_type = 'credit' THEN l.amount ELSE 0 END)  AS is_balanced
FROM journal_entries j
JOIN ledger_entries  l ON l.journal_id = j.id
GROUP BY j.id, j.reference_id, j.status;