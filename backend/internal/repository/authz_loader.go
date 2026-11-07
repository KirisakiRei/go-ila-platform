package repository

import (
	"context"

	"ila/backend/internal/authz"
)

// AuthzLoader menyambungkan penyimpanan RBAC ke pemuat permission milik
// middleware, sehingga keduanya memakai query yang sama.
type AuthzLoader struct {
	roles RoleRepository
}

// NewAuthzLoader membuat adapter permission loader.
func NewAuthzLoader(roles RoleRepository) *AuthzLoader {
	return &AuthzLoader{roles: roles}
}

// PermissionsOf membaca permission efektif sebuah akun.
func (l *AuthzLoader) PermissionsOf(ctx context.Context, userID string) ([]string, error) {
	return l.roles.PermissionsOfUser(ctx, userID)
}

// RoleCodeOf membaca kode role RBAC sebuah akun.
func (l *AuthzLoader) RoleCodeOf(ctx context.Context, userID string) (string, error) {
	return l.roles.RoleCodeOfUser(ctx, userID)
}

// Memastikan adapter memenuhi kontrak milik paket authz.
var _ authz.Loader = (*AuthzLoader)(nil)
