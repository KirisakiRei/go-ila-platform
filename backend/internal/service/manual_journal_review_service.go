package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ila/backend/internal/assistant"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"

	"github.com/google/uuid"
)

type ManualJournalReviewStore interface {
	ReviewAccounts(context.Context, string) ([]model.ManualJournalAccount, error)
	Fingerprint(context.Context, string, string) (string, error)
	Prepare(context.Context, model.ManualJournalReview) (*model.ManualJournalReview, error)
	ClaimReview(context.Context, model.ManualJournalReview, string, time.Time) (bool, error)
	AuthorizeReview(context.Context, model.ManualJournalReview, string, time.Time) (bool, error)
	FinishReview(context.Context, model.ManualJournalReview, string, string, string, time.Time) error
	ReviewSettings(context.Context, string, string) ([]model.ManualJournalReviewSetting, error)
	SaveReviewSetting(context.Context, model.ManualJournalReviewSetting) error
}
type ManualJournalReviewJournal interface {
	GetAccount(context.Context, string, string) (*model.ManualJournalAccount, error)
	Stats(context.Context, string, dto.ManualJournalFilter) (*dto.ManualJournalStats, error)
}
type ManualJournalReviewClient interface {
	Complete(context.Context, assistant.ChatRequest) (*assistant.ChatResult, error)
}
type ManualJournalReviewService struct {
	repo      ManualJournalReviewStore
	journal   ManualJournalReviewJournal
	client    ManualJournalReviewClient
	enabled   bool
	model     string
	now       func() time.Time
	pollMu    sync.Mutex
	pollAfter string
}

// Pass the existing assistant client and cfg.AIEnabled/cfg.AIModel; no chat storage is used.
func NewManualJournalReviewService(repo ManualJournalReviewStore, journal ManualJournalReviewJournal, client ManualJournalReviewClient, enabled bool, aiModel string) *ManualJournalReviewService {
	return &ManualJournalReviewService{repo: repo, journal: journal, client: client, enabled: enabled, model: aiModel, now: time.Now}
}

type ManualJournalReviewSettingsInput struct {
	AIEnabled *bool `json:"ai_enabled" binding:"required"`
}
type ManualJournalReviewSettingsView struct {
	generation         int64
	AccountID          string `json:"account_id"`
	AIEnabled          bool   `json:"ai_enabled"`
	EffectiveAIEnabled bool   `json:"effective_ai_enabled"`
}

func (s *ManualJournalReviewService) scope(ctx context.Context, user, account string, optional bool) error {
	if err := manualIdentity(user); err != nil {
		return err
	}
	if account == "" && optional {
		return nil
	}
	if err := manualID(account); err != nil {
		return err
	}
	_, err := s.journal.GetAccount(ctx, user, account)
	return err
}
func (s *ManualJournalReviewService) Settings(ctx context.Context, user, account string) (*ManualJournalReviewSettingsView, error) {
	if err := s.scope(ctx, user, account, true); err != nil {
		return nil, err
	}
	rows, err := s.repo.ReviewSettings(ctx, user, account)
	if err != nil {
		return nil, err
	}
	// Accounts default on, but sending always requires explicit global consent.
	out := &ManualJournalReviewSettingsView{AccountID: account, AIEnabled: account != ""}
	globalEnabled := false
	for _, row := range rows {
		out.generation += row.Generation
		if row.AccountID == account {
			out.AIEnabled = row.AIEnabled
		}
		if row.AccountID == "" {
			globalEnabled = row.AIEnabled
		}
	}
	out.EffectiveAIEnabled = globalEnabled && out.AIEnabled && s.enabled && s.client != nil && s.model != ""
	return out, nil
}
func (s *ManualJournalReviewService) SaveSettings(ctx context.Context, user, account string, in ManualJournalReviewSettingsInput) (*ManualJournalReviewSettingsView, error) {
	if err := s.scope(ctx, user, account, true); err != nil {
		return nil, err
	}
	if in.AIEnabled == nil {
		return nil, manualBad("ai_enabled is required")
	}
	if err := s.repo.SaveReviewSetting(ctx, model.ManualJournalReviewSetting{UserID: user, AccountID: account, AIEnabled: *in.AIEnabled, UpdatedAt: s.now()}); err != nil {
		return nil, err
	}
	return s.Settings(ctx, user, account)
}

