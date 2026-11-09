package router

import (
	"net/http"
	"strings"
	"testing"

	"ila/backend/internal/audit"
)

// Route mutasi admin yang memang TIDAK menulis audit karena tidak mengubah
// apa pun yang perlu dipertanggungjawabkan. Daftar ini harus tetap kecil.
var mutasiTanpaAudit = map[string]bool{
	"POST /api/v1/admin/auth/login":      true,
	"POST /api/v1/admin/auth/verify-2fa": true,
	"POST /api/v1/admin/auth/logout":     true,
	// Heartbeat hanya memperpanjang sesi sendiri, tidak mengubah data.
	"POST /api/v1/admin/auth/heartbeat": true,
}

// Aksi audit yang diharapkan muncul untuk setiap pasangan metode+path.
// Tes ini menangkap route mutasi baru yang lupa mencatat jejak.
var harapanAksiAudit = map[string]audit.Action{
	"POST /api/v1/admin/users/:id/suspend":     audit.UserSuspend,
	"POST /api/v1/admin/users/:id/unsuspend":   audit.UserUnsuspend,
	"POST /api/v1/admin/users/:id/ban":         audit.UserBan,
	"POST /api/v1/admin/users/:id/unban":       audit.UserUnban,
	"POST /api/v1/admin/users/:id/delete":      audit.UserDelete,
	"PATCH /api/v1/admin/users/:id/tier":       audit.UserTierChange,
	"POST /api/v1/admin/users/:id/notes":       audit.UserNoteAdd,
	"POST /api/v1/admin/accounts/:id/freeze":   audit.AccountFreeze,
	"POST /api/v1/admin/accounts/:id/unfreeze": audit.AccountUnfreeze,
	"PATCH /api/v1/admin/accounts/:id/label":   audit.AccountUpdate,
	"DELETE /api/v1/admin/accounts/:id":        audit.AccountDelete,
	"GET /api/v1/admin/users/:id/positions":    audit.UserViewSensitive,

	// 2FA admin memakai faktor milik sendiri, bukan kewenangan atas orang lain.
	"POST /api/v1/admin/2fa/setup":          audit.TwoFAEnable,
	"POST /api/v1/admin/2fa/activate":       audit.TwoFAEnable,
	"POST /api/v1/admin/2fa/disable":        audit.TwoFADisable,
	"POST /api/v1/admin/2fa/recovery-codes": audit.TwoFARecovery,

	"POST /api/v1/admin/users/:id/credit": audit.CreditAdjust,

	// Voucher saldo dan uang masuk/keluar selalu meninggalkan jejak.
	"POST /api/v1/admin/credit-codes":             audit.CreditAdjust,
	"POST /api/v1/admin/credit-codes/:id/disable": audit.CreditAdjust,
	"POST /api/v1/admin/invoices/:id/mark-paid":   audit.InvoiceMarkPaid,
	"POST /api/v1/admin/invoices/:id/refund":      audit.RefundCreate,
	"POST /api/v1/admin/trials/:id/extend":        audit.UserTierChange,
	"POST /api/v1/admin/trials/:id/end":           audit.UserTierChange,

	// Konten dan RBAC sudah mencatat sendiri lewat recordAudit di service-nya.
	"POST /api/v1/admin/legal":               audit.LegalPublish,
	"PATCH /api/v1/admin/legal/:id":          audit.LegalPublish,
	"POST /api/v1/admin/legal/:id/publish":   audit.LegalPublish,
	"POST /api/v1/admin/roles":               audit.RoleCreate,
	"PATCH /api/v1/admin/roles/:id":          audit.RolePermChange,
	"DELETE /api/v1/admin/roles/:id":         audit.RoleDelete,
	"PATCH /api/v1/admin/plans/:code":        audit.PlanPriceChange,
	"PUT /api/v1/admin/plans/:code/guard":    audit.PricingGuardChange,
	"POST /api/v1/admin/coupons":             audit.CouponCreate,
	"PATCH /api/v1/admin/coupons/:id":        audit.CouponUpdate,
	"POST /api/v1/admin/coupons/:id/disable": audit.CouponDisable,
	"PATCH /api/v1/admin/users/:id/role":     audit.UserRoleAssign,
	"POST /api/v1/admin/admins":              audit.AdminCreate,

	// Flag sistem dan mode platform selalu menyentuh jalur trading.
	"PUT /api/v1/admin/system/settings/:key":     audit.SystemFlagChange,
	"DELETE /api/v1/admin/system/settings/:key":  audit.SystemFlagChange,
	"PUT /api/v1/admin/system/mode":              audit.SystemModeChange,
	"POST /api/v1/admin/users/:id/support-notes": audit.SupportNoteAdd,

	// Konten: FAQ dan pengumuman tidak menyentuh uang maupun trading.
	"POST /api/v1/admin/faq":                 audit.CMSFaqChange,
	"PATCH /api/v1/admin/faq/:id":            audit.CMSFaqChange,
	"DELETE /api/v1/admin/faq/:id":           audit.CMSFaqChange,
	"POST /api/v1/admin/announcements":       audit.CMSAnnouncement,
	"PATCH /api/v1/admin/announcements/:id":  audit.CMSAnnouncement,
	"DELETE /api/v1/admin/announcements/:id": audit.CMSAnnouncement,
	"PUT /api/v1/admin/pages/:slug":          audit.CMSPageChange,

	// Katalog pair: menentukan simbol yang boleh dipilih pengguna.
	"POST /api/v1/admin/pair-categories":       audit.PairCategoryChange,
	"PATCH /api/v1/admin/pair-categories/:id":  audit.PairCategoryChange,
	"DELETE /api/v1/admin/pair-categories/:id": audit.PairCategoryChange,
	"POST /api/v1/admin/pair-groups":           audit.PairGroupChange,
	"PATCH /api/v1/admin/pair-groups/:id":      audit.PairGroupChange,
	"DELETE /api/v1/admin/pair-groups/:id":     audit.PairGroupChange,
	"POST /api/v1/admin/pairs":                 audit.PairChange,
	"PATCH /api/v1/admin/pairs/:id":            audit.PairChange,
	"DELETE /api/v1/admin/pairs/:id":           audit.PairChange,
}

