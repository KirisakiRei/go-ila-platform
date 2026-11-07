package service

import (
	"context"
	"errors"
	"time"

	"ila/backend/internal/authz"
	"ila/backend/internal/config"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

// AdminAuthService adalah pintu masuk pintu admin: sesi admin terpisah dari
// sesi user, dengan scope sendiri (plans/03 §2).
type AdminAuthService interface {
	Login(ctx context.Context, req dto.LoginRequest, device LoginContext) (*dto.AdminLoginResponse, error)
	Profile(ctx context.Context, userID string) (*dto.AdminProfileResponse, error)
}

// AdminPermissionReader membaca permission efektif sebuah akun.
type AdminPermissionReader interface {
	Permissions(ctx context.Context, userID string) ([]string, error)
	RoleCode(ctx context.Context, userID string) (string, error)
}

type adminAuthService struct {
	users    UserRepository
	audit    AuditRepository
	roles    RoleStore
	perms    AdminPermissionReader
	issuer   TokenIssuer
	sessions SessionRecorder
	twofaOn  bool
	now      func() time.Time
}

// AdminAuthDeps mengumpulkan dependensi layanan autentikasi admin.
type AdminAuthDeps struct {
	Users    UserRepository
	Audit    AuditRepository
	Roles    RoleStore
	Perms    AdminPermissionReader
	Issuer   TokenIssuer
	Sessions SessionRecorder
	// RequireTwoFA menandai apakah 2FA wajib. Bila true dan akun belum
	// mengaktifkan 2FA, login ditolak dengan instruksi aktivasi.
	RequireTwoFA bool
}

// NewAdminAuthService merangkai layanan autentikasi admin.
func NewAdminAuthService(deps AdminAuthDeps) AdminAuthService {
	return &adminAuthService{
		users:    deps.Users,
		audit:    deps.Audit,
		roles:    deps.Roles,
		perms:    deps.Perms,
		issuer:   deps.Issuer,
		sessions: deps.Sessions,
		twofaOn:  deps.RequireTwoFA,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// Login memverifikasi password admin lalu menerbitkan token ber-scope admin.
func (s *adminAuthService) Login(ctx context.Context, req dto.LoginRequest, device LoginContext) (*dto.AdminLoginResponse, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, apperror.ErrUnauthorized
		}
		return nil, err
	}

	// Akun non-admin diperlakukan seperti tidak ada: pintu admin tidak
	// membocorkan bahwa email itu terdaftar sebagai pengguna biasa.
	if user.Role != string(config.RoleAdmin) {
		return nil, apperror.ErrUnauthorized
	}
	if user.Status != "active" {
		return nil, apperror.ErrForbidden
	}
	if user.RoleID == nil || *user.RoleID == "" {
		// Akun admin tanpa role RBAC tidak punya kewenangan apa pun.
		return nil, apperror.ErrForbidden
	}

	if !utils.ComparePassword(user.PasswordHash, req.Password) {
		s.recordAudit(ctx, user.ID, "admin.login_failed")
		return nil, apperror.ErrUnauthorized
	}

	// 2FA belum dirancang pada fase RBAC; penanda ini memaksa penolakan
	// sampai alur 2FA tersedia agar tidak ada pintu tanpa faktor kedua.
	if s.twofaOn {
		return nil, errors.Join(apperror.ErrServiceUnavailable, errors.New("2FA admin belum aktif pada versi ini"))
	}

	expiresAt := s.now().Add(s.issuer.TTL())

	sessionID := ""
	if s.sessions != nil {
		id, err := s.sessions.Record(ctx, user.ID, "", utils.ScopeAdmin, device.UserAgent, device.IPHash, expiresAt)
		if err != nil {
			s.recordAudit(ctx, user.ID, "admin.session_record_failed")
		} else {
			sessionID = id
		}
	}

	token, expiresAt, err := s.issuer.IssueWithClaims(utils.Claims{
		UserID:    user.ID,
		Role:      user.Role,
		Scope:     utils.ScopeAdmin,
		Session:   user.SessionVersion,
		SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}

	if err := s.users.UpdateLastLogin(ctx, user.ID, s.now()); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, user.ID, "admin.login")

	perms, err := s.perms.Permissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	roleCode, err := s.perms.RoleCode(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &dto.AdminLoginResponse{
		Token:     token,
		SessionID: sessionID,
		ExpiresAt: expiresAt.Format(time.RFC3339),
		User: dto.AdminIdentity{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
		RoleCode:    roleCode,
		Permissions: perms,
	}, nil
}

// Profile mengembalikan identitas dan kewenangan sesi admin saat ini.
func (s *adminAuthService) Profile(ctx context.Context, userID string) (*dto.AdminProfileResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, apperror.ErrUnauthorized
		}
		return nil, err
	}

	perms, err := s.perms.Permissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	roleCode, err := s.perms.RoleCode(ctx, userID)
	if err != nil {
		return nil, err
	}

	roleName := roleCode
	if roleCode != "" {
		if role, err := s.roles.FindRoleByCode(ctx, roleCode); err == nil {
			roleName = role.Name
		}
	}

	return &dto.AdminProfileResponse{
		User: dto.AdminIdentity{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
		Role: dto.AdminRoleSummary{
			Code: roleCode,
			Name: roleName,
			// Superadmin ditandai agar antarmuka bisa mengunci aksi terkunci.
			IsSuperadmin: roleCode == authz.RoleSuperadmin,
		},
		Permissions: perms,
	}, nil
}

func (s *adminAuthService) recordAudit(ctx context.Context, userID, event string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, &model.AuditLog{
		UserID:    userID,
		Event:     event,
		Entity:    "user",
		EntityID:  userID,
		CreatedAt: s.now(),
	})
}
