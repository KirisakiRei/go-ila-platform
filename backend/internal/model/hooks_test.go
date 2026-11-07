package model_test

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
)

// newTestDB membuka database uji dari TEST_DB_DSN.
// Tes dilewati bila variabel kosong agar `go test ./...` tetap jalan tanpa
// PostgreSQL.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN tidak diisi")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("membuka database: %v", err)
	}
	return db
}

// TestIDTerisiOtomatis menjaga bug yang ditemukan saat uji integrasi: model
// tanpa Base tidak boleh gagal insert karena kolom id NOT NULL.
func TestIDTerisiOtomatis(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	if err := db.WithContext(ctx).AutoMigrate(
		&model.AuditLog{},
		&model.NotificationLog{},
		&model.SignalEvent{},
		&model.ExecutionRecord{},
		&model.OnboardingToken{},
	); err != nil {
		t.Fatalf("migrasi tabel uji: %v", err)
	}

	now := time.Now().UTC()
	const userID = "u-uji-id"

	rows := []struct {
		table string
		row   any
	}{
		{"audit_log", &model.AuditLog{UserID: userID, Event: "uji", CreatedAt: now}},
		{"notification_log", &model.NotificationLog{
			UserID: userID, Channel: "telegram", MessageKey: "kunci-uji", Status: "queued", CreatedAt: now,
		}},
		{"signal_event", &model.SignalEvent{
			UserID: userID, SignalID: "sig-uji", ToStatus: "READY_TO_ENTRY", CreatedAt: now,
		}},
		{"execution_record", &model.ExecutionRecord{
			UserID: userID, SignalID: "uji", BrokerAccountID: "acc-uji",
			IdempotencyKey: "kunci-uji", Status: "CLAIMED", CreatedAt: now,
		}},
	}

	for _, item := range rows {
		if err := db.WithContext(ctx).Create(item.row).Error; err != nil {
			t.Fatalf("%s: %v", item.table, err)
		}
		db.WithContext(ctx).Exec("DELETE FROM "+item.table+" WHERE user_id = ?", userID)
	}

	// OnboardingToken memakai Base, sehingga ID diisi hook Base.
	if err := db.WithContext(ctx).Create(&model.OnboardingToken{
		TokenHash: "hash-uji-id", UserID: userID, ExpiresAt: now.Add(time.Minute),
	}).Error; err != nil {
		t.Fatalf("onboarding_token: %v", err)
	}
	db.WithContext(ctx).Exec("DELETE FROM onboarding_token WHERE user_id = ?", userID)
}
