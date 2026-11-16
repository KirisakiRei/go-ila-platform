package telegram

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestPesanBertombolTetapMembawaTombol mengunci aturan Telegram hanya menerima
// satu reply_markup: tombol inline tidak boleh tertimpa elemen lain.
func TestPesanBertombolTetapMembawaTombol(t *testing.T) {
	client := &Client{}
	payload := client.pesanPayload("123", Message{
		Text: "menu",
		Buttons: [][]InlineButton{{
			{Text: "Pantau", CallbackData: CallbackPantau},
		}},
	})

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("payload tidak bisa disusun: %v", err)
	}
	if !strings.Contains(string(body), "inline_keyboard") {
		t.Fatalf("tombol inline hilang dari payload: %s", body)
	}
	if !strings.Contains(string(body), CallbackPantau) {
		t.Fatalf("callback tombol hilang dari payload: %s", body)
	}
}

func TestMenuPunyaTombolUntukSetiapHalaman(t *testing.T) {
	pesan := FormatMenuMessage("Ryn", "Plus", "https://rekavia.com/app/dashboard")

	// Dua tombol per baris, panduan sendiri di baris terakhir.
	if len(pesan.Buttons) != 3 {
		t.Fatalf("menu harus 3 baris tombol, dapat %d", len(pesan.Buttons))
	}
	if len(pesan.Buttons[0]) != 2 || len(pesan.Buttons[1]) != 2 || len(pesan.Buttons[2]) != 1 {
		t.Fatalf("susunan tombol harus 2-2-1, dapat %+v", pesan.Buttons)
	}
	total := 0
	for _, baris := range pesan.Buttons {
		for _, tombol := range baris {
			total++
			// Tombol dashboard membuka tautan, bukan mengirim callback.
			if tombol.CallbackData == "" && tombol.URL == "" {
				t.Fatalf("tombol %q harus punya callback atau URL", tombol.Text)
			}
		}
	}
	if !strings.HasSuffix(pesan.Buttons[2][0].URL, "/dashboard") {
		t.Fatalf("tombol terakhir harus menuju dashboard, dapat %q", pesan.Buttons[2][0].URL)
	}
	if total != 5 {
		t.Fatalf("menu harus 5 tombol, dapat %d", total)
	}
	if !strings.Contains(pesan.Text, "Halo, Ryn") {
		t.Fatalf("menu harus menyapa nama depan: %q", pesan.Text)
	}
}
