package runner_test

import (
	"context"
	"testing"
	"time"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/model"
)

// TestKemajuanRadarTidakMengembalikanSinyalYangSudahDieksekusi memastikan
// kiriman status menengah untuk slot yang sama tidak menimpa status user yang
// sudah lebih maju. Tanpa pagar ini radar kembali meminta entry padahal
// posisinya masih terbuka.
func TestKemajuanRadarTidakMengembalikanSinyalYangSudahDieksekusi(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	svc := newService(store, pub)

	svc.HandleSignal(context.Background(), readySignal("XAUUSD", 24*time.Hour))
	posisiSebelum, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(posisiSebelum) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin 1", len(posisiSebelum))
	}

	// Kemajuan radar berikutnya untuk slot simbol yang sama.
	svc.HandleSignal(context.Background(), &enginev1.Signal{
		SignalId:        "sig-XAUUSD",
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		Status:          enginev1.SignalStatus_WAIT_LTF_CISD,
	})

	us, err := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if err != nil {
		t.Fatalf("FindUserSignal: %v", err)
	}
	if us.Status != model.SignalStatusConsumed {
		t.Fatalf("status user = %q, ingin tetap CONSUMED", us.Status)
	}
	if us.PositionID != posisiSebelum[0].ID {
		t.Fatalf("position_id = %q, ingin tetap %q", us.PositionID, posisiSebelum[0].ID)
	}
}

// TestPenautanPosisiTidakHilangSaatStatusDiperbarui memastikan pembaruan status
// tanpa membawa posisi tidak mengosongkan tautan ke posisi yang sudah terbuka.
func TestPenautanPosisiTidakHilangSaatStatusDiperbarui(t *testing.T) {
	store := newFakeStore()
	if err := store.UpsertUserSignal(context.Background(), &model.UserSignal{
		UserID:   "u1",
		SignalID: "sig-XAUUSD",
		Status:   model.SignalStatusConsumed,
	}); err != nil {
		t.Fatalf("UpsertUserSignal: %v", err)
	}

	if err := store.SetUserSignalPosition(context.Background(), "u1", "sig-XAUUSD", "pos-1"); err != nil {
		t.Fatalf("SetUserSignalPosition: %v", err)
	}
	if err := store.UpsertUserSignal(context.Background(), &model.UserSignal{
		UserID:   "u1",
		SignalID: "sig-XAUUSD",
		Status:   model.SignalStatusSkipped,
	}); err != nil {
		t.Fatalf("UpsertUserSignal: %v", err)
	}

	us, err := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if err != nil {
		t.Fatalf("FindUserSignal: %v", err)
	}
	if us.PositionID != "pos-1" {
		t.Fatalf("position_id = %q, ingin tetap pos-1", us.PositionID)
	}
}
