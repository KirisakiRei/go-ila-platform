package service

import (
	"context"
	"strings"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// PageStore adalah kebutuhan penyimpanan halaman publik.
type PageStore interface {
	Find(ctx context.Context, slug string) (*repository.PageBundle, error)
	List(ctx context.Context, onlyPublished bool) ([]repository.PageBundle, error)
	Save(ctx context.Context, bundle *repository.PageBundle) error
}

// PageService mengelola isi halaman publik.
//
// Halaman dianggap ada bila sudah tersimpan; service tidak menebak isi bawaan
// di dalam kode. Halaman yang belum pernah disimpan dilaporkan sebagai tidak
// ditemukan, sehingga antarmuka menampilkan keadaan kosong yang jelas alih-alih
// isi lama yang tidak lagi diakui admin.
type PageService interface {
	Public(ctx context.Context, slug string) (*dto.PageResponse, error)
	Admin(ctx context.Context, slug string) (*dto.PageResponse, error)
	All(ctx context.Context) ([]dto.PageResponse, error)
	Replace(ctx context.Context, actor AdminActor, slug string, req dto.PageUpdateRequest) (*dto.PageResponse, error)
}

type pageService struct {
	store PageStore
}

// NewPageService merangkai layanan halaman.
func NewPageService(store PageStore) PageService {
	return &pageService{store: store}
}

func (s *pageService) Public(ctx context.Context, slug string) (*dto.PageResponse, error) {
	bundle, err := s.store.Find(ctx, strings.TrimSpace(slug))
	if err != nil {
		return nil, err
	}
	// Halaman yang belum diterbitkan tidak boleh bocor ke publik; bagi
	// pengunjung ia sama saja dengan halaman yang tidak ada.
	if !bundle.Page.IsPublished {
		return nil, apperror.ErrNotFound
	}
	out := toPageResponse(*bundle, true)
	return &out, nil
}

func (s *pageService) Admin(ctx context.Context, slug string) (*dto.PageResponse, error) {
	bundle, err := s.store.Find(ctx, strings.TrimSpace(slug))
	if err != nil {
		return nil, err
	}
	out := toPageResponse(*bundle, false)
	return &out, nil
}

func (s *pageService) All(ctx context.Context) ([]dto.PageResponse, error) {
	bundles, err := s.store.List(ctx, false)
	if err != nil {
		return nil, err
	}
	out := make([]dto.PageResponse, 0, len(bundles))
	for _, bundle := range bundles {
		out = append(out, toPageResponse(bundle, false))
	}
	return out, nil
}

func (s *pageService) Replace(ctx context.Context, actor AdminActor, slug string, req dto.PageUpdateRequest) (*dto.PageResponse, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, validation("slug halaman wajib diisi")
	}

	current, err := s.store.Find(ctx, slug)
	if err != nil {
		return nil, err
	}

	page := current.Page
	page.Eyebrow = strings.TrimSpace(req.Eyebrow)
	page.Title = strings.TrimSpace(req.Title)
	page.Lead = strings.TrimSpace(req.Lead)
	page.MetaTitle = strings.TrimSpace(req.MetaTitle)
	page.MetaDescription = strings.TrimSpace(req.MetaDescription)
	page.UpdatedBy = actor.ID
	if req.IsPublished != nil {
		page.IsPublished = *req.IsPublished
	}

	bundle := &repository.PageBundle{Page: page}
	seen := make(map[string]struct{}, len(req.Sections))

	for index, sectionReq := range req.Sections {
		key := strings.TrimSpace(sectionReq.Key)
		if key == "" {
			return nil, validation("setiap bagian halaman harus punya kunci")
		}
		if _, duplicate := seen[key]; duplicate {
			return nil, validation("kunci bagian halaman tidak boleh ganda: " + key)
		}
		seen[key] = struct{}{}

		kind := strings.TrimSpace(sectionReq.Kind)
		if !model.KnownSectionKind(kind) {
			return nil, validation("jenis bagian halaman tidak dikenal: " + kind)
		}

		section := model.PageSection{
			PageSlug:  slug,
			Key:       key,
			Kind:      kind,
			Eyebrow:   strings.TrimSpace(sectionReq.Eyebrow),
			Title:     strings.TrimSpace(sectionReq.Title),
			Lead:      strings.TrimSpace(sectionReq.Lead),
			Body:      strings.TrimSpace(sectionReq.Body),
			Layout:    strings.TrimSpace(sectionReq.Layout),
			SortOrder: sectionReq.SortOrder,
		}
		if sectionReq.IsPublished != nil {
			section.IsPublished = *sectionReq.IsPublished
		} else {
			section.IsPublished = true
		}
		// Urutan bagian mengikuti urutan yang dikirim, sehingga admin tidak
		// perlu mengetik nomor urut agar hasilnya sesuai layar kerjanya.
		if section.SortOrder == 0 {
			section.SortOrder = index + 1
		}

		for itemIndex, itemReq := range sectionReq.Items {
			label := strings.TrimSpace(itemReq.Label)
			if label == "" {
				return nil, validation("butir bagian " + key + " harus punya judul")
			}
			order := itemReq.SortOrder
			if order == 0 {
				order = itemIndex + 1
			}
			// Butir ditautkan lewat kunci bagian, bukan ID: ID bagian baru
			// dibuat saat tersimpan, sedangkan butir disimpan setelahnya.
			bundle.Items = append(bundle.Items, model.PageItem{
				SectionKey: key,
				Label:      label,
				Value:      strings.TrimSpace(itemReq.Value),
				Note:       strings.TrimSpace(itemReq.Note),
				Icon:       strings.TrimSpace(itemReq.Icon),
				Href:       strings.TrimSpace(itemReq.Href),
				SortOrder:  order,
			})
		}

		bundle.Sections = append(bundle.Sections, section)
	}

	if err := s.store.Save(ctx, bundle); err != nil {
		return nil, err
	}

	saved, err := s.store.Find(ctx, slug)
	if err != nil {
		return nil, err
	}
	out := toPageResponse(*saved, false)
	return &out, nil
}

