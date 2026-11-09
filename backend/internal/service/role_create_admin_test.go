package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

/*
Pembuatan akun pengelola dari panel RBAC.

Yang dijaga: akun pengelola cukup dibuat dengan username, kata sandi, dan role
— tanpa verifikasi email, trial, atau onboarding. Kata sandi wajib tersimpan
sebagai hash, dan pemberian role tetap tunduk pada larangan eskalasi
kewenangan.
*/

// auditAdminCreate adalah event audit pembuatan akun pengelola.
const auditAdminCreate = "admin.create"

// createAdminUsers mencatat user yang dibuat agar bisa diperiksa.
type createAdminUsers struct {
	fakeRoleUsers
	created *model.User
	taken   []string
}

func (c *createAdminUsers) Create(_ context.Context, user *model.User) error {
	c.created = user
	return nil
}

func (c *createAdminUsers) FindByUsername(_ context.Context, username string) (*model.User, error) {
	for _, name := range c.taken {
		if name == username {
			return &model.User{Username: username}, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (c *createAdminUsers) FindByEmail(_ context.Context, email string) (*model.User, error) {
	for _, name := range c.taken {
		if name == email {
			return &model.User{Email: email}, nil
		}
	}
	return nil, apperror.ErrNotFound
}

type createAdminFixture struct {
	svc   RoleService
	store *fakeRoleStore
	users *createAdminUsers
	audit *recordingAudit
}

func newCreateAdminFixture() *createAdminFixture {
	store := newFakeRoleStore()
	store.addRole(superRoleID, authz.RoleSuperadmin, true, authz.AllKeys()...)
	store.addRole(adminRoleID, authz.RoleAdmin, true, authz.UserView, authz.RoleEdit, authz.RoleView)
	// Role dengan kewenangan lebih luas daripada yang dimiliki pelaku.
	store.addRole("role-luas", "luas", false, authz.UserDelete, authz.UserView)
	// Role UUID harus bisa dicari lewat FindRoleByID; gunakan bentuk yang sah.
	store.addRole("11111111-1111-1111-1111-111111111111", "gudang", false, authz.UserView)

	store.userPerms[superUserID] = authz.AllKeys()
	store.userPerms[plainAdminID] = []string{authz.UserView, authz.RoleEdit, authz.RoleView}

	// RoleCodeOfUser membaca peta userRoles, jadi role pelaku wajib dicatat di
	// sini juga; tanpa itu pengujian superadmin salah menilai dirinya sendiri.
	superID := superRoleID
	plainID := customRoleID
	store.userRoles[superUserID] = &superID
	store.userRoles[plainAdminID] = &plainID

	users := &createAdminUsers{}
	users.users = map[string]*model.User{
		superUserID:  {Base: model.Base{ID: superUserID}, Role: "admin"},
		plainAdminID: {Base: model.Base{ID: plainAdminID}, Role: "admin"},
	}

	audit := &recordingAudit{}
	return &createAdminFixture{
		svc:   NewRoleService(store, users, audit, &fakePermCache{}),
		store: store,
		users: users,
		audit: audit,
	}
}

// Akun pengelola dibuat hanya dengan username, kata sandi, dan role.
func TestCreateAdminCukupUsernameDanPassword(t *testing.T) {
	f := newCreateAdminFixture()

	hasil, err := f.svc.CreateAdmin(context.Background(), superUserID, dto.CreateAdminRequest{
		Username: "Budi.Admin",
		Password: "RahasiaPanjang1",
		RoleID:   "11111111-1111-1111-1111-111111111111",
	})
	if err != nil {
		t.Fatalf("CreateAdmin error: %v", err)
	}

	user := f.users.created
	if user == nil {
		t.Fatal("akun tidak dibuat")
	}

	// Username dibersihkan: huruf kecil, tanpa spasi.
	if user.Username != "budi.admin" {
		t.Errorf("username = %q, seingin \"budi.admin\"", user.Username)
	}
	// Nama kosong memakai username.
	if user.Name != "budi.admin" {
		t.Errorf("name = %q, seharusnya mengikuti username", user.Name)
	}
	// Email disusun internal, tidak menunggu verifikasi.
	if user.Email != "budi.admin@admin.invalid" {
		t.Errorf("email = %q, seharusnya alamat internal", user.Email)
	}
	// Akun pengelola: aktif, ber-role admin, dan tidak ditahan alur pelanggan.
	if user.Role != "admin" {
		t.Errorf("role kasar = %q, seharusnya admin", user.Role)
	}
	if user.RoleID == nil || *user.RoleID != "11111111-1111-1111-1111-111111111111" {
		t.Error("role RBAC tidak terpasang saat pembuatan")
	}
	if user.Status != model.UserStatusActive {
		t.Errorf("status = %q, seharusnya aktif", user.Status)
	}
	if !user.OnboardingDone {
		t.Error("akun pengelola tidak boleh ditahan onboarding pelanggan")
	}
	if user.EmailVerifiedAt == nil {
		t.Error("akun pengelola tidak menunggu verifikasi email; email_verified_at harus terisi")
	}

	// Kata sandi tidak pernah disimpan apa adanya dan harus bisa diverifikasi.
	if user.PasswordHash == "" || user.PasswordHash == "RahasiaPanjang1" {
		t.Fatal("kata sandi tidak di-hash")
	}
	if !strings.HasPrefix(user.PasswordHash, "$argon2id$") {
		t.Errorf("hash bukan argon2id: %q", user.PasswordHash)
	}
	if !utils.ComparePassword(user.PasswordHash, "RahasiaPanjang1") {
		t.Error("hash tidak cocok dengan kata sandi yang diberikan")
	}

	// Ringkasan balasan memuat role agar panel bisa menampilkan langsung.
	if hasil == nil || hasil.RoleCode != "gudang" {
		t.Errorf("role_code balasan = %v, seharusnya gudang", hasil)
	}

	// Audit tercatat tanpa kata sandi.
	if len(f.audit.entries) != 1 {
		t.Fatalf("entri audit = %d, seharusnya 1", len(f.audit.entries))
	}
	entry := f.audit.entries[0]
	if entry.Event != string(auditAdminCreate) {
		t.Errorf("event audit = %q, seharusnya %q", entry.Event, auditAdminCreate)
	}
	if strings.Contains(entry.Metadata, "RahasiaPanjang1") {
		t.Error("kata sandi bocor ke catatan audit")
	}
}

// Username yang sudah dipakai ditolak dengan pesan yang jelas.
func TestCreateAdminUsernameDipakai(t *testing.T) {
	f := newCreateAdminFixture()
	f.users.taken = []string{"budi"}

	_, err := f.svc.CreateAdmin(context.Background(), superUserID, dto.CreateAdminRequest{
		Username: "budi",
		Password: "RahasiaPanjang1",
		RoleID:   "11111111-1111-1111-1111-111111111111",
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
	if f.users.created != nil {
		t.Error("akun tetap dibuat meski username sudah dipakai")
	}
}

// Username dengan karakter yang membingungkan ditolak.
func TestCreateAdminUsernameTidakSah(t *testing.T) {
	f := newCreateAdminFixture()

	for _, nama := range []string{"ab", "budi admin", "budi@mail", "BUDI!!"} {
		_, err := f.svc.CreateAdmin(context.Background(), superUserID, dto.CreateAdminRequest{
			Username: nama,
			Password: "RahasiaPanjang1",
			RoleID:   "11111111-1111-1111-1111-111111111111",
		})
		if !errors.Is(err, apperror.ErrBadRequest) {
			t.Errorf("username %q: error = %v; ingin BAD_REQUEST", nama, err)
		}
	}
	if f.users.created != nil {
		t.Error("akun dibuat meski username tidak sah")
	}
}

// Role dengan kewenangan di luar milik pelaku tidak boleh diberikan.
func TestCreateAdminTidakBolehMelampauiKewenangan(t *testing.T) {
	f := newCreateAdminFixture()

	_, err := f.svc.CreateAdmin(context.Background(), plainAdminID, dto.CreateAdminRequest{
		Username: "budi",
		Password: "RahasiaPanjang1",
		RoleID:   "role-luas",
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST (eskalasi kewenangan)", err)
	}
	if f.users.created != nil {
		t.Error("akun dengan role melebihi kewenangan pelaku tetap dibuat")
	}
}

// Hanya superadmin yang boleh membuat akun ber-role superadmin.
func TestCreateAdminSuperadminHanyaOlehSuperadmin(t *testing.T) {
	f := newCreateAdminFixture()

	_, err := f.svc.CreateAdmin(context.Background(), plainAdminID, dto.CreateAdminRequest{
		Username: "budi",
		Password: "RahasiaPanjang1",
		RoleID:   superRoleID,
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}

	// Superadmin boleh.
	if _, err := f.svc.CreateAdmin(context.Background(), superUserID, dto.CreateAdminRequest{
		Username: "budi",
		Password: "RahasiaPanjang1",
		RoleID:   superRoleID,
	}); err != nil {
		t.Fatalf("superadmin seharusnya boleh membuat akun superadmin: %v", err)
	}
}
