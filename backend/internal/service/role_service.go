package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"ila/backend/internal/authz"
	"ila/backend/internal/config"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// RoleService mengelola role dan penugasannya dengan lima larangan
// pengamanan di plans/04 §4.1.
type RoleService interface {
	ListRoles(ctx context.Context, actorID string) ([]dto.RoleResponse, error)
	GetRole(ctx context.Context, actorID, roleID string) (*dto.RoleDetailResponse, error)
	CreateRole(ctx context.Context, actorID string, req dto.CreateRoleRequest) (*dto.RoleResponse, error)
	UpdateRole(ctx context.Context, actorID, roleID string, req dto.UpdateRoleRequest) (*dto.RoleResponse, error)
	DeleteRole(ctx context.Context, actorID, roleID string) error
	AssignRole(ctx context.Context, actorID, userID string, req dto.AssignRoleRequest) error
	ListPermissions(ctx context.Context, actorID string) ([]dto.PermissionGroup, error)
}

// RoleStore adalah kebutuhan service atas penyimpanan RBAC.
type RoleStore interface {
	ListRoles(ctx context.Context) ([]model.Role, error)
	FindRoleByID(ctx context.Context, id string) (*model.Role, error)
	FindRoleByCode(ctx context.Context, code string) (*model.Role, error)
	ListPermissions(ctx context.Context) ([]model.Permission, error)
	CreateRole(ctx context.Context, role *model.Role, permissionKeys []string) error
	UpdateRolePermissions(ctx context.Context, roleID string, permissionKeys []string) error
	DeleteRole(ctx context.Context, roleID string) error
	PermissionsOfRole(ctx context.Context, roleID string) ([]string, error)
	PermissionsOfUser(ctx context.Context, userID string) ([]string, error)
	RoleCodeOfUser(ctx context.Context, userID string) (string, error)
	AssignRole(ctx context.Context, userID string, roleID *string) error
	CountUsersWithRole(ctx context.Context, roleID string) (int64, error)
	CountActiveAdmins(ctx context.Context) (int64, error)
}

// PermissionCache menyegarkan cache setelah kewenangan berubah.
type PermissionCache interface {
	Invalidate(userID string)
	InvalidateAll()
}

type roleService struct {
	store RoleStore
	users UserRepository
	audit AuditRepository
	cache PermissionCache
	now   func() time.Time
}

// NewRoleService merangkai layanan RBAC.
func NewRoleService(store RoleStore, users UserRepository, audit AuditRepository, cache PermissionCache) RoleService {
	return &roleService{
		store: store,
		users: users,
		audit: audit,
		cache: cache,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

// ListRoles menampilkan role beserta jumlah akun pemakainya.
func (s *roleService) ListRoles(ctx context.Context, actorID string) ([]dto.RoleResponse, error) {
	roles, err := s.store.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]dto.RoleResponse, 0, len(roles))
	for i := range roles {
		perms, err := s.store.PermissionsOfRole(ctx, roles[i].ID)
		if err != nil {
			return nil, err
		}
		count, err := s.store.CountUsersWithRole(ctx, roles[i].ID)
		if err != nil {
			return nil, err
		}
		out = append(out, toRoleResponse(roles[i], perms, count))
	}
	return out, nil
}

// GetRole menampilkan satu role beserta permission yang bisa diberikan.
func (s *roleService) GetRole(ctx context.Context, actorID, roleID string) (*dto.RoleDetailResponse, error) {
	role, err := s.store.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}

	perms, err := s.store.PermissionsOfRole(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	count, err := s.store.CountUsersWithRole(ctx, role.ID)
	if err != nil {
		return nil, err
	}

	groups, err := s.ListPermissions(ctx, actorID)
	if err != nil {
		return nil, err
	}

	catalog := make([]dto.PermissionResponse, 0, len(authz.Catalog))
	for _, group := range groups {
		catalog = append(catalog, group.Permissions...)
	}

	return &dto.RoleDetailResponse{
		RoleResponse:         toRoleResponse(*role, perms, count),
		AvailablePermissions: catalog,
	}, nil
}