// Setiap route mutasi admin harus punya kode aksi audit yang terdaftar.
func TestSetiapMutasiAdminPunyaAksiAudit(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	for _, info := range r.Routes() {
		if !strings.HasPrefix(info.Path, "/api/v1/admin/") {
			continue
		}
		switch info.Method {
		case http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete:
		default:
			continue
		}

		key := info.Method + " " + info.Path
		if mutasiTanpaAudit[key] {
			continue
		}
		action, ok := harapanAksiAudit[key]
		if !ok {
			t.Fatalf("route mutasi admin tanpa aksi audit terdaftar: %s", key)
		}
		if !audit.Known(action) {
			t.Fatalf("aksi audit %q untuk %s tidak ada di katalog", action, key)
		}
	}
}

// Pembacaan riwayat posisi wajib tercatat sebagai baca sensitif.
func TestBacaSensitifTerdaftar(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	wanted := "GET /api/v1/admin/users/:id/positions"
	for _, info := range r.Routes() {
		if info.Method+" "+info.Path == wanted {
			if !audit.Known(audit.UserViewSensitive) {
				t.Fatal("aksi baca sensitif harus ada di katalog")
			}
			return
		}
	}
	t.Fatalf("route %s tidak terpasang", wanted)
}

// Tidak boleh ada endpoint yang menghapus jejak audit.
func TestTidakAdaRouteHapusAudit(t *testing.T) {
	cfg := testConfig(t)
	r := NewBackendAPI(cfg, testDB(t), testLogger())

	for _, info := range r.Routes() {
		if info.Method == http.MethodDelete && strings.Contains(info.Path, "/audit") {
			t.Fatalf("audit tidak boleh bisa dihapus: %s", info.Path)
		}
	}
}
