package handler

import (
	"alx-wallet/internal/domain"
	"alx-wallet/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type WalletHandler struct {
	service *service.WalletService
}

func NewWalletHandler(s *service.WalletService) *WalletHandler {
	return &WalletHandler{s}
}

func (h *WalletHandler) CreateAccount(c *gin.Context) {
	var req domain.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	id, err := h.service.CreateAccount(c, req.Name, req.Balance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": id, "name": req.Name, "balance": req.Balance, "type": "user"})
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	balance, _ := h.service.GetBalance(c, id)
	c.JSON(http.StatusOK, gin.H{"balance": balance})
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	var req domain.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	err := h.service.Transfer(c, req.FromUserID, req.ToUserID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
