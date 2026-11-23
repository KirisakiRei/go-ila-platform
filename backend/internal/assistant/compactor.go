package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/timezone"
)

// CompactionOptions mengatur ambang ringkas otomatis sesi ILA AI.
type CompactionOptions struct {
	IdleAfter   time.Duration
	MaxMessages int
	MaxChars    int
	KeepRecent  int
	Model       string
}

// CompactionReason menandai mengapa compaction dijalankan.
type CompactionReason string

const (
	CompactNone   CompactionReason = ""
	CompactIdle   CompactionReason = "idle"
	CompactLength CompactionReason = "length"
	CompactManual CompactionReason = "manual"
)

// CompactionPlan menentukan pesan mana yang akan diringkas.
type CompactionPlan struct {
	Reason  CompactionReason
	UpToSeq int64
	ToFold  []model.AIMessage
}

// PlanCompaction mengevaluasi apakah sesi perlu diringkas (idle >= 2 jam atau
// jumlah/karakter pesan aktif melewati batas), lalu menentukan titik potong
// aman tanpa membelah pasangan assistant(tool_calls) <-> tool.
func PlanCompaction(session *model.AISession, active []model.AIMessage, now time.Time, opts CompactionOptions) CompactionPlan {
	if len(active) == 0 {
		return CompactionPlan{}
	}

	if opts.IdleAfter > 0 && session != nil && session.LastMessageAt != nil {
		if now.Sub(*session.LastMessageAt) >= opts.IdleAfter && len(active) >= 2 {
			return CompactionPlan{
				Reason:  CompactIdle,
				UpToSeq: active[len(active)-1].Seq,
				ToFold:  active,
			}
		}
	}

	overMessages := opts.MaxMessages > 0 && len(active) > opts.MaxMessages
	overChars := opts.MaxChars > 0 && totalActiveChars(active) > opts.MaxChars
	if !overMessages && !overChars {
		return CompactionPlan{}
	}

	keep := opts.KeepRecent
	if keep <= 0 {
		keep = 8
	}
	cut := len(active) - keep
	if cut <= 0 {
		return CompactionPlan{}
	}

	// Mundur ke awal giliran user agar pasangan assistant(tool_calls) dan tool
	// tidak pernah terpotong di tengah (OpenAI API menolak pesan tool yatim).
	cut = safeCutIndex(active, cut)
	if cut <= 0 {
		return CompactionPlan{}
	}

	return CompactionPlan{
		Reason:  CompactLength,
		UpToSeq: active[cut-1].Seq,
		ToFold:  active[:cut],
	}
}

// PlanManualCompaction meringkas seluruh pesan aktif saat pengguna memilih
// "Mulai topik baru" atau "Ringkas sesi".
func PlanManualCompaction(active []model.AIMessage) CompactionPlan {
	if len(active) == 0 {
		return CompactionPlan{}
	}
	return CompactionPlan{
		Reason:  CompactManual,
		UpToSeq: active[len(active)-1].Seq,
		ToFold:  active,
	}
}

func safeCutIndex(msgs []model.AIMessage, target int) int {
	if target <= 0 || target >= len(msgs) {
		return 0
	}
	// Cari awal turn `user` terdekat di <= target.
	for i := target; i > 0; i-- {
		if msgs[i].Role == "user" {
			return i
		}
	}
	// Jika dari 0..target tidak ada `user` lain (satu giliran panjang), cari
	// maju ke `user` berikutnya asal masih menyisakan bagian untuk diringkas.
	for i := target + 1; i < len(msgs); i++ {
		if msgs[i].Role == "user" {
			return i
		}
	}
	return 0
}

func totalActiveChars(msgs []model.AIMessage) int {
	total := 0
	for i := range msgs {
		total += len(msgs[i].Content) + len(msgs[i].ToolCallsJSON)
	}
	return total
}

// CompactionOutcome adalah hasil ringkasan sesi dan pembaruan memori awet.
type CompactionOutcome struct {
	RollingSummary string `json:"rolling_summary"`
	UserMemory     string `json:"user_memory"`
}

