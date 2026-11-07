package cryptox

import (
	"encoding/base64"
	"strings"
	"testing"
)

// testKey adalah kunci uji 32 byte dalam base64.
var testKey = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

func newTestSealer(t *testing.T) *Sealer {
	t.Helper()

	sealer, err := NewSealer(1, testKey)
	if err != nil {
		t.Fatalf("NewSealer error: %v", err)
	}
	return sealer
}

func TestSealerRoundTrip(t *testing.T) {
	sealer := newTestSealer(t)

	encrypted, err := sealer.SealString("JBSWY3DPEHPK3PXP", "user-1")
	if err != nil {
		t.Fatalf("SealString error: %v", err)
	}
	if encrypted == "JBSWY3DPEHPK3PXP" {
		t.Fatal("ciphertext tidak boleh sama dengan plaintext")
	}

	plaintext, err := sealer.OpenString(encrypted, "user-1")
	if err != nil {
		t.Fatalf("OpenString error: %v", err)
	}
	if plaintext != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("plaintext = %q; ingin nilai asal", plaintext)
	}
}

// Ciphertext satu pemilik tidak bisa dibuka sebagai milik orang lain.
func TestSealerTerikatPemilik(t *testing.T) {
	sealer := newTestSealer(t)

	encrypted, err := sealer.SealString("rahasia", "user-1")
	if err != nil {
		t.Fatalf("SealString error: %v", err)
	}

	if _, err := sealer.OpenString(encrypted, "user-2"); err == nil {
		t.Fatal("pembukaan dengan pemilik lain seharusnya gagal")
	}
}

func TestSealerMenolakCiphertextRusak(t *testing.T) {
	sealer := newTestSealer(t)

	if _, err := sealer.OpenString("bukan-base64!!", "user-1"); err == nil {
		t.Fatal("ciphertext bukan base64 seharusnya ditolak")
	}
	if _, err := sealer.OpenString("AAAA", "user-1"); err == nil {
		t.Fatal("ciphertext terlalu pendek seharusnya ditolak")
	}
}

// Dua enkripsi nilai sama menghasilkan ciphertext berbeda (nonce acak).
func TestSealerNonceAcak(t *testing.T) {
	sealer := newTestSealer(t)

	first, err := sealer.SealString("sama", "user-1")
	if err != nil {
		t.Fatalf("SealString error: %v", err)
	}
	second, err := sealer.SealString("sama", "user-1")
	if err != nil {
		t.Fatalf("SealString error: %v", err)
	}

	if first == second {
		t.Fatal("ciphertext dua enkripsi nilai sama tidak boleh identik")
	}
}

func TestSealerMenolakKunciSalah(t *testing.T) {
	if _, err := NewSealer(1, "pendek"); err == nil {
		t.Fatal("kunci bukan 32 byte seharusnya ditolak")
	}
	if _, err := NewSealer(1, "!!!bukan-base64!!!"); err == nil {
		t.Fatal("kunci bukan base64 seharusnya ditolak")
	}
}

func TestSealerVersion(t *testing.T) {
	sealer, err := NewSealer(7, testKey)
	if err != nil {
		t.Fatalf("NewSealer error: %v", err)
	}
	if sealer.Version() != 7 {
		t.Fatalf("version = %d; ingin 7", sealer.Version())
	}
}

func TestSealerTanpaPanjangTersembunyi(t *testing.T) {
	sealer := newTestSealer(t)

	short, _ := sealer.SealString("a", "user-1")
	long, _ := sealer.SealString(strings.Repeat("a", 100), "user-1")

	// Panjang ciphertext memang mengikuti panjang plaintext, sama seperti AES-GCM
	// pada umumnya; yang penting isinya tidak terbaca.
	if len(long) <= len(short) {
		t.Fatal("ciphertext nilai lebih panjang seharusnya lebih besar")
	}
}
