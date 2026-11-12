package settings

import (
	"testing"

	"ila/backend/internal/config"
)

// TestKatalogPunyaNilaiEnv mengunci kelas bug yang pernah terjadi: sebuah flag
// ada di Catalog (dan tampil di panel admin) tetapi pembaca env tidak punya
// cabang untuknya.
//
// Akibatnya nilai mentah kosong, ParseBool gagal, dan pemanggil memakai
// fallback-nya sendiri alih-alih nilai katalog. Gejalanya menyesatkan: mengubah
// variabel lingkungan seolah tidak berpengaruh, dan panel admin menampilkan
// angka yang berbeda dari yang dipakai Runner.
//
// Test ini sengaja menelusuri Catalog, bukan daftar manual, sehingga menambah
// flag baru langsung menuntut cabang di EnvFromConfig.
func TestKatalogPunyaNilaiEnv(t *testing.T) {
	cfg := &config.Config{
		AutoExecute:     true,
		DryRunAll:       true,
		MaxDailyTrades:  3,
		MaxDailyLosses:  2,
		StopAfterWin:    true,
		TimeframeSwitch: true,
		PairSwitch:      true,
	}
	baca := EnvFromConfig(cfg)

	for _, def := range Catalog {
		t.Run(def.Key, func(t *testing.T) {
			if got := baca(def.Key); got == "" {
				t.Fatalf("kunci %q tidak punya nilai env; pemanggil akan memakai fallback dan mengabaikan setelan", def.Key)
			}
		})
	}
}

// TestNilaiEnvMengikutiKonfigurasi memastikan pembaca meneruskan nilai
// konfigurasi apa adanya, termasuk saat saklar dimatikan.
func TestNilaiEnvMengikutiKonfigurasi(t *testing.T) {
	kasing := []struct {
		nama string
		cfg  *config.Config
	}{
		{"semua nyala", &config.Config{AutoExecute: true, DryRunAll: true, TimeframeSwitch: true, PairSwitch: true}},
		{"semua mati", &config.Config{}},
		{"otomatis nyala saja", &config.Config{AutoExecute: true}},
	}

	for _, k := range kasing {
		t.Run(k.nama, func(t *testing.T) {
			baca := EnvFromConfig(k.cfg)

			if got, want := baca(KeyAutoExecute), boolText(k.cfg.AutoExecute); got != want {
				t.Errorf("AUTO_EXECUTE = %q, ingin %q", got, want)
			}
			if got, want := baca(KeyDryRunAll), boolText(k.cfg.DryRunAll); got != want {
				t.Errorf("DRY_RUN_ALL = %q, ingin %q", got, want)
			}
			if got, want := baca(KeyTimeframeSwitch), boolText(k.cfg.TimeframeSwitch); got != want {
				t.Errorf("TIMEFRAME_SWITCH_ENABLED = %q, ingin %q", got, want)
			}
			if got, want := baca(KeyPairSwitch), boolText(k.cfg.PairSwitch); got != want {
				t.Errorf("PAIR_SWITCH_ENABLED = %q, ingin %q", got, want)
			}
		})
	}
}

// TestKunciTakDikenalKosong memastikan kunci liar tidak diam-diam dianggap sah.
func TestKunciTakDikenalKosong(t *testing.T) {
	baca := EnvFromConfig(&config.Config{})
	if got := baca("FLAG_YANG_TIDAK_ADA"); got != "" {
		t.Fatalf("kunci tak dikenal = %q, ingin kosong", got)
	}
}

func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
