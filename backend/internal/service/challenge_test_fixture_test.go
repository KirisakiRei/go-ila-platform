package service

import (
	"context"
	"time"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// notFound mengembalikan galat standar untuk data yang tidak ada.
func notFound() error {
	return apperror.ErrNotFound
}

// fakeChallengeStore menyimpan tantangan 2FA di memori.
type fakeChallengeStore struct {
	rows map[string]*model.TwoFactorChallenge
}

func newFakeChallengeStore() *fakeChallengeStore {
	return &fakeChallengeStore{rows: make(map[string]*model.TwoFactorChallenge)}
}

func (f *fakeChallengeStore) CreateChallenge(_ context.Context, challenge *model.TwoFactorChallenge) error {
	copied := *challenge
	f.rows[challenge.ID] = &copied
	return nil
}

func (f *fakeChallengeStore) FindChallenge(_ context.Context, id string) (*model.TwoFactorChallenge, error) {
	row, ok := f.rows[id]
	if !ok {
		return nil, notFound()
	}
	copied := *row
	return &copied, nil
}

func (f *fakeChallengeStore) IncrementChallengeAttempts(_ context.Context, id string) (int, error) {
	row, ok := f.rows[id]
	if !ok {
		return 0, notFound()
	}
	row.Attempts++
	return row.Attempts, nil
}

func (f *fakeChallengeStore) ConsumeChallenge(_ context.Context, id string, at time.Time) error {
	row, ok := f.rows[id]
	if !ok {
		return notFound()
	}
	row.ConsumedAt = &at
	return nil
}
