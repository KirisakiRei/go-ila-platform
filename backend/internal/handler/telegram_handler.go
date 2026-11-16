package handler

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/internal/telegram"
	"ila/backend/pkg/httpx"
)

// telegramSender mendefinisikan kontrak minimal untuk mengirim pesan ke Telegram.
type telegramSender interface {
	Send(ctx context.Context, chatID, text string) error
}

// telegramEditor adalah kemampuan tambahan yang dipakai tombol inline dan
// menu. Kanal tanpa dukungan ini tetap bisa mengirim notifikasi biasa.
type telegramEditor interface {
	AnswerCallback(ctx context.Context, callbackID, text string) error
	EditMessageText(ctx context.Context, chatID string, messageID int64, text string, buttons [][]telegram.InlineButton) error
	SendMessageReturningID(ctx context.Context, chatID string, msg telegram.Message) (int64, error)
}

// TelegramHandler melayani penghubungan chat dan webhook dari Telegram.
type TelegramHandler struct {
	telegram service.TelegramService
	botName  string
	secret   string
	sender   telegramSender
	menu     service.TelegramMenuService
	approve  service.ApproveService
	baseURL  string
	logger   *slog.Logger
	edit     telegramEditor
}

// NewTelegramHandler merangkai handler Telegram.
//
// sender dipakai untuk notifikasi teks biasa. Bila klien yang diberikan juga
// mendukung tombol inline, menu dan konfirmasi entry ikut aktif.
func NewTelegramHandler(telegram service.TelegramService, botUsername, webhookSecret string, sender telegramSender) *TelegramHandler {
	h := &TelegramHandler{
		telegram: telegram,
		botName:  strings.TrimSpace(botUsername),
		secret:   webhookSecret,
		sender:   sender,
	}
	if editor, ok := sender.(telegramEditor); ok {
		h.edit = editor
	}
	return h
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
	CallbackQuery *callbackUpdate `json:"callback_query"`
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
	if err := c.ShouldBindJSON(&update); err != nil {
		c.Status(http.StatusOK)
		return
	}
	if update.CallbackQuery != nil {
		h.handleCallback(c.Request.Context(), update.CallbackQuery)
		c.Status(http.StatusOK)
		return
	}
	if update.Message == nil {
		c.Status(http.StatusOK)
		return
	}

	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
	text := strings.TrimSpace(update.Message.Text)
	if aksi, kenal := telegram.AksiDariTeks(text); kenal {
		h.handleMenuTeks(c.Request.Context(), chatID, aksi)
	} else if strings.HasPrefix(text, "/start") {
		token := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
		if token != "" {
			// Kegagalan token dibalas sebagai 200: pesan balasan ke user
			// disusun kanal Telegram, bukan dibocorkan lewat status HTTP.
			_, err := h.telegram.HandleStart(c.Request.Context(), token, chatID)
			if err != nil {
				h.pasangKeyboard(c.Request.Context(), chatID, "⚠️ Gagal menghubungkan akun: tautan sudah kedaluwarsa atau tidak valid.\n\nSilakan klik 'Hubungkan Telegram' kembali dari dashboard atau pengaturan ILA Platform Anda.")
			} else {
				h.pasangKeyboard(c.Request.Context(), chatID, "✅ Akun ILA Platform Anda berhasil terhubung!\n\nNotifikasi sinyal trading dan konfirmasi entry akan dikirimkan langsung ke chat ini. Menu ada di bawah.")
			}
		} else {
			h.pasangKeyboard(c.Request.Context(), chatID, "👋 Halo! Saya bot resmi ILA Platform.\n\nPilih menu di bawah untuk memantau posisi, membuka riwayat, dan melihat ringkasan hari.\n\nKalau akun Anda belum terhubung, buka Pengaturan di dashboard ILA Platform (rekavia.com) lalu klik 'Hubungkan Telegram'.")
		}
	} else if strings.HasPrefix(text, "/help") || strings.HasPrefix(text, "/") {
		h.pasangKeyboard(c.Request.Context(), chatID, "👋 Bot ILA Platform siap mengirimkan notifikasi sinyal trading dan alert ke akun Anda.\n\nGunakan menu di bawah untuk memantau posisi, membuka riwayat, dan melihat ringkasan hari. Pengaturan akun tetap di website ILA Platform (rekavia.com).")
	} else {
		// Pesan bebas selain menu: balas dengan menu supaya keyboardnya tetap ada.
		h.pasangKeyboard(c.Request.Context(), chatID, "Pilih menu di bawah untuk melanjutkan.")
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
