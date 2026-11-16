package runner_test

import (
	"context"
	"testing"
	"time"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/model"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// siapkanDuplikat membuat satu sinyal READY berisi level harga lengkap dengan
// signal_id tertentu, meniru penerbitan ulang engine untuk setup yang sama.
func siapkanDuplikat(id string, status enginev1.SignalStatus) *enginev1.Signal {
	published := time.Now().UTC()
	return &enginev1.Signal{
		SignalId:        id,
		Symbol:          "XAUUSD",
		Status:          status,
		Action:          enginev1.SignalAction_SIGNAL_ACTION_PUBLISH,
		Direction:       enginev1.Direction_BUY,
		EntryPrice:      4120.50,
		StopLoss:        4112.30,
		TakeProfit:      4149.20,
		RiskReward:      2.30,
		PublishedAt:     timestamppb.New(published),
		ExpiresAt:       timestamppb.New(published.Add(4 * time.Hour)),
		StrategyVersion: "ila-2.0",
		Htf:             "H4",
		Ltf:             "M15",
	}
}

// TestPenerbitanUlangSetupDitahan memastikan satu setup yang diterbitkan
// berulang dengan signal_id baru hanya tercatat sekali. Sebelumnya setiap
// terbitan melahirkan baris user_signal dan notifikasi sendiri.
func TestPenerbitanUlangSetupDitahan(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "manual", true)
	svc := newService(store, pub)

	pertama := siapkanDuplikat("sig-asal", enginev1.SignalStatus_READY_TO_ENTRY)
	svc.HandleSignal(context.Background(), pertama)

	notifAwal := len(pub.signals)
	for _, id := range []string{"sig-ulang-1", "sig-ulang-2", "sig-ulang-3"} {
		svc.HandleSignal(context.Background(), siapkanDuplikat(id, enginev1.SignalStatus_READY_TO_ENTRY))
	}

	if got := len(store.userSignals); got != 1 {
		t.Fatalf("baris user_signal = %d, ingin 1 (penerbitan ulang ditahan)", got)
	}
	if got := len(pub.signals); got != notifAwal {
		t.Fatalf("notifikasi = %d, ingin tetap %d", got, notifAwal)
	}
}

// TestSetupBerbedaTidakDitahan memastikan level harga yang berbeda tetap
// dianggap setup baru, walau simbol dan timeframenya sama.
func TestSetupBerbedaTidakDitahan(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	addUser(store, "u1", "plus", "manual", true)

	svc.HandleSignal(context.Background(), siapkanDuplikat("sig-asal", enginev1.SignalStatus_READY_TO_ENTRY))
	lain := siapkanDuplikat("sig-beda", enginev1.SignalStatus_READY_TO_ENTRY)
	lain.EntryPrice = 4130.00
	svc.HandleSignal(context.Background(), lain)

	if got := len(store.userSignals); got != 2 {
		t.Fatalf("baris user_signal = %d, ingin 2 (setup berbeda)", got)
	}
}

// TestSetupKedaluwarsaBolehTerbitLagi memastikan setup yang sudah berakhir bisa
// muncul kembali: strukturnya bisa masih valid setelah jeda.
func TestSetupKedaluwarsaBolehTerbitLagi(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	addUser(store, "u1", "plus", "manual", true)

	lama := siapkanDuplikat("sig-lama", enginev1.SignalStatus_READY_TO_ENTRY)
	svc.HandleSignal(context.Background(), lama)
	store.signals["sig-lama"].Status = model.SignalStatusExpired

	svc.HandleSignal(context.Background(), siapkanDuplikat("sig-baru", enginev1.SignalStatus_READY_TO_ENTRY))

	if _, err := store.FindUserSignal(context.Background(), "u1", "sig-baru"); err != nil {
		t.Fatalf("setup setelah kedaluwarsa harus tercatat: %v", err)
	}
}

// TestPenerbitanUlangSaatPosisiTerbukaTetapDicatat memastikan user tetap
// menerima catatan penolakan \"posisi masih berjalan\" saat setup yang sama
// muncul lagi, karena memang posisinya masih terbuka.
func TestPenerbitanUlangSaatPosisiTerbukaTetapDicatat(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	svc := newService(store, pub)

	svc.HandleSignal(context.Background(), siapkanDuplikat("sig-asal", enginev1.SignalStatus_READY_TO_ENTRY))
	posisi, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(posisi) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin 1", len(posisi))
	}

	svc.HandleSignal(context.Background(), siapkanDuplikat("sig-ulang", enginev1.SignalStatus_READY_TO_ENTRY))

	us, err := store.FindUserSignal(context.Background(), "u1", "sig-ulang")
	if err != nil {
		t.Fatalf("penolakan saat posisi terbuka harus tercatat: %v", err)
	}
	if us.Status != model.SignalStatusRejected {
		t.Fatalf("status = %q, ingin REJECTED", us.Status)
	}
	if posisiSetelah, _ := store.ListOpenPositionsByUser(context.Background(), "u1"); len(posisiSetelah) != 1 {
		t.Fatalf("posisi tidak boleh bertambah, dapat %d", len(posisiSetelah))
	}
}
