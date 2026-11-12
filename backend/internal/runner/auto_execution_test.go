package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/runner"
)

// newAutoService menyusun Service dengan saklar eksekusi otomatis tertentu.
// Dipakai tes yang perlu membedakan "saklar nyala" dan "saklar mati".
func newAutoService(store *fakeStore, pub *fakePublisher, autoExecute bool) *runner.Service {
	registry := broker.NewRegistry()
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	return runner.NewService(store, registry, pub, discardLogger(), runner.Options{
		Policy:          runner.DefaultDailyPolicy(),
		SlippageGuard:   true,
		AutoExecute:     autoExecute,
		MarketOpenCheck: false,
		Settings:        ila3General(),
	})
}

// TestVerifikasiAkunTidakDitulisSaatBrokerGagal memastikan kegagalan tidak
// menaikkan waktu verifikasi.
//
// Perilaku "tertulis saat berhasil" diuji di tingkat repository
// (internal/repository/account_verify_test.go) karena di sini klien broker
// selalu gagal dibuka tanpa kredensial terenkripsi sungguhan.
func TestVerifikasiAkunTidakDitulisSaatBrokerGagal(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "manual", true)

	// Akun nonaktif tidak pernah dibaca RefreshTelemetry, jadi broker tidak
	// dihubungi dan verifikasi tidak boleh tercatat.
	store.mu.Lock()
	store.accounts["acct-u1"].IsActive = false
	store.mu.Unlock()

	svc := newService(store, pub)
	svc.RefreshTelemetry(context.Background())

	if at, ada := store.verification("acct-u1"); ada {
		t.Fatalf("akun nonaktif tidak boleh diverifikasi, tetapi tercatat pada %s", at.Format(time.RFC3339))
	}
}

// TestAutoEksekusiMengirimOrder memastikan alur lengkap pantauan → eksekusi:
// akun ber-mode auto dengan saklar nyala benar-benar membuka posisi.
func TestAutoEksekusiMengirimOrder(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newAutoService(store, pub, true)
	signal := readySignal("XAUUSD", time.Hour)

	svc.HandleSignal(context.Background(), signal)

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("membaca posisi: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("mode auto harus membuka 1 posisi, dapat %d", len(positions))
	}

	pos := positions[0]
	if pos.SourceSignalID != signal.GetSignalId() {
		t.Errorf("posisi harus tertaut ke sinyal, dapat %q", pos.SourceSignalID)
	}
	// SL/TP wajib sama persis dengan sinyal: angka inilah yang dipasang di broker.
	if pos.StopLoss != signal.GetStopLoss() {
		t.Errorf("stop loss posisi = %v, ingin %v (angka sinyal)", pos.StopLoss, signal.GetStopLoss())
	}
	if pos.TakeProfit != signal.GetTakeProfit() {
		t.Errorf("take profit posisi = %v, ingin %v (angka sinyal)", pos.TakeProfit, signal.GetTakeProfit())
	}
	if pos.LotSize <= 0 {
		t.Errorf("lot harus positif, dapat %v", pos.LotSize)
	}
	if pos.Direction != "BUY" {
		t.Errorf("arah posisi = %q, ingin BUY", pos.Direction)
	}
}

// TestAutoEksekusiMatiTidakOrder memastikan saklar global benar-benar menahan
// order: sinyal hanya dicatat siap agar user bisa mengonfirmasi sendiri.
func TestAutoEksekusiMatiTidakOrder(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newAutoService(store, pub, false)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatalf("saklar mati tidak boleh membuka posisi, dapat %d", len(positions))
	}

	last, ok := pub.last()
	if !ok {
		t.Fatal("sinyal tidak diteruskan ke user")
	}
	if last.Status != "READY_TO_ENTRY" {
		t.Errorf("status diteruskan = %q, ingin READY_TO_ENTRY", last.Status)
	}
}

// TestAutoEksekusiSekaliPerSinyal memastikan kiriman ulang sinyal yang sama
// tidak membuka posisi kedua. Ini pengaman terpenting pada uang nyata: engine
// mengirim status berulang, dan duplikat harus berhenti di klaim eksekusi.
func TestAutoEksekusiSekaliPerSinyal(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newAutoService(store, pub, true)
	signal := readySignal("XAUUSD", time.Hour)

	svc.HandleSignal(context.Background(), signal)
	svc.HandleSignal(context.Background(), signal)

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("sinyal yang sama harus menghasilkan tepat 1 posisi, dapat %d", len(positions))
	}
}
