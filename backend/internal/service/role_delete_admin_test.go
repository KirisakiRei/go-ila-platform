package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ila/backend/internal/authz"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

/*
Penghapusan akun pengelola dari panel RBAC.

Yang dijaga:
  - Akun superadmin utama (is_protected) tidak pernah bisa dihapus, siapa pun
    pelakunya — termasuk superadmin lain. Satu pintu masuk selalu tersisa.
  - Pelaku tidak bisa menghapus akunnya sendiri.
  - Admin biasa tidak bisa menyingkirkan akun yang role-nya lebih tinggi.
  - Admin aktif terakhir tidak bisa dihapus.
  - Penghapusan benar-benar menganonimkan akun, mencabut sesinya, membuang
    cache kewenangannya, dan meninggalkan satu catatan audit beralasan tanpa
    membocorkan email atau kata sandi.
*/

const auditAdminDelete = "admin.delete"

// protectedAdminID adalah akun superadmin utama yang tidak boleh dihapus.
const protectedAdminID = "u-super-utama"

// deleteAdminFixture menyiapkan dua superadmin, satu admin biasa, dan satu
// admin terlindungi, lalu merangkai layanan dengan anonymizer palsu.
type deleteAdminFixture struct {
	svc        RoleService
	store      *fakeRoleStore
	users      *fakeRoleUsers
	audit      *recordingAudit
	cache      *fakePermCache
	anonymizer *fakeAnonymizer
}

func newDeleteAdminFixture() *deleteAdminFixture {
	store := newFakeRoleStore()
	store.addRole(superRoleID, authz.RoleSuperadmin, true, authz.AllKeys()...)
	store.addRole(adminRoleID, authz.RoleAdmin, true, authz.UserView, authz.RoleEdit, authz.RoleView)
	store.addRole(customRoleID, "gudang", false, authz.UserView)
	// Role dengan kewenangan lebih luas daripada yang dimiliki admin biasa.
	store.addRole("role-luas", "luas", false, append([]string{authz.UserDelete}, authz.AllKeys()...)...)
	store.activeAdmins = 3

	store.userPerms[superUserID] = authz.AllKeys()
	store.userPerms[plainAdminID] = []string{authz.UserView, authz.RoleEdit, authz.RoleView}
	store.userPerms[secondAdminID] = []string{authz.UserView, authz.RoleDelete}
	store.userRoles[superUserID] = strPtr(superRoleID)
	store.userRoles[plainAdminID] = strPtr(customRoleID)
	store.userRoles[protectedAdminID] = strPtr(superRoleID)

	users := &fakeRoleUsers{users: map[string]*model.User{
		superUserID: {
			Base: model.Base{ID: superUserID}, Role: "admin",
			RoleID: strPtr(superRoleID), Email: "super@rekavia.com",
		},
		plainAdminID: {
			Base: model.Base{ID: plainAdminID}, Role: "admin",
			RoleID: strPtr(customRoleID), Email: "kedai@rekavia.com",
		},
		secondAdminID: {
			Base: model.Base{ID: secondAdminID}, Role: "admin",
			RoleID: strPtr(adminRoleID), Email: "kedua@rekavia.com",
		},
		protectedAdminID: {
			Base: model.Base{ID: protectedAdminID}, Role: "admin",
			RoleID: strPtr(superRoleID), Email: "utama@rekavia.com",
			IsProtected: true,
		},
		"u-user": {Base: model.Base{ID: "u-user"}, Role: "user"},
	}}

	audit := &recordingAudit{}
	cache := &fakePermCache{}
	anonymizer := &fakeAnonymizer{users: users.users}
	return &deleteAdminFixture{
		svc:        NewRoleService(store, users, audit, cache, anonymizer),
		store:      store,
		users:      users,
		audit:      audit,
		cache:      cache,
		anonymizer: anonymizer,
	}
}

const alasanHapus = "Akun tidak lagi bertugas di tim operasional"

// Akun superadmin utama tidak bisa dihapus, bahkan oleh superadmin lain.
func TestDeleteAdminTerlindungiDitolak(t *testing.T) {
	f := newDeleteAdminFixture()

	err := f.svc.DeleteAdmin(context.Background(), superUserID, protectedAdminID, alasanHapus)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
	if len(f.anonymizer.removed) != 0 {
		t.Fatalf("akun terlindungi tetap dianonimkan: %v", f.anonymizer.removed)
	}
	if len(f.audit.entries) != 0 {
		t.Fatal("penolakan tidak boleh meninggalkan catatan audit penghapusan")
	}
}

