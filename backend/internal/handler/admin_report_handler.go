package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// AdminReportHandler adalah pintu HTTP laporan bisnis dan operasional.
type AdminReportHandler struct {
	reports service.AdminReportService
}

// NewAdminReportHandler merangkai handler laporan.
func NewAdminReportHandler(reports service.AdminReportService) *AdminReportHandler {
	return &AdminReportHandler{reports: reports}
}

// Register memasang route laporan.
func (h *AdminReportHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/reports/revenue", guard(http.MethodGet, "/reports/revenue", authz.BillingView), h.Revenue)
	g.GET("/reports/customers", guard(http.MethodGet, "/reports/customers", authz.BillingView), h.Customers)
	g.GET("/reports/coupons", guard(http.MethodGet, "/reports/coupons", authz.BillingView), h.Coupons)
	g.GET("/reports/operations", guard(http.MethodGet, "/reports/operations", authz.SystemView), h.Operations)
}

// Revenue menangani GET /api/v1/admin/reports/revenue.
func (h *AdminReportHandler) Revenue(c *gin.Context) {
	result, err := h.reports.Revenue(c.Request.Context(), c.Query("from"), c.Query("to"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Laporan pendapatan", result)
}

// Customers menangani GET /api/v1/admin/reports/customers.
func (h *AdminReportHandler) Customers(c *gin.Context) {
	result, err := h.reports.Customers(c.Request.Context(), c.Query("from"), c.Query("to"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Laporan pelanggan", result)
}

// Coupons menangani GET /api/v1/admin/reports/coupons.
func (h *AdminReportHandler) Coupons(c *gin.Context) {
	result, err := h.reports.Coupons(c.Request.Context(), c.Query("from"), c.Query("to"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Laporan kupon", result)
}

// Operations menangani GET /api/v1/admin/reports/operations.
func (h *AdminReportHandler) Operations(c *gin.Context) {
	result, err := h.reports.Operations(c.Request.Context(), c.Query("from"), c.Query("to"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Laporan operasional", result)
}
