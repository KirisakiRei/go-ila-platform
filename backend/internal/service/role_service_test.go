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
)

// fakeRoleStore menyimpan role dan permission di memori untuk pengujian.
type fakeRoleStore struct {
	roles      map[string]model.Role
	rolePerms  map[string][]string
	userPerms  map[string][]string
	userRoles  map[string]*string
	created    []model.Role
	deleted    []string
	assignedTo map[string]*string
}

func newFakeRoleStore() *fakeRoleStore {
	return &fakeRoleStore{
		roles:      make(map[string]model.Role),
		rolePerms:  make(map[string][]string),
		userPerms:  make(map[string][]string),
		userRoles:  make(map[string]*string),
		assignedTo: make(map[string]*string),
	}
}

func (f *fakeRoleStore) addRole(id, code string, system bool, perms ...string) {
	f.roles[id] = model.Role{
		ID:        id,
		Code:      code,
		Name:      code,
		IsSystem:  system,
		CreatedAt: time.Now().UTC(),
	}
	f.rolePerms[id] = perms
}

func (f *fakeRoleStore) ListRoles(context.Context) ([]model.Role, error) {
	out := make([]model.Role, 0, len(f.roles))
	for _, role := range f.roles {
		out = append(out, role)
	}
	return out, nil
}

