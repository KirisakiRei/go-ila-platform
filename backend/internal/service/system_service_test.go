package service

import (
	"context"
	"testing"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
)

type fakeEngineReader struct {
	rows []model.EngineState
}

func (f *fakeEngineReader) ListEngines(context.Context) ([]model.EngineState, error) {
	return f.rows, nil
}

type fakeDailyReader struct {
	summary *model.DailySummary
}

func (f *fakeDailyReader) FindDailySummary(context.Context, string, string) (*model.DailySummary, error) {
	if f.summary == nil {
		return &model.DailySummary{}, nil
	}
	return f.summary, nil
}

type fakeStateLister struct {
	rows []model.BrokerAccountState
}

func (f *fakeStateLister) ListByUser(context.Context, string) ([]model.BrokerAccountState, error) {
	return f.rows, nil
}

func TestSystemStatusMerangkumSemua(t *testing.T) {
	accounts := &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}}
	accounts.accounts["a1"] = &model.BrokerAccount{
		Base: model.Base{ID: "a1"}, UserID: userIDTest, BrokerType: "matchtrader", IsActive: true,
	}
	accounts.accounts["a2"] = &model.BrokerAccount{
		Base: model.Base{ID: "a2"}, UserID: userIDTest, BrokerType: "matchtrader", IsActive: true,
	}

	now := time.Now().UTC()
	svc := NewSystemService(SystemServiceDeps{
		Accounts: accounts,
		States: &fakeStateLister{rows: []model.BrokerAccountState{
			{BrokerAccountID: "a1", UserID: userIDTest, Connected: true},
			{BrokerAccountID: "a2", UserID: userIDTest, Connected: false, LastError: "token kedaluwarsa"},
		}},
		Engines: &fakeEngineReader{rows: []model.EngineState{
			{ID: "engine:ila-2.0", Strategy: "ila-2.0", Connected: true, MarketState: "OPEN", MarkedAt: &now},
		}},
		Daily: &fakeDailyReader{summary: &model.DailySummary{
			UserID: userIDTest, WinCount: 1, TotalTrades: 1, LockedAt: &now, LockReason: "sudah menang",
		}},
		Positions: &fakePositionLister{},
		MaxTrades: 2,
	})

	status, err := svc.Status(context.Background(), userIDTest)
	if err != nil {
		t.Fatalf("status: %v", err)
	}

	if status.Market.State != "OPEN" || status.Market.Label != "Pasar buka" {
		t.Errorf("status pasar = %+v", status.Market)
	}
	if len(status.Engines) != 1 || !status.Engines[0].Connected {
		t.Errorf("status engine = %+v", status.Engines)
	}
	if status.Brokers.Total != 2 || status.Brokers.Connected != 1 {
		t.Errorf("status broker = %+v", status.Brokers)
	}
	if !status.Brokers.AnyError {
		t.Error("kesalahan akun harus terlihat")
	}
	if !status.Daily.Locked || status.Daily.WinCount != 1 {
		t.Errorf("ringkasan harian = %+v", status.Daily)
	}
}

// Hari dihitung dalam WIB, bukan UTC (plans/05 par 3.1).
func TestSystemStatusHariWIB(t *testing.T) {
	// 18:30 UTC = 01:30 WIB keesokan harinya.
	utc := time.Date(2026, 11, 6, 18, 30, 0, 0, time.UTC)

	svc := NewSystemService(SystemServiceDeps{
		Accounts:  &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}},
		States:    &fakeStateLister{},
		Engines:   &fakeEngineReader{},
		Daily:     &fakeDailyReader{},
		Positions: &fakePositionLister{},
		Clock:     func() time.Time { return utc },
	})

	status, err := svc.Status(context.Background(), userIDTest)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Daily.Day != "2026-11-07" {
		t.Fatalf("hari = %q, ingin 2026-11-07 (WIB)", status.Daily.Day)
	}
}

func TestSystemStatusTanpaAkun(t *testing.T) {
	svc := NewSystemService(SystemServiceDeps{
		Accounts:  &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}},
		States:    &fakeStateLister{},
		Engines:   &fakeEngineReader{},
		Daily:     &fakeDailyReader{},
		Positions: &fakePositionLister{},
	})

	status, err := svc.Status(context.Background(), userIDTest)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Brokers.Total != 0 || status.Market.State != "UNKNOWN" {
		t.Fatalf("status kosong tidak sesuai: %+v", status)
	}
}
