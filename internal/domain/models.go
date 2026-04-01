// Package domain contains the core business types and interfaces for the wallet system.
// Nothing in this package imports from infrastructure (DB, HTTP, Redis).
// This is the innermost layer of clean architecture.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ─── Sentinel Errors ─────────────────────────────────────────────────────────

var (
	ErrAccountNotFound    = errors.New("account not found")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrDuplicateTransfer  = errors.New("transfer with this reference_id already exists")
	ErrSameAccount        = errors.New("cannot transfer to the same account")
	ErrInvalidAmount      = errors.New("amount must be greater than zero")
	ErrInvalidAccountType = errors.New("invalid account type")
	ErrNegativeBalance    = errors.New("balance cannot be negative")
)

// ─── Value Types ──────────────────────────────────────────────────────────────

// Money represents an amount in the smallest currency unit (e.g. cents).
// Using int64 avoids floating-point rounding errors in financial calculations.
type Money int64

func (m Money) IsPositive() bool { return m > 0 }
func (m Money) IsZero() bool     { return m == 0 }

// AccountType represents the class of a wallet account.
type AccountType string

const (
	AccountTypeWallet AccountType = "wallet"
	AccountTypeEscrow AccountType = "escrow"
	AccountTypeSystem AccountType = "system"
)

func (t AccountType) IsValid() bool {
	switch t {
	case AccountTypeWallet, AccountTypeEscrow, AccountTypeSystem:
		return true
	}
	return false
}

// EntryType is either a debit or credit on a ledger entry.
type EntryType string

const (
	EntryTypeDebit  EntryType = "debit"
	EntryTypeCredit EntryType = "credit"
)

// TransactionStatus tracks the lifecycle of a journal entry.
type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"
	StatusCompleted TransactionStatus = "completed"
	StatusFailed    TransactionStatus = "failed"
)

// ─── Domain Models ───────────────────────────────────────────────────────────

// Account represents a wallet account. One user can have multiple accounts
// of different types (wallet, escrow, system).
type Account struct {
	ID           uuid.UUID
	Username     string
	Type         AccountType
	Balance      Money
	PasswordHash string `db:"password" json:"-"`
	CreatedAt    time.Time
}

// JournalEntry is the logical record of a financial transaction.
// Every transfer, deposit, or withdrawal has exactly one journal entry.
type JournalEntry struct {
	ID          uuid.UUID
	ReferenceID string // idempotency key — must be globally unique
	Description string
	Status      TransactionStatus
	CreatedAt   time.Time
}

// LedgerEntry is one side of a double-entry movement.
// Every JournalEntry produces exactly two LedgerEntries (one debit, one credit).
type LedgerEntry struct {
	ID        uuid.UUID
	JournalID uuid.UUID
	AccountID uuid.UUID // references accounts.id — NOT user_id
	Amount    Money
	EntryType EntryType
	CreatedAt time.Time
}

// ─── Repository Interfaces ───────────────────────────────────────────────────
// These are the ports that the service layer depends on.
// Infrastructure (postgres, redis) implements these interfaces.

type AccountRepository interface {
	Create(ctx context.Context, username string, accountType AccountType, password string) (*Account, error)
	TopUp(ctx context.Context, username string, balance Money) (*Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Account, error)
	GetByUsername(ctx context.Context, username string) (*Account, error)
}

type LedgerRepository interface {
	// GetBalance returns the live balance for an account derived from ledger entries.
	GetBalance(ctx context.Context, accountID uuid.UUID) (Money, error)

	// RecordTransfer atomically inserts a journal entry + two ledger entries
	// inside a single database transaction.
	RecordTransfer(ctx context.Context, req RecordTransferRequest) error
}

// RecordTransferRequest is the input DTO for the ledger repository.
type RecordTransferRequest struct {
	ReferenceID   string
	Description   string
	FromAccountID uuid.UUID
	ToAccountID   uuid.UUID
	Amount        Money
}

// ─── Service Interface ───────────────────────────────────────────────────────

type WalletService interface {
	CreateAccount(ctx context.Context, username string, accountType AccountType, password string) (*Account, error)
	TopUp(ctx context.Context, username string, balance Money) (*Account, error)
	GetBalance(ctx context.Context, username string) (Money, error)
	Transfer(ctx context.Context, req TransferRequest) error
}

// TransferRequest is the input DTO for a transfer operation.
type TransferRequest struct {
	FromUserID  uuid.UUID
	ToUserID    uuid.UUID
	Amount      Money
	ReferenceID string
}
