package handler

import (
	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// TimeframeSwitchHandler melayani pengaturan pasangan timeframe per akun.
//
// Pasangan timeframe itu sendiri milik aturan trading ILA (pakem engine);
// endpoint ini hanya mencatat mana yang dipantau dan mana yang dieksekusi
// otomatis.
type TimeframeSwitchHandler struct {
	switches service.TimeframeSwitchService
}

// NewTimeframeSwitchHandler merangkai handler saklar timeframe.
func NewTimeframeSwitchHandler(switches service.TimeframeSwitchService) *TimeframeSwitchHandler {
	return &TimeframeSwitchHandler{switches: switches}
}

// Register memasang route saklar timeframe pada grup sesi pengguna.
func (h *TimeframeSwitchHandler) Register(me *gin.RouterGroup) {
	me.GET("/pairs/:symbol/timeframes", h.Matrix)
	me.PUT("/accounts/:id/pairs/:symbol/timeframes", h.Set)
}

// Matrix menangani GET /api/v1/pairs/:symbol/timeframes.
func (h *TimeframeSwitchHandler) Matrix(c *gin.Context) {
	result, err := h.switches.Matrix(c.Request.Context(), middleware.CurrentUserID(c), c.Param("symbol"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Saklar timeframe per akun", result)
}

// Set menangani PUT /api/v1/accounts/:id/pairs/:symbol/timeframes.
func (h *TimeframeSwitchHandler) Set(c *gin.Context) {
	var req dto.UpdateTimeframeSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.switches.Set(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), c.Param("symbol"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Saklar timeframe diperbarui", result)
}
