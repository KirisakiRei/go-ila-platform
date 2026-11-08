package dto

// CheckoutRequest adalah permintaan membuat tagihan.
type CheckoutRequest struct {
	PlanCode   string `json:"plan_code" binding:"required,max=20"`
	CouponCode string `json:"coupon_code" binding:"omitempty,max=40"`
	UseCredit  bool   `json:"use_credit"`
}

// CheckoutResult adalah tautan pembayaran yang dikembalikan ke pengguna.
type CheckoutResult struct {
	InvoiceNumber string   `json:"invoice_number"`
	TotalIDR      int64    `json:"total_idr"`
	PaymentURL    string   `json:"payment_url,omitempty"`
	ExpiresAt     TimePair `json:"expires_at"`
	Status        string   `json:"status"`
}

// InvoiceItem adalah satu tagihan di riwayat pengguna.
type InvoiceItem struct {
	ID          string    `json:"id"`
	Number      string    `json:"number"`
	PlanCode    string    `json:"plan_code"`
	AmountIDR   int64     `json:"amount_idr"`
	DiscountIDR int64     `json:"discount_idr"`
	CreditIDR   int64     `json:"credit_idr"`
	TotalIDR    int64     `json:"total_idr"`
	Status      string    `json:"status"`
	CouponCode  string    `json:"coupon_code,omitempty"`
	CreatedAt   TimePair  `json:"created_at"`
	PaidAt      *TimePair `json:"paid_at,omitempty"`
	ExpiresAt   TimePair  `json:"expires_at"`
}

// SubscriptionItem adalah langganan aktif pengguna.
type SubscriptionItem struct {
	PlanCode    string   `json:"plan_code"`
	Status      string   `json:"status"`
	PeriodStart TimePair `json:"period_start"`
	PeriodEnd   TimePair `json:"period_end"`
	CancelAtEnd bool     `json:"cancel_at_period_end"`
}

// ManualPaymentRequest mencatat pembayaran manual oleh admin.
type ManualPaymentRequest struct {
	AmountIDR int64  `json:"amount_idr" binding:"required,min=1"`
	Note      string `json:"note" binding:"required,max=500"`
}

// RefundRequest mencatat refund manual.
type RefundRequest struct {
	PaymentID string `json:"payment_id" binding:"max=36"`
	AmountIDR int64  `json:"amount_idr" binding:"required,min=1"`
	Reason    string `json:"reason" binding:"required,max=500"`
	// GatewayRef adalah nomor bukti transfer balik dari gateway/bank.
	GatewayRef string `json:"gateway_ref" binding:"max=120"`
}

// AdminInvoiceItem adalah tagihan sebagaimana dilihat admin.
type AdminInvoiceItem struct {
	ID          string    `json:"id"`
	Number      string    `json:"number"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email"`
	PlanCode    string    `json:"plan_code"`
	AmountIDR   int64     `json:"amount_idr"`
	DiscountIDR int64     `json:"discount_idr"`
	CreditIDR   int64     `json:"credit_idr"`
	TotalIDR    int64     `json:"total_idr"`
	Status      string    `json:"status"`
	CouponCode  string    `json:"coupon_code,omitempty"`
	Gateway     string    `json:"gateway,omitempty"`
	PeriodStart TimePair  `json:"period_start"`
	PeriodEnd   TimePair  `json:"period_end"`
	CreatedAt   TimePair  `json:"created_at"`
	PaidAt      *TimePair `json:"paid_at,omitempty"`
	ExpiresAt   TimePair  `json:"expires_at"`
}
