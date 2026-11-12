package settings

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/config"
)

// storeTetap adalah OverrideStore yang selalu menjawab sama.
type storeTetap struct {
	nilai map[string]string
	err   error
}

func (s storeTetap) Load(context.Context) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.nilai, nil
}

func (s storeTetap) Save(context.Context, string, string, string, string) error { return nil }
func (s storeTetap) Delete(context.Context, string) error                       { return nil }

// TestInviteOnlyMembacaNilaiEnv memastikan jalur env benar-benar dipakai.
//
// Fallback sengaja diisi kebalikan dari nilai env: bila cabang EnvFromConfig
// hilang, pembacaan jatuh ke fallback dan tes ini gagal. Tanpa trik itu tes
// tetap hijau meski cabangnya lenyap — masalah yang pernah lolos di sini.
func TestInviteOnlyMembacaNilaiEnv(t *testing.T) {
	kasing := []struct {
		nama   string
		env    bool
		fallbk bool
	}{

		{"env nyala walau fallback mati", true, false},
		{"env mati walau fallback nyala", false, true},
	}

	for _, k := range kasing {
		t.Run(k.nama, func(t *testing.T) {
			cfg := &config.Config{InviteOnly: k.env}
			svc := NewService(storeTetap{nilai: map[string]string{}}, EnvFromConfig(cfg))

			if got := svc.Bool(context.Background(), KeyInviteOnly, k.fallbk); got != k.env {
				t.Fatalf("INVITE_ONLY = %v, ingin %v (nilai env diabaikan, fallback yang terpakai)", got, k.env)
			}
		})
	}
}

// TestInviteOnlyOverrideDatabaseMenang: admin membuka pendaftaran lewat panel,
// dan override database harus menang atas env.
func TestInviteOnlyOverrideDatabaseMenang(t *testing.T) {
	cfg := &config.Config{InviteOnly: true}
	svc := NewService(storeTetap{nilai: map[string]string{KeyInviteOnly: "false"}}, EnvFromConfig(cfg))

	if svc.Bool(context.Background(), KeyInviteOnly, true) {
		t.Fatal("override database false diabaikan; pendaftaran tetap tertutup")
	}
}

// TestInviteOnlyGagalBacaMemakaiFallbackTertutup mengunci perilaku fail-closed:
// database tidak terbaca bukan alasan membuka pendaftaran.
func TestInviteOnlyGagalBacaMemakaiFallbackTertutup(t *testing.T) {
	svc := NewService(storeTetap{err: errors.New("database mati")}, func(string) string { return "" })
	svc.SetTTL(time.Nanosecond)

	if !svc.Bool(context.Background(), KeyInviteOnly, true) {
		t.Fatal("gagal membaca flag seharusnya jatuh ke fallback tertutup")
	}
}

// TestInviteOnlyBawaanKonfigurasiTertutup memastikan Load tanpa env tetap
// tertutup: nilai bawaan di config.go adalah false-nya satu-satunya jalan
// membuka. Diuji lewat Load agar cabang default ikut terkunci.
func TestInviteOnlyBawaanKonfigurasiTertutup(t *testing.T) {
	t.Setenv("INVITE_ONLY", "")
	t.Setenv("DB_DSN", "postgres://uji:uji@127.0.0.1/uji")
	t.Setenv("JWT_SECRET", "rahasia-panjang-untuk-uji-1234567890")

	cfg, err := config.Load("uji")
	if err != nil {
		t.Skipf("konfigurasi uji tidak lengkap: %v", err)
	}
	if !cfg.InviteOnly {
		t.Fatal("INVITE_ONLY kosong seharusnya menghasilkan tertutup (fail-closed)")
	}
}
