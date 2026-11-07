package handler

import (
	"github.com/gin-gonic/gin"
	"io"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// BillingHandler adalah pintu HTTP tagihan pengguna dan webhook gateway.
type BillingHandler struct {
	billing service.BillingService
}

// NewBillingHandler merangkai handler billing.
func NewBillingHandler(billing service.BillingService) *BillingHandler {
	return &BillingHandler{billing: billing}
}

// RegisterUser memasang route billing untuk pengguna yang sudah masuk.
func (h *BillingHandler) RegisterUser(g *gin.RouterGroup) {
	g.POST("/billing/checkout", h.Checkout)
	g.GET("/billing/invoices", h.Invoices)
	g.GET("/billing/subscription", h.Subscription)
	g.POST("/billing/credit/redeem", h.RedeemCredit)
}

// Checkout menangani POST /api/v1/billing/checkout.
func (h *BillingHandler) Checkout(c *gin.Context) {
	var req dto.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.billing.Checkout(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Tagihan dibuat", result)
}

// Invoices menangani GET /api/v1/billing/invoices.
func (h *BillingHandler) Invoices(c *gin.Context) {
	items, err := h.billing.Invoices(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Riwayat tagihan", items)
}

// Subscription menangani GET /api/v1/billing/subscription.
func (h *BillingHandler) Subscription(c *gin.Context) {
	item, err := h.billing.Subscription(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Langganan aktif", item)
}

// RedeemCredit menangani POST /api/v1/billing/credit/redeem.
func (h *BillingHandler) RedeemCredit(c *gin.Context) {
	var req dto.RedeemCreditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.billing.RedeemCredit(c.Request.Context(), middleware.CurrentUserID(c), req.Code)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Saldo kredit diperbarui", result)
}

// Webhook menangani POST /api/v1/webhook/payment.
// Diautentikasi tanda tangan gateway, bukan sesi pengguna.
func (h *BillingHandler) Webhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		c.Error(httpx.WrapValidation("isi webhook tidak terbaca"))
		return
	}

	header := make(map[string]string, len(c.Request.Header))
	for name := range c.Request.Header {
		header[name] = c.Request.Header.Get(name)
	}

	if err := h.billing.HandleWebhook(c.Request.Context(), header, body); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Webhook diproses", nil)
}
