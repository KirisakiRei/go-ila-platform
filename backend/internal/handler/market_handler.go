package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	stream  *market.StreamHub
}

// NewMarketHandler merangkai handler data pasar.
func NewMarketHandler(candles *market.Service, pairs repository.PairRepository) *MarketHandler {
	return &MarketHandler{candles: candles, pairs: pairs}
}

// WithStream memasang hub live stream candle secara opsional agar pemanggil
// lama tetap kompatibel.
func (h *MarketHandler) WithStream(stream *market.StreamHub) *MarketHandler {
	h.stream = stream
	return h
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

// StreamCandles menangani GET /api/v1/market/candles/stream melalui Server-Sent
// Events (SSE). Satu koneksi upstream dipakai bersama banyak pengguna dan
// diputus otomatis saat tidak ada chart live yang menontonnya.
func (h *MarketHandler) StreamCandles(c *gin.Context) {
	if h.stream == nil {
		httpx.Error(c, http.StatusServiceUnavailable, "STREAM_UNAVAILABLE", "Live stream tidak tersedia", nil)
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(c.Query("symbol")))
	if symbol == "" {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Simbol wajib diisi", nil)
		return
	}

	timeframe := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("tf", "H4")))
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

	updates, usedFeed, unsubscribe, err := h.stream.Subscribe(pair.Symbol, feed, timeframe)
	if err != nil {
		if errors.Is(err, market.ErrStreamCapacity) || errors.Is(err, market.ErrStreamClosed) {
			httpx.Error(c, http.StatusServiceUnavailable, "STREAM_BUSY", "Kapasitas live stream penuh, beralih ke penyegaran berkala", nil)
			return
		}
		c.Error(err)
		return
	}
	defer unsubscribe()

	rc := http.NewResponseController(c.Writer)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	_ = rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
	readyMeta, _ := json.Marshal(map[string]string{
		"symbol":    pair.Symbol,
		"feed":      usedFeed,
		"timeframe": timeframe,
	})
	if _, wErr := fmt.Fprintf(c.Writer, "event: ready\ndata: %s\n\n", readyMeta); wErr != nil {
		return
	}
	c.Writer.Flush()

	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	ctx := c.Request.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pingTicker.C:
			_ = rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if _, wErr := fmt.Fprint(c.Writer, ": ping\n\n"); wErr != nil {
				return
			}
			c.Writer.Flush()
		case bar, ok := <-updates:
			if !ok {
				return
			}
			payload, mErr := json.Marshal(bar)
			if mErr != nil {
				continue
			}
			_ = rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
			if _, wErr := fmt.Fprintf(c.Writer, "event: candle\ndata: %s\n\n", payload); wErr != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}
