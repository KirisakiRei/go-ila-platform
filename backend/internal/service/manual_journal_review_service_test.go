package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"ila/backend/internal/assistant"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type reviewJournalStub struct{ stats dto.ManualJournalStats }

func (j *reviewJournalStub) GetAccount(context.Context, string, string) (*model.ManualJournalAccount, error) {
	return &model.ManualJournalAccount{}, nil
}
func (j *reviewJournalStub) Stats(context.Context, string, dto.ManualJournalFilter) (*dto.ManualJournalStats, error) {
	v := j.stats
	return &v, nil
}

type reviewClientStub struct {
	calls int
	fn    func(assistant.ChatRequest) (*assistant.ChatResult, error)
}

func (c *reviewClientStub) Complete(_ context.Context, r assistant.ChatRequest) (*assistant.ChatResult, error) {
	c.calls++
	return c.fn(r)
}
func reviewFixture(t *testing.T) (*gorm.DB, *repository.ManualJournalReviewRepository, *ManualJournalReviewService, string, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sql.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&model.User{}, &model.ManualJournalAccount{}, &model.ManualJournalEntry{}, &model.ManualJournalCapital{}, &model.ManualJournalReview{}, &model.ManualJournalReviewSetting{}); err != nil {
		t.Fatal(err)
	}
	user, account := uuid.NewString(), uuid.NewString()
	if err := db.Create(&model.User{Base: model.Base{ID: user}, Email: user + "@example.test", Username: user, Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ManualJournalAccount{Base: model.Base{ID: account}, UserID: user, Currency: "USD", InitialCapital: "1000", InitialDate: "2026-01-01"}).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewManualJournalReviewRepository(db)
	s := NewManualJournalReviewService(repo, &reviewJournalStub{stats: dto.ManualJournalStats{Trades: 5, NetPNL: "-10", WinRate: "20", MaxDrawdown: "10"}}, nil, false, "")
	s.now = func() time.Time { return time.Date(2026, 11, 26, 16, 59, 0, 0, time.UTC) }
	return db, repo, s, user, account
}
func TestManualJournalReviewConsentDefaults(t *testing.T) {
	for _, global := range []string{"unset", "false", "true"} {
		for _, accountSetting := range []string{"unset", "false", "true"} {
			t.Run(global+"/"+accountSetting, func(t *testing.T) {
				_, repo, s, user, account := reviewFixture(t)
				ctx := context.Background()
				s.enabled, s.model = true, "test"
				client := &reviewClientStub{fn: func(assistant.ChatRequest) (*assistant.ChatResult, error) {
					return &assistant.ChatResult{Content: `["Tetap evaluasi proses dengan tenang."]`}, nil
				}}
				s.client = client
				for _, preference := range []struct{ scope, value string }{{"", global}, {account, accountSetting}} {
					if preference.value == "unset" {
						continue
					}
					enabled := preference.value == "true"
					if _, err := s.SaveSettings(ctx, user, preference.scope, ManualJournalReviewSettingsInput{AIEnabled: &enabled}); err != nil {
						t.Fatal(err)
					}
				}
				globalView, err := s.Settings(ctx, user, "")
				if err != nil || globalView.AIEnabled != (global == "true") || globalView.EffectiveAIEnabled != (global == "true") {
					t.Fatalf("global settings %+v: %v", globalView, err)
				}
				want := global == "true" && accountSetting != "false"
				view, err := s.Settings(ctx, user, account)
				if err != nil || view.AIEnabled != (accountSetting != "false") || view.EffectiveAIEnabled != want {
					t.Fatalf("account settings %+v: %v", view, err)
				}
				row, err := s.Get(ctx, user, account)
				if err != nil {
					t.Fatal(err)
				}
				claimed, err := repo.ClaimReview(ctx, *row, "consent", s.now())
				if err != nil || !claimed {
					t.Fatal("claim", err)
				}
				authorized, err := repo.AuthorizeReview(ctx, *row, "consent", s.now())
				if err != nil || authorized != want {
					t.Fatalf("authorization=%v want=%v: %v", authorized, want, err)
				}
				_, source := s.generate(ctx, *row, "consent", manualReviewAggregate{}, view.EffectiveAIEnabled)
				if (client.calls == 1) != want || (!want && (client.calls != 0 || source != "rules")) {
					t.Fatalf("calls=%d source=%s wantAI=%v", client.calls, source, want)
				}
			})
		}
	}
}

type reviewAuthorizeHook struct {
	ManualJournalReviewStore
	before func()
}

