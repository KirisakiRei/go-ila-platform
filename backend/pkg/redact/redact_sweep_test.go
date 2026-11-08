package redact_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ila/backend/pkg/redact"
)

// TestTidakAdaVerbatimRahasiaDiLog menyapu sumber kode: setiap pemanggilan
// logger tidak boleh mencetak nilai rahasia secara langsung. Nilai rahasia
// wajib lewat redact.Secrets lebih dulu atau tidak dicetak sama sekali.
//
// Uji ini bukan pengganti tinjauan kode, tetapi menangkap kesalahan yang
// paling sering terjadi saat menambah log baru (plans/18 §6).
func TestTidakAdaVerbatimRahasiaDiLog(t *testing.T) {
	root := filepath.Join("..", "..")

	// Pola pencetakan langsung nilai sensitif pada log.
	forbidden := []string{
		`"password", user.PasswordHash`,
		`"token", req.Token`,
		`"secret", setup.Secret`,
		`"master_key", cfg.MasterKey`,
		`"smtp_pass", cfg.SMTPPass`,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			// Direktori uji dan aset tidak berisi log produksi.
			if info.Name() == "node_modules" || info.Name() == ".git" || info.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		body := string(raw)
		for _, pattern := range forbidden {
			if strings.Contains(body, pattern) {
				t.Errorf("%s mencetak rahasia langsung: %s", path, pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("menyapu sumber: %v", err)
	}
}

// TestPenyaringMenutupMasterKey memastikan kunci yang dipakai konfigurasi
// platform ikut tersaring, bukan hanya token JWT.
func TestPenyaringMenutupMasterKey(t *testing.T) {
	cases := []string{
		"MASTER_KEY=MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA=",
		`{"smtp_pass":"rahasia-sekali"}`,
		"api_key=abcdefghijklmnop",
		"client_secret: zzzz-1111-2222",
	}

	for _, input := range cases {
		out := redact.Secrets(input)
		if strings.Contains(out, "MDAwMDAw") || strings.Contains(out, "rahasia-sekali") ||
			strings.Contains(out, "abcdefghijklmnop") || strings.Contains(out, "zzzz-1111-2222") {
			t.Fatalf("rahasia lolos penyaring: %q", out)
		}
	}
}
