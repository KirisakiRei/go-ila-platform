package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"ila/backend/internal/assistant"
	"ila/backend/internal/assistant/knowledge"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/timezone"
	"ila/backend/pkg/apperror"
)

const maxAssistantToolRounds = 5

// AssistantMessageView adalah format satu pesan chat untuk frontend.
type AssistantMessageView struct {
	ID        string                   `json:"id"`
	Seq       int64                    `json:"seq"`
	Role      string                   `json:"role"`
	Content   string                   `json:"content"`
	ToolsUsed []assistant.UsedToolMeta `json:"tools_used,omitempty"`
	CreatedAt time.Time                `json:"created_at"`
}

// AssistantSessionView adalah ringkasan sesi tunggal ILA AI untuk user.
type AssistantSessionView struct {
	Enabled            bool                   `json:"enabled"`
	SessionID          string                 `json:"session_id,omitempty"`
	LastMessageAt      *time.Time             `json:"last_message_at,omitempty"`
	LastCompactedAt    *time.Time             `json:"last_compacted_at,omitempty"`
	CompactedBeforeSeq int64                  `json:"compacted_before_seq"`
	ActiveMessageCount int64                  `json:"active_message_count"`
	HasSummary         bool                   `json:"has_summary"`
	HasMore            bool                   `json:"has_more"`
	Messages           []AssistantMessageView `json:"messages"`
}

// AssistantServiceDeps mengumpulkan seluruh dependensi ILA AI Assistant.
type AssistantServiceDeps struct {
	Enabled      bool
	Model        string
	CompactModel string
	Repo         repository.AssistantRepository
	LLM          assistant.LLMClient
	Tools        assistant.ToolDeps
	CompactOpts  assistant.CompactionOptions
	Logger       *slog.Logger
	Clock        func() time.Time
}

// AssistantService mengatur sesi tunggal, compaction, tool loop, dan streaming ILA AI.
type AssistantService interface {
	Enabled() bool
	GetSession(ctx context.Context, userID string, beforeSeq int64, limit int) (*AssistantSessionView, error)
	CompactSession(ctx context.Context, userID string) (*AssistantSessionView, error)
	ChatStream(ctx context.Context, userID, userText string, emit func(assistant.StreamEvent) error) error
}

type assistantService struct {
	deps      AssistantServiceDeps
	userLocks sync.Map // map[string]*sync.Mutex
}

