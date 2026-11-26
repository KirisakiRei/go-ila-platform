package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestManualJournalRepositoryOwnershipFiltersAndContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	if err := db.AutoMigrate(&model.ManualJournalEntry{}, &model.ManualJournalScreenshot{}, &model.ManualJournalImportToken{}); err != nil {
		t.Fatal(err)
	}
	r := NewManualJournalRepository(db)
	ctx := context.Background()
	user, other, account := uuid.NewString(), uuid.NewString(), uuid.NewString()
	closed := time.Date(2026, 1, 1, 18, 0, 0, 0, time.UTC)
	target := model.ManualJournalEntry{UserID: user, AccountID: account, Status: "closed", Asset: "XAU", Tags: []string{"a%_!"}, Emotions: []string{"fear%_!\""}, Mistakes: []string{"rush%_!"}, Notes: "100%_! exact", ClosedAt: &closed}
	if err := r.Create(ctx, &target); err != nil {
		t.Fatal(err)
	}
	foreign := target
	foreign.Base = model.Base{}
	foreign.UserID = other
	if err := r.Create(ctx, &foreign); err != nil {
		t.Fatal(err)
	}
	extra := target
	extra.Base = model.Base{}
	extra.Notes = "other"
	extra.Tags = []string{"a"}
	extra.Emotions = []string{"fear"}
	extra.Mistakes = []string{"rush"}
	if err := r.Create(ctx, &extra); err != nil {
		t.Fatal(err)
	}
	f := model.ManualJournalFilter{Page: 1, PerPage: 1, Search: "%_!", Tag: "a%_!", AccountID: account, Status: "closed", Asset: "XAU", From: "2026-01-02", To: "2026-01-02"}
	rows, n, err := r.Entries(ctx, user, f, false)
	if err != nil || n != 1 || len(rows) != 1 || rows[0].ID != target.ID {
		t.Fatalf("filters leaked/wildcard: %+v %d %v", rows, n, err)
	}
	for _, f := range []model.ManualJournalFilter{
		{Page: 1, PerPage: 20, Emotion: "fear%_!\""},
		{Page: 1, PerPage: 20, Mistake: "rush%_!"},
		{Page: 1, PerPage: 20, Emotion: "fear%_!\"", Mistake: "rush%_!"},
	} {
		for _, report := range []bool{false, true} {
			rows, _, err := r.Entries(ctx, user, f, report)
			if err != nil || len(rows) != 1 || rows[0].ID != target.ID {
				t.Fatalf("emotion/mistake owner exact filter: %+v %v", rows, err)
			}
		}
	}
	if rows, _, err := r.Entries(ctx, user, model.ManualJournalFilter{Page: 1, PerPage: 20, Emotion: "fear%_!\"", Mistake: "rush"}, false); err != nil || len(rows) != 0 {
		t.Fatalf("filters not AND: %+v %v", rows, err)
	}
	if _, err := r.Entry(ctx, other, target.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("foreign detail: %v", err)
	}
	target.Notes = "mutated"
	if err := r.Update(ctx, other, target.ID, &target); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("foreign update %v", err)
	}
	if err := r.Delete(ctx, other, target.ID, &model.ManualJournalEntry{}); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("foreign delete %v", err)
	}
	got, err := r.Entry(ctx, user, target.ID)
	if err != nil || got.Notes != "100%_! exact" {
		t.Fatalf("foreign mutation %+v %v", got, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := r.Entries(canceled, user, f, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("context not propagated: %v", err)
	}
	shot := model.ManualJournalScreenshot{UserID: user, EntryID: target.ID, StoredAs: "screen.png"}
	if err := r.Create(ctx, &shot); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(ctx, &model.ManualJournalScreenshot{UserID: other, EntryID: target.ID}); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("foreign attachment accepted: %v", err)
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("fail_screenshot_delete", func(tx *gorm.DB) {
		if tx.Statement.Table == "manual_journal_screenshots" {
			tx.AddError(errors.New("forced screenshot delete failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, user, target.ID, &model.ManualJournalEntry{}); err == nil {
		t.Fatal("delete failure swallowed")
	}
	if _, err := r.Entry(ctx, user, target.ID); err != nil {
		t.Fatalf("entry delete not rolled back: %v", err)
	}
	if _, err := r.Screenshot(ctx, user, target.ID, shot.ID); err != nil {
		t.Fatalf("attachment lost on rollback: %v", err)
	}
	if err := db.Callback().Delete().Remove("fail_screenshot_delete"); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, user, target.ID, &model.ManualJournalEntry{}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Screenshot(ctx, user, target.ID, shot.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("attachment not tombstoned: %v", err)
	}
	if err := r.PurgeScreenshot(ctx, other, shot.ID); err != nil {
		t.Fatal(err)
	}
	pending, err := r.ScreenshotTombstones(ctx, "", "")
	if err != nil || len(pending) != 1 || pending[0].UserID != user {
		t.Fatalf("owner tombstone lost: %+v %v", pending, err)
	}
	if err := r.Update(ctx, user, target.ID, &target); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("deleted row resurrected %v", err)
	}
}

func TestManualJournalRepositoryImportRollsBack(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	if err := db.AutoMigrate(&model.ManualJournalEntry{}, &model.ManualJournalImportToken{}); err != nil {
		t.Fatal(err)
	}
	rows := make([]model.ManualJournalEntry, 51)
	for i := range rows {
		rows[i] = model.ManualJournalEntry{Base: model.Base{ID: uuid.NewString()}, UserID: "owner", AccountID: "account", Status: "draft"}
	}
	rows[50].ID = rows[0].ID
	r := NewManualJournalRepository(db)
	token := model.ManualJournalImportToken{UserID: "owner", PayloadHash: "hash", ExpiresAt: time.Now().Add(time.Hour)}
	if err := r.Create(context.Background(), &token); err != nil {
		t.Fatal(err)
	}
	if err := r.Import(context.Background(), "owner", token.ID, "hash", rows); err == nil {
		t.Fatal("duplicate primary key import accepted")
	}
	var count int64
	if err := db.Model(&model.ManualJournalEntry{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("partial batch commit: %d %v", count, err)
	}
	var saved model.ManualJournalImportToken
	if err := db.First(&saved, "id = ?", token.ID).Error; err != nil || saved.ConsumedAt != nil {
		t.Fatalf("token consumed after rollback: %+v %v", saved, err)
	}
	rows[50].ID = uuid.NewString()
	if err := r.Import(context.Background(), "owner", token.ID, "hash", rows); err != nil {
		t.Fatal("retry after rollback", err)
	}
}
