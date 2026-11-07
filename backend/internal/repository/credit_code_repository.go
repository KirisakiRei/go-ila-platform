package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// creditCodeRepository mengelola voucher saldo.
type creditCodeRepository struct {
	db *gorm.DB
}

// NewCreditCodeRepository membuat data kode kredit.
func NewCreditCodeRepository(db *gorm.DB) *creditCodeRepository {
	return &creditCodeRepository{db: db}
}

// Redeem menukar kode kredit: saldo naik dan buku besar bertambah dalam satu
// transaksi. Penukaran ganda ditolak oleh indeks unik (code_id, user_id).
func (r *creditCodeRepository) Redeem(ctx context.Context, userID, code string, now time.Time) (int64, error) {
	var balance int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.CreditCode
		err := tx.Where("code = ?", code).Limit(1).Find(&row).Error
		if err != nil {
			return fmt.Errorf("membaca kode kredit: %w", err)
		}
		if row.ID == "" || !row.IsActive {
			return errors.Join(apperror.ErrNotFound, errors.New("kode kredit tidak valid"))
		}
		if now.Before(row.StartsAt) {
			return errors.Join(apperror.ErrBadRequest, errors.New("kode kredit belum berlaku"))
		}
		if now.After(row.EndsAt) {
			return errors.Join(apperror.ErrBadRequest, errors.New("kode kredit sudah kedaluwarsa"))
		}

		var used int64
		if err := tx.Model(&model.CreditRedemption{}).Where("code_id = ?", row.ID).Count(&used).Error; err != nil {
			return fmt.Errorf("menghitung pemakaian kode kredit: %w", err)
		}
		if row.MaxRedemptions > 0 && used >= int64(row.MaxRedemptions) {
			return errors.Join(apperror.ErrBadRequest, errors.New("kode kredit sudah habis"))
		}

		var perUser int64
		if err := tx.Model(&model.CreditRedemption{}).
			Where("code_id = ? AND user_id = ?", row.ID, userID).Count(&perUser).Error; err != nil {
			return fmt.Errorf("menghitung pemakaian kode kredit pengguna: %w", err)
		}
		if row.MaxPerUser > 0 && perUser >= int64(row.MaxPerUser) {
			return errors.Join(apperror.ErrBadRequest, errors.New("anda sudah memakai kode kredit ini"))
		}

		redemption := &model.CreditRedemption{
			ID: uuid.NewString(), CodeID: row.ID, UserID: userID,
			AmountIDR: row.AmountIDR, RedeemedAt: now,
		}
		if err := tx.Create(redemption).Error; err != nil {
			return errors.Join(apperror.ErrConflict, errors.New("kode kredit ini sudah Anda pakai"))
		}

		// Saldo dan buku besar ditulis oleh jalur yang sama dengan penyesuaian admin.
		after, err := NewPricingRepository(tx).ApplyCredit(ctx, &ApplyCreditInput{
			UserID: userID, DeltaIDR: row.AmountIDR, Kind: model.CreditKindCode,
			Reason: "penukaran kode " + row.Code,
		})
		if err != nil {
			return err
		}
		balance = after
		return nil
	})
	if err != nil {
		return 0, err
	}
	return balance, nil
}

// Create menyimpan kode kredit baru.
func (r *creditCodeRepository) Create(ctx context.Context, code *model.CreditCode) error {
	if code.ID == "" {
		code.ID = uuid.NewString()
	}
	if code.CreatedAt.IsZero() {
		code.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(code).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperror.ErrAlreadyExists
		}
		return fmt.Errorf("menyimpan kode kredit: %w", err)
	}
	return nil
}
