package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// TwoFactorGate menentukan apakah sebuah akun menuntut kode 2FA saat masuk.
// Dipisahkan dari TwoFactorService agar layanan auth tidak perlu tahu detail
// TOTP, dan agar pintu admin maupun user memakai aturan yang sama.
type TwoFactorGate interface {
	// Enabled menandai 2FA aktif untuk akun ini.
	Enabled(ctx context.Context, userID string) (bool, error)
	// Verify memeriksa kode saat masuk; galat berarti kode tidak sah.
	Verify(ctx context.Context, userID, code string) error
	// ConsumeRecovery memakai satu kode pemulihan.
	ConsumeRecovery(ctx context.Context, userID, code string) (bool, error)
}

// ChallengeRepository menyimpan keadaan antara password benar dan 2FA berhasil.
type ChallengeRepository interface {
	CreateChallenge(ctx context.Context, challenge *model.TwoFactorChallenge) error
	FindChallenge(ctx context.Context, id string) (*model.TwoFactorChallenge, error)
	IncrementChallengeAttempts(ctx context.Context, id string) (int, error)
	ConsumeChallenge(ctx context.Context, id string, at time.Time) error
}

// challengeService mengurus tantangan 2FA: membuat, memeriksa, dan menutup.
type challengeService struct {
	repo ChallengeRepository
	gate TwoFactorGate
	now  func() time.Time
}

// newChallengeService merangkai pengelola tantangan 2FA.
func newChallengeService(repo ChallengeRepository, gate TwoFactorGate) *challengeService {
	return &challengeService{
		repo: repo,
		gate: gate,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// setClock menyamakan sumber waktu dengan pemiliknya agar masa berlaku
// tantangan dihitung dengan jam yang sama.
func (s *challengeService) setClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

// Issue membuat tantangan baru setelah password terbukti benar. ID yang
// dikembalikan tidak berguna untuk mengakses data apa pun.
func (s *challengeService) Issue(ctx context.Context, userID, purpose string) (string, error) {
	id, err := newChallengeID()
	if err != nil {
		return "", err
	}

	now := s.now()
	challenge := &model.TwoFactorChallenge{
		ID:        id,
		UserID:    userID,
		Purpose:   purpose,
		ExpiresAt: now.Add(ChallengeTTL),
		CreatedAt: now,
	}
	if err := s.repo.CreateChallenge(ctx, challenge); err != nil {
		return "", err
	}
	return id, nil
}

// Owner membaca pemilik dan tujuan tantangan tanpa memakainya.
func (s *challengeService) Owner(ctx context.Context, challengeID, purpose string) (string, error) {
	challenge, err := s.repo.FindChallenge(ctx, challengeID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return "", apperror.ErrUnauthorized
		}
		return "", err
	}
	if challenge.Purpose != purpose {
		return "", apperror.ErrUnauthorized
	}
	if challenge.ConsumedAt != nil || s.now().After(challenge.ExpiresAt) {
		return "", apperror.ErrUnauthorized
	}
	return challenge.UserID, nil
}

// Redeem memeriksa kode 2FA atau kode pemulihan untuk sebuah tantangan.
// Tantangan sekali pakai: berhasil berarti tertutup.
func (s *challengeService) Redeem(ctx context.Context, challengeID, userID, purpose, code string) error {
	challenge, err := s.repo.FindChallenge(ctx, challengeID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return apperror.ErrUnauthorized
		}
		return err
	}

	if challenge.UserID != userID || challenge.Purpose != purpose {
		return apperror.ErrUnauthorized
	}
	if challenge.ConsumedAt != nil {
		return apperror.ErrUnauthorized
	}
	if s.now().After(challenge.ExpiresAt) {
		return apperror.ErrUnauthorized
	}
	if challenge.Attempts >= MaxChallengeAttempts {
		// Tantangan hangus: mulai lagi dari kata sandi.
		return apperror.ErrUnauthorized
	}

	if err := s.check(ctx, userID, code); err != nil {
		if errors.Is(err, apperror.ErrBadRequest) {
			if _, incErr := s.repo.IncrementChallengeAttempts(ctx, challenge.ID); incErr != nil {
				return incErr
			}
		}
		return err
	}

	return s.repo.ConsumeChallenge(ctx, challenge.ID, s.now())
}

// check mencoba kode aplikasi lebih dulu, lalu kode pemulihan.
func (s *challengeService) check(ctx context.Context, userID, code string) error {
	if err := s.gate.Verify(ctx, userID, code); err == nil {
		return nil
	} else if !errors.Is(err, apperror.ErrBadRequest) {
		return err
	}

	used, err := s.gate.ConsumeRecovery(ctx, userID, code)
	if err != nil {
		return err
	}
	if !used {
		return validation("kode 2FA atau kode pemulihan tidak sesuai")
	}
	return nil
}

// newChallengeID membuat penanda tantangan acak 32 byte.
func newChallengeID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.New("gagal membuat tantangan 2FA")
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// LoginStep memberi tahu klien apa yang harus dilakukan berikutnya.
type LoginStep string

const (
	// StepComplete berarti sesi sudah terbit (2FA tidak aktif).
	StepComplete LoginStep = "complete"
	// StepTwoFactor berarti klien harus mengirim kode 2FA.
	StepTwoFactor LoginStep = "two_factor_required"
)

// challengeInfo adalah hasil pemeriksaan 2FA sebelum menerbitkan sesi.
type challengeInfo struct {
	Step      LoginStep
	Challenge string
	ExpiresAt time.Time
}

// requireTwoFactor memutuskan apakah akan menerbitkan sesi atau tantangan.
// Gate nil berarti 2FA tidak tersedia: login berjalan seperti biasa, bukan gagal.
func requireTwoFactor(ctx context.Context, gate TwoFactorGate, challenges *challengeService, userID, purpose string) (challengeInfo, error) {
	if gate == nil || challenges == nil {
		return challengeInfo{Step: StepComplete}, nil
	}

	enabled, err := gate.Enabled(ctx, userID)
	if err != nil {
		return challengeInfo{}, err
	}
	if !enabled {
		return challengeInfo{Step: StepComplete}, nil
	}

	id, err := challenges.Issue(ctx, userID, purpose)
	if err != nil {
		return challengeInfo{}, err
	}

	return challengeInfo{
		Step:      StepTwoFactor,
		Challenge: id,
		ExpiresAt: time.Now().UTC().Add(ChallengeTTL),
	}, nil
}

// dto untuk balasan login yang menuntut 2FA.
type twoFactorRequiredResponse struct {
	Step        string `json:"step"`
	ChallengeID string `json:"challenge_id"`
	ExpiresAt   string `json:"expires_at"`
	ExpiresIn   int    `json:"expires_in_seconds"`
	Message     string `json:"message"`
}

func newTwoFactorRequired(info challengeInfo) twoFactorRequiredResponse {
	return twoFactorRequiredResponse{
		Step:        string(StepTwoFactor),
		ChallengeID: info.Challenge,
		ExpiresAt:   info.ExpiresAt.Format(time.RFC3339),
		ExpiresIn:   int(ChallengeTTL.Seconds()),
		Message:     "Masukkan kode dari aplikasi authenticator.",
	}
}

// memastikan duaFactorService memenuhi kontrak TwoFactorGate.
var _ TwoFactorGate = (TwoFactorService)(nil)
