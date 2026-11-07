package handler

import (
	"github.com/gin-gonic/gin"

	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// PricingHandler menyajikan paket publik dan perhitungan harga.
type PricingHandler struct {
	pricing service.PricingService
}

// NewPricingHandler merangkai handler harga.
func NewPricingHandler(pricing service.PricingService) *PricingHandler {
	return &PricingHandler{pricing: pricing}
}

// Plans menangani GET /api/v1/billing/plans (publik).
func (h *PricingHandler) Plans(c *gin.Context) {
	plans, err := h.pricing.Plans(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar paket", plans)
}
