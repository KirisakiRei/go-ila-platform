package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"ila/backend/internal/model"
)

func TestJournalUsesPersistedEvaluationAndSnapshot(t *testing.T) {
	now := time.Now().UTC()
	snapshot, err := json.Marshal(model.Signal{SignalID: "s", Symbol: "EURUSD", EntryPrice: 100, PublishedAt: now.Add(-time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	detail := &mutatedJournalSignal{}
	svc := &tradingService{signalDetail: detail}
	out := svc.jurnalSkippedResponse(context.Background(), []model.UserSignal{{SignalID: "s", EvalStatus: model.EvalRunning, EvalEntryAt: &now, EvalCheckedAt: &now, EvalEvidence: "entry candle", EvalSnapshot: string(snapshot)}})
	if len(out) != 1 || !out[0].OccurredAt.Equal(now.Add(-time.Hour)) {
		t.Fatalf("signal time must come from publication: %+v", out)
	}
	if detail.calls != 0 {
		t.Fatal("snapshot unexpectedly read mutated signal")
	}
	if len(out) != 1 || out[0].EvalStatus != model.EvalRunning || out[0].EvalCheckedAt == nil || out[0].EvalEvidence != "entry candle" || out[0].EntryPrice != 100 || out[0].Symbol != "EURUSD" {
		t.Fatalf("persisted DTO lost: %+v", out)
	}
}

type mutatedJournalSignal struct{ calls int }

func (r *mutatedJournalSignal) FindSignal(context.Context, string) (*model.Signal, error) {
	r.calls++
	return &model.Signal{Symbol: "MUTATED", EntryPrice: 999}, nil
}

func TestJournalCorruptSnapshotDoesNotSubstituteMutatedLevels(t *testing.T) {
	detail := &mutatedJournalSignal{}
	svc := &tradingService{signalDetail: detail}
	rows := svc.jurnalSkippedResponse(context.Background(), []model.UserSignal{{SignalID: "s", EvalStatus: model.EvalTP, EvalSnapshot: "{broken", EvalEvidence: "saved proof"}})
	if detail.calls != 0 || len(rows) != 1 || rows[0].EntryPrice != 0 || rows[0].EvalStatus != model.EvalTP || rows[0].EvalEvidence != "saved proof" {
		t.Fatalf("corrupt snapshot substituted live data: %+v calls=%d", rows, detail.calls)
	}
}
