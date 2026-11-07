package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
)

// NotificationRepository mencegah satu notifikasi terkirim berulang.
type NotificationRepository interface {
	ClaimMessage(ctx context.Context, userID, channel, messageKey string) (bool, error)
	MarkMessageSent(ctx context.Context, messageKey string) error
	MarkMessageFailed(ctx context.Context, messageKey, reason string) error
}

type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository membuat repository antrean notifikasi.
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) ClaimMessage(ctx context.Context, userID, channel, messageKey string) (bool, error) {
	entry := &model.NotificationLog{
		UserID:     userID,
		Channel:    channel,
		MessageKey: messageKey,
		Status:     "queued",
		CreatedAt:  time.Now().UTC(),
	}

	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(entry)
	if res.Error != nil {
		return false, fmt.Errorf("mengklaim notifikasi: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

func (r *notificationRepository) MarkMessageSent(ctx context.Context, messageKey string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Where("message_key = ?", messageKey).
		Updates(map[string]any{
			"status":   "sent",
			"sent_at":  now,
			"attempts": gorm.Expr("attempts + 1"),
		})
	if res.Error != nil {
		return fmt.Errorf("menandai notifikasi terkirim: %w", res.Error)
	}
	return nil
}

func (r *notificationRepository) MarkMessageFailed(ctx context.Context, messageKey, reason string) error {
	res := r.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Where("message_key = ?", messageKey).
		Updates(map[string]any{
			"status":     "failed",
			"last_error": truncateText(reason, 255),
			"attempts":   gorm.Expr("attempts + 1"),
		})
	if res.Error != nil {
		return fmt.Errorf("menandai notifikasi gagal: %w", res.Error)
	}
	return nil
}

func truncateText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}
