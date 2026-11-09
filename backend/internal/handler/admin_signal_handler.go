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

// AdminSignalHandler adalah pintu HTTP pembaca sinyal untuk admin.
type AdminSignalHandler struct {
	signals service.AdminSignalService
}

// NewAdminSignalHandler merangkai handler sinyal admin.
func NewAdminSignalHandler(signals service.AdminSignalService) *AdminSignalHandler {
	return &AdminSignalHandler{signals: signals}
}

// Register memasang route sinyal admin.
func (h *AdminSignalHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/signals", guard(http.MethodGet, "/signals", authz.SignalView), h.List)
	g.GET("/signals/:id/trace", guard(http.MethodGet, "/signals/:id/trace", authz.SignalTrace), h.Trace)
}

func (h *AdminSignalHandler) actor(c *gin.Context) service.AdminActor {
	return service.AdminActor{
		ID:        middleware.CurrentUserID(c),
		RequestID: c.GetString(middleware.ContextRequestID),
	}
}

// List menangani GET /api/v1/admin/signals.
func (h *AdminSignalHandler) List(c *gin.Context) {
	var q dto.AdminSignalFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	items, meta, err := h.signals.List(c.Request.Context(), h.actor(c), q)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OKWithMeta(c, "Daftar sinyal", items, meta)
}

// Trace menangani GET /api/v1/admin/signals/:id/trace.
func (h *AdminSignalHandler) Trace(c *gin.Context) {
	result, err := h.signals.Trace(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Perjalanan sinyal", result)
}
