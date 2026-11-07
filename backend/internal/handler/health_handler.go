package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/pkg/httpx"
)

// HealthHandler menyediakan penanda hidup dan kesiapan proses.
type HealthHandler struct {
	serviceName string
	readyCheck  func() error
}

// NewHealthHandler merangkai handler kesehatan.
// readyCheck boleh nil bila pemeriksaan dependensi belum disiapkan.
func NewHealthHandler(serviceName string, readyCheck func() error) *HealthHandler {
	return &HealthHandler{serviceName: serviceName, readyCheck: readyCheck}
}

// Live menangani GET /health: proses hidup.
func (h *HealthHandler) Live(c *gin.Context) {
	httpx.OK(c, "ok", gin.H{"service": h.serviceName, "status": "alive"})
}

// Ready menangani GET /ready: dependensi siap menerima pekerjaan.
func (h *HealthHandler) Ready(c *gin.Context) {
	if h.readyCheck != nil {
		if err := h.readyCheck(); err != nil {
			c.Error(err)
			return
		}
	}

	httpx.OK(c, "ok", gin.H{"service": h.serviceName, "status": "ready"})
}
