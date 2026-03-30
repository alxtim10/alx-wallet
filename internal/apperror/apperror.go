// Package apperror defines structured API errors so that no raw DB or
// internal error messages are ever leaked to HTTP clients.
package apperror

import (
	"errors"
	"net/http"

	"alx-wallet/internal/domain"
)

// AppError is a structured error that carries both a user-safe message
// and an HTTP status code.
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
}

func (e *AppError) Error() string { return e.Message }

// New creates an AppError.
func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// FromDomain maps domain sentinel errors to AppErrors with correct HTTP codes.
// Any unmapped error becomes a 500 with a generic message (never leaking internals).
func FromDomain(err error) *AppError {
	switch {
	case errors.Is(err, domain.ErrAccountNotFound):
		return New(http.StatusNotFound, "account not found")
	case errors.Is(err, domain.ErrInsufficientFunds):
		return New(http.StatusUnprocessableEntity, "insufficient funds")
	case errors.Is(err, domain.ErrDuplicateTransfer):
		return New(http.StatusConflict, "a transfer with this reference_id already exists")
	case errors.Is(err, domain.ErrSameAccount):
		return New(http.StatusBadRequest, "cannot transfer to the same account")
	case errors.Is(err, domain.ErrInvalidAmount):
		return New(http.StatusBadRequest, "amount must be greater than zero")
	case errors.Is(err, domain.ErrInvalidAccountType):
		return New(http.StatusBadRequest, "invalid account type; valid values: wallet, escrow, system")
	default:
		return New(http.StatusInternalServerError, "an internal error occurred")
	}
}
