package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"

	"ila/backend/internal/authz"
	"ila/backend/internal/config"
	"ila/backend/internal/legal"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
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
		&model.UserSignal{},
		&model.SignalEvent{},
		&model.ExecutionRecord{},
		&model.BrokerAccountState{},
		&model.DailySummary{},
		&model.EngineState{},
		&model.Session{},
		&model.Role{},
		&model.Permission{},
		&model.RolePermission{},
		&model.TwoFactor{},
		&model.RecoveryCode{},
		&model.TwoFactorChallenge{},
		&model.LegalDocument{},
		&model.LegalAcceptance{},
		&model.UserAdminNote{},
		&model.Plan{},
		&model.PlanFeature{},
		&model.PriceOverride{},
		&model.PricingGuard{},
		&model.Coupon{},
		&model.CouponRedemption{},
		&model.CreditBalance{},
		&model.CreditTransaction{},
		&model.CreditCode{},
		&model.CreditRedemption{},
		&model.SubscriptionRecord{},
		&model.Invoice{},
		&model.Payment{},
		&model.Refund{},
		&model.SystemSetting{},
		&model.SystemState{},
		&model.SupportNote{},
		&model.Faq{},
		&model.Announcement{},
	}

	if err := config.AutoMigrate(ctx, db, models...); err != nil {
		logger.Error("migrasi gagal", "error", err.Error())
		os.Exit(1)
	}

	// Katalog permission dan role bawaan diselaraskan setiap migrasi agar
	// permission baru di kode ikut masuk tanpa langkah manual (plans/04 §3).
	roles := repository.NewRoleRepository(db)
	if err := roles.SyncCatalog(ctx, authz.Catalog); err != nil {
		logger.Error("katalog permission gagal diselaraskan", "error", err.Error())
		os.Exit(1)
	}
	if err := roles.SeedDefaultRoles(ctx, authz.DefaultRoles()); err != nil {
		logger.Error("role bawaan gagal disiapkan", "error", err.Error())
		os.Exit(1)
	}

	// Paket, fitur, dan batas bawah harga diisi sekali; nilai yang sudah
	// disesuaikan admin tidak ditimpa (plans/12 §9.2).
	if err := repository.SeedPlans(ctx, db); err != nil {
		logger.Error("paket bawaan gagal disiapkan", "error", err.Error())
		os.Exit(1)
	}

	// Akun support lama dan akun admin tanpa role RBAC dipindahkan ke model baru.
	moved, err := roles.MigrateLegacySupport(ctx)
	if err != nil {
		logger.Error("migrasi akun support gagal", "error", err.Error())
		os.Exit(1)
	}
	filled, err := roles.SyncAdminRoleID(ctx, authz.RoleSuperadmin)
	if err != nil {
		logger.Error("migrasi role admin gagal", "error", err.Error())
		os.Exit(1)
	}

	// Dokumen legal versi 1 diterbitkan hanya bila slug itu belum punya versi
	// sama sekali, sehingga perubahan isi oleh admin tidak tertimpa.
	seeded, err := repository.NewLegalRepository(db).SeedInitialDrafts(ctx, legal.Drafts(), time.Now().UTC())
	if err != nil {
		logger.Error("dokumen legal awal gagal disiapkan", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("migrasi selesai",
		"tables", len(models),
		"support_dipindah", moved,
		"admin_dilengkapi", filled,
		"dokumen_legal_diterbitkan", seeded,
	)
}
