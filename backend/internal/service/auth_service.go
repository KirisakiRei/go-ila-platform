package service

import (
	"context"
	"strings"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/config"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/utils"
)

// AuthService menangani pendaftaran dan masuk akun.
type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.UserResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error)
	Profile(ctx context.Context, userID string) (*dto.UserResponse, error)
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

	token, expiresAt, err := s.tokenIssuer.Issue(user.ID, user.Role)
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
