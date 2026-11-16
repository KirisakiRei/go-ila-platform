// Package telegram mengirim notifikasi ke user lewat bot.
//
// Telegram hanya kanal pemberitahuan. Tidak ada tombol yang mengeksekusi order:
// deep-link membawa user ke browser yang terautentikasi (plans/13 §4.1).
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config mengatur klien Telegram.
type Config struct {
	BotToken string
	Timeout  time.Duration
	// MinInterval adalah jeda minimum antar pesan ke chat yang sama,
	// agar tidak melanggar batas laju Telegram.
	MinInterval time.Duration
}

// Enabled menandai token tersedia.
func (c Config) Enabled() bool {
	return c.BotToken != ""
}

// Client adalah pengirim pesan yang dibatasi laju per chat.
type Client struct {
	cfg    Config
	http   *http.Client
	logger *slog.Logger
	mu     sync.Mutex
	lastAt map[string]time.Time
}

// NewClient membuat klien pengirim.
func NewClient(cfg Config, logger *slog.Logger) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MinInterval <= 0 {
		cfg.MinInterval = time.Second
	}
	return &Client{
		cfg:    cfg,
		http:   &http.Client{Timeout: cfg.Timeout},
		logger: logger,
		lastAt: make(map[string]time.Time),
	}
}

// InlineButton adalah tombol pada inline keyboard Telegram.
//
// URL membuka tautan; CallbackData mengirim tekanan kembali ke webhook.
type InlineButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

// KeyboardButton adalah satu tombol pada keyboard tetap di bawah layar.
type KeyboardButton struct {
	Text string `json:"text"`
}

// ReplyKeyboardMarkup menyusun keyboard tetap yang selalu terlihat.
type ReplyKeyboardMarkup struct {
	Keyboard       [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard bool               `json:"resize_keyboard"`
	IsPersistent   bool               `json:"is_persistent"`
	// InputFieldPlaceholder adalah teks petunjuk pada kolom ketik.
	InputFieldPlaceholder string `json:"input_field_placeholder,omitempty"`
}

// InlineKeyboardMarkup menyusun tombol-tombol pada inline keyboard.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineButton `json:"inline_keyboard"`
}

