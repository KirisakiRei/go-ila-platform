package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// AdminAuditHandler adalah pintu HTTP jejak audit.
type AdminAuditHandler struct {
	audit service.AdminAuditService
}

// NewAdminAuditHandler merangkai handler audit.
func NewAdminAuditHandler(audit service.AdminAuditService) *AdminAuditHandler {
	return &AdminAuditHandler{audit: audit}
}

// Register memasang route audit. Tidak ada route hapus: tabel append-only.
func (h *AdminAuditHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/audit", guard(http.MethodGet, "/audit", authz.AuditView), h.List)
	g.GET("/audit/:id", guard(http.MethodGet, "/audit/:id", authz.AuditView), h.Detail)
	g.GET("/audit-catalog", guard(http.MethodGet, "/audit-catalog", authz.AuditView), h.Catalog)
}

// List menangani GET /api/v1/admin/audit.
func (h *AdminAuditHandler) List(c *gin.Context) {
	var q dto.AdminAuditFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	q.ActorID = c.Query("actor_id")
	q.Action = c.Query("action")
	q.TargetType = c.Query("target_type")
	q.TargetID = c.Query("target_id")

	items, meta, err := h.audit.List(c.Request.Context(), q)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OKWithMeta(c, "Jejak audit", items, meta)
}

// Detail menangani GET /api/v1/admin/audit/:id.
func (h *AdminAuditHandler) Detail(c *gin.Context) {
	result, err := h.audit.Detail(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Detail audit", result)
}

// Catalog menangani GET /api/v1/admin/audit-catalog.
func (h *AdminAuditHandler) Catalog(c *gin.Context) {
	httpx.OK(c, "Katalog aksi audit", h.audit.Catalog())
}
