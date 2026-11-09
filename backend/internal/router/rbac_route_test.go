package router

import (
	"sort"
	"strings"
	"testing"

	"ila/backend/internal/authz"
	"ila/backend/internal/middleware"
)

// permissionBelumBerroute adalah permission yang sudah ada di katalog tetapi
// fiturnya belum dibangun di backend-api. Daftar ini sengaja eksplisit supaya
// setiap penambahan atau penghapusan route RBAC terlihat di tinjauan kode.
//
// Menghapus satu baris di sini hanya sah setelah route-nya benar-benar ada.
var permissionBelumBerroute = map[string]string{
	authz.AccountEdit:     "admin belum bisa mengubah akun broker pelanggan",
	authz.AccountBalance:  "saldo pelanggan dibaca lewat telemetri, bukan route admin",
	authz.AccountDelete:   "penghapusan akun broker belum disediakan untuk admin",
	authz.SignalView:      "panel sinyal admin belum dibangun",
	authz.SignalTrace:     "penelusuran riwayat sinyal admin belum dibangun",
	authz.TradeExport:     "ekspor riwayat posisi pelanggan belum dibangun",
	authz.BillingExport:   "ekspor laporan bisnis belum dibangun",
	authz.AuditExport:     "ekspor audit log belum dibangun",
	authz.BillingSettings: "kunci gateway pembayaran belum dapat diubah dari panel",
}

// Setiap permission yang dipakai route wajib terdaftar di katalog, sehingga
// tidak ada penjagaan route yang mengacu ke kunci asing.
func TestPermissionRouteTerdaftarDiKatalog(t *testing.T) {
	cfg := testConfig(t)
	NewBackendAPI(cfg, testDB(t), testLogger())

	routes := middleware.AdminRoutePermissions()
	if len(routes) == 0 {
		t.Fatal("tidak ada route admin yang terdeklarasi")
	}

	for key, permission := range routes {
		// "-" berarti cukup sesi admin, tanpa permission tertentu.
		if permission == "-" {
			continue
		}
		if !authz.Known(permission) {
			t.Errorf("%s dijaga permission %q yang tidak ada di katalog", key, permission)
		}
	}
}

// Setiap permission di katalog harus dipakai minimal satu route, atau tercatat
// sebagai fitur yang belum dibangun. Katalog yang tidak dipakai membuat panel
// menampilkan centang izin yang tidak berpengaruh apa pun.
func TestKatalogPermissionPunyaRoute(t *testing.T) {
	cfg := testConfig(t)
	NewBackendAPI(cfg, testDB(t), testLogger())

	used := make(map[string]bool)
	for _, permission := range middleware.AdminRoutePermissions() {
		used[permission] = true
	}

	var orphan []string
	for _, key := range authz.AllKeys() {
		if used[key] {
			if _, pending := permissionBelumBerroute[key]; pending {
				t.Errorf("%s sudah punya route; hapus dari daftar permissionBelumBerroute", key)
			}
			continue
		}
		if _, pending := permissionBelumBerroute[key]; !pending {
			orphan = append(orphan, key)
		}
	}

	if len(orphan) > 0 {
		sort.Strings(orphan)
		t.Fatalf("permission tanpa route dan tanpa catatan: %s", strings.Join(orphan, ", "))
	}
}

// Role admin bawaan harus memegang seluruh permission route, kecuali empat
// kewenangan yang memang dikunci untuk superadmin. Tes ini menjaga agar
// penambahan fitur admin baru tidak diam-diam mengunci akun admin.
func TestRoleAdminMemegangSeluruhPermissionRoute(t *testing.T) {
	cfg := testConfig(t)
	NewBackendAPI(cfg, testDB(t), testLogger())

	adminPerms := make(map[string]bool)
	for _, role := range authz.DefaultRoles() {
		if role.Code != authz.RoleAdmin {
			continue
		}
		for _, key := range role.Permissions {
			adminPerms[key] = true
		}
	}
	if len(adminPerms) == 0 {
		t.Fatal("role admin tidak ada di daftar bawaan")
	}

	// Kewenangan terkunci: hanya superadmin, walau granted lewat role.
	locked := map[string]bool{
		authz.UserDelete:        true,
		authz.SystemSettings:    true,
		authz.SystemMaintenance: true,
		authz.BillingSettings:   true,
	}

	var missing []string
	for key, permission := range middleware.AdminRoutePermissions() {
		if permission == "-" || locked[permission] {
			continue
		}
		if !adminPerms[permission] {
			missing = append(missing, key+" ("+permission+")")
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("role admin kehilangan akses: %s", strings.Join(missing, ", "))
	}
}
