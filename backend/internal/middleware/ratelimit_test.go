package middleware

import (
	"testing"
	"time"
)

func TestLoginLimiterMengunciSetelahLimaGagal(t *testing.T) {
	limiter := NewLoginLimiter()
	key := "1.2.3.4|user@contoh.id"

	for i := 0; i < loginMaxFailures; i++ {
		if limiter.Blocked(key) {
			t.Fatalf("terkunci terlalu cepat pada percobaan %d", i+1)
		}
		limiter.Fail(key)
	}

	if !limiter.Blocked(key) {
		t.Fatal("seharusnya terkunci setelah 5 kegagalan")
	}
}

func TestLoginLimiterPulihSetelahReset(t *testing.T) {
	limiter := NewLoginLimiter()
	key := "1.2.3.4|user@contoh.id"

	for i := 0; i < loginMaxFailures; i++ {
		limiter.Fail(key)
	}
	if !limiter.Blocked(key) {
		t.Fatal("seharusnya terkunci")
	}

	limiter.Reset(key)
	if limiter.Blocked(key) {
		t.Fatal("setelah login berhasil seharusnya tidak terkunci")
	}
}

func TestLoginLimiterTerbukaSetelahJendela(t *testing.T) {
	limiter := NewLoginLimiter()
	key := "1.2.3.4|user@contoh.id"

	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }

	for i := 0; i < loginMaxFailures; i++ {
		limiter.Fail(key)
	}
	if !limiter.Blocked(key) {
		t.Fatal("seharusnya terkunci")
	}

	// Lewat jendela 15 menit: hitungan diulang dari nol.
	now = now.Add(loginWindow + time.Second)
	if limiter.Blocked(key) {
		t.Fatal("setelah 15 menit seharusnya bisa mencoba lagi")
	}
}

// Kunci berbeda per IP: penyerang tidak bisa mengunci akun orang lain.
func TestLoginLimiterTerpisahPerIP(t *testing.T) {
	limiter := NewLoginLimiter()

	for i := 0; i < loginMaxFailures; i++ {
		limiter.Fail("1.2.3.4|user@contoh.id")
	}

	if limiter.Blocked("5.6.7.8|user@contoh.id") {
		t.Fatal("IP lain seharusnya tidak ikut terkunci")
	}
}