type sendMessageRequest struct {
	ChatID         string `json:"chat_id"`
	Text           string `json:"text"`
	DisablePreview bool   `json:"disable_web_page_preview"`
	ParseMode      string `json:"parse_mode,omitempty"`
	ReplyMarkup    any    `json:"reply_markup,omitempty"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
	Result      struct {
		MessageID int64 `json:"message_id"`
	} `json:"result"`
}

// Send mengirim satu pesan teks sederhana ke chat (kompatibilitas mundur).
func (c *Client) Send(ctx context.Context, chatID, text string) error {
	return c.SendMessage(ctx, chatID, Message{Text: text})
}

// SendMessage mengirim pesan Telegram lengkap dengan teks dan tombol inline keyboard.
func (c *Client) SendMessage(ctx context.Context, chatID string, msg Message) error {
	_, err := c.SendMessageReturningID(ctx, chatID, msg)
	return err
}

// SendMessageReturningID mengirim pesan dan mengembalikan message_id.
//
// Dipakai Runner untuk menyimpan jejak pesan bertombol, agar tombolnya bisa
// dibersihkan sendiri saat setup tidak berlaku lagi.
func (c *Client) SendMessageReturningID(ctx context.Context, chatID string, msg Message) (int64, error) {
	if !c.cfg.Enabled() || chatID == "" {
		return 0, nil
	}
	c.throttle(ctx, chatID)
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return c.post(ctx, "sendMessage", c.pesanPayload(chatID, msg))
}

// ChatActionTyping menandai bot sedang menyiapkan balasan, sehingga klien
// menampilkan indikator "sedang menulis" selama proses berjalan.
const ChatActionTyping = "typing"

// BotCommand adalah satu entri pada daftar perintah di menu "/".
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// SetMyCommands mendaftarkan daftar perintah ke Telegram.
//
// Tanpa ini, tombol "/" hanya menampilkan perintah bawaan kosong walau bot
// sudah bisa menanganinya.
func (c *Client) SetMyCommands(ctx context.Context, commands []BotCommand) error {
	if !c.cfg.Enabled() {
		return nil
	}
	_, err := c.post(ctx, "setMyCommands", map[string]any{"commands": commands})
	return err
}

// SendChatAction menampilkan indikator aktivitas pada chat.
func (c *Client) SendChatAction(ctx context.Context, chatID, action string) error {
	if !c.cfg.Enabled() || chatID == "" {
		return nil
	}
	_, err := c.post(ctx, "sendChatAction", map[string]any{
		"chat_id": chatID,
		"action":  action,
	})
	return err
}

// EditMessageText mengganti isi pesan yang sudah terkirim.
//
// Tanpa tombol, Telegram mempertahankan tombol lama. Karena itu keyboard kosong
// selalu dikirim eksplisit saat pesan harus kehilangan tombolnya.
func (c *Client) EditMessageText(ctx context.Context, chatID string, messageID int64, text string, buttons [][]InlineButton) error {
	if !c.cfg.Enabled() || chatID == "" {
		return nil
	}
	if buttons == nil {
		buttons = [][]InlineButton{}
	}
	_, err := c.post(ctx, "editMessageText", map[string]any{
		"chat_id":      chatID,
		"message_id":   messageID,
		"text":         text,
		"parse_mode":   "HTML",
		"reply_markup": InlineKeyboardMarkup{InlineKeyboard: buttons},
	})
	return err
}

// AnswerCallback menutup indikator memuat pada tombol yang ditekan.
//
// Telegram menampilkan spinner sampai ini dipanggil, jadi jawaban harus tetap
// dikirim walau pemrosesan gagal.
func (c *Client) AnswerCallback(ctx context.Context, callbackID, text string) error {
	if !c.cfg.Enabled() || callbackID == "" {
		return nil
	}
	_, err := c.post(ctx, "answerCallbackQuery", map[string]any{
		"callback_query_id": callbackID,
		"text":              text,
	})
	return err
}

// pesanPayload menyusun isi permintaan kirim pesan.
func (c *Client) pesanPayload(chatID string, msg Message) any {
	payload := sendMessageRequest{
		ChatID:         chatID,
		Text:           msg.Text,
		DisablePreview: true,
		// Pesan memakai <pre> untuk kotak tabel. Tanpa parse_mode, Telegram
		// menampilkan tag sebagai teks mentah.
		ParseMode: "HTML",
	}
	// Keyboard tetap didahulukan: pesan interaktif butuh menu, sedangkan
	// pesan bertombol inline sudah membawa aksinya sendiri.
	if msg.Keyboard != nil {
		payload.ReplyMarkup = *msg.Keyboard
	} else if len(msg.Buttons) > 0 {
		payload.ReplyMarkup = InlineKeyboardMarkup{InlineKeyboard: msg.Buttons}
	}
	return payload
}

// post mengirim satu permintaan ke Telegram dan membaca message_id bila ada.
func (c *Client) post(ctx context.Context, method string, payload any) (int64, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("menyusun pesan: %w", err)
	}
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/%s", c.cfg.BotToken, method)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("membuat permintaan telegram: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("mengirim pesan telegram: %w", sanitizeTransportError(err, c.cfg.BotToken))
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	var hasil apiResponse
	_ = json.NewDecoder(resp.Body).Decode(&hasil)
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("telegram menolak permintaan: %s", hasil.Description)
	}
	return hasil.Result.MessageID, nil
}

// throttle menjaga jarak antar pesan ke chat yang sama.
func (c *Client) throttle(ctx context.Context, chatID string) {
	c.mu.Lock()
	last, seen := c.lastAt[chatID]
	c.mu.Unlock()

	if seen {
		if wait := c.cfg.MinInterval - time.Since(last); wait > 0 {
			timer := time.NewTimer(wait)
			defer timer.Stop()

			// ctx boleh nil saat dipakai dari jalur tanpa request (mis. antrean).
			var done <-chan struct{}
			if ctx != nil {
				done = ctx.Done()
			}

			select {
			case <-done:
				return
			case <-timer.C:
			}
		}
	}

	c.mu.Lock()
	c.lastAt[chatID] = time.Now()
	c.mu.Unlock()
}

// DeepLink menyusun tautan konfirmasi entry dengan awalan /app bila belum ada.
// Token tidak memuat rahasia; halaman tujuan tetap mewajibkan sesi login.
func DeepLink(baseURL, token string) string {
	if baseURL == "" || token == "" {
		return ""
	}
	base := trimSlash(baseURL)
	if !strings.HasSuffix(base, "/app") {
		base += "/app"
	}
	return fmt.Sprintf("%s/approve/%s", base, url.PathEscape(token))
}

// AppURL menyusun tautan ke halaman panel aplikasi (dengan awalan /app bila belum ada).
func AppURL(baseURL, path string) string {
	if baseURL == "" {
		return ""
	}
	base := trimSlash(baseURL)
	if !strings.HasSuffix(base, "/app") {
		base += "/app"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func trimSlash(raw string) string {
	for len(raw) > 0 && raw[len(raw)-1] == '/' {
		raw = raw[:len(raw)-1]
	}
	return raw
}

// sanitizeTransportError membuang token bot dari pesan kegagalan jaringan.
//
// Token Telegram berada di path URL, dan error dari http.Do memuat URL
// lengkap. Tanpa penyaringan ini token ikut tercatat di log maupun di kolom
// last_error yang ditampilkan ke dasbor user.
func sanitizeTransportError(err error, token string) error {
	if err == nil {
		return nil
	}

	message := err.Error()
	if token != "" {
		message = strings.ReplaceAll(message, token, "<token-disembunyikan>")
	}

	// URLRequest dipertahankan bila ada, tetapi isinya sudah dibersihkan.
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		cleaned := *urlErr
		cleaned.URL = strings.ReplaceAll(urlErr.URL, token, "<token-disembunyikan>")
		cleaned.Err = errors.New(strings.ReplaceAll(urlErr.Err.Error(), token, "<token-disembunyikan>"))
		return &cleaned
	}

	return errors.New(message)
}
