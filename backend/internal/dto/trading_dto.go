package dto

import "time"

// SignalResponse adalah satu sinyal seperti yang dilihat user.
// Untuk tier Free, sinyal baru muncul setelah delay 15 menit.
type SignalResponse struct {
	SignalID        string    `json:"signal_id"`
	Symbol          string    `json:"symbol"`
	StrategyVersion string    `json:"strategy_version"`
	EngineVersion   string    `json:"engine_version"`
	Direction       string    `json:"direction"`
	EntryPrice      float64   `json:"entry_price"`
	StopLoss        float64   `json:"stop_loss"`
	TakeProfit      float64   `json:"take_profit"`
	RiskReward      float64   `json:"risk_reward"`
	Status          string    `json:"status"`
	Summary         string    `json:"summary,omitempty"`
	NextAction      string    `json:"next_action,omitempty"`
	RejectReason    string    `json:"reject_reason,omitempty"`
	PublishedAt     time.Time `json:"published_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	CanConfirm      bool      `json:"can_confirm"`
	Delayed         bool      `json:"delayed"`
}

// PositionResponse adalah posisi hidup atau tertutup milik user.
type PositionResponse struct {
	ID              string     `json:"id"`
	Symbol          string     `json:"symbol"`
	StrategyVersion string     `json:"strategy_version"`
	Direction       string     `json:"direction"`
	TicketNumber    string     `json:"ticket_number"`
	LotSize         float64    `json:"lot_size"`
	OpenPrice       float64    `json:"open_price"`
	StopLoss        float64    `json:"stop_loss"`
	TakeProfit      float64    `json:"take_profit"`
	ExitPrice       float64    `json:"exit_price"`
	ProfitLoss      float64    `json:"profit_loss"`
	OutcomeR        float64    `json:"outcome_r"`
	RiskPercent     float64    `json:"risk_percent"`
	Status          string     `json:"status"`
	IsDryRun        bool       `json:"is_dry_run"`
	OpenedAt        time.Time  `json:"opened_at"`
	OpenedAtPair    TimePair   `json:"opened_at_pair"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	ClosedAtPair    *TimePair  `json:"closed_at_pair,omitempty"`
	// PnLKnown=false berarti hasil belum terbaca dari broker. Antarmuka wajib
	// menampilkan "—", bukan US$0,00 yang menyesatkan (plans/19 C9).
	PnLKnown bool `json:"pnl_known"`
}

// DailySummaryResponse menampilkan sisa kuota dan alasan lock hari ini.
type DailySummaryResponse struct {
	Day         string `json:"day"`
	WinCount    int    `json:"win_count"`
	LossCount   int    `json:"loss_count"`
	TotalTrades int    `json:"total_trades"`
	MaxTrades   int    `json:"max_trades"`
	Locked      bool   `json:"locked"`
	LockReason  string `json:"lock_reason,omitempty"`
}