// ListPermissions menampilkan katalog permission bergrup untuk tampilan admin.
func (s *roleService) ListPermissions(ctx context.Context, actorID string) ([]dto.PermissionGroup, error) {
	rows, err := s.store.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	ownedList, err := s.store.PermissionsOfUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	owned := make(map[string]struct{}, len(ownedList))
	for _, key := range ownedList {
		owned[key] = struct{}{}
	}

	groups := make([]dto.PermissionGroup, 0, 8)
	index := make(map[string]int, 8)
	for _, row := range rows {
		_, canGrant := owned[row.Key]

		pos, ok := index[row.Group]
		if !ok {
			groups = append(groups, dto.PermissionGroup{Group: row.Group})
			pos = len(groups) - 1
			index[row.Group] = pos
		}
		groups[pos].Permissions = append(groups[pos].Permissions, dto.PermissionResponse{
			Key:       row.Key,
			Group:     row.Group,
			Label:     row.Label,
			Risk:      row.RiskLabel,
			Grantable: canGrant,
		})
	}
	return groups, nil
}

// CreateRole membuat role baru. Permission yang diberikan tidak boleh melebihi
// milik pembuatnya (larangan #4).
func (s *roleService) CreateRole(ctx context.Context, actorID string, req dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	code := strings.ToLower(strings.TrimSpace(req.Code))
	if code == "" {
		return nil, validation("kode role wajib diisi")
	}

	keys, err := s.normalizePermissions(req.Permissions)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCanGrant(ctx, actorID, keys); err != nil {
		return nil, err
	}

	now := s.now()
	role := &model.Role{
		Code:        code,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsSystem:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.CreateRole(ctx, role, keys); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, actorID, "role.create", role.ID, code+": "+strings.Join(keys, ","))
	s.invalidateAll()

	response := toRoleResponse(*role, keys, 0)
	return &response, nil
}

