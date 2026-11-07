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

type auditRepository struct {
	db *gorm.DB
}

// NewAuditRepository membuat repository audit log append-only.
func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Record(ctx context.Context, entry *model.AuditLog) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(entry).Error; err != nil {
		return fmt.Errorf("menulis audit log: %w", err)
	}
	return nil
}

type approvalTokenRepository struct {
	db *gorm.DB
}

// NewApprovalTokenRepository membuat repository token konfirmasi entry.
func NewApprovalTokenRepository(db *gorm.DB) ApprovalTokenRepository {
	return &approvalTokenRepository{db: db}
}

func (r *approvalTokenRepository) Create(ctx context.Context, token *model.ApprovalToken) error {
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("membuat approval token: %w", err)
	}
	return nil
}

func (r *approvalTokenRepository) FindByHash(ctx context.Context, hash string) (*model.ApprovalToken, error) {
	var token model.ApprovalToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca approval token: %w", err)
	}
	return &token, nil
}

func (r *approvalTokenRepository) MarkUsed(ctx context.Context, id string, at time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.ApprovalToken{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", at)
	if res.Error != nil {
		return fmt.Errorf("menandai approval token terpakai: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		// Token sudah dipakai proses lain: tolak, jangan diam-diam lolos.
		return apperror.ErrConflict
	}
	return nil
}
