package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/market"
	"ila/backend/internal/repository"
	"ila/backend/pkg/httpx"
)

// MarketHandler menyajikan data candle untuk chart dashboard.
//
// Feed yang dipakai selalu feed milik pair yang bersangkutan, bukan pilihan
// dari klien: chart harus membaca lilin yang sama dengan yang dievaluasi engine.
type MarketHandler struct {
	candles *market.Service
	pairs   repository.PairRepository
}

// NewMarketHandler merangkai handler data pasar.
func NewMarketHandler(candles *market.Service, pairs repository.PairRepository) *MarketHandler {
	return &MarketHandler{candles: candles, pairs: pairs}
}

// CandlesResponse adalah bentuk balasan endpoint candle.
type CandlesResponse struct {
	Symbol    string          `json:"symbol"`
	Feed      string          `json:"feed"`
	Timeframe string          `json:"timeframe"`
	Candles   []market.Candle `json:"candles"`
}

// Candles menangani GET /api/v1/market/candles.
//
// Simbol wajib ada di katalog pair agar endpoint tidak bisa dipakai menarik
// instrumen apa pun di luar yang dikelola admin.
func (h *MarketHandler) Candles(c *gin.Context) {
	symbol := strings.ToUpper(strings.TrimSpace(c.Query("symbol")))
	if symbol == "" {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Simbol wajib diisi", nil)
		return
	}

	timeframe := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("tf", "H4")))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))

	if market.MapResolution(timeframe) == "" {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Timeframe tidak didukung", nil)
		return
	}

	pair, err := h.pairs.FindPairBySymbol(c.Request.Context(), symbol)
	if err != nil {
		c.Error(err)
		return
	}

	feed := strings.TrimSpace(c.Query("feed"))
	if feed == "" {
		feed = pair.FeedSource
	}

	candles, usedFeed, err := h.candles.Candles(c.Request.Context(), pair.Symbol, feed, timeframe, limit)
	if err != nil {
		// Kegagalan feed bukan kesalahan klien: diteruskan ke ErrorHandler agar
		// dicatat sebagai kesalahan internal dan tidak membocorkan detail.
		c.Error(err)
		return
	}

	httpx.OK(c, "Candle pasar", CandlesResponse{
		Symbol:    pair.Symbol,
		Feed:      usedFeed,
		Timeframe: timeframe,
		Candles:   candles,
	})
}
