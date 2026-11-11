package handler

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// telegramSender mendefinisikan kontrak minimal untuk mengirim pesan ke Telegram.
type telegramSender interface {
	Send(ctx context.Context, chatID, text string) error
}

// TelegramHandler melayani penghubungan chat dan webhook dari Telegram.
type TelegramHandler struct {
	telegram service.TelegramService
	botName  string
	secret   string
	sender   telegramSender
}

// NewTelegramHandler merangkai handler Telegram.
func NewTelegramHandler(telegram service.TelegramService, botUsername, webhookSecret string, sender telegramSender) *TelegramHandler {
	return &TelegramHandler{
		telegram: telegram,
		botName:  strings.TrimSpace(botUsername),
		secret:   webhookSecret,
		sender:   sender,
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
			_, err := h.telegram.HandleStart(c.Request.Context(), token, chatID)
			if err != nil {
				if h.sender != nil {
					_ = h.sender.Send(c.Request.Context(), chatID, "⚠️ Gagal menghubungkan akun: tautan sudah kedaluwarsa atau tidak valid.\n\nSilakan klik 'Hubungkan Telegram' kembali dari dashboard atau pengaturan ILA Platform Anda.")
				}
			} else {
				if h.sender != nil {
					_ = h.sender.Send(c.Request.Context(), chatID, "✅ Akun ILA Platform Anda berhasil terhubung!\n\nNotifikasi sinyal trading dan konfirmasi entry akan dikirimkan langsung ke chat ini.")
				}
			}
		} else {
			if h.sender != nil {
				_ = h.sender.Send(c.Request.Context(), chatID, "👋 Halo! Saya bot resmi ILA Platform.\n\nUntuk menghubungkan akun Anda, silakan buka Pengaturan di dashboard ILA Platform (rekavia.com) dan klik 'Hubungkan Telegram'.")
			}
		}
	} else if strings.HasPrefix(text, "/help") || strings.HasPrefix(text, "/") {
		if h.sender != nil {
			_ = h.sender.Send(c.Request.Context(), chatID, "👋 Bot ILA Platform siap mengirimkan notifikasi sinyal trading dan alert ke akun Anda.\n\nUntuk menghubungkan atau mengelola akun, kunjungi menu Pengaturan di web ILA Platform (rekavia.com).")
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
