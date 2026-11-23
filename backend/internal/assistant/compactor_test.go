package assistant

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
)

type fakeCompactLLM struct {
	reply string
	err   error
	calls int
}

func (f *fakeCompactLLM) Complete(_ context.Context, _ ChatRequest) (*ChatResult, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return &ChatResult{Content: f.reply}, nil
}

func (f *fakeCompactLLM) Stream(_ context.Context, _ ChatRequest, _ func(string) error) (*ChatResult, error) {
	return nil, nil
}

func TestPlanCompaction_IdleTrigger(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	last := now.Add(-2*time.Hour - time.Minute)
	sess := &model.AISession{ID: "s1", LastMessageAt: &last}
	msgs := []model.AIMessage{
		{Seq: 1, Role: "user", Content: "Halo"},
		{Seq: 2, Role: "assistant", Content: "Hai"},
	}

	plan := PlanCompaction(sess, msgs, now, CompactionOptions{
		IdleAfter:   2 * time.Hour,
		MaxMessages: 20,
		MaxChars:    24000,
		KeepRecent:  8,
	})
	if plan.Reason != CompactIdle || plan.UpToSeq != 2 || len(plan.ToFold) != 2 {
		t.Fatalf("idle compaction tidak sesuai: %+v", plan)
	}
}

func TestPlanCompaction_LengthPreservesToolPair(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	last := now.Add(-5 * time.Minute)
	sess := &model.AISession{ID: "s1", LastMessageAt: &last}

	// Buat 22 pesan dengan pasangan tool call tepat di batas potong 8 pesan terakhir.
	var msgs []model.AIMessage
	for i := 1; i <= 12; i++ {
		role := "user"
		if i%2 == 0 {
			role = "assistant"
		}
		msgs = append(msgs, model.AIMessage{Seq: int64(i), Role: role, Content: "pesan"})
	}
	// Seq 13-16 adalah 1 giliran user -> assistant(tool_calls) -> tool -> assistant
	msgs = append(msgs,
		model.AIMessage{Seq: 13, Role: "user", Content: "cek"},
		model.AIMessage{Seq: 14, Role: "assistant", ToolCallsJSON: `[{"id":"c1"}]`},
		model.AIMessage{Seq: 15, Role: "tool", ToolCallID: "c1", Content: "{}"},
		model.AIMessage{Seq: 16, Role: "assistant", Content: "hasil"},
	)
	for i := 17; i <= 22; i++ {
		role := "user"
		if i%2 == 0 {
			role = "assistant"
		}
		msgs = append(msgs, model.AIMessage{Seq: int64(i), Role: role, Content: "lanjutan"})
	}

	plan := PlanCompaction(sess, msgs, now, CompactionOptions{
		IdleAfter:   2 * time.Hour,
		MaxMessages: 20,
		MaxChars:    24000,
		KeepRecent:  8, // target awal jatuh di index 14 (assistant tool_calls) / 15 (tool)
	})
	if plan.Reason != CompactLength {
		t.Fatalf("harus CompactLength, dapat %q", plan.Reason)
	}
	if plan.UpToSeq != 12 {
		t.Fatalf("cut harus mundur ke sebelum seq 13 agar pasangan tool tidak terbelah, dapat UpToSeq=%d", plan.UpToSeq)
	}
}

func TestSummarizeSession_ParsesJSONWithFence(t *testing.T) {
	llm := &fakeCompactLLM{
		reply: "```json\n{\"rolling_summary\":\"Bahas XAUUSD H4\",\"user_memory\":\"- Fokus XAUUSD\"}\n```",
	}
	out, err := SummarizeSession(context.Background(), llm, "m1", "", "", []model.AIMessage{
		{Role: "user", Content: "analisa xau", CreatedAt: time.Now()},
	})
	if err != nil {
		t.Fatalf("SummarizeSession error: %v", err)
	}
	if out.RollingSummary != "Bahas XAUUSD H4" || out.UserMemory != "- Fokus XAUUSD" {
		t.Fatalf("hasil compaction salah: %+v", out)
	}
}
