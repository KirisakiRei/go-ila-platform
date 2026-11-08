package authz

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeLoader struct {
	perms []string
	role  string
	calls int
	err   error
}

func (f *fakeLoader) PermissionsOf(context.Context, string) ([]string, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.perms, nil
}

func (f *fakeLoader) RoleCodeOf(context.Context, string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.role, nil
}

func TestHasDanPermissions(t *testing.T) {
	service := NewService(&fakeLoader{perms: []string{UserView, UserBan}, role: RoleAdmin})

	ctx := context.Background()
	if ok, err := service.Has(ctx, "u1", UserBan); err != nil || !ok {
		t.Fatalf("Has(UserBan) = %v, %v; ingin true", ok, err)
	}
	if ok, err := service.Has(ctx, "u1", SystemSettings); err != nil || ok {
		t.Fatalf("Has(SystemSettings) = %v, %v; ingin false", ok, err)
	}

	perms, err := service.Permissions(ctx, "u1")
	if err != nil {
		t.Fatalf("Permissions error: %v", err)
	}
	// Urutan mengikuti katalog, bukan urutan database.
	if len(perms) != 2 || perms[0] != UserView || perms[1] != UserBan {
		t.Fatalf("Permissions = %v; ingin [%s %s]", perms, UserView, UserBan)
	}
}

func TestCacheTidakMemanggilUlangDalamTTL(t *testing.T) {
	loader := &fakeLoader{perms: []string{UserView}}
	service := NewService(loader)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := service.Has(ctx, "u1", UserView); err != nil {
			t.Fatalf("Has error: %v", err)
		}
	}
	if loader.calls != 1 {
		t.Fatalf("loader dipanggil %d kali; ingin 1", loader.calls)
	}
}

func TestCacheKedaluwarsaSetelahTTL(t *testing.T) {
	loader := &fakeLoader{perms: []string{UserView}}
	service := NewService(loader)

	now := time.Now().UTC()
	service.now = func() time.Time { return now }

	ctx := context.Background()
	if _, err := service.Has(ctx, "u1", UserView); err != nil {
		t.Fatalf("Has error: %v", err)
	}

	// Perubahan kewenangan terjadi di DB, tetapi cache belum habis.
	loader.perms = []string{UserView, UserBan}
	if ok, _ := service.Has(ctx, "u1", UserBan); ok {
		t.Fatal("cache seharusnya masih menyembunyikan permission baru")
	}

	now = now.Add(CacheTTL + time.Second)
	if ok, err := service.Has(ctx, "u1", UserBan); err != nil || !ok {
		t.Fatalf("setelah TTL Has(UserBan) = %v, %v; ingin true", ok, err)
	}
	if loader.calls != 2 {
		t.Fatalf("loader dipanggil %d kali; ingin 2", loader.calls)
	}
}

func TestInvalidateMembuangCacheSatuAkun(t *testing.T) {
	loader := &fakeLoader{perms: []string{UserView}}
	service := NewService(loader)

	ctx := context.Background()
	if _, err := service.Has(ctx, "u1", UserView); err != nil {
		t.Fatalf("Has error: %v", err)
	}

	service.Invalidate("u1")
	loader.perms = []string{UserBan}
	if ok, err := service.Has(ctx, "u1", UserBan); err != nil || !ok {
		t.Fatalf("setelah Invalidate Has(UserBan) = %v, %v; ingin true", ok, err)
	}
}

func TestInvalidateAllMembuangSeluruhCache(t *testing.T) {
	loader := &fakeLoader{perms: []string{UserView}}
	service := NewService(loader)

	ctx := context.Background()
	if _, err := service.Has(ctx, "u1", UserView); err != nil {
		t.Fatalf("Has error: %v", err)
	}
	if _, err := service.Has(ctx, "u2", UserView); err != nil {
		t.Fatalf("Has error: %v", err)
	}

	service.InvalidateAll()
	if _, err := service.Has(ctx, "u3", UserView); err != nil {
		t.Fatalf("Has error: %v", err)
	}
	if loader.calls != 3 {
		t.Fatalf("loader dipanggil %d kali; ingin 3", loader.calls)
	}
}

func TestErrorLoaderDiteruskan(t *testing.T) {
	sentinel := errors.New("database mati")
	service := NewService(&fakeLoader{err: sentinel})

	if _, err := service.Has(context.Background(), "u1", UserView); !errors.Is(err, sentinel) {
		t.Fatalf("error = %v; ingin %v", err, sentinel)
	}
}

func TestKatalogTanpaDuplikat(t *testing.T) {
	seen := make(map[string]struct{}, len(Catalog))
	for _, p := range Catalog {
		if _, dup := seen[p.Key]; dup {
			t.Fatalf("permission ganda: %s", p.Key)
		}
		seen[p.Key] = struct{}{}
		if p.Group == "" || p.Label == "" || p.Risk == "" {
			t.Fatalf("permission %s belum lengkap", p.Key)
		}
	}
}

func TestRoleBawaanSuperadminPunyaSemua(t *testing.T) {
	var superadmin *DefaultRole
	for i := range DefaultRoles() {
		if DefaultRoles()[i].Code == RoleSuperadmin {
			superadmin = &DefaultRoles()[i]
			break
		}
	}
	if superadmin == nil {
		t.Fatal("role superadmin tidak ada di daftar bawaan")
	}
	if len(superadmin.Permissions) != len(Catalog) {
		t.Fatalf("superadmin punya %d permission; ingin %d", len(superadmin.Permissions), len(Catalog))
	}
}

func TestAdminTidakPunyaKewenanganTerkunci(t *testing.T) {
	locked := map[string]bool{
		SystemSettings:    true,
		SystemMaintenance: true,
		BillingSettings:   true,
		UserDelete:        true,
	}

	for _, role := range DefaultRoles() {
		if role.Code != RoleAdmin {
			continue
		}
		for _, key := range role.Permissions {
			if locked[key] {
				t.Fatalf("admin seharusnya tidak punya %s", key)
			}
		}
		return
	}
	t.Fatal("role admin tidak ada di daftar bawaan")
}

func TestSuperadminOnlyTidakDapatDidelegasikan(t *testing.T) {
	// Admin biasa yang diberi izin lewat role tetap ditolak: kewenangan
	// kritis dikunci ke role superadmin (plans/04 §6).
	admin := NewService(&fakeLoader{perms: []string{SystemSettings, SystemMaintenance}, role: RoleAdmin})
	ctx := context.Background()
	if ok, _ := admin.Has(ctx, "a1", SystemSettings); ok {
		t.Fatal("admin biasa tidak boleh memegang system.settings")
	}

	super := NewService(&fakeLoader{perms: []string{SystemSettings}, role: RoleSuperadmin})
	if ok, _ := super.Has(ctx, "s1", SystemSettings); !ok {
		t.Fatal("superadmin harus memegang system.settings")
	}
}

func TestSuperadminOnly(t *testing.T) {
	if !SuperadminOnly(SystemSettings) || !SuperadminOnly(SystemMaintenance) {
		t.Fatal("flag sistem harus terkunci ke superadmin")
	}
	if SuperadminOnly(UserBan) {
		t.Fatal("user.ban bukan kewenangan terkunci")
	}
}

func TestKnown(t *testing.T) {
	if !Known(UserView) {
		t.Fatal("user.view harus dikenal")
	}
	if Known("tidak.ada") {
		t.Fatal("kunci asing tidak boleh dikenal")
	}
}
