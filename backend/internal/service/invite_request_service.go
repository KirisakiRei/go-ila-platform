package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

const templateInviteApproved = "invite_approved"

// InviteRequestStore adalah kebutuhan penyimpanan permohonan undangan.
type InviteRequestStore interface {
	Create(ctx context.Context, req *model.InviteRequest) error
	FindByID(ctx context.Context, id string) (*model.InviteRequest, error)
	HasPendingByEmail(ctx context.Context, email string) (bool, error)
	List(ctx context.Context, status string, page, perPage int) ([]model.InviteRequest, int64, error)
	CountPending(ctx context.Context) (int64, error)
	MarkApproved(ctx context.Context, id, inviteID, actorID, note string, reviewedAt time.Time) error
	MarkRejected(ctx context.Context, id, actorID, reason string, reviewedAt time.Time) error
}

// InviteEmailEnqueuer menitipkan surel undangan ke antrean email outbox.
type InviteEmailEnqueuer interface {
	EnqueueWithToken(ctx context.Context, row *model.EmailOutbox, rawToken string) error
}

// InviteRequestService menangani pengajuan undangan dari halaman publik.
type InviteRequestService interface {
	Submit(ctx context.Context, req dto.SubmitInviteRequest) error
}

// AdminInviteRequestService menangani tinjauan permohonan undangan di panel admin.
type AdminInviteRequestService interface {
	List(ctx context.Context, status string, page, perPage int) (*dto.InviteRequestList, dto.PaginationMeta, error)
	Approve(ctx context.Context, actor AdminActor, id string, req dto.ApproveInviteRequestBody) (*dto.ApprovedInviteRequestResult, error)
	Reject(ctx context.Context, actor AdminActor, id string, reason string) error
}

type inviteRequestService struct {
	store       InviteRequestStore
	invites     AdminInviteService
	outbox      InviteEmailEnqueuer
	smtpEnabled bool
	audit       AdminAudit
	logger      *slog.Logger
	clock       func() time.Time
}

// NewInviteRequestService merangkai layanan permohonan undangan.
func NewInviteRequestService(
	store InviteRequestStore,
	invites AdminInviteService,
	outbox InviteEmailEnqueuer,
	smtpEnabled bool,
	recorder AdminAudit,
	logger *slog.Logger,
) *inviteRequestService {
	if logger == nil {
		logger = slog.Default()
	}
	return &inviteRequestService{
		store:       store,
		invites:     invites,
		outbox:      outbox,
		smtpEnabled: smtpEnabled,
		audit:       recorder,
		logger:      logger,
		clock:       func() time.Time { return time.Now().UTC() },
	}
}

// Submit mencatat permohonan akses dari beranda publik.
func (s *inviteRequestService) Submit(ctx context.Context, req dto.SubmitInviteRequest) error {
	// Field honeypot: bila terisi oleh bot, kembalikan sukses diam-diam tanpa menulis ke DB.
	if strings.TrimSpace(req.Website) != "" {
		return nil
	}

	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	contact := strings.TrimSpace(req.Contact)
	note := strings.TrimSpace(req.Note)

	if len(name) < 2 {
		return validation("nama minimal 2 karakter")
	}
	if !strings.Contains(email, "@") {
		return validation("alamat email tidak sah")
	}

	exists, err := s.store.HasPendingByEmail(ctx, email)
	if err != nil {
		return err
	}
	if exists {
		return errors.Join(apperror.ErrConflict, errors.New("Permohonan dengan alamat email ini sudah masuk dan sedang ditinjau."))
	}

	row := &model.InviteRequest{
		Name:    name,
		Email:   email,
		Contact: contact,
		Note:    note,
		Status:  model.InviteRequestPending,
	}
	return s.store.Create(ctx, row)
}

// List mengembalikan daftar permohonan berpaginasi beserta jumlah pending.
func (s *inviteRequestService) List(ctx context.Context, status string, page, perPage int) (*dto.InviteRequestList, dto.PaginationMeta, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	rows, total, err := s.store.List(ctx, strings.TrimSpace(status), page, perPage)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}
	pending, err := s.store.CountPending(ctx)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	items := make([]dto.InviteRequestItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toInviteRequestItem(row))
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages < 1 && total > 0 {
		totalPages = 1
	}

	meta := dto.PaginationMeta{
		CurrentPage: page,
		PerPage:     perPage,
		TotalItems:  total,
		TotalPages:  totalPages,
	}
	return &dto.InviteRequestList{
		Items:        items,
		PendingCount: pending,
	}, meta, nil
}

