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

// TestPasarTutupDievaluasiUlang mengunci perilaku yang disengaja: penolakan
// karena pasar tutup bukan akhir cerita.
//
// Status user signal menjadi REJECTED, tetapi sinyal globalnya tetap READY.
// Saat pasar dibuka dan engine mengirim ulang, evaluasi berjalan lagi dan user
// menerima notifikasi baru. Tes ini menjaga agar perilaku itu tidak berubah
// menjadi terminal tanpa sengaja.
func TestPasarTutupDievaluasiUlang(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "manual", true)

	svc := newServiceDenganCekPasar(store, pub)
	svc.SetMarketState(enginev1.MarketState_MARKET_CLOSED)

	if svc.MarketOpen() {
		t.Fatal("status pasar tertutup tidak terbaca")
	}

	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	terakhir, ada := pub.last()
	if !ada {
		t.Fatal("setup saat pasar tutup seharusnya tetap menghasilkan notifikasi")
	}
	if terakhir.Reason != string(runner.RejectMarketClosed) {
		t.Fatalf("alasan = %q, ingin penolakan pasar tutup", terakhir.Reason)
	}

	// Inilah yang membuat evaluasi ulang mungkin: sinyal globalnya tidak ditutup.
	if got := store.signals["sig-XAUUSD"].Status; got == model.SignalStatusExpired {
		t.Fatal("sinyal tidak boleh dianggap kedaluwarsa karena pasar tutup")
	}
}

// newServiceDenganCekPasar menyalakan pemeriksaan jam pasar yang pada tes lain
// sengaja dimatikan agar alur trading bisa diuji terpisah.
func newServiceDenganCekPasar(store *fakeStore, pub *fakePublisher) *runner.Service {
	registry := broker.NewRegistry()
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	return runner.NewService(store, registry, pub, discardLogger(), runner.Options{
		Policy:          runner.DefaultDailyPolicy(),
		MarketOpenCheck: true,
		Settings:        ila3General(),
	})
}
