package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/model"
	"ila/backend/internal/service"
)

type fakeTelegramSender struct {
	mu       sync.Mutex
	messages []struct {
		chatID string
		text   string
	}
}

func (s *fakeTelegramSender) Send(_ context.Context, chatID, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, struct {
		chatID string
		text   string
	}{chatID: chatID, text: text})
	return nil
}

type fakeTelegramService struct {
	createLinkFn  func(ctx context.Context, userID, botUsername string) (*service.TelegramLink, error)
	handleStartFn func(ctx context.Context, rawToken, chatID string) (*service.TelegramStartResult, error)
	disconnectFn  func(ctx context.Context, userID string) error
	resolveChatFn func(ctx context.Context, chatID string) (*model.User, error)
}

func (f *fakeTelegramService) CreateLink(ctx context.Context, userID, botUsername string) (*service.TelegramLink, error) {
	if f.createLinkFn != nil {
		return f.createLinkFn(ctx, userID, botUsername)
	}
	return &service.TelegramLink{URL: "https://t.me/testbot?start=xyz", ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func (f *fakeTelegramService) HandleStart(ctx context.Context, rawToken, chatID string) (*service.TelegramStartResult, error) {
	if f.handleStartFn != nil {
		return f.handleStartFn(ctx, rawToken, chatID)
	}
	return &service.TelegramStartResult{UserID: "u1", ChatID: chatID, Message: "ok"}, nil
}

func (f *fakeTelegramService) Disconnect(ctx context.Context, userID string) error {
	if f.disconnectFn != nil {
		return f.disconnectFn(ctx, userID)
	}
	return nil
}

func (f *fakeTelegramService) ResolveChat(ctx context.Context, chatID string) (*model.User, error) {
	if f.resolveChatFn != nil {
		return f.resolveChatFn(ctx, chatID)
	}
	return nil, nil
}

func TestTelegramWebhookSecretAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeTelegramService{}
	sender := &fakeTelegramSender{}
	h := NewTelegramHandler(svc, "testbot", "rahasia123", sender)

	r := gin.New()
	r.POST("/webhook", h.Webhook)

	// Tanpa header rahasia: 401
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// Dengan header cocok: 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{}`))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "rahasia123")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestTelegramWebhookStartReplies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("start dengan token sah mengirim konfirmasi sukses", func(t *testing.T) {
		sender := &fakeTelegramSender{}
		svc := &fakeTelegramService{
			handleStartFn: func(_ context.Context, rawToken, chatID string) (*service.TelegramStartResult, error) {
				if rawToken != "token-sah" {
					return nil, errors.New("token tidak dikenal")
				}
				return &service.TelegramStartResult{UserID: "u-1", ChatID: chatID}, nil
			},
		}
		h := NewTelegramHandler(svc, "testbot", "", sender)

		r := gin.New()
		r.POST("/webhook", h.Webhook)

		payload, _ := json.Marshal(map[string]any{
			"message": map[string]any{
				"text": "/start token-sah",
				"chat": map[string]any{"id": 12345678},
			},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if len(sender.messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(sender.messages))
		}
		if sender.messages[0].chatID != "12345678" {
			t.Fatalf("expected chatID 12345678, got %s", sender.messages[0].chatID)
		}
		if !strings.Contains(sender.messages[0].text, "berhasil terhubung") {
			t.Fatalf("pesan tidak memuat konfirmasi: %s", sender.messages[0].text)
		}
	})

	t.Run("start dengan token kedaluwarsa mengirim pesan gagal", func(t *testing.T) {
		sender := &fakeTelegramSender{}
		svc := &fakeTelegramService{
			handleStartFn: func(_ context.Context, _, _ string) (*service.TelegramStartResult, error) {
				return nil, errors.New("tautan kedaluwarsa")
			},
		}
		h := NewTelegramHandler(svc, "testbot", "", sender)

		r := gin.New()
		r.POST("/webhook", h.Webhook)

		payload, _ := json.Marshal(map[string]any{
			"message": map[string]any{
				"text": "/start token-basi",
				"chat": map[string]any{"id": 8888},
			},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if len(sender.messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(sender.messages))
		}
		if !strings.Contains(sender.messages[0].text, "Gagal menghubungkan") {
			t.Fatalf("pesan tidak memuat info gagal: %s", sender.messages[0].text)
		}
	})

	t.Run("start tanpa token mengirim panduan bot", func(t *testing.T) {
		sender := &fakeTelegramSender{}
		svc := &fakeTelegramService{}
		h := NewTelegramHandler(svc, "testbot", "", sender)

		r := gin.New()
		r.POST("/webhook", h.Webhook)

		payload, _ := json.Marshal(map[string]any{
			"message": map[string]any{
				"text": "/start",
				"chat": map[string]any{"id": 9999},
			},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if len(sender.messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(sender.messages))
		}
		if !strings.Contains(sender.messages[0].text, "Halo! Saya bot resmi ILA Platform") {
			t.Fatalf("pesan tidak memuat sapaan: %s", sender.messages[0].text)
		}
	})

	t.Run("help mengirim panduan", func(t *testing.T) {
		sender := &fakeTelegramSender{}
		svc := &fakeTelegramService{}
		h := NewTelegramHandler(svc, "testbot", "", sender)

		r := gin.New()
		r.POST("/webhook", h.Webhook)

		payload, _ := json.Marshal(map[string]any{
			"message": map[string]any{
				"text": "/help",
				"chat": map[string]any{"id": 9999},
			},
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if len(sender.messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(sender.messages))
		}
		if !strings.Contains(sender.messages[0].text, "Bot ILA Platform siap mengirimkan") {
			t.Fatalf("pesan tidak memuat panduan: %s", sender.messages[0].text)
		}
	})
}
