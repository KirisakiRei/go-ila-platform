package repository

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
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

Uji ini WAJIB memakai PostgreSQL, bukan SQLite.

Alasan: query memakai `SELECT DISTINCT` sekaligus `ORDER BY` sebuah ekspresi.
PostgreSQL menolaknya (SQLSTATE 42P10) sedangkan SQLite menerimanya. Versi
SQLite membuat bug ini lolos semua uji lokal dan baru muncul di produksi
sebagai HTTP 500 pada /api/v1/signals — dashboard tampak kosong padahal data
radar ada. Bila TEST_POSTGRES_DSN kosong, uji dilewati dengan alasan jelas.
*/

// newQueryTestDB menyiapkan basis data uji PostgreSQL. Setiap uji memakai
// penanda unik sehingga data uji lain tidak saling mengganggu.
func newQueryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN kosong; uji PostgreSQL dilewati " +
			"(tanpa PostgreSQL, regresi DISTINCT/ORDER BY tidak akan tertangkap)")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
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

	// Penanda unik per uji agar data tidak bercampur dengan uji lain.
	// Panjang dijaga: subscription.user_id varchar(36), signal.symbol varchar(20).
	userID := uuid.NewString()
	symbol := "UJI" + userID[:6]

	t.Cleanup(func() {
		db.Exec("DELETE FROM signal WHERE symbol = ?", symbol)
		db.Exec("DELETE FROM subscription WHERE user_id = ?", userID)
	})

	if err := db.Create(&model.Subscription{
		Base:     model.Base{ID: uuid.NewString()},
		Symbol:   symbol,
		UserID:   userID,
		IsActive: true,
	}).Error; err != nil {
		t.Fatalf("menyiapkan langganan: %v", err)
	}

	// Baris radar: identitas tetap, sehingga published_at-nya paling tua.
	radar := model.Signal{
		Base:            model.Base{ID: uuid.NewString()},
		SignalID:        "status-" + symbol + "-H1-M5",
		Symbol:          symbol,
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
			Base:            model.Base{ID: uuid.NewString()},
			SignalID:        fmt.Sprintf("sig-%s-H1-M5-%d", symbol, i),
			Symbol:          symbol,
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
	rows, err := repo.ListForUser(ctx, userID, now.Add(-7*24*time.Hour), 100)
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

// TestListForUserTidakMemakaiDistinct mengunci regresi yang pernah mematikan
// seluruh dashboard.
//
// Dulu kueri memakai JOIN ke subscription lalu menambal duplikatnya dengan
// SELECT DISTINCT. PostgreSQL menolak DISTINCT yang ORDER BY-nya memuat
// ekspresi di luar select list (SQLSTATE 42P10), sehingga /api/v1/signals
// gagal 500 dan radar serta riwayat tampak kosong sepenuhnya.
//
// Uji ini berjalan tanpa PostgreSQL: ia memeriksa SQL yang benar-benar
// dikirim ke basis data. Inilah satu-satunya cara menangkap kelas bug ini di
// lingkungan yang tidak punya Postgres (uji DSN sebelumnya dilewati).
func TestListForUserTidakMemakaiDistinct(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:distinct?mode=memory&cache=shared"), &gorm.Config{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("membuka basis data: %v", err)
	}

	repo := NewSignalRepository(db)
	var sql string
	db.Callback().Query().After("gorm:query").Register("uji:tangkap_sql", func(d *gorm.DB) {
		sql = d.Statement.SQL.String()
	})

	// Kueri cukup dijalankan untuk membentuk SQL; hasilnya tidak dipakai.
	_, _ = repo.ListForUser(context.Background(), uuid.NewString(), time.Now().Add(-24*time.Hour), 10)

	if sql == "" {
		t.Fatalf("SQL tidak tertangkap; uji tidak dapat menyimpulkan apa pun")
	}
	if strings.Contains(strings.ToUpper(sql), "DISTINCT") {
		t.Fatalf("kueri tidak boleh memakai DISTINCT (penyebab SQLSTATE 42P10):\n%s", sql)
	}
	if !strings.Contains(strings.ToUpper(sql), "EXISTS") {
		t.Fatalf("kueri harus menyaring langganan dengan EXISTS:\n%s", sql)
	}
}
