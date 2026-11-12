// Package audit mendefinisikan katalog aksi audit dan penyaring rahasia.
//
// Katalog dipakai bersama oleh perekam audit, tampilan admin, dan tes pemindai
// rute, sehingga tidak ada aksi yang tercatat dengan nama yang tidak dikenal
// (plans/24 §7.2).
package audit

// Action adalah kode aksi audit dengan format grup.aksi.
type Action string

// Sesi & autentikasi.
const (
	LoginSuccess   Action = "auth.login.success"
	LoginFailed    Action = "auth.login.failed"
	Logout         Action = "auth.logout"
	LogoutAll      Action = "auth.logout_all"
	PasswordChange Action = "auth.password.change"
	TwoFAEnable    Action = "auth.2fa.enable"
	TwoFADisable   Action = "auth.2fa.disable"
	TwoFARecovery  Action = "auth.2fa.recovery_used"
	SessionRevoke  Action = "auth.session.revoked"
)

// Admin terhadap pengguna.
const (
	UserSuspend       Action = "user.suspend"
	UserUnsuspend     Action = "user.unsuspend"
	UserBan           Action = "user.ban"
	UserUnban         Action = "user.unban"
	UserDelete        Action = "user.delete"
	UserTierChange    Action = "user.tier.change"
	UserRoleAssign    Action = "user.role.assign"
	UserViewSensitive Action = "user.view.sensitive"
	UserExport        Action = "user.export"
	UserNoteAdd       Action = "user.note.add"
	SupportNoteAdd    Action = "support.note.add"
	// UserCreate mencatat pembuatan akun pelanggan oleh admin.
	UserCreate Action = "user.create"
	// AdminCreate mencatat pembuatan akun pengelola beserta role awalnya.
	// Kata sandi tidak pernah ikut tercatat.
	AdminCreate Action = "admin.create"
	// AdminDelete mencatat penghapusan akun pengelola. Baris pengguna hanya
	// dianonimkan, sehingga identitas pada catatan audit ini menjadi satu-satunya
	// jejak akun yang dihapus.
	AdminDelete Action = "admin.delete"
	// Undangan komunitas: pembuatan tautan, pencabutan, dan pemakaian.
	// Kode mentah tidak pernah masuk catatan audit.
	InviteCreate Action = "invite.create"
	InviteRevoke Action = "invite.revoke"
	InviteRedeem Action = "invite.redeem"
)

// Akun broker.
const (
	AccountCreate       Action = "account.create"
	AccountUpdate       Action = "account.update"
	AccountDelete       Action = "account.delete"
	AccountFreeze       Action = "account.freeze"
	AccountUnfreeze     Action = "account.unfreeze"
	AccountViewCredMeta Action = "account.view.credential_meta"
)

// Uang.
const (
	InvoiceCreate      Action = "invoice.create"
	InvoicePaid        Action = "invoice.paid"
	InvoiceVoid        Action = "invoice.void"
	PaymentReceived    Action = "payment.received"
	InvoiceMarkPaid    Action = "invoice.paid.manual"
	PaymentFailed      Action = "payment.failed"
	RefundCreate       Action = "refund.create"
	CreditIssue        Action = "credit.issue"
	CreditAdjust       Action = "credit.adjust"
	CreditRevoke       Action = "credit.revoke"
	PlanPriceChange    Action = "plan.price.change"
	PricingGuardChange Action = "pricing_guard.change"
	CouponCreate       Action = "coupon.create"
	CouponUpdate       Action = "coupon.update"
	CouponDisable      Action = "coupon.disable"
)

// Sistem & tata kelola.
const (
	SystemFlagChange Action = "system.flag.change"
	SystemModeChange Action = "system.mode.change"
	LegalPublish     Action = "legal.publish"
	CMSFaqChange     Action = "cms.faq.change"
	CMSAnnouncement  Action = "cms.announcement.change"
	CMSPageChange    Action = "cms.page.change"
	RoleCreate       Action = "role.create"
	RoleUpdate       Action = "role.update"
	RoleDelete       Action = "role.delete"
	RolePermChange   Action = "role.permission.change"
)

// Ekspor data oleh admin.
const (
	AuditLogExport       Action = "audit.export"
	BusinessReportExport Action = "report.export"
)

// Katalog pair yang dikelola admin.
const (
	PairCategoryChange Action = "pair.category.change"
	PairGroupChange    Action = "pair.group.change"
	PairChange         Action = "pair.change"
)

// Catalog memetakan kode aksi ke keterangan dan penanda sensitif.
type Entry struct {
	Action    Action
	Group     string
	Label     string
	Sensitive bool
}

