package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerRepository struct {
	db *pgxpool.Pool
}

func NewLedgerRepository(db *pgxpool.Pool) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) GetBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	var balance float64
	err := r.db.QueryRow(ctx, `
		SELECT balance
		FROM accounts
		WHERE user_id = $1
	`, userID).Scan(&balance)
	// err := r.db.QueryRow(ctx, `
	// 	SELECT COALESCE(SUM(
	// 		CASE
	// 			WHEN entry_type = 'credit' THEN amount
	// 			WHEN entry_type = 'debit' THEN -amount
	// 		END
	// 	),0)
	// 	FROM ledger_entries
	// 	WHERE account_id = $1
	// `, accountID).Scan(&balance)
	return balance, err
}

func (r *LedgerRepository) Transfer(ctx context.Context, from, to uuid.UUID, amount float64, refID string) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	journalID := uuid.New()

	_, err = tx.Exec(ctx,
		`INSERT INTO journal_entries (id, reference_id, description)
		 VALUES ($1,$2,'transfer')`,
		journalID, refID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (id,journal_id,account_id,amount,entry_type)
		 VALUES ($1,$2,$3,$4,'debit')`,
		uuid.New(), journalID, from, amount)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE accounts SET balance = balance - $1 WHERE user_id = $2`,
		amount, from)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO ledger_entries (id,journal_id,account_id,amount,entry_type)
		 VALUES ($1,$2,$3,$4,'credit')`,
		uuid.New(), journalID, to, amount)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE accounts SET balance = balance + $1 WHERE user_id = $2`,
		amount, to)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