func toPageResponse(bundle repository.PageBundle, publicOnly bool) dto.PageResponse {
	itemsBySection := make(map[string][]dto.PageItemResponse, len(bundle.Sections))
	for _, item := range bundle.Items {
		itemsBySection[item.SectionID] = append(itemsBySection[item.SectionID], dto.PageItemResponse{
			Label:     item.Label,
			Value:     item.Value,
			Note:      item.Note,
			Icon:      item.Icon,
			Href:      item.Href,
			SortOrder: item.SortOrder,
		})
	}

	sections := make([]dto.PageSectionResponse, 0, len(bundle.Sections))
	for _, section := range bundle.Sections {
		if publicOnly && !section.IsPublished {
			continue
		}
		items := itemsBySection[section.ID]
		if items == nil {
			items = []dto.PageItemResponse{}
		}
		sections = append(sections, dto.PageSectionResponse{
			Key:         section.Key,
			Kind:        section.Kind,
			Eyebrow:     section.Eyebrow,
			Title:       section.Title,
			Lead:        section.Lead,
			Body:        section.Body,
			Layout:      section.Layout,
			SortOrder:   section.SortOrder,
			IsPublished: section.IsPublished,
			Items:       items,
		})
	}

	return dto.PageResponse{
		Slug:            bundle.Page.Slug,
		Eyebrow:         bundle.Page.Eyebrow,
		Title:           bundle.Page.Title,
		Lead:            bundle.Page.Lead,
		MetaTitle:       bundle.Page.MetaTitle,
		MetaDescription: bundle.Page.MetaDescription,
		IsPublished:     bundle.Page.IsPublished,
		UpdatedAt:       bundle.Page.UpdatedAt,
		Sections:        sections,
	}
}
