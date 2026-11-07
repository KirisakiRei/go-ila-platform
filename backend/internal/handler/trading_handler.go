package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/middleware"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/httpx"
)

// TradingHandler menyajikan sinyal dan posisi untuk dashboard.
type TradingHandler struct {
	trading service.TradingService
}

// NewTradingHandler merangkai handler tampilan trading.
func NewTradingHandler(trading service.TradingService) *TradingHandler {
	return &TradingHandler{trading: trading}
}

// Signals menangani GET /api/v1/signals.
func (h *TradingHandler) Signals(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	result, err := h.trading.Signals(c.Request.Context(), middleware.CurrentUserID(c), limit)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar sinyal", result)
}

// Positions menangani GET /api/v1/positions.
func (h *TradingHandler) Positions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	result, err := h.trading.Positions(c.Request.Context(), middleware.CurrentUserID(c), limit)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar posisi", result)
}
