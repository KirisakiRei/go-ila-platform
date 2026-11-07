package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/legal"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// fakeLegalStore menyimpan dokumen legal dan persetujuan di memori.
type fakeLegalStore struct {
	docs        []model.LegalDocument
	acceptances []model.LegalAcceptance
}

func (f *fakeLegalStore) addPublished(slug string, version int, requireReaccept bool) {
	now := time.Now().UTC().Add(-time.Hour)
	f.docs = append(f.docs, model.LegalDocument{
		ID:              slug + "-v" + itoaInt(version),
		Slug:            slug,
		Version:         version,
		Title:           legal.Title(slug),
		BodyMD:          "isi",
		RequireReaccept: requireReaccept,
		PublishedAt:     &now,
		EffectiveAt:     now,
	})
}

func (f *fakeLegalStore) LatestPublished(_ context.Context, slug string) (*model.LegalDocument, error) {
	var best *model.LegalDocument
	for i := range f.docs {
		doc := f.docs[i]
		if doc.Slug != slug || doc.PublishedAt == nil {
			continue
		}
		if best == nil || doc.Version > best.Version {
			best = &doc
		}
	}
	if best == nil {
		return nil, apperror.ErrNotFound
	}
	return best, nil
}

// ListPublished meniru query asli: satu versi tertinggi per slug.
func (f *fakeLegalStore) ListPublished(context.Context) ([]model.LegalDocument, error) {
	highest := make(map[string]model.LegalDocument)
	for _, doc := range f.docs {
		if doc.PublishedAt == nil {
			continue
		}
		current, ok := highest[doc.Slug]
		if !ok || doc.Version > current.Version {
			highest[doc.Slug] = doc
		}
	}

	out := make([]model.LegalDocument, 0, len(highest))
	for _, doc := range highest {
		out = append(out, doc)
	}
	return out, nil
}