// Catalog adalah daftar lengkap aksi yang dikenali.
var Catalog = []Entry{
	{LoginSuccess, "Sesi", "Login berhasil", false},
	{LoginFailed, "Sesi", "Login gagal", false},
	{Logout, "Sesi", "Logout", false},
	{LogoutAll, "Sesi", "Logout semua perangkat", false},
	{PasswordChange, "Sesi", "Ganti kata sandi", true},
	{TwoFAEnable, "Sesi", "2FA diaktifkan", false},
	{TwoFADisable, "Sesi", "2FA dimatikan", true},
	{TwoFARecovery, "Sesi", "Kode pemulihan dipakai", true},
	{SessionRevoke, "Sesi", "Sesi perangkat dicabut", false},

	{UserSuspend, "Pengguna", "Akun ditangguhkan", false},
	{UserUnsuspend, "Pengguna", "Penangguhan dibuka", false},
	{UserBan, "Pengguna", "Akun diblokir", true},
	{UserUnban, "Pengguna", "Blokir dibuka", true},
	{UserDelete, "Pengguna", "Akun dihapus", true},
	{UserTierChange, "Pengguna", "Tier diubah", true},
	{UserRoleAssign, "Pengguna", "Role ditugaskan", false},
	{UserViewSensitive, "Pengguna", "Membuka data sensitif pengguna", true},
	{UserExport, "Pengguna", "Ekspor data pengguna", true},
	{UserNoteAdd, "Pengguna", "Catatan internal ditambah", false},
	{SupportNoteAdd, "Pengguna", "Catatan dukungan ditambah", false},
	{UserCreate, "Pengguna", "Akun pelanggan dibuat oleh admin", false},
	{AdminCreate, "Pengguna", "Akun pengelola dibuat", true},
	{AdminDelete, "Pengguna", "Akun pengelola dihapus", true},

	{InviteCreate, "Undangan", "Tautan undangan dibuat", false},
	{InviteRevoke, "Undangan", "Tautan undangan dicabut", true},
	{InviteRedeem, "Undangan", "Akun dibuat lewat undangan", false},

	{AccountCreate, "Akun Broker", "Akun broker ditambah", false},
	{AccountUpdate, "Akun Broker", "Akun broker diubah", false},
	{AccountDelete, "Akun Broker", "Akun broker dihapus", true},
	{AccountFreeze, "Akun Broker", "Eksekusi akun dibekukan", false},
	{AccountUnfreeze, "Akun Broker", "Pembekuan dibuka", false},
	{AccountViewCredMeta, "Akun Broker", "Metadata kredensial dilihat", true},

	{InvoiceCreate, "Uang", "Invoice terbit", false},
	{InvoicePaid, "Uang", "Invoice lunas", false},
	{InvoiceVoid, "Uang", "Invoice dibatalkan", true},
	{PaymentReceived, "Uang", "Pembayaran diterima", false},
	{InvoiceMarkPaid, "Uang", "Pembayaran manual dicatat", true},
	{PaymentFailed, "Uang", "Pembayaran gagal", false},
	{RefundCreate, "Uang", "Refund dicatat", true},
	{CreditIssue, "Uang", "Kredit diberikan", true},
	{CreditAdjust, "Uang", "Kredit disesuaikan", true},
	{CreditRevoke, "Uang", "Kredit dicabut", true},
	{PlanPriceChange, "Uang", "Harga paket diubah", true},
	{PricingGuardChange, "Uang", "Batas bawah harga diubah", true},
	{CouponCreate, "Uang", "Kupon dibuat", false},
	{CouponUpdate, "Uang", "Kupon diubah", false},
	{CouponDisable, "Uang", "Kupon dimatikan", false},

	{SystemFlagChange, "Sistem", "Flag sistem diubah", true},
	{SystemModeChange, "Sistem", "Mode pemeliharaan diubah", true},
	{LegalPublish, "Sistem", "Dokumen legal diterbitkan", false},
	{CMSFaqChange, "Konten", "FAQ diubah", false},
	{CMSAnnouncement, "Konten", "Pengumuman diubah", false},
	{CMSPageChange, "Konten", "Isi halaman publik diubah", false},
	{PairCategoryChange, "Katalog Pair", "Kategori pair diubah", false},
	{PairGroupChange, "Katalog Pair", "Sub-kategori pair diubah", false},
	{PairChange, "Katalog Pair", "Pair diubah", false},

	{RoleCreate, "Tata Kelola", "Role dibuat", false},
	{RoleUpdate, "Tata Kelola", "Role diubah", true},
	{RoleDelete, "Tata Kelola", "Role dihapus", true},
	{RolePermChange, "Tata Kelola", "Permission role berubah", true},

	{AuditLogExport, "Tata Kelola", "Audit log diekspor", true},
	{BusinessReportExport, "Uang", "Laporan bisnis diekspor", true},
}

// Known menandai aksi yang terdaftar di katalog.
func Known(a Action) bool {
	for _, entry := range Catalog {
		if entry.Action == a {
			return true
		}
	}
	return false
}

// Groups mengelompokkan katalog untuk pilihan filter di admin.
func Groups() []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 8)
	for _, entry := range Catalog {
		if _, ok := seen[entry.Group]; ok {
			continue
		}
		seen[entry.Group] = struct{}{}
		out = append(out, entry.Group)
	}
	return out
}

// Destructive menandai aksi yang wajib beralasan minimal 10 karakter.
func Destructive(a Action) bool {
	switch a {
	case UserSuspend, UserBan, UserDelete, UserTierChange, AccountFreeze,
		AccountDelete, RefundCreate, CreditAdjust, CreditRevoke, InvoiceVoid,
		PlanPriceChange, PricingGuardChange, RoleDelete, AdminDelete:
		return true
	default:
		return false
	}
}
