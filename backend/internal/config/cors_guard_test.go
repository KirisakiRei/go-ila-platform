package config_test

import (
	"strings"
	"testing"

	"ila/backend/internal/config"
)

// baseEnv mengisi konfigurasi produksi minimal yang sah.
func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_DSN", "postgres://user:pass@db:5432/ila")
	t.Setenv("JWT_SECRET", strings.Repeat("a", 40))
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("COOKIE_SECURE", "true")
	t.Setenv("MASTER_KEY", strings.Repeat("b", 32))
	t.Setenv("SESSION_COOKIE_NAME", "ila_session")
	t.Setenv("CSRF_COOKIE_NAME", "ila_csrf")
}

// Di produksi, daftar asal wajib diisi. Default localhost membuat frontend
// sendiri ditolak CORS tanpa pesan jelas (plans/21 D3).
func TestProduksiTanpaAllowedOriginsDitolak(t *testing.T) {
	baseEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "")

	if _, err := config.Load("backend-api"); err == nil {
		t.Fatal("produksi tanpa ALLOWED_ORIGINS seharusnya ditolak")
	}
}

// Wildcard tidak boleh dipakai karena cookie sesi butuh asal eksplisit.
func TestProduksiWildcardDitolak(t *testing.T) {
	baseEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "*")

	if _, err := config.Load("backend-api"); err == nil {
		t.Fatal("ALLOWED_ORIGINS=* seharusnya ditolak di produksi")
	}
}

// Alamat lokal tidak boleh ikut di produksi.
func TestProduksiLocalhostDitolak(t *testing.T) {
	baseEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:3000")

	if _, err := config.Load("backend-api"); err == nil {
		t.Fatal("localhost di produksi seharusnya ditolak")
	}
}

// Konfigurasi produksi yang benar harus lolos.
func TestProduksiLengkapDiterima(t *testing.T) {
	baseEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "https://app.ilabot.id")

	if _, err := config.Load("backend-api"); err != nil {
		t.Fatalf("konfigurasi sah ditolak: %v", err)
	}
}
