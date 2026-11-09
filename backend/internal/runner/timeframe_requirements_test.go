package runner_test

import (
	"context"
	"testing"

	"ila/backend/internal/repository"
)

// TestComputeTimeframeRequirements_MenggabungkanPasanganDipantau memastikan
// kebutuhan engine memuat gabungan pasangan yang dipantau, dan tidak memuat
// pasangan yang tidak dipantau.
func TestComputeTimeframeRequirements_MenggabungkanPasanganDipantau(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	addUser(store, "u2", "plus", "auto", true)

	// u1 memantau W1-H4; u2 memantau H4-M15.
	store.timeframes["acct-u1|XAUUSD|ila-2.0|W1|H4"] = repository.TimeframeState{Watch: true, Auto: true}
	store.timeframes["acct-u2|XAUUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: false}
	// Tidak dipantau: tidak boleh diminta ke engine.
	store.timeframes["acct-u1|XAUUSD|ila-2.0|M15|M5"] = repository.TimeframeState{Watch: false, Auto: false}

	svc := newTimeframeService(store, pub)
	reqs, err := svc.ComputeTimeframeRequirements(context.Background(), "XAUUSD")
	if err != nil {
		t.Fatalf("ComputeTimeframeRequirements: %v", err)
	}

	list := reqs["ila-2.0"]
	if len(list) != 2 {
		t.Fatalf("pasangan = %d (%v), ingin 2 (W1-H4 dan H4-M15)", len(list), list)
	}
	// Urutan dijamin stabil oleh implementasi.
	if list[0].Htf != "H4" || list[0].Ltf != "M15" {
		t.Fatalf("pasangan pertama = %s-%s, ingin H4-M15", list[0].Htf, list[0].Ltf)
	}
	if list[1].Htf != "W1" || list[1].Ltf != "H4" {
		t.Fatalf("pasangan kedua = %s-%s, ingin W1-H4", list[1].Htf, list[1].Ltf)
	}
}

// TestComputeTimeframeRequirements_KosongSaatFlagMati memastikan perilaku lama
// tidak berubah selama fitur timeframe belum dinyalakan.
func TestComputeTimeframeRequirements_KosongSaatFlagMati(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	store.timeframes["acct-u1|XAUUSD|ila-2.0|W1|H4"] = repository.TimeframeState{Watch: true, Auto: true}

	svc := newService(store, pub) // tanpa flag
	reqs, err := svc.ComputeTimeframeRequirements(context.Background(), "XAUUSD")
	if err != nil {
		t.Fatalf("ComputeTimeframeRequirements: %v", err)
	}
	if len(reqs) != 0 {
		t.Fatalf("reqs = %v, ingin kosong saat flag mati", reqs)
	}
}
