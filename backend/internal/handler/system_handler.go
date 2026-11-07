package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/middleware"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/httpx"
)

// SystemHandler menyajikan status pasar, engine, broker, dan kuota harian.
type SystemHandler struct {
	system service.SystemService
}

// NewSystemHandler merangkai handler status sistem.
func NewSystemHandler(system service.SystemService) *SystemHandler {
	return &SystemHandler{system: system}
}

// Status menangani GET /api/v1/system/status.
func (h *SystemHandler) Status(c *gin.Context) {
	result, err := h.system.Status(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Status sistem", result)
}
