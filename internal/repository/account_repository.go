package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccountRepository struct {
	db *pgxpool.Pool
}

func NewAccountRepository(db *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Create(ctx context.Context, id, userID uuid.UUID, accType string, name string, balance float64) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO accounts (id, user_id, type, name, balance) VALUES ($1,$2,$3,$4,$5)`,
		id, userID, accType, name, balance)
	return err
}
