package repository

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/authz"
	"ila/backend/internal/model"
)

/*
Penandaan superadmin utama.

Yang dijaga: setelah migrasi berjalan, tepat SATU akun superadmin aktif
bertanda `is_protected`. Tanda itu tidak pernah berpindah ke akun lain pada
migrasi berikutnya, karena memindahkannya akan menggeser akun yang tidak bisa
dihapus tanpa sepengetahuan pemilik workspace.
*/

// newProtectTestDB menyiapkan basis data sementara berisi tabel role dan
// pengguna saja. Berkas dipakai (bukan `file::memory:?cache=shared`) agar tidak
// dipakai bersama test lain dalam satu proses.
func newProtectTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// SQLITE_DQS(0) mematikan kutip ganda sebagai string, sehingga `"user"`
	// dibaca sebagai nama tabel persis seperti di Postgres. Tanpa itu, kueri
	// yang benar di produksi akan gagal di pengujian.
	dsn := filepath.ToSlash(filepath.Join(t.TempDir(), "protect.db")) + "?_pragma=SQLITE_DQS(0)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("membuka basis data uji: %v", err)
	}
	if err := db.AutoMigrate(&model.Role{}, &model.User{}); err != nil {
		t.Fatalf("menyiapkan skema: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("mengambil koneksi: %v", err)
	}
	// Koneksi ditutup sebelum direktori sementara dibersihkan; di Windows
	// berkas SQLite yang masih terbuka tidak bisa dihapus.
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// seedSuperadmin memasang role superadmin dan akun-akun pengelolanya.
func seedSuperadmin(t *testing.T, db *gorm.DB, akun []model.User) {
	t.Helper()

	role := model.Role{
		ID:       "role-super",
		Code:     authz.RoleSuperadmin,
		Name:     "Superadmin",
		IsSystem: true,
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("membuat role superadmin: %v", err)
	}
	adminRole := model.Role{
		ID:       "role-admin",
		Code:     authz.RoleAdmin,
		Name:     "Admin",
		IsSystem: true,
	}
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("membuat role admin: %v", err)
	}

	for i := range akun {
		if akun[i].RoleID == nil {
			id := role.ID
			akun[i].RoleID = &id
		}
		if akun[i].Email == "" {
			akun[i].Email = akun[i].Username + "@admin.invalid"
		}
		if akun[i].CreatedAt.IsZero() {
			akun[i].CreatedAt = time.Now().UTC()
		}
		if err := db.Create(&akun[i]).Error; err != nil {
			t.Fatalf("membuat akun %s: %v", akun[i].Username, err)
		}
	}
}

// protectedEmails membaca akun yang bertanda terlindungi.
func protectedEmails(t *testing.T, db *gorm.DB) []string {
	t.Helper()

	var email []string
	if err := db.Model(&model.User{}).Where("is_protected = ?", true).
		Order("email ASC").Pluck("email", &email).Error; err != nil {
		t.Fatalf("membaca akun terlindungi: %v", err)
	}
	return email
}

// Kueri wajib mengutip nama tabel `user`.
//
// Bug yang dijaga: `user` adalah kata kunci SQL di Postgres. Kueri tanpa kutip
// membuat Postgres membacanya sebagai fungsi USER(), dan migrasi gagal dengan
// "column u.role_id does not exist" — kejadian nyata di server. Pengujian ini
// membaca SQL yang benar-benar akan dikirim GORM.
func TestProtectPrimarySuperadminMengutipNamaTabel(t *testing.T) {
	db := newProtectTestDB(t)
	seedSuperadmin(t, db, []model.User{
		{Base: model.Base{ID: "u-tua"}, Username: "tua", Role: "admin", Status: model.UserStatusActive},
	})

	// ToSQL hanya menyusun kueri, tidak mengeksekusinya.
	sqlTeks := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var ids []string
		return tx.Model(&model.User{}).
			Joins(`JOIN role r ON r.id = "user".role_id`).
			Where("r.code = ?", "superadmin").
			Where(`"user".status = ?`, "active").
			Where(`"user".deleted_at IS NULL`).
			Order(`"user".created_at ASC`).
			Limit(1).
			Pluck(`"user".id`, &ids)
	})

	if !strings.Contains(sqlTeks, `"user"`) {
		t.Fatalf("nama tabel tidak dikutip, kueri akan gagal di Postgres: %s", sqlTeks)
	}
}

