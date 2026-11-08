package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/middleware"
	"ila/backend/internal/runner"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
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

// ExportPositions menangani GET /api/v1/positions/export.
// Rentang tanggal memakai zona hari akuntansi yang sama dengan Runner.
func (h *TradingHandler) ExportPositions(c *gin.Context) {
	from := parseDayParam(c.Query("from"))
	to := parseDayParam(c.Query("to"))
	if !from.IsZero() && !to.IsZero() && to.Before(from) {
		c.Error(httpx.WrapValidation("tanggal akhir harus setelah tanggal mulai"))
		return
	}

	result, err := h.trading.ExportPositions(c.Request.Context(), middleware.CurrentUserID(c), from, to)
	if err != nil {
		c.Error(err)
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+result.Filename+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", result.Content)
}

// parseDayParam membaca tanggal YYYY-MM-DD pada zona hari akuntansi.
// Nilai tak sah diabaikan agar ekspor tetap berjalan tanpa filter.
func parseDayParam(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	day, err := time.ParseInLocation("2006-01-02", value, runner.WIB)
	if err != nil {
		return time.Time{}
	}
	return day
}
