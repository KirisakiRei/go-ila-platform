package main

import (
	"testing"

	"ila/backend/internal/model"
)

func TestStatusTerminal(t *testing.T) {
	terminal := []string{
		model.SignalStatusConsumed, model.SignalStatusCancelled,
		model.SignalStatusExpired, model.SignalStatusSuperseded,
		model.SignalStatusRejected, model.SignalStatusSkipped,
	}
	for _, status := range terminal {
		if !statusTerminal(status) {
			t.Fatalf("%s harus dianggap status akhir", status)
		}
	}

	// Status menengah masih bisa menjadi entry, tombolnya tidak boleh dilepas.
	for _, status := range []string{
		model.SignalStatusReadyToEntry,
		model.SignalStatusWaitLTFCISD,
		model.SignalStatusWaitHTFCandleClose,
	} {
		if statusTerminal(status) {
			t.Fatalf("%s bukan status akhir", status)
		}
	}
}

func TestAlasanTidakBerlaku(t *testing.T) {
	if got := alasanTidakBerlaku(model.SignalStatusRejected, "Harga sudah lewat."); got != "Harga sudah lewat." {
		t.Fatalf("alasan dari Runner harus dipakai apa adanya, dapat %q", got)
	}
	if got := alasanTidakBerlaku(model.SignalStatusExpired, ""); got == "" {
		t.Fatal("status akhir harus punya kalimat penjelas")
	}
	if got := alasanTidakBerlaku("STATUS_ASING", ""); got != "" {
		t.Fatalf("status tak dikenal tidak perlu kalimat, dapat %q", got)
	}
}
