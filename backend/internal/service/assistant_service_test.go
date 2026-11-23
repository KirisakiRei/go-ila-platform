package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/assistant"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

type stubAssistantLLM struct {
	streamFn   func(round int, req assistant.ChatRequest, onDelta func(string) error) (*assistant.ChatResult, error)
	completeFn func(req assistant.ChatRequest) (*assistant.ChatResult, error)
	rounds     int
}

func (s *stubAssistantLLM) Complete(_ context.Context, req assistant.ChatRequest) (*assistant.ChatResult, error) {
	if s.completeFn != nil {
		return s.completeFn(req)
	}
	return &assistant.ChatResult{Content: `{"rolling_summary":"Ringkas","user_memory":"- Memori"}`}, nil
}

func (s *stubAssistantLLM) Stream(_ context.Context, req assistant.ChatRequest, onDelta func(string) error) (*assistant.ChatResult, error) {
	s.rounds++
	return s.streamFn(s.rounds, req, onDelta)
}

type stubAssistantAccounts struct {
	added string
}

func (s *stubAssistantAccounts) List(_ context.Context, _ string) ([]dto.AccountResponse, error) {
	return nil, nil
}
func (s *stubAssistantAccounts) ListSubscriptions(_ context.Context, _ string) ([]dto.SubscriptionResponse, error) {
	return nil, nil
}
func (s *stubAssistantAccounts) Subscribe(_ context.Context, _ string, req dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error) {
	s.added = req.Symbol
	return &dto.SubscriptionResponse{ID: "sub-1", Symbol: req.Symbol, IsActive: true}, nil
}
func (s *stubAssistantAccounts) Unsubscribe(_ context.Context, _, _ string) error { return nil }

func openAssistantSvcTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("sqlite error: %v", err)
	}
	if err := db.AutoMigrate(&model.AISession{}, &model.AIMessage{}, &model.AIUserMemory{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}
	return db
}

func TestAssistantService_ToolCallStreamAndIdleCompact(t *testing.T) {
	ctx := context.Background()
	db := openAssistantSvcTestDB(t)
	repo := repository.NewAssistantRepository(db)

	now := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	acc := &stubAssistantAccounts{}
	llm := &stubAssistantLLM{
		streamFn: func(round int, req assistant.ChatRequest, onDelta func(string) error) (*assistant.ChatResult, error) {
			if round == 1 {
				return &assistant.ChatResult{
					ToolCalls: []assistant.ToolCall{{
						ID:       "tc_1",
						Type:     "function",
						Function: assistant.ToolCallFunction{Name: "subscribe_pair", Arguments: `{"symbol":"USDJPY"}`},
					}},
				}, nil
			}
			_ = onDelta("USDJPY ")
			_ = onDelta("sudah dipantau.")
			return &assistant.ChatResult{Content: "USDJPY sudah dipantau."}, nil
		},
	}

	svc := NewAssistantService(AssistantServiceDeps{
		Enabled: true,
		Model:   "test-model",
		Repo:    repo,
		LLM:     llm,
		Tools:   assistant.ToolDeps{Accounts: acc},
		CompactOpts: assistant.CompactionOptions{
			IdleAfter:   2 * time.Hour,
			MaxMessages: 20,
			MaxChars:    24000,
			KeepRecent:  8,
		},
		Clock: func() time.Time { return now },
	})

	var events []assistant.StreamEvent
	err := svc.ChatStream(ctx, "user-1", "Pantau USDJPY", func(ev assistant.StreamEvent) error {
		events = append(events, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream error: %v", err)
	}
	if acc.added != "USDJPY" {
		t.Fatalf("subscribe_pair tidak tereksekusi: %q", acc.added)
	}

	var types []string
	for _, ev := range events {
		types = append(types, ev.Type)
	}
	joined := strings.Join(types, ",")
	for _, expected := range []string{"status", "tool_start", "tool_done", "action_applied", "delta", "done"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("event %s tidak muncul pada stream: %s", expected, joined)
		}
	}

	// Majukan waktu > 2 jam untuk memicu idle compaction otomatis di chat berikutnya.
	now = now.Add(3 * time.Hour)
	llm.rounds = 0
	var sawSystemSummary bool
	llm.streamFn = func(_ int, req assistant.ChatRequest, onDelta func(string) error) (*assistant.ChatResult, error) {
		sawSystemSummary = strings.Contains(req.Messages[0].Content, "[RINGKASAN SESI SEBELUMNYA]") &&
			strings.Contains(req.Messages[0].Content, "Ringkas")
		_ = onDelta("Siap!")
		return &assistant.ChatResult{Content: "Siap!"}, nil
	}

	var events2 []assistant.StreamEvent
	err = svc.ChatStream(ctx, "user-1", "Lanjut bahas", func(ev assistant.StreamEvent) error {
		events2 = append(events2, ev)
		return nil
	})
	if err != nil {
		t.Fatalf("ChatStream kedua error: %v", err)
	}
	if !sawSystemSummary {
		t.Fatalf("rolling_summary tidak disuntikkan ke system prompt setelah idle compaction")
	}
	var doneEv assistant.StreamEvent
	for _, ev := range events2 {
		if ev.Type == "done" {
			doneEv = ev
		}
	}
	if !doneEv.Compact {
		t.Fatalf("expected done.Compact=true after idle compaction")
	}

	view, err := svc.GetSession(ctx, "user-1", 0, 20)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if len(view.Messages) != 4 {
		t.Fatalf("riwayat tampilan diharapkan tetap menyimpan 4 pesan user+assistant, dapat %d", len(view.Messages))
	}
	if view.CompactedBeforeSeq == 0 {
		t.Fatalf("CompactedBeforeSeq harus > 0")
	}
}
