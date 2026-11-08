package runner

import (
	"context"
	"strings"

	"ila/backend/internal/settings"
)

// Flags adalah pengaturan platform yang bisa diubah admin tanpa deploy ulang.
// Nilainya dibaca ulang setiap evaluasi sinyal, sehingga rem darurat berlaku
// dalam hitungan detik (plans/11 Fase I).
type Flags interface {
	Bool(ctx context.Context, key string, fallback bool) bool
	Int(ctx context.Context, key string, fallback int) int
}

// Mode menandai tingkat layanan platform yang berlaku saat ini.
type Mode string

const (
	// ModeNormal: seluruh alur berjalan seperti biasa.
	ModeNormal Mode = "normal"
	// ModeReadOnly: sinyal baru dilewati, posisi berjalan tetap dipantau.
	ModeReadOnly Mode = "read-only"
	// ModeMaintenance: sama seperti read-only; dibedakan untuk komunikasi.
	ModeMaintenance Mode = "maintenance"
)

// ModedState adalah baris mode platform yang dibutuhkan Runner.
type ModedState struct {
	Mode string
}

// ModeSource membaca mode platform. Bila nil, mode dianggap normal.
type ModeSource interface {
	State(ctx context.Context) (*ModedState, error)
}

// flagsOrEnv memakai flag bila tersedia; selain itu nilai env dari Options.
func (s *Service) autoExecute(ctx context.Context) bool {
	if s.opts.Flags == nil {
		return s.opts.AutoExecute
	}
	return s.opts.Flags.Bool(ctx, settings.KeyAutoExecute, s.opts.AutoExecute)
}

// dryRunForced menandai simulasi yang dipaksa seluruh platform.
func (s *Service) dryRunForced(ctx context.Context) bool {
	if s.opts.Flags == nil {
		return s.opts.DryRunAll
	}
	return s.opts.Flags.Bool(ctx, settings.KeyDryRunAll, s.opts.DryRunAll)
}

// marketOpenRequired menandai pemeriksaan jam pasar.
func (s *Service) marketOpenRequired(ctx context.Context) bool {
	if s.opts.Flags == nil {
		return s.opts.MarketOpenCheck
	}
	return s.opts.Flags.Bool(ctx, settings.KeyMarketOpenOnly, s.opts.MarketOpenCheck)
}

// legalGateActive menandai penahanan karena dokumen wajib.
func (s *Service) legalGateActive(ctx context.Context) bool {
	if s.opts.Flags == nil {
		return true
	}
	return s.opts.Flags.Bool(ctx, settings.KeyLegalGate, true)
}

// policy membaca pagar aturan harian yang berlaku saat ini.
func (s *Service) policy(ctx context.Context) DailyPolicy {
	current := s.opts.Policy
	if s.opts.Flags == nil {
		return current
	}
	return DailyPolicy{
		MaxTrades:    s.opts.Flags.Int(ctx, settings.KeyMaxDailyTrades, current.MaxTrades),
		StopAfterWin: s.opts.Flags.Bool(ctx, settings.KeyStopAfterWin, current.StopAfterWin),
		MaxLosses:    s.opts.Flags.Int(ctx, settings.KeyMaxDailyLosses, current.MaxLosses),
	}.Clamp()
}

// maintenanceMode membaca mode platform dari database. Kegagalan membaca
// dianggap normal agar masalah database tidak menghentikan eksekusi yang sah.
func (s *Service) maintenanceMode(ctx context.Context) Mode {
	if s.opts.Mode == nil {
		return ModeNormal
	}
	state, err := s.opts.Mode.State(ctx)
	if err != nil || state == nil {
		return ModeNormal
	}
	switch strings.TrimSpace(state.Mode) {
	case string(ModeReadOnly):
		return ModeReadOnly
	case string(ModeMaintenance):
		return ModeMaintenance
	default:
		return ModeNormal
	}
}
