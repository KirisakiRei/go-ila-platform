package handler

import (
	"github.com/gin-gonic/gin"

	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// SessionHandler menampilkan dan mencabut perangkat yang sedang masuk.
type SessionHandler struct {
	sessions         service.SessionService
	currentSessionID func(*gin.Context) string
}

// NewSessionHandler merangkai handler sesi.
// currentSessionID membaca ID sesi dari klaim token perangkat ini.
func NewSessionHandler(sessions service.SessionService, currentSessionID func(*gin.Context) string) *SessionHandler {
	return &SessionHandler{sessions: sessions, currentSessionID: currentSessionID}
}

// List menangani GET /api/v1/me/sessions.
func (h *SessionHandler) List(c *gin.Context) {
	items, err := h.sessions.List(
		c.Request.Context(),
		middleware.CurrentUserID(c),
		"user",
		h.currentSessionID(c),
	)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar sesi aktif", items)
}

// Revoke menangani DELETE /api/v1/me/sessions/:id.
func (h *SessionHandler) Revoke(c *gin.Context) {
	if err := h.sessions.Revoke(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Sesi dicabut", gin.H{"revoked": true})
}
