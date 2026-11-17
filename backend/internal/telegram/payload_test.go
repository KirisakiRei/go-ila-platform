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
	if len(pesan.Buttons) != 4 {
		t.Fatalf("menu harus 3 baris tombol, dapat %d", len(pesan.Buttons))
	}
	if len(pesan.Buttons[0]) != 1 || len(pesan.Buttons[1]) != 2 || len(pesan.Buttons[2]) != 1 {
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
	if total != 6 {
		t.Fatalf("menu harus 5 tombol, dapat %d", total)
	}
	if !strings.Contains(pesan.Text, "Halo, Ryn") {
		t.Fatalf("menu harus menyapa nama depan: %q", pesan.Text)
	}
}

func TestEveryMenuKeyboardHasValidActions(t *testing.T) {
	messages := []Message{FormatMenuMessage("Ryn", "PLUS", "https://rekavia.com/app/dashboard"), FormatBantuanMessage(), FormatPanduanMessage(), FormatRiwayatMenu(), FormatPengaturanMessage("https://rekavia.com/app/pengaturan")}
	for page := 1; page <= 3; page++ {
		messages = append(messages, FormatPantauMessage(PantauParams{Halaman: page, TotalHalaman: 3}), FormatRiwayatMessage(RiwayatParams{Halaman: page, TotalHalaman: 3, TotalPosisi: 11}))
	}
	for _, msg := range messages {
		for _, row := range msg.Buttons {
			for _, button := range row {
				if (button.URL == "") == (button.CallbackData == "") {
					t.Fatalf("exactly one action required: %+v", button)
				}
				if len(button.CallbackData) > 64 {
					t.Fatalf("callback too long: %+v", button)
				}
				if button.Style != "" && button.Style != "primary" && button.Style != "success" && button.Style != "danger" {
					t.Fatalf("unsupported style: %+v", button)
				}
			}
		}
	}
	body, err := json.Marshal(FormatMenuMessage("Ryn", "PLUS", "").Buttons)
	if err != nil || !strings.Contains(string(body), "primary") {
		t.Fatalf("missing button style: %s %v", body, err)
	}
	if strings.Contains(FormatMenuMessage("Ryn", "PLUS", "").Text, "PLUS") {
		t.Fatal("home must not advertise package")
	}
}

func TestRadarAlwaysIdentifiesSourceStrategy(t *testing.T) {
	for _, version := range []string{"ila-2.0", "ila-3.0"} {
		label := StrategyLabel(version)
		for _, msg := range []Message{FormatRadarHTFMessage(RadarHTFParams{StrategyVersion: version}), FormatRadarSweepMessage(RadarHTFParams{StrategyVersion: version}), FormatRadarLTFMessage(RadarLTFParams{StrategyVersion: version}), FormatSetupDetailedMessage(SetupMessageParams{StrategyVersion: version})} {
			if !strings.Contains(msg.Text, label) {
				t.Fatalf("strategy missing: %s", msg.Text)
			}
		}
	}
}

func TestMenuLinksDoNotRepeatApplicationPath(t *testing.T) {
	for _, base := range []string{"https://rekavia.com", "https://rekavia.com/app", "https://rekavia.com/app/dashboard", "https://rekavia.com/app/pengaturan"} {
		if got := AppURL(base, "/akun"); got != "https://rekavia.com/app/akun" {
			t.Fatalf("bad application link: %s", got)
		}
	}
}
