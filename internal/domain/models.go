package domain

import "github.com/google/uuid"

type Account struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Type   string
}

type TransferRequest struct {
	FromAccountID uuid.UUID `json:"from_account_id"`
	ToAccountID   uuid.UUID `json:"to_account_id"`
	Amount        float64   `json:"amount"`
}
