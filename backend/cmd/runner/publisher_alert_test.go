package main

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"

	"ila/backend/internal/runner"
	"ila/backend/internal/settings"
	"ila/backend/internal/telegram"
)

// flagUji adalah sumbu kuota harian yang bisa diatur tes.
type flagUji map[string]int

func (f flagUji) Bool(_ context.Context, _ string, fallback bool) bool { return fallback }
func (f flagUji) Int(_ context.Context, key string, fallback int) int {
	if v, ada := f[key]; ada {
		return v
	}
	return fallback
}

func publisherUji(flags runner.Flags) *runnerPublisher {
	return &runnerPublisher{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		flags:  flags,
	}
}

// TestKuotaAngkaIkutPengaturan memastikan batas kuota di notifikasi memakai
// angka yang berlaku, bukan nilai bawaan. Notifikasi yang menyebut angka salah
// membuat user mengira masih ada sisa trade padahal tidak.
func TestKuotaAngkaIkutPengaturan(t *testing.T) {
	p := publisherUji(flagUji{settings.KeyMaxDailyTrades: 3})

	if got := p.kuotaAngka(context.Background(), "u-1", runner.RejectDailyQuota, true); got != 3 {
		t.Fatalf("batas kuota = %d, ingin 3 dari pengaturan", got)
	}
}

// TestKuotaAngkaDijepitPagar memastikan angka dari pengaturan tetap dijepit
// pada batas aman, sama seperti yang dilakukan Runner saat menahan setup.
func TestKuotaAngkaDijepitPagar(t *testing.T) {
	p := publisherUji(flagUji{settings.KeyMaxDailyTrades: 99})

	if got := p.kuotaAngka(context.Background(), "u-1", runner.RejectDailyQuota, true); got != 3 {
		t.Fatalf("batas kuota = %d, ingin dijepit ke 3", got)
	}
}

// TestKuotaAngkaHanyaUntukGrupKuota memastikan alasan lain tidak memicu
// pembacaan kuota yang tidak relevan.
func TestKuotaAngkaHanyaUntukGrupKuota(t *testing.T) {
	p := publisherUji(flagUji{settings.KeyMaxDailyTrades: 3})

	if got := p.kuotaAngka(context.Background(), "u-1", runner.RejectSlippage, true); got != 0 {
		t.Fatalf("alasan non-kuota tidak boleh mengisi angka kuota, dapat %d", got)
	}
}

// TestFormatClosedMenampilkanTimeframe memastikan pesan posisi ditutup
// menyebut pasangan timeframe dari sinyal asalnya.
func TestFormatClosedMenampilkanTimeframe(t *testing.T) {
	msg := telegram.FormatClosedMessage(telegram.ClosedParams{
		Symbol: "XAUUSD", Direction: "SELL", Htf: "H4", Ltf: "M15",
		Result: "Target tercapai", EntryPrice: 3880.24, ExitPrice: 3852.64,
		ProfitLoss: 143.5, OutcomeR: 2, Duration: "3 jam 12 menit",
	})

	for _, ingin := range []string{"XAUUSD · H4 ➔ M15", "Tahap 6/6", "2,00R", "$143,50"} {
		if !strings.Contains(msg.Text, ingin) {
			t.Errorf("pesan closed tidak memuat %q:\n%s", ingin, msg.Text)
		}
	}
}
