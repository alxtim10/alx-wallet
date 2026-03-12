package domain

import "github.com/google/uuid"

type Account struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Name    string
	Balance float64
	Type    string
}

type TransferRequest struct {
	FromUserID uuid.UUID `json:"from_user_id"`
	ToUserID   uuid.UUID `json:"to_user_id"`
	Amount     float64   `json:"amount"`
}

type CreateAccountRequest struct {
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}
