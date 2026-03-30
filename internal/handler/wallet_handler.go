// Package handler contains the HTTP layer — request parsing, validation,
// and mapping domain errors to appropriate HTTP responses.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"alx-wallet/internal/apperror"
	"alx-wallet/internal/domain"
)

// WalletHandler handles all wallet HTTP routes.
type WalletHandler struct {
	svc domain.WalletService
	log *zap.Logger
}

func NewWalletHandler(svc domain.WalletService, log *zap.Logger) *WalletHandler {
	return &WalletHandler{svc: svc, log: log}
}

// RegisterRoutes wires all routes onto the given engine.
func (h *WalletHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")
	{
		v1.POST("/accounts", h.CreateAccount)
		v1.GET("/accounts/:user_id/balance", h.GetBalance)
		v1.POST("/transfers", h.Transfer)
	}
}

// ── POST /v1/accounts ────────────────────────────────────────────────────────

type createAccountRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
	Type   string `json:"type"    binding:"required,oneof=wallet escrow system"`
}

type createAccountResponse struct {
	AccountID string `json:"account_id"`
	UserID    string `json:"user_id"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

func (h *WalletHandler) CreateAccount(c *gin.Context) {
	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID) // already validated by binding tag

	acc, err := h.svc.CreateAccount(c.Request.Context(), userID, domain.AccountType(req.Type))
	if err != nil {
		appErr := apperror.FromDomain(err)
		c.JSON(appErr.Code, appErr)
		return
	}

	c.JSON(http.StatusCreated, createAccountResponse{
		AccountID: acc.ID.String(),
		UserID:    acc.UserID.String(),
		Type:      string(acc.Type),
		CreatedAt: acc.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// ── GET /v1/accounts/:user_id/balance ────────────────────────────────────────

type balanceResponse struct {
	UserID  string `json:"user_id"`
	Balance int64  `json:"balance"`
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid UUID"})
		return
	}

	balance, err := h.svc.GetBalance(c.Request.Context(), userID)
	if err != nil {
		appErr := apperror.FromDomain(err)
		c.JSON(appErr.Code, appErr)
		return
	}

	c.JSON(http.StatusOK, balanceResponse{
		UserID:  userID.String(),
		Balance: int64(balance),
	})
}

// ── POST /v1/transfers ───────────────────────────────────────────────────────

type transferRequest struct {
	FromUserID  string `json:"from_user_id"  binding:"required,uuid"`
	ToUserID    string `json:"to_user_id"    binding:"required,uuid"`
	Amount      int64  `json:"amount"        binding:"required,min=1"`
	ReferenceID string `json:"reference_id"  binding:"required,min=1,max=128"`
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	var req transferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fromID, _ := uuid.Parse(req.FromUserID)
	toID, _ := uuid.Parse(req.ToUserID)

	err := h.svc.Transfer(c.Request.Context(), domain.TransferRequest{
		FromUserID:  fromID,
		ToUserID:    toID,
		Amount:      domain.Money(req.Amount),
		ReferenceID: req.ReferenceID,
	})
	if err != nil {
		// Log unexpected errors at error level; expected domain errors at warn.
		var appErr *apperror.AppError
		mapped := apperror.FromDomain(err)
		if mapped.Code >= 500 {
			h.log.Error("transfer failed", zap.Error(err))
		} else {
			h.log.Warn("transfer rejected", zap.Error(err))
		}
		_ = errors.As(err, &appErr)
		c.JSON(mapped.Code, mapped)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
