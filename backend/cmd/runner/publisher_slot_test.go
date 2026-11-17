package main

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// storeSlot menyajikan jumlah peristiwa slot tetap untuk menguji kunci notifikasi.
type storeSlot struct {
	repository.RunnerRepository
	jumlah int
	err    error
	dipanggil bool
}

func (s *storeSlot) HitungKejadianSlot(context.Context, string, string, string, string, string) (int, error) {
	s.dipanggil = true
	return s.jumlah, s.err
}

// Kasus USTEC 17 Nov 2026: tahap 3 dikirim atas nama baris status
// (status-USTEC-H4-M15), lalu posisi dibuka atas nama sinyal lain
// (sig-USTEC-H4-M15-...-3). Kunci notifikasi harus memakai slot, bukan
// signal_id, supaya riwayat kedua identitas itu menyatu dan kabar tahap
// berikutnya tidak dianggap pengulangan.
func TestKejadianSlotMemakaiSatuKunciUntukSatuSetup(t *testing.T) {
	p := publisherUji(nil)
	store := &storeSlot{jumlah: 2}
	p.store = store

	sinyal := &model.Signal{
		Symbol: "USTEC", Htf: "H4", Ltf: "M15", Direction: "BUY",
	}
	if got := p.kejadianSlot(context.Background(), "u-1", sinyal); got != 2 {
		t.Fatalf("nomor kejadian slot = %d, ingin 2", got)
	}
	if !store.dipanggil {
		t.Fatal("penghitung slot harus dipanggil, bukan penghitung per signal_id")
	}
}

// Kegagalan baca tidak boleh menghilangkan notifikasi: nilainya jatuh ke nol.
func TestKejadianSlotSaatGagalBaca(t *testing.T) {
	p := publisherUji(nil)
	p.store = &storeSlot{err: apperror.ErrNotFound}
	sinyal := &model.Signal{Symbol: "USTEC", Htf: "H4", Ltf: "M15", Direction: "BUY"}
	if got := p.kejadianSlot(context.Background(), "u-1", sinyal); got != 0 {
		t.Fatalf("nomor kejadian slot = %d, ingin 0 saat baca gagal", got)
	}
}

// Jam siklus memakai zona New York sebenarnya. Notifikasi 17 Nov 2026 tertulis
// "11:18 NY" padahal jam lokal 09:00 WIB = 12:18 NY: offset tetap -5 meleset
// satu jam karena November masih EDT (UTC-4).
func TestSiklusCandleMemakaiZonaNewYork(t *testing.T) {
	// 17 Nov 2026 09:00 WIB = 02:00 UTC = 21:00 NY (16 Nov, EST belum mulai).
	terbit := time.Date(2026, 11, 17, 2, 0, 0, 0, time.UTC)
	got := siklusCandle("H4", terbit)
	if got != "Candle H4 · 21:00 NY" {
		t.Fatalf("siklus = %q, ingin \"Candle H4 · 21:00 NY\"", got)
	}
}

func TestSiklusCandleKosongTanpaHtf(t *testing.T) {
	if got := siklusCandle("", time.Now()); got != "" {
		t.Fatalf("siklus tanpa HTF = %q, ingin kosong", got)
	}
}
