package service

import (
	"context"
	"errors"
	"strings"
	"time"

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
}

type authService struct {
	userRepo    UserRepository
	auditRepo   AuditRepository
	tokenIssuer TokenIssuer
	now         func() time.Time
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

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error) {
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

	return toUserResponse(user), nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
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

	token, expiresAt, err := s.tokenIssuer.IssueWithClaims(utils.Claims{
		UserID:  user.ID,
		Role:    user.Role,
		Scope:   utils.ScopeUser,
		Session: user.SessionVersion,
	})
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID, s.now()); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, user.ID, "auth.login", "user", user.ID)

	return &dto.LoginResponse{
		Token:     token,
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
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Tier:      user.Tier,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
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
