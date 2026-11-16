package telegram

import (
	"strings"
	"testing"
	"time"
)

func TestPantauTanpaHargaMenampilkanStrip(t *testing.T) {
	pesan := FormatPantauMessage(PantauParams{
		Diperbarui: time.Now(),
		Posisi: []BarisPantau{{
			Ticket:     "8492015",
			Symbol:     "XAUUSD",
			Direction:  "SELL",
			Lots:       0.1,
			EntryPrice: 3880.24,
			// NowPrice nol: harga live belum tersedia.
		}},
		HargaTersedia: false,
	})

	if strings.Contains(pesan.Text, "3858") {
		t.Fatal("harga lama tidak boleh dipakai sebagai harga kini")
	}
	if !strings.Contains(pesan.Text, "Kini") || !strings.Contains(pesan.Text, "—") {
		t.Fatalf("kolom harga kini harus diisi tanda hubung, dapat %q", pesan.Text)
	}
	if strings.Contains(pesan.Text, "Untung/Rugi") {
		t.Fatal("untung/rugi tidak boleh tampil tanpa harga live")
	}
}

func TestPantauDenganHargaMenampilkanHasil(t *testing.T) {
	pesan := FormatPantauMessage(PantauParams{
		Diperbarui: time.Now(),
		Posisi: []BarisPantau{{
			Ticket:     "8492015",
			Symbol:     "XAUUSD",
			Direction:  "SELL",
			Lots:       0.1,
			EntryPrice: 3880.24,
			NowPrice:   3866.10,
			OutcomeR:   0.94,
			ProfitLoss: 67.20,
			HasPnL:     true,
		}},
		HargaTersedia: true,
	})

	if !strings.Contains(pesan.Text, "Untung/Rugi") {
		t.Fatalf("hasil harus tampil saat harga tersedia, dapat %q", pesan.Text)
	}
}

func TestPaginasiRiwayatMentok(t *testing.T) {
	baris := []BarisRiwayat{{Symbol: "XAUUSD", Direction: "BUY", OutcomeR: 2, ClosedAt: time.Now()}}

	pertama := FormatRiwayatMessage(RiwayatParams{Baris: baris, Halaman: 1, TotalHalaman: 3, TotalPosisi: 11})
	if nav := navHalaman(t, pertama); nav[0].CallbackData != "" {
		t.Fatal("tombol sebelumnya harus nonaktif di halaman pertama")
	}

	terakhir := FormatRiwayatMessage(RiwayatParams{Baris: baris, Halaman: 3, TotalHalaman: 3, TotalPosisi: 11})
	if nav := navHalaman(t, terakhir); nav[2].CallbackData != "" {
		t.Fatal("tombol berikutnya harus nonaktif di halaman terakhir")
	}

	tengah := FormatRiwayatMessage(RiwayatParams{Baris: baris, Halaman: 2, TotalHalaman: 3, TotalPosisi: 11})
	nav := navHalaman(t, tengah)
	if nav[0].CallbackData == "" || nav[2].CallbackData == "" {
		t.Fatal("di halaman tengah kedua tombol harus aktif")
	}
	if !strings.Contains(nav[1].Text, "2/3") {
		t.Fatalf("penanda halaman harus 2/3, dapat %q", nav[1].Text)
	}
}

// navHalaman mengambil baris navigasi (sebelumnya, penanda, berikutnya).
func navHalaman(t *testing.T, pesan Message) []InlineButton {
	t.Helper()
	if len(pesan.Buttons) == 0 || len(pesan.Buttons[0]) != 3 {
		t.Fatalf("navigasi riwayat harus tiga tombol, dapat %+v", pesan.Buttons)
	}
	return pesan.Buttons[0]
}

func TestAksiDariTeks(t *testing.T) {
	aksi, kenal := AksiDariTeks(LabelPantau)
	if !kenal || aksi != CallbackPantau {
		t.Fatalf("label pantau harus memetakan ke %q, dapat %q", CallbackPantau, aksi)
	}
	aksi, kenal = AksiDariTeks(LabelRiwayat)
	if !kenal || aksi != CallbackRiwayat+"0" {
		t.Fatalf("label riwayat harus membuka halaman pertama, dapat %q", aksi)
	}
}
