package service

import (
	"context"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// CMSStore adalah kebutuhan penyimpanan konten.
type CMSStore interface {
	FAQs(ctx context.Context, onlyActive bool) ([]model.Faq, error)
	FindFaq(ctx context.Context, id string) (*model.Faq, error)
	SaveFaq(ctx context.Context, faq *model.Faq) error
	DeleteFaq(ctx context.Context, id string) error
	Announcements(ctx context.Context, onlyLive bool) ([]model.Announcement, error)
	FindAnnouncement(ctx context.Context, id string) (*model.Announcement, error)
	SaveAnnouncement(ctx context.Context, row *model.Announcement) error
	DeleteAnnouncement(ctx context.Context, id string) error
}

// CMSService mengelola FAQ dan pengumuman.
type CMSService interface {
	FAQs(ctx context.Context, admin bool) ([]dto.FaqItem, error)
	SaveFaq(ctx context.Context, id string, req dto.FaqRequest) (*dto.FaqItem, error)
	DeleteFaq(ctx context.Context, id string) error
	Announcements(ctx context.Context, admin bool) ([]dto.AnnouncementItem, error)
	SaveAnnouncement(ctx context.Context, actor AdminActor, id string, req dto.AnnouncementRequest) (*dto.AnnouncementItem, error)
	DeleteAnnouncement(ctx context.Context, id string) error
}

type cmsService struct {
	store CMSStore
	now   func() time.Time
}

// NewCMSService merangkai layanan konten.
func NewCMSService(store CMSStore) CMSService {
	return &cmsService{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func (s *cmsService) FAQs(ctx context.Context, admin bool) ([]dto.FaqItem, error) {
	rows, err := s.store.FAQs(ctx, !admin)
	if err != nil {
		return nil, err
	}
	out := make([]dto.FaqItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toFaqItem(row))
	}
	return out, nil
}

func (s *cmsService) SaveFaq(ctx context.Context, id string, req dto.FaqRequest) (*dto.FaqItem, error) {
	faq := &model.Faq{}
	if id != "" {
		existing, err := s.store.FindFaq(ctx, id)
		if err != nil {
			return nil, err
		}
		faq = existing
	}
	faq.Category = strings.TrimSpace(req.Category)
	faq.Question = strings.TrimSpace(req.Question)
	faq.AnswerMD = strings.TrimSpace(req.AnswerMD)
	faq.SortOrder = req.SortOrder
	if req.IsActive != nil {
		faq.IsActive = *req.IsActive
	} else if id == "" {
		faq.IsActive = true
	}
	if err := s.store.SaveFaq(ctx, faq); err != nil {
		return nil, err
	}
	item := toFaqItem(*faq)
	return &item, nil
}

func (s *cmsService) DeleteFaq(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return apperror.ErrNotFound
	}
	return s.store.DeleteFaq(ctx, id)
}

func (s *cmsService) Announcements(ctx context.Context, admin bool) ([]dto.AnnouncementItem, error) {
	rows, err := s.store.Announcements(ctx, !admin)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AnnouncementItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAnnouncementItem(row))
	}
	return out, nil
}

func (s *cmsService) SaveAnnouncement(ctx context.Context, actor AdminActor, id string, req dto.AnnouncementRequest) (*dto.AnnouncementItem, error) {
	if !model.KnownAnnouncementLevel(strings.TrimSpace(req.Level)) {
		return nil, validation("tingkat pengumuman tidak dikenal")
	}

	starts := s.now()
	if strings.TrimSpace(req.StartsAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartsAt))
		if err != nil {
			return nil, validation("waktu mulai harus berformat RFC3339")
		}
		starts = parsed.UTC()
	}
	var ends *time.Time
	if strings.TrimSpace(req.EndsAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndsAt))
		if err != nil {
			return nil, validation("waktu selesai harus berformat RFC3339")
		}
		utc := parsed.UTC()
		if !utc.After(starts) {
			return nil, validation("waktu selesai harus setelah waktu mulai")
		}
		ends = &utc
	}

	row := &model.Announcement{}
	if id != "" {
		existing, err := s.store.FindAnnouncement(ctx, id)
		if err != nil {
			return nil, err
		}
		row = existing
	} else {
		row.CreatedBy = actor.ID
	}
	row.Title = strings.TrimSpace(req.Title)
	row.Body = strings.TrimSpace(req.Body)
	row.Level = strings.TrimSpace(req.Level)
	row.StartsAt = starts
	row.EndsAt = ends
	if req.IsActive != nil {
		row.IsActive = *req.IsActive
	} else if id == "" {
		row.IsActive = true
	}

	if err := s.store.SaveAnnouncement(ctx, row); err != nil {
		return nil, err
	}
	item := toAnnouncementItem(*row)
	return &item, nil
}

func (s *cmsService) DeleteAnnouncement(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return apperror.ErrNotFound
	}
	return s.store.DeleteAnnouncement(ctx, id)
}

func toFaqItem(row model.Faq) dto.FaqItem {
	return dto.FaqItem{
		ID:        row.ID,
		Category:  row.Category,
		Question:  row.Question,
		AnswerMD:  row.AnswerMD,
		SortOrder: row.SortOrder,
		IsActive:  row.IsActive,
		UpdatedAt: row.UpdatedAt,
	}
}

func toAnnouncementItem(row model.Announcement) dto.AnnouncementItem {
	return dto.AnnouncementItem{
		ID:        row.ID,
		Title:     row.Title,
		Body:      row.Body,
		Level:     row.Level,
		StartsAt:  row.StartsAt,
		EndsAt:    row.EndsAt,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
	}
}
