package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/driver/postgres"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ila/backend/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func evalTestDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "eval.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Signal{}, &model.UserSignal{}, &model.Position{}, &model.ExecutionRecord{}); err != nil {
		t.Fatal(err)
	}
	closeEvalDB(t, db)
	return db, path
}

func closeEvalDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
}

func seedEval(t *testing.T, db *gorm.DB, id string, now time.Time) model.UserSignal {
	t.Helper()
	signal := model.Signal{SignalID: id, Symbol: "EURUSD", Direction: "BUY", Ltf: "M1", EntryPrice: 100, StopLoss: 90, TakeProfit: 120, PublishedAt: now.Add(-time.Hour), ExpiresAt: now.Add(-time.Minute)}
	if err := db.Create(&signal).Error; err != nil {
		t.Fatal(err)
	}
	row := model.UserSignal{Base: model.Base{CreatedAt: now.Add(-time.Hour)}, UserID: "u", SignalID: id, Status: model.SignalStatusExpired}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row
}

func TestSignalEvalRestartLeaseSnapshotAndGuard(t *testing.T) {
	db, path := evalTestDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	row := seedEval(t, db, "restart", now)
	ctx := context.Background()
	repo := NewSignalEvalRepository(db)
	jobs, err := repo.Claim(ctx, now, 1)
	if err != nil || len(jobs) != 1 {
		t.Fatalf("claim: %v %v", jobs, err)
	}
	entry := now.Add(-5 * time.Minute)
	checked := now.Add(-time.Minute)
	running := &model.UserSignalEval{Status: model.EvalRunning, EntryAt: &entry, CheckedAt: &checked}
	if err := repo.Save(ctx, jobs[0], running); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Signal{}).Where("signal_id = ?", row.SignalID).Update("entry_price", 200).Error; err != nil {
		t.Fatal(err)
	}
	db2, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	closeEvalDB(t, db2)
	repo = NewSignalEvalRepository(db2)
	if got, err := repo.Claim(ctx, now.Add(time.Minute), 1); err != nil || len(got) != 0 {
		t.Fatalf("lease lost: %v %v", got, err)
	}
	resumed, err := repo.Claim(ctx, now.Add(3*time.Minute), 1)
	if err != nil || len(resumed) != 1 {
		t.Fatalf("resume: %v %v", resumed, err)
	}
	job := resumed[0]
	if job.Signal.EntryPrice != 100 || job.UserSignal.EvalEntryAt == nil || !job.UserSignal.EvalEntryAt.Equal(entry) || job.UserSignal.EvalCheckedAt == nil || !job.UserSignal.EvalCheckedAt.Equal(checked) {
		t.Fatalf("checkpoint/snapshot lost: %+v", job)
	}
	if err := repo.Save(ctx, job, &model.UserSignalEval{Status: model.EvalNoEntry, CheckedAt: &now}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, jobs[0], &model.UserSignalEval{Status: model.EvalSL, EntryAt: &entry, CheckedAt: &now}); err != nil {
		t.Fatal(err)
	}
	var got model.UserSignal
	if err := db.First(&got, "id = ?", row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.EvalStatus != model.EvalRunning {
		t.Fatalf("regressed/stale write: %s", got.EvalStatus)
	}
	if err := repo.Save(ctx, job, &model.UserSignalEval{Status: model.EvalTP, EntryAt: &entry, CheckedAt: &now, Evidence: "TP candle"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, job, running); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&got, "id = ?", row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.EvalStatus != model.EvalTP {
		t.Fatalf("terminal overwritten: %s", got.EvalStatus)
	}
	var evidence struct {
		Setup      model.Signal `json:"setup"`
		EntryAt    *time.Time   `json:"entry_at"`
		EntryPrice float64      `json:"entry_price"`
		Evaluation string       `json:"evaluation"`
	}
	if err := json.Unmarshal([]byte(got.EvalEvidence), &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.Setup.EntryPrice != 100 || evidence.EntryPrice != 100 || evidence.EntryAt == nil || !evidence.EntryAt.Equal(entry) || evidence.Evaluation != "TP candle" {
		t.Fatalf("entry snapshot missing: %s", got.EvalEvidence)
	}
}

func TestSignalEvalBrokerExclusionAtClaimAndSave(t *testing.T) {
	for _, relation := range []string{"consumed", "linked", "position", "claim", "completed"} {
		t.Run(relation, func(t *testing.T) {
			db, _ := evalTestDB(t)
			now := time.Now().UTC().Truncate(time.Second)
			row := seedEval(t, db, "relation", now)
			repo := NewSignalEvalRepository(db)
			ctx := context.Background()
			jobs, err := repo.Claim(ctx, now, 1)
			if err != nil || len(jobs) != 1 {
				t.Fatalf("claim: %v %v", jobs, err)
			}
			switch relation {
			case "consumed":
				err = db.Model(&row).Update("status", model.SignalStatusConsumed).Error
			case "linked":
				err = db.Model(&row).Update("position_id", "p").Error
			case "position":
				err = db.Create(&model.Position{UserID: "u", SourceSignalID: row.SignalID, BrokerAccountID: "b", Symbol: "EURUSD", Direction: "BUY", Status: model.PositionStatusClosedTP}).Error
			default:
				status := "CLAIMED"
				if relation == "completed" {
					status = "COMPLETED"
				}
				err = db.Create(&model.ExecutionRecord{ID: "e", UserID: "u", SignalID: row.SignalID, BrokerAccountID: "b", IdempotencyKey: "key", Status: status}).Error
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.Save(ctx, jobs[0], &model.UserSignalEval{Status: model.EvalTP, CheckedAt: &now}); err != nil {
				t.Fatal(err)
			}
			var got model.UserSignal
			if err := db.First(&got, "id = ?", row.ID).Error; err != nil {
				t.Fatal(err)
			}
			if got.EvalStatus != "" {
				t.Fatalf("broker row evaluated: %+v", got)
			}
			next, err := repo.Claim(ctx, now.Add(3*time.Minute), 1)
			if err != nil || len(next) != 0 {
				t.Fatalf("broker row claimed: %v %v", next, err)
			}
		})
	}
}

func TestSignalEvalBoundedFairRetry(t *testing.T) {
	db, _ := evalTestDB(t)
	now := time.Now().UTC()
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		row := seedEval(t, db, id, now)
		if err := db.Model(&model.Signal{}).Where("signal_id = ?", row.SignalID).Update("symbol", id).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSignalEvalRepository(db)
	first, err := repo.Claim(ctx, now, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first: %v %v", first, err)
	}
	// No Save simulates a feed outage or process crash. It must rotate, then retry.
	second, err := repo.Claim(ctx, now.Add(3*time.Minute), 1)
	if err != nil || len(second) != 1 || second[0].UserSignal.ID == first[0].UserSignal.ID {
		t.Fatalf("starvation: %v %v", second, err)
	}
	all, err := repo.Claim(ctx, now.Add(6*time.Minute), 100000)
	if err != nil || len(all) > SignalEvalBatchLimit {
		t.Fatalf("bound: %v %v", all, err)
	}
	found := false
	for _, job := range all {
		if job.UserSignal.ID == first[0].UserSignal.ID {
			found = true
		}
		if job.UserSignal.EvalStatus != "" || job.UserSignal.EvalCheckedAt != nil {
			t.Fatal("failure fabricated result")
		}
	}
	if !found {
		t.Fatal("failed job never retried")
	}
}

func TestSignalEvalCanonicalEarliestNotWinningOutcome(t *testing.T) {
	db, _ := evalTestDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()
	first := seedEval(t, db, "first", now)
	later := seedEval(t, db, "later", now.Add(time.Minute))
	otherFeed := seedEval(t, db, "feed", now)
	otherStrategy := seedEval(t, db, "strategy", now)
	if err := db.Model(&model.Signal{}).Where("signal_id = ?", otherFeed.SignalID).Update("feed_source", "other").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Signal{}).Where("signal_id = ?", otherStrategy.SignalID).Update("strategy_version", "other").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&later).Update("eval_status", model.EvalTP).Error; err != nil {
		t.Fatal(err)
	}
	jobs, err := NewSignalEvalRepository(db).Claim(ctx, now.Add(2*time.Minute), 100)
	if err != nil || len(jobs) != 3 {
		t.Fatalf("canonical jobs: %v %v", jobs, err)
	}
	for _, job := range jobs {
		if job.UserSignal.ID == later.ID {
			t.Fatal("later winner selected")
		}
	}
	rows, err := NewJournalRepository(db).ListSkipped(ctx, "u", JournalFilter{}, 0, 100)
	if err != nil || len(rows) != 3 {
		t.Fatalf("journal canonical: %v %v", rows, err)
	}
	found := false
	for _, row := range rows {
		if row.ID == first.ID {
			found = true
		}
		if row.ID == later.ID {
			t.Fatal("journal selected later winner")
		}
	}
	if !found {
		t.Fatal("first publication missing")
	}
}

func TestSignalEvalCohortsOnlyOverlapValidity(t *testing.T) {
	db, _ := evalTestDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()
	first := seedEval(t, db, "old", now)
	boundary := seedEval(t, db, "boundary", now)
	fresh := seedEval(t, db, "fresh", now)
	expires := now.Add(-time.Minute)
	for id, published := range map[string]time.Time{boundary.SignalID: expires, fresh.SignalID: expires.Add(time.Second)} {
		if err := db.Model(&model.Signal{}).Where("signal_id = ?", id).Updates(map[string]any{"published_at": published, "expires_at": expires}).Error; err != nil {
			t.Fatal(err)
		}
	}
	// At the boundary it is still a duplicate; one second beyond expiry it is new.
	jobs, err := NewSignalEvalRepository(db).Claim(ctx, now, 100)
	if err != nil || len(jobs) != 2 {
		t.Fatalf("cohort claims: %v %v", jobs, err)
	}
	ids := map[string]bool{}
	for _, job := range jobs {
		ids[job.UserSignal.ID] = true
	}
	if !ids[first.ID] || !ids[fresh.ID] || ids[boundary.ID] {
		t.Fatalf("wrong claim cohorts: %v", ids)
	}
	journal := NewJournalRepository(db)
	rows, err := journal.ListSkipped(ctx, "u", JournalFilter{}, 0, 100)
	if err != nil || len(rows) != 2 {
		t.Fatalf("journal cohorts: %v %v", rows, err)
	}
	count, err := journal.CountSkipped(ctx, "u", JournalFilter{})
	if err != nil || count != 2 {
		t.Fatalf("journal count=%d err=%v", count, err)
	}
}

func TestJournalEvalStatusFilterListCountAndCanonical(t *testing.T) {
	db, _ := evalTestDB(t)
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()
	statuses := []string{"", model.EvalPending, model.EvalRunning, model.EvalTP, model.EvalSL, model.EvalNoEntry, model.EvalUnknown}
	for i, status := range statuses {
		id := fmt.Sprintf("status-%d", i)
		row := seedEval(t, db, id, now)
		if err := db.Model(&model.Signal{}).Where("signal_id = ?", id).Update("symbol", id).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&row).Update("eval_status", status).Error; err != nil {
			t.Fatal(err)
		}
	}
	// A duplicate with a different outcome must not become canonical just to match the filter.
	duplicate := seedEval(t, db, "duplicate", now.Add(time.Second))
	if err := db.Model(&model.Signal{}).Where("signal_id = ?", duplicate.SignalID).Update("symbol", "status-0").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&duplicate).Update("eval_status", model.EvalTP).Error; err != nil {
		t.Fatal(err)
	}
	nullRow := seedEval(t, db, "null", now)
	if err := db.Model(&model.Signal{}).Where("signal_id = ?", nullRow.SignalID).Update("symbol", "null").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&nullRow).Update("eval_status", nil).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewSignalEvalRepository(db).Claim(ctx, now, 100); err != nil {
		t.Fatal(err)
	}
	repo := NewJournalRepository(db)
	for _, status := range statuses {
		want := 1
		if status == "" {
			want = 8
		}
		if status == model.EvalPending {
			want = 3
		}
		filter := JournalFilter{EvalStatus: status}
		rows, err := repo.ListSkipped(ctx, "u", filter, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		count, err := repo.CountSkipped(ctx, "u", filter)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != want || count != want {
			t.Fatalf("%q list=%d count=%d want=%d", status, len(rows), count, want)
		}
		for _, row := range rows {
			if row.ID == duplicate.ID {
				t.Fatal("filter changed canonical selection")
			}
		}
		page, err := repo.ListSkipped(ctx, "u", filter, 0, 1)
		if err != nil || len(page) != 1 {
			t.Fatalf("page: %v %v", page, err)
		}
	}
}

