package utils

import (
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
