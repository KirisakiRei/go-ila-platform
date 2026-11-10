package runner

import (
	"context"
	"sync/atomic"
	"testing"
)

// Pengujian ini mengunci janji bahwa perubahan saklar timeframe benar-benar
// sampai ke engine.
//
// Sebelumnya pasangan hanya dikirim sekali saat stream terbuka. User yang
// mengubah saklar di dashboard tidak melihat akibat apa pun sampai Runner
// kehilangan koneksi, dan itu terbaca sebagai "saklarnya rusak".

// hitungPasanganDari menyusun sidik pasangan tanpa basis data.
//
// SyncSubscriptions memanggil ComputeTimeframeRequirements yang butuh store;
// pengujian di sini memakai sidik yang disuntikkan langsung agar tidak
// bergantung pada basis data.
func sidikDari(daftar ...string) string {
	out := ""
	for _, d := range daftar {
		out += d + "\n"
	}
	return out
}

func TestPasanganBerubahMemicuKirimUlang(t *testing.T) {
	var panggilan int32
	svc := &Service{logger: silent()}
	svc.SetSymbolSync("ila-2.0", func(context.Context, []string, []string) error {
		atomic.AddInt32(&panggilan, 1)
		return nil
	})

	// Panggilan pertama: simbol terkirim, sidik pasangan dikunci.
	err := svc.syncDenganSidik(context.Background(), "ila-2.0", []string{"XAUUSD"}, sidikDari("ila-2.0:H4/M15"))
	if err != nil {
		t.Fatalf("sinkron pertama: %v", err)
	}
	if got := atomic.LoadInt32(&panggilan); got != 1 {
		t.Fatalf("panggilan pertama = %d, ingin 1", got)
	}

	// Sama persis: tidak boleh mengirim apa pun.
	if err := svc.syncDenganSidik(context.Background(), "ila-2.0", []string{"XAUUSD"}, sidikDari("ila-2.0:H4/M15")); err != nil {
		t.Fatalf("sinkron kedua: %v", err)
	}
	if got := atomic.LoadInt32(&panggilan); got != 1 {
		t.Fatalf("panggilan tidak berubah = %d, ingin tetap 1", got)
	}

	// Simbol sama tetapi pasangan berubah: wajib dikirim ulang.
	if err := svc.syncDenganSidik(context.Background(), "ila-2.0", []string{"XAUUSD"}, sidikDari("ila-2.0:W/H4")); err != nil {
		t.Fatalf("sinkron ketiga: %v", err)
	}
	if got := atomic.LoadInt32(&panggilan); got != 2 {
		t.Fatalf("perubahan pasangan harus mengirim ulang, panggilan = %d ingin 2", got)
	}
}

func TestReconnectMengirimUlangPasangan(t *testing.T) {
	var panggilan int32
	svc := &Service{logger: silent()}
	svc.SetSymbolSync("ila-2.0", func(context.Context, []string, []string) error {
		atomic.AddInt32(&panggilan, 1)
		return nil
	})

	if err := svc.syncDenganSidik(context.Background(), "ila-2.0", []string{"XAUUSD"}, sidikDari("ila-2.0:H4/M15")); err != nil {
		t.Fatalf("sinkron pertama: %v", err)
	}

	// Engine bisa lupa setelah putus: memasang ulang penyelaras harus
	// membuang sidik, bukan menganggap semuanya sudah terkirim.
	svc.SetSymbolSync("ila-2.0", func(context.Context, []string, []string) error {
		atomic.AddInt32(&panggilan, 1)
		return nil
	})

	if err := svc.syncDenganSidik(context.Background(), "ila-2.0", []string{"XAUUSD"}, sidikDari("ila-2.0:H4/M15")); err != nil {
		t.Fatalf("sinkron setelah reconnect: %v", err)
	}
	if got := atomic.LoadInt32(&panggilan); got != 2 {
		t.Fatalf("setelah reconnect harus mengirim ulang, panggilan = %d ingin 2", got)
	}
}
