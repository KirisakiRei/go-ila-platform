package runner_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sort"
	"testing"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// fakePairSource menyimulasikan sumber daftar simbol aktif.
type fakePairSource struct {
	symbols []string
	err     error
	calls   int
}

func (s *fakePairSource) EnabledPairSymbols(_ context.Context) ([]string, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.symbols, nil
}

func TestCachedPairGate_CacheAndExpiration(t *testing.T) {
	src := &fakePairSource{symbols: []string{"XAUUSD", "USTEC"}}
	gate := runner.NewCachedPairGate(src, 10*time.Minute)

	ctx := context.Background()

	// Panggilan pertama memukul sumber.
	symbols, err := gate.EnabledSymbols(ctx)
	if err != nil {
		t.Fatalf("EnabledSymbols: %v", err)
	}
	if !symbols["XAUUSD"] || !symbols["USTEC"] || symbols["EURUSD"] {
		t.Fatalf("himpunan simbol salah: %v", symbols)
	}
	if src.calls != 1 {
		t.Fatalf("calls = %d, ingin 1", src.calls)
	}

	// Panggilan kedua dalam masa TTL mengambil dari cache.
	symbols2, err := gate.EnabledSymbols(ctx)
	if err != nil {
		t.Fatalf("EnabledSymbols kedua: %v", err)
	}
	if len(symbols2) != 2 || src.calls != 1 {
		t.Fatalf("seharusnya cache hit: calls = %d", src.calls)
	}
}

func TestCachedPairGate_StaleOnError(t *testing.T) {
	src := &fakePairSource{symbols: []string{"XAUUSD"}}
	gate := runner.NewCachedPairGate(src, 1*time.Millisecond)

	ctx := context.Background()
	_, err := gate.EnabledSymbols(ctx)
	if err != nil {
		t.Fatalf("isi cache awal: %v", err)
	}

	// Biarkan TTL habis, lalu buat sumber error.
	time.Sleep(5 * time.Millisecond)
	src.err = errors.New("db disconnect")

	// Gate harus menyajikan data stale ketimbang menggagalkan proses trading.
	symbols, err := gate.EnabledSymbols(ctx)
	if err != nil {
		t.Fatalf("harus menyajikan stale: %v", err)
	}
	if !symbols["XAUUSD"] {
		t.Fatalf("stale data hilang: %v", symbols)
	}
}

// TestComputeRequirements_FilteredByPairGate memastikan simbol yang dinonaktifkan
// di katalog tidak pernah diminta ke engine, meskipun ada pelanggan aktif.
func TestComputeRequirements_FilteredByPairGate(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	// User Plus memantau XAUUSD dan EURUSD.
	store.users["u1"] = &model.User{Base: model.Base{ID: "u1"}, Tier: "plus", Status: "active"}
	store.subs = append(store.subs,
		model.Subscription{UserID: "u1", Symbol: "XAUUSD", Strategy: "ila-2.0", IsActive: true},
		model.Subscription{UserID: "u1", Symbol: "EURUSD", Strategy: "ila-2.0", IsActive: true},
	)

	// Gate hanya mengaktifkan XAUUSD; EURUSD dinonaktifkan admin.
	src := &fakePairSource{symbols: []string{"XAUUSD"}}
	gate := runner.NewCachedPairGate(src, time.Minute)

	registry := broker.NewRegistry()
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	svc := runner.NewService(store, registry, pub, slog.New(slog.NewTextHandler(io.Discard, nil)), runner.Options{
		Policy:          runner.DefaultDailyPolicy(),
		SlippageGuard:   false,
		AutoExecute:     true,
		MarketOpenCheck: false,
		Settings:        runner.NewSettings(runner.ILA3General),
		PairGate:        gate,
	})

	reqs, err := svc.ComputeRequirements(context.Background(), []runner.StrategyLocation{
		{Strategy: "ila-2.0", Addr: "ila2:9090"},
	})
	if err != nil {
		t.Fatalf("ComputeRequirements: %v", err)
	}

	// EURUSD harus tersaring keluar.
	got := reqs["ila-2.0"]
	if len(got) != 1 || got[0] != "XAUUSD" {
		t.Fatalf("ila-2.0 = %v, ingin [XAUUSD] saja", got)
	}
}

