package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/authstate"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository membuat repository akun di atas koneksi GORM.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperror.ErrAlreadyExists
		}
		return fmt.Errorf("membuat user: %w", err)
	}
	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	return r.findOne(ctx, "id = ?", id)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.findOne(ctx, "email = ?", email)
}

func (r *userRepository) FindByTelegramChatID(ctx context.Context, chatID string) (*model.User, error) {
	return r.findOne(ctx, "telegram_chat_id = ?", chatID)
}

func (r *userRepository) findOne(ctx context.Context, query string, args ...any) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where(query, args...).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca user: %w", err)
	}
	return &user, nil
}

// ListTrials membaca pengguna yang memakai atau pernah memakai uji coba.
// status: active (masih berjalan), ended (sudah lewat), kosong berarti semua.
func (r *userRepository) ListTrials(ctx context.Context, status string, limit int) ([]model.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("trial_used = ?", true)
	now := time.Now().UTC()
	switch status {
	case "active":
		query = query.Where("trial_expires_at > ?", now)
	case "ended":
		query = query.Where("trial_expires_at <= ?", now)
	}

	var rows []model.User
	if err := query.Order("trial_expires_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca daftar trial: %w", err)
	}
	return rows, nil
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id string, at time.Time) error {
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Update("last_login_at", at).Error
	if err != nil {
		return fmt.Errorf("memperbarui last login: %w", err)
	}
	return nil
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("memperbarui user: %w", err)
	}
	return nil
}

func (r *userRepository) CountByTier(ctx context.Context, tier string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("tier = ?", tier).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung user per tier: %w", err)
	}
	return total, nil
}

// FindSessionUser membaca kolom yang dibutuhkan middleware sesi saja.
// Query ringan agar pemeriksaan tiap request tetap murah.
func (r *userRepository) FindSessionUser(ctx context.Context, id string) (*authstate.User, error) {
	var row struct {
		ID             string
		Role           string
		Status         string
		SessionVersion int
	}
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Select("id, role, status, session_version").
		Where("id = ?", id).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca sesi user: %w", err)
	}
	return &authstate.User{
		ID:             row.ID,
		Role:           row.Role,
		Status:         row.Status,
		SessionVersion: row.SessionVersion,
	}, nil
}

// RevokeSessions menaikkan session_version sehingga seluruh token lama ditolak.
// Dipakai saat ganti password, ban, atau perubahan role.
func (r *userRepository) RevokeSessions(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		UpdateColumn("session_version", gorm.Expr("session_version + 1")).Error
	if err != nil {
		return fmt.Errorf("mencabut sesi: %w", err)
	}
	return nil
}

// UpdatePassword menyimpan hash baru sekaligus menaikkan session_version
// dalam satu pernyataan agar tidak ada celah antara keduanya.
func (r *userRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash":   passwordHash,
			"session_version": gorm.Expr("session_version + 1"),
		}).Error
	if err != nil {
		return fmt.Errorf("memperbarui password: %w", err)
	}
	return nil
}
