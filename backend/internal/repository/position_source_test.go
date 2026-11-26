package repository

import (
	"context"
	"testing"

	"ila/backend/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPositionSourceMetadataLookup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&model.Signal{}); err != nil {
		t.Fatal(err)
	}
	for _, signal := range []model.Signal{
		{SignalID: "a", Symbol: "XAUUSD", Htf: "H4", Ltf: "M5", FeedSource: "ICMARKETS", EntryPrice: 999},
		{SignalID: "b", Symbol: "EURUSD", Htf: "D1", Ltf: "H1"},
	} {
		if err := db.Create(&signal).Error; err != nil {
			t.Fatal(err)
		}
	}
	// Assert the concrete production repository retains its batch capability.
	reader, ok := NewSignalRepository(db).(interface {
		FindPositionSources(context.Context, []string) (map[string]model.Signal, error)
	})
	if !ok {
		t.Fatal("production repository lacks batch lookup")
	}
	ctx := context.Background()
	rows, err := reader.FindPositionSources(ctx, []string{"a", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows["a"].Htf != "H4" || rows["a"].Ltf != "M5" || rows["a"].FeedSource != "ICMARKETS" || rows["a"].EntryPrice != 0 {
		t.Fatalf("unexpected source projection: %+v", rows)
	}
	if rows, err := reader.FindPositionSources(ctx, nil); err != nil || len(rows) != 0 {
		t.Fatalf("empty lookup: %+v %v", rows, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := reader.FindPositionSources(canceled, []string{"a"}); err == nil {
		t.Fatal("query ignored canceled context")
	}
}
