package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/internal/pagecontent"
)

// SeedPages mengisi isi awal halaman publik yang belum ada sama sekali.
//
// Halaman yang sudah tersimpan tidak disentuh: setelah admin menyunting isi,
// migrasi ulang tidak boleh menimpanya dengan teks awal. Dengan begitu seed
// aman dijalankan setiap kali migrasi berjalan.
func SeedPages(ctx context.Context, db *gorm.DB, seeds []pagecontent.Seed, at time.Time) (int, error) {
	seeded := 0

	for _, seed := range seeds {
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var existing int64
			if err := tx.Model(&model.Page{}).Where("slug = ?", seed.Slug).Count(&existing).Error; err != nil {
				return fmt.Errorf("memeriksa halaman %s: %w", seed.Slug, err)
			}
			if existing > 0 {
				return nil
			}

			page := &model.Page{
				Slug:            seed.Slug,
				Eyebrow:         seed.Eyebrow,
				Title:           seed.Title,
				Lead:            seed.Lead,
				MetaTitle:       seed.MetaTitle,
				MetaDescription: seed.MetaDescription,
				IsPublished:     true,
				UpdatedBy:       "system",
				Base:            model.Base{CreatedAt: at, UpdatedAt: at},
			}
			if err := tx.Create(page).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return nil
				}
				return fmt.Errorf("menyimpan halaman %s: %w", seed.Slug, err)
			}

			for index, sectionSeed := range seed.Sections {
				section := &model.PageSection{
					PageSlug:    seed.Slug,
					Key:         sectionSeed.Key,
					Kind:        sectionSeed.Kind,
					Eyebrow:     sectionSeed.Eyebrow,
					Title:       sectionSeed.Title,
					Lead:        sectionSeed.Lead,
					Body:        sectionSeed.Body,
					Layout:      sectionSeed.Layout,
					SortOrder:   index + 1,
					IsPublished: true,
				}
				if err := tx.Create(section).Error; err != nil {
					return fmt.Errorf("menyimpan bagian %s/%s: %w", seed.Slug, sectionSeed.Key, err)
				}

				for itemIndex, itemSeed := range sectionSeed.Items {
					item := &model.PageItem{
						SectionID: section.ID,
						Label:     itemSeed.Label,
						Value:     itemSeed.Value,
						Note:      itemSeed.Note,
						Icon:      itemSeed.Icon,
						Href:      itemSeed.Href,
						SortOrder: itemIndex + 1,
					}
					if err := tx.Create(item).Error; err != nil {
						return fmt.Errorf("menyimpan butir %s/%s: %w", seed.Slug, sectionSeed.Key, err)
					}
				}
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
