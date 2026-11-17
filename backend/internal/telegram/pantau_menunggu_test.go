package telegram

import (
	"strings"
	"testing"
	"time"
)

// barisMenungguUji menyusun satu setup menunggu di tahap yang ditentukan.
func barisMenungguUji(symbol, htf, ltf, tahap, teks string) BarisMenunggu {
	return BarisMenunggu{
		Symbol:       symbol,
		Htf:          htf,
		Ltf:          ltf,
		Direction:    "SELL",
		Tahap:        tahap,
		TahapTeks:    teks,
		Target:       110207,
		StopLoss:     110732,
		TriggerHarga: 110582,
		Sejak:        time.Date(2026, 11, 16, 16, 0, 0, 0, time.UTC),
		Lama:         "1 jam 5 menit",
	}
}

// TestPantauMenampilkanPairTimeframeDanTahap memastikan setiap setup menunggu
// menyebut pair, pasangan timeframe, dan tahapnya. Tanpa itu trader hanya
// melihat nama pair tanpa tahu sedang menunggu apa.
func TestPantauMenampilkanPairTimeframeDanTahap(t *testing.T) {
	pesan := FormatPantauMessage(PantauParams{
		Diperbarui: time.Now(),
		Menunggu:   []BarisMenunggu{barisMenungguUji("EURUSD", "H1", "M5", "3/6", "Cari entry di M5")},
	})

	// 16:00 UTC = 23:00 WIB dan 11:00 NY; kedua zona wajib tampil agar sesi
	// pasar terbaca tanpa menghitung manual.
	for _, ingin := range []string{"EURUSD · H1 ➔ M5", "3/6", "Cari entry di M5", "Sejak", "1 jam 5 menit", "23:00 WIB", "11:00 NY"} {
		if !strings.Contains(pesan.Text, ingin) {
			t.Fatalf("pesan pantauan tidak memuat %q:\n%s", ingin, pesan.Text)
		}
	}
}

// TestPantauMelewatiHargaKosong memastikan baris harga yang belum tersedia tidak
// ditampilkan sebagai tanda hubung bertumpuk.
func TestPantauMelewatiHargaKosong(t *testing.T) {
	pesan := FormatPantauMessage(PantauParams{
		Diperbarui: time.Now(),
		Menunggu: []BarisMenunggu{{
			Symbol: "XAUUSD", Htf: "H4", Ltf: "M15",
			Direction: "BUY", Tahap: "1/6", TahapTeks: "Baca arah H4",
		}},
	})

	if strings.Contains(pesan.Text, "Target") || strings.Contains(pesan.Text, "Trigger") {
		t.Fatalf("harga yang belum ada tidak boleh ditampilkan:\n%s", pesan.Text)
	}
	if !strings.Contains(pesan.Text, "Baca arah H4") {
		t.Fatalf("tahap harus tetap tampil:\n%s", pesan.Text)
	}
}

// TestTombolHalamanPantau memastikan navigasi memakai bentuk
// ‹ Sebelumnya | 1/2 | Berikutnya › dan tombol yang tidak mungkin ditekan tidak
// memiliki aksi.
func TestTombolHalamanPantau(t *testing.T) {
	pesan := FormatPantauMessage(PantauParams{
		Diperbarui:   time.Now(),
		Menunggu:     []BarisMenunggu{barisMenungguUji("EURUSD", "H1", "M5", "3/6", "Cari entry di M5")},
		Halaman:      1,
		TotalHalaman: 2,
	})

	navigasi := pesan.Buttons[0]
	if len(navigasi) != 3 {
		t.Fatalf("navigasi harus tiga tombol, dapat %d", len(navigasi))
	}
	if navigasi[0].CallbackData != CallbackNoop {
		t.Fatal("tombol sebelumnya harus tanpa aksi di halaman pertama")
	}
	if navigasi[1].Text != "Kembali" {
		t.Fatalf("penanda halaman = %q, ingin 1/2", navigasi[1].Text)
	}
	if navigasi[2].CallbackData != CallbackPantauHalaman+"2" {
		t.Fatalf("tombol berikutnya = %q, ingin %s2", navigasi[2].CallbackData, CallbackPantauHalaman)
	}
	if !punyaTombolMenu(pesan) {
		t.Fatal("halaman pantauan harus punya jalan kembali ke menu")
	}
}

// TestTombolHalamanPantauTerakhir memastikan tombol berikutnya nonaktif di
// halaman terakhir.
func TestTombolHalamanPantauTerakhir(t *testing.T) {
	pesan := FormatPantauMessage(PantauParams{
		Diperbarui:   time.Now(),
		Menunggu:     []BarisMenunggu{barisMenungguUji("EURUSD", "H1", "M5", "3/6", "Cari entry di M5")},
		Halaman:      2,
		TotalHalaman: 2,
	})

	navigasi := pesan.Buttons[0]
	if navigasi[0].CallbackData != CallbackPantauHalaman+"1" {
		t.Fatalf("tombol sebelumnya = %q, ingin %s1", navigasi[0].CallbackData, CallbackPantauHalaman)
	}
	if navigasi[2].CallbackData != CallbackNoop {
		t.Fatal("tombol berikutnya harus tanpa aksi di halaman terakhir")
	}
}

// punyaTombolMenu melaporkan apakah salah satu tombol kembali ke menu utama.
func punyaTombolMenu(pesan Message) bool {
	for _, baris := range pesan.Buttons {
		for _, tombol := range baris {
			if tombol.CallbackData == CallbackMenu {
				return true
			}
		}
	}
	return false
}
