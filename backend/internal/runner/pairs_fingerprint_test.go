package runner_test

import (
	"context"
	"sync"
	"testing"

	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
)

// Pengujian ini mengunci bug kedua: sidik pasangan timeframe dulu dibentuk
// TANPA identitas simbol, sehingga perubahan switch satu simbol bisa tidak
// mengubah sidik. Kalau sidik tidak berubah, penyelarasan ke engine tidak
// pernah terpicu dan perubahan di dashboard tidak sampai ke engine.
//
// Skenario di bawah dipilih khusus agar daftar pasangan GABUNGAN tetap sama
// persis dan status restricted tidak berubah. Dengan begitu satu-satunya yang
// bisa membedakan kedua kondisi adalah identitas simbol — inilah yang
// membuktikan perbaikan bekerja, dan akan gagal bila simbol dibuang lagi.

func siapkanSinkron(t *testing.T, store *fakeStore) (*runner.Service, func() int) {
	t.Helper()
	addUser(store, "u1", "plus", "auto", true)

	svc := newTimeframeService(store, &fakePublisher{})

	var mu sync.Mutex
	panggilan := 0
	svc.SetSymbolSync("ila-2.0", func(context.Context, []string, []string) error {
		mu.Lock()
		panggilan++
		mu.Unlock()
		return nil
	})

	return svc, func() int {
		mu.Lock()
		defer mu.Unlock()
		return panggilan
	}
}

func TestSidikPasanganBerubahSaatSimbolLainDiubah(t *testing.T) {
	store := newFakeStore()
	// XAUUSD memantau H4-M15, EURUSD memantau W-H4.
	store.timeframes["acct-u1|XAUUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}
	store.timeframes["acct-u1|EURUSD|ila-2.0|W|H4"] = repository.TimeframeState{Watch: true, Auto: true}

	svc, hitung := siapkanSinkron(t, store)
	ctx := context.Background()
	simbol := []string{"XAUUSD", "EURUSD"}

	if err := svc.SyncSubscriptions(ctx, "ila-2.0", simbol); err != nil {
		t.Fatalf("sinkron pertama: %v", err)
	}
	if got := hitung(); got != 1 {
		t.Fatalf("sinkron pertama harus memanggil engine, dapat %d", got)
	}

	// Tukar pasangan antar simbol. Daftar GABUNGAN (H4/M15 + W/H4) tetap sama
	// persis, jadi sidik lama (tanpa simbol) tidak akan berubah sama sekali.
	store.timeframes["acct-u1|XAUUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: false, Auto: false}
	store.timeframes["acct-u1|EURUSD|ila-2.0|W|H4"] = repository.TimeframeState{Watch: false, Auto: false}
	store.timeframes["acct-u1|XAUUSD|ila-2.0|W|H4"] = repository.TimeframeState{Watch: true, Auto: true}
	store.timeframes["acct-u1|EURUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}

	if err := svc.SyncSubscriptions(ctx, "ila-2.0", simbol); err != nil {
		t.Fatalf("sinkron kedua: %v", err)
	}
	if got := hitung(); got != 2 {
		t.Fatalf("sidik harus berubah walau gabungan pasangan tetap sama "+
			"(panggilan engine %d, ingin 2) — ini regresi bug sidik tanpa simbol", got)
	}
}

func TestSidikPasanganMembedakanDuaSimbol(t *testing.T) {
	// Dua simbol dengan pair identik tetap harus diperlakukan berbeda, karena
	// engine menerima kebijakan per simbol.
	store := newFakeStore()
	store.timeframes["acct-u1|XAUUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}

	svc, hitung := siapkanSinkron(t, store)
	ctx := context.Background()

	if err := svc.SyncSubscriptions(ctx, "ila-2.0", []string{"XAUUSD"}); err != nil {
		t.Fatalf("sinkron XAUUSD: %v", err)
	}

	store.timeframes["acct-u1|EURUSD|ila-2.0|H4|M15"] = repository.TimeframeState{Watch: true, Auto: true}
	if err := svc.SyncSubscriptions(ctx, "ila-2.0", []string{"XAUUSD", "EURUSD"}); err != nil {
		t.Fatalf("sinkron dua simbol: %v", err)
	}

	if got := hitung(); got != 2 {
		t.Fatalf("simbol baru dengan pair sama tetap harus mengirim ulang (%d, ingin 2)", got)
	}
}
