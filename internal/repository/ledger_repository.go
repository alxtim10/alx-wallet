package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"alx-wallet/internal/domain"
)

// LedgerRepo implements domain.LedgerRepository backed by PostgreSQL.
type LedgerRepo struct {
	db *pgxpool.Pool
}

func NewLedgerRepo(db *pgxpool.Pool) *LedgerRepo {
	return &LedgerRepo{db: db}
}

// GetBalance computes the live balance for an account from the ledger.
//
// Balance = SUM(credit amounts) - SUM(debit amounts)
//
// Using COALESCE guarantees we return 0 for accounts with no entries yet.
func (r *LedgerRepo) GetBalance(ctx context.Context, accountID uuid.UUID) (domain.Money, error) {
	var balance int64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(
		    SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE 0 END) -
		    SUM(CASE WHEN entry_type = 'debit'  THEN amount ELSE 0 END),
		 0)
		 FROM ledger_entries
		 WHERE account_id = $1`,
		accountID,
	).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}
	return domain.Money(balance), nil
}

// RecordTransfer executes an atomic double-entry transfer:
//
//	BEGIN (SERIALIZABLE)
//	  INSERT journal_entries   ← unique constraint on reference_id (idempotency)
//	  INSERT ledger_entries    (debit  from_account)
//	  INSERT ledger_entries    (credit to_account)
//	COMMIT
//
// If any step fails the whole transaction is rolled back.
// A duplicate reference_id raises domain.ErrDuplicateTransfer.
func (r *LedgerRepo) RecordTransfer(ctx context.Context, req domain.RecordTransferRequest) error {
	return pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	}, func(tx pgx.Tx) error {
		journalID := uuid.New()
		now := time.Now().UTC()

		// 1. Insert journal entry — unique constraint on reference_id fires here.
		_, err := tx.Exec(ctx,
			`INSERT INTO journal_entries (id, reference_id, description, status, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			journalID, req.ReferenceID, req.Description, domain.StatusCompleted, now,
		)
		if err != nil {
			if isPgUniqueViolation(err) {
				return domain.ErrDuplicateTransfer
			}
			return fmt.Errorf("insert journal entry: %w", err)
		}

		// 2. Debit the sender.
		_, err = tx.Exec(ctx,
			`INSERT INTO ledger_entries (id, journal_id, account_id, amount, entry_type, created_at)
			 VALUES ($1, $2, $3, $4, 'debit', $5)`,
			uuid.New(), journalID, req.FromAccountID, int64(req.Amount), now,
		)
		if err != nil {
			return fmt.Errorf("insert debit entry: %w", err)
		}

		// 3. Credit the receiver.
		_, err = tx.Exec(ctx,
			`INSERT INTO ledger_entries (id, journal_id, account_id, amount, entry_type, created_at)
			 VALUES ($1, $2, $3, $4, 'credit', $5)`,
			uuid.New(), journalID, req.ToAccountID, int64(req.Amount), now,
		)
		if err != nil {
			return fmt.Errorf("insert credit entry: %w", err)
		}

		return nil
	})
}
