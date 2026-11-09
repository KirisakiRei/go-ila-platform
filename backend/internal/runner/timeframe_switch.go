package runner

import (
	"context"

	"ila/backend/internal/repository"
	"ila/backend/internal/settings"
)

// TimeframeAction adalah keputusan Runner untuk satu sinyal pada satu akun.
type TimeframeAction string

const (
	// TimeframeIkuti berarti saklar timeframe tidak membatasi apa pun: sinyal
	// diproses seperti biasa.
	TimeframeIkuti TimeframeAction = ""
	// TimeframePeluang berarti sinyal TIDAK dieksekusi otomatis, tetapi tetap
	// dicatat sebagai peluang agar user bisa memutuskan.
	TimeframePeluang TimeframeAction = "peluang"
	// TimeframeAbaikan berarti pasangan ini tidak dipantau akun tersebut, jadi
	// sinyalnya tidak relevan sama sekali untuk akun itu.
	TimeframeAbaikan TimeframeAction = "abaikan"
)

// timeframeSwitchActive menandai apakah pemilihan timeframe per akun berlaku.
//
// Bawaan mati: engine belum mengirim htf/ltf, sehingga semua sinyal akan
// dianggap "tidak diketahui" dan penyaringan hanya akan menyesatkan. Flag
// dinyalakan admin setelah data timeframe benar-benar ada.
func (s *Service) timeframeSwitchActive(ctx context.Context) bool {
	if s.opts.Flags == nil {
		return false
	}
	return s.opts.Flags.Bool(ctx, settings.KeyTimeframeSwitch, false)
}

// putuskanTimeframe menentukan apakah sinyal ini boleh dieksekusi otomatis,
// hanya boleh ditampilkan sebagai peluang, atau tidak relevan bagi akun.
//
// Aturannya sengaja sederhana dan dapat dijelaskan ke user:
//   - Sinyal tanpa htf/ltf (engine lama) tidak pernah disaring: diproses biasa.
//   - Akun yang belum punya baris saklar untuk metode ini juga diproses biasa,
//     supaya penambahan fitur tidak tiba-tiba mematikan akun yang sudah jalan.
//   - Pasangan yang dipantau + auto  -> diproses biasa.
//   - Pasangan yang dipantau, tanpa auto -> peluang (tampil, tanpa eksekusi).
//   - Pasangan yang tidak dipantau -> diabaikan.
//
// Kegagalan membaca saklar tidak boleh mematikan trading, jadi dianggap
// "ikuti" (perilaku lama).
func (s *Service) putuskanTimeframe(ctx context.Context, accountID, symbol, strategy, htf, ltf string) TimeframeAction {
	if !s.timeframeSwitchActive(ctx) {
		return TimeframeIkuti
	}
	// Tanpa data timeframe dari engine, tidak ada dasar menyaring.
	if htf == "" || ltf == "" {
		return TimeframeIkuti
	}

	switches, err := s.store.TimeframeSwitches(ctx, symbol)
	if err != nil {
		s.logger.Warn("gagal membaca saklar timeframe; sinyal diproses seperti biasa",
			"account_id", accountID, "symbol", symbol, "error", err.Error())
		return TimeframeIkuti
	}

	perMetode, ada := switches[accountID]
	if !ada {
		// Belum pernah diatur: jangan ubah perilaku akun ini.
		return TimeframeIkuti
	}
	pasangan, ada := perMetode[strategy]
	if !ada {
		// Metode ini belum punya saklar timeframe untuk akun ini.
		return TimeframeIkuti
	}

	state, ada := pasangan[repository.TimeframeKey{Htf: htf, Ltf: ltf}]
	if !ada {
		// Akun sudah mengatur metode ini, tetapi pasangan ini tidak termasuk
		// yang dipantau. Sinyalnya bukan untuk akun ini.
		return TimeframeAbaikan
	}
	if !state.Watch {
		return TimeframeAbaikan
	}
	if !state.Auto {
		return TimeframePeluang
	}
	return TimeframeIkuti
}