// TestSignalFannedOutPerAccount_MultipleAccountsForSameUser memastikan satu
// user dengan dua akun aktif (tier Plus) sama-sama mengeksekusi order saat
// sinyal datang, bukan hanya akun pertama.
func TestSignalFannedOutPerAccount_MultipleAccountsForSameUser(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	store.users["u-plus"] = &model.User{
		Base: model.Base{ID: "u-plus"}, Name: "User Plus", Tier: "plus", Status: "active",
	}

	// Dua akun aktif, keduanya mode auto.
	store.accounts["acc-1"] = &model.BrokerAccount{
		Base: model.Base{ID: "acc-1"}, UserID: "u-plus", BrokerType: "paper",
		Mode: "auto", IsActive: true, IsDryRun: true, RiskPercent: 0.5,
	}
	store.accounts["acc-2"] = &model.BrokerAccount{
		Base: model.Base{ID: "acc-2"}, UserID: "u-plus", BrokerType: "paper",
		Mode: "auto", IsActive: true, IsDryRun: true, RiskPercent: 0.5,
	}

	// Langganan tanpa broker_account_id (berlaku untuk seluruh akun aktif user).
	store.subs = append(store.subs, model.Subscription{
		UserID: "u-plus", Symbol: "XAUUSD", Strategy: "ila-2.0", IsActive: true,
	})

	svc := newService(store, pub)
	sig := readySignal("XAUUSD", time.Hour)
	svc.HandleSignal(context.Background(), sig)

	// Pastikan terbuka DUA posisi: satu di acc-1, satu di acc-2.
	positions, err := store.ListOpenPositionsByUser(context.Background(), "u-plus")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	if len(positions) != 2 {
		t.Fatalf("posisi terbuka = %d, ingin tepat 2 (satu per akun)", len(positions))
	}

	accIDs := []string{positions[0].BrokerAccountID, positions[1].BrokerAccountID}
	sort.Strings(accIDs)
	if accIDs[0] != "acc-1" || accIDs[1] != "acc-2" {
		t.Fatalf("posisi akun = %v, ingin [acc-1, acc-2]", accIDs)
	}
}

// TestSignalFannedOutPerAccount_SelectivePairPerAccount memastikan pilihan pair
// per akun dihormati: jika acc-1 memilih EURUSD dan acc-2 memilih XAUUSD, sinyal
// XAUUSD hanya membuka posisi di acc-2.
func TestSignalFannedOutPerAccount_SelectivePairPerAccount(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	store.users["u-plus"] = &model.User{
		Base: model.Base{ID: "u-plus"}, Name: "User Plus", Tier: "plus", Status: "active",
	}

	store.accounts["acc-1"] = &model.BrokerAccount{
		Base: model.Base{ID: "acc-1"}, UserID: "u-plus", BrokerType: "paper",
		Mode: "auto", IsActive: true, IsDryRun: true, RiskPercent: 0.5,
	}
	store.accounts["acc-2"] = &model.BrokerAccount{
		Base: model.Base{ID: "acc-2"}, UserID: "u-plus", BrokerType: "paper",
		Mode: "auto", IsActive: true, IsDryRun: true, RiskPercent: 0.5,
	}

	acc1 := "acc-1"
	acc2 := "acc-2"
	store.subs = append(store.subs,
		model.Subscription{UserID: "u-plus", Symbol: "EURUSD", Strategy: "ila-2.0", IsActive: true, BrokerAccountID: &acc1},
		model.Subscription{UserID: "u-plus", Symbol: "XAUUSD", Strategy: "ila-2.0", IsActive: true, BrokerAccountID: &acc2},
	)

	svc := newService(store, pub)
	sig := readySignal("XAUUSD", time.Hour)
	svc.HandleSignal(context.Background(), sig)

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u-plus")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin tepat 1", len(positions))
	}
	if positions[0].BrokerAccountID != "acc-2" {
		t.Fatalf("posisi dibuka di %s, ingin acc-2", positions[0].BrokerAccountID)
	}
}
