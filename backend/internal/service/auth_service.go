package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ila/backend/internal/authstate"
	"ila/backend/internal/config"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

// AuthService menangani pendaftaran dan masuk akun.
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	Profile(ctx context.Context, userID string) (*dto.UserResponse, error)
	// ChangePassword mencabut semua sesi lain setelah password berganti.
	ChangePassword(ctx context.Context, userID string, req dto.ChangePasswordRequest) error
	// LogoutAll menaikkan session_version sehingga semua token lama ditolak.
	LogoutAll(ctx context.Context, userID string) error
	// SetIdleWarning mengatur apakah modal "masih di sini?" tampil.
	SetIdleWarning(ctx context.Context, userID string, enabled bool) error
	// VerifyTwoFactor menuntaskan masuk setelah kode 2FA diperiksa.
	VerifyTwoFactor(ctx context.Context, req dto.VerifyTwoFactorRequest, device LoginContext) (*dto.LoginResponse, error)
}

// SessionRecorder mencatat sesi baru agar user bisa melihat daftarnya.
// Opsional: bila nil, login tetap berjalan seperti sebelumnya.
type SessionRecorder interface {
	Record(ctx context.Context, userID, tokenID, scope, userAgent, ipHash string, expiresAt time.Time) (string, error)
}

type authService struct {
	userRepo    UserRepository
	auditRepo   AuditRepository
	tokenIssuer TokenIssuer
	sessions    SessionRecorder
	// legal memvalidasi centang Syarat & Ketentuan dan mencatat persetujuan.
	legal LegalSignupRecorder
	// gate & challenges membuat 2FA opsional untuk user.
	gate       TwoFactorGate
	challenges *challengeService
	now        func() time.Time
}

// NewAuthService merangkai layanan akun.
func NewAuthService(userRepo UserRepository, auditRepo AuditRepository, tokenIssuer TokenIssuer) AuthService {
	return &authService{
		userRepo:    userRepo,
		auditRepo:   auditRepo,
		tokenIssuer: tokenIssuer,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

// LegalSignupRecorder memvalidasi dan mencatat persetujuan saat mendaftar.
type LegalSignupRecorder interface {
	RequireSignup(req dto.RegisterRequest) error
	AcceptSignup(ctx context.Context, userID string, meta AcceptanceMeta) error
}

// WithTwoFactor menambahkan 2FA opsional ke pintu masuk user.
func WithTwoFactor(s AuthService, gate TwoFactorGate, challenges ChallengeRepository) AuthService {
	if svc, ok := s.(*authService); ok && gate != nil && challenges != nil {
		svc.gate = gate
		svc.challenges = newChallengeService(challenges, gate)
		// Jam disamakan agar masa berlaku tantangan dihitung konsisten.
		svc.challenges.setClock(svc.now)
	}
	return s
}

// WithLegal menghubungkan layanan legal ke alur pendaftaran.
func WithLegal(s AuthService, recorder LegalSignupRecorder) AuthService {
	if svc, ok := s.(*authService); ok {
		svc.legal = recorder
	}
	return s
}

// WithSessionRecorder menambahkan pencatatan sesi ke layanan autentikasi.
func WithSessionRecorder(s AuthService, recorder SessionRecorder) AuthService {
	if svc, ok := s.(*authService); ok {
		svc.sessions = recorder
	}
	return s
}

// LoginContext membawa informasi perangkat untuk pencatatan sesi.
type LoginContext struct {
	UserAgent string
	IPHash    string
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error) {
	// Persetujuan legal diperiksa sebelum menyentuh database: mendaftar tanpa
	// centang bukan kesalahan yang bisa diperbaiki setelah akun dibuat.
	if s.legal != nil {
		if err := s.legal.RequireSignup(req); err != nil {
			return nil, err
		}
	}

	email := normalizeEmail(req.Email)

	_, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		return nil, apperror.ErrAlreadyExists
	}
	if err != apperror.ErrNotFound {
		return nil, err
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	now := s.now()
	user := &model.User{
		Name:          strings.TrimSpace(req.Name),
		Email:         email,
		PasswordHash:  hash,
		Role:          string(config.RoleUser),
		Tier:          string(config.TierFree),
		Status:        "active",
		NotifyEnabled: true,
	}
	user.CreatedAt = now
	user.UpdatedAt = now

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, user.ID, "auth.register", "user", user.ID)

	// Persetujuan dicatat setelah akun ada. Kegagalan di sini tidak
	// menggagalkan pendaftaran, tetapi tercatat untuk ditindaklanjuti.
	if s.legal != nil {
		if err := s.legal.AcceptSignup(ctx, user.ID, AcceptanceMeta{}); err != nil {
			s.recordAudit(ctx, user.ID, "auth.legal_record_failed", "user", user.ID)
		}
	}

	return toUserResponse(user), nil
}

// Login dengan pencatatan perangkat kosong; dipakai pemanggil lama.
func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	return s.LoginWithDevice(ctx, req, LoginContext{})
}