func (r *reviewAuthorizeHook) AuthorizeReview(ctx context.Context, row model.ManualJournalReview, token string, now time.Time) (bool, error) {
	if r.before != nil {
		fn := r.before
		r.before = nil
		fn()
	}
	return r.ManualJournalReviewStore.AuthorizeReview(ctx, row, token, now)
}

func TestManualJournalReviewUserInactiveBeforeSend(t *testing.T) {
	for _, field := range []string{"status", "suspended_at", "banned_at", "deleted_at", "deleted_at_user"} {
		for _, retry := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/retry=%v", field, retry), func(t *testing.T) {
				db, repo, s, user, account := reviewFixture(t)
				ctx := context.Background()
				s.enabled, s.model = true, "test"
				deactivate := func() {
					var value any = s.now()
					if field == "status" {
						value = "inactive"
					}
					if err := db.Model(&model.User{}).Where("id = ?", user).Update(field, value).Error; err != nil {
						t.Fatal(err)
					}
				}
				client := &reviewClientStub{fn: func(assistant.ChatRequest) (*assistant.ChatResult, error) {
					if !retry {
						t.Fatal("provider called after user became inactive")
					}
					deactivate()
					return nil, errors.New("first attempt failed")
				}}
				s.client = client
				yes := true
				if _, err := s.SaveSettings(ctx, user, "", ManualJournalReviewSettingsInput{AIEnabled: &yes}); err != nil {
					t.Fatal(err)
				}
				if !retry {
					s.repo = &reviewAuthorizeHook{ManualJournalReviewStore: repo, before: func() {
						var claimed model.ManualJournalReview
						if err := db.Where("user_id = ? AND account_id = ?", user, account).First(&claimed).Error; err != nil {
							t.Fatal(err)
						}
						if claimed.Status != "running" || claimed.Claim == "" {
							t.Fatal("user status must change after claim")
						}
						deactivate()
					}}
				}
				if err := s.ReviewAccount(ctx, user, account); err != nil {
					t.Fatal(err)
				}
				wantCalls := 0
				if retry {
					wantCalls = 1
				}
				if client.calls != wantCalls {
					t.Fatalf("calls=%d want=%d", client.calls, wantCalls)
				}
				row, err := s.Get(ctx, user, account)
				if err != nil || row.Source != "rules" {
					t.Fatalf("fallback %+v: %v", row, err)
				}
			})
		}
	}
}

