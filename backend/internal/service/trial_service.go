package service

import (
	"context"
	"errors"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// TrialDuration adalah masa uji coba (plans/14 par 5.5.1).
const TrialDuration = 7 * 24 * time.Hour

// TrialService mengaktifkan uji coba 7 hari untuk user.
//
// Trial setara Plus tetapi seluruh eksekusinya simulasi: uji coba hanya
// menilai kualitas sinyal, bukan menaruh modal (plans/14 par 5.5.2).
type TrialService interface {
	Start(ctx context.Context, userID string) (*dto.TrialResponse, error)
	Status(ctx context.Context, userID string) (*dto.TrialResponse, error)
}

type trialService struct {
	users UserRepository
	audit AuditRepository
	clock func() time.Time
}

// NewTrialService membuat service uji coba.
func NewTrialService(users UserRepository, audit AuditRepository) TrialService {
	return &trialService{
		users: users,
		audit: audit,
		clock: func() time.Time { return time.Now().UTC() },
	}
}

// Start mengaktifkan trial sekali per user. Percobaan kedua ditolak.
func (s *trialService) Start(ctx context.Context, userID string) (*dto.TrialResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.TrialUsed {
		return nil, errors.Join(apperror.ErrConflict, errors.New("Uji coba hanya tersedia sekali per akun."))
	}
	if InTrialPeriod(user, s.clock()) {
		return nil, errors.Join(apperror.ErrConflict, errors.New("Uji coba Anda masih berjalan."))
	}

	now := s.clock()
	expiresAt := now.Add(TrialDuration)

	user.TrialUsed = true
	user.TrialStartedAt = &now
	user.TrialExpiresAt = &expiresAt

	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, user.ID, "trial.start", "user", user.ID)

	return toTrialResponse(user, now), nil
}

// Status melaporkan sisa masa uji coba.
func (s *trialService) Status(ctx context.Context, userID string) (*dto.TrialResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toTrialResponse(user, s.clock()), nil
}

func (s *trialService) recordAudit(ctx context.Context, userID, event, entity, entityID string) {
	_ = s.audit.Record(ctx, &model.AuditLog{
		UserID:    userID,
		Event:     event,
		Entity:    entity,
		EntityID:  entityID,
		CreatedAt: s.clock(),
	})
}

// InTrialPeriod menandai user yang sedang di dalam masa uji coba.
func InTrialPeriod(user *model.User, now time.Time) bool {
	if user == nil || user.TrialExpiresAt == nil {
		return false
	}
	return now.Before(*user.TrialExpiresAt)
}

func toTrialResponse(user *model.User, now time.Time) *dto.TrialResponse {
	out := &dto.TrialResponse{
		Used:           user.TrialUsed,
		Active:         InTrialPeriod(user, now),
		DurationDays:   int(TrialDuration.Hours() / 24),
		TrialStartedAt: user.TrialStartedAt,
		TrialExpiresAt: user.TrialExpiresAt,
		DryRunOnly:     true,
	}
	if out.Active && user.TrialExpiresAt != nil {
		// Sisa hari dibulatkan ke atas: 6 hari 3 jam dihitung 7 hari.
		remaining := user.TrialExpiresAt.Sub(now)
		out.DaysRemaining = int((remaining + 24*time.Hour - 1) / (24 * time.Hour))
	}
	return out
}
