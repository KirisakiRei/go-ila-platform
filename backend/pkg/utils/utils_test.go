package utils

import (
	"sync"
	"testing"
	"time"
)

func TestJWTIssueAndVerify(t *testing.T) {
	issuer := NewJWTIssuer("rahasia-uji-yang-cukup-panjang-untuk-hs256", time.Hour)

	token, expiresAt, err := issuer.Issue("user-1", "user")
	if err != nil {
		t.Fatalf("issue gagal: %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Error("masa berlaku harus di masa depan")
	}

	claims, err := issuer.Verify(token)
	if err != nil {
		t.Fatalf("verify gagal: %v", err)
	}
	if claims.UserID != "user-1" || claims.Role != "user" {
		t.Errorf("claims tidak sesuai: %+v", claims)
	}
}

func TestJWTRejectsForeignSignature(t *testing.T) {
	issuer := NewJWTIssuer("rahasia-uji-yang-cukup-panjang-untuk-hs256", time.Hour)
	other := NewJWTIssuer("rahasia-lain-yang-juga-cukup-panjang-1234", time.Hour)

	token, _, err := other.Issue("user-1", "user")
	if err != nil {
		t.Fatalf("issue gagal: %v", err)
	}

	if _, err := issuer.Verify(token); err == nil {
		t.Fatal("token dengan tanda tangan asing harus ditolak")
	}
}

func TestArgon2idSemaphoreAndMalformedPHCSafe(t *testing.T) {
	// Parameter PHC rusak (threads=0 atau memory berlebih) tidak boleh panic.
	badHashes := []string{
		"$argon2id$v=19$m=65536,t=3,p=0$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"$argon2id$v=19$m=9999999,t=3,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"$argon2id$v=19$m=65536,t=0,p=4$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	}
	for _, bad := range badHashes {
		if ComparePassword(bad, "rahasia12345") {
			t.Fatalf("hash rusak %q seharusnya ditolak", bad)
		}
	}

	// Banyak login paralel harus tetap berjalan rapi melewati semaphore.
	hash, err := HashPassword("sandi-uji-aman-123")
	if err != nil {
		t.Fatalf("HashPassword gagal: %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !ComparePassword(hash, "sandi-uji-aman-123") {
				t.Errorf("ComparePassword seharusnya cocok")
			}
		}()
	}
	wg.Wait()
}
