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
	// VerifyTwoFactor menuntaskan login setelah kode 2FA diperiksa.
	VerifyTwoFactor(ctx context.Context, req dto.VerifyTwoFactorRequest, device LoginContext) (*dto.AdminLoginResponse, error)
	Profile(ctx context.Context, userID string) (*dto.AdminProfileResponse, error)
}

// AdminPermissionReader membaca permission efektif sebuah akun.
type AdminPermissionReader interface {
	Permissions(ctx context.Context, userID string) ([]string, error)
	RoleCode(ctx context.Context, userID string) (string, error)
}

type adminAuthService struct {
	users      UserRepository
	audit      AuditRepository
	roles      RoleStore
	perms      AdminPermissionReader
	issuer     TokenIssuer
	sessions   SessionRecorder
	gate       TwoFactorGate
	challenges *challengeService
	now        func() time.Time
}

// AdminAuthDeps mengumpulkan dependensi layanan autentikasi admin.
type AdminAuthDeps struct {
	Users    UserRepository
	Audit    AuditRepository
	Roles    RoleStore
	Perms    AdminPermissionReader
	Issuer   TokenIssuer
	Sessions SessionRecorder
	// TwoFactor membuat 2FA opsional sebagai tambahan proteksi. Bila nil,
	// pintu admin bebas 2FA seperti sebelumnya.
	TwoFactor TwoFactorGate
	// Challenges menyimpan keadaan antara kata sandi dan kode 2FA.
	Challenges ChallengeRepository
}

// NewAdminAuthService merangkai layanan autentikasi admin.
func NewAdminAuthService(deps AdminAuthDeps) AdminAuthService {
	var gate TwoFactorGate
	var challenges *challengeService
	if deps.TwoFactor != nil && deps.Challenges != nil {
		gate = deps.TwoFactor
		challenges = newChallengeService(deps.Challenges, gate)
	}

	now := func() time.Time { return time.Now().UTC() }
	if challenges != nil {
		challenges.setClock(now)
	}

	return &adminAuthService{
		users:      deps.Users,
		audit:      deps.Audit,
		roles:      deps.Roles,
		perms:      deps.Perms,
		issuer:     deps.Issuer,
		sessions:   deps.Sessions,
		gate:       gate,
		challenges: challenges,
		now:        now,
	}
}

// Login memverifikasi password admin lalu menerbitkan token ber-scope admin.
func (s *adminAuthService) Login(ctx context.Context, req dto.LoginRequest, device LoginContext) (*dto.AdminLoginResponse, error) {
	user, err := s.users.FindByIdentifier(ctx, normalizeIdentifier(req.Identifier))
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

	// 2FA opsional: bila akun mengaktifkannya, sesi belum terbit sampai kode
	// diverifikasi. Klien hanya menerima challenge ID yang tidak berguna
	// untuk mengakses data (plans/05 §4.1).
	pending, err := requireTwoFactor(ctx, s.gate, s.challenges, user.ID, PurposeAdmin)
	if err != nil {
		return nil, err
	}
	if pending.Step == StepTwoFactor {
		detail := newTwoFactorRequired(pending)
		return &dto.AdminLoginResponse{
			Step:        detail.Step,
			ChallengeID: detail.ChallengeID,
			ExpiresAt:   detail.ExpiresAt,
			ExpiresIn:   detail.ExpiresIn,
			User: dto.AdminIdentity{
				ID:    user.ID,
				Name:  user.Name,
				Email: user.Email,
				Role:  user.Role,
			},
		}, nil
	}

	return s.issueSession(ctx, user, device)
}

// VerifyTwoFactor menuntaskan login admin: kode diperiksa, lalu sesi terbit.
func (s *adminAuthService) VerifyTwoFactor(ctx context.Context, req dto.VerifyTwoFactorRequest, device LoginContext) (*dto.AdminLoginResponse, error) {
	if s.challenges == nil {
		return nil, apperror.ErrUnauthorized
	}

	// Pemilik tantangan dibaca dulu: pemeriksaan kode butuh identitas akun.
	owner, err := s.challenges.Owner(ctx, req.ChallengeID, PurposeAdmin)
	if err != nil {
		return nil, err
	}

	if err := s.challenges.Redeem(ctx, req.ChallengeID, owner, PurposeAdmin, req.Code); err != nil {
		if errors.Is(err, apperror.ErrBadRequest) {
			s.recordAudit(ctx, owner, "admin.login_failed")
		}
		return nil, err
	}

	user, err := s.users.FindByID(ctx, owner)
	if err != nil {
		return nil, err
	}
	// Keadaan akun bisa berubah selama tantangan berjalan.
	if user.Role != string(config.RoleAdmin) || user.Status != "active" || user.RoleID == nil {
		return nil, apperror.ErrForbidden
	}

	return s.issueSession(ctx, user, device)
}

// issueSession menerbitkan token admin beserta identitas dan kewenangannya.
func (s *adminAuthService) issueSession(ctx context.Context, user *model.User, device LoginContext) (*dto.AdminLoginResponse, error) {
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
		Step:      string(StepComplete),
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