func TestFrozenCohortDoesNotChainOrFollowMutableSignal(t *testing.T) {
	db, _ := evalTestDB(t)
	ctx := context.Background()
	start := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	rows := map[string]model.UserSignal{}
	for _, spec := range []struct {
		id             string
		offset, expiry time.Duration
	}{{"A", 0, time.Hour}, {"B", 30 * time.Minute, 90 * time.Minute}, {"C", 75 * time.Minute, 2 * time.Hour}} {
		row := seedEval(t, db, spec.id, start)
		rows[spec.id] = row
		if err := db.Model(&model.Signal{}).Where("signal_id = ?", spec.id).Updates(map[string]any{"published_at": start.Add(spec.offset), "expires_at": start.Add(spec.expiry)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSignalEvalRepository(db)
	jobs, err := repo.Claim(ctx, start.Add(3*time.Hour), 100)
	if err != nil || len(jobs) != 2 {
		t.Fatalf("roots A/C: %v %v", jobs, err)
	}
	for _, job := range jobs {
		if job.Signal.SignalID == "B" {
			t.Fatal("hidden B became root")
		}
		entry := job.Signal.PublishedAt
		checked := entry.Add(time.Minute)
		if err := repo.Save(ctx, job, &model.UserSignalEval{Status: model.EvalRunning, EntryAt: &entry, CheckedAt: &checked, Evidence: "running"}); err != nil {
			t.Fatal(err)
		}
	}
	var b model.UserSignal
	if err := db.First(&b, "id = ?", rows["B"].ID).Error; err != nil {
		t.Fatal(err)
	}
	if b.EvalCanonicalID != rows["A"].ID {
		t.Fatalf("B cohort: %+v", b)
	}
	// Mutating feed and extending A through C cannot change either frozen root.
	if err := db.Model(&model.Signal{}).Where("signal_id = ?", "A").Updates(map[string]any{"feed_source": "mutated", "expires_at": start.Add(24 * time.Hour), "entry_price": 999}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Signal{}).Where("signal_id = ?", "B").Updates(map[string]any{"expires_at": start.Add(48 * time.Hour), "feed_source": "mutated"}).Error; err != nil {
		t.Fatal(err)
	}
	resumed, err := repo.Claim(ctx, start.Add(4*time.Hour), 100)
	if err != nil || len(resumed) != 2 {
		t.Fatalf("running roots disappeared: %v %v", resumed, err)
	}
	for _, job := range resumed {
		if job.Signal.FeedSource != "" || job.Signal.EntryPrice != 100 || job.UserSignal.EvalCheckedAt == nil {
			t.Fatalf("mutable snapshot leak: %+v", job)
		}
	}
	listed, err := NewJournalRepository(db).ListSkipped(ctx, "u", JournalFilter{}, 0, 100)
	if err != nil || len(listed) != 2 {
		t.Fatalf("journal roots lost: %v %v", listed, err)
	}
	// A fresh unclassified signal uses the frozen A/C windows, not extended mutable A/B.
	fresh := seedEval(t, db, "D", start)
	if err := db.Model(&model.Signal{}).Where("signal_id = ?", fresh.SignalID).Updates(map[string]any{"published_at": start.Add(3 * time.Hour), "expires_at": start.Add(4 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	next, err := repo.Claim(ctx, start.Add(5*time.Hour), 100)
	if err != nil || len(next) != 3 {
		t.Fatalf("new frozen-window cohort missing: %v %v", next, err)
	}
}

func TestExistingSnapshotsAreNeverRededuplicated(t *testing.T) {
	db, _ := evalTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, id := range []string{"committed-a", "committed-b"} {
		row := seedEval(t, db, id, now)
		var signal model.Signal
		if err := db.Where("signal_id = ?", id).First(&signal).Error; err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(signal)
		if err != nil {
			t.Fatal(err)
		}
		if err := db.Model(&row).Updates(map[string]any{"eval_snapshot": string(data), "eval_status": model.EvalTP}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if _, err := NewSignalEvalRepository(db).Claim(ctx, now, 100); err != nil {
		t.Fatal(err)
	}
	rows, err := NewJournalRepository(db).ListSkipped(ctx, "u", JournalFilter{EvalStatus: model.EvalTP}, 0, 100)
	if err != nil || len(rows) != 2 {
		t.Fatalf("committed results disappeared: %v %v", rows, err)
	}
}

func TestForwardPostgresClaimAndCheckpoint(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN kosong")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	closeEvalDB(t, db)
	if err := db.AutoMigrate(&model.Signal{}, &model.UserSignal{}, &model.Position{}, &model.ExecutionRecord{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	id := fmt.Sprintf("forward-pg-%d", now.UnixNano())
	row := seedEval(t, db, id, now)
	t.Cleanup(func() {
		if err := db.Unscoped().Delete(&model.UserSignal{}, "id = ?", row.ID).Error; err != nil {
			t.Error(err)
		}
		if err := db.Unscoped().Delete(&model.Signal{}, "signal_id = ?", id).Error; err != nil {
			t.Error(err)
		}
	})
	repo := NewSignalEvalRepository(db)
	jobs, err := repo.Claim(context.Background(), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	var candidate *SignalEvalCandidate
	for i := range jobs {
		if jobs[i].UserSignal.ID == row.ID {
			candidate = &jobs[i]
		}
	}
	if candidate == nil {
		t.Fatal("Postgres claim did not return candidate")
	}
	entry, checked := candidate.Signal.PublishedAt, now
	if err := repo.Save(context.Background(), *candidate, &model.UserSignalEval{Status: model.EvalRunning, EntryAt: &entry, CheckedAt: &checked, Evidence: "checkpoint"}); err != nil {
		t.Fatal(err)
	}
	var saved model.UserSignal
	if err := db.First(&saved, "id = ?", row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.EvalStatus != model.EvalRunning || saved.EvalCheckedAt == nil {
		t.Fatalf("checkpoint lost: %+v", saved)
	}
}
