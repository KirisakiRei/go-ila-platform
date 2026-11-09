package repository

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/authz"
	"ila/backend/internal/model"
)

// newRoleTestDB menyiapkan basis data sementara berisi tabel RBAC saja.
//
// Basis data memakai file unik per test: `file::memory:?cache=shared` akan
// dipakai bersama seluruh test dalam satu proses, sehingga baris dari test lain
// ikut terhitung dan hasil uji menjadi tidak dapat dipercaya.
func newRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := filepath.ToSlash(filepath.Join(t.TempDir(), "rbac.db"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("membuka basis data uji: %v", err)
	}
	if err := db.AutoMigrate(&model.Role{}, &model.Permission{}, &model.RolePermission{}); err != nil {
		t.Fatalf("menyiapkan skema: %v", err)
	}

	// Koneksi ditutup di akhir test. Di Windows berkas SQLite tidak bisa
	// dihapus selama masih terbuka, dan pembersihan direktori sementara gagal.
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		_ = sqlDB.Close()
	})

	return db
}

// permissionsOf membaca permission satu role dalam bentuk himpunan.
func permissionsOf(t *testing.T, db *gorm.DB, code string) map[string]bool {
	t.Helper()

	var role model.Role
	if err := db.Where("code = ?", code).Take(&role).Error; err != nil {
		t.Fatalf("role %s tidak ditemukan: %v", code, err)
	}

	var keys []string
	if err := db.Model(&model.RolePermission{}).
		Where("role_id = ?", role.ID).
		Pluck("permission_id", &keys).Error; err != nil {
		t.Fatalf("membaca permission role %s: %v", code, err)
	}

	set := make(map[string]bool, len(keys))
	for _, key := range keys {
		set[key] = true
	}
	return set
}

/*
Permission baru di katalog harus sampai ke role bawaan yang sudah ada.

Bug yang dijaga di sini: `SeedDefaultRoles` dulu melewati role yang sudah
terdaftar tanpa menambal apa pun. Akibatnya setelah `role.create` dan
`role.delete` ditambahkan ke katalog, role `superadmin` di server tidak pernah
memegang keduanya — katalog berisi 37 izin tetapi superadmin hanya 35.
*/
func TestSeedDefaultRolesMenambalPermissionBaruPadaRoleYangSudahAda(t *testing.T) {
	db := newRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	// Katalog lama: belum mengenal role.create/role.delete.
	lama := make([]authz.Permission, 0, len(authz.Catalog))
	for _, p := range authz.Catalog {
		if p.Key == authz.RoleCreate || p.Key == authz.RoleDelete {
			continue
		}
		lama = append(lama, p)
	}
	if err := repo.SyncCatalog(ctx, lama); err != nil {
		t.Fatalf("menyelaraskan katalog lama: %v", err)
	}

	// Role bawaan versi lama, tanpa dua permission baru.
	rolesLama := make([]authz.DefaultRole, 0, 4)
	for _, def := range authz.DefaultRoles() {
		bersih := make([]string, 0, len(def.Permissions))
		for _, key := range def.Permissions {
			if key == authz.RoleCreate || key == authz.RoleDelete {
				continue
			}
			bersih = append(bersih, key)
		}
		def.Permissions = bersih
		rolesLama = append(rolesLama, def)
	}
	if err := repo.SeedDefaultRoles(ctx, rolesLama); err != nil {
		t.Fatalf("menyiapkan role lama: %v", err)
	}

	if permissionsOf(t, db, authz.RoleSuperadmin)[authz.RoleCreate] {
		t.Fatal("prasyarat salah: role.create seharusnya belum ada sebelum migrasi")
	}

	// Migrasi berikutnya membawa katalog penuh dan role bawaan penuh.
	if err := repo.SyncCatalog(ctx, authz.Catalog); err != nil {
		t.Fatalf("menyelaraskan katalog baru: %v", err)
	}
	if err := repo.SeedDefaultRoles(ctx, authz.DefaultRoles()); err != nil {
		t.Fatalf("menjalankan seed ulang: %v", err)
	}

	superadmin := permissionsOf(t, db, authz.RoleSuperadmin)
	for _, key := range authz.AllKeys() {
		if !superadmin[key] {
			t.Errorf("superadmin tidak memegang %s setelah migrasi", key)
		}
	}

	admin := permissionsOf(t, db, authz.RoleAdmin)
	if !admin[authz.RoleCreate] || !admin[authz.RoleDelete] {
		t.Error("admin bawaan tidak menerima role.create/role.delete setelah migrasi")
	}
	// Kewenangan yang memang dijaga tidak boleh ikut masuk lewat penambalan.
	if admin[authz.UserDelete] {
		t.Error("admin tidak boleh memegang user.delete")
	}
}