// NewAssistantService membuat layanan ILA AI.
func NewAssistantService(deps AssistantServiceDeps) AssistantService {
	if deps.Clock == nil {
		deps.Clock = func() time.Time { return time.Now().UTC() }
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if deps.CompactOpts.Model == "" {
		if strings.TrimSpace(deps.CompactModel) != "" {
			deps.CompactOpts.Model = strings.TrimSpace(deps.CompactModel)
		} else {
			deps.CompactOpts.Model = strings.TrimSpace(deps.Model)
		}
	}
	return &assistantService{deps: deps}
}

func (s *assistantService) Enabled() bool {
	return s.deps.Enabled && s.deps.LLM != nil && strings.TrimSpace(s.deps.Model) != ""
}

func (s *assistantService) lockUser(userID string) func() {
	val, _ := s.userLocks.LoadOrStore(userID, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (s *assistantService) GetSession(ctx context.Context, userID string, beforeSeq int64, limit int) (*AssistantSessionView, error) {
	if !s.Enabled() {
		return &AssistantSessionView{
			Enabled:  false,
			Messages: []AssistantMessageView{},
		}, nil
	}

	sess, err := s.deps.Repo.GetOrCreateSession(ctx, userID)
	if err != nil {
		return nil, err
	}
	msgs, hasMore, err := s.deps.Repo.ListDisplayMessages(ctx, sess.ID, beforeSeq, limit)
	if err != nil {
		return nil, err
	}
	activeCount, err := s.deps.Repo.CountActiveMessages(ctx, sess.ID)
	if err != nil {
		return nil, err
	}

	return &AssistantSessionView{
		Enabled:            true,
		SessionID:          sess.ID,
		LastMessageAt:      sess.LastMessageAt,
		LastCompactedAt:    sess.LastCompactedAt,
		CompactedBeforeSeq: sess.CompactedBeforeSeq,
		ActiveMessageCount: activeCount,
		HasSummary:         strings.TrimSpace(sess.RollingSummary) != "",
		HasMore:            hasMore,
		Messages:           toAssistantMessageViews(msgs),
	}, nil
}

func (s *assistantService) CompactSession(ctx context.Context, userID string) (*AssistantSessionView, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("%w: asisten AI belum diaktifkan", apperror.ErrBadRequest)
	}

	unlock := s.lockUser(userID)
	defer unlock()

	sess, err := s.deps.Repo.GetOrCreateSession(ctx, userID)
	if err != nil {
		return nil, err
	}
	active, err := s.deps.Repo.ListActiveMessages(ctx, sess.ID, 200)
	if err != nil {
		return nil, err
	}

	plan := assistant.PlanManualCompaction(active)
	if len(plan.ToFold) > 0 {
		mem, err := s.deps.Repo.GetUserMemory(ctx, userID)
		if err != nil {
			return nil, err
		}
		out, err := assistant.SummarizeSession(ctx, s.deps.LLM, s.deps.CompactOpts.Model, sess.RollingSummary, mem.MemoryMarkdown, plan.ToFold)
		if err != nil {
			return nil, fmt.Errorf("gagal meringkas sesi: %w", err)
		}
		if err := s.deps.Repo.ApplyCompaction(ctx, sess.ID, userID, plan.UpToSeq, out.RollingSummary, out.UserMemory, s.deps.Clock()); err != nil {
			return nil, err
		}
	}

	// Lepas lock sebentar sebelum membaca ulang view tidak wajib, tapi aman
	// langsung baca karena masih dalam kendali caller tunggal ini.
	sess, err = s.deps.Repo.GetOrCreateSession(ctx, userID)
	if err != nil {
		return nil, err
	}
	msgs, hasMore, err := s.deps.Repo.ListDisplayMessages(ctx, sess.ID, 0, 20)
	if err != nil {
		return nil, err
	}
	activeCount, err := s.deps.Repo.CountActiveMessages(ctx, sess.ID)
	if err != nil {
		return nil, err
	}

	return &AssistantSessionView{
		Enabled:            true,
		SessionID:          sess.ID,
		LastMessageAt:      sess.LastMessageAt,
		LastCompactedAt:    sess.LastCompactedAt,
		CompactedBeforeSeq: sess.CompactedBeforeSeq,
		ActiveMessageCount: activeCount,
		HasSummary:         strings.TrimSpace(sess.RollingSummary) != "",
		HasMore:            hasMore,
		Messages:           toAssistantMessageViews(msgs),
	}, nil
}

func (s *assistantService) ChatStream(ctx context.Context, userID, userText string, emit func(assistant.StreamEvent) error) error {
	if !s.Enabled() {
		return fmt.Errorf("%w: ILA AI belum diaktifkan oleh admin", apperror.ErrBadRequest)
	}
	userText = strings.TrimSpace(userText)
	if userText == "" || len([]rune(userText)) > 2000 {
		return fmt.Errorf("%w: pesan wajib 1 sampai 2000 karakter", apperror.ErrBadRequest)
	}

	baseKnowledge, err := knowledge.BaseSystemPrompt()
	if err != nil {
		return err
	}

	unlock := s.lockUser(userID)
	defer unlock()

	now := s.deps.Clock()
	sess, err := s.deps.Repo.GetOrCreateSession(ctx, userID)
	if err != nil {
		return err
	}
	mem, err := s.deps.Repo.GetUserMemory(ctx, userID)
	if err != nil {
		return err
	}
	active, err := s.deps.Repo.ListActiveMessages(ctx, sess.ID, 200)
	if err != nil {
		return err
	}

	compacted := false
	plan := assistant.PlanCompaction(sess, active, now, s.deps.CompactOpts)
	if plan.Reason != assistant.CompactNone && len(plan.ToFold) > 0 {
		_ = emit(assistant.StreamEvent{
			Type:  "status",
			Stage: "compacting",
			Label: "Menyusun ringkasan konteks percakapan...",
		})
		out, compErr := assistant.SummarizeSession(ctx, s.deps.LLM, s.deps.CompactOpts.Model, sess.RollingSummary, mem.MemoryMarkdown, plan.ToFold)
		if compErr != nil {
			s.deps.Logger.Warn("compaction gagal, memakai jendela pesan terbaru",
				"user_id", userID, "reason", plan.Reason, "error", compErr.Error())
			active = fallbackRecentWindow(active, s.deps.CompactOpts.KeepRecent)
		} else {
			if applyErr := s.deps.Repo.ApplyCompaction(ctx, sess.ID, userID, plan.UpToSeq, out.RollingSummary, out.UserMemory, now); applyErr != nil {
				s.deps.Logger.Warn("menyimpan hasil compaction gagal", "user_id", userID, "error", applyErr.Error())
			} else {
				compacted = true
				sess.RollingSummary = out.RollingSummary
				mem.MemoryMarkdown = out.UserMemory
				var remaining []model.AIMessage
				for _, m := range active {
					if m.Seq > plan.UpToSeq {
						remaining = append(remaining, m)
					}
				}
				active = remaining
			}
		}
	}

	if err := emit(assistant.StreamEvent{
		Type:  "status",
		Stage: "thinking",
		Label: "Sedang berpikir...",
	}); err != nil {
		return err
	}

	sysPrompt := buildSystemPrompt(baseKnowledge, mem.MemoryMarkdown, sess.RollingSummary, now)
	llmMessages := make([]assistant.ChatMessage, 0, len(active)+10)
	llmMessages = append(llmMessages, assistant.ChatMessage{Role: "system", Content: sysPrompt})
	llmMessages = append(llmMessages, toLLMHistory(active)...)
	llmMessages = append(llmMessages, assistant.ChatMessage{Role: "user", Content: userText})

	pending := []model.AIMessage{
		{Role: "user", Content: userText, CreatedAt: now},
	}
	var usedTools []assistant.UsedToolMeta
	toolsSchema := assistant.ToolDefinitions()
	temp := 0.2

	var finalAssistantContent string
	for round := 0; round < maxAssistantToolRounds; round++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Pada putaran terakhir, matikan tool agar LLM wajib menutup dengan jawaban teks.
		roundTools := toolsSchema
		if round == maxAssistantToolRounds-1 {
			roundTools = nil
		}

		streamStarted := false
		res, err := s.deps.LLM.Stream(ctx, assistant.ChatRequest{
			Model:       s.deps.Model,
			Messages:    llmMessages,
			Tools:       roundTools,
			Temperature: &temp,
			MaxTokens:   2048,
		}, func(delta string) error {
			if delta == "" {
				return nil
			}
			if !streamStarted {
				streamStarted = true
				if err := emit(assistant.StreamEvent{
					Type:  "status",
					Stage: "streaming",
					Label: "Menyusun jawaban...",
				}); err != nil {
					return err
				}
			}
			return emit(assistant.StreamEvent{Type: "delta", Text: delta})
		})
		if err != nil {
			return err
		}

		if len(res.ToolCalls) == 0 {
			finalAssistantContent = strings.TrimSpace(res.Content)
			if finalAssistantContent == "" {
				finalAssistantContent = "Maaf, saya belum menerima jawaban yang lengkap. Silakan ulangi pertanyaan Anda."
				_ = emit(assistant.StreamEvent{Type: "delta", Text: finalAssistantContent})
			}
			usedJSON, _ := json.Marshal(usedTools)
			if len(usedTools) == 0 {
				usedJSON = nil
			}
			pending = append(pending, model.AIMessage{
				Role:          "assistant",
				Content:       finalAssistantContent,
				ToolsUsedJSON: string(usedJSON),
				CreatedAt:     s.deps.Clock(),
			})
			break
		}

		tcJSON, _ := json.Marshal(res.ToolCalls)
		pending = append(pending, model.AIMessage{
			Role:          "assistant",
			Content:       res.Content,
			ToolCallsJSON: string(tcJSON),
			CreatedAt:     s.deps.Clock(),
		})
		llmMessages = append(llmMessages, assistant.ChatMessage{
			Role:      "assistant",
			Content:   res.Content,
			ToolCalls: res.ToolCalls,
		})

		for _, tc := range res.ToolCalls {
			label := assistant.ToolStartLabel(tc.Function.Name, tc.Function.Arguments)
			if err := emit(assistant.StreamEvent{
				Type:  "tool_start",
				Name:  tc.Function.Name,
				Label: label,
			}); err != nil {
				return err
			}

			execRes := assistant.ExecuteTool(ctx, s.deps.Tools, userID, tc.Function.Name, tc.Function.Arguments)
			okFlag := execRes.OK
			if err := emit(assistant.StreamEvent{
				Type:    "tool_done",
				Name:    tc.Function.Name,
				OK:      &okFlag,
				Summary: execRes.Summary,
			}); err != nil {
				return err
			}
			if execRes.Action != nil {
				if err := emit(assistant.StreamEvent{
					Type:   "action_applied",
					Kind:   execRes.Action.Kind,
					Symbol: execRes.Action.Symbol,
				}); err != nil {
					return err
				}
			}

			usedTools = append(usedTools, assistant.UsedToolMeta{
				Name:    tc.Function.Name,
				Label:   label,
				OK:      execRes.OK,
				Summary: execRes.Summary,
			})
			pending = append(pending, model.AIMessage{
				Role:       "tool",
				Content:    execRes.OutputJSON,
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				CreatedAt:  s.deps.Clock(),
			})
			llmMessages = append(llmMessages, assistant.ChatMessage{
				Role:       "tool",
				Content:    execRes.OutputJSON,
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
			})
		}

		if err := emit(assistant.StreamEvent{
			Type:  "status",
			Stage: "thinking",
			Label: "Menganalisis hasil data...",
		}); err != nil {
			return err
		}
	}

	saved, err := s.deps.Repo.AppendMessages(ctx, sess.ID, userID, s.deps.Clock(), pending)
	if err != nil {
		return err
	}
	last := saved[len(saved)-1]
	return emit(assistant.StreamEvent{
		Type:    "done",
		MsgID:   last.ID,
		Seq:     last.Seq,
		Compact: compacted,
	})
}

func buildSystemPrompt(baseKnowledge, userMemory, rollingSummary string, now time.Time) string {
	var b strings.Builder
	b.WriteString(baseKnowledge)
	b.WriteString("\n\n---\n\n")
	b.WriteString(fmt.Sprintf("[WAKTU SAAT INI]\n- WIB: %s\n- New York: %s\n",
		now.In(timezone.WIB).Format("02 Jan 2006 15:04 WIB (Monday)"),
		now.In(timezone.NY).Format("02 Jan 2006 15:04 NY (Monday)"),
	))
	if strings.TrimSpace(userMemory) != "" {
		b.WriteString("\n[MEMORI PENGGUNA]\n")
		b.WriteString(strings.TrimSpace(userMemory))
		b.WriteString("\n")
	}
	if strings.TrimSpace(rollingSummary) != "" {
		b.WriteString("\n[RINGKASAN SESI SEBELUMNYA]\n")
		b.WriteString(strings.TrimSpace(rollingSummary))
		b.WriteString("\n")
	}
	return b.String()
}

func toLLMHistory(active []model.AIMessage) []assistant.ChatMessage {
	out := make([]assistant.ChatMessage, 0, len(active))
	for _, m := range active {
		msg := assistant.ChatMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			Name:       m.ToolName,
		}
		if m.ToolCallsJSON != "" {
			var tcs []assistant.ToolCall
			if json.Unmarshal([]byte(m.ToolCallsJSON), &tcs) == nil {
				msg.ToolCalls = tcs
			}
		}
		out = append(out, msg)
	}
	return out
}

func toAssistantMessageViews(rows []model.AIMessage) []AssistantMessageView {
	out := make([]AssistantMessageView, 0, len(rows))
	for _, r := range rows {
		var tools []assistant.UsedToolMeta
		if r.ToolsUsedJSON != "" {
			_ = json.Unmarshal([]byte(r.ToolsUsedJSON), &tools)
		}
		out = append(out, AssistantMessageView{
			ID:        r.ID,
			Seq:       r.Seq,
			Role:      r.Role,
			Content:   r.Content,
			ToolsUsed: tools,
			CreatedAt: r.CreatedAt,
		})
	}
	return out
}

func fallbackRecentWindow(active []model.AIMessage, keep int) []model.AIMessage {
	if keep <= 0 {
		keep = 8
	}
	if len(active) <= keep {
		return active
	}
	start := len(active) - keep
	for i := start; i < len(active); i++ {
		if active[i].Role == "user" {
			return active[i:]
		}
	}
	return active[start:]
}
