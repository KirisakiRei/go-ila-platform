package runner_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/model"
	"ila/backend/internal/runner"
	"ila/backend/internal/settings"
)

// toggleFlags adalah pengganti sumber flag untuk pengujian.
type toggleFlags struct {
	values map[string]bool
}

func (f *toggleFlags) Bool(_ context.Context, key string, fallback bool) bool {
	if value, ok := f.values[key]; ok {
		return value
	}
	return fallback
}

func (f *toggleFlags) Int(_ context.Context, _ string, fallback int) int { return fallback }

// newSwitchService merangkai Runner dengan saklar metode per akun aktif.
func newSwitchService(store *fakeStore, pub *fakePublisher) *runner.Service {
	return newSwitchServiceDenganFlag(store, pub, true)
}

// newSwitchServiceDenganFlag merangkai Runner dengan nilai flag saklar tertentu.
func newSwitchServiceDenganFlag(store *fakeStore, pub *fakePublisher, aktif bool) *runner.Service {
	registry := broker.NewRegistry()
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	return runner.NewService(store, registry, pub, slog.New(slog.NewTextHandler(io.Discard, nil)), runner.Options{
		Policy:          runner.DefaultDailyPolicy(),
		SlippageGuard:   false,
		AutoExecute:     true,
		MarketOpenCheck: false,
		Settings:        ila3General(),
		Flags: &toggleFlags{values: map[string]bool{
			settings.KeyPairSwitch: aktif,
		}},
	})
}

// siapkanAkunPlusXAU menyiapkan satu user Plus dengan satu akun aktif dan
// langganan XAUUSD, lalu mengembalikan store-nya.
func siapkanAkunPlusXAU(store *fakeStore, accountID string) {
	store.users["u-switch"] = &model.User{
		Base: model.Base{ID: "u-switch"}, Tier: "plus", Status: "active",
	}
	store.accounts[accountID] = &model.BrokerAccount{
		Base: model.Base{ID: accountID}, UserID: "u-switch", BrokerType: "paper",
		Mode: "auto", IsActive: true, IsDryRun: true, RiskPercent: 0.5,
	}
	store.subs = append(store.subs, model.Subscription{
		UserID: "u-switch", Symbol: "XAUUSD", IsActive: true,
	})
}

// TestSaklarMetode_MematikanMetodeMenahanSinyal memastikan metode yang dimatikan
// pemilik akun tidak menghasilkan posisi, walau langganan masih aktif.
func TestSaklarMetode_MematikanMetodeMenahanSinyal(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	siapkanAkunPlusXAU(store, "acc-1")

	// Saklar akun: ILA 2.0 dimatikan untuk XAUUSD.
	store.switches["XAUUSD|acc-1"] = map[string]bool{"ila-2.0": false}

	svc := newSwitchService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u-switch")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("posisi terbuka = %d, ingin 0 karena metode dimatikan", len(positions))
	}
}

// TestSaklarMetode_MenyalaTetapEksekusi memastikan saklar yang menyala tidak
// mengganggu eksekusi normal.
func TestSaklarMetode_MenyalaTetapEksekusi(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	siapkanAkunPlusXAU(store, "acc-1")

	store.switches["XAUUSD|acc-1"] = map[string]bool{"ila-2.0": true}

	svc := newSwitchService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u-switch")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin 1", len(positions))
	}
}

// TestSaklarMetode_FlagMatiMemakaiPerilakuLama memastikan rem darurat tetap
// berfungsi: bila admin mematikan flag, saklar diabaikan dan Runner kembali ke
// perilaku lama (kolom langganan menentukan metode).
func TestSaklarMetode_FlagMatiMemakaiPerilakuLama(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	siapkanAkunPlusXAU(store, "acc-1")
	store.switches["XAUUSD|acc-1"] = map[string]bool{"ila-2.0": false}

	// Flag dimatikan eksplisit oleh admin.
	svc := newSwitchServiceDenganFlag(store, pub, false)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u-switch")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin 1 (flag mati, saklar diabaikan)", len(positions))
	}
}

// TestSaklarMetode_FlagBawaanMenyala memastikan tanpa konfigurasi flag apa pun,
// saklar per akun langsung berlaku.
func TestSaklarMetode_FlagBawaanMenyala(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	siapkanAkunPlusXAU(store, "acc-1")
	store.switches["XAUUSD|acc-1"] = map[string]bool{"ila-2.0": false}

	// Flags nil: Runner langsung memakai bawaan (menyala).
	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u-switch")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("posisi terbuka = %d, ingin 0 karena saklar bawaan sudah berlaku", len(positions))
	}
}

// TestSaklarMetode_SemuaMetodeMatiTidakEksekusi memastikan akun yang seluruh
// metodenya dimatikan tetap dihormati. Kasus ini rawan: peta saklar hanya
// memuat metode yang menyala, jadi tanpa daftar akun terkonfigurasi akun ini
// akan keliru dianggap "belum pernah diatur".
func TestSaklarMetode_SemuaMetodeMatiTidakEksekusi(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	siapkanAkunPlusXAU(store, "acc-1")

	store.switches["XAUUSD|acc-1"] = map[string]bool{"ila-2.0": false, "ila-3.0": false}

	svc := newSwitchService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u-switch")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("posisi terbuka = %d, ingin 0 karena semua metode dimatikan", len(positions))
	}
}

// TestComputeRequirements_MengikutiSaklarMetode memastikan engine hanya diminta
// menghitung metode yang menyala pada salah satu akun.
func TestComputeRequirements_MengikutiSaklarMetode(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	siapkanAkunPlusXAU(store, "acc-1")
	store.switches["XAUUSD|acc-1"] = map[string]bool{"ila-3.0": true}

	svc := newSwitchService(store, pub)
	reqs, err := svc.ComputeRequirements(context.Background(), []runner.StrategyLocation{
		{Strategy: "ila-2.0", Addr: "ila2:9090"},
		{Strategy: "ila-3.0", Addr: "ila3:9090"},
	})
	if err != nil {
		t.Fatalf("ComputeRequirements: %v", err)
	}

	if len(reqs["ila-2.0"]) != 0 {
		t.Fatalf("ila-2.0 = %v, ingin kosong karena saklarnya mati", reqs["ila-2.0"])
	}
	if got := reqs["ila-3.0"]; len(got) != 1 || got[0] != "XAUUSD" {
		t.Fatalf("ila-3.0 = %v, ingin [XAUUSD]", got)
	}
}
