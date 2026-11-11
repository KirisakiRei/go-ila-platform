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

type sendMessageRequest struct {
	ChatID         string `json:"chat_id"`
	Text           string `json:"text"`
	DisablePreview bool   `json:"disable_web_page_preview"`
	ParseMode      string `json:"parse_mode,omitempty"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// Send mengirim satu pesan teks ke chat. Panggilan boleh gagal: kegagalan
// Telegram tidak boleh dianggap persetujuan atau menghalangi trading.
func (c *Client) Send(ctx context.Context, chatID, text string) error {
	if !c.cfg.Enabled() {
		return nil
	}
	if chatID == "" {
		return nil
	}

	c.throttle(ctx, chatID)
	if err := ctx.Err(); err != nil {
		return err
	}

	body, err := json.Marshal(sendMessageRequest{
		ChatID:         chatID,
		Text:           text,
		DisablePreview: true,
	})
	if err != nil {
		return fmt.Errorf("menyusun pesan: %w", err)
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.cfg.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("membuat permintaan telegram: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("mengirim pesan telegram: %w", sanitizeTransportError(err, c.cfg.BotToken))
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		var apiErr apiResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("telegram menolak pesan: %s", apiErr.Description)
	}

	return nil
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
