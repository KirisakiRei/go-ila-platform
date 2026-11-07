package utils

import (
	"testing"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"
)

func TestNormalizeCapsPerPage(t *testing.T) {
	q := dto.PaginationQuery{Page: 0, PerPage: 500, Order: "aneh"}
	Normalize(&q)

	if q.Page != 1 {
		t.Errorf("page default harus 1, dapat %d", q.Page)
	}
	if q.PerPage != 100 {
		t.Errorf("per_page harus dijepit ke 100, dapat %d", q.PerPage)
	}
	if q.Order != "desc" {
		t.Errorf("order default harus desc, dapat %q", q.Order)
	}
}

func TestOffset(t *testing.T) {
	if got := Offset(3, 20); got != 40 {
		t.Errorf("Offset(3,20) = %d, want 40", got)
	}
}

func TestBuildMeta(t *testing.T) {
	meta := BuildMeta(2, 10, 25)
	if meta.TotalPages != 3 {
		t.Errorf("total_pages = %d, want 3", meta.TotalPages)
	}
	if meta.TotalItems != 25 {
		t.Errorf("total_items = %d, want 25", meta.TotalItems)
	}
}

func TestSafeSortColumnRejectsUnknown(t *testing.T) {
	allowed := map[string]string{"email": "email", "created": "created_at"}

	if got := SafeSortColumn("created", allowed, "id"); got != "created_at" {
		t.Errorf("kolom sah harus dipetakan, dapat %q", got)
	}
	if got := SafeSortColumn("password_hash", allowed, "id"); got != "id" {
		t.Errorf("kolom tak dikenal harus jatuh ke fallback, dapat %q", got)
	}
}

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
