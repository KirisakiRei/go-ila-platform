package telegram

import (
	"testing"

	"ila/backend/internal/model"
)

// Kasus USTEC 17 Nov 2026: setup melompat dari tahap 1 ke tahap 3 tanpa kabar
// tahap 2. Pemetaan tahap harus tetap lurus: sweep adalah tahap 1, bukan 2.
func TestTahapTidakMelompat(t *testing.T) {
	kasus := []struct {
		status string
		ingin  Tahap
	}{
		{model.SignalStatusWaitHTFSweep, TahapArah},
		{model.SignalStatusWaitHTFCandleClose, TahapSweep},
		{model.SignalStatusWaitLTFCISD, TahapTrigger},
		{model.SignalStatusReadyToEntry, TahapSiap},
		{"WAIT_ILA3_LTF_CISD_1", TahapTrigger},
	}
	for _, k := range kasus {
		if hasil := TahapDariStatusRadar(k.status); hasil != k.ingin {
			t.Errorf("TahapDariStatusRadar(%q) = %d, ingin %d", k.status, hasil, k.ingin)
		}
	}
}

// Status terminal bukan tahap radar, jadi tidak boleh muncul di daftar pantau.
func TestStatusTerminalBukanTahap(t *testing.T) {
	for _, status := range []string{model.SignalStatusConsumed, model.SignalStatusExpired, ""} {
		if hasil := TahapDariStatusRadar(status); hasil != 0 {
			t.Errorf("TahapDariStatusRadar(%q) = %d, ingin 0", status, hasil)
		}
	}
}

// Tahap harus ditulis "1/6" seperti yang dipakai notifikasi.
func TestFormatTahap(t *testing.T) {
	if got := TahapArah.String(); got != "1/6" {
		t.Fatalf("TahapArah = %q, ingin 1/6", got)
	}
	if got := TahapSelesai.String(); got != "6/6" {
		t.Fatalf("TahapSelesai = %q, ingin 6/6", got)
	}
	if got := Tahap(0).String(); got != "" {
		t.Fatalf("tahap kosong = %q, ingin string kosong", got)
	}
}
