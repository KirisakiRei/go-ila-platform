package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/legal"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// LegalRepository menyimpan dokumen legal berversi dan bukti persetujuannya.
type LegalRepository interface {
	// Terbit
	LatestPublished(ctx context.Context, slug string) (*model.LegalDocument, error)
	ListPublished(ctx context.Context) ([]model.LegalDocument, error)
	FindByID(ctx context.Context, id string) (*model.LegalDocument, error)
	ListAll(ctx context.Context, slug string) ([]model.LegalDocument, error)

	// Pengelolaan
	CreateDraft(ctx context.Context, doc *model.LegalDocument) error
	UpdateDraft(ctx context.Context, doc *model.LegalDocument) error
	Publish(ctx context.Context, id string, at time.Time) error
	NextVersion(ctx context.Context, slug string) (int, error)

	// Persetujuan
	RecordAcceptance(ctx context.Context, entry *model.LegalAcceptance) error
	AcceptancesOfUser(ctx context.Context, userID string) ([]model.LegalAcceptance, error)
	LatestAcceptance(ctx context.Context, userID, slug string) (*model.LegalAcceptance, error)
	CountAcceptances(ctx context.Context, slug string, version int) (int64, error)

	// SeedInitialDrafts menerbitkan versi 1 dokumen yang belum pernah ada.
	SeedInitialDrafts(ctx context.Context, drafts []legal.Draft, at time.Time) (int, error)
}

type legalRepository struct {
	db *gorm.DB
}

// NewLegalRepository membuat repository dokumen legal di atas koneksi GORM.
func NewLegalRepository(db *gorm.DB) LegalRepository {
	return &legalRepository{db: db}
}

// LatestPublished membaca versi terbit terbaru sebuah slug. Versi yang
// dijadwalkan berlaku di masa depan belum dianggap berlaku.
func (r *legalRepository) LatestPublished(ctx context.Context, slug string) (*model.LegalDocument, error) {
	var doc model.LegalDocument
	err := r.db.WithContext(ctx).
		Where("slug = ? AND published_at IS NOT NULL AND effective_at <= ?", slug, time.Now().UTC()).
		Order("version DESC").
		Take(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca dokumen legal: %w", err)
	}
	return &doc, nil
}

// ListPublished membaca satu versi terbit terbaru untuk setiap slug.
func (r *legalRepository) ListPublished(ctx context.Context) ([]model.LegalDocument, error) {
	var docs []model.LegalDocument
	err := r.db.WithContext(ctx).
		Where("published_at IS NOT NULL AND effective_at <= ?", time.Now().UTC()).
		Order("slug ASC, version DESC").
		Find(&docs).Error
	if err != nil {
		return nil, fmt.Errorf("membaca daftar dokumen legal: %w", err)
	}

	// Ambil hanya versi tertinggi per slug.
	seen := make(map[string]struct{}, len(docs))
	out := make([]model.LegalDocument, 0, len(docs))
	for _, doc := range docs {
		if _, ok := seen[doc.Slug]; ok {
			continue
		}
		seen[doc.Slug] = struct{}{}
		out = append(out, doc)
	}
	return out, nil
}

func (r *legalRepository) FindByID(ctx context.Context, id string) (*model.LegalDocument, error) {
	var doc model.LegalDocument
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca dokumen legal: %w", err)
	}
	return &doc, nil
}

// ListAll membaca seluruh versi, termasuk draf, untuk tampilan admin.
// slug kosong berarti semua jenis dokumen.
func (r *legalRepository) ListAll(ctx context.Context, slug string) ([]model.LegalDocument, error) {
	query := r.db.WithContext(ctx).Order("slug ASC, version DESC")
	if slug != "" {
		query = query.Where("slug = ?", slug)
	}

	var docs []model.LegalDocument
	if err := query.Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("membaca seluruh dokumen legal: %w", err)
	}
	return docs, nil
}

func (r *legalRepository) CreateDraft(ctx context.Context, doc *model.LegalDocument) error {
	if err := r.db.WithContext(ctx).Create(doc).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return apperror.ErrAlreadyExists
		}
		return fmt.Errorf("membuat draf dokumen legal: %w", err)
	}
	return nil
}

// UpdateDraft hanya menyentuh dokumen yang belum terbit. Dokumen terbit tidak
// pernah berubah: perubahan berarti versi baru (plans/09 §2.1).
func (r *legalRepository) UpdateDraft(ctx context.Context, doc *model.LegalDocument) error {
	result := r.db.WithContext(ctx).Model(&model.LegalDocument{}).
		Where("id = ? AND published_at IS NULL", doc.ID).
		Updates(map[string]any{
			"title":            doc.Title,
			"body_md":          doc.BodyMD,
			"require_reaccept": doc.RequireReaccept,
			"effective_at":     doc.EffectiveAt,
			"updated_at":       doc.UpdatedAt,
		})
	if result.Error != nil {
		return fmt.Errorf("memperbarui draf dokumen legal: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrConflict
	}
	return nil
}

func (r *legalRepository) Publish(ctx context.Context, id string, at time.Time) error {
	result := r.db.WithContext(ctx).Model(&model.LegalDocument{}).
		Where("id = ? AND published_at IS NULL", id).
		UpdateColumn("published_at", at)
	if result.Error != nil {
		return fmt.Errorf("menerbitkan dokumen legal: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrConflict
	}
	return nil
}

// NextVersion menghitung nomor versi berikutnya untuk sebuah slug.
func (r *legalRepository) NextVersion(ctx context.Context, slug string) (int, error) {
	var highest int
	err := r.db.WithContext(ctx).Model(&model.LegalDocument{}).
		Where("slug = ?", slug).
		Select("COALESCE(MAX(version), 0)").
		Scan(&highest).Error
	if err != nil {
		return 0, fmt.Errorf("membaca versi tertinggi: %w", err)
	}
	return highest + 1, nil
}

func (r *legalRepository) RecordAcceptance(ctx context.Context, entry *model.LegalAcceptance) error {
	if err := r.db.WithContext(ctx).Create(entry).Error; err != nil {
		return fmt.Errorf("menyimpan persetujuan legal: %w", err)
	}
	return nil
}

func (r *legalRepository) AcceptancesOfUser(ctx context.Context, userID string) ([]model.LegalAcceptance, error) {
	var out []model.LegalAcceptance
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("accepted_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, fmt.Errorf("membaca persetujuan legal: %w", err)
	}
	return out, nil
}

// LatestAcceptance membaca persetujuan terakhir sebuah akun untuk satu slug.
func (r *legalRepository) LatestAcceptance(ctx context.Context, userID, slug string) (*model.LegalAcceptance, error) {
	var entry model.LegalAcceptance
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND document_slug = ?", userID, slug).
		Order("version DESC, accepted_at DESC").
		Take(&entry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca persetujuan legal: %w", err)
	}
	return &entry, nil
}

func (r *legalRepository) CountAcceptances(ctx context.Context, slug string, version int) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.LegalAcceptance{}).
		Where("document_slug = ? AND version = ?", slug, version).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung persetujuan legal: %w", err)
	}
	return total, nil
}
