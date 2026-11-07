package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
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
