package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// BugReportRepository menyimpan laporan bug pengguna dan lampirannya.
type BugReportRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewBugReportRepository merangkai repository laporan bug.
func NewBugReportRepository(db *gorm.DB) *BugReportRepository {
	return &BugReportRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// Create menyimpan laporan baru.
func (r *BugReportRepository) Create(ctx context.Context, report *model.BugReport) error {
	if err := r.db.WithContext(ctx).Create(report).Error; err != nil {
		return fmt.Errorf("menyimpan laporan bug: %w", err)
	}
	return nil
}

// AddAttachment menyimpan satu lampiran.
func (r *BugReportRepository) AddAttachment(ctx context.Context, att *model.BugReportAttachment) error {
	if err := r.db.WithContext(ctx).Create(att).Error; err != nil {
		return fmt.Errorf("menyimpan lampiran: %w", err)
	}
	return nil
}

// FindByID membaca satu laporan.
func (r *BugReportRepository) FindByID(ctx context.Context, id string) (*model.BugReport, error) {
	var row model.BugReport
	if err := r.db.WithContext(ctx).Where("id = ?", id).Limit(1).Find(&row).Error; err != nil {
		return nil, fmt.Errorf("membaca laporan bug: %w", err)
	}
	if row.ID == "" {
		return nil, apperror.ErrNotFound
	}
	return &row, nil
}

// ListByUser membaca laporan milik satu pengguna.
func (r *BugReportRepository) ListByUser(ctx context.Context, userID string, limit int) ([]model.BugReport, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []model.BugReport
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca laporan pengguna: %w", err)
	}
	return rows, nil
}

// List membaca laporan untuk panel admin dengan penyaring status dan kategori.
func (r *BugReportRepository) List(ctx context.Context, status, category string, limit int) ([]model.BugReport, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	var rows []model.BugReport
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca daftar laporan: %w", err)
	}
	return rows, nil
}

// AttachmentsOf membaca seluruh lampiran satu laporan.
func (r *BugReportRepository) AttachmentsOf(ctx context.Context, reportID string) ([]model.BugReportAttachment, error) {
	var rows []model.BugReportAttachment
	if err := r.db.WithContext(ctx).
		Where("report_id = ?", reportID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca lampiran: %w", err)
	}
	return rows, nil
}

// FindAttachment membaca satu lampiran berdasarkan id dan laporan induknya.
//
// Dua kunci sekaligus supaya lampiran tidak bisa dijangkau lewat id laporan
// milik orang lain.
func (r *BugReportRepository) FindAttachment(ctx context.Context, reportID, attachmentID string) (*model.BugReportAttachment, error) {
	var row model.BugReportAttachment
	if err := r.db.WithContext(ctx).
		Where("id = ? AND report_id = ?", attachmentID, reportID).Limit(1).Find(&row).Error; err != nil {
		return nil, fmt.Errorf("membaca lampiran: %w", err)
	}
	if row.ID == "" {
		return nil, apperror.ErrNotFound
	}
	return &row, nil
}

// UpdateStatus mengubah status dan catatan admin satu laporan.
func (r *BugReportRepository) UpdateStatus(ctx context.Context, id, status, adminNote string) error {
	now := r.now()
	result := r.db.WithContext(ctx).Model(&model.BugReport{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "admin_note": adminNote, "updated_at": now})
	if result.Error != nil {
		return fmt.Errorf("memperbarui laporan: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
