package service

import (
	"context"
	"errors"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

// sessionRecorder menyimpan sesi baru ke tabel session.
type sessionRecorder struct {
	repo repository.SessionRepository
	now  func() time.Time
}

// NewSessionRecorder membuat pencatat sesi di atas repository.
func NewSessionRecorder(repo repository.SessionRepository) SessionRecorder {
	return &sessionRecorder{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// Record menyimpan satu sesi. Bila tokenID kosong, penanda dibuat acak.
func (r *sessionRecorder) Record(ctx context.Context, userID, tokenID, scope, userAgent, ipHash string, expiresAt time.Time) (string, error) {
	if tokenID == "" {
		generated, err := repository.NewTokenID()
		if err != nil {
			return "", err
		}
		tokenID = generated
	}

	record := &model.Session{
		UserID:     userID,
		Scope:      scope,
		TokenID:    tokenID,
		IPHash:     ipHash,
		UserAgent:  truncate(userAgent, 255),
		LastSeenAt: r.now(),
		ExpiresAt:  expiresAt,
	}

	if err := r.repo.Create(ctx, record); err != nil {
		return "", errors.Join(err)
	}
	return record.ID, nil
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
