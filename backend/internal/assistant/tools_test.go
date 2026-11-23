package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/market"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

type fakeAccounts struct {
	subs      []dto.SubscriptionResponse
	accounts  []dto.AccountResponse
	subErr    error
	unsubIDs  []string
	subSymbol string
}

func (f *fakeAccounts) List(_ context.Context, _ string) ([]dto.AccountResponse, error) {
	return f.accounts, nil
}
func (f *fakeAccounts) ListSubscriptions(_ context.Context, _ string) ([]dto.SubscriptionResponse, error) {
	return f.subs, nil
}
func (f *fakeAccounts) Subscribe(_ context.Context, _ string, req dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error) {
	if f.subErr != nil {
		return nil, f.subErr
	}
	f.subSymbol = req.Symbol
	return &dto.SubscriptionResponse{ID: "sub-new", Symbol: req.Symbol, IsActive: true}, nil
}
func (f *fakeAccounts) Unsubscribe(_ context.Context, _, id string) error {
	f.unsubIDs = append(f.unsubIDs, id)
	return nil
}

type fakeCandles struct {
	candles []market.Candle
	err     error
}

func (f *fakeCandles) Candles(_ context.Context, _, _, _ string, _ int) ([]market.Candle, string, error) {
	if f.err != nil {
		return nil, "FOREXCOM", f.err
	}
	return f.candles, "FOREXCOM", nil
}

type fakePairs struct{}

func (fakePairs) Catalog(_ context.Context) (dto.PairCatalogResponse, error) {
	return dto.PairCatalogResponse{
		Categories: []dto.PairCategoryView{
			{
				Key:   "forex",
				Label: "Forex",
				Groups: []dto.PairGroupView{
					{
						Key:   "major",
						Label: "Major",
						Pairs: []dto.PairItem{{Symbol: "USDJPY", DisplayName: "USD/JPY"}},
					},
				},
			},
		},
		Symbols: []string{"USDJPY"},
	}, nil
}
func (fakePairs) ResolveSymbol(_ context.Context, symbol string) (string, error) {
	if symbol == "GOLD" {
		return "XAUUSD", nil
	}
	return symbol, nil
}

func TestExecuteTool_SubscribeAndUnsubscribe(t *testing.T) {
	acc := &fakeAccounts{
		subs: []dto.SubscriptionResponse{{ID: "sub-1", Symbol: "EURUSD", IsActive: true}},
	}
	deps := ToolDeps{Accounts: acc, Pairs: fakePairs{}}

	res := ExecuteTool(context.Background(), deps, "u1", "subscribe_pair", `{"symbol":"USDJPY"}`)
	if !res.OK || acc.subSymbol != "USDJPY" {
		t.Fatalf("subscribe gagal: %+v", res)
	}
	if res.Action == nil || res.Action.Kind != "subscription_added" || res.Action.Symbol != "USDJPY" {
		t.Fatalf("event action subscribe salah: %+v", res.Action)
	}

	// Idempoten jika sudah ada
	resAlready := ExecuteTool(context.Background(), deps, "u1", "subscribe_pair", `{"symbol":"EURUSD"}`)
	if !resAlready.OK || resAlready.Action != nil {
		t.Fatalf("subscribe pair aktif tidak boleh trigger action lagi: %+v", resAlready)
	}

	// Unsubscribe
	resUnsub := ExecuteTool(context.Background(), deps, "u1", "unsubscribe_pair", `{"symbol":"EURUSD"}`)
	if !resUnsub.OK || len(acc.unsubIDs) != 1 || acc.unsubIDs[0] != "sub-1" {
		t.Fatalf("unsubscribe gagal: %+v", resUnsub)
	}
	if resUnsub.Action == nil || resUnsub.Action.Kind != "subscription_removed" {
		t.Fatalf("event action unsubscribe salah: %+v", resUnsub.Action)
	}

	// Kuota penuh diteruskan rapi
	acc.subErr = apperror.ErrForbidden
	resFull := ExecuteTool(context.Background(), deps, "u1", "subscribe_pair", `{"symbol":"GBPUSD"}`)
	if resFull.OK {
		t.Fatalf("diharapkan gagal saat kuota penuh")
	}
}

func TestExecuteTool_CandlesUnavailableHonest(t *testing.T) {
	deps := ToolDeps{
		Candles: &fakeCandles{err: errors.New("feed timeout")},
		Pairs:   fakePairs{},
	}
	res := ExecuteTool(context.Background(), deps, "u1", "inspect_trade_candles", `{"symbol":"GOLD","timeframe":"M15"}`)
	if !res.OK {
		t.Fatalf("expected structured JSON result, got error: %+v", res)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(res.OutputJSON), &body); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if body["available"] != false {
		t.Fatalf("harus melaporkan available=false saat feed gagal: %v", body)
	}
}

type fakeJournal struct{}

func (fakeJournal) ListPositions(_ context.Context, _ string, _ repository.JournalFilter, _ int) ([]model.Position, error) {
	return []model.Position{{
		Base:           model.Base{ID: "p1"},
		SourceSignalID: "sig-1",
		Symbol:         "XAUUSD",
		Direction:      "SELL",
		Status:         model.PositionStatusClosedSL,
		OutcomeR:       -1,
		OpenedAt:       time.Date(2026, 11, 20, 2, 0, 0, 0, time.UTC),
	}}, nil
}
func (fakeJournal) ListSkipped(_ context.Context, _ string, _ repository.JournalFilter, _, _ int) ([]model.UserSignal, error) {
	return nil, nil
}
func (fakeJournal) Stats(_ context.Context, _ string, _ repository.JournalFilter) (repository.JournalStats, error) {
	return repository.JournalStats{TotalPositions: 1, ClosedSL: 1, Losses: 1, TotalR: -1}, nil
}

func TestExecuteTool_JournalIncludesSourceSignalID(t *testing.T) {
	deps := ToolDeps{Journal: fakeJournal{}, Pairs: fakePairs{}}
	res := ExecuteTool(context.Background(), deps, "u1", "get_positions_and_journal", `{"symbol":"XAUUSD"}`)
	if !res.OK {
		t.Fatalf("journal tool gagal: %+v", res)
	}
	var body struct {
		Positions []map[string]any `json:"positions"`
	}
	if err := json.Unmarshal([]byte(res.OutputJSON), &body); err != nil {
		t.Fatalf("json parse: %v", err)
	}
	if len(body.Positions) != 1 || body.Positions[0]["source_signal_id"] != "sig-1" {
		t.Fatalf("source_signal_id tidak terbawa: %+v", body.Positions)
	}
}
