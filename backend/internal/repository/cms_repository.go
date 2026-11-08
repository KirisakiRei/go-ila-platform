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

// CMSRepository menyimpan FAQ dan pengumuman.
// Konten sengaja di database agar bisa diubah tanpa deploy (plans/11 J2).
type CMSRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewCMSRepository merangkai repository konten.
func NewCMSRepository(db *gorm.DB) *CMSRepository {
	return &CMSRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// FAQs membaca FAQ. `onlyActive` dipakai endpoint publik.
func (r *CMSRepository) FAQs(ctx context.Context, onlyActive bool) ([]model.Faq, error) {
	var rows []model.Faq
	query := r.db.WithContext(ctx)
	if onlyActive {
		query = query.Where("is_active")
	}
	if err := query.Order("category ASC, sort_order ASC, created_at ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca FAQ: %w", err)
	}
	return rows, nil
}

// FindFaq membaca satu FAQ berdasarkan ID.
func (r *CMSRepository) FindFaq(ctx context.Context, id string) (*model.Faq, error) {
	var row model.Faq
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca FAQ: %w", err)
	}
	return &row, nil
}

// SaveFaq menyimpan FAQ baru atau yang sudah ada.
func (r *CMSRepository) SaveFaq(ctx context.Context, faq *model.Faq) error {
	if err := r.db.WithContext(ctx).Save(faq).Error; err != nil {
		return fmt.Errorf("menyimpan FAQ: %w", err)
	}
	return nil
}

// DeleteFaq menghapus satu FAQ.
func (r *CMSRepository) DeleteFaq(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Faq{}).Error; err != nil {
		return fmt.Errorf("menghapus FAQ: %w", err)
	}
	return nil
}

// Announcements membaca pengumuman. `onlyLive` membatasi pada yang sedang
// tayang, sehingga pengumuman kedaluwarsa tidak perlu dihapus manual.
func (r *CMSRepository) Announcements(ctx context.Context, onlyLive bool) ([]model.Announcement, error) {
	var rows []model.Announcement
	query := r.db.WithContext(ctx)
	if onlyLive {
		now := r.now()
		query = query.Where("is_active AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)", now, now)
	}
	if err := query.Order("starts_at DESC").Limit(100).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca pengumuman: %w", err)
	}
	return rows, nil
}

// FindAnnouncement membaca satu pengumuman.
func (r *CMSRepository) FindAnnouncement(ctx context.Context, id string) (*model.Announcement, error) {
	var row model.Announcement
	if err := r.db.WithContext(ctx).Where("id = ?", id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca pengumuman: %w", err)
	}
	return &row, nil
}

// SaveAnnouncement menyimpan pengumuman baru atau yang sudah ada.
func (r *CMSRepository) SaveAnnouncement(ctx context.Context, row *model.Announcement) error {
	if err := r.db.WithContext(ctx).Save(row).Error; err != nil {
		return fmt.Errorf("menyimpan pengumuman: %w", err)
	}
	return nil
}

// DeleteAnnouncement menghapus satu pengumuman.
func (r *CMSRepository) DeleteAnnouncement(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Announcement{}).Error; err != nil {
		return fmt.Errorf("menghapus pengumuman: %w", err)
	}
	return nil
}
