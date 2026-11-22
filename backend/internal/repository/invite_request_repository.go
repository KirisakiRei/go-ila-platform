package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// InviteRequestRepository menyimpan permohonan akses dari beranda publik.
type InviteRequestRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewInviteRequestRepository merangkai repository permohonan undangan.
func NewInviteRequestRepository(db *gorm.DB) *InviteRequestRepository {
	return &InviteRequestRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// Create menyimpan permohonan baru.
func (r *InviteRequestRepository) Create(ctx context.Context, req *model.InviteRequest) error {
	if err := r.db.WithContext(ctx).Create(req).Error; err != nil {
		return fmt.Errorf("menyimpan permohonan undangan: %w", err)
	}
	return nil
}

// FindByID membaca satu permohonan.
func (r *InviteRequestRepository) FindByID(ctx context.Context, id string) (*model.InviteRequest, error) {
	var row model.InviteRequest
	if err := r.db.WithContext(ctx).Where("id = ?", id).Limit(1).Find(&row).Error; err != nil {
		return nil, fmt.Errorf("membaca permohonan undangan: %w", err)
	}
	if row.ID == "" {
		return nil, apperror.ErrNotFound
	}
	return &row, nil
}

// HasPendingByEmail memeriksa apakah alamat surel masih punya permohonan pending.
func (r *InviteRequestRepository) HasPendingByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.InviteRequest{}).
		Where("email = ? AND status = ?", email, model.InviteRequestPending).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("memeriksa permohonan duplikat: %w", err)
	}
	return count > 0, nil
}

// HasRecentByEmail memeriksa apakah alamat surel pernah mengajukan sejak waktu `since`,
// guna mencegah spam ulang setelah permohonan ditolak atau diproses.
func (r *InviteRequestRepository) HasRecentByEmail(ctx context.Context, email string, since time.Time) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.InviteRequest{}).
		Where("email = ? AND created_at >= ?", email, since).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("memeriksa jeda permohonan: %w", err)
	}
	return count > 0, nil
}

// List membaca permohonan secara berpaginasi, terbaru lebih dulu.
func (r *InviteRequestRepository) List(ctx context.Context, status string, page, perPage int) ([]model.InviteRequest, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	query := r.db.WithContext(ctx).Model(&model.InviteRequest{})
	switch status {
	case model.InviteRequestPending, model.InviteRequestApproved, model.InviteRequestRejected:
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("menghitung permohonan undangan: %w", err)
	}

	var rows []model.InviteRequest
	offset := (page - 1) * perPage
	if err := query.Order("created_at DESC").Offset(offset).Limit(perPage).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("membaca daftar permohonan undangan: %w", err)
	}
	return rows, total, nil
}

// CountPending menghitung jumlah permohonan yang masih menunggu tinjauan admin.
func (r *InviteRequestRepository) CountPending(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.InviteRequest{}).
		Where("status = ?", model.InviteRequestPending).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("menghitung permohonan pending: %w", err)
	}
	return count, nil
}

// MarkApproved menandai permohonan sudah disetujui dan tertaut ke undangan yang dibuat.
// Kondisi status = 'pending' di WHERE mencegah dua admin menyetujui baris sama sekaligus.
func (r *InviteRequestRepository) MarkApproved(ctx context.Context, id, inviteID, actorID, note string, reviewedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.InviteRequest{}).
		Where("id = ? AND status = ?", id, model.InviteRequestPending).
		Updates(map[string]any{
			"status":      model.InviteRequestApproved,
			"invite_id":   inviteID,
			"reviewed_by": actorID,
			"reviewed_at": reviewedAt,
			"review_note": note,
			"updated_at":  reviewedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("memperbarui status permohonan: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrConflict
	}
	return nil
}

// MarkRejected menandai permohonan ditolak.
func (r *InviteRequestRepository) MarkRejected(ctx context.Context, id, actorID, reason string, reviewedAt time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.InviteRequest{}).
		Where("id = ? AND status = ?", id, model.InviteRequestPending).
		Updates(map[string]any{
			"status":      model.InviteRequestRejected,
			"reviewed_by": actorID,
			"reviewed_at": reviewedAt,
			"review_note": reason,
			"updated_at":  reviewedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("menolak permohonan undangan: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrConflict
	}
	return nil
}
