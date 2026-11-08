package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/legal"
	"ila/backend/internal/model"
)

// SeedInitialDrafts menerbitkan versi 1 setiap dokumen legal yang belum pernah
// ada. Slug yang sudah punya versi apa pun dilewati, sehingga perubahan isi
// oleh admin tidak tertimpa saat migrasi dijalankan ulang.
func (r *legalRepository) SeedInitialDrafts(ctx context.Context, drafts []legal.Draft, at time.Time) (int, error) {
	seeded := 0

	for _, draft := range drafts {
		err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var existing int64
			if err := tx.Model(&model.LegalDocument{}).Where("slug = ?", draft.Slug).Count(&existing).Error; err != nil {
				return fmt.Errorf("memeriksa dokumen %s: %w", draft.Slug, err)
			}
			if existing > 0 {
				var v1 model.LegalDocument
				if err := tx.Where("slug = ? AND version = 1", draft.Slug).First(&v1).Error; err == nil {
					if strings.Contains(v1.BodyMD, "[nama badan usaha]") || strings.Contains(v1.BodyMD, "[tanggal berlaku]") || strings.TrimSpace(v1.BodyMD) == "" {
						_ = tx.Model(&v1).Updates(map[string]any{
							"body_md": draft.BodyMD,
							"title":   draft.Title,
						}).Error
					}
				}
				return nil
			}

			doc := &model.LegalDocument{
				Slug:            draft.Slug,
				Version:         1,
				Title:           draft.Title,
				BodyMD:          draft.BodyMD,
				RequireReaccept: draft.RequireReaccept,
				PublishedAt:     &at,
				EffectiveAt:     at,
				CreatedBy:       "system",
				CreatedAt:       at,
				UpdatedAt:       at,
			}
			if err := tx.Create(doc).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return nil
				}
				return fmt.Errorf("menerbitkan dokumen %s: %w", draft.Slug, err)
			}

			seeded++
			return nil
		})
		if err != nil {
			return seeded, err
		}
	}

	return seeded, nil
}
