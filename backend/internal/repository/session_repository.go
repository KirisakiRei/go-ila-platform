package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// SessionRepository menyimpan dan mencabut sesi per perangkat.
type SessionRepository interface {
	Create(ctx context.Context, s *model.Session) error
	FindByTokenID(ctx context.Context, tokenID string) (*model.Session, error)
	ListByUser(ctx context.Context, userID, scope string) ([]model.Session, error)
	Touch(ctx context.Context, id string, at time.Time) error
	Revoke(ctx context.Context, id, userID string) error
	RevokeAll(ctx context.Context, userID, scope string) error
}

type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository membuat repository sesi di atas koneksi GORM.
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, s *model.Session) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return fmt.Errorf("membuat sesi: %w", err)
	}
	return nil
}

func (r *sessionRepository) FindByTokenID(ctx context.Context, tokenID string) (*model.Session, error) {
	var s model.Session
	err := r.db.WithContext(ctx).Where("token_id = ?", tokenID).Take(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca sesi: %w", err)
	}
	return &s, nil
}

func (r *sessionRepository) ListByUser(ctx context.Context, userID, scope string) ([]model.Session, error) {
	query := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now().UTC()).
		Order("last_seen_at DESC")

	if scope != "" {
		query = query.Where("scope = ?", scope)
	}

	var out []model.Session
	if err := query.Find(&out).Error; err != nil {
		return nil, fmt.Errorf("membaca daftar sesi: %w", err)
	}
	return out, nil
}

func (r *sessionRepository) Touch(ctx context.Context, id string, at time.Time) error {
	err := r.db.WithContext(ctx).Model(&model.Session{}).
		Where("id = ?", id).
		UpdateColumn("last_seen_at", at).Error
	if err != nil {
		return fmt.Errorf("memperbarui sesi: %w", err)
	}
	return nil
}

// Revoke mencabut satu sesi milik user. Sesi orang lain tidak bisa dicabut.
func (r *sessionRepository) Revoke(ctx context.Context, id, userID string) error {
	result := r.db.WithContext(ctx).Model(&model.Session{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", id, userID).
		UpdateColumn("revoked_at", time.Now().UTC())
	if result.Error != nil {
		return fmt.Errorf("mencabut sesi: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *sessionRepository) RevokeAll(ctx context.Context, userID, scope string) error {
	query := r.db.WithContext(ctx).Model(&model.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID)
	if scope != "" {
		query = query.Where("scope = ?", scope)
	}

	if err := query.UpdateColumn("revoked_at", time.Now().UTC()).Error; err != nil {
		return fmt.Errorf("mencabut semua sesi: %w", err)
	}
	return nil
}

// SessionTokenID menghasilkan penanda sesi dari token mentah.
// Yang disimpan adalah hash, sehingga bocornya tabel tidak memberi token sah.
func SessionTokenID(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// NewTokenID membuat penanda sesi acak tanpa menyimpan token asli.
func NewTokenID() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("membuat penanda sesi: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
