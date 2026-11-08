package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// Masa berlaku tautan surel (plans/17 §3).
const (
	verifyTokenTTL = 24 * time.Hour
	resetTokenTTL  = 30 * time.Minute
	// tokenRateWindow membatasi penerbitan tautan per alamat surel.
	tokenRateWindow = time.Hour
	tokenRateMax    = 3
)

// Nama template surel yang dikenali worker.
const (
	templateVerify = "verify_email"
	templateReset  = "reset_password"
)

// EmailAccountStore adalah kebutuhan penyimpanan alur akun.
type EmailAccountStore interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	UpdatePassword(ctx context.Context, id, passwordHash string) error
	MarkEmailVerified(ctx context.Context, userID string) error
	Update(ctx context.Context, user *model.User) error
	Enqueue(ctx context.Context, row *model.EmailOutbox) error
	EnqueueWithToken(ctx context.Context, row *model.EmailOutbox, rawToken string) error
	CreateToken(ctx context.Context, token *model.EmailToken) error
	FindToken(ctx context.Context, hash, purpose string) (*model.EmailToken, error)
	ConsumeToken(ctx context.Context, id string) (bool, error)
	CountTokensSince(ctx context.Context, userID, purpose string, since time.Time) (int64, error)
	ListByUser(ctx context.Context, userID string) ([]model.BrokerAccount, error)
	ListActiveByUser(ctx context.Context, userID string) ([]model.Subscription, error)
}

// EmailAccountService menjalankan verifikasi email, lupa sandi, dan onboarding.
type EmailAccountService interface {
	ForgotPassword(ctx context.Context, email, ipHash string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	VerifyEmail(ctx context.Context, token string) error
	ResendVerification(ctx context.Context, userID string) error
	Onboarding(ctx context.Context, userID string) (*dto.OnboardingState, error)
	SaveOnboarding(ctx context.Context, userID string, req dto.OnboardingRequest) (*dto.OnboardingState, error)
	Banner(ctx context.Context, userID string) (*dto.SubscriptionBanner, error)
}

type emailAccountService struct {
	store EmailAccountStore
	audit interface {
		Record(ctx context.Context, entry *model.AuditLog) error
	}
	baseURL string
	now     func() time.Time
}

// NewEmailAccountService merangkai layanan alur akun.
func NewEmailAccountService(store EmailAccountStore, recorder interface {
	Record(ctx context.Context, entry *model.AuditLog) error
}, baseURL string) EmailAccountService {
	return &emailAccountService{
		store:   store,
		audit:   recorder,
		baseURL: strings.TrimRight(baseURL, "/"),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// ForgotPassword selalu melaporkan sukses. Perbedaan hasil tidak boleh
// terlihat dari luar agar alamat surel tidak bisa dipetakan (plans/17 §3.2).
func (s *emailAccountService) ForgotPassword(ctx context.Context, email, ipHash string) error {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return nil
	}
	user, err := s.store.FindByEmail(ctx, normalized)
	if err != nil || user == nil {
		return nil
	}
	// Akun terhapus atau diblokir tidak dikirimi tautan reset.
	if user.DeletedAtUser != nil || user.Status == model.UserStatusBanned {
		return nil
	}

	count, err := s.store.CountTokensSince(ctx, user.ID, model.TokenPurposeReset, s.now().Add(-tokenRateWindow))
	if err != nil {
		return nil
	}
	if count >= tokenRateMax {
		return nil
	}

	raw, hash, err := newToken()
	if err != nil {
		return err
	}
	token := &model.EmailToken{
		UserID:    user.ID,
		Purpose:   model.TokenPurposeReset,
		TokenHash: hash,
		ExpiresAt: s.now().Add(resetTokenTTL),
		CreatedAt: s.now(),
	}
	if err := s.store.CreateToken(ctx, token); err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]string{
		// Payload tidak memuat tautan bertoken: kolom ini bisa terbaca dari
		// backup dan panel dukungan (plans/17 §2).
		"token_id": token.ID,
		"name":     user.Name,
	})
	if err := s.store.EnqueueWithToken(ctx, &model.EmailOutbox{
		ToEmail:  user.Email,
		Template: templateReset,
		Payload:  string(payload),
		Status:   model.EmailPending,
	}, raw); err != nil {
		return err
	}

	s.record(ctx, user.ID, audit.PasswordChange, ipHash, map[string]any{"step": "reset_requested"})
	return nil
}

