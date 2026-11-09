package dto

// AdminSignalFilter menyaring daftar sinyal admin.
// Rentang tanggal wajib agar tabel sinyal tidak dipindai seluruhnya.
type AdminSignalFilter struct {
	Page     int    `form:"page"`
	PerPage  int    `form:"per_page"`
	Query    string `form:"q"`
	Sort     string `form:"sort"`
	Order    string `form:"order"`
	From     string `form:"from"`
	To       string `form:"to"`
	Symbol   string `form:"symbol"`
	Status   string `form:"status"`
	Strategy string `form:"strategy"`
}

// AdminSignalItem adalah satu sinyal dari sudut pandang admin.
type AdminSignalItem struct {
	SignalID        string   `json:"signal_id"`
	Symbol          string   `json:"symbol"`
	StrategyVersion string   `json:"strategy_version"`
	EngineVersion   string   `json:"engine_version"`
	Direction       string   `json:"direction"`
	EntryPrice      float64  `json:"entry_price"`
	StopLoss        float64  `json:"stop_loss"`
	TakeProfit      float64  `json:"take_profit"`
	RiskReward      float64  `json:"risk_reward"`
	Status          string   `json:"status"`
	Summary         string   `json:"summary"`
	NextAction      string   `json:"next_action"`
	PublishedAt     TimePair `json:"published_at"`
	ExpiresAt       TimePair `json:"expires_at"`
	CancelReason    string   `json:"cancel_reason,omitempty"`
	// Recipients adalah jumlah user yang menerima sinyal ini.
	Recipients int64 `json:"recipients"`
	// Executed adalah jumlah user yang benar-benar entry.
	Executed int64 `json:"executed"`
}

// AdminSignalRecipient adalah nasib sinyal pada satu user.
type AdminSignalRecipient struct {
	UserID          string    `json:"user_id"`
	UserName        string    `json:"user_name"`
	UserEmail       string    `json:"user_email"`
	BrokerAccountID string    `json:"broker_account_id,omitempty"`
	Status          string    `json:"status"`
	RejectReason    string    `json:"reject_reason,omitempty"`
	PositionID      string    `json:"position_id,omitempty"`
	DecidedAt       *TimePair `json:"decided_at,omitempty"`
	CreatedAt       TimePair  `json:"created_at"`
}

// AdminSignalEvent adalah satu perubahan status sinyal.
type AdminSignalEvent struct {
	UserID     string   `json:"user_id"`
	FromStatus string   `json:"from_status,omitempty"`
	ToStatus   string   `json:"to_status"`
	Reason     string   `json:"reason,omitempty"`
	CreatedAt  TimePair `json:"created_at"`
}

// AdminSignalExecution adalah klaim eksekusi satu sinyal pada satu akun.
type AdminSignalExecution struct {
	UserID          string   `json:"user_id"`
	BrokerAccountID string   `json:"broker_account_id"`
	Status          string   `json:"status"`
	CreatedAt       TimePair `json:"created_at"`
}

// AdminSignalTrace adalah perjalanan lengkap satu sinyal.
type AdminSignalTrace struct {
	Signal     AdminSignalItem        `json:"signal"`
	Recipients []AdminSignalRecipient `json:"recipients"`
	Events     []AdminSignalEvent     `json:"events"`
	Executions []AdminSignalExecution `json:"executions"`
	Positions  []AdminPositionItem    `json:"positions"`
}
