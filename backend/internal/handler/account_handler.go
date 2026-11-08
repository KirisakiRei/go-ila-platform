package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// AccountHandler adalah pintu HTTP untuk akun broker dan langganan simbol.
type AccountHandler struct {
	accounts service.AccountService
}

// NewAccountHandler merangkai handler akun.
func NewAccountHandler(accounts service.AccountService) *AccountHandler {
	return &AccountHandler{accounts: accounts}
}

// List menangani GET /api/v1/accounts.
func (h *AccountHandler) List(c *gin.Context) {
	result, err := h.accounts.List(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar akun broker", result)
}

// Create menangani POST /api/v1/accounts.
func (h *AccountHandler) Create(c *gin.Context) {
	var req dto.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.accounts.Create(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateCredential) {
			c.Error(httpx.WrapValidation("token ini sudah dipakai akun broker lain; setiap akun memakai token berbeda"))
			return
		}
		c.Error(err)
		return
	}
	httpx.Created(c, "Akun broker terhubung", result)
}

// Update menangani PATCH /api/v1/accounts/:id.
func (h *AccountHandler) Update(c *gin.Context) {
	var req dto.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.accounts.Update(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req)
	if err != nil {
		if errors.Is(err, service.ErrDuplicateCredential) {
			c.Error(httpx.WrapValidation("token ini sudah dipakai akun broker lain; setiap akun memakai token berbeda"))
			return
		}
		c.Error(err)
		return
	}
	httpx.OK(c, "Akun broker diperbarui", result)
}

// Delete menangani DELETE /api/v1/accounts/:id.
func (h *AccountHandler) Delete(c *gin.Context) {
	if err := h.accounts.Delete(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Akun broker diputus dan kredensial dihapus", nil)
}

// ListSubscriptions menangani GET /api/v1/subscriptions.
func (h *AccountHandler) ListSubscriptions(c *gin.Context) {
	result, err := h.accounts.ListSubscriptions(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar simbol dipantau", result)
}

// Subscribe menangani POST /api/v1/subscriptions.
func (h *AccountHandler) Subscribe(c *gin.Context) {
	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.accounts.Subscribe(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Simbol dipantau", result)
}

// Unsubscribe menangani DELETE /api/v1/subscriptions/:id.
func (h *AccountHandler) Unsubscribe(c *gin.Context) {
	if err := h.accounts.Unsubscribe(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Simbol berhenti dipantau", nil)
}

// Strategies menangani GET /api/v1/strategies.
func (h *AccountHandler) Strategies(c *gin.Context) {
	result, err := h.accounts.Strategies(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Ketersediaan strategi", result)
}

// CTraderAuthURL menangani GET /api/v1/accounts/ctrader/auth-url.
func (h *AccountHandler) CTraderAuthURL(c *gin.Context) {
	result, err := h.accounts.GetCTraderAuthURL(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Tautan otorisasi cTrader", result)
}

// CTraderCallback menangani POST /api/v1/accounts/ctrader/callback.
func (h *AccountHandler) CTraderCallback(c *gin.Context) {
	var req dto.CTraderCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.accounts.ConnectCTrader(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Akun cTrader berhasil dihubungkan", result)
}
