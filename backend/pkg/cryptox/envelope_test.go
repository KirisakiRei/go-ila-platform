package cryptox

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func randomKey(t *testing.T) string {
	t.Helper()
	// Kunci statis 32 byte untuk pengujian; bukan kunci produksi.
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, KeySize))
}

func newTestKeyring(t *testing.T) *Keyring {
	t.Helper()
	ring, err := NewKeyring(1, randomKey(t), nil)
	if err != nil {
		t.Fatalf("membuat keyring: %v", err)
	}
	return ring
}

func TestSealAndOpenRoundTrip(t *testing.T) {
	ring := newTestKeyring(t)
	plaintext := []byte(`{"platform":"matchtrader","token":"rahasia-sekali"}`)
	aad := []byte("account-123")

	env, err := ring.Seal(plaintext, aad)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	got, err := ring.Open(env, aad)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("plaintext tidak sama setelah dekripsi")
	}
}

// Ciphertext tidak boleh memuat plaintext apa pun.
func TestSealProducesCiphertext(t *testing.T) {
	ring := newTestKeyring(t)
	plaintext := []byte("token-rahasia")

	env, err := ring.Seal(plaintext, nil)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Contains(env.Ciphertext, plaintext) {
		t.Fatal("ciphertext memuat plaintext")
	}
}

// Envelope terikat ke akun: blob yang dipindah ke akun lain harus gagal.
func TestOpenWithDifferentAssociatedData(t *testing.T) {
	ring := newTestKeyring(t)

	env, err := ring.Seal([]byte("rahasia"), []byte("account-1"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if _, err := ring.Open(env, []byte("account-2")); err == nil {
		t.Fatal("membuka dengan associated data berbeda seharusnya gagal")
	}
}

// Setiap enkripsi memakai nonce dan kunci user yang baru.
func TestSealUsesFreshKeys(t *testing.T) {
	ring := newTestKeyring(t)

	first, err := ring.Seal([]byte("sama"), nil)
	if err != nil {
		t.Fatalf("Seal pertama: %v", err)
	}
	second, err := ring.Seal([]byte("sama"), nil)
	if err != nil {
		t.Fatalf("Seal kedua: %v", err)
	}

	if bytes.Equal(first.Ciphertext, second.Ciphertext) {
		t.Fatal("ciphertext identik untuk plaintext sama")
	}
	if bytes.Equal(first.UserKeyWrapped, second.UserKeyWrapped) {
		t.Fatal("kunci user dipakai ulang")
	}
}

// Kunci lama tetap bisa membuka data setelah rotasi kunci.
func TestLegacyKeyStillReadable(t *testing.T) {
	oldRing, err := NewKeyring(1, randomKey(t), nil)
	if err != nil {
		t.Fatalf("keyring lama: %v", err)
	}

	env, err := oldRing.Seal([]byte("data lama"), nil)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	newKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x7f}, KeySize))
	rotated, err := NewKeyring(2, newKey, map[int]string{1: randomKey(t)})
	if err != nil {
		t.Fatalf("keyring hasil rotasi: %v", err)
	}

	got, err := rotated.Open(env, nil)
	if err != nil {
		t.Fatalf("Open dengan kunci lama: %v", err)
	}
	if string(got) != "data lama" {
		t.Fatalf("isi = %q", got)
	}
}

func TestRejectsWrongKeySize(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte("terlalu-pendek"))
	if _, err := NewKeyring(1, short, nil); err == nil {
		t.Fatal("kunci pendek seharusnya ditolak")
	}
}

func TestOpenRejectsUnknownVersion(t *testing.T) {
	ring := newTestKeyring(t)

	env, err := ring.Seal([]byte("x"), nil)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	env.KeyVersion = 99

	if _, err := ring.Open(env, nil); err == nil {
		t.Fatal("versi kunci tak dikenal seharusnya ditolak")
	}
}
