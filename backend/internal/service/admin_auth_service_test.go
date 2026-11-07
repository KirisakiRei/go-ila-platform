package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

// adminAuthFixture menyiapkan layanan login admin dengan satu akun admin.
type adminAuthFixture struct {
	svc   AdminAuthService
	users *fakeRoleUsers
	store *fakeRoleStore
}

func newAdminAuthFixture(t *testing.T) *adminAuthFixture {
	t.Helper()

	hash, err := utils.HashPassword("rahasia-panjang-123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	superID := superRoleID
	store := newFakeRoleStore()
	store.addRole(superRoleID, authz.RoleSuperadmin, true, authz.AllKeys()...)

	users := &fakeRoleUsers{users: map[string]*model.User{
		"a1": {
			Base:         model.Base{ID: "a1", CreatedAt: time.Now().UTC()},
			Name:         "Admin Satu",
			Email:        "admin@ila.id",
			PasswordHash: hash,
			Role:         "admin",
			RoleID:       &superID,
			Status:       "active",
		},
		"a2": {
			Base:         model.Base{ID: "a2"},
			Name:         "Admin Tanpa Role",
			Email:        "noro le@ila.id",
			PasswordHash: hash,
			Role:         "admin",
			Status:       "active",
		},
		"u1": {
			Base:         model.Base{ID: "u1"},
			Name:         "Pelanggan",
			Email:        "user@ila.id",
			PasswordHash: hash,
			Role:         "user",
			Status:       "active",
		},
	}}

	svc := NewAdminAuthService(AdminAuthDeps{
		Users:    users,
		Roles:    store,
		Perms:    &stubPermReader{users: users},
		Issuer:   utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour),
		Sessions: nil,
	})

	return &adminAuthFixture{svc: svc, users: users, store: store}
}

// stubPermReader membaca permission dari role yang melekat pada akun.
type stubPermReader struct {
	users *fakeRoleUsers
}

func (s *stubPermReader) Permissions(_ context.Context, userID string) ([]string, error) {
	user, ok := s.users.users[userID]
	if !ok || user.RoleID == nil {
		return nil, nil
	}
	if *user.RoleID == superRoleID {
		return authz.AllKeys(), nil
	}
	return nil, nil
}

func (s *stubPermReader) RoleCode(_ context.Context, userID string) (string, error) {
	user, ok := s.users.users[userID]
	if !ok || user.RoleID == nil {
		return "", nil
	}
	if *user.RoleID == superRoleID {
		return authz.RoleSuperadmin, nil
	}
	return "", nil
}

// Admin sah boleh masuk dan mendapat token ber-scope admin.
func TestAdminLoginBerhasil(t *testing.T) {
	f := newAdminAuthFixture(t)

	result, err := f.svc.Login(context.Background(), dto.LoginRequest{
		Email: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{UserAgent: "uji"})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if result.RoleCode != authz.RoleSuperadmin {
		t.Fatalf("role = %q; ingin %q", result.RoleCode, authz.RoleSuperadmin)
	}
	if len(result.Permissions) != len(authz.AllKeys()) {
		t.Fatalf("permission = %d; ingin %d", len(result.Permissions), len(authz.AllKeys()))
	}

	token := result.Token
	claims, err := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour).Verify(token)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if claims.Scope != utils.ScopeAdmin {
		t.Fatalf("scope = %q; ingin admin", claims.Scope)
	}
}

// Pelanggan biasa tidak boleh masuk pintu admin, dan pesannya tidak membedakan
// antara "bukan admin" dan "salah password".
func TestPelangganTidakBisaMasukPintuAdmin(t *testing.T) {
	f := newAdminAuthFixture(t)

	_, err := f.svc.Login(context.Background(), dto.LoginRequest{
		Email: "user@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatalf("error = %v; ingin UNAUTHORIZED", err)
	}
}

// Password salah juga 401 dengan pesan yang sama.
func TestPasswordSalah401(t *testing.T) {
	f := newAdminAuthFixture(t)

	_, err := f.svc.Login(context.Background(), dto.LoginRequest{
		Email: "admin@ila.id", Password: "salah",
	}, LoginContext{})
	if !errors.Is(err, apperror.ErrUnauthorized) {
		t.Fatalf("error = %v; ingin UNAUTHORIZED", err)
	}
}

// Admin tanpa role RBAC tidak punya kewenangan apa pun: ditolak.
func TestAdminTanpaRoleRbacDitolak(t *testing.T) {
	f := newAdminAuthFixture(t)

	_, err := f.svc.Login(context.Background(), dto.LoginRequest{
		Email: "noro le@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("error = %v; ingin FORBIDDEN", err)
	}
}

// Akun admin yang ditangguhkan tidak boleh masuk.
func TestAdminDitangguhkanDitolak(t *testing.T) {
	f := newAdminAuthFixture(t)
	f.users.users["a1"].Status = "suspended"

	_, err := f.svc.Login(context.Background(), dto.LoginRequest{
		Email: "admin@ila.id", Password: "rahasia-panjang-123",
	}, LoginContext{})
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("error = %v; ingin FORBIDDEN", err)
	}
}

// Profil mengembalikan identitas dan kewenangan sesi admin.
func TestAdminProfile(t *testing.T) {
	f := newAdminAuthFixture(t)

	profile, err := f.svc.Profile(context.Background(), "a1")
	if err != nil {
		t.Fatalf("Profile error: %v", err)
	}
	if !profile.Role.IsSuperadmin {
		t.Fatal("superadmin seharusnya ditandai")
	}
	if profile.User.Email != "admin@ila.id" {
		t.Fatalf("email = %q", profile.User.Email)
	}
}
