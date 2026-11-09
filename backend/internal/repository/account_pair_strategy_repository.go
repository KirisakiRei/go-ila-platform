package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// AccountPairStrategyRepository menyimpan saklar metode per akun dan per pair.
type AccountPairStrategyRepository interface {
	// ListByUser mengambil seluruh saklar milik satu pengguna.
	ListByUser(ctx context.Context, userID string) ([]model.AccountPairStrategy, error)
	// ListByAccount mengambil saklar satu akun.
	ListByAccount(ctx context.Context, accountID string) ([]model.AccountPairStrategy, error)
	// Replace menimpa seluruh saklar satu akun untuk pair tertentu.
	Replace(ctx context.Context, accountID, userID, symbol string, ucapan map[string]bool) error
	// Ensure membuat saklar bawaan bila belum ada (idempoten).
	Ensure(ctx context.Context, rows []model.AccountPairStrategy) error
}

type accountPairStrategyRepository struct {
	db *gorm.DB
}

// NewAccountPairStrategyRepository merangkai penyimpanan saklar metode.
func NewAccountPairStrategyRepository(db *gorm.DB) AccountPairStrategyRepository {
	return &accountPairStrategyRepository{db: db}
}

func (r *accountPairStrategyRepository) ListByUser(ctx context.Context, userID string) ([]model.AccountPairStrategy, error) {
	var rows []model.AccountPairStrategy
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca saklar metode: %w", err)
	}
	return rows, nil
}

func (r *accountPairStrategyRepository) ListByAccount(ctx context.Context, accountID string) ([]model.AccountPairStrategy, error) {
	var rows []model.AccountPairStrategy
	if err := r.db.WithContext(ctx).
		Where("broker_account_id = ?", accountID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca saklar metode akun: %w", err)
	}
	return rows, nil
}

// Replace menulis ulang saklar satu akun untuk satu simbol dalam satu transaksi.
//
// Hanya metode yang dikirim yang disentuh; metode lain milik simbol berbeda
// tidak terpengaruh.
func (r *accountPairStrategyRepository) Replace(ctx context.Context, accountID, userID, symbol string, ucapan map[string]bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for strategy, enabled := range ucapan {
			var row model.AccountPairStrategy
			err := tx.Where("broker_account_id = ? AND symbol = ? AND strategy = ?",
				accountID, symbol, strategy).Take(&row).Error
			if err == nil {
				if err := tx.Model(&row).Update("is_enabled", enabled).Error; err != nil {
					return fmt.Errorf("memperbarui saklar %s: %w", strategy, err)
				}
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("membaca saklar %s: %w", strategy, err)
			}
			newRow := &model.AccountPairStrategy{
				BrokerAccountID: accountID,
				UserID:          userID,
				Symbol:          symbol,
				Strategy:        strategy,
				IsEnabled:       enabled,
			}
			if err := tx.Create(newRow).Error; err != nil {
				return fmt.Errorf("membuat saklar %s: %w", strategy, err)
			}
		}
		return nil
	})
}

// Ensure membuat baris yang belum ada tanpa mengubah yang sudah ada.
func (r *accountPairStrategyRepository) Ensure(ctx context.Context, rows []model.AccountPairStrategy) error {
	for i := range rows {
		var count int64
		if err := r.db.WithContext(ctx).Model(&model.AccountPairStrategy{}).
			Where("broker_account_id = ? AND symbol = ? AND strategy = ?",
				rows[i].BrokerAccountID, rows[i].Symbol, rows[i].Strategy).
			Count(&count).Error; err != nil {
			return fmt.Errorf("memeriksa saklar metode: %w", err)
		}
		if count > 0 {
			continue
		}
		if err := r.db.WithContext(ctx).Create(&rows[i]).Error; err != nil {
			return fmt.Errorf("menyiapkan saklar metode: %w", err)
		}
	}
	return nil
}