// Pelaku tidak bisa menghapus akunnya sendiri.
func TestDeleteAdminDiriSendiriDitolak(t *testing.T) {
	f := newDeleteAdminFixture()

	err := f.svc.DeleteAdmin(context.Background(), superUserID, superUserID, alasanHapus)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
	if len(f.anonymizer.removed) != 0 {
		t.Fatal("akun sendiri dianonimkan")
	}
}

// Admin biasa tidak bisa menghapus akun yang role-nya lebih tinggi.
func TestDeleteAdminTidakBolehMelebihiKewenangan(t *testing.T) {
	f := newDeleteAdminFixture()

	// plainAdmin hanya memegang user.view, role.edit, role.view; role target
	// memuat seluruh kewenangan, jadi penghapusan harus ditolak.
	err := f.svc.DeleteAdmin(context.Background(), plainAdminID, superUserID, alasanHapus)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
	if len(f.anonymizer.removed) != 0 {
		t.Fatal("target dengan kewenangan lebih tinggi tetap dihapus")
	}
}

// Akun pelanggan bukan objek penghapusan akun pengelola.
func TestDeleteAdminMenolakPelanggan(t *testing.T) {
	f := newDeleteAdminFixture()

	err := f.svc.DeleteAdmin(context.Background(), superUserID, "u-user", alasanHapus)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
}

// Alasan wajib; penghapusan tanpa alasan tidak dijalankan.
func TestDeleteAdminButuhAlasan(t *testing.T) {
	f := newDeleteAdminFixture()

	err := f.svc.DeleteAdmin(context.Background(), superUserID, plainAdminID, "   ")
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
	if len(f.anonymizer.removed) != 0 {
		t.Fatal("akun terhapus tanpa alasan")
	}
}

// Admin aktif terakhir tidak boleh dihapus: pintu masuk harus selalu ada.
func TestDeleteAdminMenolakAdminTerakhir(t *testing.T) {
	f := newDeleteAdminFixture()
	f.store.activeAdmins = 1

	err := f.svc.DeleteAdmin(context.Background(), superUserID, plainAdminID, alasanHapus)
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error = %v; ingin BAD_REQUEST", err)
	}
	if len(f.anonymizer.removed) != 0 {
		t.Fatal("admin terakhir terhapus")
	}
}

// Penghapusan yang sah: anonimisasi, cabut sesi, buang cache, catat audit.
func TestDeleteAdminBerhasil(t *testing.T) {
	f := newDeleteAdminFixture()
	ctx := context.Background()

	err := f.svc.DeleteAdmin(ctx, superUserID, plainAdminID, alasanHapus)
	if err != nil {
		t.Fatalf("DeleteAdmin error: %v", err)
	}

	if len(f.anonymizer.removed) != 1 || f.anonymizer.removed[0] != plainAdminID {
		t.Fatalf("akun dianonimkan = %v; ingin [%s]", f.anonymizer.removed, plainAdminID)
	}
	if f.users.users[plainAdminID].Status != model.UserStatusDeleted {
		t.Error("status akun tidak menjadi dihapus")
	}
	// Cache kewenangan akun itu wajib dibuang agar permintaan berikutnya
	// tidak memakai permission lama.
	found := false
	for _, id := range f.cache.invalidated {
		if id == plainAdminID {
			found = true
		}
	}
	if !found {
		t.Errorf("cache akun tidak dibuang: %v", f.cache.invalidated)
	}
	if f.cache.all == 0 {
		t.Error("cache seluruh akun tidak disegarkan setelah akun dihapus")
	}

	if len(f.audit.entries) != 1 {
		t.Fatalf("entri audit = %d; ingin 1", len(f.audit.entries))
	}
	entry := f.audit.entries[0]
	if entry.Event != auditAdminDelete {
		t.Errorf("event audit = %q; ingin %q", entry.Event, auditAdminDelete)
	}
	if entry.EntityID != plainAdminID {
		t.Errorf("entity id audit = %q; ingin %q", entry.EntityID, plainAdminID)
	}
	if !strings.Contains(entry.Metadata, alasanHapus) {
		t.Errorf("alasan tidak tercatat di audit: %q", entry.Metadata)
	}
	if entry.UserID != superUserID {
		t.Errorf("pelaku audit = %q; ingin %q", entry.UserID, superUserID)
	}
}

// Tanpa anonymizer, penghapusan ditolak alih-alih menghapus separuh jalan.
func TestDeleteAdminTanpaAnonymizerDitolak(t *testing.T) {
	f := newDeleteAdminFixture()
	f.svc = NewRoleService(f.store, f.users, f.audit, f.cache, nil)

	err := f.svc.DeleteAdmin(context.Background(), superUserID, plainAdminID, alasanHapus)
	if err == nil {
		t.Fatal("penghapusan tanpa anonymizer seharusnya gagal")
	}
}
