package model

import "time"

// Plan adalah paket langganan yang dijual. Harga disalin ke invoice saat
// terbit, jadi perubahan di sini tidak mengubah invoice lama (plans/08 §3.3).
type Plan struct {
	Code        string    `gorm:"primaryKey;type:varchar(20)" json:"code"`
	Name        string    `gorm:"type:varchar(80);not null" json:"name"`
	PriceIDR    int64     `gorm:"not null;default:0;column:price_idr" json:"price_idr"`
	Interval    string    `gorm:"type:varchar(20);not null;default:monthly" json:"interval"`
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	SortOrder   int       `gorm:"not null;default:0" json:"sort_order"`
	Description string    `gorm:"type:varchar(255);not null;default:''" json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (Plan) TableName() string {
	return "plan"
}

// PlanFeature menyimpan batas dan fitur paket sebagai data, bukan kode.
// Dengan begitu menambah paket tidak perlu mengubah biner.
type PlanFeature struct {
	PlanCode   string `gorm:"type:varchar(20);primaryKey" json:"plan_code"`
	FeatureKey string `gorm:"type:varchar(60);primaryKey" json:"feature_key"`
	Value      string `gorm:"type:varchar(120);not null" json:"value"`
}

// TableName menetapkan nama tabel.
func (PlanFeature) TableName() string {
	return "plan_feature"
}

// Kunci fitur yang dikenali. Nilai disimpan sebagai teks lalu dikonversi
// sesuai kunci saat dibaca (plans/08 §3.2).
const (
	FeatureAccountLimit = "account_limit"
	FeatureMaxSymbols   = "max_symbols"
	FeatureAutoExecute  = "auto_execute"
	FeatureSignalDelay  = "signal_delay_minutes"
	FeatureStrategies   = "strategies"
	FeatureILA3Status   = "ila3_status"
	FeatureReasonAI     = "reason_ai"
	// FeatureMaxTotalRisk membatasi jumlah risiko seluruh akun aktif user,
	// dalam persen. Mencegah user memecah risiko ke banyak akun hingga melewati
	// kendali platform.
	FeatureMaxTotalRisk = "max_total_risk"
)

// PriceOverride adalah harga kampanye untuk satu paket.
// Maksimal satu kampanye aktif per paket ditegakkan di service.
type PriceOverride struct {
	Base
	PlanCode  string    `gorm:"type:varchar(20);index;not null" json:"plan_code"`
	PriceIDR  int64     `gorm:"not null;column:price_idr" json:"price_idr"`
	StartsAt  time.Time `json:"starts_at"`
	EndsAt    time.Time `json:"ends_at"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	Label     string    `gorm:"type:varchar(120);not null;default:''" json:"label"`
	CreatedBy string    `gorm:"type:varchar(36);not null" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (PriceOverride) TableName() string {
	return "price_override"
}

// PricingGuard adalah batas bawah harga: pengaman agar salah ketik diskon
// tidak menjual Plus seharga Rp 1 (plans/14 §6).
type PricingGuard struct {
	PlanCode    string    `gorm:"primaryKey;type:varchar(20)" json:"plan_code"`
	MinPriceIDR int64     `gorm:"not null;column:min_price_idr" json:"min_price_idr"`
	UpdatedBy   string    `gorm:"type:varchar(36);not null" json:"updated_by"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (PricingGuard) TableName() string {
	return "pricing_guard"
}

// Jenis potongan kupon.
const (
	DiscountPercent = "percent"
	DiscountFixed   = "fixed"
)

// Coupon adalah kode potongan sekali pakai per invoice.
type Coupon struct {
	Base
	Code              string    `gorm:"type:varchar(40);uniqueIndex;not null" json:"code"`
	Description       string    `gorm:"type:varchar(255);not null;default:''" json:"description"`
	DiscountType      string    `gorm:"type:varchar(10);not null" json:"discount_type"`
	DiscountValue     int64     `gorm:"not null" json:"discount_value"`
	MaxDiscountIDR    *int64    `gorm:"column:max_discount_idr" json:"max_discount_idr"`
	MinAmountIDR      int64     `gorm:"not null;default:0;column:min_amount_idr" json:"min_amount_idr"`
	AppliesToPlans    string    `gorm:"type:varchar(120);not null;default:''" json:"applies_to_plans"`
	FirstPurchaseOnly bool      `gorm:"not null;default:false" json:"first_purchase_only"`
	StartsAt          time.Time `json:"starts_at"`
	EndsAt            time.Time `json:"ends_at"`
	MaxRedemptions    int       `gorm:"not null;default:0" json:"max_redemptions"`
	MaxPerUser        int       `gorm:"not null;default:1" json:"max_per_user"`
	IsActive          bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedBy         string    `gorm:"type:varchar(36);not null" json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (Coupon) TableName() string {
	return "coupon"
}

// CouponRedemption mencatat pemakaian kupon oleh satu user.
//
// Unik (coupon_id, user_id) menegakkan aturan "satu kali per user" di tingkat
// basis data, bukan hanya lewat hitungan: dua checkout bersamaan tidak bisa
// memakai kupon yang sama dua kali (plans/14 §5).
type CouponRedemption struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	CouponID    string    `gorm:"type:varchar(36);not null;uniqueIndex:uq_coupon_user" json:"coupon_id"`
	UserID      string    `gorm:"type:varchar(36);not null;uniqueIndex:uq_coupon_user" json:"user_id"`
	InvoiceID   string    `gorm:"type:varchar(36);index;not null" json:"invoice_id"`
	DiscountIDR int64     `gorm:"not null;column:discount_idr" json:"discount_idr"`
	RedeemedAt  time.Time `json:"redeemed_at"`
}

// TableName menetapkan nama tabel.
func (CouponRedemption) TableName() string {
	return "coupon_redemption"
}

// CreditBalance adalah saldo kredit pengguna. Nilainya tidak boleh negatif.
type CreditBalance struct {
	UserID     string    `gorm:"primaryKey;type:varchar(36)" json:"user_id"`
	BalanceIDR int64     `gorm:"not null;default:0;column:balance_idr" json:"balance_idr"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (CreditBalance) TableName() string {
	return "credit_balance"
}

// Jenis baris buku besar kredit.
const (
	CreditKindCode   = "code"
	CreditKindManual = "manual"
	CreditKindUsed   = "used"
	CreditKindRefund = "refund"
)

// CreditCode adalah voucher saldo yang bisa ditukar pengguna.
// Nilainya dipakai sekali per kode per pengguna (plans/14 §5).
type CreditCode struct {
	Base
	Code           string    `gorm:"type:varchar(40);uniqueIndex;not null" json:"code"`
	AmountIDR      int64     `gorm:"not null;column:amount_idr" json:"amount_idr"`
	MaxRedemptions int       `gorm:"not null;default:1" json:"max_redemptions"`
	MaxPerUser     int       `gorm:"not null;default:1" json:"max_per_user"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	IsActive       bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedBy      string    `gorm:"type:varchar(36);not null" json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (CreditCode) TableName() string {
	return "credit_code"
}

// CreditRedemption mencatat penukaran kode kredit oleh satu pengguna.
// Unik (code_id, user_id) mencegah kode yang sama dipakai dua kali.
type CreditRedemption struct {
	ID         string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	CodeID     string    `gorm:"type:varchar(36);uniqueIndex:uq_credit_redeem;not null" json:"code_id"`
	UserID     string    `gorm:"type:varchar(36);uniqueIndex:uq_credit_redeem;not null" json:"user_id"`
	AmountIDR  int64     `gorm:"not null;column:amount_idr" json:"amount_idr"`
	RedeemedAt time.Time `json:"redeemed_at"`
}

// TableName menetapkan nama tabel.
func (CreditRedemption) TableName() string {
	return "credit_redemption"
}

// CreditTransaction adalah buku besar kredit. Setiap perubahan saldo wajib
// punya baris di sini dalam transaksi DB yang sama (plans/12 §5.7).
type CreditTransaction struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       string    `gorm:"type:varchar(36);index;not null" json:"user_id"`
	DeltaIDR     int64     `gorm:"not null;column:delta_idr" json:"delta_idr"`
	BalanceAfter int64     `gorm:"not null" json:"balance_after"`
	Kind         string    `gorm:"type:varchar(20);not null" json:"kind"`
	InvoiceID    *string   `gorm:"type:varchar(36)" json:"invoice_id"`
	Reason       string    `gorm:"type:varchar(255);not null;default:''" json:"reason"`
	ActorID      *string   `gorm:"type:varchar(36)" json:"actor_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (CreditTransaction) TableName() string {
	return "credit_transaction"
}
