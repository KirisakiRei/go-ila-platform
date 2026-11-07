package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// TwoFactorRepository menyimpan pengaturan TOTP, kode pemulihan, dan keadaan
// antara saat masuk dua langkah.
type TwoFactorRepository interface {
	FindByUser(ctx context.Context, userID string) (*model.TwoFactor, error)
	Upsert(ctx context.Context, settings *model.TwoFactor) error
	Deactivate(ctx context.Context, userID string) error
	MarkUsed(ctx context.Context, userID string, step int64, at time.Time) error

	ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error
	ListRecoveryCodes(ctx context.Context, userID string) ([]model.RecoveryCode, error)
	ConsumeRecoveryCode(ctx context.Context, userID, hash string, at time.Time) (bool, error)
	DeleteRecoveryCodes(ctx context.Context, userID string) error
	CountUnusedRecoveryCodes(ctx context.Context, userID string) (int64, error)

	CreateChallenge(ctx context.Context, challenge *model.TwoFactorChallenge) error
	FindChallenge(ctx context.Context, id string) (*model.TwoFactorChallenge, error)
	IncrementChallengeAttempts(ctx context.Context, id string) (int, error)
	ConsumeChallenge(ctx context.Context, id string, at time.Time) error
	DeleteExpiredChallenges(ctx context.Context, before time.Time) (int64, error)
}

type twoFactorRepository struct {
	db *gorm.DB
}

// NewTwoFactorRepository membuat repository 2FA di atas koneksi GORM.
func NewTwoFactorRepository(db *gorm.DB) TwoFactorRepository {
	return &twoFactorRepository{db: db}
}

func (r *twoFactorRepository) FindByUser(ctx context.Context, userID string) (*model.TwoFactor, error) {
	var settings model.TwoFactor
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Take(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca pengaturan 2FA: %w", err)
	}
	return &settings, nil
}

// Upsert menyimpan secret baru atau memperbarui yang sudah ada.
// ActivatedAt ikut ditulis karena aktivasi dan penyimpanan terjadi bersamaan.
func (r *twoFactorRepository) Upsert(ctx context.Context, settings *model.TwoFactor) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"secret_enc", "secret_version", "activated_at", "last_used_step", "last_used_at", "updated_at",
		}),
	}).Create(settings).Error
	if err != nil {
		return fmt.Errorf("menyimpan pengaturan 2FA: %w", err)
	}
	return nil
}

func (r *twoFactorRepository) Deactivate(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.TwoFactor{}).Error; err != nil {
		return fmt.Errorf("menonaktifkan 2FA: %w", err)
	}
	return nil
}

// MarkUsed mencatat periode TOTP yang sudah dipakai agar kode yang sama tidak
// bisa dipakai dua kali (replay).
func (r *twoFactorRepository) MarkUsed(ctx context.Context, userID string, step int64, at time.Time) error {
	err := r.db.WithContext(ctx).Model(&model.TwoFactor{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{"last_used_step": step, "last_used_at": at, "updated_at": at}).Error
	if err != nil {
		return fmt.Errorf("mencatat pemakaian kode 2FA: %w", err)
	}
	return nil
}

func (r *twoFactorRepository) ReplaceRecoveryCodes(ctx context.Context, userID string, hashes []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.RecoveryCode{}).Error; err != nil {
			return fmt.Errorf("membersihkan kode pemulihan: %w", err)
		}
		if len(hashes) == 0 {
			return nil
		}

		now := time.Now().UTC()
		rows := make([]model.RecoveryCode, 0, len(hashes))
		for _, hash := range hashes {
			rows = append(rows, model.RecoveryCode{UserID: userID, CodeHash: hash, CreatedAt: now})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return fmt.Errorf("menyimpan kode pemulihan: %w", err)
		}
		return nil
	})
}

func (r *twoFactorRepository) ListRecoveryCodes(ctx context.Context, userID string) ([]model.RecoveryCode, error) {
	var out []model.RecoveryCode
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&out).Error
	if err != nil {
		return nil, fmt.Errorf("membaca kode pemulihan: %w", err)
	}
	return out, nil
}

// ConsumeRecoveryCode menandai satu kode terpakai. Pemakaian dicatat sebagai
// syarat, sehingga dua permintaan bersamaan tidak bisa memakai kode yang sama.
func (r *twoFactorRepository) ConsumeRecoveryCode(ctx context.Context, userID, hash string, at time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.RecoveryCode{}).
		Where("user_id = ? AND code_hash = ? AND used_at IS NULL", userID, hash).
		UpdateColumn("used_at", at)
	if result.Error != nil {
		return false, fmt.Errorf("memakai kode pemulihan: %w", result.Error)
	}
	return result.RowsAffected > 0, nil
}

func (r *twoFactorRepository) DeleteRecoveryCodes(ctx context.Context, userID string) error {
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.RecoveryCode{}).Error; err != nil {
		return fmt.Errorf("menghapus kode pemulihan: %w", err)
	}
	return nil
}

func (r *twoFactorRepository) CountUnusedRecoveryCodes(ctx context.Context, userID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.RecoveryCode{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung kode pemulihan: %w", err)
	}
	return total, nil
}

func (r *twoFactorRepository) CreateChallenge(ctx context.Context, challenge *model.TwoFactorChallenge) error {
	if err := r.db.WithContext(ctx).Create(challenge).Error; err != nil {
		return fmt.Errorf("membuat tantangan 2FA: %w", err)
	}
	return nil
}

func (r *twoFactorRepository) FindChallenge(ctx context.Context, id string) (*model.TwoFactorChallenge, error) {
	var challenge model.TwoFactorChallenge
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&challenge).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca tantangan 2FA: %w", err)
	}
	return &challenge, nil
}

// IncrementChallengeAttempts menaikkan hitungan salah kode dan mengembalikan
// nilai terbarunya.
func (r *twoFactorRepository) IncrementChallengeAttempts(ctx context.Context, id string) (int, error) {
	result := r.db.WithContext(ctx).Model(&model.TwoFactorChallenge{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1"))
	if result.Error != nil {
		return 0, fmt.Errorf("mencatat percobaan 2FA: %w", result.Error)
	}

	challenge, err := r.FindChallenge(ctx, id)
	if err != nil {
		return 0, err
	}
	return challenge.Attempts, nil
}

func (r *twoFactorRepository) ConsumeChallenge(ctx context.Context, id string, at time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.TwoFactorChallenge{}).
		Where("id = ? AND consumed_at IS NULL", id).
		UpdateColumn("consumed_at", at)
	if result.Error != nil {
		return fmt.Errorf("menutup tantangan 2FA: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrConflict
	}
	return nil
}

// DeleteExpiredChallenges membersihkan tantangan lama agar tabel tidak
// menumpuk. Mengembalikan jumlah baris yang dihapus.
func (r *twoFactorRepository) DeleteExpiredChallenges(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Delete(&model.TwoFactorChallenge{})
	if result.Error != nil {
		return 0, fmt.Errorf("membersihkan tantangan 2FA kedaluwarsa: %w", result.Error)
	}
	return result.RowsAffected, nil
}
