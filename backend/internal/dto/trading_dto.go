package dto

import "time"

// SignalCandle adalah satu lilin yang menyertai rincian sweep/CISD.
//
// Chart memakai ini untuk menempatkan garis dan panah tepat pada lilin yang
// dimaksud engine, tanpa perlu menebak dari waktu.
type SignalCandle struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	IsClosed  bool      `json:"is_closed"`
}

// SignalFairValueGap adalah area FVG yang terbentuk pada sweep HTF.
type SignalFairValueGap struct {
	Type      string     `json:"type,omitempty"`
	Top       float64    `json:"top"`
	Bottom    float64    `json:"bottom"`
	Candle3At *time.Time `json:"candle3_time,omitempty"`
	Mitigated bool       `json:"mitigated"`
}

// SignalHtfSweep adalah rincian sweep timeframe besar untuk tampilan radar.
type SignalHtfSweep struct {
	Detected        bool                `json:"detected"`
	SweepInProgress bool                `json:"sweep_in_progress"`
	Direction       string              `json:"direction"`
	FVG             *SignalFairValueGap `json:"fvg,omitempty"`
	SweepCandle     *SignalCandle       `json:"sweep_candle,omitempty"`
	SweptCandle     *SignalCandle       `json:"swept_candle,omitempty"`
	StopLoss        float64             `json:"stop_loss"`
	TargetDOL       float64             `json:"target_dol"`
	Details         string              `json:"details"`
	SweepTime       *time.Time          `json:"sweep_time,omitempty"`
}

// SignalCISD adalah rincian konfirmasi momentum timeframe kecil untuk tampilan radar.
type SignalCISD struct {
	Detected    bool          `json:"detected"`
	Direction   string        `json:"direction"`
	RefCandle   *SignalCandle `json:"ref_candle,omitempty"`
	BreakCandle *SignalCandle `json:"break_candle,omitempty"`
	CISDLevel   float64       `json:"cisd_level"`
	ExtremeSL   float64       `json:"extreme_sl"`
	EntryPrice  float64       `json:"entry_price"`
	Details     string        `json:"details"`
	BreakTime   *time.Time    `json:"break_time,omitempty"`
}

// SignalResponse adalah satu sinyal seperti yang dilihat user.
// Untuk tier Free, sinyal baru muncul setelah delay 15 menit.
type SignalResponse struct {
	SignalID        string `json:"signal_id"`
	Symbol          string `json:"symbol"`
	StrategyVersion string `json:"strategy_version"`
	EngineVersion   string `json:"engine_version"`
	// Htf/Ltf adalah pasangan timeframe sumber sinyal. Kosong bila engine belum
	// mengirimkannya; UI tidak boleh menampilkan apa pun dalam kasus itu.
	Htf string `json:"htf"`
	Ltf string `json:"ltf"`
	// FeedSource adalah sumber candle yang dipakai engine saat sinyal ini
	// dihitung, mis. "FOREXCOM". Dashboard memakai nilai ini sebagai label
	// broker pada chart.
	FeedSource   string          `json:"feed_source,omitempty"`
	Direction    string          `json:"direction"`
	EntryPrice   float64         `json:"entry_price"`
	StopLoss     float64         `json:"stop_loss"`
	TakeProfit   float64         `json:"take_profit"`
	RiskReward   float64         `json:"risk_reward"`
	Status       string          `json:"status"`
	Summary      string          `json:"summary,omitempty"`
	NextAction   string          `json:"next_action,omitempty"`
	RejectReason string          `json:"reject_reason,omitempty"`
	HtfSweep     *SignalHtfSweep `json:"htf_sweep,omitempty"`
	Cisd         *SignalCISD     `json:"cisd,omitempty"`
	PublishedAt  time.Time       `json:"published_at"`
	// UpdatedAt adalah kapan baris sinyal terakhir diperbarui (mis. ringkasan
	// dan masa berlaku disegarkan engine). UI memakainya untuk label "Update"
	// agar tidak tampak macet pada waktu terbit pertama.
	UpdatedAt  time.Time `json:"updated_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	CanConfirm bool      `json:"can_confirm"`
	Delayed    bool      `json:"delayed"`
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
