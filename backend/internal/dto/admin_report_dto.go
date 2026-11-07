package dto

// RevenueDay adalah pendapatan satu hari.
type RevenueDay struct {
	Date      string `json:"date"`
	PaidIDR   int64  `json:"paid_idr"`
	RefundIDR int64  `json:"refund_idr"`
	Count     int64  `json:"count"`
}

// RevenueReport adalah laporan pendapatan pada satu rentang.
type RevenueReport struct {
	From      TimePair     `json:"from"`
	To        TimePair     `json:"to"`
	GrossIDR  int64        `json:"gross_idr"`
	RefundIDR int64        `json:"refund_idr"`
	NetIDR    int64        `json:"net_idr"`
	Daily     []RevenueDay `json:"daily"`
}

// CustomerReport adalah ringkasan pelanggan.
type CustomerReport struct {
	From         TimePair `json:"from"`
	To           TimePair `json:"to"`
	NewUsers     int64    `json:"new_users"`
	PaidUsers    int64    `json:"paid_users"`
	Canceled     int64    `json:"canceled"`
	TrialStarted int64    `json:"trial_started"`
}

// CouponUsageItem adalah pemakaian satu kupon.
type CouponUsageItem struct {
	Code        string `json:"code"`
	Uses        int64  `json:"uses"`
	DiscountIDR int64  `json:"discount_idr"`
}

// CouponReport adalah laporan pemakaian kupon.
type CouponReport struct {
	From  TimePair          `json:"from"`
	To    TimePair          `json:"to"`
	Items []CouponUsageItem `json:"items"`
}

// OperationReport adalah ringkasan operasional: eksekusi dan notifikasi.
type OperationReport struct {
	From                TimePair `json:"from"`
	To                  TimePair `json:"to"`
	PositionsOpened     int64    `json:"positions_opened"`
	PositionsClosed     int64    `json:"positions_closed"`
	PositionsDryRun     int64    `json:"positions_dry_run"`
	NotificationsSent   int64    `json:"notifications_sent"`
	NotificationsFailed int64    `json:"notifications_failed"`
}
