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

// newJournalTestDB menyiapkan koneksi PostgreSQL untuk pengujian jurnal.
//
// Jurnal memakai agregat SQL (GROUP BY, SUM CASE) dan perbandingan rentang
// waktu. Perilaku itu berbeda antara SQLite dan PostgreSQL, sehingga
// pengujiannya harus memakai PostgreSQL seperti produksi.
func newJournalTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(&model.Position{}, &model.UserSignal{}, &model.Signal{}); err != nil {
		t.Fatalf("migrasi: %v", err)
	}
	return db
}

// buatPosisi mencatat satu posisi uji lalu mendaftarkan pembersihannya.
func buatPosisi(t *testing.T, db *gorm.DB, userID string, pos *model.Position) {
	t.Helper()

	pos.ID = uuid.NewString()
	pos.UserID = userID
	pos.BrokerAccountID = "acct-uji"
	if pos.OpenedAt.IsZero() {
		pos.OpenedAt = time.Now().UTC()
	}
	if err := db.Create(pos).Error; err != nil {
		t.Fatalf("buat posisi: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(pos) })
}

// TestJournalStatsMemisahkanPosisiTerbuka memastikan posisi yang masih berjalan
// tidak menyeret winrate dan total R. Tanpa pemisahan ini, rata-rata R akan
// tampak lebih buruk daripada kenyataan setiap kali ada posisi baru dibuka.
func TestJournalStatsMemisahkanPosisiTerbuka(t *testing.T) {
	db := newJournalTestDB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	userID := uuid.NewString()

	buatPosisi(t, db, userID, &model.Position{
		Symbol: "XAUUSD", Direction: "BUY", Status: model.PositionStatusClosedTP,
		ProfitLoss: 100, OutcomeR: 2.5, StrategyVersion: "ila-2.0",
	})
	buatPosisi(t, db, userID, &model.Position{
		Symbol: "XAUUSD", Direction: "SELL", Status: model.PositionStatusClosedSL,
		ProfitLoss: -50, OutcomeR: -1, StrategyVersion: "ila-2.0",
	})
	buatPosisi(t, db, userID, &model.Position{
		Symbol: "EURUSD", Direction: "BUY", Status: model.PositionStatusOpen,
		StrategyVersion: "ila-2.0",
	})

	stats, err := repo.Stats(ctx, userID, JournalFilter{})
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if stats.TotalPositions != 3 {
		t.Fatalf("TotalPositions = %d, ingin 3", stats.TotalPositions)
	}
	if stats.OpenPositions != 1 {
		t.Fatalf("OpenPositions = %d, ingin 1", stats.OpenPositions)
	}
	if stats.ClosedTP != 1 || stats.ClosedSL != 1 {
		t.Fatalf("ClosedTP/ClosedSL = %d/%d, ingin 1/1", stats.ClosedTP, stats.ClosedSL)
	}
	if stats.Wins != 1 || stats.Losses != 1 {
		t.Fatalf("Wins/Losses = %d/%d, ingin 1/1", stats.Wins, stats.Losses)
	}
	// 2.5 + (-1.0) = 1.5, posisi terbuka tidak dihitung.
	if stats.TotalR < 1.49 || stats.TotalR > 1.51 {
		t.Fatalf("TotalR = %v, ingin 1.5 (posisi terbuka dikecualikan)", stats.TotalR)
	}
	if stats.TotalProfit != 50 {
		t.Fatalf("TotalProfit = %v, ingin 50", stats.TotalProfit)
	}
}

// TestJournalFilterStrategyDanRentang memastikan filter bekerja pada query
// agregat, bukan hanya pada daftar.
func TestJournalFilterStrategyDanRentang(t *testing.T) {
	db := newJournalTestDB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	userID := uuid.NewString()
	sekarang := time.Now().UTC()

	buatPosisi(t, db, userID, &model.Position{
		Symbol: "XAUUSD", Direction: "BUY", Status: model.PositionStatusClosedTP,
		ProfitLoss: 100, OutcomeR: 2.5, StrategyVersion: "ila-2.0",
		OpenedAt: sekarang.Add(-2 * time.Hour),
	})
	buatPosisi(t, db, userID, &model.Position{
		Symbol: "EURUSD", Direction: "SELL", Status: model.PositionStatusClosedSL,
		ProfitLoss: -30, OutcomeR: -1, StrategyVersion: "ila-3.0",
		OpenedAt: sekarang.Add(-1 * time.Hour),
	})
	// Di luar rentang: 40 hari lalu.
	buatPosisi(t, db, userID, &model.Position{
		Symbol: "XAUUSD", Direction: "BUY", Status: model.PositionStatusClosedTP,
		ProfitLoss: 500, OutcomeR: 5, StrategyVersion: "ila-2.0",
		OpenedAt: sekarang.AddDate(0, 0, -40),
	})

	semua, err := repo.Stats(ctx, userID, JournalFilter{})
	if err != nil {
		t.Fatalf("Stats semua: %v", err)
	}
	if semua.TotalPositions != 3 {
		t.Fatalf("TotalPositions = %d, ingin 3", semua.TotalPositions)
	}

	hanyaIla2, err := repo.Stats(ctx, userID, JournalFilter{StrategyVersion: "ila-2.0"})
	if err != nil {
		t.Fatalf("Stats ila-2.0: %v", err)
	}
	if hanyaIla2.TotalPositions != 2 {
		t.Fatalf("ila-2.0 TotalPositions = %d, ingin 2", hanyaIla2.TotalPositions)
	}

	sebulan, err := repo.Stats(ctx, userID, JournalFilter{From: sekarang.AddDate(0, 0, -30)})
	if err != nil {
		t.Fatalf("Stats 30 hari: %v", err)
	}
	if sebulan.TotalPositions != 2 {
		t.Fatalf("30 hari TotalPositions = %d, ingin 2", sebulan.TotalPositions)
	}

	daftar, err := repo.ListPositions(ctx, userID, JournalFilter{Symbol: "XAUUSD"}, 10)
	if err != nil {
		t.Fatalf("ListPositions: %v", err)
	}
	if len(daftar) != 2 {
		t.Fatalf("XAUUSD = %d baris, ingin 2", len(daftar))
	}
	// Terbaru lebih dulu.
	if daftar[0].OpenedAt.Before(daftar[1].OpenedAt) {
		t.Fatal("ListPositions harus mengurutkan terbaru lebih dulu")
	}
}

// TestJournalSkippedMengabaikanYangSudahJadiPosisi memastikan sinyal CONSUMED
// tidak muncul sebagai peluang terlewat - kalau muncul, satu setup terhitung
// dua kali.
func TestJournalSkippedMengabaikanYangSudahJadiPosisi(t *testing.T) {
	db := newJournalTestDB(t)
	repo := NewJournalRepository(db)
	ctx := context.Background()

	userID := uuid.NewString()
	prefix := userID[:8]

	buat := func(suffix, status, positionID string) {
		us := &model.UserSignal{
			Base:       model.Base{ID: uuid.NewString(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
			UserID:     userID,
			SignalID:   "sig-uji-" + prefix + "-" + suffix,
			Status:     status,
			PositionID: positionID,
		}
		if err := db.Create(us).Error; err != nil {
			t.Fatalf("buat user_signal: %v", err)
		}
		t.Cleanup(func() { db.Unscoped().Delete(us) })
	}

	buat("tolak", model.SignalStatusRejected, "")
	buat("lewat", model.SignalStatusSkipped, "")
	buat("peluang", model.SignalStatusOpportunity, "")
	buat("sudah", model.SignalStatusConsumed, "pos-1")

	rows, err := repo.ListSkipped(ctx, userID, JournalFilter{}, 50)
	if err != nil {
		t.Fatalf("ListSkipped: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("skipped = %d baris, ingin 3 (CONSUMED dikecualikan)", len(rows))
	}
	for _, row := range rows {
		if row.Status == model.SignalStatusConsumed {
			t.Fatal("sinyal CONSUMED tidak boleh muncul di daftar tidak tereksekusi")
		}
	}
}
