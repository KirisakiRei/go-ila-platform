package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/legal"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// LegalService mengurus dokumen legal berversi dan bukti persetujuannya.
type LegalService interface {
	// Published mengembalikan daftar dokumen yang sedang berlaku (publik).
	Published(ctx context.Context) ([]dto.LegalDocumentResponse, error)
	// BySlug mengembalikan versi berlaku sebuah dokumen (publik).
	BySlug(ctx context.Context, slug string) (*dto.LegalDocumentResponse, error)
	// ForUser melaporkan status persetujuan akun atas setiap dokumen wajib.
	ForUser(ctx context.Context, userID string) (*dto.LegalStatusResponse, error)
	// Accept mencatat persetujuan satu dokumen.
	Accept(ctx context.Context, userID, slug string, req dto.AcceptLegalRequest, meta AcceptanceMeta) (*dto.LegalStatusResponse, error)
	// AcceptSignup mencatat persetujuan awal saat mendaftar.
	AcceptSignup(ctx context.Context, userID string, meta AcceptanceMeta) error
	// Pending melaporkan dokumen wajib yang belum disetujui akun.
	Pending(ctx context.Context, userID string) ([]dto.LegalPendingItem, error)
	// RequireSignup memastikan centang pendaftaran lengkap.
	RequireSignup(req dto.RegisterRequest) error
	// RequireAccountLink memastikan disclaimer risiko sudah disetujui.
	RequireAccountLink(ctx context.Context, userID string) error

	// Pengelolaan oleh admin.
	ListAll(ctx context.Context, slug string) ([]dto.LegalDocumentResponse, error)
	CreateDraft(ctx context.Context, actorID string, req dto.LegalDocumentRequest) (*dto.LegalDocumentResponse, error)
	UpdateDraft(ctx context.Context, actorID, id string, req dto.LegalDocumentRequest) (*dto.LegalDocumentResponse, error)
	Publish(ctx context.Context, actorID, id string, req dto.PublishLegalRequest) (*dto.LegalDocumentResponse, error)
}

// AcceptanceMeta adalah bukti tambahan saat mencatat persetujuan.
type AcceptanceMeta struct {
	IPHash    string
	UserAgent string
}

// LegalRepository adalah kebutuhan service atas penyimpanan legal.
type LegalRepository interface {
	LatestPublished(ctx context.Context, slug string) (*model.LegalDocument, error)
	ListPublished(ctx context.Context) ([]model.LegalDocument, error)
	FindByID(ctx context.Context, id string) (*model.LegalDocument, error)
	ListAll(ctx context.Context, slug string) ([]model.LegalDocument, error)
	CreateDraft(ctx context.Context, doc *model.LegalDocument) error
	UpdateDraft(ctx context.Context, doc *model.LegalDocument) error
	Publish(ctx context.Context, id string, at time.Time) error
	NextVersion(ctx context.Context, slug string) (int, error)
	RecordAcceptance(ctx context.Context, entry *model.LegalAcceptance) error
	AcceptancesOfUser(ctx context.Context, userID string) ([]model.LegalAcceptance, error)
	LatestAcceptance(ctx context.Context, userID, slug string) (*model.LegalAcceptance, error)
	CountAcceptances(ctx context.Context, slug string, version int) (int64, error)
}

type legalService struct {
	repo  LegalRepository
	audit AuditRepository
	now   func() time.Time
}

