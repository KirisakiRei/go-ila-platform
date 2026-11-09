package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

type brokerAccountRepository struct {
	db *gorm.DB
}

// NewBrokerAccountRepository membuat repository akun broker.
func NewBrokerAccountRepository(db *gorm.DB) BrokerAccountRepository {
	return &brokerAccountRepository{db: db}
}

func (r *brokerAccountRepository) Create(ctx context.Context, account *model.BrokerAccount) error {
	if err := r.db.WithContext(ctx).Create(account).Error; err != nil {
		return fmt.Errorf("membuat broker account: %w", err)
	}
	return nil
}

func (r *brokerAccountRepository) FindByID(ctx context.Context, id string) (*model.BrokerAccount, error) {
	var account model.BrokerAccount
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca broker account: %w", err)
	}
	return &account, nil
}

// ListByUser selalu dibatasi userID agar tidak ada akses lintas akun.
func (r *brokerAccountRepository) ListByUser(ctx context.Context, userID string) ([]model.BrokerAccount, error) {
	var accounts []model.BrokerAccount
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&accounts).Error
	if err != nil {
		return nil, fmt.Errorf("membaca broker account milik user: %w", err)
	}
	return accounts, nil
}

func (r *brokerAccountRepository) CountActiveByUser(ctx context.Context, userID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.BrokerAccount{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung broker account aktif: %w", err)
	}
	return total, nil
}

func (r *brokerAccountRepository) Update(ctx context.Context, account *model.BrokerAccount) error {
	if err := r.db.WithContext(ctx).Save(account).Error; err != nil {
		return fmt.Errorf("memperbarui broker account: %w", err)
	}
	return nil
}

type credentialRepository struct {
	db *gorm.DB
}

// NewCredentialRepository membuat repository blob kredensial terenkripsi.
func NewCredentialRepository(db *gorm.DB) CredentialRepository {
	return &credentialRepository{db: db}
}

// Upsert menulis atau mengganti blob kredensial untuk satu akun broker.
func (r *credentialRepository) Upsert(ctx context.Context, credential *model.Credential) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Credential
		findErr := tx.Where("broker_account_id = ?", credential.BrokerAccountID).First(&existing).Error
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return tx.Create(credential).Error
		}

		existing.UserKeyWrapped = credential.UserKeyWrapped
		existing.UserKeyNonce = credential.UserKeyNonce
		existing.Ciphertext = credential.Ciphertext
		existing.Nonce = credential.Nonce
		existing.KeyVersion = credential.KeyVersion
		existing.TokenFingerprint = credential.TokenFingerprint
		existing.RotatedAt = credential.RotatedAt
		return tx.Save(&existing).Error
	})
	if err != nil {
		return fmt.Errorf("menyimpan kredensial: %w", err)
	}
	return nil
}

func (r *credentialRepository) FindByAccount(ctx context.Context, accountID string) (*model.Credential, error) {
	var credential model.Credential
	err := r.db.WithContext(ctx).Where("broker_account_id = ?", accountID).First(&credential).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca kredensial: %w", err)
	}
	return &credential, nil
}

// DeleteByAccount menghapus permanen, bukan soft delete.
// Pencabutan kredensial harus benar-benar menghilangkan data.
func (r *credentialRepository) DeleteByAccount(ctx context.Context, accountID string) error {
	err := r.db.WithContext(ctx).Unscoped().
		Where("broker_account_id = ?", accountID).
		Delete(&model.Credential{}).Error
	if err != nil {
		return fmt.Errorf("menghapus kredensial: %w", err)
	}
	return nil
}

type subscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository membuat repository langganan simbol dan strategi.
func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(ctx context.Context, sub *model.Subscription) error {
	if err := r.db.WithContext(ctx).Create(sub).Error; err != nil {
		return fmt.Errorf("membuat subscription: %w", err)
	}
	return nil
}

func (r *subscriptionRepository) ListActiveByUser(ctx context.Context, userID string) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true).
		Order("symbol ASC").
		Find(&subs).Error
	if err != nil {
		return nil, fmt.Errorf("membaca subscription user: %w", err)
	}
	return subs, nil
}

