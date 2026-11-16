package runner_test

import (
	"testing"
	"time"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/broker"
	"ila/backend/internal/runner"
)

// Kasus nyata 16 Nov 2026: Runner terhubung ke engine pada Sabtu 14 Nov dan
// menerima status CLOSED. Koneksinya bertahan sampai Senin siang, dan karena
// engine dulu hanya mengirim status sekali di awal stream, Runner terus
// memegang CLOSED. Akibatnya semua setup ditahan dengan alasan
// "Pasar sedang tutup." padahal pasar sudah buka.
//
// Uji ini mengunci dua hal:
//  1. Status terbaru dari engine tetap dihormati selama masih segar.
//  2. Status yang sudah basi tidak dipercaya; keputusan jatuh ke jadwal pasar.

// jamPasarBuka mengembalikan Senin 10:00 UTC, saat pasar forex buka.
func jamPasarBuka() time.Time {
	return time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
}

// jamPasarTutup mengembalikan Sabtu 12:00 waktu New York.
func jamPasarTutup() time.Time {
	return time.Date(2026, 11, 14, 12, 0, 0, 0, runner.NY)
}

// buatServicePasar membuat Runner dengan jam yang bisa digeser oleh tes.
func buatServicePasar(t *testing.T, jamFunc func() time.Time) *runner.Service {
	t.Helper()
	registry := broker.NewRegistry()
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})
	return runner.NewService(newFakeStore(), registry, &fakePublisher{}, discardLogger(), runner.Options{
		Policy:          runner.DefaultDailyPolicy(),
		AutoExecute:     true,
		MarketOpenCheck: true,
		Clock:           jamFunc,
	})
}

// TestStatusPasarBasiTidakDipercaya adalah inti perbaikan: status CLOSED yang
// diterima jauh sebelumnya tidak boleh menahan eksekusi saat pasar sudah buka.
func TestStatusPasarBasiTidakDipercaya(t *testing.T) {
	sekarang := jamPasarTutup()
	svc := buatServicePasar(t, func() time.Time { return sekarang })

	// Status diterima Sabtu, ketika pasar memang tutup.
	svc.SetMarketState(enginev1.MarketState_MARKET_CLOSED)
	if svc.MarketOpen() {
		t.Fatal("prasyarat: CLOSED yang baru diterima harus dihormati")
	}

	// Dua hari kemudian tanpa kabar baru dari engine, pasar sudah buka.
	sekarang = jamPasarBuka()

	if !svc.MarketOpen() {
		t.Fatal("status CLOSED dari Sabtu tidak boleh dipakai pada Senin; " +
			"pasar seharusnya dianggap buka menurut jadwal lokal")
	}
}

// TestStatusPasarSegarTetapDipakai memastikan jaring pengaman tidak menimpa
// status yang benar-benar baru dari engine.
func TestStatusPasarSegarTetapDipakai(t *testing.T) {
	sekarang := jamPasarBuka()
	svc := buatServicePasar(t, func() time.Time { return sekarang })

	svc.SetMarketState(enginev1.MarketState_MARKET_CLOSED)

	if svc.MarketOpen() {
		t.Fatal("status CLOSED yang baru saja diterima harus dihormati")
	}
}

// TestStatusPasarBasiSaatPasarTutup memastikan jadwal lokal juga menutup pasar
// dengan benar ketika memang akhir pekan. Status terakhir dari engine adalah
// OPEN yang sudah basi (misalnya terakhir diterima Jumat), sehingga tidak boleh
// membuat pasar dianggap buka sepanjang akhir pekan.
func TestStatusPasarBasiSaatPasarTutup(t *testing.T) {
	// Jumat 10:00 UTC: pasar buka, status OPEN diterima dari engine.
	jumat := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	sekarang := jumat
	svc := buatServicePasar(t, func() time.Time { return sekarang })

	svc.SetMarketState(enginev1.MarketState_MARKET_OPEN)
	if !svc.MarketOpen() {
		t.Fatal("prasyarat: OPEN yang baru diterima harus dihormati")
	}

	// Sabtu tanpa kabar baru lagi: jadwal lokal yang menentukan.
	sekarang = jamPasarTutup()

	if svc.MarketOpen() {
		t.Fatal("pasar seharusnya tutup pada Sabtu menurut jadwal lokal")
	}
}
