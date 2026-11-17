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
	SendChatAction(ctx context.Context, chatID, action string) error
}

// TelegramHandler melayani penghubungan chat dan webhook dari Telegram.
type TelegramHandler struct {
	telegram      service.TelegramService
	botName       string
	secret        string
	requireSecret bool
	sender        telegramSender
	menu          service.TelegramMenuService
	approve       service.ApproveService
	baseURL       string
	logger        *slog.Logger
	edit          telegramEditor
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

// RequireSecret menolak webhook di produksi bila TELEGRAM_WEBHOOK_SECRET belum
// diisi, sehingga pintu bot tidak pernah terbuka tanpa autentikasi.
func (h *TelegramHandler) RequireSecret(required bool) *TelegramHandler {
	h.requireSecret = required
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

	// Indikator "sedang menulis" dipasang sebelum pekerjaan dimulai, karena
	// menyusun halaman butuh beberapa query database.
	if h.edit != nil {
		_ = h.edit.SendChatAction(c.Request.Context(), chatID, telegram.ChatActionTyping)
	}

	if strings.HasPrefix(text, "/") {
		h.handlePerintah(c.Request.Context(), chatID, text)
	} else {
		// Pesan bebas: tampilkan menu alih-alih balasan kosong.
		h.handleMenuTeks(c.Request.Context(), chatID, telegram.CallbackMenu)
	}

	c.Status(http.StatusOK)
}

// handlePerintah melayani perintah chat seperti /pantau dan /riwayat.
func (h *TelegramHandler) handlePerintah(ctx context.Context, chatID, text string) {
	perintah, _, _ := strings.Cut(text, " ")
	if strings.EqualFold(strings.TrimPrefix(perintah, "/"), telegram.CommandMulai) {
		token := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
		if token != "" {
			// Kegagalan token dibalas sebagai 200: balasan ke user disusun kanal
			// Telegram, bukan dibocorkan lewat status HTTP.
			if _, err := h.telegram.HandleStart(ctx, token, chatID); err != nil {
				h.kirimTeks(ctx, chatID, "⚠️ Gagal menghubungkan akun: tautan sudah kedaluwarsa atau tidak valid.\n\nSilakan klik 'Hubungkan Telegram' kembali dari dashboard atau pengaturan BOT Trading ILA Anda.")
			} else {
				h.kirimTeks(ctx, chatID, "✅ Akun Anda berhasil terhubung ke BOT Trading ILA!\n\nNotifikasi sinyal trading dan konfirmasi entry akan dikirimkan langsung ke chat ini. Gunakan tombol di bawah pesan ini.")
			}
			return
		}
		h.handleMenuTeks(ctx, chatID, telegram.CallbackMenu)
		return
	}

	aksi, kenal := telegram.AksiDariPerintah(text)
	if !kenal {
		h.handleMenuTeks(ctx, chatID, telegram.CallbackMenu)
		return
	}
	h.handleMenuTeks(ctx, chatID, aksi)
}

// authorized memeriksa header rahasia webhook dari Telegram.
// Tanpa rahasia yang cocok, permintaan dianggap bukan dari Telegram.
func (h *TelegramHandler) authorized(c *gin.Context) bool {
	if h.secret == "" {
		return !h.requireSecret
	}
	got := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.secret)) == 1
}
