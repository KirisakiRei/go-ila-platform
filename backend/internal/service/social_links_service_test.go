package service_test

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/service"
	"ila/backend/internal/settings"
)

func TestSocialLinksUpdateAndValidation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	ctx := context.Background()
	repo := repository.NewSettingsRepository(db)
	flags := settings.NewService(repo, nil)
	svc := service.NewSocialLinksService(flags, repo, nil)
	actor := service.AdminActor{ID: "admin-1"}

	// Skema berbahaya ditolak.
	if _, err := svc.Update(ctx, actor, dto.SocialLinks{Discord: "javascript:alert(1)"}); err == nil {
		t.Fatalf("javascript: URL seharusnya ditolak")
	}
	if _, err := svc.Update(ctx, actor, dto.SocialLinks{X: "http://x.com/ila"}); err == nil {
		t.Fatalf("http:// tanpa TLS seharusnya ditolak")
	}

	// Simpan valid & pastikan cache langsung diperbarui.
	saved, err := svc.Update(ctx, actor, dto.SocialLinks{
		Discord:  "https://discord.gg/imaginary",
		X:        "https://x.com/imaginaryliq",
		Telegram: "https://t.me/imaginaryliq",
	})
	if err != nil {
		t.Fatalf("update valid gagal: %v", err)
	}
	got := svc.Get(ctx)
	if got != saved || got.Discord != "https://discord.gg/imaginary" {
		t.Fatalf("nilai terbaca tidak cocok: got=%+v saved=%+v", got, saved)
	}

	// Menghapus salah satu tautan dengan string kosong.
	if _, err := svc.Update(ctx, actor, dto.SocialLinks{
		Discord:  saved.Discord,
		X:        "",
		Telegram: saved.Telegram,
	}); err != nil {
		t.Fatalf("clear X gagal: %v", err)
	}
	got = svc.Get(ctx)
	if got.X != "" || got.Discord == "" || got.Telegram == "" {
		t.Fatalf("clear X tidak bersih: %+v", got)
	}
}
