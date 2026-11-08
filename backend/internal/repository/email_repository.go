package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// EmailRepository menyimpan antrean surel dan token sekali pakai.
type EmailRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewEmailRepository merangkai repository surel.
func NewEmailRepository(db *gorm.DB) *EmailRepository {
	return &EmailRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// Enqueue menambahkan satu surel ke antrean.
func (r *EmailRepository) Enqueue(ctx context.Context, row *model.EmailOutbox) error {
	if row.ScheduledAt.IsZero() {
		row.ScheduledAt = r.now()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = r.now()
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return fmt.Errorf("menambah antrean surel: %w", err)
	}
	return nil
}

// Pending mengambil surel yang siap dikirim.
// Status gagal tidak diambil lagi agar tidak berputar selamanya.
func (r *EmailRepository) Pending(ctx context.Context, limit int) ([]model.EmailOutbox, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var rows []model.EmailOutbox
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ?", model.EmailPending, r.now()).
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca antrean surel: %w", err)
	}
	return rows, nil
}

// MarkSent menandai surel terkirim.
func (r *EmailRepository) MarkSent(ctx context.Context, id string) error {
	now := r.now()
	err := r.db.WithContext(ctx).Model(&model.EmailOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": model.EmailSent, "sent_at": now, "attempts": gorm.Expr("attempts + 1")}).Error
	if err != nil {
		return fmt.Errorf("menandai surel terkirim: %w", err)
	}
	return nil
}

// MarkFailed menandai percobaan gagal. Pesan tetap di antrean selama
// percobaan belum melewati batas.
func (r *EmailRepository) MarkFailed(ctx context.Context, id string, reason string, retryAt time.Time, final bool) error {
	status := model.EmailPending
	if final {
		status = model.EmailFailed
	}
	err := r.db.WithContext(ctx).Model(&model.EmailOutbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       status,
			"last_error":   trimReason(reason),
			"scheduled_at": retryAt,
			"attempts":     gorm.Expr("attempts + 1"),
		}).Error
	if err != nil {
		return fmt.Errorf("menandai surel gagal: %w", err)
	}
	return nil
}

// CreateToken menyimpan token baru sekaligus menghanguskan token sejenis
// yang belum dipakai, sehingga hanya satu tautan yang berlaku.
func (r *EmailRepository) CreateToken(ctx context.Context, token *model.EmailToken) error {
	if token.CreatedAt.IsZero() {
		token.CreatedAt = r.now()
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.EmailToken{}).
			Where("user_id = ? AND purpose = ? AND used_at IS NULL", token.UserID, token.Purpose).
			Update("used_at", token.CreatedAt).Error; err != nil {
			return err
		}
		return tx.Create(token).Error
	})
	if err != nil {
		return fmt.Errorf("menyimpan token surel: %w", err)
	}
	return nil
}

// FindToken membaca token berdasarkan hash-nya.
func (r *EmailRepository) FindToken(ctx context.Context, hash, purpose string) (*model.EmailToken, error) {
	var row model.EmailToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND purpose = ?", hash, purpose).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca token surel: %w", err)
	}
	return &row, nil
}

// ConsumeToken menandai token terpakai. Nilai kembali false berarti token
// sudah dipakai lebih dulu oleh permintaan lain (balapan).
func (r *EmailRepository) ConsumeToken(ctx context.Context, id string) (bool, error) {
	now := r.now()
	result := r.db.WithContext(ctx).Model(&model.EmailToken{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", now)
	if result.Error != nil {
		return false, fmt.Errorf("memakai token surel: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

// CountTokensSince menghitung token terbit pada jendela waktu tertentu.
// Dipakai membatasi permintaan ulang (plans/17 §3.2).
func (r *EmailRepository) CountTokensSince(ctx context.Context, userID, purpose string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.EmailToken{}).
		Where("user_id = ? AND purpose = ? AND created_at >= ?", userID, purpose, since).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung token surel: %w", err)
	}
	return count, nil
}

// MarkEmailVerified menandai alamat surel pengguna terverifikasi.
func (r *EmailRepository) MarkEmailVerified(ctx context.Context, userID string) error {
	now := r.now()
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{"email_verified_at": now}).Error
	if err != nil {
		return fmt.Errorf("menandai email terverifikasi: %w", err)
	}
	return nil
}

func trimReason(reason string) string {
	if len(reason) > 250 {
		return reason[:250]
	}
	return reason
}
