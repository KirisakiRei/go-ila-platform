package repository

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

/*
Pembacaan sinyal untuk dashboard user.

Yang dijaga:
  - Baris radar (`status-*`) tidak boleh terpotong oleh limit meskipun
    `published_at`-nya jauh lebih lama daripada riwayat sinyal lain. Bila
    terpotong, matriks timeframe tampak kosong padahal radar masih bergerak.
  - Hasil tetap dibatasi pada simbol yang benar-benar dilanggan user.
*/

// newQueryTestDB menyiapkan basis data sementara berisi tabel sinyal dan
// langganan. Berkas dipakai agar tidak berbagi keadaan dengan tes lain.
func newQueryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := filepath.ToSlash(filepath.Join(t.TempDir(), "query.db"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("membuka basis data uji: %v", err)
	}
	if err := db.AutoMigrate(&model.Signal{}, &model.Subscription{}); err != nil {
		t.Fatalf("menyiapkan skema: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("membuka koneksi basis data: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	return db
}

func TestListForUserMempertahankanBarisRadar(t *testing.T) {
	db := newQueryTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	if err := db.Create(&model.Subscription{
		Symbol:   "XAUUSD",
		UserID:   "u-1",
		IsActive: true,
	}).Error; err != nil {
		t.Fatalf("menyiapkan langganan: %v", err)
	}

	// Baris radar: identitas tetap, sehingga published_at-nya paling tua.
	radar := model.Signal{
		SignalID:        "status-XAUUSD-H1-M5",
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		Direction:       "SELL",
		Status:          "WAIT_HTF_SWEEP",
		PublishedAt:     now.Add(-72 * time.Hour),
		ExpiresAt:       now.Add(24 * time.Hour),
	}
	if err := db.Create(&radar).Error; err != nil {
		t.Fatalf("menyimpan baris radar: %v", err)
	}

	// Riwayat sinyal yang jauh lebih baru dan menumpuk melebihi limit.
	for i := 0; i < 120; i++ {
		s := model.Signal{
			SignalID:        fmt.Sprintf("sig-XAUUSD-H1-M5-%d", i),
			Symbol:          "XAUUSD",
			StrategyVersion: "ila-2.0",
			Direction:       "SELL",
			Status:          model.SignalStatusExpired,
			PublishedAt:     now.Add(-time.Duration(i) * time.Minute),
			ExpiresAt:       now.Add(-time.Duration(i) * time.Minute),
		}
		if err := db.Create(&s).Error; err != nil {
			t.Fatalf("menyimpan sinyal riwayat: %v", err)
		}
	}

	repo := NewSignalRepository(db)
	rows, err := repo.ListForUser(ctx, "u-1", now.Add(-7*24*time.Hour), 100)
	if err != nil {
		t.Fatalf("membaca sinyal user: %v", err)
	}

	if len(rows) == 0 {
		t.Fatal("hasil kosong, seharusnya memuat baris radar")
	}
	if rows[0].SignalID != radar.SignalID {
		t.Fatalf("baris radar harus berada di urutan pertama, dapat %q", rows[0].SignalID)
	}

	adaRadar := false
	for _, r := range rows {
		if r.SignalID == radar.SignalID {
			adaRadar = true
			break
		}
	}
	if !adaRadar {
		t.Fatal("baris radar terpotong oleh limit; matriks timeframe akan tampak kosong")
	}
}
