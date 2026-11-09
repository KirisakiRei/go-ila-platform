package handler

import (
	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// RiskHandler melayani simulasi risiko. Endpoint ini adalah satu-satunya tempat
// hitungan lot dibuat, sehingga pratinjau di dashboard dan eksekusi Runner
// memakai rumus yang sama.
type RiskHandler struct {
	sim service.RiskSimulator
}

// NewRiskHandler merangkai handler simulasi risiko.
func NewRiskHandler(sim service.RiskSimulator) *RiskHandler {
	return &RiskHandler{sim: sim}
}

// Register memasang route simulasi risiko pada grup sesi pengguna.
func (h *RiskHandler) Register(me *gin.RouterGroup) {
	me.POST("/calc/risk-sim", h.Simulate)
}

// Simulate menangani POST /api/v1/calc/risk-sim.
func (h *RiskHandler) Simulate(c *gin.Context) {
	var req dto.RiskSimRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.sim.SimulateRisk(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Simulasi risiko", result)
}
