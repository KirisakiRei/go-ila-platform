package runner

import (
	"context"

	"ila/backend/internal/settings"
)

// pairSwitchActive menandai apakah saklar metode per akun × pair berlaku.
//
// Bawaan menyala: saklar per akun adalah perilaku yang diinginkan produk.
// Flag tetap disediakan sebagai rem darurat — mematikannya mengembalikan Runner
// ke perilaku lama (kolom langganan menentukan metode) tanpa deploy ulang.
func (s *Service) pairSwitchActive(ctx context.Context) bool {
	if s.opts.Flags == nil {
		return true
	}
	return s.opts.Flags.Bool(ctx, settings.KeyPairSwitch, true)
}

// akunMengizinkanMetode memeriksa saklar satu akun untuk satu metode pada pair
// yang sedang dievaluasi.
//
// Aturan: akun yang belum punya baris saklar untuk pair ini diperlakukan
// sebagai bawaan (ILA 2.0 menyala, ILA 3.0 mengikuti hak tier yang sudah
// diperiksa sebelumnya). Kegagalan membaca saklar tidak boleh mematikan
// trading, jadi dianggap izinkan — kecuali memang ada baris yang mematikan.
func (s *Service) akunMengizinkanMetode(ctx context.Context, accountID, symbol, strategy string) bool {
	if !s.pairSwitchActive(ctx) {
		return true
	}

	saklar, err := s.store.EnabledAccountStrategies(ctx, symbol)
	if err != nil {
		s.logger.Warn("gagal membaca saklar metode akun; metode dianggap menyala",
			"account_id", accountID, "symbol", symbol, "error", err.Error())
		return true
	}

	diatur, err := s.store.ConfiguredAccounts(ctx, symbol)
	if err != nil {
		s.logger.Warn("gagal membaca daftar akun dengan saklar metode; metode dianggap menyala",
			"account_id", accountID, "symbol", symbol, "error", err.Error())
		return true
	}

	if !diatur[accountID] {
		// Belum pernah diatur: perlakukan seperti bawaan.
		return strategy == "ila-2.0"
	}
	return saklar[accountID][strategy]
}
