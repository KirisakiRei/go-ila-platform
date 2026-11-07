package handler

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/middleware"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/httpx"
)

// TelegramHandler melayani penghubungan chat dan webhook dari Telegram.
type TelegramHandler struct {
	telegram service.TelegramService
	botName  string
	secret   string
}

// NewTelegramHandler merangkai handler Telegram.
func NewTelegramHandler(telegram service.TelegramService, botUsername, webhookSecret string) *TelegramHandler {
	return &TelegramHandler{
		telegram: telegram,
		botName:  strings.TrimSpace(botUsername),
		secret:   webhookSecret,
	}
}

// CreateLink menangani POST /api/v1/telegram/link.
func (h *TelegramHandler) CreateLink(c *gin.Context) {
	result, err := h.telegram.CreateLink(c.Request.Context(), middleware.CurrentUserID(c), h.botName)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Tautan hubungkan Telegram", result)
}

// Disconnect menangani DELETE /api/v1/telegram/link.
func (h *TelegramHandler) Disconnect(c *gin.Context) {
	if err := h.telegram.Disconnect(c.Request.Context(), middleware.CurrentUserID(c)); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Telegram diputus", nil)
}

// webhookUpdate adalah bagian pesan yang dipakai dari payload Telegram.
type webhookUpdate struct {
	Message *struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

// Webhook menangani POST /api/v1/telegram/webhook.
//
// Selalu membalas 200 agar Telegram tidak mengulang kiriman. Chat yang belum
// terhubung tidak menerima data apa pun (plans/13 §3.4).
func (h *TelegramHandler) Webhook(c *gin.Context) {
	if !h.authorized(c) {
		c.Status(http.StatusUnauthorized)
		return
	}

	var update webhookUpdate
	if err := c.ShouldBindJSON(&update); err != nil || update.Message == nil {
		c.Status(http.StatusOK)
		return
	}

	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
	text := strings.TrimSpace(update.Message.Text)

	if strings.HasPrefix(text, "/start") {
		token := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
		if token != "" {
			// Kegagalan token dibalas sebagai 200: pesan balasan ke user
			// disusun kanal Telegram, bukan dibocorkan lewat status HTTP.
			_, _ = h.telegram.HandleStart(c.Request.Context(), token, chatID)
		}
	}

	c.Status(http.StatusOK)
}

// authorized memeriksa header rahasia webhook dari Telegram.
// Tanpa rahasia yang cocok, permintaan dianggap bukan dari Telegram.
func (h *TelegramHandler) authorized(c *gin.Context) bool {
	if h.secret == "" {
		return true
	}
	got := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.secret)) == 1
}
