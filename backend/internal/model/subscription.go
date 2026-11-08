package model

import "time"

// Status langganan. Masa tenggang adalah keadaan normal pada gateway lokal
// yang tidak punya auto-debit (plans/08 §7).
const (
	SubStatusActive   = "active"
	SubStatusGrace    = "grace"
	SubStatusPastDue  = "past_due"
	SubStatusCanceled = "canceled"
)

// Status invoice.
const (
	InvoicePending   = "pending"
	InvoicePaid      = "paid"
	InvoiceExpired   = "expired"
	InvoiceCancelled = "cancelled"
	InvoiceRefunded  = "refunded"
)

// SubscriptionRecord mencatat periode langganan satu pengguna.
// Terpisah dari invoice karena invoice adalah bukti uang, langganan adalah
// jangka waktu akses (plans/08 §2.1).
type SubscriptionRecord struct {
	ID                 string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID             string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	PlanCode           string     `gorm:"type:varchar(20);not null" json:"plan_code"`
	Status             string     `gorm:"type:varchar(20);not null" json:"status"`
	CurrentPeriodStart time.Time  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `gorm:"index" json:"current_period_end"`
	CancelAtPeriodEnd  bool       `gorm:"not null;default:false" json:"cancel_at_period_end"`
	CanceledAt         *time.Time `json:"canceled_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (SubscriptionRecord) TableName() string {
	return "subscription_record"
}

// Invoice adalah bukti tagihan. Nilai harga disalin saat terbit sehingga
// perubahan harga berikutnya tidak mengubah dokumen lama.
type Invoice struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Number      string     `gorm:"type:varchar(30);uniqueIndex;not null" json:"number"`
	UserID      string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	PlanCode    string     `gorm:"type:varchar(20);not null" json:"plan_code"`
	AmountIDR   int64      `gorm:"not null;column:amount_idr" json:"amount_idr"`
	DiscountIDR int64      `gorm:"not null;default:0;column:discount_idr" json:"discount_idr"`
	CreditIDR   int64      `gorm:"not null;default:0;column:credit_idr" json:"credit_idr"`
	TaxRate     float64    `gorm:"type:numeric(5,2);not null;default:0" json:"tax_rate"`
	TaxIDR      int64      `gorm:"not null;default:0;column:tax_idr" json:"tax_idr"`
	TotalIDR    int64      `gorm:"not null;column:total_idr" json:"total_idr"`
	Status      string     `gorm:"type:varchar(20);index;not null" json:"status"`
	CouponCode  string     `gorm:"type:varchar(40)" json:"coupon_code"`
	PeriodStart time.Time  `json:"period_start"`
	PeriodEnd   time.Time  `json:"period_end"`
	ExpiresAt   time.Time  `json:"expires_at"`
	PaidAt      *time.Time `json:"paid_at"`
	Gateway     string     `gorm:"type:varchar(30);not null;default:''" json:"gateway"`
	GatewayRef  string     `gorm:"type:varchar(120);not null;default:''" json:"gateway_ref"`
	IssuerName  string     `gorm:"type:varchar(160);not null;default:''" json:"issuer_name"`
	IssuerTaxID string     `gorm:"type:varchar(40);not null;default:''" json:"issuer_tax_id"`
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (Invoice) TableName() string {
	return "invoice"
}

// Payment adalah satu percobaan bayar yang diterima dari gateway.
// Unik (gateway, gateway_txn_id) adalah inti idempotensi: webhook yang sama
// tidak boleh menambah periode dua kali (plans/12 §5.10).
type Payment struct {
	ID        string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InvoiceID string `gorm:"type:varchar(36);index;not null" json:"invoice_id"`
	Gateway   string `gorm:"type:varchar(30);not null;uniqueIndex:uq_payment_txn" json:"gateway"`
	// Idempotensi webhook bergantung pada indeks gabungan ini: ON CONFLICT
	// (gateway, gateway_txn_id) menolak pencatatan ganda dari gateway yang sama.
	GatewayTxnID   string    `gorm:"type:varchar(120);not null;uniqueIndex:uq_payment_txn" json:"gateway_txn_id"`
	AmountIDR      int64     `gorm:"not null;column:amount_idr" json:"amount_idr"`
	FeeIDR         int64     `gorm:"not null;default:0;column:fee_idr" json:"fee_idr"`
	Status         string    `gorm:"type:varchar(20);not null" json:"status"`
	SignatureValid bool      `gorm:"not null;default:false" json:"signature_valid"`
	RawPayload     string    `gorm:"type:text" json:"-"`
	ReceivedAt     time.Time `json:"received_at"`
}

// TableName menetapkan nama tabel.
func (Payment) TableName() string {
	return "payment"
}

// Refund selalu manual: tidak ada pengembalian otomatis ke kartu.
type Refund struct {
	ID         string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	PaymentID  string    `gorm:"type:varchar(36);index;not null" json:"payment_id"`
	AmountIDR  int64     `gorm:"not null;column:amount_idr" json:"amount_idr"`
	Reason     string    `gorm:"type:varchar(500);not null" json:"reason"`
	ActorID    string    `gorm:"type:varchar(36);not null" json:"actor_id"`
	GatewayRef string    `gorm:"type:varchar(120);not null;default:''" json:"gateway_ref"`
	Status     string    `gorm:"type:varchar(20);not null;default:pending" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (Refund) TableName() string {
	return "refund"
}
