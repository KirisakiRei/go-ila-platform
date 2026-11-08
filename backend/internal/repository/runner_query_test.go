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

// TestPemantauSimbolPostgres menjalankan query sungguhan ke PostgreSQL.
//
// "user" adalah kata kunci SQL. Tanpa kutip, JOIN ke tabel itu gagal dan
// Runner tidak pernah mengirim sinyal ke siapa pun. Bug ini tidak terlihat di
// SQLite, jadi pengujiannya harus memakai PostgreSQL.
//
// Dilewati bila TEST_POSTGRES_DSN tidak diisi.
func TestPemantauSimbolPostgres(t *testing.T) {
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

	ctx := context.Background()
	if err := db.AutoMigrate(&model.User{}, &model.Subscription{}); err != nil {
		t.Fatalf("migrasi: %v", err)
	}

	userID := uuid.NewString()
	now := time.Now().UTC()

	user := &model.User{
		Base:  model.Base{ID: userID, CreatedAt: now, UpdatedAt: now},
		Email: userID + "@uji.local", PasswordHash: "x", Name: "Uji",
		Role: "user", Status: "active", Tier: model.TierPlus, SessionVersion: 1,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("buat user: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(user) })

	sub := &model.Subscription{
		Base:   model.Base{ID: uuid.NewString(), CreatedAt: now, UpdatedAt: now},
		UserID: userID, Symbol: "XAUUSD", Strategy: "ila-2.0", IsActive: true,
	}
	if err := db.Create(sub).Error; err != nil {
		t.Fatalf("buat langganan: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(sub) })

	repo := NewRunnerRepository(db)
	users, err := repo.ListUsersBySymbol(ctx, "XAUUSD")
	if err != nil {
		t.Fatalf("ListUsersBySymbol gagal: %v", err)
	}
	if len(users) != 1 || users[0].ID != userID {
		t.Fatalf("pemantau = %d user, ingin tepat 1", len(users))
	}
}

// Langganan yang dicabut tidak boleh ikut dianggap pemantau.
func TestPemantauSimbolMengabaikanYangNonaktif(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN kosong")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("koneksi: %v", err)
	}
	ctx := context.Background()
	if err := db.AutoMigrate(&model.User{}, &model.Subscription{}); err != nil {
		t.Fatalf("migrasi: %v", err)
	}

	userID := uuid.NewString()
	now := time.Now().UTC()
	user := &model.User{
		Base:  model.Base{ID: userID, CreatedAt: now, UpdatedAt: now},
		Email: userID + "@uji.local", PasswordHash: "x", Name: "Uji",
		Role: "user", Status: "active", Tier: model.TierPlus, SessionVersion: 1,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("buat user: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(user) })

	sub := &model.Subscription{
		Base:   model.Base{ID: uuid.NewString(), CreatedAt: now, UpdatedAt: now},
		UserID: userID, Symbol: "EURUSD", Strategy: "ila-2.0", IsActive: false,
	}
	if err := db.Create(sub).Error; err != nil {
		t.Fatalf("buat langganan: %v", err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(sub) })

	repo := NewRunnerRepository(db)
	users, err := repo.ListUsersBySymbol(ctx, "EURUSD")
	if err != nil {
		t.Fatalf("ListUsersBySymbol gagal: %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("langganan nonaktif tidak boleh dianggap pemantau, dapat %d", len(users))
	}
}
