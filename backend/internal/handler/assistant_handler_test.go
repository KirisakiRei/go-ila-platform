package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/assistant"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
)

type stubAssistantService struct {
	enabled bool
}

func (s *stubAssistantService) Enabled() bool { return s.enabled }

func (s *stubAssistantService) GetSession(_ context.Context, _ string, _ int64, _ int) (*service.AssistantSessionView, error) {
	return &service.AssistantSessionView{
		Enabled:  s.enabled,
		Messages: []service.AssistantMessageView{},
	}, nil
}

func (s *stubAssistantService) CompactSession(_ context.Context, _ string) (*service.AssistantSessionView, error) {
	return &service.AssistantSessionView{
		Enabled:    s.enabled,
		HasSummary: true,
		Messages:   []service.AssistantMessageView{},
	}, nil
}

func (s *stubAssistantService) ChatStream(_ context.Context, _, text string, emit func(assistant.StreamEvent) error) error {
	_ = emit(assistant.StreamEvent{Type: "status", Stage: "thinking", Label: "Sedang berpikir..."})
	_ = emit(assistant.StreamEvent{Type: "delta", Text: "Balasan: " + text})
	return emit(assistant.StreamEvent{Type: "done", MsgID: "m1", Seq: 2})
}

func TestAssistantHandler_SessionAndSSEChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, "user-1")
		c.Next()
	})

	h := NewAssistantHandler(&stubAssistantService{enabled: true})
	h.Register(r.Group("/api/v1"), nil)

	// GET /session
	req := httptest.NewRequest(http.MethodGet, "/api/v1/assistant/session", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET session status=%d body=%s", rec.Code, rec.Body.String())
	}
	var sessResp struct {
		Success bool                         `json:"success"`
		Data    service.AssistantSessionView `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sessResp); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}
	if !sessResp.Success || !sessResp.Data.Enabled {
		t.Fatalf("session response tak sesuai: %+v", sessResp)
	}

	// POST /chat SSE
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/assistant/chat", strings.NewReader(`{"message":"halo"}`))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("POST chat status=%d body=%s", rec2.Code, rec2.Body.String())
	}
	if ct := rec2.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type bukan SSE: %q", ct)
	}
	body := rec2.Body.String()
	for _, want := range []string{"event: status", "event: delta", "Balasan: halo", "event: done"} {
		if !strings.Contains(body, want) {
			t.Fatalf("SSE tidak memuat %q: %s", want, body)
		}
	}
}

func TestAssistantHandler_DisabledReturns503OnChat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, "user-1")
		c.Next()
	})
	h := NewAssistantHandler(&stubAssistantService{enabled: false})
	h.Register(r.Group("/api/v1"), nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assistant/chat", strings.NewReader(`{"message":"halo"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when disabled, got %d", rec.Code)
	}
}