// NewLegalService merangkai layanan legal.
func NewLegalService(repo LegalRepository, audit AuditRepository) LegalService {
	return &legalService{
		repo:  repo,
		audit: audit,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// Published mengembalikan dokumen yang sedang berlaku, tanpa perlu sesi.
func (s *legalService) Published(ctx context.Context) ([]dto.LegalDocumentResponse, error) {
	docs, err := s.repo.ListPublished(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]dto.LegalDocumentResponse, 0, len(docs))
	for _, doc := range docs {
		out = append(out, toLegalResponse(doc))
	}
	return out, nil
}

// BySlug mengembalikan versi berlaku satu dokumen.
func (s *legalService) BySlug(ctx context.Context, slug string) (*dto.LegalDocumentResponse, error) {
	if !legal.Known(slug) {
		return nil, apperror.ErrNotFound
	}

	doc, err := s.repo.LatestPublished(ctx, slug)
	if err != nil {
		return nil, err
	}
	response := toLegalResponse(*doc)
	return &response, nil
}

// ForUser melaporkan, untuk setiap dokumen, versi berlaku dan versi yang sudah
// disetujui akun.
func (s *legalService) ForUser(ctx context.Context, userID string) (*dto.LegalStatusResponse, error) {
	published, err := s.repo.ListPublished(ctx)
	if err != nil {
		return nil, err
	}

	acceptances, err := s.repo.AcceptancesOfUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	accepted := make(map[string]int, len(acceptances))
	for _, entry := range acceptances {
		if entry.Version > accepted[entry.DocumentSlug] {
			accepted[entry.DocumentSlug] = entry.Version
		}
	}

	items := make([]dto.LegalStatusItem, 0, len(published))
	pending := make([]dto.LegalPendingItem, 0)
	for _, doc := range published {
		doneVersion, has := accepted[doc.Slug]
		upToDate := has && doneVersion >= doc.Version

		item := dto.LegalStatusItem{
			Slug:            doc.Slug,
			Title:           doc.Title,
			CurrentVersion:  doc.Version,
			AcceptedVersion: doneVersion,
			Accepted:        upToDate,
			EffectiveAt:     doc.EffectiveAt.Format(time.RFC3339),
			Effective:       dto.NewTimePair(doc.EffectiveAt),
			RequireReaccept: doc.RequireReaccept,
		}
		items = append(items, item)

		if !upToDate && requiresAtSignup(doc.Slug) {
			pending = append(pending, dto.LegalPendingItem{
				Slug:           doc.Slug,
				Title:          doc.Title,
				CurrentVersion: doc.Version,
				Required:       true,
			})
		}
	}

	return &dto.LegalStatusResponse{
		Documents: items,
		Pending:   pending,
		// Blocked menandai bahwa eksekusi & pembayaran ditahan sampai setuju.
		Blocked: len(pending) > 0,
	}, nil
}

// Accept mencatat persetujuan akun atas versi terbit yang berlaku.
func (s *legalService) Accept(ctx context.Context, userID, slug string, req dto.AcceptLegalRequest, meta AcceptanceMeta) (*dto.LegalStatusResponse, error) {
	if !legal.Known(slug) {
		return nil, apperror.ErrNotFound
	}

	doc, err := s.repo.LatestPublished(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Versi yang disetujui harus versi yang sedang berlaku: mencegah akun
	// menyetujui versi lama sambil melewati versi baru.
	if req.Version != 0 && req.Version != doc.Version {
		return nil, validation("versi dokumen sudah berubah; muat ulang halaman")
	}

	if err := s.record(ctx, userID, doc, meta); err != nil {
		return nil, err
	}

	return s.ForUser(ctx, userID)
}

// AcceptSignup mencatat persetujuan dokumen yang diwajibkan saat mendaftar.
func (s *legalService) AcceptSignup(ctx context.Context, userID string, meta AcceptanceMeta) error {
	for _, slug := range legal.SignupSlugs() {
		doc, err := s.repo.LatestPublished(ctx, slug)
		if err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				// Dokumen belum diterbitkan: tidak adil menahan pendaftaran.
				continue
			}
			return err
		}
		if err := s.record(ctx, userID, doc, meta); err != nil {
			return err
		}
	}
	return nil
}

// Pending mengembalikan dokumen wajib saat daftar yang belum disetujui.
func (s *legalService) Pending(ctx context.Context, userID string) ([]dto.LegalPendingItem, error) {
	status, err := s.ForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return status.Pending, nil
}

// RequireSignup memastikan pemohon mencentang seluruh dokumen wajib.
func (s *legalService) RequireSignup(req dto.RegisterRequest) error {
	if !req.AcceptTerms {
		return validation("persetujuan Syarat & Ketentuan dan Kebijakan Privasi wajib dicentang")
	}
	return nil
}

// RequireAccountLink memastikan disclaimer risiko sudah disetujui sebelum akun
// broker dihubungkan. Sebelum ada akun, user belum menanggung risiko finansial.
func (s *legalService) RequireAccountLink(ctx context.Context, userID string) error {
	for _, slug := range legal.DemoAccountSlugs() {
		doc, err := s.repo.LatestPublished(ctx, slug)
		if err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				continue
			}
			return err
		}

		last, err := s.repo.LatestAcceptance(ctx, userID, slug)
		if err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				return validation("setujui " + doc.Title + " sebelum menghubungkan akun broker")
			}
			return err
		}
		if last.Version < doc.Version {
			return validation("setujui versi terbaru " + doc.Title + " sebelum menghubungkan akun broker")
		}
	}
	return nil
}

// ListAll menampilkan seluruh versi termasuk draf untuk admin.
func (s *legalService) ListAll(ctx context.Context, slug string) ([]dto.LegalDocumentResponse, error) {
	docs, err := s.repo.ListAll(ctx, slug)
	if err != nil {
		return nil, err
	}

	out := make([]dto.LegalDocumentResponse, 0, len(docs))
	for _, doc := range docs {
		out = append(out, toLegalResponse(doc))
	}
	return out, nil
}

