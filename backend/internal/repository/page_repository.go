package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// PageRepository menyimpan halaman publik beserta bagian dan butirnya.
//
// Ketiga tabel selalu dibaca dan ditulis bersama, jadi repository ini sengaja
// menyatukannya: tidak ada pemanggil yang butuh bagian tanpa halamannya.
type PageRepository struct {
	db *gorm.DB
}

// NewPageRepository merangkai repository halaman.
func NewPageRepository(db *gorm.DB) *PageRepository {
	return &PageRepository{db: db}
}

// PageBundle adalah satu halaman lengkap dengan bagian dan butirnya.
type PageBundle struct {
	Page     model.Page
	Sections []model.PageSection
	Items    []model.PageItem
}

// Find memuat satu halaman berdasarkan slug.
func (r *PageRepository) Find(ctx context.Context, slug string) (*PageBundle, error) {
	var page model.Page
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).Take(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca halaman %s: %w", slug, err)
	}
	return r.load(ctx, page)
}

// List memuat seluruh halaman. `onlyPublished` dipakai endpoint publik.
func (r *PageRepository) List(ctx context.Context, onlyPublished bool) ([]PageBundle, error) {
	var pages []model.Page
	query := r.db.WithContext(ctx)
	if onlyPublished {
		query = query.Where("is_published")
	}
	if err := query.Order("slug ASC").Find(&pages).Error; err != nil {
		return nil, fmt.Errorf("membaca daftar halaman: %w", err)
	}

	out := make([]PageBundle, 0, len(pages))
	for _, page := range pages {
		bundle, err := r.load(ctx, page)
		if err != nil {
			return nil, err
		}
		out = append(out, *bundle)
	}
	return out, nil
}

// load melengkapi satu halaman dengan bagian dan butirnya.
func (r *PageRepository) load(ctx context.Context, page model.Page) (*PageBundle, error) {
	var sections []model.PageSection
	if err := r.db.WithContext(ctx).
		Where("page_slug = ?", page.Slug).
		Order("sort_order ASC, created_at ASC").
		Find(&sections).Error; err != nil {
		return nil, fmt.Errorf("membaca bagian halaman %s: %w", page.Slug, err)
	}

	items := []model.PageItem{}
	if len(sections) > 0 {
		ids := make([]string, 0, len(sections))
		for _, section := range sections {
			ids = append(ids, section.ID)
		}
		if err := r.db.WithContext(ctx).
			Where("section_id IN ?", ids).
			Order("sort_order ASC, created_at ASC").
			Find(&items).Error; err != nil {
			return nil, fmt.Errorf("membaca butir halaman %s: %w", page.Slug, err)
		}
	}

	return &PageBundle{Page: page, Sections: sections, Items: items}, nil
}

// Save mengganti seluruh isi satu halaman dalam satu transaksi.
//
// Bagian dan butir lama dihapus lalu ditulis ulang dari keadaan yang dikirim
// panel admin. Pendekatan ini menjaga hasil akhirnya sama persis dengan yang
// dilihat admin di layarnya, termasuk urutan dan penghapusan.
func (r *PageRepository) Save(ctx context.Context, bundle *PageBundle) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		page := bundle.Page
		if err := tx.Save(&page).Error; err != nil {
			return fmt.Errorf("menyimpan halaman %s: %w", page.Slug, err)
		}

		if err := tx.Where("page_slug = ?", page.Slug).Delete(&model.PageSection{}).Error; err != nil {
			return fmt.Errorf("membersihkan bagian halaman %s: %w", page.Slug, err)
		}
		for index := range bundle.Sections {
			section := &bundle.Sections[index]
			// ID dikosongkan agar dibuat ulang: bagian lama sudah dihapus, dan
			// ID lama tidak boleh dipakai ulang karena butirnya ikut terhapus.
			section.ID = ""
			section.PageSlug = page.Slug
			if err := tx.Create(section).Error; err != nil {
				return fmt.Errorf("menyimpan bagian %s: %w", section.Key, err)
			}
		}

		// Service menautkan butir lewat kunci bagian; setelah bagian punya ID
		// barulah butir bisa ditulis dengan tautan yang sah.
		idByKey := make(map[string]string, len(bundle.Sections))
		for _, section := range bundle.Sections {
			idByKey[section.Key] = section.ID
		}
		for index := range bundle.Items {
			item := &bundle.Items[index]
			item.ID = ""
			sectionID, ok := idByKey[item.SectionKey]
			if !ok {
				return fmt.Errorf("butir %s menunjuk bagian yang tidak ada: %s", item.Label, item.SectionKey)
			}
			item.SectionID = sectionID
			if err := tx.Create(item).Error; err != nil {
				return fmt.Errorf("menyimpan butir %s: %w", item.Label, err)
			}
		}

		return nil
	})
}
