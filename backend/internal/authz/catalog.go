// Package authz adalah katalog tunggal permission dan role bawaan platform.
//
// Deklarasi di sini yang dipakai middleware, seed database, dan pengujian,
// sehingga tidak mungkin ada permission yang dipakai route tetapi tidak
// terdaftar (plans/04 §2).
package authz

// Tingkat risiko dipakai di tampilan admin agar pemberi izin sadar dampaknya.
const (
	RiskLow    = "rendah"
	RiskMedium = "sedang"
	RiskHigh   = "tinggi"
)

// Permission beserta label manusiawinya.
type Permission struct {
	Key   string
	Group string
	Label string
	Risk  string
}

// Pelanggan.
const (
	UserView    = "user.view"
	UserEdit    = "user.edit"
	UserSuspend = "user.suspend"
	UserBan     = "user.ban"
	UserDelete  = "user.delete"
	UserTier    = "user.tier"
	UserLegal   = "user.legal"
	UserExport  = "user.export"
)

// Akun broker & langganan.
const (
	AccountView      = "account.view"
	AccountEdit      = "account.edit"
	AccountFreeze    = "account.freeze"
	AccountBalance   = "account.balance"
	AccountDelete    = "account.delete"
	SubscriptionEdit = "subscription.edit"
)

// Trading.
const (
	TradeView   = "trade.view"
	TradeExport = "trade.export"
	SignalView  = "signal.view"
	SignalTrace = "signal.trace"
)

// Billing.
const (
	BillingView    = "billing.view"
	BillingManual  = "billing.manual"
	BillingRefund  = "billing.refund"
	BillingPlan    = "billing.plan"
	BillingVoucher = "billing.voucher"
	BillingExport  = "billing.export"
)

// Konten.
const (
	CMSView    = "cms.view"
	CMSEdit    = "cms.edit"
	CMSPublish = "cms.publish"
)

// Tata kelola.
const (
	RoleView    = "role.view"
	RoleEdit    = "role.edit"
	AuditView   = "audit.view"
	AuditExport = "audit.export"
)

// Sistem.
const (
	SystemView        = "system.view"
	SystemSettings    = "system.settings"
	SystemMaintenance = "system.maintenance"
)

// Catalog adalah daftar lengkap permission yang dikenali sistem.
var Catalog = []Permission{
	{UserView, "Pelanggan", "Lihat daftar & detail pelanggan", RiskLow},
	{UserEdit, "Pelanggan", "Ubah profil pelanggan", RiskMedium},
	{UserSuspend, "Pelanggan", "Tangguhkan akun", RiskMedium},
	{UserBan, "Pelanggan", "Blokir akun permanen", RiskHigh},
	{UserDelete, "Pelanggan", "Hapus akun", RiskHigh},
	{UserTier, "Pelanggan", "Ubah tier manual", RiskHigh},
	{UserLegal, "Pelanggan", "Lihat bukti persetujuan legal", RiskLow},
	{UserExport, "Pelanggan", "Ekspor data pelanggan", RiskHigh},

	{AccountView, "Akun Broker", "Lihat akun broker tanpa kredensial", RiskLow},
	{AccountEdit, "Akun Broker", "Ubah label, mode, dan risiko", RiskMedium},
	{AccountFreeze, "Akun Broker", "Bekukan eksekusi akun", RiskMedium},
	{AccountBalance, "Akun Broker", "Lihat saldo & ekuitas", RiskHigh},
	{AccountDelete, "Akun Broker", "Hapus akun beserta kredensial", RiskHigh},
	{SubscriptionEdit, "Akun Broker", "Atur simbol & strategi pelanggan", RiskMedium},

	{TradeView, "Trading", "Lihat riwayat posisi", RiskLow},
	{TradeExport, "Trading", "Ekspor riwayat posisi", RiskHigh},
	{SignalView, "Trading", "Lihat sinyal terbit", RiskLow},
	{SignalTrace, "Trading", "Telusuri perjalanan sinyal", RiskLow},

	{BillingView, "Billing", "Lihat invoice & pembayaran", RiskLow},
	{BillingManual, "Billing", "Catat pembayaran manual", RiskHigh},
	{BillingRefund, "Billing", "Tandai refund", RiskHigh},
	{BillingPlan, "Billing", "Ubah harga & fitur paket", RiskHigh},
	{BillingVoucher, "Billing", "Buat kupon & beri kredit", RiskHigh},
	{BillingExport, "Billing", "Ekspor laporan bisnis", RiskHigh},

	{CMSView, "Konten", "Lihat dokumen, FAQ, pengumuman", RiskLow},
	{CMSEdit, "Konten", "Ubah draf konten", RiskMedium},
	{CMSPublish, "Konten", "Terbitkan konten", RiskHigh},

	{RoleView, "Tata Kelola", "Lihat role & permission", RiskLow},
	{RoleEdit, "Tata Kelola", "Kelola role & penugasan", RiskHigh},
	{AuditView, "Tata Kelola", "Lihat audit log", RiskMedium},
	{AuditExport, "Tata Kelola", "Ekspor audit log", RiskHigh},

	{SystemView, "Sistem", "Lihat status engine & runner", RiskLow},
	{SystemSettings, "Sistem", "Lihat nilai flag sistem", RiskMedium},
	{SystemMaintenance, "Sistem", "Picu operasi pemeliharaan", RiskHigh},
}

