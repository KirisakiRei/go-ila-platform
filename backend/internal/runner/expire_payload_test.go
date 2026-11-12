package runner_test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/model"
)

// Pengujian ini mengunci satu bug nyata di produksi: engine mengirim penarikan
// sinyal (EXPIRE / CANCEL / SUPERSEDE) lewat payload Signal yang HANYA memuat
// identitas dan Action — tanpa Status dan tanpa Direction.
//
// Sebelum diperbaiki, payload itu jatuh ke validasi "sinyal tidak lengkap" dan
// memunculkan WARN 5 menit setelah setiap sinyal terbit. WARN-nya menyesatkan:
// terbaca seolah ada sinyal siap eksekusi yang dibuang, padahal itu rutin
// kedaluwarsa.

func sinyalSiapEntry(id string) *enginev1.Signal {
	now := time.Now()
	return &enginev1.Signal{
		SignalId:        id,
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		Htf:             "H4",
		Ltf:             "M15",
		Status:          enginev1.SignalStatus_READY_TO_ENTRY,
		Action:          enginev1.SignalAction_SIGNAL_ACTION_PUBLISH,
		Direction:       enginev1.Direction_BUY,
		EntryPrice:      4100,
		StopLoss:        4090,
		TakeProfit:      4120,
		RiskReward:      2,
		PublishedAt:     timestamppb.New(now),
		ExpiresAt:       timestamppb.New(now.Add(5 * time.Minute)),
	}
}

// penarikan meniru bentuk payload expire dari engine: identitas + Action saja.
func penarikan(id string, action enginev1.SignalAction) *enginev1.Signal {
	return &enginev1.Signal{
		SignalId:    id,
		Symbol:      "XAUUSD",
		Htf:         "H4",
		Ltf:         "M15",
		Action:      action,
		PublishedAt: timestamppb.New(time.Now()),
	}
}

func TestPayloadExpireMenandaiSinyalKedaluwarsa(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	ctx := context.Background()

	const id = "sig-XAUUSD-H4-M15-1"
	svc.HandleSignal(ctx, sinyalSiapEntry(id))

	if got := store.signals[id].Status; got != model.SignalStatusReadyToEntry {
		t.Fatalf("status awal = %q, ingin READY_TO_ENTRY", got)
	}

	// Inilah yang dulu gagal: payload tanpa Status/Direction.
	svc.HandleSignal(ctx, penarikan(id, enginev1.SignalAction_SIGNAL_ACTION_EXPIRE))

	if got := store.signals[id].Status; got != model.SignalStatusExpired {
		t.Fatalf("status setelah expire = %q, ingin EXPIRED", got)
	}
}

func TestPayloadCancelMenandaiSinyalBatal(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	ctx := context.Background()

	const id = "sig-XAUUSD-H4-M15-2"
	svc.HandleSignal(ctx, sinyalSiapEntry(id))
	svc.HandleSignal(ctx, penarikan(id, enginev1.SignalAction_SIGNAL_ACTION_CANCEL))

	if got := store.signals[id].Status; got != model.SignalStatusCancelled {
		t.Fatalf("status setelah cancel = %q, ingin CANCELLED", got)
	}
}

func TestPayloadExpireTidakMembuatSinyalBaru(t *testing.T) {
	// Penarikan untuk sinyal yang tidak dikenal tidak boleh membuat baris
	// sampah dengan harga nol.
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})

	svc.HandleSignal(context.Background(), penarikan("sig-tidak-ada", enginev1.SignalAction_SIGNAL_ACTION_EXPIRE))

	if _, ada := store.signals["sig-tidak-ada"]; ada {
		t.Fatalf("penarikan sinyal tak dikenal tidak boleh membuat baris baru")
	}
}

func TestPayloadPublishTetapDivProsesBiasa(t *testing.T) {
	// Jalur PUBLISH tidak boleh ikut dialihkan ke HandleUpdate: sinyal baru
	// harus tetap lengkap diproses (harga, arah, status).
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})

	const id = "sig-XAUUSD-H4-M15-3"
	svc.HandleSignal(context.Background(), sinyalSiapEntry(id))

	row, ada := store.signals[id]
	if !ada {
		t.Fatalf("sinyal publish harus tersimpan")
	}
	if row.EntryPrice != 4100 || row.Direction != "BUY" {
		t.Fatalf("sinyal publish kehilangan data: harga=%v arah=%q", row.EntryPrice, row.Direction)
	}
}
