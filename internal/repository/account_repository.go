// Package repository implements domain.AccountRepository and domain.LedgerRepository
// against PostgreSQL using pgx/v5.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"alx-wallet/internal/domain"
)

const pgErrUniqueViolation = "23505"

// AccountRepo implements domain.AccountRepository backed by PostgreSQL.
type AccountRepo struct {
	db *pgxpool.Pool
}

func NewAccountRepo(db *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{db: db}
}

// Create inserts a new account and returns the fully populated domain model.
func (r *AccountRepo) Create(ctx context.Context, userID uuid.UUID, accountType domain.AccountType) (*domain.Account, error) {
	acc := &domain.Account{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      accountType,
		CreatedAt: time.Now().UTC(),
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO accounts (id, user_id, type, created_at)
		 VALUES ($1, $2, $3, $4)`,
		acc.ID, acc.UserID, acc.Type, acc.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("account create: %w", err)
	}
	return acc, nil
}

// GetByID fetches a single account by its primary key.
func (r *AccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, type, created_at FROM accounts WHERE id = $1`,
		id,
	)
	return scanAccount(row)
}

// GetByUserID fetches a user's account of a specific type.
func (r *AccountRepo) GetByUserID(ctx context.Context, userID uuid.UUID, accountType domain.AccountType) (*domain.Account, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, user_id, type, created_at
		 FROM   accounts
		 WHERE  user_id = $1 AND type = $2
		 LIMIT  1`,
		userID, accountType,
	)
	return scanAccount(row)
}

func scanAccount(row pgx.Row) (*domain.Account, error) {
	var a domain.Account
	err := row.Scan(&a.ID, &a.UserID, &a.Type, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("scan account: %w", err)
	}
	return &a, nil
}

// isPgUniqueViolation checks if a pgconn error is a unique constraint violation.
func isPgUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation
}