// ResetPassword menukar token menjadi kata sandi baru, lalu mencabut seluruh
// sesi supaya perangkat lain langsung keluar (plans/17 §3.3).
func (s *emailAccountService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	row, err := s.store.FindToken(ctx, hashToken(token), model.TokenPurposeReset)
	if err != nil {
		return tokenError(apperror.ErrBadRequest, "Tautan tidak sah")
	}
	if row.UsedAt != nil {
		return tokenError(apperror.ErrBadRequest, "Tautan sudah pernah dipakai")
	}
	if s.now().After(row.ExpiresAt) {
		return tokenError(apperror.ErrBadRequest, "Tautan sudah kedaluwarsa")
	}
	ok, err := s.store.ConsumeToken(ctx, row.ID)
	if err != nil {
		return err
	}
	if !ok {
		return tokenError(apperror.ErrBadRequest, "Tautan sudah pernah dipakai")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("membuat hash sandi: %w", err)
	}
	if err := s.store.UpdatePassword(ctx, row.UserID, string(hash)); err != nil {
		return err
	}
	s.record(ctx, row.UserID, audit.PasswordChange, "", map[string]any{"step": "reset_done"})
	return nil
}

// VerifyEmail menandai alamat terbukti milik pengguna.
func (s *emailAccountService) VerifyEmail(ctx context.Context, token string) error {
	row, err := s.store.FindToken(ctx, hashToken(token), model.TokenPurposeVerify)
	if err != nil {
		return tokenError(apperror.ErrBadRequest, "Tautan tidak sah")
	}
	if row.UsedAt != nil {
		return tokenError(apperror.ErrBadRequest, "Tautan sudah pernah dipakai")
	}
	if s.now().After(row.ExpiresAt) {
		return tokenError(apperror.ErrBadRequest, "Tautan sudah kedaluwarsa")
	}
	ok, err := s.store.ConsumeToken(ctx, row.ID)
	if err != nil {
		return err
	}
	if !ok {
		return tokenError(apperror.ErrBadRequest, "Tautan sudah pernah dipakai")
	}
	if err := s.store.MarkEmailVerified(ctx, row.UserID); err != nil {
		return err
	}
	s.record(ctx, row.UserID, audit.LoginSuccess, "", map[string]any{"step": "email_verified"})
	return nil
}

// ResendVerification menerbitkan tautan verifikasi baru.
func (s *emailAccountService) ResendVerification(ctx context.Context, userID string) error {
	user, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.EmailVerifiedAt != nil {
		return validation("email sudah terverifikasi")
	}
	count, err := s.store.CountTokensSince(ctx, user.ID, model.TokenPurposeVerify, s.now().Add(-tokenRateWindow))
	if err != nil {
		return err
	}
	if count >= tokenRateMax {
		return tokenError(apperror.ErrConflict, "Terlalu banyak permintaan. Coba lagi nanti.")
	}

	raw, hash, err := newToken()
	if err != nil {
		return err
	}
	token := &model.EmailToken{
		UserID:    user.ID,
		Purpose:   model.TokenPurposeVerify,
		TokenHash: hash,
		ExpiresAt: s.now().Add(verifyTokenTTL),
		CreatedAt: s.now(),
	}
	if err := s.store.CreateToken(ctx, token); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{
		"token_id": token.ID,
		"name":     user.Name,
	})
	return s.store.EnqueueWithToken(ctx, &model.EmailOutbox{
		ToEmail:  user.Email,
		Template: templateVerify,
		Payload:  string(payload),
		Status:   model.EmailPending,
	}, raw)
}

func (s *emailAccountService) record(ctx context.Context, userID string, action audit.Action, ipHash string, meta map[string]any) {
	if s.audit == nil {
		return
	}
	encoded := ""
	if len(meta) > 0 {
		if raw, err := json.Marshal(audit.Scrub(meta)); err == nil {
			encoded = string(raw)
		}
	}
	_ = s.audit.Record(ctx, &model.AuditLog{
		UserID:    userID,
		ActorID:   userID,
		ActorType: "user",
		Event:     string(action),
		Action:    string(action),
		Metadata:  encoded,
		IPHash:    ipHash,
		CreatedAt: s.now(),
	})
}

// tokenError membungkus kode HTTP dengan pesan yang bisa dibaca pengguna.
func tokenError(base error, message string) error {
	return errors.Join(base, errors.New(message))
}

// validatePassword menerapkan aturan panjang minimal kata sandi baru.
func validatePassword(password string) error {
	if len([]rune(password)) < 8 {
		return errors.Join(apperror.ErrBadRequest, errors.New("kata sandi minimal 8 karakter"))
	}
	return nil
}

// newToken membuat token acak beserta hash penyimpanannya.
func newToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("membuat token: %w", err)
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashToken(raw), nil
}

// hashToken memakai SHA-256: token acak 256 bit tidak butuh hash lambat.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}