// SummarizeSession meminta model ringkas merangkum potongan pesan menjadi
// rolling_summary baru dan user_memory yang diperbarui.
func SummarizeSession(
	ctx context.Context,
	llm LLMClient,
	modelName string,
	prevSummary string,
	prevMemory string,
	folded []model.AIMessage,
) (CompactionOutcome, error) {
	if len(folded) == 0 {
		return CompactionOutcome{RollingSummary: prevSummary, UserMemory: prevMemory}, nil
	}

	var transcript strings.Builder
	for _, m := range folded {
		ts := m.CreatedAt.In(timezone.WIB).Format("02 Jan 15:04 WIB")
		switch m.Role {
		case "user":
			fmt.Fprintf(&transcript, "[%s] USER: %s\n", ts, trimForCompact(m.Content, 1200))
		case "assistant":
			if strings.TrimSpace(m.Content) != "" {
				fmt.Fprintf(&transcript, "[%s] ASSISTANT: %s\n", ts, trimForCompact(m.Content, 1500))
			} else if m.ToolCallsJSON != "" {
				fmt.Fprintf(&transcript, "[%s] ASSISTANT memanggil tool: %s\n", ts, trimForCompact(m.ToolCallsJSON, 300))
			}
		case "tool":
			fmt.Fprintf(&transcript, "[%s] TOOL(%s): %s\n", ts, m.ToolName, trimForCompact(m.Content, 700))
		}
	}

	sysPrompt := `Anda adalah modul compaction memori untuk ILA AI Assistant.
Tugas Anda:
1. Perbarui "rolling_summary" dengan menggabungkan ringkasan sebelumnya dan transkrip baru secara padat, kronologis, dan faktual (maksimal 2500 karakter). Pertahankan simbol pair, timeframe, angka harga/R penting, status setup/posisi, dan keputusan user. DILARANG mengarang fakta yang tidak ada di transkrip.
2. Perbarui "user_memory" (maksimal 1200 karakter, format bullet Markdown) berisi fakta awet tentang user (misal pair favorit, gaya belajar/penjelasan yang diminta, kebiasaan trading yang disebut). Pertahankan fakta lama yang masih relevan, buang yang sudah usang.

Kembalikan HANYA JSON valid tanpa teks tambahan:
{"rolling_summary":"...","user_memory":"..."}`

	userPrompt := fmt.Sprintf(
		"RINGKASAN SEBELUMNYA:\n%s\n\nMEMORI PENGGUNA SEBELUMNYA:\n%s\n\nTRANSKRIP YANG AKAN DIRINGKAS:\n%s",
		nonEmptyOr(prevSummary, "(kosong)"),
		nonEmptyOr(prevMemory, "(kosong)"),
		transcript.String(),
	)

	temp := 0.1
	res, err := llm.Complete(ctx, ChatRequest{
		Model:       modelName,
		Temperature: &temp,
		MaxTokens:   1200,
		Messages: []ChatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return CompactionOutcome{}, err
	}

	out, ok := parseCompactionJSON(res.Content)
	if !ok {
		fallback := strings.TrimSpace(res.Content)
		if fallback == "" {
			fallback = prevSummary
		}
		return CompactionOutcome{
			RollingSummary: trimForCompact(fallback, 3500),
			UserMemory:     prevMemory,
		}, nil
	}
	if strings.TrimSpace(out.RollingSummary) == "" {
		out.RollingSummary = prevSummary
	}
	if strings.TrimSpace(out.UserMemory) == "" {
		out.UserMemory = prevMemory
	}
	out.RollingSummary = trimForCompact(out.RollingSummary, 3500)
	out.UserMemory = trimForCompact(out.UserMemory, 1800)
	return out, nil
}

func parseCompactionJSON(raw string) (CompactionOutcome, bool) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	start := strings.IndexByte(cleaned, '{')
	end := strings.LastIndexByte(cleaned, '}')
	if start >= 0 && end > start {
		cleaned = cleaned[start : end+1]
	}
	var out CompactionOutcome
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return CompactionOutcome{}, false
	}
	return out, true
}

func trimForCompact(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func nonEmptyOr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
