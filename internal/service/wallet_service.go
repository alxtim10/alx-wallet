package service

import (
	"alx-wallet/internal/repository"
	"context"

	"github.com/google/uuid"
)

type WalletService struct {
	accountRepo *repository.AccountRepository
	ledgerRepo  *repository.LedgerRepository
}

func NewWalletService(a *repository.AccountRepository, l *repository.LedgerRepository) *WalletService {
	return &WalletService{a, l}
}

func (s *WalletService) CreateAccount(ctx context.Context, name string, balance float64) (uuid.UUID, error) {
	id := uuid.New()
	return id, s.accountRepo.Create(ctx, id, id, "user", name, balance)
}

func (s *WalletService) GetBalance(ctx context.Context, accountID uuid.UUID) (float64, error) {
	return s.ledgerRepo.GetBalance(ctx, accountID)
}

func (s *WalletService) Transfer(ctx context.Context, from, to uuid.UUID, amount float64) error {
	return s.ledgerRepo.Transfer(ctx, from, to, amount, uuid.New().String())
}
