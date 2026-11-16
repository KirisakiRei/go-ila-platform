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
	pesan := FormatMenuMessage("Ryn", "Plus")

	if len(pesan.Buttons) != 5 {
		t.Fatalf("menu harus punya 5 tombol, dapat %d", len(pesan.Buttons))
	}
	for _, baris := range pesan.Buttons {
		if len(baris) != 1 || baris[0].CallbackData == "" {
			t.Fatalf("setiap tombol menu harus satu baris dengan callback: %+v", baris)
		}
	}
	if !strings.Contains(pesan.Text, "Halo, Ryn") {
		t.Fatalf("menu harus menyapa nama depan: %q", pesan.Text)
	}
}