// LoginWithDevice mencatat sesi perangkat baru agar bisa dilihat dan dicabut.
func (s *authService) LoginWithDevice(ctx context.Context, req dto.LoginRequest, device LoginContext) (*dto.LoginResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if err == apperror.ErrNotFound {
			// Pesan sengaja sama dengan password salah agar tidak membocorkan
			// email mana yang terdaftar.
			return nil, apperror.ErrUnauthorized
		}
		return nil, err
	}

	if user.Status != "active" {
		return nil, apperror.ErrForbidden
	}

	if !utils.ComparePassword(user.PasswordHash, req.Password) {
		s.recordAudit(ctx, user.ID, "auth.login_failed", "user", user.ID)
		return nil, apperror.ErrUnauthorized
	}

	// 2FA opsional: bila aktif, sesi belum terbit sampai kode diverifikasi.
	pending, err := requireTwoFactor(ctx, s.gate, s.challenges, user.ID, PurposeUser)
	if err != nil {
		return nil, err
	}
	if pending.Step == StepTwoFactor {
		detail := newTwoFactorRequired(pending)
		return &dto.LoginResponse{
			Step:        detail.Step,
			ChallengeID: detail.ChallengeID,
			ExpiresAt:   detail.ExpiresAt,
			ExpiresIn:   detail.ExpiresIn,
			User:        *toUserResponse(user),
		}, nil
	}

	expiresAt := s.now().Add(s.tokenIssuer.TTL())

	// Sesi dicatat lebih dulu agar token bisa membawa ID-nya. Bila pencatatan
	// gagal, login tetap jalan: pelacakan perangkat bukan syarat masuk.
	sessionID := ""
	if s.sessions != nil {
		id, err := s.sessions.Record(ctx, user.ID, "", utils.ScopeUser, device.UserAgent, device.IPHash, expiresAt)
		if err != nil {
			s.recordAudit(ctx, user.ID, "auth.session_record_failed", "user", user.ID)
		} else {
			sessionID = id
		}
	}

	token, expiresAt, err := s.tokenIssuer.IssueWithClaims(utils.Claims{
		UserID:    user.ID,
		Role:      user.Role,
		Scope:     utils.ScopeUser,
		Session:   user.SessionVersion,
		SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID, s.now()); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, user.ID, "auth.login", "user", user.ID)

	return &dto.LoginResponse{
		Step:      string(StepComplete),
		Token:     token,
		SessionID: sessionID,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User:      *toUserResponse(user),
	}, nil
}

