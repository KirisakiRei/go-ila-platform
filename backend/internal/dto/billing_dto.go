package dto

// PriceQuote adalah rincian harga sebelum invoice dibuat.
// Nilai ini yang disalin ke invoice saat terbit (plans/14 §3.2).
type PriceQuote struct {
	PlanCode    string `json:"plan_code"`
	PlanName    string `json:"plan_name"`
	Subtotal    int64  `json:"subtotal"`
	CouponCode  string `json:"coupon_code,omitempty"`
	CouponLabel string `json:"coupon_label,omitempty"`
	Discount    int64  `json:"discount"`
	CreditUsed  int64  `json:"credit_used"`
	Total       int64  `json:"total"`
	Promo       string `json:"promo,omitempty"`
}

// PublicPlan adalah paket yang tampil di halaman harga.
type PublicPlan struct {
	Code           string            `json:"code"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	PriceIDR       int64             `json:"price_idr"`
	Interval       string            `json:"interval"`
	Promo          string            `json:"promo,omitempty"`
	Features       map[string]string `json:"features"`
	ILA3ComingSoon bool              `json:"ila3_coming_soon"`
	Active         bool              `json:"active"`
}

// QuoteRequest adalah permintaan hitung harga.
type QuoteRequest struct {
	PlanCode   string `json:"plan_code" binding:"required,max=20"`
	CouponCode string `json:"coupon_code" binding:"omitempty,max=40"`
	UseCredit  bool   `json:"use_credit"`
}

// PlanRequest menyimpan perubahan paket oleh admin.
type PlanRequest struct {
	Name        string            `json:"name" binding:"required,max=80"`
	PriceIDR    int64             `json:"price_idr" binding:"required,min=0"`
	Description string            `json:"description" binding:"max=255"`
	IsActive    *bool             `json:"is_active"`
	Features    map[string]string `json:"features"`
	Reason      string            `json:"reason" binding:"required,max=500"`
}

// GuardRequest mengubah batas bawah harga satu paket.
type GuardRequest struct {
	MinPriceIDR int64  `json:"min_price_idr" binding:"required,min=1"`
	Reason      string `json:"reason" binding:"required,max=500"`
}

// CouponRequest membuat atau mengubah kupon.
type CouponRequest struct {
	Code              string `json:"code" binding:"required,max=40"`
	Description       string `json:"description" binding:"max=255"`
	DiscountType      string `json:"discount_type" binding:"required,oneof=percent fixed"`
	DiscountValue     int64  `json:"discount_value" binding:"required,min=1"`
	MaxDiscountIDR    *int64 `json:"max_discount_idr" binding:"omitempty,min=1"`
	MinAmountIDR      int64  `json:"min_amount_idr" binding:"omitempty,min=0"`
	AppliesToPlans    string `json:"applies_to_plans" binding:"max=120"`
	FirstPurchaseOnly bool   `json:"first_purchase_only"`
	StartsAt          string `json:"starts_at" binding:"required"`
	EndsAt            string `json:"ends_at" binding:"required"`
	MaxRedemptions    int    `json:"max_redemptions" binding:"omitempty,min=0"`
	MaxPerUser        int    `json:"max_per_user" binding:"omitempty,min=1"`
	Reason            string `json:"reason" binding:"max=500"`
}

// CouponItem adalah baris daftar kupon beserta pemakaiannya.
type CouponItem struct {
	ID             string   `json:"id"`
	Code           string   `json:"code"`
	Description    string   `json:"description"`
	DiscountType   string   `json:"discount_type"`
	DiscountValue  int64    `json:"discount_value"`
	MaxDiscountIDR *int64   `json:"max_discount_idr,omitempty"`
	MinAmountIDR   int64    `json:"min_amount_idr"`
	AppliesToPlans string   `json:"applies_to_plans"`
	FirstOnly      bool     `json:"first_purchase_only"`
	StartsAt       TimePair `json:"starts_at"`
	EndsAt         TimePair `json:"ends_at"`
	MaxRedemptions int      `json:"max_redemptions"`
	MaxPerUser     int      `json:"max_per_user"`
	IsActive       bool     `json:"is_active"`
	Redeemed       int64    `json:"redeemed"`
	// Status dihitung server agar tampilan tidak menebak.
	Status string `json:"status"`
}

// CreditRequest memberi atau menyesuaikan kredit satu pengguna.
type CreditRequest struct {
	AmountIDR int64  `json:"amount_idr" binding:"required"`
	Reason    string `json:"reason" binding:"required,max=255"`
}

// CreditSummary adalah saldo beserta buku besarnya.
type CreditSummary struct {
	UserID  string            `json:"user_id"`
	Balance int64             `json:"balance_idr"`
	History []CreditEntryItem `json:"history"`
}

// CreditEntryItem adalah satu baris buku besar kredit.
type CreditEntryItem struct {
	ID           string   `json:"id"`
	DeltaIDR     int64    `json:"delta_idr"`
	BalanceAfter int64    `json:"balance_after"`
	Kind         string   `json:"kind"`
	Reason       string   `json:"reason"`
	CreatedAt    TimePair `json:"created_at"`
}

// CreditCodeRequest membuat kode kredit yang bisa ditukar pengguna.
type CreditCodeRequest struct {
	Code           string `json:"code" binding:"required,max=40"`
	AmountIDR      int64  `json:"amount_idr" binding:"required,min=1"`
	MaxRedemptions int    `json:"max_redemptions" binding:"omitempty,min=1"`
	MaxPerUser     int    `json:"max_per_user" binding:"omitempty,min=1"`
	StartsAt       string `json:"starts_at" binding:"required"`
	EndsAt         string `json:"ends_at" binding:"required"`
	Reason         string `json:"reason" binding:"max=500"`
}

// CreditCodeItem adalah satu baris daftar kode kredit beserta pemakaiannya.
type CreditCodeItem struct {
	ID             string   `json:"id"`
	Code           string   `json:"code"`
	AmountIDR      int64    `json:"amount_idr"`
	MaxRedemptions int      `json:"max_redemptions"`
	MaxPerUser     int      `json:"max_per_user"`
	StartsAt       TimePair `json:"starts_at"`
	EndsAt         TimePair `json:"ends_at"`
	IsActive       bool     `json:"is_active"`
	Redeemed       int64    `json:"redeemed"`
	// Status dihitung server agar tampilan tidak menebak.
	Status string `json:"status"`
}

// RedeemCreditRequest menukar kode kredit.
type RedeemCreditRequest struct {
	Code string `json:"code" binding:"required,max=40"`
}