func TestManualJournalReviewHashAndWIB(t *testing.T) {
	db, _, s, user, account := reviewFixture(t)
	ctx := context.Background()
	first, err := s.Get(ctx, user, account)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.Get(ctx, user, account)
	if err != nil || first.Revision != again.Revision {
		t.Fatalf("unstable revision: %v", err)
	}
	entry := model.ManualJournalEntry{UserID: user, AccountID: account}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatal(err)
	}
	changed, err := s.Get(ctx, user, account)
	if err != nil || changed.Revision == first.Revision {
		t.Fatalf("insert not invalidated: %v", err)
	}
	if err := db.Model(&entry).Update("updated_at", s.now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	edited, err := s.Get(ctx, user, account)
	if err != nil || edited.Revision == changed.Revision {
		t.Fatalf("edit not invalidated: %v", err)
	}
	if err := db.Delete(&entry).Error; err != nil {
		t.Fatal(err)
	}
	deleted, err := s.Get(ctx, user, account)
	if err != nil || deleted.Revision == edited.Revision {
		t.Fatalf("delete not invalidated: %v", err)
	}
	s.now = func() time.Time { return time.Date(2026, 11, 26, 17, 0, 0, 0, time.UTC) }
	tomorrow, err := s.Get(ctx, user, account)
	if err != nil || tomorrow.Date != "2026-11-27" || tomorrow.Revision == deleted.Revision {
		t.Fatalf("WIB rollover: %+v %v", tomorrow, err)
	}
}
func TestManualJournalReviewFallbackDedupAndPrivacy(t *testing.T) {
	_, _, s, user, account := reviewFixture(t)
	ctx := context.Background()
	if err := s.ReviewAccount(ctx, user, account); err != nil {
		t.Fatal(err)
	}
	row, err := s.Get(ctx, user, account)
	if err != nil || row.Status != "ready" || row.Source != "rules" || row.Text == "" {
		t.Fatalf("disabled fallback: %+v %v", row, err)
	}
	c := &reviewClientStub{fn: func(r assistant.ChatRequest) (*assistant.ChatResult, error) {
		for _, m := range r.Messages {
			if strings.Contains(m.Content, "PRIVATE_NOTE") {
				t.Fatal("raw prose leaked")
			}
		}
		return nil, errors.New("upstream down")
	}}
	j := s.journal.(*reviewJournalStub)
	j.stats.Breakdowns = map[string]map[string]dto.ManualJournalGroup{"setup": {"PRIVATE_NOTE": {Trades: 5}}}
	s.client = c
	s.enabled = true
	s.model = "test"
	yes := true
	if _, err := s.SaveSettings(ctx, user, "", ManualJournalReviewSettingsInput{AIEnabled: &yes}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReviewAccount(ctx, user, account); err != nil {
		t.Fatal(err)
	}
	row, err = s.Get(ctx, user, account)
	if err != nil || row.Source != "rules" || c.calls != 2 {
		t.Fatalf("bounded fallback %+v calls=%d err=%v", row, c.calls, err)
	}
	if err := s.ReviewAccount(ctx, user, account); err != nil {
		t.Fatal(err)
	}
	if c.calls != 2 {
		t.Fatal("ready revision called AI again")
	}
	no := false
	if _, err := s.SaveSettings(ctx, user, "", ManualJournalReviewSettingsInput{AIEnabled: &no}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReviewAccount(ctx, user, account); err != nil {
		t.Fatal(err)
	}
	if c.calls != 2 {
		t.Fatal("opt-out called AI")
	}
}
func TestManualJournalReviewStaleAndClaim(t *testing.T) {
	db, repo, s, user, account := reviewFixture(t)
	ctx := context.Background()
	row, err := s.Get(ctx, user, account)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := repo.ClaimReview(ctx, *row, "first", s.now())
	if err != nil || !ok {
		t.Fatal("first claim", err)
	}
	ok, err = repo.ClaimReview(ctx, *row, "second", s.now())
	if err != nil || ok {
		t.Fatal("duplicate claim", err)
	}
	newer := *row
	newer.Revision = "new"
	newer.Status = "pending"
	if _, err := repo.Prepare(ctx, newer); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishReview(ctx, *row, "first", "stale", "ai", s.now()); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Prepare(ctx, newer)
	if err != nil || got.Text != "" || got.Status != "pending" {
		t.Fatalf("stale overwrite %+v %v", got, err)
	}
	ok, err = repo.ClaimReview(ctx, newer, "new", s.now())
	if err != nil || !ok {
		t.Fatal(err)
	}
	if err := repo.FinishReview(ctx, newer, "new", "expired", "ai", s.now().Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err = repo.Prepare(ctx, newer)
	if err != nil || got.Text != "" {
		t.Fatal("expired lease published", err)
	}
	no := false
	if _, err := s.SaveSettings(ctx, user, account, ManualJournalReviewSettingsInput{AIEnabled: &no}); err != nil {
		t.Fatal(err)
	}
	if err := repo.FinishReview(ctx, newer, "new", "privacy stale", "ai", s.now()); err != nil {
		t.Fatal(err)
	}
	var persisted model.ManualJournalReview
	if err := db.WithContext(ctx).Where("user_id = ? AND account_id = ? AND date = ?", user, account, row.Date).First(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.Text != "" || persisted.Claim != "" {
		t.Fatal("opt-out did not revoke result")
	}
}

func TestManualJournalReviewText(t *testing.T) {
	for _, bad := range []string{"plain text", `[]`, `["One.","Two.","Three."]`, `["One. Two."]`, `["Buy now!"]`} {
		if _, ok := manualReviewText(bad); ok {
			t.Fatalf("accepted %q", bad)
		}
	}
	if _, ok := manualReviewText(`["Tetap catat prosesmu dengan tenang.","Jaga batas risiko yang nyaman."]`); !ok {
		t.Fatal("rejected valid sentences")
	}
}

func TestManualJournalReviewAIAndMidflightInvalidation(t *testing.T) {
	db, _, s, user, account := reviewFixture(t)
	ctx := context.Background()
	s.enabled = true
	s.model = "test"
	s.client = &reviewClientStub{fn: func(r assistant.ChatRequest) (*assistant.ChatResult, error) {
		if r.Model != "test" || len(r.Tools) != 0 {
			t.Fatal("unexpected model/tools")
		}
		if !strings.Contains(r.Messages[0].Content, "all recorded history") || !strings.Contains(r.Messages[0].Content, "never describe it as a yesterday recap") {
			t.Fatal("review period not explicit in prompt")
		}
		return &assistant.ChatResult{Content: `["Tetap evaluasi proses dengan tenang."]`}, nil
	}}
	yes := true
	if _, err := s.SaveSettings(ctx, user, "", ManualJournalReviewSettingsInput{AIEnabled: &yes}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReviewAccount(ctx, user, account); err != nil {
		t.Fatal(err)
	}
	row, err := s.Get(ctx, user, account)
	if err != nil || row.Source != "ai" {
		t.Fatalf("AI result %+v %v", row, err)
	}
	s.model = "changed"
	s.client = &reviewClientStub{fn: func(assistant.ChatRequest) (*assistant.ChatResult, error) {
		if err := db.Create(&model.ManualJournalEntry{UserID: user, AccountID: account}).Error; err != nil {
			t.Fatal(err)
		}
		return &assistant.ChatResult{Content: `["Hasil lama tidak boleh diterbitkan."]`}, nil
	}}
	if err := s.ReviewAccount(ctx, user, account); err != nil {
		t.Fatal(err)
	}
	row, err = s.Get(ctx, user, account)
	if err != nil || row.Status != "pending" || row.Text != "" {
		t.Fatalf("midflight stale published %+v %v", row, err)
	}
}

type reviewPrepareHook struct {
	ManualJournalReviewStore
	before func()
}

func (r *reviewPrepareHook) Prepare(ctx context.Context, row model.ManualJournalReview) (*model.ManualJournalReview, error) {
	if r.before != nil {
		fn := r.before
		r.before = nil
		fn()
	}
	return r.ManualJournalReviewStore.Prepare(ctx, row)
}

func TestManualJournalReviewOptOutBeforeClaim(t *testing.T) {
	for _, userWide := range []bool{false, true} {
		t.Run(fmt.Sprint(userWide), func(t *testing.T) {
			db, repo, s, user, account := reviewFixture(t)
			ctx := context.Background()
			s.enabled = true
			s.model = "test"
			client := &reviewClientStub{fn: func(assistant.ChatRequest) (*assistant.ChatResult, error) {
				t.Fatal("sent after opt-out")
				return nil, nil
			}}
			s.client = client
			yes := true
			if _, err := s.SaveSettings(ctx, user, "", ManualJournalReviewSettingsInput{AIEnabled: &yes}); err != nil {
				t.Fatal(err)
			}
			s.repo = &reviewPrepareHook{ManualJournalReviewStore: repo, before: func() {
				scope := account
				if userWide {
					scope = ""
				}
				if err := repo.SaveReviewSetting(ctx, model.ManualJournalReviewSetting{UserID: user, AccountID: scope, AIEnabled: false, UpdatedAt: s.now()}); err != nil {
					t.Fatal(err)
				}
			}}
			if err := s.ReviewAccount(ctx, user, account); err != nil {
				t.Fatal(err)
			}
			var cached model.ManualJournalReview
			if err := db.Where("user_id = ? AND account_id = ?", user, account).First(&cached).Error; err != nil {
				t.Fatal(err)
			}
			if cached.Revision != "" || cached.Claim != "" || client.calls != 0 {
				t.Fatalf("stale prepare revived claim: %+v calls=%d", cached, client.calls)
			}
			if err := s.ReviewAccount(ctx, user, account); err != nil {
				t.Fatal(err)
			}
			row, err := s.Get(ctx, user, account)
			if err != nil || row.Source != "rules" {
				t.Fatalf("opt-out fallback %+v %v", row, err)
			}
		})
	}
}

func TestManualJournalReviewOptOutDuringRetry(t *testing.T) {
	for _, reenable := range []bool{false, true} {
		t.Run(fmt.Sprint(reenable), func(t *testing.T) {
			_, repo, s, user, account := reviewFixture(t)
			ctx := context.Background()
			s.enabled = true
			s.model = "test"
			client := &reviewClientStub{fn: func(assistant.ChatRequest) (*assistant.ChatResult, error) {
				for _, enabled := range []bool{false, true} {
					if enabled && !reenable {
						break
					}
					if err := repo.SaveReviewSetting(ctx, model.ManualJournalReviewSetting{UserID: user, AIEnabled: enabled, UpdatedAt: s.now()}); err != nil {
						t.Fatal(err)
					}
				}
				return nil, errors.New("first attempt failed")
			}}
			s.client = client
			yes := true
			if _, err := s.SaveSettings(ctx, user, "", ManualJournalReviewSettingsInput{AIEnabled: &yes}); err != nil {
				t.Fatal(err)
			}
			if err := s.ReviewAccount(ctx, user, account); err != nil {
				t.Fatal(err)
			}
			if client.calls != 1 {
				t.Fatalf("retry sent after preference change: %d", client.calls)
			}
			row, err := s.Get(ctx, user, account)
			if err != nil || row.Text != "" || row.Status != "pending" {
				t.Fatalf("stale output %+v %v", row, err)
			}
		})
	}
}

func TestManualJournalReviewExistingClaimCannotReviveAfterOptOut(t *testing.T) {
	_, repo, s, user, account := reviewFixture(t)
	ctx := context.Background()
	s.enabled = true
	s.model = "test"
	s.client = &reviewClientStub{}
	yes := true
	if _, err := s.SaveSettings(ctx, user, "", ManualJournalReviewSettingsInput{AIEnabled: &yes}); err != nil {
		t.Fatal(err)
	}
	stale, err := s.Get(ctx, user, account)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := repo.ClaimReview(ctx, *stale, "old", s.now())
	if err != nil || !claimed {
		t.Fatal("claim", err)
	}
	authorized, err := repo.AuthorizeReview(ctx, *stale, "old", s.now())
	if err != nil || !authorized {
		t.Fatal("initial authorization", err)
	}
	for _, enabled := range []bool{false, true} {
		if err := repo.SaveReviewSetting(ctx, model.ManualJournalReviewSetting{UserID: user, AccountID: account, AIEnabled: enabled, UpdatedAt: s.now()}); err != nil {
			t.Fatal(err)
		}
		cached, err := repo.Prepare(ctx, *stale)
		if err != nil {
			t.Fatal(err)
		}
		if cached.Revision != "" || cached.Claim != "" {
			t.Fatal("stale prepare revived cache")
		}
		claimed, err = repo.ClaimReview(ctx, *stale, "revived", s.now())
		if err != nil || claimed {
			t.Fatal("stale claim authorized", err)
		}
		authorized, err = repo.AuthorizeReview(ctx, *stale, "old", s.now())
		if err != nil || authorized {
			t.Fatal("revoked send authorized", err)
		}
	}
}

type reviewPollStore struct {
	ManualJournalReviewStore
	accounts []model.ManualJournalAccount
	scans    []string
}

func (r *reviewPollStore) ReviewAccounts(ctx context.Context, after string) ([]model.ManualJournalAccount, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.scans = append(r.scans, after)
	var out []model.ManualJournalAccount
	for _, a := range r.accounts {
		if a.ID > after {
			out = append(out, a)
		}
	}
	return out, nil
}

type reviewPollJournal struct {
	reviewJournalStub
	visited      []string
	firstTimeout bool
	overlap      func()
}

func (j *reviewPollJournal) Stats(ctx context.Context, user string, f dto.ManualJournalFilter) (*dto.ManualJournalStats, error) {
	j.visited = append(j.visited, f.AccountID)
	if j.overlap != nil {
		fn := j.overlap
		j.overlap = nil
		fn()
	}
	if j.firstTimeout {
		j.firstTimeout = false
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return j.reviewJournalStub.Stats(ctx, user, f)
}
func TestManualJournalReviewPollResumesAfterTimeout(t *testing.T) {
	_, repo, s, user, account := reviewFixture(t)
	ids := []string{account, uuid.NewString(), uuid.NewString()}
	sort.Strings(ids)
	store := &reviewPollStore{ManualJournalReviewStore: repo}
	for _, id := range ids {
		store.accounts = append(store.accounts, model.ManualJournalAccount{Base: model.Base{ID: id}, UserID: user})
	}
	journal := &reviewPollJournal{firstTimeout: true}
	s.repo = store
	s.journal = journal
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := s.Poll(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout not propagated: %v", err)
	}
	if len(journal.visited) != 1 || journal.visited[0] != ids[0] || s.pollAfter != ids[0] {
		t.Fatalf("advanced past unprocessed account: %v cursor=%s", journal.visited, s.pollAfter)
	}
	journal.overlap = func() {
		if err := s.Poll(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Successful review takes a second snapshot; account one must not be retried this tick.
	want := []string{ids[0], ids[1], ids[1], ids[2], ids[2]}
	if !reflect.DeepEqual(journal.visited, want) {
		t.Fatalf("visits=%v want=%v", journal.visited, want)
	}
	if !reflect.DeepEqual(store.scans, []string{"", ids[0]}) || s.pollAfter != "" {
		t.Fatalf("resume/reset scans=%v cursor=%s", store.scans, s.pollAfter)
	}
	visits, scans := len(journal.visited), len(store.scans)
	if err := s.Poll(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expired context: %v", err)
	}
	if len(journal.visited) != visits || len(store.scans) != scans {
		t.Fatal("expired tick performed work")
	}
	if err := s.Poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.scans[len(store.scans)-1] != "" || journal.visited[visits] != ids[0] {
		t.Fatal("full scan did not restart at beginning")
	}
}
