package handler

import (
	"github.com/gin-gonic/gin"

	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// PricingHandler menyajikan paket publik dan perhitungan harga.
type PricingHandler struct {
	pricing service.PricingService
	// gate nil berarti daftar paket tetap terbuka.
	gate service.AccessGate
}

// NewPricingHandler merangkai handler harga.
func NewPricingHandler(pricing service.PricingService) *PricingHandler {
	return &PricingHandler{pricing: pricing}
}

// SetAccessGate memasang penjaga mode undangan.
func (h *PricingHandler) SetAccessGate(gate service.AccessGate) {
	h.gate = gate
}

// Plans menangani GET /api/v1/billing/plans (publik).
func (h *PricingHandler) Plans(c *gin.Context) {
	// Selama mode undangan tidak ada yang bisa dibeli, jadi daftar paket
	// tidak perlu ditawarkan. Halaman yang masih memintanya menerima daftar
	// kosong — lebih jujur daripada 404 yang membuat UI tampak rusak.
	if h.gate != nil && h.gate.InviteOnly(c.Request.Context()) {
		httpx.OK(c, "Daftar paket", []any{})
		return
	}

	plans, err := h.pricing.Plans(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar paket", plans)
}
