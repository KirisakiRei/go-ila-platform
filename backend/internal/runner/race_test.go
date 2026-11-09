package runner_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/runner"
)

// readySignalID membuat sinyal siap eksekusi dengan ID tertentu. Dipakai untuk
// mensimulasikan dua sinyal BERBEDA yang tiba hampir bersamaan.
func readySignalID(id, symbol string, ttl time.Duration) *enginev1.Signal {
	published := time.Now().UTC()
	return &enginev1.Signal{
		SignalId:        id,
		Symbol:          symbol,
		Status:          enginev1.SignalStatus_READY_TO_ENTRY,
		Action:          enginev1.SignalAction_SIGNAL_ACTION_PUBLISH,
		Direction:       enginev1.Direction_BUY,
		EntryPrice:      4120.50,
		StopLoss:        4112.30,
		TakeProfit:      4149.20,
		RiskReward:      2.30,
		PublishedAt:     timestamppb.New(published),
		ExpiresAt:       timestamppb.New(published.Add(ttl)),
		StrategyVersion: "ila-2.0",
	}
}

// TestEksekusiSerentak_DuaSinyalBerbedaPadaAkunSama menguji apakah satu akun
// bisa membuka DUA posisi ketika dua sinyal berbeda diproses bersamaan.
//
// Pagar EvaluateDaily memeriksa posisi yang SUDAH terbuka. Bila kedua sinyal
// melewati pemeriksaan itu sebelum posisi pertama tercatat, keduanya akan
// mengeksekusi dan akun menanggung risiko dobel — persis skenario dua pasangan
// timeframe pada pair yang sama.
func TestEksekusiSerentak_DuaSinyalBerbedaPadaAkunSama(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	addUser(store, "u1", "plus", "auto", true)

	svc := newService(store, pub)

	// Dua sinyal berbeda untuk simbol yang sama, diproses serentak.
	sigA := readySignalID("sig-A", "XAUUSD", time.Hour)
	sigB := readySignalID("sig-B", "XAUUSD", time.Hour)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		svc.HandleSignal(context.Background(), sigA)
	}()
	go func() {
		defer wg.Done()
		svc.HandleSignal(context.Background(), sigB)
	}()
	wg.Wait()

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}

	t.Logf("posisi terbuka setelah dua sinyal serentak: %d", len(positions))
	for _, p := range positions {
		t.Logf("  akun=%s simbol=%s sinyal=%s", p.BrokerAccountID, p.Symbol, p.SourceSignalID)
	}

	if len(positions) > 1 {
		t.Fatalf("akun membuka %d posisi dari dua sinyal berbeda; "+
			"seharusnya maksimal 1 karena satu akun hanya boleh punya satu posisi terbuka",
			len(positions))
	}
}

// TestApproveSerentak_TidakMelahirkanDuaOrder memastikan jalur konfirmasi
// manual juga terlindungi. Tanpa kunci akun, dua klik "Eksekusi" yang tiba
// bersamaan bisa melahirkan dua order pada akun yang sama.
func TestApproveSerentak_TidakMelahirkanDuaOrder(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	addUser(store, "u1", "plus", "manual", true)
	svc := newService(store, pub)

	// Sinyal harus tersimpan lebih dulu agar Approve bisa membacanya.
	sig := readySignalID("sig-manual", "XAUUSD", time.Hour)
	svc.HandleSignal(context.Background(), sig)

	var wg sync.WaitGroup
	alasan := make([]runner.RejectReason, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			alasan[idx], _ = svc.Approve(context.Background(), "u1", "sig-manual", "acct-u1")
		}(i)
	}
	wg.Wait()

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	t.Logf("posisi=%d alasan=[%q, %q]", len(positions), alasan[0], alasan[1])
	if len(positions) > 1 {
		t.Fatalf("dua konfirmasi serentak melahirkan %d order; seharusnya maksimal 1", len(positions))
	}
}

// TestEksekusiBerurutan_SinyalKeduaDitolak memastikan pagar bekerja pada
// pemrosesan berurutan: sinyal kedua harus ditolak selama posisi pertama masih
// terbuka. Ini pembanding untuk tes serentak di atas.
func TestEksekusiBerurutan_SinyalKeduaDitolak(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	addUser(store, "u1", "plus", "auto", true)

	svc := newService(store, pub)

	svc.HandleSignal(context.Background(), readySignalID("sig-A", "XAUUSD", time.Hour))
	svc.HandleSignal(context.Background(), readySignalID("sig-B", "XAUUSD", time.Hour))

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("ListOpenPositionsByUser: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin tepat 1 (sinyal kedua harus ditolak)", len(positions))
	}
}
