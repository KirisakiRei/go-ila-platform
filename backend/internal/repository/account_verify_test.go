package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

// newAccountVerifyDB menyiapkan basis data uji dengan tabel akun broker.
//
// Sengaja memakai PostgreSQL, bukan SQLite: driver SQLite butuh cgo yang tidak
// tersedia pada build image (CGO_ENABLED=0), sedangkan PostgreSQL sudah dipakai
// pengujian repository lain di repo ini.
//
// Dilewati bila TEST_POSTGRES_DSN tidak diisi.
func newAccountVerifyDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN kosong; uji PostgreSQL dilewati")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("koneksi: %v", err)
	}
	if err := db.AutoMigrate(&model.BrokerAccount{}); err != nil {
		t.Fatalf("migrasi: %v", err)
	}
	return db
}

// buatAkunUji menyimpan satu akun broker dan mendaftarkan pembersihannya.
func buatAkunUji(t *testing.T, db *gorm.DB, mode string) string {
	t.Helper()

	id := uuid.NewString()
	account := &model.BrokerAccount{
		Base:        model.Base{ID: id},
		UserID:      uuid.NewString(),
		BrokerType:  "matchtrader",
		Mode:        mode,
		RiskMode:    "percent",
		RiskPercent: 0.5,
		IsActive:    true,
		LastError:   "koneksi terputus sebelumnya",
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("membuat akun: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&model.BrokerAccount{}, "id = ?", id) })
	return id
}

// TestMarkAccountVerifiedMengisiWaktu memastikan kolom last_verified_at
// benar-benar ditulis ketika broker menjawab.
//
// Kolom itu dipakai dashboard untuk membedakan "kredensial tersimpan" dari
// "kredensial masih berlaku". Kode sebelumnya hanya membacanya, sehingga akun
// yang jelas hidup tetap tampil belum pernah diverifikasi.
func TestMarkAccountVerifiedMengisiWaktu(t *testing.T) {
	db := newAccountVerifyDB(t)
	ctx := context.Background()

	id := buatAkunUji(t, db, "auto")
	repo := NewRunnerRepository(db)

	saat := time.Date(2026, 5, 4, 10, 15, 0, 0, time.UTC)
	if err := repo.MarkAccountVerified(ctx, id, saat); err != nil {
		t.Fatalf("menandai terverifikasi: %v", err)
	}

	var hasil model.BrokerAccount
	if err := db.WithContext(ctx).First(&hasil, "id = ?", id).Error; err != nil {
		t.Fatalf("membaca ulang akun: %v", err)
	}
	if hasil.LastVerifiedAt == nil {
		t.Fatal("last_verified_at masih kosong setelah verifikasi")
	}
	if !hasil.LastVerifiedAt.Equal(saat) {
		t.Errorf("last_verified_at = %s, ingin %s", hasil.LastVerifiedAt, saat)
	}
	// Keberhasilan membaca akun berarti galat sebelumnya sudah tidak berlaku.
	if hasil.LastError != "" {
		t.Errorf("last_error harus dikosongkan, masih %q", hasil.LastError)
	}
	// Kolom lain tidak boleh ikut tertimpa oleh perubahan user yang bersamaan.
	if hasil.Mode != "auto" {
		t.Errorf("mode berubah menjadi %q, seharusnya tetap auto", hasil.Mode)
	}
	if hasil.RiskPercent != 0.5 {
		t.Errorf("risk_percent berubah menjadi %v, seharusnya tetap 0.5", hasil.RiskPercent)
	}
}

// TestMarkAccountVerifiedTidakMengubahAkunLain memastikan penandaan tidak
// mengenai akun lain.
func TestMarkAccountVerifiedTidakMengubahAkunLain(t *testing.T) {
	db := newAccountVerifyDB(t)
	ctx := context.Background()

	satu := buatAkunUji(t, db, "auto")
	dua := buatAkunUji(t, db, "auto")

	repo := NewRunnerRepository(db)
	if err := repo.MarkAccountVerified(ctx, satu, time.Now().UTC()); err != nil {
		t.Fatalf("menandai terverifikasi: %v", err)
	}

	var lain model.BrokerAccount
	if err := db.WithContext(ctx).First(&lain, "id = ?", dua).Error; err != nil {
		t.Fatalf("membaca akun lain: %v", err)
	}
	if lain.LastVerifiedAt != nil {
		t.Error("akun lain tidak boleh ikut tertandai terverifikasi")
	}
}
