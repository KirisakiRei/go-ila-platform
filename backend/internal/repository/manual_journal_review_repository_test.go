package repository

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestManualJournalReviewRepositoryTenantAndPolling(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sql.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := db.AutoMigrate(&model.User{}, &model.ManualJournalAccount{}, &model.ManualJournalEntry{}, &model.ManualJournalCapital{}, &model.ManualJournalReview{}, &model.ManualJournalReviewSetting{}); err != nil {
		t.Fatal(err)
	}
	active := model.User{Email: "active@example.test", Username: "active", Status: "active"}
	suspended := model.User{Email: "suspended@example.test", Username: "suspended", Status: "active", SuspendedAt: timePointerReview(time.Now())}
	for _, u := range []*model.User{&active, &suspended} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	a := model.ManualJournalAccount{UserID: active.ID, Currency: "USD", InitialCapital: "100", InitialDate: "2026-01-01"}
	b := model.ManualJournalAccount{UserID: suspended.ID, Currency: "USD", InitialCapital: "100", InitialDate: "2026-01-01"}
	for _, v := range []*model.ManualJournalAccount{&a, &b} {
		if err := db.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	repo := NewManualJournalReviewRepository(db)
	rows, err := repo.ReviewAccounts(ctx, "")
	if err != nil || len(rows) != 1 || rows[0].ID != a.ID {
		t.Fatalf("active scope %+v %v", rows, err)
	}
	rows, err = repo.ReviewAccounts(ctx, a.ID)
	if err != nil || len(rows) != 0 {
		t.Fatalf("keyset %+v %v", rows, err)
	}
	hash, err := repo.Fingerprint(ctx, active.ID, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.ManualJournalEntry{UserID: suspended.ID, AccountID: a.ID}).Error; err != nil {
		t.Fatal(err)
	}
	after, err := repo.Fingerprint(ctx, active.ID, a.ID)
	if err != nil || hash != after {
		t.Fatal("foreign data affected revision", err)
	}
	row := model.ManualJournalReview{UserID: active.ID, AccountID: a.ID, Date: "2026-11-26", Revision: "r", Status: "pending"}
	if _, err := repo.Prepare(ctx, row); err != nil {
		t.Fatal(err)
	}
	foreign := row
	foreign.UserID = suspended.ID
	ok, err := repo.ClaimReview(ctx, foreign, "foreign", time.Now())
	if err != nil || ok {
		t.Fatal("foreign claim", err)
	}
	if err := repo.SaveReviewSetting(ctx, model.ManualJournalReviewSetting{UserID: suspended.ID, AIEnabled: false}); err != nil {
		t.Fatal(err)
	}
	settings, err := repo.ReviewSettings(ctx, active.ID, a.ID)
	if err != nil || len(settings) != 0 {
		t.Fatal("foreign settings leaked", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := repo.ReviewAccounts(canceled, ""); err == nil {
		t.Fatal("ignored cancellation")
	}
}
func timePointerReview(v time.Time) *time.Time { return &v }
