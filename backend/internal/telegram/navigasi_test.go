package telegram

import (
	"testing"
	"time"
)

// TestSetiapHalamanPunyaJalanKembaliKeMenu mengunci tombol "Menu" pada semua
// halaman. Tanpa itu, trader terjebak di satu halaman tanpa tombol apa pun.
func TestSetiapHalamanPunyaJalanKembaliKeMenu(t *testing.T) {
	halaman := map[string]Message{
		"menu":   FormatMenuMessage("Ryn", "Plus"),
		"pantau": FormatPantauMessage(PantauParams{Diperbarui: time.Now()}),
		"riwayat": FormatRiwayatMessage(RiwayatParams{
			Baris:   []BarisRiwayat{{Symbol: "XAUUSD", ClosedAt: time.Now()}},
			Halaman: 1, TotalHalaman: 2, TotalPosisi: 6,
		}),
		"riwayat kosong": FormatRiwayatMessage(RiwayatParams{}),
		"hari":           FormatHariMessage(HariParams{Hari: "16 Nov 2026"}),
		"pengaturan":     FormatPengaturanMessage(""),
		"bantuan":        FormatBantuanMessage(),
	}

	for nama, pesan := range halaman {
		tombol := 0
		adaMenu := false
		for _, baris := range pesan.Buttons {
			for _, b := range baris {
				tombol++
				if b.CallbackData == CallbackMenu {
					adaMenu = true
				}
			}
		}
		// Halaman menu utama sendiri memuat semua tombol tujuan.
		if nama == "menu" {
			adaMenu = len(pesan.Buttons) > 0
		}
		if !adaMenu {
			t.Fatalf("halaman %q tidak punya tombol kembali ke menu", nama)
		}
		if tombol == 0 {
			t.Fatalf("halaman %q tidak punya tombol apa pun", nama)
		}
	}
}
