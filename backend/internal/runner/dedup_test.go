package runner_test

import (
	"context"
	"testing"
	"time"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/broker"
	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// Dua entri akun yang menunjuk akun broker sama hanya boleh menghasilkan
// satu order. Tanpa penyaringan ini, satu sinyal dieksekusi dua kali.
func TestAkunKembarHanyaSekali(t *testing.T) {
	accounts := []model.BrokerAccount{
		{Base: model.Base{ID: "a1"}, BrokerType: "matchtrader", ExternalAccountID: "1089", IsActive: true},
		{Base: model.Base{ID: "a2"}, BrokerType: "matchtrader", ExternalAccountID: "1089", IsActive: true},
		{Base: model.Base{ID: "a3"}, BrokerType: "matchtrader", ExternalAccountID: "2099", IsActive: true},
	}

	seen := map[string]struct{}{}
	var kept []string
	for _, account := range accounts {
		key := runner.AccountIdentity(account)
		if key != "" {
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
		}
		kept = append(kept, account.ID)
	}

	if len(kept) != 2 || kept[0] != "a1" || kept[1] != "a3" {
		t.Fatalf("akun yang dipakai: %v; ingin [a1 a3]", kept)
	}
}

// Akun tanpa penanda broker tidak digabung: setiap entri dianggap berbeda.
func TestAkunTanpaIdentitasTidakDigabung(t *testing.T) {
	if runner.AccountIdentity(model.BrokerAccount{Base: model.Base{ID: "x"}}) != "" {
		t.Fatal("akun tanpa identitas broker harus mengembalikan string kosong")
	}
}

// Akun yang sudah dihapus tidak boleh dieksekusi walau sisa subscription dan
// akun broker masih tertinggal di basis data.
func TestUserTerhapusTidakDieksekusi(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	store.users["u1"].Status = model.UserStatusDeleted

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatal("akun terhapus tidak boleh membuka posisi")
	}
}

// Engine yang mengaku pasar buka pada Sabtu harus dilaporkan, bukan diabaikan.
// Ini penanda paling kuat bahwa engine memakai zona waktu yang salah.
func TestAnomaliJadwalPasarlDilaporkan(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	var reported []runner.MarketScheduleAnomaly
	registry := broker.NewRegistry()
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	// Sabtu 12:00 waktu New York.
	saturday := time.Date(2026, 3, 14, 12, 0, 0, 0, runner.NY)
	svc := runner.NewService(store, registry, pub, discardLogger(), runner.Options{
		Policy:          runner.DefaultDailyPolicy(),
		AutoExecute:     true,
		MarketOpenCheck: false,
		Settings:        ila3General(),
		Clock:           func() time.Time { return saturday },
		OnMarketAnomaly: func(anomalies []runner.MarketScheduleAnomaly) {
			reported = append(reported, anomalies...)
		},
	})

	svc.SetMarketState(enginev1.MarketState_MARKET_OPEN)

	if len(reported) != 1 || reported[0] != runner.AnomalyWeekendOpen {
		t.Fatalf("anomali akhir pekan harus dilaporkan, dapat %v", reported)
	}
}