func (f *fakeRoleStore) FindRoleByID(_ context.Context, id string) (*model.Role, error) {
	role, ok := f.roles[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return &role, nil
}

func (f *fakeRoleStore) FindRoleByCode(_ context.Context, code string) (*model.Role, error) {
	for _, role := range f.roles {
		if role.Code == code {
			return &role, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakeRoleStore) ListPermissions(context.Context) ([]model.Permission, error) {
	out := make([]model.Permission, 0, len(authz.Catalog))
	for _, p := range authz.Catalog {
		out = append(out, model.Permission{Key: p.Key, Group: p.Group, Label: p.Label, RiskLabel: p.Risk})
	}
	return out, nil
}

func (f *fakeRoleStore) CreateRole(_ context.Context, role *model.Role, keys []string) error {
	role.ID = "role-" + role.Code
	f.roles[role.ID] = *role
	f.rolePerms[role.ID] = keys
	f.created = append(f.created, *role)
	return nil
}

func (f *fakeRoleStore) UpdateRolePermissions(_ context.Context, roleID string, keys []string) error {
	f.rolePerms[roleID] = keys
	return nil
}

func (f *fakeRoleStore) DeleteRole(_ context.Context, roleID string) error {
	delete(f.roles, roleID)
	f.deleted = append(f.deleted, roleID)
	return nil
}

func (f *fakeRoleStore) PermissionsOfRole(_ context.Context, roleID string) ([]string, error) {
	return f.rolePerms[roleID], nil
}

func (f *fakeRoleStore) PermissionsOfUser(_ context.Context, userID string) ([]string, error) {
	return f.userPerms[userID], nil
}

func (f *fakeRoleStore) RoleCodeOfUser(_ context.Context, userID string) (string, error) {
	roleID := f.userRoles[userID]
	if roleID == nil {
		return "", nil
	}
	if role, ok := f.roles[*roleID]; ok {
		return role.Code, nil
	}
	return "", nil
}

func (f *fakeRoleStore) AssignRole(_ context.Context, userID string, roleID *string) error {
	f.assignedTo[userID] = roleID
	f.userRoles[userID] = roleID
	return nil
}

func (f *fakeRoleStore) CountUsersWithRole(_ context.Context, roleID string) (int64, error) {
	var total int64
	for _, id := range f.userRoles {
		if id != nil && *id == roleID {
			total++
		}
	}
	return total, nil
}

func (f *fakeRoleStore) CountActiveAdmins(context.Context) (int64, error) {
	return 1, nil
}

// fakeRoleUsers memenuhi kebutuhan pembacaan akun pada layanan role.
type fakeRoleUsers struct {
	users map[string]*model.User
}

func (f *fakeRoleUsers) Create(context.Context, *model.User) error { return nil }

func (f *fakeRoleUsers) FindByID(_ context.Context, id string) (*model.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return user, nil
}

func (f *fakeRoleUsers) FindByEmail(_ context.Context, email string) (*model.User, error) {
	for _, user := range f.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakeRoleUsers) FindByUsername(context.Context, string) (*model.User, error) {
	return nil, apperror.ErrNotFound
}

func (f *fakeRoleUsers) FindByIdentifier(_ context.Context, identifier string) (*model.User, error) {
	for _, user := range f.users {
		if user.Email == identifier || (user.Username != "" && user.Username == identifier) {
			return user, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakeRoleUsers) FindByTelegramChatID(context.Context, string) (*model.User, error) {
	return nil, apperror.ErrNotFound
}

func (f *fakeRoleUsers) UpdateLastLogin(context.Context, string, time.Time) error { return nil }
func (f *fakeRoleUsers) Update(context.Context, *model.User) error                { return nil }

func (f *fakeRoleUsers) RevokeSessions(context.Context, string) error { return nil }

func (f *fakeRoleUsers) UpdatePassword(context.Context, string, string) error { return nil }

// fakePermCache mencatat permintaan pembuangan cache.
type fakePermCache struct {
	invalidated []string
	all         int
}

func (f *fakePermCache) Invalidate(userID string) { f.invalidated = append(f.invalidated, userID) }
func (f *fakePermCache) InvalidateAll()           { f.all++ }

// roleFixture menyiapkan layanan role dengan satu superadmin, satu admin, dan
// satu akun admin tanpa role RBAC.
type roleFixture struct {
	svc   RoleService
	store *fakeRoleStore
	cache *fakePermCache
}

const (
	superRoleID   = "role-super"
	adminRoleID   = "role-admin"
	customRoleID  = "role-custom"
	superUserID   = "u-super"
	plainAdminID  = "u-admin"
	secondAdminID = "u-admin-2"
)

func newRoleFixture() *roleFixture {
	store := newFakeRoleStore()
	store.addRole(superRoleID, authz.RoleSuperadmin, true, authz.AllKeys()...)
	store.addRole(adminRoleID, authz.RoleAdmin, true, authz.UserView, authz.RoleEdit, authz.RoleView)
	store.addRole(customRoleID, "gudang", false, authz.UserView)

	store.userPerms[superUserID] = authz.AllKeys()
	store.userPerms[plainAdminID] = []string{authz.UserView, authz.RoleEdit, authz.RoleView}
	store.userPerms[secondAdminID] = []string{authz.UserView}

	superID := superRoleID
	plainID := customRoleID
	store.userRoles[superUserID] = &superID
	store.userRoles[plainAdminID] = &plainID

	users := &fakeRoleUsers{users: map[string]*model.User{
		plainAdminID:  {Base: model.Base{ID: plainAdminID}, Role: "admin", RoleID: strPtr(customRoleID)},
		secondAdminID: {Base: model.Base{ID: secondAdminID}, Role: "admin"},
		superUserID:   {Base: model.Base{ID: superUserID}, Role: "admin", RoleID: strPtr(superRoleID)},
		"u-user":      {Base: model.Base{ID: "u-user"}, Role: "user"},
	}}

	cache := &fakePermCache{}
	return &roleFixture{
		svc:   NewRoleService(store, users, nil, cache),
		store: store,
		cache: cache,
	}
}

// Larangan #1: admin tidak boleh mengubah role miliknya sendiri.
func TestAssignRoleDiriSendiriDitolak(t *testing.T) {
	f := newRoleFixture()

	err := f.svc.AssignRole(context.Background(), plainAdminID, plainAdminID, dto.AssignRoleRequest{RoleID: strPtr(superRoleID)})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Larangan #2: role superadmin tidak bisa diubah atau dihapus siapa pun.
func TestUbahRoleSuperadminDitolak(t *testing.T) {
	f := newRoleFixture()
	ctx := context.Background()

	if _, err := f.svc.UpdateRole(ctx, superUserID, superRoleID, dto.UpdateRoleRequest{
		Permissions: []string{authz.UserView},
	}); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("UpdateRole error = %v; ingin BAD_REQUEST", err)
	}

	if err := f.svc.DeleteRole(ctx, superUserID, superRoleID); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("DeleteRole error = %v; ingin BAD_REQUEST", err)
	}
}

// Role bawaan lain juga tidak bisa diubah atau dihapus.
func TestUbahRoleSistemDitolak(t *testing.T) {
	f := newRoleFixture()
	ctx := context.Background()

	if _, err := f.svc.UpdateRole(ctx, superUserID, adminRoleID, dto.UpdateRoleRequest{
		Permissions: []string{authz.UserView},
	}); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("UpdateRole error = %v; ingin BAD_REQUEST", err)
	}
}

// Larangan #4: tidak boleh memberi permission yang tidak dimiliki sendiri.
func TestBeriPermissionYangTidakDimilikiDitolak(t *testing.T) {
	f := newRoleFixture()

	// plainAdminID hanya punya user.view & role.*, bukan user.ban.
	_, err := f.svc.CreateRole(context.Background(), plainAdminID, dto.CreateRoleRequest{
		Code:        "penyusup",
		Name:        "Penyusup",
		Permissions: []string{authz.UserBan},
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
	if len(f.store.created) != 0 {
		t.Fatalf("role seharusnya tidak dibuat, tetapi ada %d", len(f.store.created))
	}
}

// Superadmin boleh memberikan seluruh katalog.
func TestSuperadminBisaMembuatRole(t *testing.T) {
	f := newRoleFixture()

	role, err := f.svc.CreateRole(context.Background(), superUserID, dto.CreateRoleRequest{
		Code:        "gudang",
		Name:        "Gudang",
		Permissions: []string{authz.UserView, authz.BillingView},
	})
	if err != nil {
		t.Fatalf("CreateRole error: %v", err)
	}
	if role.Code != "gudang" || len(role.Permissions) != 2 {
		t.Fatalf("role = %+v; isi tidak sesuai", role)
	}
	if f.cache.all == 0 {
		t.Fatal("cache permission seharusnya dibuang seluruhnya")
	}
}

// Larangan #5: role yang masih dipakai tidak boleh dihapus.
func TestHapusRoleYangDipakaiDitolak(t *testing.T) {
	f := newRoleFixture()
	// plainAdminID sudah memakai customRoleID pada fikstur.
	err := f.svc.DeleteRole(context.Background(), superUserID, customRoleID)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Larangan #3: admin aktif terakhir tidak boleh kehilangan role.
func TestCabutRoleAdminTerakhirDitolak(t *testing.T) {
	f := newRoleFixture()

	// plainAdminID satu-satunya admin aktif (CountActiveAdmins = 1); mencabut
	// role-nya harus ditolak agar platform tidak terkunci.
	err := f.svc.AssignRole(context.Background(), superUserID, plainAdminID, dto.AssignRoleRequest{RoleID: nil})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Hanya superadmin yang boleh memberikan role superadmin.
func TestMemberiRoleSuperadminHanyaSuperadmin(t *testing.T) {
	f := newRoleFixture()

	err := f.svc.AssignRole(context.Background(), plainAdminID, secondAdminID, dto.AssignRoleRequest{RoleID: strPtr(superRoleID)})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Permission asing ditolak sebelum menyentuh penyimpanan.
func TestPermissionAsingDitolak(t *testing.T) {
	f := newRoleFixture()

	_, err := f.svc.CreateRole(context.Background(), superUserID, dto.CreateRoleRequest{
		Code: "aneh", Name: "Aneh", Permissions: []string{"tidak.ada"},
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Role tanpa permission ditolak: role kosong tidak ada gunanya.
func TestRoleTanpaPermissionDitolak(t *testing.T) {
	f := newRoleFixture()

	_, err := f.svc.CreateRole(context.Background(), superUserID, dto.CreateRoleRequest{
		Code: "kosong", Name: "Kosong", Permissions: []string{"  "},
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Akun yang bukan admin tidak bisa diberi role RBAC.
func TestAssignKeAkunBukanAdminDitolak(t *testing.T) {
	f := newRoleFixture()

	err := f.svc.AssignRole(context.Background(), superUserID, "u-user", dto.AssignRoleRequest{RoleID: strPtr(adminRoleID)})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Perubahan permission dicatat per kunci, bukan satu baris gabungan.
func TestUpdateRoleMencatatSelisihPermission(t *testing.T) {
	f := newRoleFixture()
	ctx := context.Background()
	store := newFakeRoleStore()
	store.addRole(customRoleID, "gudang", false, authz.UserView)
	store.userPerms[superUserID] = authz.AllKeys()
	store.userRoles[superUserID] = strPtr(superRoleID)

	audit := &recordingAudit{}
	users := &fakeRoleUsers{users: map[string]*model.User{
		superUserID: {Base: model.Base{ID: superUserID}, Role: "admin"},
	}}
	svc := NewRoleService(store, users, audit, &fakePermCache{})

	if _, err := svc.UpdateRole(ctx, superUserID, customRoleID, dto.UpdateRoleRequest{
		Permissions: []string{authz.UserView, authz.BillingView},
	}); err != nil {
		t.Fatalf("UpdateRole error: %v", err)
	}

	var granted int
	for _, entry := range audit.entries {
		if entry.Event == "role.update.grant" && entry.Metadata == authz.BillingView {
			granted++
		}
	}
	if granted != 1 {
		t.Fatalf("pemberian billing.view tercatat %d kali; ingin 1", granted)
	}
	_ = f
}

// Katalog permission dikelompokkan dan menandai yang tidak bisa diberikan.
func TestListPermissionsMenandaiGrantable(t *testing.T) {
	f := newRoleFixture()

	groups, err := f.svc.ListPermissions(context.Background(), plainAdminID)
	if err != nil {
		t.Fatalf("ListPermissions error: %v", err)
	}
	if len(groups) == 0 {
		t.Fatal("katalog kosong")
	}

	found := false
	for _, group := range groups {
		for _, p := range group.Permissions {
			if p.Key == authz.UserBan {
				found = true
				if p.Grantable {
					t.Fatal("user.ban seharusnya tidak bisa diberikan oleh admin ini")
				}
			}
		}
	}
	if !found {
		t.Fatal("user.ban tidak ada di katalog")
	}
}

// strPtr mengembalikan alamat salinan string.
func strPtr(v string) *string { return &v }

// recordingAudit menyimpan entri audit untuk pemeriksaan.
type recordingAudit struct {
	entries []model.AuditLog
}

func (r *recordingAudit) Record(_ context.Context, entry *model.AuditLog) error {
	r.entries = append(r.entries, *entry)
	return nil
}
