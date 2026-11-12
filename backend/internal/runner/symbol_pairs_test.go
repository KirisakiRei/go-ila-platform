package runner_test

import (
	"context"
	"testing"

	"ila/backend/internal/repository"
)

// Pengujian ini mengunci perbaikan terpenting: switch timeframe milik satu
// simbol TIDAK boleh bocor ke simbol lain. Bug lama menggabungkan semua simbol
// menjadi satu daftar global, sehingga mematikan satu pair pada XAUUSD ikut
// mematikan pair itu di EURUSD.

func TestComputeSymbolPairs_PerSimbolTidakBocor(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	// XAUUSD: 2 pair dipantau.
	store.timeframes["acct-u1|XAUUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}
	store.timeframes["acct-u1|XAUUSD|ila-2.0|H1|M5"] = repository.TimeframeState{Watch: true, Auto: true}
	// EURUSD: 4 pair dipantau.
	store.timeframes["acct-u1|EURUSD|ila-2.0|W|H4"] = repository.TimeframeState{Watch: true, Auto: true}
	store.timeframes["acct-u1|EURUSD|ila-2.0|D1|H1"] = repository.TimeframeState{Watch: true, Auto: true}
	store.timeframes["acct-u1|EURUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}
	store.timeframes["acct-u1|EURUSD|ila-2.0|H1|M5"] = repository.TimeframeState{Watch: true, Auto: true}

	svc := newTimeframeService(store, pub)

	xau, err := svc.ComputeSymbolPairs(context.Background(), "XAUUSD")
	if err != nil {
		t.Fatalf("ComputeSymbolPairs(XAUUSD): %v", err)
	}
	eur, err := svc.ComputeSymbolPairs(context.Background(), "EURUSD")
	if err != nil {
		t.Fatalf("ComputeSymbolPairs(EURUSD): %v", err)
	}

	if got := len(xau["ila-2.0"].Pairs); got != 2 {
		t.Fatalf("XAUUSD harus 2 pair, dapat %d: %v", got, xau["ila-2.0"].Pairs)
	}
	if got := len(eur["ila-2.0"].Pairs); got != 4 {
		t.Fatalf("EURUSD harus 4 pair, dapat %d: %v", got, eur["ila-2.0"].Pairs)
	}
	if !xau["ila-2.0"].Restricted || !eur["ila-2.0"].Restricted {
		t.Fatalf("kedua simbol sudah diatur, Restricted harus true")
	}
}

func TestComputeSymbolPairs_SimbolBelumDiaturTidakRestricted(t *testing.T) {
	// Simbol tanpa baris saklar harus memakai daftar bawaan engine, BUKAN
	// dianggap "semua dimatikan". Ini yang mencegah simbol mati diam-diam.
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	store.timeframes["acct-u1|XAUUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}

	svc := newTimeframeService(store, pub)

	eur, err := svc.ComputeSymbolPairs(context.Background(), "EURUSD")
	if err != nil {
		t.Fatalf("ComputeSymbolPairs: %v", err)
	}
	if len(eur) != 0 {
		t.Fatalf("simbol belum diatur tidak boleh punya entri, dapat %v", eur)
	}
}

func TestComputeSymbolPairs_SemuaMatiRestrictedKosong(t *testing.T) {
	// Semua switch dimatikan (Opsi A): Restricted=true dengan daftar kosong.
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	store.timeframes["acct-u1|XAUUSD|ila-3.0|D|15M"] = repository.TimeframeState{Watch: false, Auto: false}

	svc := newTimeframeService(store, pub)

	got, err := svc.ComputeSymbolPairs(context.Background(), "XAUUSD")
	if err != nil {
		t.Fatalf("ComputeSymbolPairs: %v", err)
	}
	policy, ada := got["ila-3.0"]
	if !ada {
		t.Fatalf("metode yang punya baris saklar harus tetap muncul")
	}
	if !policy.Restricted {
		t.Fatalf("harus Restricted")
	}
	if len(policy.Pairs) != 0 {
		t.Fatalf("daftar harus kosong, dapat %v", policy.Pairs)
	}
}

func TestComputeSymbolPairs_FlagMatiTidakRestricted(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	store.timeframes["acct-u1|XAUUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}

	// newService tanpa flag: fitur timeframe belum dinyalakan.
	svc := newService(store, pub)

	got, err := svc.ComputeSymbolPairs(context.Background(), "XAUUSD")
	if err != nil {
		t.Fatalf("ComputeSymbolPairs: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("flag mati harus menghasilkan kebijakan kosong, dapat %v", got)
	}
}