// UpdateRole mengganti permission sebuah role.
func (s *roleService) UpdateRole(ctx context.Context, actorID, roleID string, req dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	role, err := s.store.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if err := guardSystemRole(*role, "diubah"); err != nil {
		return nil, err
	}

	keys, err := s.normalizePermissions(req.Permissions)
	if err != nil {
		return nil, err
	}
	// Larangan #1 dan #4: tidak boleh menaikkan kewenangan diri sendiri.
	if err := s.ensureCanGrant(ctx, actorID, keys); err != nil {
		return nil, err
	}

	before, err := s.store.PermissionsOfRole(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	if err := s.store.UpdateRolePermissions(ctx, role.ID, keys); err != nil {
		return nil, err
	}

	// Perubahan permission dicatat per permission (plans/04 §7).
	s.recordPermissionDiff(ctx, actorID, role.ID, before, keys)
	s.invalidateAll()

	count, err := s.store.CountUsersWithRole(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	response := toRoleResponse(*role, keys, count)
	return &response, nil
}

// DeleteRole menghapus role yang tidak dipakai dan bukan bawaan sistem.
func (s *roleService) DeleteRole(ctx context.Context, actorID, roleID string) error {
	role, err := s.store.FindRoleByID(ctx, roleID)
	if err != nil {
		return err
	}
	if err := guardSystemRole(*role, "dihapus"); err != nil {
		return err
	}

	// Larangan #5: role yang masih dipakai tidak boleh dihapus.
	count, err := s.store.CountUsersWithRole(ctx, role.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		return validation("role masih dipakai oleh " + strconv.FormatInt(count, 10) + " akun; pindahkan dulu")
	}

	if err := s.store.DeleteRole(ctx, role.ID); err != nil {
		return err
	}

	s.recordAudit(ctx, actorID, "role.delete", role.ID, role.Code)
	s.invalidateAll()
	return nil
}

// AssignRole menugaskan role RBAC ke sebuah akun.
func (s *roleService) AssignRole(ctx context.Context, actorID, userID string, req dto.AssignRoleRequest) error {
	// Larangan #1: tidak boleh mengubah role sendiri.
	if actorID == userID {
		return validation("tidak dapat mengubah role akun sendiri")
	}

	target, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	// Hanya akun ber-role kasar admin yang boleh dipegangkan role RBAC.
	if target.Role != string(config.RoleAdmin) {
		return validation("akun ini bukan akun admin")
	}

	var role *model.Role
	if req.RoleID != nil && strings.TrimSpace(*req.RoleID) != "" {
		role, err = s.store.FindRoleByID(ctx, strings.TrimSpace(*req.RoleID))
		if err != nil {
			return err
		}

		// Larangan #2: role superadmin hanya boleh dipegang superadmin.
		if role.Code == authz.RoleSuperadmin {
			actorRole, err := s.store.RoleCodeOfUser(ctx, actorID)
			if err != nil {
				return err
			}
			if actorRole != authz.RoleSuperadmin {
				return validation("hanya superadmin dapat memberikan role superadmin")
			}
		}

		perms, err := s.store.PermissionsOfRole(ctx, role.ID)
		if err != nil {
			return err
		}
		// Larangan #4: tidak boleh memberi yang tidak dimiliki.
		if err := s.ensureCanGrant(ctx, actorID, perms); err != nil {
			return err
		}
	}

	// Larangan #3: jangan sampai tidak ada admin tersisa tanpa kewenangan.
	if target.RoleID != nil && *target.RoleID != "" && role == nil {
		active, err := s.store.CountActiveAdmins(ctx)
		if err != nil {
			return err
		}
		if active <= 1 {
			return validation("tidak dapat mencabut role admin aktif terakhir")
		}
	}

	var roleID *string
	roleCode := ""
	if role != nil {
		roleID = &role.ID
		roleCode = role.Code
	}

	if err := s.store.AssignRole(ctx, userID, roleID); err != nil {
		return err
	}

	// Kewenangan berubah seketika: sesi dan cache akun target dibuang.
	if err := s.users.RevokeSessions(ctx, userID); err != nil {
		return err
	}
	cacheInvalidator(s.cache, userID)

	s.recordAudit(ctx, actorID, "role.assign", userID, roleCode)
	return nil
}

// ensureCanGrant memastikan seluruh permission yang diberikan dimiliki pelaku.
func (s *roleService) ensureCanGrant(ctx context.Context, actorID string, wanted []string) error {
	actorPerms, err := s.store.PermissionsOfUser(ctx, actorID)
	if err != nil {
		return err
	}

	owned := make(map[string]struct{}, len(actorPerms))
	for _, key := range actorPerms {
		owned[key] = struct{}{}
	}

	for _, key := range wanted {
		if _, ok := owned[key]; !ok {
			// Pesan tidak menyebut permission mana yang kurang.
			return validation("anda tidak dapat memberikan izin yang tidak anda miliki")
		}
	}
	return nil
}

// normalizePermissions membuang duplikat dan menolak kunci di luar katalog.
func (s *roleService) normalizePermissions(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if !authz.Known(key) {
			return nil, validation("permission tidak dikenal: " + key)
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	if len(out) == 0 {
		return nil, validation("role harus memiliki minimal satu permission")
	}
	return out, nil
}

// recordPermissionDiff mencatat selisih permission per kunci.
func (s *roleService) recordPermissionDiff(ctx context.Context, actorID, roleID string, before, after []string) {
	beforeSet := make(map[string]struct{}, len(before))
	for _, key := range before {
		beforeSet[key] = struct{}{}
	}
	afterSet := make(map[string]struct{}, len(after))
	for _, key := range after {
		afterSet[key] = struct{}{}
	}

	for _, key := range after {
		if _, ok := beforeSet[key]; !ok {
			s.recordAudit(ctx, actorID, "role.update.grant", roleID, key)
		}
	}
	for _, key := range before {
		if _, ok := afterSet[key]; !ok {
			s.recordAudit(ctx, actorID, "role.update.revoke", roleID, key)
		}
	}
}

func (s *roleService) recordAudit(ctx context.Context, actorID, event, entityID, detail string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, &model.AuditLog{
		UserID:    actorID,
		Event:     event,
		Entity:    "role",
		EntityID:  entityID,
		Metadata:  detail,
		CreatedAt: s.now(),
	})
}

func (s *roleService) invalidateAll() {
	if s.cache != nil {
		s.cache.InvalidateAll()
	}
}

// cacheInvalidator membuang cache satu akun bila cache tersedia.
func cacheInvalidator(cache PermissionCache, userID string) {
	if cache != nil {
		cache.Invalidate(userID)
	}
}

// guardSystemRole menolak perubahan pada role superadmin dan role bawaan.
func guardSystemRole(role model.Role, action string) error {
	if role.Code == authz.RoleSuperadmin {
		return validation("role superadmin tidak dapat " + action)
	}
	if role.IsSystem {
		return validation("role bawaan sistem tidak dapat " + action)
	}
	return nil
}

// toRoleResponse memetakan model role ke bentuk yang dikirim ke klien.
func toRoleResponse(role model.Role, perms []string, userCount int64) dto.RoleResponse {
	if perms == nil {
		perms = []string{}
	}
	return dto.RoleResponse{
		ID:          role.ID,
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		Editable:    !role.IsSystem,
		Permissions: perms,
		UserCount:   userCount,
		CreatedAt:   role.CreatedAt.Format(time.RFC3339),
	}
}

// validation membungkus pesan menjadi error BAD_REQUEST.
func validation(message string) error {
	return errors.Join(apperror.ErrBadRequest, errors.New(message))
}
