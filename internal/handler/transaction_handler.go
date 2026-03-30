package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"alx-wallet/internal/repository"
)

// TransactionHandler handles transaction history routes.
type TransactionHandler struct {
	repo *repository.TransactionRepo
	log  *zap.Logger
}

func NewTransactionHandler(repo *repository.TransactionRepo, log *zap.Logger) *TransactionHandler {
	return &TransactionHandler{repo: repo, log: log}
}

// RegisterRoutes adds transaction routes to the engine.
func (h *TransactionHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/v1")
	v1.GET("/accounts/:user_id/transactions", h.ListTransactions)
}

// transactionResponse is the JSON shape for a single ledger entry.
type transactionResponse struct {
	JournalID   string `json:"journal_id"`
	ReferenceID string `json:"reference_id"`
	Description string `json:"description"`
	Amount      int64  `json:"amount"`
	EntryType   string `json:"entry_type"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// listTransactionsResponse wraps results with pagination metadata.
type listTransactionsResponse struct {
	Data   []transactionResponse `json:"data"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
	Count  int                   `json:"count"`
}

// ListTransactions handles GET /v1/accounts/:user_id/transactions
//
// Query params:
//
//	limit  (default 20, max 100)
//	offset (default 0)
//
// Example:
//
//	GET /v1/accounts/uuid/transactions?limit=10&offset=0
func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id must be a valid UUID"})
		return
	}

	limit := parseIntQuery(c, "limit", 20)
	offset := parseIntQuery(c, "offset", 0)

	records, err := h.repo.ListByUserID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.log.Error("list transactions failed",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch transactions"})
		return
	}

	resp := make([]transactionResponse, 0, len(records))
	for _, r := range records {
		resp = append(resp, transactionResponse{
			JournalID:   r.JournalID.String(),
			ReferenceID: r.ReferenceID,
			Description: r.Description,
			Amount:      int64(r.Amount),
			EntryType:   string(r.EntryType),
			Status:      string(r.Status),
			CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, listTransactionsResponse{
		Data:   resp,
		Limit:  limit,
		Offset: offset,
		Count:  len(resp),
	})
}

func parseIntQuery(c *gin.Context, key string, fallback int) int {
	if v := c.Query(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i >= 0 {
			return i
		}
	}
	return fallback
}
