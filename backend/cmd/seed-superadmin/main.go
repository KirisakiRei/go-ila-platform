// Perintah seed-superadmin membuat akun dengan kewenangan penuh.
//
// Dijalankan sekali saat penyiapan, atau saat akun superadmin terkunci.
// Password dibaca dari argumen atau SEED_ADMIN_PASSWORD agar tidak tertinggal
// di riwayat shell.
//
//	go run ./cmd/seed-superadmin -email admin@domain -password '...'
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"ila/backend/internal/authz"
	"ila/backend/internal/config"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	email := flag.String("email", "", "email akun superadmin")
	password := flag.String("password", "", "password akun superadmin (kosong berarti dibaca dari SEED_ADMIN_PASSWORD)")
	name := flag.String("name", "Superadmin", "nama yang ditampilkan")
	username := flag.String("username", "superadmin", "username untuk masuk")
	flag.Parse()

	if *password == "" {
		*password = os.Getenv("SEED_ADMIN_PASSWORD")
	}
	if *email == "" || *password == "" {
		logger.Error("email dan password wajib diisi")
		os.Exit(1)
	}
	if len(*password) < 10 {
		logger.Error("password minimal 10 karakter")
		os.Exit(1)
	}

	cfg, err := config.Load("seed-superadmin")
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

	hash, err := utils.HashPassword(*password)
	if err != nil {
		logger.Error("password gagal di-hash", "error", err.Error())
		os.Exit(1)
	}

	now := time.Now().UTC()
	userRepo := repository.NewUserRepository(db)
	emailBersih := strings.ToLower(strings.TrimSpace(*email))

	user, err := userRepo.FindByEmail(ctx, emailBersih)
	switch {
	case err == nil:
		// Akun sudah ada: password dan role diperbarui agar bisa dipakai lagi
		// bila akun terkunci. session_version naik supaya sesi lama gugur.
		user.PasswordHash = hash
		user.Role = string(config.RoleAdmin)
		user.Status = "active"
		user.SessionVersion++
		if user.EmailVerifiedAt == nil {
			user.EmailVerifiedAt = &now
		}
		if err := userRepo.Update(ctx, user); err != nil {
			logger.Error("akun gagal diperbarui", "error", err.Error())
			os.Exit(1)
		}
		logger.Info("akun superadmin diperbarui", "email", user.Email, "id", user.ID)

	case errors.Is(err, apperror.ErrNotFound):
		user = &model.User{
			Name:            *name,
			Username:        strings.ToLower(strings.TrimSpace(*username)),
			Email:           emailBersih,
			PasswordHash:    hash,
			Role:            string(config.RoleAdmin),
			Tier:            string(config.TierPlus),
			Status:          "active",
			NotifyEnabled:   true,
			IdleWarning:     true,
			EmailVerifiedAt: &now,
		}
		user.CreatedAt = now
		user.UpdatedAt = now
		if err := userRepo.Create(ctx, user); err != nil {
			logger.Error("akun gagal dibuat", "error", err.Error())
			os.Exit(1)
		}
		logger.Info("akun superadmin dibuat", "email", user.Email, "id", user.ID)

	default:
		logger.Error("pencarian akun gagal", "error", err.Error())
		os.Exit(1)
	}

	// Role RBAC dilekatkan agar permission panel admin terbaca.
	roleRepo := repository.NewRoleRepository(db)
	role, err := roleRepo.FindRoleByCode(ctx, authz.RoleSuperadmin)
	if err != nil {
		logger.Error("role superadmin tidak ada; jalankan migrate lebih dulu", "error", err.Error())
		os.Exit(1)
	}
	if err := roleRepo.AssignRole(ctx, user.ID, &role.ID); err != nil {
		logger.Error("role gagal dilekatkan", "error", err.Error())
		os.Exit(1)
	}

	fmt.Printf("\nAkun superadmin siap.\n  email    : %s\n  username : %s\n  role     : %s\n  masuk    : %s/app/login\n\n",
		user.Email, user.Username, authz.RoleSuperadmin, strings.TrimRight(cfg.PublicURL, "/"))
}
