package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// BacktestHandler melayani backtest ILA 2.0 atas data historis.
//
// Seluruh endpoint di sini menempel pada grup sesi pengguna, dan setiap operasi
// yang menyentuh satu run memeriksa pemiliknya di service. Tidak ada endpoint
// yang menerima ID run tanpa membandingkannya dengan sesi.
type BacktestHandler struct {
	backtest *service.BacktestService
}

// NewBacktestHandler merangkai handler backtest.
func NewBacktestHandler(backtest *service.BacktestService) *BacktestHandler {
	return &BacktestHandler{backtest: backtest}
}

// Register memasang seluruh route backtest pada grup sesi pengguna.
func (h *BacktestHandler) Register(me *gin.RouterGroup) {
	grup := me.Group("/backtest")
	grup.GET("/coverage", h.Cakupan)
	grup.GET("/quota", h.Kuota)
	grup.GET("/history", h.Riwayat)
	grup.POST("", h.Buat)
	grup.GET("/:id", h.Ambil)
	grup.DELETE("/:id", h.Hapus)
	grup.GET("/:id/trades", h.TradeDaftar)
	grup.GET("/:id/trades/:seq", h.TradeAmbil)
}

// Cakupan menangani GET /api/v1/backtest/coverage.
func (h *BacktestHandler) Cakupan(c *gin.Context) {
	periksaUlang := c.Query("periksa_ulang") == "1" || c.Query("periksa_ulang") == "true"
	hasil, err := h.backtest.Cakupan(c.Request.Context(), middleware.CurrentUserID(c),
		c.Query("symbol"), c.Query("feed"), periksaUlang)
	if err != nil {
		_ = httpx.Fail(c, err)
		return
	}
	httpx.OK(c, "Cakupan data historis", hasil)
}

// Kuota menangani GET /api/v1/backtest/quota.
func (h *BacktestHandler) Kuota(c *gin.Context) {
	hasil, err := h.backtest.Kuota(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		_ = httpx.Fail(c, err)
		return
	}
	httpx.OK(c, "Sisa jatah backtest", hasil)
}

// Riwayat menangani GET /api/v1/backtest/history.
func (h *BacktestHandler) Riwayat(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	daftar, total, err := h.backtest.Riwayat(c.Request.Context(), middleware.CurrentUserID(c), limit, offset)
	if err != nil {
		_ = httpx.Fail(c, err)
		return
	}
	httpx.OKWithMeta(c, "Riwayat backtest", daftar, gin.H{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Buat menangani POST /api/v1/backtest.
func (h *BacktestHandler) Buat(c *gin.Context) {
	var req dto.BacktestCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = httpx.Fail(c, httpx.WrapValidation(err.Error()))
		return
	}

	hasil, err := h.backtest.Buat(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		_ = httpx.Fail(c, err)
		return
	}
	httpx.Created(c, "Backtest dimasukkan ke antrean", hasil)
}

// Ambil menangani GET /api/v1/backtest/:id.
func (h *BacktestHandler) Ambil(c *gin.Context) {
	hasil, err := h.backtest.Ambil(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		_ = httpx.Fail(c, err)
		return
	}
	httpx.OK(c, "Detail backtest", hasil)
}

// Hapus menangani DELETE /api/v1/backtest/:id.
func (h *BacktestHandler) Hapus(c *gin.Context) {
	if err := h.backtest.Hapus(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		_ = httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c, "Backtest dihapus")
}

// TradeDaftar menangani GET /api/v1/backtest/:id/trades.
func (h *BacktestHandler) TradeDaftar(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	daftar, total, err := h.backtest.TradeDaftar(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), limit, offset)
	if err != nil {
		_ = httpx.Fail(c, err)
		return
	}
	httpx.OKWithMeta(c, "Daftar entry", daftar, gin.H{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// TradeAmbil menangani GET /api/v1/backtest/:id/trades/:seq.
func (h *BacktestHandler) TradeAmbil(c *gin.Context) {
	seq, err := strconv.Atoi(c.Param("seq"))
	if err != nil || seq <= 0 {
		_ = httpx.Fail(c, httpx.WrapValidation("nomor entry tidak sah"))
		return
	}

	hasil, err := h.backtest.TradeAmbil(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), seq)
	if err != nil {
		_ = httpx.Fail(c, err)
		return
	}
	httpx.OK(c, "Detail entry", hasil)
}
