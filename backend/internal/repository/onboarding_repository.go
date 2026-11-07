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

// OnboardingTokenRepository menyimpan token /start Telegram sebagai hash.
type OnboardingTokenRepository interface {
	Create(ctx context.Context, token *model.OnboardingToken) error
	FindByHash(ctx context.Context, hash string) (*model.OnboardingToken, error)
	MarkUsed(ctx context.Context, id string, at time.Time) error
}

type onboardingTokenRepository struct {
	db *gorm.DB
}

// NewOnboardingTokenRepository membuat repository token onboarding.
func NewOnboardingTokenRepository(db *gorm.DB) OnboardingTokenRepository {
	return &onboardingTokenRepository{db: db}
}

func (r *onboardingTokenRepository) Create(ctx context.Context, token *model.OnboardingToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("membuat token onboarding: %w", err)
	}
	return nil
}

func (r *onboardingTokenRepository) FindByHash(ctx context.Context, hash string) (*model.OnboardingToken, error) {
	var token model.OnboardingToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca token onboarding: %w", err)
	}
	return &token, nil
}

// MarkUsed mencatat pemakaian; token yang sudah dipakai tidak bisa dipakai lagi.
func (r *onboardingTokenRepository) MarkUsed(ctx context.Context, id string, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.OnboardingToken{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", at)
	if res.Error != nil {
		return fmt.Errorf("menandai token onboarding terpakai: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrConflict
	}
	return nil
}
