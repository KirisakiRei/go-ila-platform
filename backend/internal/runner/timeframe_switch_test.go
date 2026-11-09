package runner_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/broker"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/settings"
)

// newTimeframeService merangkai Runner dengan saklar timeframe aktif.
func newTimeframeService(store *fakeStore, pub *fakePublisher) *runner.Service {
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
			settings.KeyTimeframeSwitch: true,
		}},
	})
}

// signalDenganTimeframe membuat sinyal siap eksekusi dengan pasangan timeframe.
func signalDenganTimeframe(id, symbol, htf, ltf string) *enginev1.Signal {
	sig := readySignalID(id, symbol, time.Hour)
	sig.Htf = htf
	sig.Ltf = ltf
	return sig
}

// TestTimeframeAutoMenyalaTetapEksekusi: pasangan dipantau dan diizinkan auto.
func TestTimeframeAutoMenyalaTetapEksekusi(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	store.timeframes["acct-u1|XAUUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}

	svc := newTimeframeService(store, pub)
	svc.HandleSignal(context.Background(), signalDenganTimeframe("sig-1", "XAUUSD", "H4", "M15"))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi = %d, ingin 1 karena auto menyala", len(positions))
	}
}

// TestTimeframeAutoMatiMenjadiPeluang: pasangan dipantau tetapi TIDAK diberi
// izin auto. Sinyal harus tetap tercatat sebagai peluang, bukan ditolak, dan
// tidak boleh mengeksekusi order.
func TestTimeframeAutoMatiMenjadiPeluang(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	store.timeframes["acct-u1|XAUUSD|ila-2.0|W1|H4"] = repository.TimeframeState{Watch: true, Auto: false}

	svc := newTimeframeService(store, pub)
	svc.HandleSignal(context.Background(), signalDenganTimeframe("sig-w", "XAUUSD", "W1", "H4"))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatalf("posisi = %d, ingin 0 karena auto tidak diizinkan", len(positions))
	}

	us, err := store.FindUserSignal(context.Background(), "u1", "sig-w")
	if err != nil || us == nil {
		t.Fatalf("sinyal peluang harus tercatat: %v", err)
	}
	if us.Status != model.SignalStatusOpportunity {
		t.Fatalf("status = %q, ingin %q (bukan REJECTED)", us.Status, model.SignalStatusOpportunity)
	}
	if us.RejectReason == "" {
		t.Fatal("pesan peluang harus diisi agar user tahu alasannya")
	}
}

// TestTimeframeTidakDipantauTidakMemproses: pasangan yang tidak dipantau tidak
// relevan bagi akun, sehingga tidak ada posisi dan tidak ada catatan peluang.
func TestTimeframeTidakDipantauTidakMemproses(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	// Akun hanya memantau W1-H4; sinyal datang dari H4-M15.
	store.timeframes["acct-u1|XAUUSD|ila-2.0|W1|H4"] = repository.TimeframeState{Watch: true, Auto: true}

	svc := newTimeframeService(store, pub)
	svc.HandleSignal(context.Background(), signalDenganTimeframe("sig-m", "XAUUSD", "H4", "M15"))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatalf("posisi = %d, ingin 0 karena pasangan tidak dipantau", len(positions))
	}
	if _, err := store.FindUserSignal(context.Background(), "u1", "sig-m"); err == nil {
		t.Fatal("pasangan yang tidak dipantau tidak boleh menghasilkan catatan user_signal")
	}
}

// TestTimeframeFlagMatiMemakaiPerilakuLama: selama flag mati, saklar diabaikan
// sehingga pemasangan lama tidak berubah perilakunya.
func TestTimeframeFlagMatiMemakaiPerilakuLama(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	store.timeframes["acct-u1|XAUUSD|ila-2.0|W1|H4"] = repository.TimeframeState{Watch: true, Auto: false}

	svc := newService(store, pub) // tanpa flag timeframe
	svc.HandleSignal(context.Background(), signalDenganTimeframe("sig-x", "XAUUSD", "H4", "M15"))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi = %d, ingin 1 karena flag mati berarti perilaku lama", len(positions))
	}
}

// TestTimeframeTanpaDataEngineTetapDiproses: sinyal tanpa htf/ltf (engine lama)
// tidak boleh disaring, karena tidak ada dasar untuk menyaring.
func TestTimeframeTanpaDataEngineTetapDiproses(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	store.timeframes["acct-u1|XAUUSD|ila-2.0|W1|H4"] = repository.TimeframeState{Watch: true, Auto: false}

	svc := newTimeframeService(store, pub)
	svc.HandleSignal(context.Background(), readySignalID("sig-nt", "XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi = %d, ingin 1 karena sinyal tanpa timeframe tidak disaring", len(positions))
	}
}
