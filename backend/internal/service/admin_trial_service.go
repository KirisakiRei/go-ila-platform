package service

import (
	"context"
	"errors"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// AdminTrialService mengelola uji coba dari panel admin.
//
// Trial adalah satu kali per akun. Admin hanya boleh memperpanjang atau
// menghentikan; memberi trial baru ke akun yang sudah pernah memakainya akan
// membuka pintu penyalahgunaan (daftar ulang berulang).
type AdminTrialService interface {
	Trials(ctx context.Context, status string, limit int) ([]dto.AdminTrialItem, error)
	Extend(ctx context.Context, actor AdminActor, userID string, req dto.AdminTrialExtendRequest) (*dto.AdminTrialItem, error)
	End(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error
}

type adminTrialService struct {
	store trialAdminStore
	audit AdminAudit
	now   func() time.Time
}

// trialAdminStore adalah data yang dibutuhkan kelola trial.
type trialAdminStore interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	ListTrials(ctx context.Context, status string, limit int) ([]model.User, error)
}

// NewAdminTrialService merangkai kelola trial.
func NewAdminTrialService(store trialAdminStore, recorder AdminAudit) AdminTrialService {
	return &adminTrialService{
		store: store, audit: recorder,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// Trials membaca pengguna yang memakai atau pernah memakai uji coba.
func (s *adminTrialService) Trials(ctx context.Context, status string, limit int) ([]dto.AdminTrialItem, error) {
	rows, err := s.store.ListTrials(ctx, status, limit)
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]dto.AdminTrialItem, 0, len(rows))
	for i := range rows {
		out = append(out, toAdminTrialItem(&rows[i], now))
	}
	return out, nil
}

// Extend menambah hari uji coba yang masih berjalan.
func (s *adminTrialService) Extend(ctx context.Context, actor AdminActor, userID string, req dto.AdminTrialExtendRequest) (*dto.AdminTrialItem, error) {
	if err := ValidateReason(audit.UserTierChange, req.Reason); err != nil {
		return nil, err
	}
	if req.Days <= 0 || req.Days > 30 {
		return nil, validation("perpanjangan trial antara 1 dan 30 hari")
	}
	// Memperpanjang trial sendiri berarti memberi diri akses tanpa bayar.
	if actor.ID == userID {
		return nil, validation("tidak dapat memperpanjang trial akun sendiri")
	}

	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if user.TrialExpiresAt == nil || !now.Before(*user.TrialExpiresAt) {
		return nil, errors.Join(apperror.ErrConflict, errors.New("trial akun ini tidak sedang berjalan"))
	}

	before := *user.TrialExpiresAt
	after := before.AddDate(0, 0, req.Days)
	user.TrialExpiresAt = &after
	if err := s.store.Update(ctx, user); err != nil {
		return nil, err
	}

	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserTierChange,
		TargetType: "user", TargetID: userID, Reason: req.Reason, RequestID: actor.RequestID,
		Before: map[string]any{"trial_expires_at": before},
		After:  map[string]any{"trial_expires_at": after, "days": req.Days},
	}); err != nil {
		return nil, err
	}

	item := toAdminTrialItem(user, now)
	return &item, nil
}

// End menghentikan trial lebih awal tanpa menghapus jejak pemakaiannya.
func (s *adminTrialService) End(ctx context.Context, actor AdminActor, userID string, req dto.UserActionRequest) error {
	reason, _, err := destructiveReason(audit.UserTierChange, req)
	if err != nil {
		return err
	}
	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	now := s.now()
	if user.TrialExpiresAt == nil || !now.Before(*user.TrialExpiresAt) {
		return nil
	}

	ended := now
	user.TrialExpiresAt = &ended
	if err := s.store.Update(ctx, user); err != nil {
		return err
	}
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserTierChange,
		TargetType: "user", TargetID: userID, Reason: reason, RequestID: actor.RequestID,
		Before: map[string]any{"trial_active": true},
		After:  map[string]any{"trial_active": false},
	})
}

// toAdminTrialItem menentukan status trial untuk tampilan admin.
func toAdminTrialItem(user *model.User, now time.Time) dto.AdminTrialItem {
	item := dto.AdminTrialItem{
		UserID: user.ID, Email: user.Email, Name: user.Name, Tier: user.Tier,
		TrialUsed: user.TrialUsed, Status: "never",
		TierDuringTrial: model.TierTrial,
		AccountLimit:    model.MaxAccountsDuringTrial,
	}
	if user.TrialStartedAt != nil {
		pair := dto.NewTimePair(*user.TrialStartedAt)
		item.StartedAt = &pair
	}
	if user.TrialExpiresAt == nil {
		return item
	}
	pair := dto.NewTimePair(*user.TrialExpiresAt)
	item.ExpiresAt = &pair
	if now.Before(*user.TrialExpiresAt) {
		item.Status = "active"
		remaining := user.TrialExpiresAt.Sub(now)
		item.DaysRemaining = int((remaining + 24*time.Hour - 1) / (24 * time.Hour))
		return item
	}
	item.Status = "ended"
	return item
}
