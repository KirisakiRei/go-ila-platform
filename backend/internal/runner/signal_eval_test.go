package runner_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"ila/backend/internal/market"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestForwardMonitorRestartGroupedFetchAndFailureRetry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitor.db")
	open := func() *gorm.DB {
		db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := sqlDB.Close(); err != nil {
				t.Error(err)
			}
		})
		return db
	}
	db := open()
	if err := db.AutoMigrate(&model.Signal{}, &model.UserSignal{}, &model.Position{}, &model.ExecutionRecord{}); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	now := start.Add(2 * time.Minute)
	signal := model.Signal{SignalID: "forward", Symbol: "EURUSD", FeedSource: "test", Ltf: "M1", Direction: "BUY", EntryPrice: 100, StopLoss: 90, TakeProfit: 120, RiskReward: 2, PublishedAt: start, ExpiresAt: start.Add(time.Minute)}
	if err := db.Create(&signal).Error; err != nil {
		t.Fatal(err)
	}
	for _, user := range []string{"u1", "u2"} {
		if err := db.Create(&model.UserSignal{UserID: user, SignalID: signal.SignalID, Status: model.SignalStatusSkipped}).Error; err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	failing := true
	candles := []market.Candle{{Time: start.Unix(), Open: 101, High: 105, Low: 101, Close: 102}}
	fetch := func(ctx context.Context, symbol, feed, tf string, limit int) ([]market.Candle, error) {
		calls++
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("missing fetch timeout")
		}
		if limit != runner.EvalCandleLimit || symbol != "EURUSD" || feed != "test" || tf != "M1" {
			t.Fatal("wrong feed contract")
		}
		if failing {
			return nil, errors.New("offline")
		}
		return candles, nil
	}
	run := func(db *gorm.DB) {
		svc := runner.NewService(newFakeStore(), nil, &fakePublisher{}, discardLogger(), runner.Options{Clock: func() time.Time { return now }, CandleFetcher: fetch, Evaluations: repository.NewSignalEvalRepository(db)})
		svc.MonitorSignalEvaluations(context.Background())
	}
	rows := func() []model.UserSignal {
		var out []model.UserSignal
		if err := db.Order("user_id").Find(&out).Error; err != nil {
			t.Fatal(err)
		}
		return out
	}
	run(db)
	if calls != 1 {
		t.Fatalf("fetch not grouped: %d", calls)
	}
	for _, row := range rows() {
		if row.EvalStatus != "" || row.EvalCheckedAt != nil {
			t.Fatal("feed failure fabricated result")
		}
	}
	now = now.Add(3 * time.Minute)
	failing = false
	run(db)
	for _, row := range rows() {
		if row.EvalStatus != model.EvalRunning || row.EvalEntryAt == nil || !row.EvalEntryAt.Equal(start) || row.EvalCheckedAt == nil {
			t.Fatalf("running not persisted: %+v", row)
		}
	}
	// Theoretical entry is at publication, even though no candle retested its price.
	// A new runner resumes after TTL without needing the original candle.
	db = open()
	now = now.Add(3 * time.Minute)
	candles = []market.Candle{{Time: start.Add(time.Minute).Unix(), Open: 102, High: 121, Low: 101, Close: 120}}
	run(db)
	for _, row := range rows() {
		if row.EvalStatus != model.EvalTP || row.EvalEntryAt == nil || !row.EvalEntryAt.Equal(start) {
			t.Fatalf("restart lost entry/outcome: %+v", row)
		}
	}
	if calls != 3 {
		t.Fatalf("expected one fetch per batch, got %d", calls)
	}
	// OHLC from the publication's partial candle cannot establish whether TP
	// happened before or after publication. It must not become a fake win.
	partialStart := start.Add(30 * time.Second)
	partial := model.Signal{SignalID: "partial", Symbol: "EURUSD", FeedSource: "test", Ltf: "M1", Direction: "BUY", EntryPrice: 101, StopLoss: 90, TakeProfit: 120, PublishedAt: partialStart, ExpiresAt: start.Add(time.Minute)}
	if err := db.Create(&partial).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserSignal{UserID: "partial-user", SignalID: partial.SignalID, Status: model.SignalStatusSkipped}).Error; err != nil {
		t.Fatal(err)
	}
	candles = []market.Candle{{Time: start.Unix(), Open: 101, High: 121, Low: 100, Close: 102}}
	run(db)
	var unsafe model.UserSignal
	if err := db.Where("user_id = ?", "partial-user").First(&unsafe).Error; err != nil {
		t.Fatal(err)
	}
	if unsafe.EvalStatus != model.EvalUnknown {
		t.Fatalf("partial publication candle produced unsafe result: %+v", unsafe)
	}
}