// ListByUser mengembalikan seluruh langganan termasuk yang nonaktif,
// agar user melihat apa yang pernah dipilihnya.
func (r *subscriptionRepository) ListByUser(ctx context.Context, userID string) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("symbol ASC, strategy ASC").
		Find(&subs).Error
	if err != nil {
		return nil, fmt.Errorf("membaca subscription user: %w", err)
	}
	return subs, nil
}

// FindActive mencari langganan aktif untuk satu pasangan (symbol, strategy).
func (r *subscriptionRepository) FindActive(ctx context.Context, userID, symbol, strategy string) (*model.Subscription, error) {
	var sub model.Subscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND symbol = ? AND strategy = ? AND is_active = ?", userID, symbol, strategy, true).
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca subscription: %w", err)
	}
	return &sub, nil
}

// Deactivate hanya mengubah baris milik user sendiri.
func (r *subscriptionRepository) Deactivate(ctx context.Context, id, userID string) error {
	res := r.db.WithContext(ctx).Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_active", false)
	if res.Error != nil {
		return fmt.Errorf("menonaktifkan subscription: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// ListActiveByAccount membaca langganan aktif yang berlaku untuk satu akun.
//
// Aturan transisi: langganan lama tanpa broker_account_id dianggap berlaku
// untuk semua akun user. Ini menjaga data sebelum katalog pair tetap
// dieksekusi tanpa perlu backfill paksa.
func (r *subscriptionRepository) ListActiveByAccount(ctx context.Context, userID, accountID string) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ? AND (broker_account_id IS NULL OR broker_account_id = ?)", userID, true, accountID).
		Order("symbol ASC, strategy ASC").
		Find(&subs).Error
	if err != nil {
		return nil, fmt.Errorf("membaca subscription akun: %w", err)
	}
	return subs, nil
}

// ReplaceAccountPairs mengganti seluruh pilihan simbol satu akun secara atomik.
//
// Langganan lama yang tidak dipilih lagi dinonaktifkan (bukan dihapus) agar
// jejak historis tetap ada. Baris yang sudah ada diaktifkan kembali sehingga
// pencabutan tidak menghapus data. Strategi dipertahankan bila simbol sama.
func (r *subscriptionRepository) ReplaceAccountPairs(ctx context.Context, userID, accountID string, symbols []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		wanted := make(map[string]bool, len(symbols))
		for _, s := range symbols {
			wanted[s] = true
		}

		// Nonaktifkan langganan akun ini yang tidak lagi dipilih.
		var existing []model.Subscription
		if err := tx.Where("user_id = ? AND broker_account_id = ?", userID, accountID).Find(&existing).Error; err != nil {
			return fmt.Errorf("membaca subscription akun: %w", err)
		}

		byPairKey := make(map[string]*model.Subscription, len(existing))
		for i := range existing {
			byPairKey[strings.ToUpper(existing[i].Symbol)] = &existing[i]
		}

		for i := range existing {
			key := strings.ToUpper(existing[i].Symbol)
			if !wanted[key] && existing[i].IsActive {
				if err := tx.Model(&model.Subscription{}).Where("id = ?", existing[i].ID).Update("is_active", false).Error; err != nil {
					return fmt.Errorf("menonaktifkan subscription: %w", err)
				}
			}
		}

		// Aktifkan atau buat baris untuk simbol yang dipilih.
		for _, raw := range symbols {
			canonical := strings.ToUpper(raw)
			if row, ok := byPairKey[canonical]; ok {
				if row.IsActive {
					continue
				}
				if err := tx.Model(&model.Subscription{}).Where("id = ?", row.ID).Update("is_active", true).Error; err != nil {
					return fmt.Errorf("mengaktifkan subscription: %w", err)
				}
				continue
			}

			sub := &model.Subscription{
				UserID:          userID,
				Symbol:          canonical,
				IsActive:        true,
				BrokerAccountID: strPtr(accountID),
			}
			if err := tx.Create(sub).Error; err != nil {
				return fmt.Errorf("membuat subscription: %w", err)
			}
		}

		return nil
	})
}

// strPtr adalah pembantu kecil untuk kolom nullable.
func strPtr(v string) *string { return &v }