func (f *fakeLegalStore) FindByID(_ context.Context, id string) (*model.LegalDocument, error) {
	for i := range f.docs {
		if f.docs[i].ID == id {
			doc := f.docs[i]
			return &doc, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakeLegalStore) ListAll(_ context.Context, slug string) ([]model.LegalDocument, error) {
	var out []model.LegalDocument
	for _, doc := range f.docs {
		if slug == "" || doc.Slug == slug {
			out = append(out, doc)
		}
	}
	return out, nil
}

func (f *fakeLegalStore) CreateDraft(_ context.Context, doc *model.LegalDocument) error {
	if doc.ID == "" {
		doc.ID = doc.Slug + "-draft"
	}
	f.docs = append(f.docs, *doc)
	return nil
}

func (f *fakeLegalStore) UpdateDraft(_ context.Context, doc *model.LegalDocument) error {
	for i := range f.docs {
		if f.docs[i].ID == doc.ID {
			f.docs[i] = *doc
			return nil
		}
	}
	return apperror.ErrNotFound
}

func (f *fakeLegalStore) Publish(_ context.Context, id string, at time.Time) error {
	for i := range f.docs {
		if f.docs[i].ID == id {
			f.docs[i].PublishedAt = &at
			return nil
		}
	}
	return apperror.ErrNotFound
}

func (f *fakeLegalStore) NextVersion(_ context.Context, slug string) (int, error) {
	highest := 0
	for _, doc := range f.docs {
		if doc.Slug == slug && doc.Version > highest {
			highest = doc.Version
		}
	}
	return highest + 1, nil
}

func (f *fakeLegalStore) RecordAcceptance(_ context.Context, entry *model.LegalAcceptance) error {
	f.acceptances = append(f.acceptances, *entry)
	return nil
}

func (f *fakeLegalStore) AcceptancesOfUser(_ context.Context, userID string) ([]model.LegalAcceptance, error) {
	var out []model.LegalAcceptance
	for _, entry := range f.acceptances {
		if entry.UserID == userID {
			out = append(out, entry)
		}
	}
	return out, nil
}

func (f *fakeLegalStore) LatestAcceptance(_ context.Context, userID, slug string) (*model.LegalAcceptance, error) {
	var best *model.LegalAcceptance
	for i := range f.acceptances {
		entry := f.acceptances[i]
		if entry.UserID != userID || entry.DocumentSlug != slug {
			continue
		}
		if best == nil || entry.Version > best.Version {
			best = &entry
		}
	}
	if best == nil {
		return nil, apperror.ErrNotFound
	}
	return best, nil
}

func (f *fakeLegalStore) CountAcceptances(_ context.Context, slug string, version int) (int64, error) {
	var total int64
	for _, entry := range f.acceptances {
		if entry.DocumentSlug == slug && entry.Version == version {
			total++
		}
	}
	return total, nil
}

func (f *fakeLegalStore) SeedInitialDrafts(context.Context, []legal.Draft, time.Time) (int, error) {
	return 0, nil
}

// itoaInt mengubah int menjadi teks untuk ID uji.
func itoaInt(v int) string {
	if v == 0 {
		return "0"
	}
	digits := ""
	for v > 0 {
		digits = string(rune('0'+v%10)) + digits
		v /= 10
	}
	return digits
}

// newLegalFixture menyiapkan layanan legal dengan lima dokumen terbit.
func newLegalFixture() (*legalService, *fakeLegalStore) {
	store := &fakeLegalStore{}
	for _, slug := range []string{
		legal.SlugTOS, legal.SlugPrivacy, legal.SlugRisk, legal.SlugRefund, legal.SlugFairUse,
	} {
		store.addPublished(slug, 1, false)
	}
	return NewLegalService(store, nil).(*legalService), store
}

// Registrasi tanpa centang ditolak.
func TestRegistrasiTanpaCentangDitolak(t *testing.T) {
	svc, _ := newLegalFixture()

	err := svc.RequireSignup(dto.RegisterRequest{Name: "A", Email: "a@b.c", AcceptTerms: false})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}

	if err := svc.RequireSignup(dto.RegisterRequest{AcceptTerms: true}); err != nil {
		t.Fatalf("dengan centang seharusnya lolos, tetapi: %v", err)
	}
}

// Persetujuan dicatat beserta versi, sehingga bisa dibuktikan kemudian.
func TestAcceptMencatatVersi(t *testing.T) {
	svc, store := newLegalFixture()
	ctx := context.Background()

	if err := svc.AcceptSignup(ctx, "u1", AcceptanceMeta{UserAgent: "uji"}); err != nil {
		t.Fatalf("AcceptSignup error: %v", err)
	}

	// Tiga dokumen wajib daftar dicatat, bukan lima.
	if len(store.acceptances) != len(legal.SignupSlugs()) {
		t.Fatalf("persetujuan tercatat %d; ingin %d", len(store.acceptances), len(legal.SignupSlugs()))
	}
	for _, entry := range store.acceptances {
		if entry.Version != 1 {
			t.Fatalf("versi tercatat %d; ingin 1", entry.Version)
		}
	}
}

// Versi baru terbit dengan wajib-setuju-ulang → user ditandai belum setuju.
func TestVersiBaruMembuatPending(t *testing.T) {
	svc, store := newLegalFixture()
	ctx := context.Background()

	if err := svc.AcceptSignup(ctx, "u1", AcceptanceMeta{}); err != nil {
		t.Fatalf("AcceptSignup error: %v", err)
	}

	status, err := svc.ForUser(ctx, "u1")
	if err != nil {
		t.Fatalf("ForUser error: %v", err)
	}
	if status.Blocked {
		t.Fatal("setelah menyetujui semua dokumen wajib, status tidak boleh terblokir")
	}

	// Terbitkan versi 2 ToS yang wajib disetujui ulang.
	store.addPublished(legal.SlugTOS, 2, true)

	status, err = svc.ForUser(ctx, "u1")
	if err != nil {
		t.Fatalf("ForUser error: %v", err)
	}
	if !status.Blocked {
		t.Fatal("versi baru yang wajib disetujui seharusnya memblokir fitur")
	}
	if len(status.Pending) == 0 || status.Pending[0].Slug != legal.SlugTOS {
		t.Fatalf("pending = %+v; ingin tos", status.Pending)
	}
}

// Menyetujui versi lama sambil melewati versi baru harus ditolak.
func TestAcceptVersiLamaDitolak(t *testing.T) {
	svc, store := newLegalFixture()
	ctx := context.Background()

	store.addPublished(legal.SlugTOS, 2, true)

	if _, err := svc.Accept(ctx, "u1", legal.SlugTOS, dto.AcceptLegalRequest{Version: 1}, AcceptanceMeta{}); err == nil {
		t.Fatal("menyetujui versi 1 saat versi 2 berlaku seharusnya ditolak")
	}

	if _, err := svc.Accept(ctx, "u1", legal.SlugTOS, dto.AcceptLegalRequest{Version: 2}, AcceptanceMeta{}); err != nil {
		t.Fatalf("menyetujui versi berlaku seharusnya lolos: %v", err)
	}
}

// Hubungkan akun broker wajib disclaimer risiko.
func TestHubungkanAkunWajibDisclaimerRisiko(t *testing.T) {
	svc, _ := newLegalFixture()
	ctx := context.Background()

	err := svc.RequireAccountLink(ctx, "u1")
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}

	if _, err := svc.Accept(ctx, "u1", legal.SlugRisk, dto.AcceptLegalRequest{}, AcceptanceMeta{}); err != nil {
		t.Fatalf("Accept error: %v", err)
	}

	if err := svc.RequireAccountLink(ctx, "u1"); err != nil {
		t.Fatalf("setelah menyetujui disclaimer seharusnya lolos: %v", err)
	}
}

// Dokumen yang sudah terbit tidak dapat diubah.
func TestDokumenTerbitTidakBisaDiubah(t *testing.T) {
	svc, store := newLegalFixture()
	ctx := context.Background()

	published := store.docs[0]
	_, err := svc.UpdateDraft(ctx, "admin", published.ID, dto.LegalDocumentRequest{BodyMD: "baru"})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Menerbitkan dokumen mencabut status draf.
func TestPublishMenjadikanBerlaku(t *testing.T) {
	svc, store := newLegalFixture()
	ctx := context.Background()

	draft, err := svc.CreateDraft(ctx, "admin", dto.LegalDocumentRequest{
		Slug:   legal.SlugTOS,
		BodyMD: "isi baru",
	})
	if err != nil {
		t.Fatalf("CreateDraft error: %v", err)
	}
	if draft.Version != 2 {
		t.Fatalf("versi draf = %d; ingin 2", draft.Version)
	}
	if draft.Published {
		t.Fatal("draf seharusnya belum terbit")
	}

	published, err := svc.Publish(ctx, "admin", draft.ID, dto.PublishLegalRequest{Reason: "perubahan biaya"})
	if err != nil {
		t.Fatalf("Publish error: %v", err)
	}
	if !published.Published {
		t.Fatal("dokumen seharusnya terbit")
	}

	// Versi lama tetap tersimpan.
	if len(store.docs) < 2 {
		t.Fatal("versi lama seharusnya tetap ada")
	}
}

// Slug asing ditolak pada pembuatan draf.
func TestCreateDraftSlugAsingDitolak(t *testing.T) {
	svc, _ := newLegalFixture()

	_, err := svc.CreateDraft(context.Background(), "admin", dto.LegalDocumentRequest{Slug: "tidak_ada", BodyMD: "x"})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}
