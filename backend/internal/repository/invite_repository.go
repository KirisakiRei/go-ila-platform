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

// InviteRepository menyimpan undangan dan catatan pemakaiannya.
type InviteRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewInviteRepository merangkai repository undangan.
func NewInviteRepository(db *gorm.DB) *InviteRepository {
	return &InviteRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// Create menyimpan undangan baru.
func (r *InviteRepository) Create(ctx context.Context, invite *model.InviteCode) error {
	if err := r.db.WithContext(ctx).Create(invite).Error; err != nil {
		return fmt.Errorf("membuat undangan: %w", err)
	}
	return nil
}

// FindByHash mencari undangan dari hash kode.
func (r *InviteRepository) FindByHash(ctx context.Context, hash string) (*model.InviteCode, error) {
	var row model.InviteCode
	err := r.db.WithContext(ctx).Where("code_hash = ?", hash).Limit(1).Find(&row).Error
	if err != nil {
		return nil, fmt.Errorf("membaca undangan: %w", err)
	}
	if row.ID == "" {
		return nil, apperror.ErrNotFound
	}
	return &row, nil
}

// List membaca undangan untuk panel admin, terbaru lebih dulu.
func (r *InviteRepository) List(ctx context.Context, status string, limit int) ([]model.InviteCode, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	switch status {
	case "aktif":
		query = query.Where("is_active = ? AND expires_at > ? AND used_count < max_uses", true, r.now())
	case "habis":
		query = query.Where("used_count >= max_uses")
	case "kedaluwarsa":
		query = query.Where("is_active = ? AND expires_at <= ?", true, r.now())
	case "dicabut":
		query = query.Where("is_active = ?", false)
	}
	var rows []model.InviteCode
	if err := query.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca daftar undangan: %w", err)
	}
	return rows, nil
}

// FindByID membaca satu undangan.
func (r *InviteRepository) FindByID(ctx context.Context, id string) (*model.InviteCode, error) {
	var row model.InviteCode
	if err := r.db.WithContext(ctx).Where("id = ?", id).Limit(1).Find(&row).Error; err != nil {
		return nil, fmt.Errorf("membaca undangan: %w", err)
	}
	if row.ID == "" {
		return nil, apperror.ErrNotFound
	}
	return &row, nil
}

// Revoke mencabut undangan. Baris tidak dihapus agar riwayat pemakaian utuh.
func (r *InviteRepository) Revoke(ctx context.Context, id string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.InviteCode{}).
		Where("id = ? AND is_active = ?", id, true).
		Updates(map[string]any{"is_active": false, "updated_at": now})
	if result.Error != nil {
		return fmt.Errorf("mencabut undangan: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// KlaimKuota memotong satu kuota secara atomik.
//
// Inilah titik yang menentukan berapa banyak akun boleh lahir dari satu undangan.
// Syarat kuota ada di WHERE, bukan di kode Go: dua permintaan yang datang
// bersamaan tidak bisa sama-sama lolos, karena database yang memutuskan.
//
// Mengembalikan ErrConflict bila kuota habis, undangan tidak aktif, atau sudah
// kedaluwarsa — ketiganya sengaja tidak dibedakan bagi pemanggil.
func (r *InviteRepository) KlaimKuota(ctx context.Context, id string, now time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.InviteCode{}).
		Where("id = ? AND is_active = ? AND expires_at > ? AND used_count < max_uses", id, true, now).
		Updates(map[string]any{
			"used_count": gorm.Expr("used_count + 1"),
			"updated_at": now,
		})
	if result.Error != nil {
		return fmt.Errorf("memakai kuota undangan: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrConflict
	}
	return nil
}

// KembalikanKuota mengembalikan satu kuota yang sudah dipotong tetapi gagal
// dipakai. Dipanggil saat pembuatan akun gagal, sehingga pendaftaran yang tidak
// jadi tidak menghabiskan jatah orang lain.
func (r *InviteRepository) KembalikanKuota(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Model(&model.InviteCode{}).
		Where("id = ? AND used_count > 0", id).
		Update("used_count", gorm.Expr("used_count - 1"))
	if result.Error != nil {
		return fmt.Errorf("mengembalikan kuota undangan: %w", result.Error)
	}
	return nil
}

// CatatPemakaian menyimpan hubungan akun dengan undangan yang dipakainya.
func (r *InviteRepository) CatatPemakaian(ctx context.Context, redemption *model.InviteRedemption) error {
	if err := r.db.WithContext(ctx).Create(redemption).Error; err != nil {
		return fmt.Errorf("mencatat pemakaian undangan: %w", err)
	}
	return nil
}

// ListPemakaian membaca siapa saja yang memakai satu undangan.
func (r *InviteRepository) ListPemakaian(ctx context.Context, inviteID string) ([]model.InviteRedemption, error) {
	var rows []model.InviteRedemption
	if err := r.db.WithContext(ctx).
		Where("invite_id = ?", inviteID).Order("used_at ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca pemakaian undangan: %w", err)
	}
	return rows, nil
}

// ErrUndanganDipakai dipakai saat satu akun mencoba memakai undangan kedua kali.
var ErrUndanganDipakai = errors.New("akun sudah memakai undangan")
