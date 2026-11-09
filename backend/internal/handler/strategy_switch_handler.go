package handler

import (
	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// StrategySwitchHandler melayani saklar metode analisis per akun dan per pair.
//
// Halaman Sinyal memakai matriks ini untuk menampilkan saklar tiap akun pada
// satu pair, sementara daftar pair pantauan tetap global.
type StrategySwitchHandler struct {
	switches service.StrategySwitchService
}

// NewStrategySwitchHandler merangkai handler saklar metode.
func NewStrategySwitchHandler(switches service.StrategySwitchService) *StrategySwitchHandler {
	return &StrategySwitchHandler{switches: switches}
}

// Register memasang route saklar metode pada grup sesi pengguna.
func (h *StrategySwitchHandler) Register(me *gin.RouterGroup) {
	me.GET("/pairs/:symbol/strategies", h.Matrix)
	me.PUT("/accounts/:id/pairs/:symbol/strategies", h.Set)
}

// Matrix menangani GET /api/v1/pairs/:symbol/strategies.
func (h *StrategySwitchHandler) Matrix(c *gin.Context) {
	result, err := h.switches.Matrix(c.Request.Context(), middleware.CurrentUserID(c), c.Param("symbol"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Saklar metode per akun", result)
}

// Set menangani PUT /api/v1/accounts/:id/pairs/:symbol/strategies.
func (h *StrategySwitchHandler) Set(c *gin.Context) {
	var req dto.UpdateStrategySwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.switches.Set(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), c.Param("symbol"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Saklar metode diperbarui", result)
}
