package settings

import (
	"strconv"

	"ila/backend/internal/config"
)

// EnvFromConfig mengembalikan pembaca nilai bawaan flag dari konfigurasi proses.
//
// Satu implementasi ini dipakai backend-api dan Runner. Sebelumnya keduanya
// punya salinan sendiri (router.envFlag dan cmd/runner.envFlagReader) yang
// menyimpang: dua saklar yang bawaannya NYALA di katalog tidak punya cabang di
// salinan backend-api, sehingga panel admin menampilkan OFF sementara Runner
// menjalankannya sebagai ON.
//
// Aturan penting yang dikunci di sini: SETIAP kunci di Catalog wajib punya
// nilai di sini. Kunci yang tidak punya cabang menghasilkan string kosong,
// ParseBool gagal, dan pemanggil jatuh ke fallback-nya sendiri — bukan ke
// nilai katalog. Akibatnya variabel lingkungan tampak tidak berpengaruh.
// TestKatalogPunyaNilaiEnv menahan regresi itu.
func EnvFromConfig(cfg *config.Config) func(key string) string {
	return func(key string) string {
		switch key {
		case KeyAutoExecute:
			return strconv.FormatBool(cfg.AutoExecute)
		case KeyDryRunAll:
			return strconv.FormatBool(cfg.DryRunAll)
		case KeyMaxDailyTrades:
			return strconv.Itoa(cfg.MaxDailyTrades)
		case KeyMaxDailyLosses:
			return strconv.Itoa(cfg.MaxDailyLosses)
		case KeyStopAfterWin:
			return strconv.FormatBool(cfg.StopAfterWin)
		// Dua flag ini tidak punya variabel lingkungan: nilainya memang selalu
		// aktif, dicatat eksplisit agar panel menampilkan keadaan sebenarnya.
		case KeyMarketOpenOnly:
			return "true"
		case KeyLegalGate:
			return "true"
		// Saklar yang bawaannya NYALA mengikuti katalog.
		case KeyTimeframeSwitch:
			return strconv.FormatBool(cfg.TimeframeSwitch)
		case KeyPairSwitch:
			return strconv.FormatBool(cfg.PairSwitch)
		// Gate akses: hanya admin yang boleh membuka pendaftaran mandiri.
		case KeyInviteOnly:
			return strconv.FormatBool(cfg.InviteOnly)
		}
		return ""
	}
}
