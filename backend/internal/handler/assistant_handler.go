package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/assistant"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// AssistantHandler menangani endpoint ILA AI Assistant untuk dashboard user.
type AssistantHandler struct {
	svc service.AssistantService
}

// NewAssistantHandler membuat handler ILA AI Assistant.
func NewAssistantHandler(svc service.AssistantService) *AssistantHandler {
	return &AssistantHandler{svc: svc}
}

// Register memasang route asisten pada grup terautentikasi user.
func (h *AssistantHandler) Register(me *gin.RouterGroup, chatLimiter gin.HandlerFunc) {
	group := me.Group("/assistant")
	group.GET("/session", h.GetSession)
	group.POST("/compact", h.Compact)
	if chatLimiter != nil {
		group.POST("/chat", chatLimiter, h.Chat)
	} else {
		group.POST("/chat", h.Chat)
	}
}

// GetSession menangani GET /api/v1/assistant/session.
func (h *AssistantHandler) GetSession(c *gin.Context) {
	if h.svc == nil {
		httpx.OK(c, "Status asisten", &service.AssistantSessionView{
			Enabled:  false,
			Messages: []service.AssistantMessageView{},
		})
		return
	}
	beforeSeq, _ := strconv.ParseInt(c.Query("before_seq"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	res, err := h.svc.GetSession(c.Request.Context(), middleware.CurrentUserID(c), beforeSeq, limit)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Sesi ILA AI", res)
}

// Compact menangani POST /api/v1/assistant/compact.
func (h *AssistantHandler) Compact(c *gin.Context) {
	if h.svc == nil || !h.svc.Enabled() {
		httpx.Error(c, http.StatusServiceUnavailable, "ASSISTANT_UNAVAILABLE", "ILA AI belum diaktifkan.", nil)
		return
	}
	res, err := h.svc.CompactSession(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Konteks percakapan berhasil diringkas", res)
}

type chatRequestBody struct {
	Message string `json:"message" binding:"required,min=1,max=2000"`
}

// Chat menangani POST /api/v1/assistant/chat dengan output SSE streaming.
func (h *AssistantHandler) Chat(c *gin.Context) {
	if h.svc == nil || !h.svc.Enabled() {
		httpx.Error(c, http.StatusServiceUnavailable, "ASSISTANT_UNAVAILABLE", "ILA AI belum dikonfigurasi oleh admin.", nil)
		return
	}

	var req chatRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Pesan tidak valid (maksimal 2000 karakter).", nil)
		return
	}
	msg := strings.TrimSpace(req.Message)
	if msg == "" {
		httpx.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Pesan tidak boleh kosong.", nil)
		return
	}

	rc := http.NewResponseController(c.Writer)
	// Perpanjang deadline melebihi WriteTimeout server HTTP (30s) agar proses
	// compaction + thinking + tool calling tidak terputus di tengah jalan.
	_ = rc.SetWriteDeadline(time.Now().Add(120 * time.Second))

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	writeEvent := func(ev assistant.StreamEvent) error {
		_ = rc.SetWriteDeadline(time.Now().Add(60 * time.Second))
		payload, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Type, payload); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	err := h.svc.ChatStream(c.Request.Context(), middleware.CurrentUserID(c), msg, writeEvent)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		_ = writeEvent(assistant.StreamEvent{
			Type:    "error",
			Code:    "ASSISTANT_ERROR",
			Message: sanitizeAssistantError(err),
		})
	}
}

func sanitizeAssistantError(err error) string {
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "bearer ") {
		return "Terjadi gangguan koneksi ke layanan AI."
	}
	if strings.Contains(msg, "upstream LLM status") {
		return "Layanan AI sedang sibuk atau mengalami gangguan sementara. Silakan coba lagi sebentar."
	}
	return msg
}
