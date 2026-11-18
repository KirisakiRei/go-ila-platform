package repository

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
)

func TestGlobalSignalCohortClassificationAndOutcomePropagation(t *testing.T) {
	db, _ := evalTestDB(t)
	ctx := context.Background()
	repo := NewSignalEvalRepository(db)

	t1 := time.Date(2026, 11, 18, 1, 35, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 18, 2, 0, 0, 0, time.UTC)

	// Dua terbitan identik TANPA baris user_signal sama sekali (meniru pair watchlist).
	s1 := model.Signal{
		SignalID:        "sig-XAG-1",
		Symbol:          "XAGUSD",
		StrategyVersion: "ila-3.0",
		FeedSource:      "FOREXCOM",
		Htf:             "4H",
		Ltf:             "5M",
		Direction:       "BUY",
		EntryPrice:      46.967,
		StopLoss:        46.4585,
		TakeProfit:      48.23825,
		RiskReward:      2.5,
		Status:          model.SignalStatusExpired,
		PublishedAt:     t1,
		ExpiresAt:       t1.Add(5 * time.Minute),
	}
	s2 := s1
	s2.SignalID = "sig-XAG-2"
	s2.Htf = "H4" // alias timeframe yang sama tetap harus satu kohort
	s2.Ltf = "M5"
	s2.PublishedAt = t2
	s2.ExpiresAt = t2.Add(5 * time.Minute)

	if err := db.Create(&s1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&s2).Error; err != nil {
		t.Fatal(err)
	}

	now := t2.Add(10 * time.Minute)
	jobs, err := repo.ClaimSignals(ctx, now, 10)
	if err != nil {
		t.Fatalf("ClaimSignals gagal: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("seharusnya hanya 1 akar setup yang diklaim, dapat %d", len(jobs))
	}
	if jobs[0].Signal.SignalID != "sig-XAG-1" {
		t.Fatalf("akar kanonik seharusnya terbitan pertama sig-XAG-1, dapat %q", jobs[0].Signal.SignalID)
	}

	tpAt := t2.Add(25 * time.Minute)
	checked := t2.Add(30 * time.Minute)
	out := &model.UserSignalEval{
		Status:    model.EvalTP,
		EntryAt:   &t1,
		At:        &tpAt,
		Price:     48.23825,
		RLower:    2.5,
		CheckedAt: &checked,
		Evidence:  `{"method":"signal_forward_v1"}`,
	}
	if err := repo.SaveSignal(ctx, jobs[0], out); err != nil {
		t.Fatalf("SaveSignal gagal: %v", err)
	}

	var stored []model.Signal
	if err := db.Order("published_at ASC").Find(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 {
		t.Fatalf("jumlah baris signal = %d, ingin 2", len(stored))
	}
	for _, row := range stored {
		if row.SetupCanonicalID != "sig-XAG-1" {
			t.Fatalf("signal %s punya canonical %q, ingin sig-XAG-1", row.SignalID, row.SetupCanonicalID)
		}
		if row.SetupEvalStatus != model.EvalTP || row.SetupEvalRLower != 2.5 {
			t.Fatalf("signal %s belum tersinkron status TP: %+v", row.SignalID, row)
		}
	}
}
