package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
)

// TestExpireStaleSignalsMenutupSinyalLewat memastikan sinyal yang bisa
// dieksekusi tetapi sudah lewat masa berlaku ditutup, sedangkan sinyal yang
// masih hidup dan sinyal yang sudah dieksekusi tidak disentuh.
func TestExpireStaleSignalsMenutupSinyalLewat(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})

	now := time.Now().UTC()
	store.signals["sig-lama"] = &model.Signal{
		SignalID:  "sig-lama",
		Status:    model.SignalStatusReadyToEntry,
		ExpiresAt: now.Add(-time.Minute),
	}
	store.signals["sig-baru"] = &model.Signal{
		SignalID:  "sig-baru",
		Status:    model.SignalStatusReadyToEntry,
		ExpiresAt: now.Add(time.Hour),
	}
	store.signals["sig-acked"] = &model.Signal{
		SignalID:  "sig-acked",
		Status:    model.SignalStatusAcked,
		ExpiresAt: now.Add(-time.Minute),
	}
	store.signals["sig-consumed"] = &model.Signal{
		SignalID:  "sig-consumed",
		Status:    model.SignalStatusConsumed,
		ExpiresAt: now.Add(-time.Minute),
	}

	svc.ExpireStaleSignals(context.Background())

	if got := store.signals["sig-lama"].Status; got != model.SignalStatusExpired {
		t.Errorf("sinyal lewat masa berlaku = %q, ingin EXPIRED", got)
	}
	if got := store.signals["sig-baru"].Status; got != model.SignalStatusReadyToEntry {
		t.Errorf("sinyal yang masih hidup = %q, ingin tetap READY_TO_ENTRY", got)
	}
	// ACKED berarti user sudah menerima sinyal; mengubahnya menjadi EXPIRED
	// akan menyamarkan jejak eksekusi.
	if got := store.signals["sig-acked"].Status; got != model.SignalStatusAcked {
		t.Errorf("sinyal ACKED = %q, ingin tetap ACKED", got)
	}
	if got := store.signals["sig-consumed"].Status; got != model.SignalStatusConsumed {
		t.Errorf("sinyal CONSUMED = %q, ingin tetap CONSUMED", got)
	}
}

// TestExpireStaleSignalsTidakMenyentuhStatusMenunggu menjaga radar satu slot.
// Baris WAIT_* memakai identitas tetap dan dikirim ulang engine; bila ditandai
// EXPIRED, Runner menganggapnya final selamanya dan slot itu mati.
func TestExpireStaleSignalsTidakMenyentuhStatusMenunggu(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})

	now := time.Now().UTC()
	for i, status := range []string{
		model.SignalStatusWaitHTFSweep,
		model.SignalStatusWaitLTFCISD,
		"WAIT_ILA3_MTF_REVERSAL",
	} {
		id := "status-" + status
		store.signals[id] = &model.Signal{
			SignalID:  id,
			Status:    status,
			ExpiresAt: now.Add(-time.Duration(i+1) * time.Minute),
		}
	}

	svc.ExpireStaleSignals(context.Background())

	for id, sig := range store.signals {
		if sig.Status == model.SignalStatusExpired {
			t.Errorf("baris status %s tidak boleh ditandai EXPIRED", id)
		}
	}
}

// TestExpireStaleSignalsIdempotent memastikan penyapu aman dipanggil berulang.
func TestExpireStaleSignalsIdempotent(t *testing.T) {
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})

	store.signals["sig-lama"] = &model.Signal{
		SignalID:  "sig-lama",
		Status:    model.SignalStatusOpportunity,
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
	}

	svc.ExpireStaleSignals(context.Background())
	svc.ExpireStaleSignals(context.Background())

	if got := store.signals["sig-lama"].Status; got != model.SignalStatusExpired {
		t.Errorf("status akhir = %q, ingin EXPIRED", got)
	}
}
