package service

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

type fakeSignalLister struct {
	signals []model.Signal
}

func (f *fakeSignalLister) ListForUser(_ context.Context, _ string, _ time.Time, _ int) ([]model.Signal, error) {
	return f.signals, nil
}

type fakePositionLister struct {
	positions []model.Position
}

func (f *fakePositionLister) ListByUser(_ context.Context, _ string, _ int) ([]model.Position, error) {
	return f.positions, nil
}
func (f *fakePositionLister) FindOpenByUser(_ context.Context, _ string) ([]model.Position, error) {
	var out []model.Position
	for _, p := range f.positions {
		if p.Status == model.PositionStatusOpen {
			out = append(out, p)
		}
	}
	return out, nil
}

type fakeUserSignalReader struct {
	status map[string]*model.UserSignal
}

func (f *fakeUserSignalReader) FindUserSignal(_ context.Context, _, signalID string) (*model.UserSignal, error) {
	if us, ok := f.status[signalID]; ok {
		return us, nil
	}
	return nil, apperror.ErrNotFound
}

func newSignal(signalID string, publishedAt, expiresAt time.Time) model.Signal {
	return model.Signal{
		SignalID:        signalID,
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		Direction:       "BUY",
		EntryPrice:      4100,
		StopLoss:        4090,
		TakeProfit:      4130,
		Status:          model.SignalStatusReadyToEntry,
		PublishedAt:     publishedAt,
		ExpiresAt:       expiresAt,
	}
}

func newTradingServiceFor(t *testing.T, user *model.User, signals []model.Signal, now time.Time) TradingService {
	t.Helper()

	users := newFakeUserRepo()
	users.users[user.Email] = user

	svc := NewTradingService(
		users,
		&fakeSignalLister{signals: signals},
		&fakePositionLister{},
		&fakeUserSignalReader{status: map[string]*model.UserSignal{}},
	)
	svc.(*tradingService).clock = func() time.Time { return now }
	return svc
}

// TestFreeDelayMenahanSinyalBaru memastikan sinyal belum terlihat sebelum 15 menit.
func TestFreeDelayMenahanSinyalBaru(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	signals := []model.Signal{newSignal("sig-1", now.Add(-5*time.Minute), now.Add(30*time.Minute))}

	svc := newTradingServiceFor(t, newTestUser("free"), signals, now)

	list, err := svc.Signals(context.Background(), userIDTest, 0)
	if err != nil {
		t.Fatalf("signals: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("sinyal Free seharusnya belum tampil, dapat %d", len(list))
	}
}

// TestFreeRadarStatusTidakTertunda memastikan status pemantauan radar (WAIT_*)
// langsung tampil untuk pengguna Free tanpa tertahan jeda 15 menit.
func TestFreeRadarStatusTidakTertunda(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	radarSig := newSignal("sig-radar", now.Add(-time.Minute), now.Add(4*time.Minute))
	radarSig.Status = model.SignalStatusWaitHTFSweep

	svc := newTradingServiceFor(t, newTestUser("free"), []model.Signal{radarSig}, now)

	list, err := svc.Signals(context.Background(), userIDTest, 0)
	if err != nil {
		t.Fatalf("signals: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("status radar seharusnya langsung tampil untuk Free, dapat %d", len(list))
	}
	if list[0].Delayed {
		t.Error("status radar tidak boleh bertanda delayed")
	}
}

// TestFreeMenampilkanSetelahDelay memastikan sinyal muncul setelah jeda lewat.
func TestFreeMenampilkanSetelahDelay(t *testing.T) {
	published := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	now := published.Add(16 * time.Minute)
	signals := []model.Signal{newSignal("sig-1", published, published.Add(60*time.Minute))}

	svc := newTradingServiceFor(t, newTestUser("free"), signals, now)

	list, err := svc.Signals(context.Background(), userIDTest, 0)
	if err != nil {
		t.Fatalf("signals: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("sinyal seharusnya tampil, dapat %d", len(list))
	}
	if !list[0].Delayed {
		t.Error("sinyal Free seharusnya bertanda delayed")
	}
}

// TestFreeSinyalKedaluwarsaSebelumDelayTidakTampil menjaga aturan plans/14 §3.1.
func TestFreeSinyalKedaluwarsaSebelumDelayTidakTampil(t *testing.T) {
	published := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	now := published.Add(20 * time.Minute)
	// TTL hanya 10 menit: habis sebelum jeda 15 menit selesai.
	signals := []model.Signal{newSignal("sig-1", published, published.Add(10*time.Minute))}

	svc := newTradingServiceFor(t, newTestUser("free"), signals, now)

	list, err := svc.Signals(context.Background(), userIDTest, 0)
	if err != nil {
		t.Fatalf("signals: %v", err)
	}
	if len(list) != 0 {
		t.Fatal("sinyal yang TTL habis sebelum delay tidak boleh tampil")
	}
}

// TestBasicTanpaDelay memastikan tier berbayar melihat sinyal seketika.
func TestBasicTanpaDelay(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	signals := []model.Signal{newSignal("sig-1", now.Add(-10*time.Second), now.Add(30*time.Minute))}

	svc := newTradingServiceFor(t, newTestUser("basic"), signals, now)

	list, err := svc.Signals(context.Background(), userIDTest, 0)
	if err != nil {
		t.Fatalf("signals: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("sinyal Basic seharusnya langsung tampil, dapat %d", len(list))
	}
	if list[0].Delayed {
		t.Error("sinyal Basic tidak boleh bertanda delayed")
	}
}

// TestCanConfirmHanyaSaatSiap memastikan tombol konfirmasi mengikuti status.
func TestCanConfirmHanyaSaatSiap(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	signals := []model.Signal{newSignal("sig-1", now.Add(-time.Minute), now.Add(30*time.Minute))}

	users := newFakeUserRepo()
	users.users["u1@contoh.id"] = newTestUser("plus")

	svc := NewTradingService(
		users,
		&fakeSignalLister{signals: signals},
		&fakePositionLister{},
		&fakeUserSignalReader{status: map[string]*model.UserSignal{
			"sig-1": {SignalID: "sig-1", UserID: userIDTest, Status: model.SignalStatusSkipped},
		}},
	)
	svc.(*tradingService).clock = func() time.Time { return now }

	list, err := svc.Signals(context.Background(), userIDTest, 0)
	if err != nil {
		t.Fatalf("signals: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("jumlah sinyal=%d, want 1", len(list))
	}
	if list[0].CanConfirm {
		t.Error("sinyal yang sudah dilewati tidak boleh bisa dikonfirmasi")
	}
}

func TestPositionsMapping(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)

	users := newFakeUserRepo()
	users.users["u1@contoh.id"] = newTestUser("basic")

	positions := &fakePositionLister{positions: []model.Position{
		{
			Base:   model.Base{ID: "pos-1"},
			Symbol: "XAUUSD", Direction: "BUY",
			TicketNumber: "W1", LotSize: 0.06, OpenPrice: 4100,
			Status: model.PositionStatusOpen, OpenedAt: now,
		},
	}}

	svc := NewTradingService(users, &fakeSignalLister{}, positions, &fakeUserSignalReader{})

	list, err := svc.Positions(context.Background(), userIDTest, 0)
	if err != nil {
		t.Fatalf("positions: %v", err)
	}
	if len(list) != 1 || list[0].TicketNumber != "W1" {
		t.Fatalf("posisi tidak sesuai: %+v", list)
	}
}
