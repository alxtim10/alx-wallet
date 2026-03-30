package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"alx-wallet/internal/domain"
)

// TransactionRecord is a denormalised view of a ledger entry for the history API.
type TransactionRecord struct {
	JournalID   uuid.UUID
	ReferenceID string
	Description string
	Amount      domain.Money
	EntryType   domain.EntryType
	Status      domain.TransactionStatus
	CreatedAt   time.Time
}

// TransactionRepo provides read-only queries for the transaction history API.
// It is separate from LedgerRepo to keep write and read paths cleanly separated.
type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

// ListByUserID returns paginated ledger entries for a user's wallet account,
// ordered by most recent first.
//
// limit:  max rows per page  (capped at 100)
// offset: rows to skip       (for page-based pagination)
func (r *TransactionRepo) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]TransactionRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := r.db.Query(ctx, `
		SELECT
		    j.id          AS journal_id,
		    j.reference_id,
		    j.description,
		    l.amount,
		    l.entry_type,
		    j.status,
		    l.created_at
		FROM ledger_entries  l
		JOIN journal_entries j ON j.id = l.journal_id
		JOIN accounts        a ON a.id = l.account_id
		WHERE a.user_id = $1
		  AND a.type    = 'wallet'
		ORDER BY l.created_at DESC
		LIMIT  $2
		OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	defer rows.Close()

	var records []TransactionRecord
	for rows.Next() {
		var rec TransactionRecord
		if err := rows.Scan(
			&rec.JournalID,
			&rec.ReferenceID,
			&rec.Description,
			&rec.Amount,
			&rec.EntryType,
			&rec.Status,
			&rec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}
