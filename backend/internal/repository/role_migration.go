package repository

import (
	"context"
	"fmt"

	"ila/backend/internal/authz"
	"ila/backend/internal/model"
)

// MigrateLegacySupport memindahkan akun ber-role kasar "support" ke model RBAC:
// role kasar menjadi "user" dan role_id diisi role bawaan support.
//
// Aman dijalankan berulang: akun yang sudah dipindahkan tidak disentuh lagi.
func (r *roleRepository) MigrateLegacySupport(ctx context.Context) (int64, error) {
	role, err := r.FindRoleByCode(ctx, authz.RoleSupport)
	if err != nil {
		return 0, fmt.Errorf("membaca role support: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("role = ?", string("support")).
		Updates(map[string]any{"role": "user", "role_id": role.ID})
	if result.Error != nil {
		return 0, fmt.Errorf("memindahkan akun support lama: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// SyncAdminRoleID mengisi role_id untuk akun admin yang belum punya role RBAC.
// Tanpa ini, akun admin lama akan ditolak di pintu admin.
//
// defaultCode dipakai agar pemanggil menentukan role awal (biasanya superadmin
// untuk pemilik platform, atau admin untuk staf).
func (r *roleRepository) SyncAdminRoleID(ctx context.Context, defaultCode string) (int64, error) {
	role, err := r.FindRoleByCode(ctx, defaultCode)
	if err != nil {
		return 0, fmt.Errorf("membaca role %s: %w", defaultCode, err)
	}

	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("role = ? AND (role_id IS NULL OR role_id = ?)", "admin", "").
		UpdateColumn("role_id", role.ID)
	if result.Error != nil {
		return 0, fmt.Errorf("mengisi role admin bawaan: %w", result.Error)
	}
	return result.RowsAffected, nil
}