// Approve menyetujui permohonan, membuat tautan undangan terkunci ke email pemohon,
// lalu menitipkan surel ke email outbox bila SMTP aktif.
func (s *inviteRequestService) Approve(ctx context.Context, actor AdminActor, id string, req dto.ApproveInviteRequestBody) (*dto.ApprovedInviteRequestResult, error) {
	row, err := s.store.FindByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if row.Status != model.InviteRequestPending {
		return nil, errors.Join(apperror.ErrConflict, errors.New("Permohonan ini sudah diproses sebelumnya."))
	}

	now := s.clock()
	tier := strings.TrimSpace(req.Tier)
	if tier == "" {
		tier = model.TierPlus
	}
	expiresDays := req.ExpiresDays
	if expiresDays <= 0 {
		expiresDays = 7
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		note = fmt.Sprintf("Permohonan akses: %s", row.Name)
		if row.Contact != "" {
			note = fmt.Sprintf("%s (%s)", note, row.Contact)
		}
		if len(note) > 255 {
			note = note[:255]
		}
	}

	created, err := s.invites.Create(ctx, actor, dto.CreateInviteRequest{
		Tier:      tier,
		MaxUses:   1,
		ExpiresAt: now.Add(time.Duration(expiresDays) * 24 * time.Hour),
		Email:     row.Email,
		Note:      note,
	})
	if err != nil {
		return nil, err
	}

	if err := s.store.MarkApproved(ctx, row.ID, created.ID, actor.ID, note, now); err != nil {
		// Jika terjadi race condition karena admin lain menyetujui duluan, cabut
		// undangan yang barusan dibuat agar tidak ada tautan yatim.
		_ = s.invites.Revoke(ctx, actor, created.ID, "batal: permohonan sudah diproses admin lain")
		return nil, err
	}

	emailQueued := false
	if s.smtpEnabled && s.outbox != nil {
		payload, _ := json.Marshal(map[string]string{
			"name":      row.Name,
			"invite_id": created.ID,
		})
		errQueue := s.outbox.EnqueueWithToken(ctx, &model.EmailOutbox{
			ToEmail:  row.Email,
			Template: templateInviteApproved,
			Payload:  string(payload),
			Status:   model.EmailPending,
		}, created.Code)
		if errQueue != nil {
			s.logger.Warn("gagal menitipkan surel undangan ke antrean", "request_id", row.ID, "error", errQueue.Error())
		} else {
			emailQueued = true
		}
	}

	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{
			ActorID: actor.ID, Action: audit.InviteRequestApprove,
			TargetType: "invite_request", TargetID: row.ID,
			RequestID: actor.RequestID,
			After: map[string]any{
				"email":        row.Email,
				"invite_id":    created.ID,
				"tier":         tier,
				"email_queued": emailQueued,
			},
		}); err != nil {
			return nil, err
		}
	}

	row.Status = model.InviteRequestApproved
	row.InviteID = created.ID
	row.ReviewedBy = actor.ID
	row.ReviewedAt = &now
	row.ReviewNote = note

	return &dto.ApprovedInviteRequestResult{
		Request:     toInviteRequestItem(*row),
		Invite:      *created,
		EmailQueued: emailQueued,
	}, nil
}

// Reject menolak permohonan akses yang masih pending.
func (s *inviteRequestService) Reject(ctx context.Context, actor AdminActor, id string, reason string) error {
	row, err := s.store.FindByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if row.Status != model.InviteRequestPending {
		return errors.Join(apperror.ErrConflict, errors.New("Permohonan ini sudah diproses sebelumnya."))
	}

	now := s.clock()
	cleanReason := strings.TrimSpace(reason)
	if err := s.store.MarkRejected(ctx, row.ID, actor.ID, cleanReason, now); err != nil {
		return err
	}

	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{
			ActorID: actor.ID, Action: audit.InviteRequestReject,
			TargetType: "invite_request", TargetID: row.ID,
			RequestID: actor.RequestID,
			Reason:    cleanReason,
			After:     map[string]any{"email": row.Email},
		}); err != nil {
			return err
		}
	}
	return nil
}

func toInviteRequestItem(row model.InviteRequest) dto.InviteRequestItem {
	return dto.InviteRequestItem{
		ID:         row.ID,
		Name:       row.Name,
		Email:      row.Email,
		Contact:    row.Contact,
		Note:       row.Note,
		Status:     row.Status,
		InviteID:   row.InviteID,
		ReviewedBy: row.ReviewedBy,
		ReviewedAt: row.ReviewedAt,
		ReviewNote: row.ReviewNote,
		CreatedAt:  row.CreatedAt,
	}
}
