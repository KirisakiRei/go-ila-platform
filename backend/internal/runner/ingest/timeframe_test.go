package ingest

import (
	"testing"

	enginev1 "ila/backend/gen/engine/v1"
)

// TestNormalizeTimeframe memastikan penulisan timeframe dari engine yang
// berbeda-beda tetap dibakukan, karena kontraknya belum ditetapkan.
func TestNormalizeTimeframe(t *testing.T) {
	cases := map[string]string{
		"M15":    "M15",
		"m15":    "M15",
		"15m":    "15M",
		" 15M ":  "15M",
		"H4":     "H4",
		"h4":     "H4",
		"W1":     "W1",
		"1W":     "1W",
		"D1":     "D1",
		"H4-M15": "H4M15",
		"":       "",
		"   ":    "",
		"H1":     "H1",
		"M5":     "M5",
		"MN1":    "MN1",
	}
	for input, want := range cases {
		if got := NormalizeTimeframe(input); got != want {
			t.Errorf("NormalizeTimeframe(%q) = %q, ingin %q", input, got, want)
		}
	}
}

// TestConvertSignalMenyimpanTimeframe memastikan htf/ltf benar-benar
// diteruskan ke entitas penyimpanan — sebelumnya nilai ini dibuang.
func TestConvertSignalMenyimpanTimeframe(t *testing.T) {
	in := validSignal()
	in.Htf = "H4"
	in.Ltf = "m15"

	signal, ok := ConvertSignal(in)
	if !ok {
		t.Fatal("sinyal lengkap seharusnya diterima")
	}
	if signal.Htf != "H4" {
		t.Errorf("Htf = %q, ingin H4", signal.Htf)
	}
	if signal.Ltf != "M15" {
		t.Errorf("Ltf = %q, ingin M15 (dibakukan)", signal.Ltf)
	}
}

// TestConvertSignalTanpaTimeframeTetapSah memastikan sinyal dari engine yang
// belum mengirim timeframe tidak ditolak: kolomnya dibiarkan kosong, bukan
// diisi nilai karangan.
func TestConvertSignalTanpaTimeframeTetapSah(t *testing.T) {
	signal, ok := ConvertSignal(kosongkanTimeframe(validSignal()))
	if !ok {
		t.Fatal("sinyal tanpa timeframe tetap harus diterima")
	}
	if signal.Htf != "" || signal.Ltf != "" {
		t.Fatalf("timeframe kosong diharapkan, dapat Htf=%q Ltf=%q", signal.Htf, signal.Ltf)
	}
}

// kosongkanTimeframe meniru engine versi lama yang belum mengirim htf/ltf.
func kosongkanTimeframe(in *enginev1.Signal) *enginev1.Signal {
	in.Htf = ""
	in.Ltf = ""
	return in
}