// Penambalan hanya menyentuh permission yang BARU masuk katalog. Permission
// yang sudah ada sebelum migrasi dan sengaja tidak dipegang sebuah role bawaan
// tidak boleh dihidupkan kembali.
func TestSeedDefaultRolesTidakMenghidupkanPermissionLamaYangDicabut(t *testing.T) {
	db := newRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	if err := repo.SyncCatalog(ctx, authz.Catalog); err != nil {
		t.Fatalf("menyelaraskan katalog: %v", err)
	}
	if err := repo.SeedDefaultRoles(ctx, authz.DefaultRoles()); err != nil {
		t.Fatalf("menyiapkan role bawaan: %v", err)
	}

	// Admin sengaja mencabut billing.voucher dari role finance.
	var finance model.Role
	if err := db.Where("code = ?", authz.RoleFinance).Take(&finance).Error; err != nil {
		t.Fatalf("role finance tidak ditemukan: %v", err)
	}
	if err := db.Where("role_id = ? AND permission_id = ?", finance.ID, authz.BillingVoucher).
		Delete(&model.RolePermission{}).Error; err != nil {
		t.Fatalf("mencabut permission: %v", err)
	}

	if err := repo.SeedDefaultRoles(ctx, authz.DefaultRoles()); err != nil {
		t.Fatalf("menjalankan seed ulang: %v", err)
	}

	if permissionsOf(t, db, authz.RoleFinance)[authz.BillingVoucher] {
		t.Error("seed ulang menghidupkan permission lama yang sengaja dicabut admin")
	}
}

// Seed harus idempoten: dijalankan berkali-kali tidak menggandakan baris.
func TestSeedDefaultRolesIdempoten(t *testing.T) {
	db := newRoleTestDB(t)
	repo := NewRoleRepository(db)
	ctx := context.Background()

	if err := repo.SyncCatalog(ctx, authz.Catalog); err != nil {
		t.Fatalf("menyelaraskan katalog: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := repo.SeedDefaultRoles(ctx, authz.DefaultRoles()); err != nil {
			t.Fatalf("seed ke-%d: %v", i+1, err)
		}
	}

	var total int64
	if err := db.Model(&model.RolePermission{}).Count(&total).Error; err != nil {
		t.Fatalf("menghitung role_permission: %v", err)
	}

	// Dihitung per role: dua role bawaan boleh berbagi permission yang sama,
	// jadi yang dicek adalah tidak ada baris kembar pada role yang sama.
	harapan := int64(0)
	for _, def := range authz.DefaultRoles() {
		unik := make(map[string]bool, len(def.Permissions))
		for _, key := range def.Permissions {
			unik[key] = true
		}
		harapan += int64(len(unik))

		if got := int64(len(permissionsOf(t, db, def.Code))); got != int64(len(unik)) {
			t.Errorf("role %s: %d permission tersimpan, seharusnya %d", def.Code, got, len(unik))
		}
	}
	if total != harapan {
		t.Errorf("baris role_permission = %d, seharusnya %d (duplikat?)", total, harapan)
	}

	var roles int64
	if err := db.Model(&model.Role{}).Count(&roles).Error; err != nil {
		t.Fatalf("menghitung role: %v", err)
	}
	if roles != int64(len(authz.DefaultRoles())) {
		t.Errorf("role = %d, seharusnya %d", roles, len(authz.DefaultRoles()))
	}
}
