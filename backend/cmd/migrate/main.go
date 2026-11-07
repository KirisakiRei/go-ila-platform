package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/config"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
)

// main menjalankan migrasi tabel dari struct model.
// Dipakai saat pengembangan dan saat penyiapan awal; produksi tetap disarankan
// memakai migrasi SQL yang eksplisit.
func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load("migrate")
	if err != nil {
		logger.Error("konfigurasi tidak sah", "error", err.Error())
		os.Exit(1)
	}

	ctx := context.Background()

	db, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		logger.Error("database tidak siap", "error", err.Error())
		os.Exit(1)
	}

	models := []any{
		&model.User{},
		&model.BrokerAccount{},
		&model.Credential{},
		&model.Subscription{},
		&model.Signal{},
		&model.Position{},
		&model.AuditLog{},
		&model.ApprovalToken{},
		&model.OnboardingToken{},
		&model.NotificationLog{},
	}

	if err := config.AutoMigrate(ctx, db, models...); err != nil {
		logger.Error("migrasi gagal", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("migrasi selesai", "tables", len(models))
}
