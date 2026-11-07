package dto

// UserListItem adalah satu baris di daftar pelanggan admin.
// Hanya memuat data tingkat 1: tanpa saldo dan tanpa kredensial.
type UserListItem struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Tier           string    `json:"tier"`
	Status         string    `json:"status"`
	Role           string    `json:"role"`
	RoleCode       string    `json:"role_code,omitempty"`
	TelegramLinked bool      `json:"telegram_linked"`
	AccountCount   int64     `json:"account_count"`
	CreatedAt      TimePair  `json:"created_at"`
	LastLoginAt    *TimePair `json:"last_login_at,omitempty"`
	TierExpiresAt  *TimePair `json:"tier_expires_at,omitempty"`
}

// UserDetail adalah profil lengkap satu pelanggan untuk halaman admin.
type UserDetail struct {
	UserListItem
	TrialUsed     bool      `json:"trial_used"`
	TrialExpires  *TimePair `json:"trial_expires_at,omitempty"`
	GraceUntil    *TimePair `json:"grace_until,omitempty"`
	SuspendedAt   *TimePair `json:"suspended_at,omitempty"`
	BannedAt      *TimePair `json:"banned_at,omitempty"`
	BanReason     string    `json:"ban_reason,omitempty"`
	BanCategory   string    `json:"ban_category,omitempty"`
	InternalNote  string    `json:"internal_note,omitempty"`
	IdleWarning   bool      `json:"idle_warning"`
	NotifyEnabled bool      `json:"notify_enabled"`
	TwoFactorOn   bool      `json:"two_factor_enabled"`
	// Subscription ringkas agar admin tahu sumber tier: langganan atau manual.
	Subscription *AdminSubscriptionBrief `json:"subscription,omitempty"`
}

// AdminSubscriptionBrief adalah ringkasan langganan aktif.
type AdminSubscriptionBrief struct {
	PlanCode    string   `json:"plan_code"`
	Status      string   `json:"status"`
	PeriodStart TimePair `json:"period_start"`
	PeriodEnd   TimePair `json:"period_end"`
	CancelAtEnd bool     `json:"cancel_at_period_end"`
}

// AdminAccountItem adalah akun broker dari sudut pandang admin.
// Kredensial TIDAK PERNAH ada di sini; nilai tersembunyi hanya penanda.
type AdminAccountItem struct {
	ID                string    `json:"id"`
	Label             string    `json:"label"`
	BrokerType        string    `json:"broker_type"`
	ExternalAccountID string    `json:"external_account_id"`
	Mode              string    `json:"mode"`
	RiskMode          string    `json:"risk_mode"`
	RiskPercent       float64   `json:"risk_percent"`
	IsActive          bool      `json:"is_active"`
	IsFrozen          bool      `json:"is_frozen"`
	IsDryRun          bool      `json:"is_dry_run"`
	FrozenAt          *TimePair `json:"frozen_at,omitempty"`
	FrozenReason      string    `json:"frozen_reason,omitempty"`
	LastVerifiedAt    *TimePair `json:"last_verified_at,omitempty"`
	LastError         string    `json:"last_error,omitempty"`
	// Balance hanya terisi bila admin memegang account.balance DAN aksesnya
	// tercatat di audit (plans/07 §2 tingkat 3).
	Balance *AdminAccountBalance `json:"balance,omitempty"`
}

// AdminAccountBalance adalah saldo dari telemetry Runner, bukan dari broker.
type AdminAccountBalance struct {
	Balance    float64  `json:"balance"`
	Equity     float64  `json:"equity"`
	Currency   string   `json:"currency"`
	ObservedAt TimePair `json:"observed_at"`
}

// AdminPositionItem adalah satu posisi dalam riwayat pelanggan.
type AdminPositionItem struct {
	ID              string    `json:"id"`
	Symbol          string    `json:"symbol"`
	Direction       string    `json:"direction"`
	StrategyVersion string    `json:"strategy_version"`
	Status          string    `json:"status"`
	OpenPrice       float64   `json:"open_price"`
	ExitPrice       float64   `json:"exit_price"`
	ProfitLoss      float64   `json:"profit_loss"`
	OutcomeR        float64   `json:"outcome_r"`
	IsDryRun        bool      `json:"is_dry_run"`
	OpenedAt        TimePair  `json:"opened_at"`
	ClosedAt        *TimePair `json:"closed_at,omitempty"`
}

// AdminNoteItem adalah catatan internal beserta penulisnya.
type AdminNoteItem struct {
	ID         string   `json:"id"`
	AuthorID   string   `json:"author_id"`
	AuthorName string   `json:"author_name"`
	Body       string   `json:"body"`
	CreatedAt  TimePair `json:"created_at"`
}

// AdminLegalAcceptance adalah bukti persetujuan dokumen oleh user.
type AdminLegalAcceptance struct {
	Slug       string   `json:"slug"`
	Title      string   `json:"title"`
	Version    string   `json:"version"`
	AcceptedAt TimePair `json:"accepted_at"`
	IPHash     string   `json:"ip_hash"`
}

// UserActionRequest adalah permintaan aksi terhadap pengguna.
// Konfirmasi berlapis dilakukan di klien, tetapi server tetap memeriksa nilai
// yang wajib agar tidak bisa dilewati (plans/07 §4.4).
type UserActionRequest struct {
	Reason   string `json:"reason" binding:"required,max=500"`
	Category string `json:"category" binding:"omitempty,max=40"`
	// ConfirmEmail wajib sama persis dengan email akun untuk ban dan hapus.
	ConfirmEmail string `json:"confirm_email" binding:"omitempty,max=150"`
	// Acknowledge wajib benar untuk hapus akun.
	Acknowledge bool `json:"acknowledge"`
}

// UserTierRequest mengubah tier manual.
type UserTierRequest struct {
	Tier   string `json:"tier" binding:"required,oneof=free basic plus"`
	Reason string `json:"reason" binding:"required,max=500"`
	// Days mengisi masa berlaku bila tier bukan free. 0 berarti tanpa batas.
	Days int `json:"days" binding:"omitempty,min=0,max=3650"`
}

// UserNoteRequest menambah catatan internal.
type UserNoteRequest struct {
	Body string `json:"body" binding:"required,max=2000"`
}

// AccountFreezeRequest membekukan atau membuka pembekuan akun broker.
type AccountFreezeRequest struct {
	Reason string `json:"reason" binding:"required,max=500"`
}
