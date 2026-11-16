package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// TelegramMessageRepository menyimpan jejak pesan bot yang masih bertombol.
type TelegramMessageRepository interface {
	Simpan(ctx context.Context, pesan *model.TelegramMessage) error
	// BelumDibersihkan mengembalikan pesan bertombol untuk satu sinyal.
	BelumDibersihkan(ctx context.Context, signalID string) ([]model.TelegramMessage, error)
	TandaiDibersihkan(ctx context.Context, id string, waktu time.Time) error
	// HapusLama membuang catatan yang sudah tidak diperlukan.
	HapusLama(ctx context.Context, sebelum time.Time) (int64, error)
}

type telegramMessageRepository struct {
	db *gorm.DB
}

// NewTelegramMessageRepository merangkai penyimpanan jejak pesan Telegram.
func NewTelegramMessageRepository(db *gorm.DB) TelegramMessageRepository {
	return &telegramMessageRepository{db: db}
}

func (r *telegramMessageRepository) Simpan(ctx context.Context, pesan *model.TelegramMessage) error {
	if err := r.db.WithContext(ctx).Create(pesan).Error; err != nil {
		return fmt.Errorf("menyimpan jejak pesan telegram: %w", err)
	}
	return nil
}

func (r *telegramMessageRepository) BelumDibersihkan(ctx context.Context, signalID string) ([]model.TelegramMessage, error) {
	var rows []model.TelegramMessage
	err := r.db.WithContext(ctx).
		Where("signal_id = ? AND cleared_at IS NULL", signalID).
		Limit(20).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca jejak pesan telegram: %w", err)
	}
	return rows, nil
}

func (r *telegramMessageRepository) TandaiDibersihkan(ctx context.Context, id string, waktu time.Time) error {
	err := r.db.WithContext(ctx).
		Model(&model.TelegramMessage{}).
		Where("id = ?", id).
		Update("cleared_at", waktu).Error
	if err != nil {
		return fmt.Errorf("menandai pesan telegram dibersihkan: %w", err)
	}
	return nil
}

func (r *telegramMessageRepository) HapusLama(ctx context.Context, sebelum time.Time) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("created_at < ?", sebelum).
		Delete(&model.TelegramMessage{})
	if res.Error != nil {
		return 0, fmt.Errorf("menghapus jejak pesan lama: %w", res.Error)
	}
	return res.RowsAffected, nil
}
