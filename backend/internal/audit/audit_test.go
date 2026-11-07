package audit

import (
	"strings"
	"testing"
)

func TestScrubRemovesSecrets(t *testing.T) {
	state := map[string]any{
		"email":              "a@contoh.id",
		"password_hash":      "$2a$10$rahasia",
		"mt_token":           "eyJhbGciOi...",
		"api_secret":         "s3cr3t",
		"credential_blob":    []byte{1, 2, 3},
		"recovery_codes":     []string{"a", "b"},
		"device_fingerprint": "abc",
		"tier":               "plus",
	}
	payload := Scrub(state)

	for _, leaked := range []string{"rahasia", "eyJhbGciOi", "s3cr3t"} {
		if strings.Contains(payload, leaked) {
			t.Fatalf("rahasia %q bocor ke audit: %s", leaked, payload)
		}
	}
	if !strings.Contains(payload, "a@contoh.id") {
		t.Fatal("data tidak sensitif harus tetap ada")
	}
	if !strings.Contains(payload, "plus") {
		t.Fatal("tier harus tetap ada")
	}
	if ContainsSensitive(payload) {
		t.Fatalf("hasil saringan tidak boleh ditandai masih sensitif: %s", payload)
	}
}

func TestContainsSensitiveDetectsLeak(t *testing.T) {
	leaky := `{"password_hash":"$2a$10$x"}`
	if !ContainsSensitive(leaky) {
		t.Fatal("payload dengan password harus terdeteksi")
	}
	clean := `{"tier":"plus"}`
	if ContainsSensitive(clean) {
		t.Fatal("payload bersih tidak boleh ditandai")
	}
}

func TestDestructiveActionsRequireReason(t *testing.T) {
	for _, action := range []Action{UserBan, UserSuspend, UserDelete, UserTierChange, RefundCreate} {
		if !Destructive(action) {
			t.Fatalf("%s harus dianggap destruktif", action)
		}
	}
	for _, action := range []Action{LoginSuccess, UserViewSensitive, CouponCreate} {
		if Destructive(action) {
			t.Fatalf("%s tidak destruktif", action)
		}
	}
}

func TestCatalogHasNoDuplicate(t *testing.T) {
	seen := map[Action]bool{}
	for _, entry := range Catalog {
		if seen[entry.Action] {
			t.Fatalf("aksi duplikat di katalog: %s", entry.Action)
		}
		seen[entry.Action] = true
		if entry.Action == "" || entry.Label == "" {
			t.Fatal("aksi dan label wajib terisi")
		}
	}
}
