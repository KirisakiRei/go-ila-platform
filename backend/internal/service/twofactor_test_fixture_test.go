package service

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/totp"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/cryptox"
)

// fakeTwoFactorStore menyimpan 2FA di memori.
type fakeTwoFactorStore struct {
	settings map[string]*model.TwoFactor
	codes    map[string][]model.RecoveryCode
}

func newFakeTwoFactorStore() *fakeTwoFactorStore {
	return &fakeTwoFactorStore{
		settings: make(map[string]*model.TwoFactor),
		codes:    make(map[string][]model.RecoveryCode),
	}
}

func (f *fakeTwoFactorStore) FindByUser(_ context.Context, userID string) (*model.TwoFactor, error) {
	settings, ok := f.settings[userID]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	copied := *settings
	return &copied, nil
}

func (f *fakeTwoFactorStore) Upsert(_ context.Context, settings *model.TwoFactor) error {
	copied := *settings
	f.settings[settings.UserID] = &copied
	return nil
}

func (f *fakeTwoFactorStore) Deactivate(_ context.Context, userID string) error {
	delete(f.settings, userID)
	delete(f.codes, userID)
	return nil
}

func (f *fakeTwoFactorStore) MarkUsed(_ context.Context, userID string, step int64, at time.Time) error {
	if settings, ok := f.settings[userID]; ok {
		settings.LastUsedStep = step
		settings.LastUsedAt = &at
	}
	return nil
}

func (f *fakeTwoFactorStore) ReplaceRecoveryCodes(_ context.Context, userID string, hashes []string) error {
	rows := make([]model.RecoveryCode, 0, len(hashes))
	for _, hash := range hashes {
		rows = append(rows, model.RecoveryCode{UserID: userID, CodeHash: hash})
	}
	f.codes[userID] = rows
	return nil
}

func (f *fakeTwoFactorStore) ConsumeRecoveryCode(_ context.Context, userID, hash string, at time.Time) (bool, error) {
	rows := f.codes[userID]
	for i := range rows {
		if rows[i].CodeHash == hash && rows[i].UsedAt == nil {
			rows[i].UsedAt = &at
			f.codes[userID] = rows
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeTwoFactorStore) CountUnusedRecoveryCodes(_ context.Context, userID string) (int64, error) {
	var total int64
	for _, row := range f.codes[userID] {
		if row.UsedAt == nil {
			total++
		}
	}
	return total, nil
}

// testMasterKey adalah kunci uji 32 byte dalam base64.
const testMasterKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

// twoFactorFixture menyiapkan layanan 2FA dengan jam yang bisa digeser.
type twoFactorFixture struct {
	svc   TwoFactorService
	store *fakeTwoFactorStore
	clock *time.Time
}

func newTwoFactorFixture(t *testing.T) *twoFactorFixture {
	t.Helper()

	sealer, err := cryptox.NewSealer(1, testMasterKey)
	if err != nil {
		t.Fatalf("NewSealer error: %v", err)
	}

	store := newFakeTwoFactorStore()

	// Jam disimpan di fikstur agar tes bisa menggeser waktu (mis. periode
	// TOTP berikutnya) tanpa menunggu 30 detik sungguhan.
	clock := time.Unix(1_700_000_000, 0).UTC()
	svc := NewTwoFactorService(store, sealer).(*twoFactorService)
	svc.now = func() time.Time { return clock }

	return &twoFactorFixture{svc: svc, store: store, clock: &clock}
}

// codeFor menghitung kode TOTP yang sah pada waktu tertentu.
func codeFor(t *testing.T, secret string, at time.Time) string {
	t.Helper()

	code, err := totp.Generate(secret, at)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	return code
}
