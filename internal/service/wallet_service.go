// Package service implements the business logic for the wallet system.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"alx-wallet/internal/domain"
)

const balanceCacheTTL = 30 * time.Second

// WalletSvc implements domain.WalletService.
type WalletSvc struct {
	accounts domain.AccountRepository
	ledger   domain.LedgerRepository
	cache    *redis.Client
	log      *zap.Logger
}

func NewWalletService(
	accounts domain.AccountRepository,
	ledger domain.LedgerRepository,
	cache *redis.Client,
	log *zap.Logger,
) *WalletSvc {
	return &WalletSvc{
		accounts: accounts,
		ledger:   ledger,
		cache:    cache,
		log:      log,
	}
}

func (s *WalletSvc) GetAllAccounts(ctx context.Context) ([]*domain.Account, error) {
	accounts, err := s.accounts.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

// CreateAccount creates a new wallet account for the given user.
func (s *WalletSvc) CreateAccount(ctx context.Context, username string, accountType domain.AccountType, password string) (*domain.Account, error) {
	if !accountType.IsValid() {
		return nil, domain.ErrInvalidAccountType
	}

	acc, err := s.accounts.Create(ctx, username, accountType, password)
	if err != nil {
		s.log.Error("failed to create account", zap.String("username", username), zap.Error(err))
		return nil, err
	}

	s.log.Info("account created",
		zap.String("account_id", acc.ID.String()),
		zap.String("username", username),
		zap.String("type", string(accountType)),
	)
	return acc, nil
}

// TopUp top up the balance for a user's wallet account.
func (s *WalletSvc) TopUp(ctx context.Context, username string, balance domain.Money) (*domain.Account, error) {
	if !balance.IsPositive() {
		return nil, domain.ErrInvalidAmount
	}
	acc, err := s.accounts.TopUp(ctx, username, balance)
	if err != nil {
		s.log.Error("failed to top up account", zap.String("username", username), zap.Error(err))
		return nil, err
	}

	return acc, nil
}

// GetBalance returns the current balance for a user's wallet account.
// It uses a short-lived Redis cache to avoid hitting Postgres on every read.
func (s *WalletSvc) GetBalance(ctx context.Context, username string) (domain.Money, error) {
	acc, err := s.accounts.GetByUsername(ctx, username)
	if err != nil {
		return 0, err
	}

	// Try cache first.
	cacheKey := balanceCacheKey(acc.ID)
	if cached, err := s.cache.Get(ctx, cacheKey).Int64(); err == nil {
		s.log.Debug("balance cache hit", zap.String("id", acc.ID.String()))
		return domain.Money(cached), nil
	}

	balance, err := s.ledger.GetBalance(ctx, acc.ID)
	if err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}

	// Populate cache — best effort, don't fail on cache errors.
	if err := s.cache.Set(ctx, cacheKey, int64(balance), balanceCacheTTL).Err(); err != nil {
		s.log.Warn("failed to set balance cache", zap.Error(err))
	}

	return balance, nil
}

// Transfer moves funds from one user to another atomically.
//
// Guarantees:
//  1. Amount must be positive.
//  2. Sender and receiver must be different users.
//  3. Sender must have sufficient funds (read-before-write guard).
//  4. reference_id is idempotent — duplicate submissions return ErrDuplicateTransfer.
//  5. The double-entry write is atomic: both sides commit or neither does.
func (s *WalletSvc) Transfer(ctx context.Context, req domain.TransferRequest) error {
	// ── Validation ────────────────────────────────────────────────────────────
	if !req.Amount.IsPositive() {
		return domain.ErrInvalidAmount
	}
	if req.FromUserID == req.ToUserID {
		return domain.ErrSameAccount
	}

	// ── Resolve accounts ──────────────────────────────────────────────────────
	fromAcc, err := s.accounts.GetByID(ctx, req.FromUserID)
	if err != nil {
		return fmt.Errorf("sender: %w", err)
	}
	toAcc, err := s.accounts.GetByID(ctx, req.ToUserID)
	if err != nil {
		return fmt.Errorf("receiver: %w", err)
	}

	// ── Balance check (FIX 2) ─────────────────────────────────────────────────
	// Read the live balance from the DB (not cache) to avoid stale reads
	// during high-concurrency scenarios before writing.
	balance, err := s.ledger.GetBalance(ctx, fromAcc.ID)
	if err != nil {
		return fmt.Errorf("check balance: %w", err)
	}
	if balance < req.Amount {
		s.log.Warn("insufficient funds",
			zap.String("from_account", fromAcc.ID.String()),
			zap.Int64("balance", int64(balance)),
			zap.Int64("requested", int64(req.Amount)),
		)
		return domain.ErrInsufficientFunds
	}

	// ── Atomic double-entry (FIX 1 + FIX 3) ──────────────────────────────────
	err = s.ledger.RecordTransfer(ctx, domain.RecordTransferRequest{
		ReferenceID:   req.ReferenceID,
		Description:   fmt.Sprintf("transfer %s → %s", req.FromUserID, req.ToUserID),
		FromAccountID: fromAcc.ID,
		ToAccountID:   toAcc.ID,
		Amount:        req.Amount,
	})
	if err != nil {
		return err // domain.ErrDuplicateTransfer bubbles up as-is
	}

	// ── Invalidate balance cache for both accounts ────────────────────────────
	s.cache.Del(ctx, balanceCacheKey(fromAcc.ID), balanceCacheKey(toAcc.ID))

	s.log.Info("transfer recorded",
		zap.String("reference_id", req.ReferenceID),
		zap.String("from", req.FromUserID.String()),
		zap.String("to", req.ToUserID.String()),
		zap.Int64("amount", int64(req.Amount)),
	)
	return nil
}

func balanceCacheKey(accountID uuid.UUID) string {
	return fmt.Sprintf("balance:%s", accountID)
}
