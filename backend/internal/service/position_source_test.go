package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

type positionSourceFixture struct {
	fakeSignalLister
	calls int
	ids   []string
	err   error
}

func (f *positionSourceFixture) FindPositionSources(_ context.Context, ids []string) (map[string]model.Signal, error) {
	f.calls++
	f.ids = ids
	return map[string]model.Signal{"source": {SignalID: "source", Htf: "H4", Ltf: "M5", FeedSource: "ICMARKETS", EntryPrice: 999}}, f.err
}

type positionJournalFixture struct{ rows []model.Position }

func (f *positionJournalFixture) ListPositions(context.Context, string, repository.JournalFilter, int) ([]model.Position, error) {
	return f.rows, nil
}
func (*positionJournalFixture) Stats(context.Context, string, repository.JournalFilter) (repository.JournalStats, error) {
	return repository.JournalStats{}, nil
}
func (*positionJournalFixture) ListSkipped(context.Context, string, repository.JournalFilter, int, int) ([]model.UserSignal, error) {
	return nil, nil
}
func (*positionJournalFixture) CountSkipped(context.Context, string, repository.JournalFilter) (int, error) {
	return 0, nil
}

func TestPositionSourcesBatchAndEndpointParity(t *testing.T) {
	rows := []model.Position{
		{SourceSignalID: "source", BrokerAccountID: "broker", OpenPrice: 100, StopLoss: 90, TakeProfit: 120, ExitPrice: 119},
		{SourceSignalID: "source", OpenPrice: 101},
		{SourceSignalID: "missing"},
		{},
	}
	source := &positionSourceFixture{}
	svc := NewTradingService(nil, source, &fakePositionLister{positions: rows}, nil).
		WithJournal(&positionJournalFixture{rows: rows}, nil)
	positions, err := svc.Positions(context.Background(), "user", 100)
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 1 || !reflect.DeepEqual(source.ids, []string{"source", "missing"}) {
		t.Fatalf("not a deduplicated batch: %+v", source)
	}
	journal, err := svc.Journal(context.Background(), "user", JournalQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if source.calls != 2 || !reflect.DeepEqual(positions, journal.Positions) {
		t.Fatalf("endpoint mapping differs: %+v / %+v", positions, journal.Positions)
	}
	got := positions[0]
	if got.SourceSignalID != "source" || got.Htf != "H4" || got.Ltf != "M5" || got.FeedSource != "ICMARKETS" || got.BrokerAccountID != "broker" {
		t.Fatalf("lost metadata: %+v", got)
	}
	if got.OpenPrice != 100 || got.StopLoss != 90 || got.TakeProfit != 120 || got.ExitPrice != 119 {
		t.Fatalf("source overwrote broker prices: %+v", got)
	}
	if positions[2].Htf != "" || positions[3].SourceSignalID != "" {
		t.Fatalf("missing source should stay unknown: %+v", positions)
	}
	source.err = errors.New("lookup failed")
	if _, err := svc.Positions(context.Background(), "user", 100); !errors.Is(err, source.err) {
		t.Fatalf("lost lookup error: %v", err)
	}
	if _, err := svc.Journal(context.Background(), "user", JournalQuery{}); !errors.Is(err, source.err) {
		t.Fatalf("lost journal lookup error: %v", err)
	}
}

func TestPositionSourcesSkipEmptyLists(t *testing.T) {
	source := &positionSourceFixture{}
	svc := NewTradingService(nil, source, &fakePositionLister{positions: []model.Position{{}}}, nil)
	if _, err := svc.Positions(context.Background(), "user", 100); err != nil {
		t.Fatal(err)
	}
	if source.calls != 0 {
		t.Fatal("manual positions must not query signal metadata")
	}
}