// Only numeric aggregate fields are persisted in the hash or sent to the assistant.
type manualReviewAggregate struct {
	Trades      int    `json:"trades"`
	Wins        int    `json:"wins"`
	Losses      int    `json:"losses"`
	Breakeven   int    `json:"breakeven"`
	NetPNL      string `json:"net_pnl"`
	WinRate     string `json:"win_rate"`
	MaxDrawdown string `json:"max_drawdown"`
}

func manualReviewDay(now time.Time) string { return now.In(manualJournalWIB).Format("2006-01-02") }
func manualReviewRevision(fingerprint, day, aiModel string, enabled bool, generation int64, a manualReviewAggregate) (string, error) {
	b, err := json.Marshal([]any{"manual-review-v2", fingerprint, day, aiModel, enabled, generation, a})
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func (s *ManualJournalReviewService) snapshot(ctx context.Context, user, account string) (model.ManualJournalReview, manualReviewAggregate, bool, error) {
	var row model.ManualJournalReview
	var a manualReviewAggregate
	if err := s.scope(ctx, user, account, false); err != nil {
		return row, a, false, err
	}
	settings, err := s.Settings(ctx, user, account)
	if err != nil {
		return row, a, false, err
	}
	day := manualReviewDay(s.now())
	before, err := s.repo.Fingerprint(ctx, user, account)
	if err != nil {
		return row, a, false, err
	}
	stats, err := s.journal.Stats(ctx, user, dto.ManualJournalFilter{AccountID: account, To: day})
	if err != nil {
		return row, a, false, err
	}
	after, err := s.repo.Fingerprint(ctx, user, account)
	if err != nil {
		return row, a, false, err
	}
	if before != after {
		return row, a, false, errors.New("journal changed during review snapshot; retry next poll")
	}
	a = manualReviewAggregate{stats.Trades, stats.Wins, stats.Losses, stats.Breakeven, stats.NetPNL, stats.WinRate, stats.MaxDrawdown}
	revision, err := manualReviewRevision(after, day, s.model, settings.EffectiveAIEnabled, settings.generation, a)
	row = model.ManualJournalReview{SettingsGeneration: settings.generation, UserID: user, AccountID: account, Date: day, Revision: revision, Status: "pending", UpdatedAt: s.now()}
	return row, a, settings.EffectiveAIEnabled, err
}

// Get enqueues by persistent upsert only; HTTP never waits for the model.
func (s *ManualJournalReviewService) Get(ctx context.Context, user, account string) (*model.ManualJournalReview, error) {
	row, _, _, err := s.snapshot(ctx, user, account)
	if err != nil {
		return nil, err
	}
	return s.repo.Prepare(ctx, row)
}
func manualReviewFallback(a manualReviewAggregate) string {
	if a.Trades == 0 {
		return "Belum ada trade tertutup untuk ditinjau. Tetap catat prosesmu dengan tenang."
	}
	if a.Trades < 20 {
		return "Sampel trade masih kecil, jadi belum perlu menarik kesimpulan besar. Terus jaga konsistensi pencatatan dan batas risiko."
	}
	n, err := manualRat(a.NetPNL)
	if err == nil && n.Sign() < 0 {
		return "Hasil agregat masih negatif, dan itu bisa menjadi bahan evaluasi yang tenang. Tinjau kembali batas risiko sebelum melanjutkan."
	}
	return "Terima kasih sudah menjaga pencatatan tradingmu. Tetap evaluasi proses dan pertahankan batas risiko yang nyaman."
}
func manualReviewText(content string) (string, bool) {
	var sentences []string
	if json.Unmarshal([]byte(content), &sentences) != nil || len(sentences) < 1 || len(sentences) > 2 {
		return "", false
	}
	for i, v := range sentences {
		v = strings.TrimSpace(v)
		if utf8.RuneCountInString(v) < 8 || utf8.RuneCountInString(v) > 180 || !strings.HasSuffix(v, ".") || strings.ContainsAny(strings.TrimSuffix(v, "."), ".!?\n\r<>`") {
			return "", false
		}
		sentences[i] = v
	}
	return strings.Join(sentences, " "), true
}
func (s *ManualJournalReviewService) generate(ctx context.Context, row model.ManualJournalReview, token string, a manualReviewAggregate, enabled bool) (string, string) {
	fallback := manualReviewFallback(a)
	if !enabled {
		return fallback, "rules"
	}
	payload, err := json.Marshal(a)
	if err != nil {
		return fallback, "rules"
	}
	// Two attempts share a 20-second deadline, well below the one-minute claim lease.
	callCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	for attempt := 0; attempt < 2; attempt++ {
		if callCtx.Err() != nil {
			break
		}
		// A committed opt-out revokes generation and claim; fail closed even on read errors.
		authorized, err := s.repo.AuthorizeReview(callCtx, row, token, s.now())
		if err != nil || !authorized {
			break
		}
		result, err := s.client.Complete(callCtx, assistant.ChatRequest{Model: s.model, MaxTokens: 180, Messages: []assistant.ChatMessage{
			{Role: "system", Content: "Write a gentle trading journal reflection in Indonesian using only the numeric aggregate. It covers all recorded history through the current WIB date, not yesterday or a single day; never describe it as a yesterday recap. Return a JSON array of 1 or 2 short sentences, each ending in a period. No markdown, questions, trade instructions, predictions, diagnosis, guarantees, or invented facts. Do not infer discipline or causes from profit. If fewer than 20 trades, acknowledge limited evidence. Encourage calm process review and risk limits."},
			{Role: "user", Content: string(payload)},
		}})
		if err == nil && result != nil && len(result.ToolCalls) == 0 {
			if text, ok := manualReviewText(result.Content); ok {
				return text, "ai"
			}
		}
	}
	return fallback, "rules"
}

// ReviewAccount is also the optional post-save hook. Poll is sufficient without hooks.
func (s *ManualJournalReviewService) ReviewAccount(ctx context.Context, user, account string) error {
	row, a, enabled, err := s.snapshot(ctx, user, account)
	if err != nil {
		return err
	}
	cached, err := s.repo.Prepare(ctx, row)
	if err != nil {
		return err
	}
	if cached.Revision != row.Revision {
		return nil
	}
	token := uuid.NewString()
	claimed, err := s.repo.ClaimReview(ctx, row, token, s.now())
	if err != nil || !claimed {
		return err
	}
	text, source := s.generate(ctx, row, token, a, enabled)
	if err := ctx.Err(); err != nil {
		return err
	}
	latest, _, _, err := s.snapshot(ctx, user, account)
	if err != nil {
		return err
	}
	if latest.Date != row.Date || latest.Revision != row.Revision {
		_, err = s.repo.Prepare(ctx, latest)
		return err
	}
	return s.repo.FinishReview(ctx, row, token, text, source, s.now())
}

// Poll resumes between ticks on this service instance; restarting the process resets the cursor.
func (s *ManualJournalReviewService) Poll(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Overlapping ticks skip rather than wait behind a potentially slow provider call.
	if !s.pollMu.TryLock() {
		return nil
	}
	defer s.pollMu.Unlock()
	var failures error
	for {
		if err := ctx.Err(); err != nil {
			return errors.Join(failures, err)
		}
		accounts, err := s.repo.ReviewAccounts(ctx, s.pollAfter)
		if err != nil {
			return errors.Join(failures, err)
		}
		for _, account := range accounts {
			if err := ctx.Err(); err != nil {
				return errors.Join(failures, err)
			}
			if err := s.ReviewAccount(ctx, account.UserID, account.ID); err != nil {
				failures = errors.Join(failures, fmt.Errorf("manual review account %s: %w", account.ID, err))
			}
			// Advance even on failure/cancellation so one slow account cannot starve the rest.
			s.pollAfter = account.ID
		}
		if len(accounts) < 100 {
			s.pollAfter = ""
			return errors.Join(failures, ctx.Err())
		}
	}
}
