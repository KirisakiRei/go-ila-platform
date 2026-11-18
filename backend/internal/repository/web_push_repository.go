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

const (
	// MaxPushDevicesPerUser membatasi jumlah perangkat per akun agar tabel tetap
	// ramping dan pengiriman paralel selalu cepat.
	MaxPushDevicesPerUser = 10
)

// WebPushRepository mengelola penyimpanan kunci VAPID dan langganan perangkat.
type WebPushRepository interface {
	EnsureSchema(ctx context.Context) error
	GetVAPID(ctx context.Context) (*model.WebPushVAPID, error)
	SaveVAPIDIfAbsent(ctx context.Context, record *model.WebPushVAPID) (*model.WebPushVAPID, error)
	ListByUser(ctx context.Context, userID string, page, perPage int) ([]model.WebPushSubscription, int64, error)
	ListActiveByUser(ctx context.Context, userID string, limit int) ([]model.WebPushSubscription, error)
	CountByUser(ctx context.Context, userID string) (int64, error)
	Upsert(ctx context.Context, sub *model.WebPushSubscription) (*model.WebPushSubscription, error)
	UpdatePreferences(ctx context.Context, userID, id string, notifyEntry, notifyExit *bool) (*model.WebPushSubscription, error)
	DeleteByID(ctx context.Context, userID, id string) error
	DeleteExpiredByID(ctx context.Context, id string) error
	DeleteByEndpointHash(ctx context.Context, userID, endpointHash string) error
	TouchLastSent(ctx context.Context, id string, at time.Time) error
}

type webPushRepository struct {
	db *gorm.DB
}

// NewWebPushRepository membuat repository Web Push berbasis GORM.
func NewWebPushRepository(db *gorm.DB) WebPushRepository {
	return &webPushRepository{db: db}
}

func (r *webPushRepository) EnsureSchema(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&model.WebPushVAPID{}, &model.WebPushSubscription{})
}

func (r *webPushRepository) GetVAPID(ctx context.Context) (*model.WebPushVAPID, error) {
	var row model.WebPushVAPID
	err := r.db.WithContext(ctx).Where("id = ?", 1).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca kunci VAPID: %w", err)
	}
	return &row, nil
}

func (r *webPushRepository) SaveVAPIDIfAbsent(ctx context.Context, record *model.WebPushVAPID) (*model.WebPushVAPID, error) {
	record.ID = 1
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).
		Create(record)
	if res.Error != nil {
		return nil, fmt.Errorf("menyimpan kunci VAPID: %w", res.Error)
	}
	return r.GetVAPID(ctx)
}

func (r *webPushRepository) ListByUser(ctx context.Context, userID string, page, perPage int) ([]model.WebPushSubscription, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	var total int64
	if err := r.db.WithContext(ctx).Model(&model.WebPushSubscription{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("menghitung perangkat push: %w", err)
	}

	var items []model.WebPushSubscription
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("memuat perangkat push: %w", err)
	}

	return items, total, nil
}

func (r *webPushRepository) ListActiveByUser(ctx context.Context, userID string, limit int) ([]model.WebPushSubscription, error) {
	if limit <= 0 || limit > MaxPushDevicesPerUser {
		limit = MaxPushDevicesPerUser
	}
	var items []model.WebPushSubscription
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND (notify_entry = ? OR notify_exit = ?)", userID, true, true).
		Order("updated_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("memuat perangkat aktif push: %w", err)
	}
	return items, nil
}

func (r *webPushRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&model.WebPushSubscription{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *webPushRepository) Upsert(ctx context.Context, sub *model.WebPushSubscription) (*model.WebPushSubscription, error) {
	var saved model.WebPushSubscription

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.WebPushSubscription
		findErr := tx.Where("endpoint_hash = ?", sub.EndpointHash).First(&existing).Error
		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		now := time.Now().UTC()
		if findErr == nil {
			updates := map[string]any{
				"user_id":      sub.UserID,
				"endpoint":     sub.Endpoint,
				"p256dh":       sub.P256dh,
				"auth":         sub.Auth,
				"device_label": sub.DeviceLabel,
				"notify_entry": sub.NotifyEntry,
				"notify_exit":  sub.NotifyExit,
				"updated_at":   now,
			}
			if err := tx.Model(&model.WebPushSubscription{}).
				Where("id = ?", existing.ID).
				Updates(updates).Error; err != nil {
				return err
			}
			return tx.Where("id = ?", existing.ID).First(&saved).Error
		}

		sub.CreatedAt = now
		sub.UpdatedAt = now
		if err := tx.Create(sub).Error; err != nil {
			return err
		}
		saved = *sub

		// Pangkas perangkat paling lama bila melebihi batas per akun.
		var ids []string
		if err := tx.Model(&model.WebPushSubscription{}).
			Where("user_id = ?", sub.UserID).
			Order("updated_at DESC").
			Offset(MaxPushDevicesPerUser).
			Limit(50).
			Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) > 0 {
			if err := tx.Where("id IN ?", ids).Delete(&model.WebPushSubscription{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("menyimpan langganan Web Push: %w", err)
	}
	return &saved, nil
}

func (r *webPushRepository) UpdatePreferences(ctx context.Context, userID, id string, notifyEntry, notifyExit *bool) (*model.WebPushSubscription, error) {
	var sub model.WebPushSubscription
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, err
	}

	updates := map[string]any{
		"updated_at": time.Now().UTC(),
	}
	if notifyEntry != nil {
		updates["notify_entry"] = *notifyEntry
	}
	if notifyExit != nil {
		updates["notify_exit"] = *notifyExit
	}

	if err := r.db.WithContext(ctx).Model(&model.WebPushSubscription{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *webPushRepository) DeleteByID(ctx context.Context, userID, id string) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.WebPushSubscription{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *webPushRepository) DeleteExpiredByID(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.WebPushSubscription{}).Error
}

func (r *webPushRepository) DeleteByEndpointHash(ctx context.Context, userID, endpointHash string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND endpoint_hash = ?", userID, endpointHash).
		Delete(&model.WebPushSubscription{}).Error
}

func (r *webPushRepository) TouchLastSent(ctx context.Context, id string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.WebPushSubscription{}).
		Where("id = ?", id).
		Update("last_sent_at", at.UTC()).Error
}
