package main

import (
	"testing"

	"ila/backend/internal/config"
	"ila/backend/internal/settings"
)

// Pengujian ini mengunci satu bug yang pernah terjadi: saklar timeframe
// dinyalakan lewat variabel lingkungan, tetapi Runner tetap menyaring seolah
// saklarnya mati.
//
// Sebabnya envFlagReader tidak punya cabang untuk kunci itu, sehingga nilainya
// kosong, ParseBool gagal, dan pemanggil jatuh ke fallback `false` yang dikirim
// pemanggil — bukan default katalog yang `true`.

func TestEnvFlagReaderMengembalikanNilaiSaklar(t *testing.T) {
	kasing := []struct {
		nama string
		cfg  *config.Config
	}{
		{"both nyala", &config.Config{TimeframeSwitch: true, PairSwitch: true}},
		{"both mati", &config.Config{TimeframeSwitch: false, PairSwitch: false}},
		{"timeframe nyala saja", &config.Config{TimeframeSwitch: true, PairSwitch: false}},
	}

	for _, k := range kasing {
		t.Run(k.nama, func(t *testing.T) {
			baca := envFlagReader(k.cfg)

			got := baca(settings.KeyTimeframeSwitch)
			if got == "" {
				t.Fatal("nilai timeframe switch kosong; pemanggil akan memakai fallback dan mengabaikan setelan")
			}
			if got != boolText(k.cfg.TimeframeSwitch) {
				t.Errorf("timeframe switch = %q, ingin %q", got, boolText(k.cfg.TimeframeSwitch))
			}

			gotPair := baca(settings.KeyPairSwitch)
			if gotPair == "" {
				t.Fatal("nilai pair switch kosong")
			}
			if gotPair != boolText(k.cfg.PairSwitch) {
				t.Errorf("pair switch = %q, ingin %q", gotPair, boolText(k.cfg.PairSwitch))
			}
		})
	}
}

// Setiap kunci saklar wajib punya padanan di pembaca env. Bila tidak, mengubah
// nilainya lewat variabel lingkungan tidak akan pernah berpengaruh.
func TestKunciSaklarDapatDibacaDariEnv(t *testing.T) {
	baca := envFlagReader(&config.Config{TimeframeSwitch: true, PairSwitch: true})

	// Daftar ini sengaja ditulis eksplisit: menambah kunci baru di sini memaksa
	// penulis sadar bahwa pembaca env juga harus diperbarui.
	wajib := []string{settings.KeyTimeframeSwitch, settings.KeyPairSwitch}
	for _, key := range wajib {
		if baca(key) == "" {
			t.Errorf("kunci %q tidak terbaca dari env; setelan akan diabaikan", key)
		}
	}
}

func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