// CreateDraft membuat versi baru sebagai draf. Versi lama tidak diubah.
func (s *legalService) CreateDraft(ctx context.Context, actorID string, req dto.LegalDocumentRequest) (*dto.LegalDocumentResponse, error) {
	if !legal.Known(req.Slug) {
		return nil, validation("jenis dokumen tidak dikenal: " + req.Slug)
	}

	version, err := s.repo.NextVersion(ctx, req.Slug)
	if err != nil {
		return nil, err
	}

	now := s.now()
	effectiveAt := now
	if strings.TrimSpace(req.EffectiveAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.EffectiveAt)
		if err != nil {
			return nil, validation("effective_at harus berformat RFC3339")
		}
		effectiveAt = parsed.UTC()
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = legal.Title(req.Slug)
	}

	doc := &model.LegalDocument{
		Slug:            req.Slug,
		Version:         version,
		Title:           title,
		BodyMD:          req.BodyMD,
		RequireReaccept: req.RequireReaccept,
		EffectiveAt:     effectiveAt,
		CreatedBy:       actorID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.CreateDraft(ctx, doc); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, actorID, "legal.draft_created", doc.ID, req.Slug)
	response := toLegalResponse(*doc)
	return &response, nil
}

// UpdateDraft hanya boleh menyentuh dokumen yang belum terbit.
func (s *legalService) UpdateDraft(ctx context.Context, actorID, id string, req dto.LegalDocumentRequest) (*dto.LegalDocumentResponse, error) {
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if doc.PublishedAt != nil {
		return nil, validation("dokumen yang sudah terbit tidak dapat diubah; buat versi baru")
	}

	if strings.TrimSpace(req.BodyMD) != "" {
		doc.BodyMD = req.BodyMD
	}
	if strings.TrimSpace(req.Title) != "" {
		doc.Title = strings.TrimSpace(req.Title)
	}
	doc.RequireReaccept = req.RequireReaccept
	if strings.TrimSpace(req.EffectiveAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.EffectiveAt)
		if err != nil {
			return nil, validation("effective_at harus berformat RFC3339")
		}
		doc.EffectiveAt = parsed.UTC()
	}
	doc.UpdatedAt = s.now()

	if err := s.repo.UpdateDraft(ctx, doc); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, actorID, "legal.draft_updated", doc.ID, doc.Slug)
	response := toLegalResponse(*doc)
	return &response, nil
}

// Publish menerbitkan draf sehingga menjadi versi yang berlaku.
func (s *legalService) Publish(ctx context.Context, actorID, id string, req dto.PublishLegalRequest) (*dto.LegalDocumentResponse, error) {
	doc, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if doc.PublishedAt != nil {
		return nil, validation("dokumen ini sudah terbit")
	}

	// Perubahan wajib-setuju-ulang memutus eksekusi pengguna yang belum setuju,
	// jadi alasannya dicatat.
	now := s.now()
	if err := s.repo.Publish(ctx, id, now); err != nil {
		return nil, err
	}

	doc.PublishedAt = &now
	s.recordAudit(ctx, actorID, "legal.published", doc.ID, strings.TrimSpace(req.Reason)+" "+doc.Slug)

	response := toLegalResponse(*doc)
	return &response, nil
}

// record menyimpan satu persetujuan. Persetujuan yang sudah ada dengan versi
// sama tidak dicatat dua kali.
func (s *legalService) record(ctx context.Context, userID string, doc *model.LegalDocument, meta AcceptanceMeta) error {
	if last, err := s.repo.LatestAcceptance(ctx, userID, doc.Slug); err == nil && last.Version >= doc.Version {
		return nil
	} else if err != nil && !errors.Is(err, apperror.ErrNotFound) {
		return err
	}

	entry := &model.LegalAcceptance{
		UserID:       userID,
		DocumentSlug: doc.Slug,
		Version:      doc.Version,
		AcceptedAt:   s.now(),
		IPHash:       meta.IPHash,
		UserAgent:    truncate(meta.UserAgent, 255),
	}
	if err := s.repo.RecordAcceptance(ctx, entry); err != nil {
		return err
	}

	s.recordAudit(ctx, userID, "legal.accepted", doc.ID, doc.Slug+" v"+strconv.Itoa(doc.Version))
	return nil
}

func (s *legalService) recordAudit(ctx context.Context, actorID, event, entityID, detail string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, &model.AuditLog{
		UserID:    actorID,
		Event:     event,
		Entity:    "legal",
		EntityID:  entityID,
		Metadata:  detail,
		CreatedAt: s.now(),
	})
}

// requiresAtSignup menandai dokumen yang menahan fitur bila belum disetujui.
func requiresAtSignup(slug string) bool {
	for _, s := range legal.SignupSlugs() {
		if s == slug {
			return true
		}
	}
	return false
}

// toLegalResponse memetakan dokumen ke bentuk yang dikirim ke klien.
func toLegalResponse(doc model.LegalDocument) dto.LegalDocumentResponse {
	response := dto.LegalDocumentResponse{
		ID:              doc.ID,
		Slug:            doc.Slug,
		Version:         doc.Version,
		Title:           doc.Title,
		BodyMD:          doc.BodyMD,
		RequireReaccept: doc.RequireReaccept,
		EffectiveAt:     doc.EffectiveAt.Format(time.RFC3339),
		Effective:       dto.NewTimePair(doc.EffectiveAt),
		Published:       doc.PublishedAt != nil,
	}
	if doc.PublishedAt != nil {
		response.PublishedAt = doc.PublishedAt.Format(time.RFC3339)
		response.Published = true
	}
	return response
}