// VerifyTwoFactor menuntaskan masuk user setelah kode 2FA diperiksa.
func (s *authService) VerifyTwoFactor(ctx context.Context, req dto.VerifyTwoFactorRequest, device LoginContext) (*dto.LoginResponse, error) {
	if s.challenges == nil {
		return nil, apperror.ErrUnauthorized
	}

	owner, err := s.challenges.Owner(ctx, req.ChallengeID, PurposeUser)
	if err != nil {
		return nil, err
	}

	if err := s.challenges.Redeem(ctx, req.ChallengeID, owner, PurposeUser, req.Code); err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, owner)
	if err != nil {
		return nil, apperror.ErrUnauthorized
	}
	// Keadaan akun bisa berubah selama tantangan berjalan.
	if !authstate.CanUsePlatform(user.Status) {
		return nil, apperror.ErrForbidden
	}

	return s.issueSession(ctx, user, device)
}

// issueSession menerbitkan token user beserta data akunnya.
func (s *authService) issueSession(ctx context.Context, user *model.User, device LoginContext) (*dto.LoginResponse, error) {
	expiresAt := s.now().Add(s.tokenIssuer.TTL())

	sessionID := ""
	if s.sessions != nil {
		id, err := s.sessions.Record(ctx, user.ID, "", utils.ScopeUser, device.UserAgent, device.IPHash, expiresAt)
		if err != nil {
			s.recordAudit(ctx, user.ID, "auth.session_record_failed", "user", user.ID)
		} else {
			sessionID = id
		}
	}

	token, expiresAt, err := s.tokenIssuer.IssueWithClaims(utils.Claims{
		UserID:    user.ID,
		Role:      user.Role,
		Scope:     utils.ScopeUser,
		Session:   user.SessionVersion,
		SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID, s.now()); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, user.ID, "auth.login", "user", user.ID)

	return &dto.LoginResponse{
		Step:      string(StepComplete),
		Token:     token,
		SessionID: sessionID,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User:      *toUserResponse(user),
	}, nil
}

func (s *authService) Profile(ctx context.Context, userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toUserResponse(user), nil
}

// SetIdleWarning hanya mengubah tampilan peringatan. Batas diam di server
// tidak ikut berubah: mematikannya tidak memperpanjang sesi (plans/23 §5).
func (s *authService) SetIdleWarning(ctx context.Context, userID string, enabled bool) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	user.IdleWarning = enabled
	return s.userRepo.Update(ctx, user)
}

func (s *authService) recordAudit(ctx context.Context, userID, event, entity, entityID string) {
	_ = s.auditRepo.Record(ctx, &model.AuditLog{
		UserID:    userID,
		Event:     event,
		Entity:    entity,
		EntityID:  entityID,
		CreatedAt: s.now(),
	})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func toUserResponse(user *model.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:          user.ID,
		Name:        user.Name,
		Email:       user.Email,
		Role:        user.Role,
		Tier:        user.Tier,
		IdleWarning: user.IdleWarning,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	}
}

// ChangePassword memverifikasi password lama, menyimpan hash baru, lalu
// mencabut semua sesi. Sesi perangkat ini ikut berakhir sehingga pengguna
// diminta masuk ulang dengan password baru.
func (s *authService) ChangePassword(ctx context.Context, userID string, req dto.ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == apperror.ErrNotFound {
			return apperror.ErrUnauthorized
		}
		return err
	}

	if !utils.ComparePassword(user.PasswordHash, req.CurrentPassword) {
		s.recordAudit(ctx, userID, "auth.password_failed", "user", userID)
		return apperror.ErrUnauthorized
	}

	if req.CurrentPassword == req.NewPassword {
		return errors.Join(apperror.ErrBadRequest, errors.New("password baru harus berbeda dari password lama"))
	}

	hash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hash); err != nil {
		return err
	}

	s.recordAudit(ctx, userID, "auth.password_change", "user", userID)
	return nil
}

// LogoutAll menaikkan session_version sehingga semua token lama ditolak.
func (s *authService) LogoutAll(ctx context.Context, userID string) error {
	if err := s.userRepo.RevokeSessions(ctx, userID); err != nil {
		return err
	}
	s.recordAudit(ctx, userID, "auth.logout_all", "user", userID)
	return nil
}
