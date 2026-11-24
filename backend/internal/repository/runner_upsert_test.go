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

func TestRunnerUpsertSignalMemperbaruiArahDanKonteksRadar(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:runner_upsert_radar?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("buka sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Signal{}); err != nil {
		t.Fatalf("migrasi: %v", err)
	}

	repo := NewRunnerRepository(db)
	ctx := context.Background()
	t1 := time.Date(2026, 11, 9, 17, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 24, 6, 0, 0, 0, time.UTC)

	awal := &model.Signal{
		SignalID:        "status-XAUUSD-D1-H1",
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		EngineVersion:   "2.0.0",
		Htf:             "D1",
		Ltf:             "H1",
		Direction:       "BUY",
		Status:          model.SignalStatusWaitLTFCISD,
		Summary:         "lama buy",
		NextAction:      "lama buy",
		HtfSweepJSON:    `{"direction":"BULLISH"}`,
		PublishedAt:     t1,
		ExpiresAt:       t1.Add(time.Hour),
	}
	if err := repo.UpsertSignal(ctx, awal); err != nil {
		t.Fatalf("upsert awal: %v", err)
	}

	baru := &model.Signal{
		SignalID:        "status-XAUUSD-D1-H1",
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		EngineVersion:   "2.1.1",
		Htf:             "D1",
		Ltf:             "H1",
		Direction:       "SELL",
		Status:          model.SignalStatusWaitLTFCISD,
		Summary:         "-CISD belum terkonfirmasi",
		NextAction:      "-CISD belum terkonfirmasi",
		HtfSweepJSON:    `{"direction":"BEARISH"}`,
		CisdJSON:        `{"direction":"BEARISH","cisdLevel":3632.97}`,
		PublishedAt:     t2,
		ExpiresAt:       t2.Add(time.Hour),
	}
	if err := repo.UpsertSignal(ctx, baru); err != nil {
		t.Fatalf("upsert baru: %v", err)
	}

	got, err := repo.FindSignal(ctx, "status-XAUUSD-D1-H1")
	if err != nil {
		t.Fatalf("baca kembali: %v", err)
	}
	if got.Direction != "SELL" || got.EngineVersion != "2.1.1" || got.HtfSweepJSON != baru.HtfSweepJSON || got.CisdJSON != baru.CisdJSON || !got.PublishedAt.Equal(t2) {
		t.Fatalf("radar tidak terbarui penuh: dir=%q eng=%q htf=%q cisd=%q pub=%v",
			got.Direction, got.EngineVersion, got.HtfSweepJSON, got.CisdJSON, got.PublishedAt)
	}
}
