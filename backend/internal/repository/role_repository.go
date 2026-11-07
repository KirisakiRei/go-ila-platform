package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/authz"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// RoleRepository menyimpan role, permission, dan penugasannya.
type RoleRepository interface {
	// Katalog
	SyncCatalog(ctx context.Context, permissions []authz.Permission) error
	SeedDefaultRoles(ctx context.Context, roles []authz.DefaultRole) error
	ListPermissions(ctx context.Context) ([]model.Permission, error)

	// Role
	ListRoles(ctx context.Context) ([]model.Role, error)
	FindRoleByID(ctx context.Context, id string) (*model.Role, error)
	FindRoleByCode(ctx context.Context, code string) (*model.Role, error)
	CreateRole(ctx context.Context, role *model.Role, permissionKeys []string) error
	UpdateRolePermissions(ctx context.Context, roleID string, permissionKeys []string) error
	DeleteRole(ctx context.Context, roleID string) error
	PermissionsOfRole(ctx context.Context, roleID string) ([]string, error)

	// Penugasan
	PermissionsOfUser(ctx context.Context, userID string) ([]string, error)
	RoleCodeOfUser(ctx context.Context, userID string) (string, error)
	AssignRole(ctx context.Context, userID string, roleID *string) error
	CountUsersWithRole(ctx context.Context, roleID string) (int64, error)
	CountActiveAdmins(ctx context.Context) (int64, error)

	// Migrasi data sekali jalan.
	MigrateLegacySupport(ctx context.Context) (int64, error)
	SyncAdminRoleID(ctx context.Context, defaultCode string) (int64, error)
}

type roleRepository struct {
	db *gorm.DB
}

// NewRoleRepository membuat repository RBAC di atas koneksi GORM.
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

// SyncCatalog menyelaraskan tabel permission dengan katalog kode. Katalog
// adalah sumber kebenaran, jadi permission baru masuk dan label ikut diperbarui.
func (r *roleRepository) SyncCatalog(ctx context.Context, permissions []authz.Permission) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, p := range permissions {
			row := model.Permission{
				Key:       p.Key,
				Group:     p.Group,
				Label:     p.Label,
				RiskLabel: p.Risk,
			}
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "key"}},
				DoUpdates: clause.AssignmentColumns([]string{"group", "label", "risk_label"}),
			}).Create(&row).Error
			if err != nil {
				return fmt.Errorf("menyelaraskan permission %s: %w", p.Key, err)
			}
		}
		return nil
	})
}

// SeedDefaultRoles membuat role bawaan bila belum ada. Role yang sudah ada
// tidak diubah agar kustomisasi admin tidak tertimpa setiap start.
func (r *roleRepository) SeedDefaultRoles(ctx context.Context, roles []authz.DefaultRole) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, def := range roles {
			var existing model.Role
			err := tx.Where("code = ?", def.Code).Take(&existing).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("membaca role %s: %w", def.Code, err)
			}

			role := model.Role{
				Code:        def.Code,
				Name:        def.Name,
				Description: def.Description,
				IsSystem:    true,
			}
			if err := tx.Create(&role).Error; err != nil {
				return fmt.Errorf("membuat role %s: %w", def.Code, err)
			}
			if err := replaceRolePermissions(tx, role.ID, def.Permissions); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *roleRepository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var out []model.Permission
	if err := r.db.WithContext(ctx).Order("\"group\", key").Find(&out).Error; err != nil {
		return nil, fmt.Errorf("membaca permission: %w", err)
	}
	return out, nil
}

func (r *roleRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	var out []model.Role
	if err := r.db.WithContext(ctx).Order("is_system DESC, code ASC").Find(&out).Error; err != nil {
		return nil, fmt.Errorf("membaca role: %w", err)
	}
	return out, nil
}

func (r *roleRepository) FindRoleByID(ctx context.Context, id string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("id = ?", id).Take(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca role: %w", err)
	}
	return &role, nil
}

func (r *roleRepository) FindRoleByCode(ctx context.Context, code string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("code = ?", code).Take(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca role: %w", err)
	}
	return &role, nil
}

func (r *roleRepository) CreateRole(ctx context.Context, role *model.Role, permissionKeys []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(role).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return apperror.ErrAlreadyExists
			}
			return fmt.Errorf("membuat role: %w", err)
		}
		return replaceRolePermissions(tx, role.ID, permissionKeys)
	})
}

func (r *roleRepository) UpdateRolePermissions(ctx context.Context, roleID string, permissionKeys []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return replaceRolePermissions(tx, roleID, permissionKeys)
	})
}

func (r *roleRepository) DeleteRole(ctx context.Context, roleID string) error {
	result := r.db.WithContext(ctx).Where("id = ?", roleID).Delete(&model.Role{})
	if result.Error != nil {
		return fmt.Errorf("menghapus role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *roleRepository) PermissionsOfRole(ctx context.Context, roleID string) ([]string, error) {
	var keys []string
	err := r.db.WithContext(ctx).
		Model(&model.RolePermission{}).
		Where("role_id = ?", roleID).
		Pluck("permission_id", &keys).Error
	if err != nil {
		return nil, fmt.Errorf("membaca permission role: %w", err)
	}
	return keys, nil
}

func (r *roleRepository) PermissionsOfUser(ctx context.Context, userID string) ([]string, error) {
	var keys []string
	err := r.db.WithContext(ctx).
		Table("role_permission rp").
		Joins(`JOIN "user" u ON u.role_id = rp.role_id`).
		Where("u.id = ?", userID).
		Pluck("rp.permission_id", &keys).Error
	if err != nil {
		return nil, fmt.Errorf("membaca permission user: %w", err)
	}
	return keys, nil
}

func (r *roleRepository) RoleCodeOfUser(ctx context.Context, userID string) (string, error) {
	var code string
	err := r.db.WithContext(ctx).
		Table(`"user" u`).
		Joins("LEFT JOIN role r ON r.id = u.role_id").
		Where("u.id = ?", userID).
		Select("COALESCE(r.code, '')").
		Scan(&code).Error
	if err != nil {
		return "", fmt.Errorf("membaca role user: %w", err)
	}
	return code, nil
}

func (r *roleRepository) AssignRole(ctx context.Context, userID string, roleID *string) error {
	result := r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		UpdateColumn("role_id", roleID)
	if result.Error != nil {
		return fmt.Errorf("menugaskan role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *roleRepository) CountUsersWithRole(ctx context.Context, roleID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("role_id = ?", roleID).Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung pengguna role: %w", err)
	}
	return total, nil
}

// CountActiveAdmins menghitung akun ber-role kasar admin yang masih aktif.
// Dipakai mencegah penghapusan admin terakhir.
func (r *roleRepository) CountActiveAdmins(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("role = ? AND status = ?", "admin", "active").
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung admin aktif: %w", err)
	}
	return total, nil
}

func replaceRolePermissions(tx *gorm.DB, roleID string, keys []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
		return fmt.Errorf("membersihkan permission role: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}

	rows := make([]model.RolePermission, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, model.RolePermission{RoleID: roleID, PermissionID: key})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return fmt.Errorf("menyimpan permission role: %w", err)
	}
	return nil
}
