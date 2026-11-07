package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// AdminBillingHandler adalah pintu HTTP pengelolaan uang.
type AdminBillingHandler struct {
	billing service.AdminBillingService
}

// NewAdminBillingHandler merangkai handler uang.
func NewAdminBillingHandler(billing service.AdminBillingService) *AdminBillingHandler {
	return &AdminBillingHandler{billing: billing}
}

func (h *AdminBillingHandler) actor(c *gin.Context) service.AdminActor {
	return service.AdminActor{
		ID:        middleware.CurrentUserID(c),
		RequestID: c.GetString(middleware.ContextRequestID),
	}
}

// Register memasang route paket, kupon, dan kredit.
func (h *AdminBillingHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/plans", guard(http.MethodGet, "/plans", authz.BillingView), h.Plans)
	g.PATCH("/plans/:code", guard(http.MethodPatch, "/plans/:code", authz.BillingPlan), h.UpdatePlan)
	g.PUT("/plans/:code/guard", guard(http.MethodPut, "/plans/:code/guard", authz.BillingPlan), h.SetGuard)

	g.GET("/coupons", guard(http.MethodGet, "/coupons", authz.BillingView), h.Coupons)
	g.POST("/coupons", guard(http.MethodPost, "/coupons", authz.BillingVoucher), h.CreateCoupon)
	g.PATCH("/coupons/:id", guard(http.MethodPatch, "/coupons/:id", authz.BillingVoucher), h.UpdateCoupon)
	g.POST("/coupons/:id/disable", guard(http.MethodPost, "/coupons/:id/disable", authz.BillingVoucher), h.DisableCoupon)

	g.GET("/users/:id/credit", guard(http.MethodGet, "/users/:id/credit", authz.BillingView), h.Credit)
	g.POST("/users/:id/credit", guard(http.MethodPost, "/users/:id/credit", authz.BillingVoucher), h.AdjustCredit)
}

// Plans menangani GET /api/v1/admin/plans.
func (h *AdminBillingHandler) Plans(c *gin.Context) {
	plans, err := h.billing.Plans(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar paket", plans)
}

// UpdatePlan menangani PATCH /api/v1/admin/plans/:code.
func (h *AdminBillingHandler) UpdatePlan(c *gin.Context) {
	var req dto.PlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.billing.UpdatePlan(c.Request.Context(), h.actor(c), c.Param("code"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Paket diperbarui", nil)
}

// SetGuard menangani PUT /api/v1/admin/plans/:code/guard.
func (h *AdminBillingHandler) SetGuard(c *gin.Context) {
	var req dto.GuardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.billing.SetGuard(c.Request.Context(), h.actor(c), c.Param("code"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Batas bawah harga diperbarui", nil)
}

// Coupons menangani GET /api/v1/admin/coupons.
func (h *AdminBillingHandler) Coupons(c *gin.Context) {
	items, err := h.billing.Coupons(c.Request.Context(), c.Query("status"), c.Query("q"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar kupon", items)
}

// CreateCoupon menangani POST /api/v1/admin/coupons.
func (h *AdminBillingHandler) CreateCoupon(c *gin.Context) {
	var req dto.CouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	item, err := h.billing.CreateCoupon(c.Request.Context(), h.actor(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Kupon dibuat", item)
}

// UpdateCoupon menangani PATCH /api/v1/admin/coupons/:id.
func (h *AdminBillingHandler) UpdateCoupon(c *gin.Context) {
	var req dto.CouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.billing.UpdateCoupon(c.Request.Context(), h.actor(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kupon diperbarui", nil)
}

// DisableCoupon menangani POST /api/v1/admin/coupons/:id/disable.
func (h *AdminBillingHandler) DisableCoupon(c *gin.Context) {
	var req struct {
		Reason string `json:"reason" binding:"max=500"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.billing.DisableCoupon(c.Request.Context(), h.actor(c), c.Param("id"), req.Reason); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kupon dihentikan", nil)
}

// Credit menangani GET /api/v1/admin/users/:id/credit.
func (h *AdminBillingHandler) Credit(c *gin.Context) {
	summary, err := h.billing.CreditSummary(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Saldo kredit", summary)
}

// AdjustCredit menangani POST /api/v1/admin/users/:id/credit.
func (h *AdminBillingHandler) AdjustCredit(c *gin.Context) {
	var req dto.CreditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	summary, err := h.billing.AdjustCredit(c.Request.Context(), h.actor(c), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kredit disesuaikan", summary)
}
