package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/middleware"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/httpx"
)

// TrialHandler melayani aktivasi dan status uji coba.
type TrialHandler struct {
	trial service.TrialService
}

// NewTrialHandler merangkai handler uji coba.
func NewTrialHandler(trial service.TrialService) *TrialHandler {
	return &TrialHandler{trial: trial}
}

// Status menangani GET /api/v1/trial.
func (h *TrialHandler) Status(c *gin.Context) {
	result, err := h.trial.Status(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Status uji coba", result)
}

// Start menangani POST /api/v1/trial.
func (h *TrialHandler) Start(c *gin.Context) {
	result, err := h.trial.Start(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Uji coba 7 hari dimulai", result)
}
