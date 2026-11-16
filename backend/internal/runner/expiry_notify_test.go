package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// TestExpireMengabariPenungguHarga memastikan user yang sempat diberi tahu
// bahwa setup menunggu harga kembali ikut dikabari saat jendelanya habis.
// Tanpa kabar kedua, mereka hanya melihat pesan "menunggu" tanpa pernah tahu
// setupnya berakhir.
func TestExpireMengabariPenungguHarga(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	svc := newService(store, pub)

	store.signals["sig-1"] = &model.Signal{
		SignalID:  "sig-1",
		Status:    model.SignalStatusReadyToEntry,
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
	}
	store.userSignals["u-1|sig-1"] = &model.UserSignal{
		UserID:       "u-1",
		SignalID:     "sig-1",
		Status:       model.SignalStatusRejected,
		RejectReason: "Harga sudah bergerak terlalu jauh dari level entry.",
	}

	svc.ExpireStaleSignals(context.Background())

	if got := store.userSignals["u-1|sig-1"].Status; got != model.SignalStatusExpired {
		t.Errorf("status user signal = %q, ingin EXPIRED", got)
	}

	var dikabari bool
	for _, panggilan := range pub.signals {
		if panggilan.SignalID == "sig-1" && panggilan.Status == model.SignalStatusRejected {
			dikabari = true
		}
	}
	if !dikabari {
		t.Fatal("user penunggu harga tidak dikabari saat setup dibatalkan")
	}
}

// TestExpireTidakMengabariUserLain memastikan user yang tertahan karena alasan
// lain tidak menerima pesan batal tambahan.
func TestExpireTidakMengabariUserLain(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	svc := newService(store, pub)

	store.signals["sig-2"] = &model.Signal{
		SignalID:  "sig-2",
		Status:    model.SignalStatusReadyToEntry,
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
	}
	store.userSignals["u-2|sig-2"] = &model.UserSignal{
		UserID:       "u-2",
		SignalID:     "sig-2",
		Status:       model.SignalStatusRejected,
		RejectReason: string(runner.RejectDailyQuota),
	}

	svc.ExpireStaleSignals(context.Background())

	if len(pub.signals) != 0 {
		t.Fatalf("user tertahan kuota tidak boleh dikabari ulang, dapat %d pesan",
			len(pub.signals))
	}
}
