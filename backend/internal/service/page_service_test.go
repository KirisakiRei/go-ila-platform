package service_test

import (
	"context"
	"testing"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/service"
	"ila/backend/pkg/apperror"
)

// memPageStore adalah penyimpanan halaman di memori untuk pengujian.
// Ia meniru perilaku repository: bagian dan butir ditulis ulang seluruhnya.
type memPageStore struct {
	pages    map[string]model.Page
	sections []model.PageSection
	items    []model.PageItem
	nextID   int
}

func newMemPageStore() *memPageStore {
	return &memPageStore{pages: map[string]model.Page{}}
}

func (m *memPageStore) id() string {
	m.nextID++
	return "id-" + string(rune('a'+m.nextID))
}

func (m *memPageStore) Find(_ context.Context, slug string) (*repository.PageBundle, error) {
	page, ok := m.pages[slug]
	if !ok {
		return nil, apperror.ErrNotFound
	}

	bundle := &repository.PageBundle{Page: page}
	for _, section := range m.sections {
		if section.PageSlug != slug {
			continue
		}
		bundle.Sections = append(bundle.Sections, section)
	}
	for _, section := range bundle.Sections {
		for _, item := range m.items {
			if item.SectionID == section.ID {
				bundle.Items = append(bundle.Items, item)
			}
		}
	}
	return bundle, nil
}

func (m *memPageStore) List(_ context.Context, onlyPublished bool) ([]repository.PageBundle, error) {
	out := []repository.PageBundle{}
	for slug, page := range m.pages {
		if onlyPublished && !page.IsPublished {
			continue
		}
		bundle, err := m.Find(context.Background(), slug)
		if err != nil {
			return nil, err
		}
		out = append(out, *bundle)
	}
	return out, nil
}

func (m *memPageStore) Save(_ context.Context, bundle *repository.PageBundle) error {
	m.pages[bundle.Page.Slug] = bundle.Page

	keptSections := m.sections[:0]
	for _, section := range m.sections {
		if section.PageSlug != bundle.Page.Slug {
			keptSections = append(keptSections, section)
		}
	}
	m.sections = keptSections

	keptItems := m.items[:0]
	for _, item := range m.items {
		// Butir bagian lama sudah tidak ada bagiannya.
		stillThere := false
		for _, section := range bundle.Sections {
			if section.ID == item.SectionID {
				stillThere = true
				break
			}
		}
		if stillThere && item.SectionID != "" {
			keptItems = append(keptItems, item)
		}
	}
	m.items = keptItems

	idByKey := map[string]string{}
	for index := range bundle.Sections {
		section := &bundle.Sections[index]
		section.ID = m.id()
		idByKey[section.Key] = section.ID
		m.sections = append(m.sections, *section)
	}
	for index := range bundle.Items {
		item := &bundle.Items[index]
		item.ID = m.id()
		item.SectionID = idByKey[item.SectionKey]
		m.items = append(m.items, *item)
	}
	return nil
}

func seedPage(store *memPageStore, slug string, published bool) {
	store.pages[slug] = model.Page{Slug: slug, Title: "Judul", IsPublished: published}
}

func TestHalamanYangBelumDiterbitkanTidakBocorKePublik(t *testing.T) {
	store := newMemPageStore()
	seedPage(store, "tentang", false)
	svc := service.NewPageService(store)

	if _, err := svc.Public(context.Background(), "tentang"); err == nil {
		t.Fatal("halaman yang belum diterbitkan seharusnya tidak terbaca publik")
	}

	// Admin tetap bisa membukanya untuk disunting.
	if _, err := svc.Admin(context.Background(), "tentang"); err != nil {
		t.Fatalf("admin harus bisa membuka halaman draf: %v", err)
	}
}

func TestSimpanHalamanMenggantiSeluruhIsi(t *testing.T) {
	store := newMemPageStore()
	seedPage(store, "fitur", true)
	svc := service.NewPageService(store)
	actor := service.AdminActor{ID: "admin-1"}

	publish := true
	req := dto.PageUpdateRequest{
		Title:       "Judul baru",
		IsPublished: &publish,
		Sections: []dto.PageSectionRequest{
			{
				Key:  "kemampuan",
				Kind: model.SectionFeatures,
				Items: []dto.PageItemRequest{
					{Label: "Kartu pertama", Value: "Isi pertama", Note: "Batasannya"},
				},
			},
		},
	}

	first, err := svc.Replace(context.Background(), actor, "fitur", req)
	if err != nil {
		t.Fatalf("simpan pertama gagal: %v", err)
	}
	if len(first.Sections) != 1 || len(first.Sections[0].Items) != 1 {
		t.Fatalf("isi tersimpan tidak sesuai: %+v", first.Sections)
	}

	// Simpan ulang dengan isi lebih sedikit: butir lama harus benar-benar hilang.
	req.Sections[0].Items = nil
	second, err := svc.Replace(context.Background(), actor, "fitur", req)
	if err != nil {
		t.Fatalf("simpan kedua gagal: %v", err)
	}
	if len(second.Sections) != 1 {
		t.Fatalf("jumlah bagian = %d; ingin 1", len(second.Sections))
	}
	if len(second.Sections[0].Items) != 0 {
		t.Fatalf("butir lama masih tersisa: %+v", second.Sections[0].Items)
	}
}

func TestKunciBagianGandaDitolak(t *testing.T) {
	store := newMemPageStore()
	seedPage(store, "fitur", true)
	svc := service.NewPageService(store)

	_, err := svc.Replace(context.Background(), service.AdminActor{ID: "a"}, "fitur", dto.PageUpdateRequest{
		Sections: []dto.PageSectionRequest{
			{Key: "sama", Kind: model.SectionProse},
			{Key: "sama", Kind: model.SectionProse},
		},
	})
	if err == nil {
		t.Fatal("kunci bagian ganda seharusnya ditolak")
	}
}

func TestJenisBagianTidakDikenalDitolak(t *testing.T) {
	store := newMemPageStore()
	seedPage(store, "fitur", true)
	svc := service.NewPageService(store)

	_, err := svc.Replace(context.Background(), service.AdminActor{ID: "a"}, "fitur", dto.PageUpdateRequest{
		Sections: []dto.PageSectionRequest{{Key: "x", Kind: "html-bebas"}},
	})
	if err == nil {
		t.Fatal("jenis bagian di luar daftar seharusnya ditolak")
	}
}

func TestButirTanpaJudulDitolak(t *testing.T) {
	store := newMemPageStore()
	seedPage(store, "fitur", true)
	svc := service.NewPageService(store)

	_, err := svc.Replace(context.Background(), service.AdminActor{ID: "a"}, "fitur", dto.PageUpdateRequest{
		Sections: []dto.PageSectionRequest{{
			Key:   "kemampuan",
			Kind:  model.SectionCards,
			Items: []dto.PageItemRequest{{Value: "tanpa judul"}},
		}},
	})
	if err == nil {
		t.Fatal("butir tanpa judul seharusnya ditolak")
	}
}