// Superadmin tertua ditandai, dan hanya satu.
func TestProtectPrimarySuperadminMenandaiYangTertua(t *testing.T) {
	db := newProtectTestDB(t)
	awal := time.Date(2026, 11, 8, 13, 0, 0, 0, time.UTC)
	seedSuperadmin(t, db, []model.User{
		{Base: model.Base{ID: "u-muda", CreatedAt: awal.Add(3 * time.Hour)}, Username: "muda", Role: "admin", Status: model.UserStatusActive},
		{Base: model.Base{ID: "u-tua", CreatedAt: awal}, Username: "tua", Role: "admin", Status: model.UserStatusActive},
	})

	ditandai, err := ProtectPrimarySuperadmin(context.Background(), db)
	if err != nil {
		t.Fatalf("ProtectPrimarySuperadmin error: %v", err)
	}
	if !ditandai {
		t.Fatal("migrasi tidak menandai akun apa pun")
	}

	got := protectedEmails(t, db)
	if len(got) != 1 || got[0] != "tua@admin.invalid" {
		t.Fatalf("akun terlindungi = %v; ingin [tua@admin.invalid]", got)
	}
}

// Migrasi berikutnya tidak memindahkan tanda ke akun lain.
func TestProtectPrimarySuperadminIdempoten(t *testing.T) {
	db := newProtectTestDB(t)
	awal := time.Date(2026, 11, 8, 13, 0, 0, 0, time.UTC)
	seedSuperadmin(t, db, []model.User{
		{Base: model.Base{ID: "u-tua", CreatedAt: awal}, Username: "tua", Role: "admin", Status: model.UserStatusActive},
	})
	if _, err := ProtectPrimarySuperadmin(context.Background(), db); err != nil {
		t.Fatalf("penandaan pertama error: %v", err)
	}

	// Superadmin yang lebih tua muncul belakangan (mis. hasil pemulihan data).
	superRoleID := "role-super"
	if err := db.Create(&model.User{
		Base:     model.Base{ID: "u-lebih-tua", CreatedAt: awal.Add(-48 * time.Hour)},
		Username: "lebih-tua", Email: "lebih-tua@admin.invalid", Role: "admin",
		RoleID: &superRoleID,
		Status: model.UserStatusActive,
	}).Error; err != nil {
		t.Fatalf("membuat akun tambahan: %v", err)
	}

	ditandai, err := ProtectPrimarySuperadmin(context.Background(), db)
	if err != nil {
		t.Fatalf("penandaan kedua error: %v", err)
	}
	if ditandai {
		t.Fatal("migrasi kedua memindahkan tanda ke akun lain")
	}

	got := protectedEmails(t, db)
	if len(got) != 1 || got[0] != "tua@admin.invalid" {
		t.Fatalf("akun terlindungi = %v; ingin tetap [tua@admin.invalid]", got)
	}
}

// Akun nonaktif tidak dipilih: akun itulah yang tidak bisa dipakai masuk.
func TestProtectPrimarySuperadminMelewatiAkunNonaktif(t *testing.T) {
	db := newProtectTestDB(t)
	awal := time.Date(2026, 11, 8, 13, 0, 0, 0, time.UTC)
	seedSuperadmin(t, db, []model.User{
		{Base: model.Base{ID: "u-beku", CreatedAt: awal}, Username: "beku", Role: "admin", Status: model.UserStatusSuspended},
		{Base: model.Base{ID: "u-aktif", CreatedAt: awal.Add(time.Hour)}, Username: "aktif", Role: "admin", Status: model.UserStatusActive},
	})

	if _, err := ProtectPrimarySuperadmin(context.Background(), db); err != nil {
		t.Fatalf("ProtectPrimarySuperadmin error: %v", err)
	}
	got := protectedEmails(t, db)
	if len(got) != 1 || got[0] != "aktif@admin.invalid" {
		t.Fatalf("akun terlindungi = %v; ingin [aktif@admin.invalid]", got)
	}
}

// Tanpa superadmin, migrasi tidak gagal dan tidak menandai apa pun.
func TestProtectPrimarySuperadminTanpaSuperadmin(t *testing.T) {
	db := newProtectTestDB(t)
	if err := db.Create(&model.Role{ID: "role-admin", Code: authz.RoleAdmin, Name: "Admin"}).Error; err != nil {
		t.Fatalf("membuat role admin: %v", err)
	}

	ditandai, err := ProtectPrimarySuperadmin(context.Background(), db)
	if err != nil {
		t.Fatalf("ProtectPrimarySuperadmin error: %v", err)
	}
	if ditandai {
		t.Fatal("tidak ada superadmin, tidak seharusnya ada yang ditandai")
	}
	if got := protectedEmails(t, db); len(got) != 0 {
		t.Fatalf("akun terlindungi = %v; ingin kosong", got)
	}
}