// Codes role bawaan.
const (
	RoleSuperadmin = "superadmin"
	RoleAdmin      = "admin"
	RoleSupport    = "support"
	RoleFinance    = "finance"
)

// AllKeys mengembalikan seluruh kunci permission.
func AllKeys() []string {
	keys := make([]string, 0, len(Catalog))
	for _, p := range Catalog {
		keys = append(keys, p.Key)
	}
	return keys
}

// Known menandai permission yang terdaftar.
func Known(key string) bool {
	for _, p := range Catalog {
		if p.Key == key {
			return true
		}
	}
	return false
}

// DefaultRole adalah role bawaan beserta permission-nya.
type DefaultRole struct {
	Code        string
	Name        string
	Description string
	Permissions []string
}

// DefaultRoles mengembalikan empat role bawaan sesuai plans/04 §3.
func DefaultRoles() []DefaultRole {
	all := AllKeys()

	// Admin: seluruh permission kecuali empat yang dijaga.
	excluded := map[string]bool{
		SystemMaintenance: true,
		SystemSettings:    true,
		UserDelete:        true,
	}
	adminPerms := make([]string, 0, len(all))
	for _, key := range all {
		if !excluded[key] {
			adminPerms = append(adminPerms, key)
		}
	}

	return []DefaultRole{
		{
			Code:        RoleSuperadmin,
			Name:        "Superadmin",
			Description: "Kendali penuh termasuk flag sistem.",
			Permissions: all,
		},
		{
			Code:        RoleAdmin,
			Name:        "Admin",
			Description: "Pengelola harian tanpa akses flag sistem dan hapus akun.",
			Permissions: adminPerms,
		},
		{
			Code:        RoleSupport,
			Name:        "Support",
			Description: "Bantuan pelanggan: baca data dan telusuri sinyal.",
			Permissions: []string{
				UserView, UserLegal,
				AccountView, TradeView, SignalView, SignalTrace,
				SystemView,
			},
		},
		{
			Code:        RoleFinance,
			Name:        "Finance",
			Description: "Mengurus uang, harga, dan penyesuaian tier.",
			Permissions: []string{
				BillingView, BillingManual, BillingRefund,
				BillingPlan, BillingVoucher, BillingExport,
				UserTier, UserView, AuditView,
			},
		},
	}
}

// SuperadminOnly menandai kewenangan yang tidak bisa didelegasikan lewat RBAC
// (plans/04 §6). Dikunci ke role superadmin, bukan menjadi permission.
func SuperadminOnly(permissionKey string) bool {
	switch permissionKey {
	case SystemSettings, SystemMaintenance:
		return true
	default:
		return false
	}
}
