package repository

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

// dbUjiBacktest menyiapkan basis data SQLite dalam memori untuk menguji
// repository backtest tanpa Postgres.
func dbUjiBacktest(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:backtest?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("membuka basis data: %v", err)
	}
	if err := db.AutoMigrate(&model.BacktestRun{}, &model.BacktestTrade{}); err != nil {
		t.Fatalf("migrasi: %v", err)
	}
	return db
}

// TestDaftarMengembalikanRunLamaTanpaPaginasi menghalangi bug yang pernah
// terjadi: menghitung total lalu memakai query yang SAMA untuk membaca baris.
//
// Query yang dipakai ulang setelah Count membawa SELECT count(*) di statement
// GORM, sehingga Find berikutnya tidak mengembalikan baris. Akibatnya riwayat
// tampak kosong sampai ada run baru dibuat dalam permintaan yang sama.
func TestDaftarMengembalikanRunLamaTanpaPaginasi(t *testing.T) {
	db := dbUjiBacktest(t)
	repo := NewBacktestRepository(db)
	ctx := context.Background()

	// Dua run lama, keduanya sudah selesai.
	for i, simbol := range []string{"XAUUSD", "EURUSD"} {
		run := &model.BacktestRun{
			UserID:      "user-1",
			Symbol:      simbol,
			HTF:         "D1",
			LTF:         "H1",
			RangeFrom:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			RangeTo:     time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			Equity:      10000,
			RiskPercent: 0.5,
			Strategy:    "ila-2.0",
			StrategyVer: "ila-2.0-v1.0.0",
			Status:      model.BacktestStatusDone,
		}
		if err := repo.Buat(ctx, run); err != nil {
			t.Fatalf("menyimpan run: %v", err)
		}
		// Beri jarak waktu supaya urutan created_at tidak seri.
		if err := db.Model(&model.BacktestRun{}).Where("id = ?", run.ID).
			Update("created_at", time.Now().UTC().Add(time.Duration(i)*time.Minute)).Error; err != nil {
			t.Fatalf("mengatur waktu: %v", err)
		}
	}

	daftar, total, err := repo.Daftar(ctx, "user-1", 30, 0)
	if err != nil {
		t.Fatalf("membaca daftar: %v", err)
	}
	if total != 2 {
		t.Fatalf("total: dapat %d, harap 2", total)
	}
	if len(daftar) != 2 {
		t.Fatalf("baris: dapat %d, harap 2 — query count mencemari pembacaan", len(daftar))
	}
	// Terbaru lebih dulu.
	if daftar[0].Symbol != "EURUSD" {
		t.Fatalf("urutan salah: baris pertama %s, harap EURUSD", daftar[0].Symbol)
	}
}

// TestDaftarMilikOrangLainTidakTerbaca memastikan penyaringan pemilik berlaku
// pada daftar, bukan hanya pada pengambilan satu run.
func TestDaftarMilikOrangLainTidakTerbaca(t *testing.T) {
	db := dbUjiBacktest(t)
	repo := NewBacktestRepository(db)
	ctx := context.Background()

	run := &model.BacktestRun{
		UserID: "user-1", Symbol: "XAUUSD", HTF: "D1", LTF: "H1",
		RangeFrom: time.Now().UTC().AddDate(0, -1, 0), RangeTo: time.Now().UTC(),
		Equity: 10000, RiskPercent: 0.5, Strategy: "ila-2.0",
		StrategyVer: "ila-2.0-v1.0.0", Status: model.BacktestStatusDone,
	}
	if err := repo.Buat(ctx, run); err != nil {
		t.Fatalf("menyimpan run: %v", err)
	}

	daftar, total, err := repo.Daftar(ctx, "user-2", 30, 0)
	if err != nil {
		t.Fatalf("membaca daftar: %v", err)
	}
	if total != 0 || len(daftar) != 0 {
		t.Fatalf("user-2 melihat %d baris milik user-1", len(daftar))
	}
}

// TestTradeDaftarMengembalikanSeluruhEntry menghalangi bug yang sama pada
// daftar entry: menghitung lalu membaca dengan query yang sama.
func TestTradeDaftarMengembalikanSeluruhEntry(t *testing.T) {
	db := dbUjiBacktest(t)
	repo := NewBacktestRepository(db)
	ctx := context.Background()

	selesai := time.Now().UTC()
	runs := []model.BacktestTrade{
		{RunID: "run-1", Seq: 1, Direction: "BUY", EntryTime: selesai, ExitTime: selesai,
			EntryPrice: 1, StopLoss: 0.5, TakeProfit: 2, PlannedRR: 2, Outcome: "TP", ActualR: 1},
		{RunID: "run-1", Seq: 2, Direction: "SELL", EntryTime: selesai, ExitTime: selesai,
			EntryPrice: 1, StopLoss: 1.5, TakeProfit: 0.5, PlannedRR: 2, Outcome: "SL", ActualR: -1},
	}
	if err := db.Create(&runs).Error; err != nil {
		t.Fatalf("menyimpan trade: %v", err)
	}

	daftar, total, err := repo.TradeDaftar(ctx, "run-1", 50, 0)
	if err != nil {
		t.Fatalf("membaca trade: %v", err)
	}
	if total != 2 {
		t.Fatalf("total: dapat %d, harap 2", total)
	}
	if len(daftar) != 2 {
		t.Fatalf("baris: dapat %d, harap 2 — query count mencemari pembacaan", len(daftar))
	}
	if daftar[0].Seq != 1 || daftar[1].Seq != 2 {
		t.Fatalf("urutan seq salah: %d, %d", daftar[0].Seq, daftar[1].Seq)
	}
}
