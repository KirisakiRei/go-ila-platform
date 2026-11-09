package model

import "time"

// Status sinyal mengikuti state machine di plans/07 dan disimpan apa adanya
// di kolom signal.status.
const (
	SignalStatusWaitHTFSweep       = "WAIT_HTF_SWEEP"
	SignalStatusWaitHTFCandleClose = "WAIT_HTF_CANDLE_CLOSE"
	SignalStatusWaitLTFCISD        = "WAIT_LTF_CISD"
	SignalStatusReadyToEntry       = "READY_TO_ENTRY"
	SignalStatusInvalidated        = "INVALIDATED"
	SignalStatusAcked              = "ACKED"
	SignalStatusConsumed           = "CONSUMED"
	SignalStatusExpired            = "EXPIRED"
	SignalStatusCancelled          = "CANCELLED"
	SignalStatusSuperseded         = "SUPERSEDED"
	SignalStatusRejected           = "REJECTED"
	// SignalStatusOpportunity menandai sinyal yang TIDAK dieksekusi otomatis
	// tetapi tetap ditampilkan sebagai peluang: user memilih untuk mengeksekusi
	// manual atau melewatinya.
	//
	// Sengaja terpisah dari REJECTED. REJECTED berarti sistem memutuskan sinyal
	// tidak boleh dijalankan (mis. akun dibekukan); OPPORTUNITY berarti sistem
	// hanya menunggu keputusan user. Mencampur keduanya membuat statistik dan
	// tampilan menyesatkan.
	SignalStatusOpportunity = "OPPORTUNITY"
	SignalStatusSkipped     = "SKIPPED"
)

// IsTerminalSignal menandai status sinyal yang tidak bisa dieksekusi lagi.
func IsTerminalSignal(status string) bool {
	switch status {
	case SignalStatusConsumed, SignalStatusExpired, SignalStatusCancelled,
		SignalStatusSuperseded, SignalStatusRejected, SignalStatusSkipped:
		return true
	default:
		return false
	}
}

// Signal adalah sinyal yang diterima dari engine.
// Sinyal bisa dibatalkan, ditolak, atau kedaluwarsa; ini bukan posisi.
type Signal struct {
	Base
	SignalID        string `gorm:"type:varchar(64);uniqueIndex;not null" json:"signal_id"`
	Symbol          string `gorm:"type:varchar(20);index;not null" json:"symbol"`
	StrategyVersion string `gorm:"type:varchar(20);not null" json:"strategy_version"`
	EngineVersion   string `gorm:"type:varchar(40)" json:"engine_version"`
	// Htf dan Ltf adalah pasangan timeframe yang menerbitkan sinyal ini, mis.
	// H4/M15. Kosong berarti engine belum mengirimkannya (kontrak menyusul),
	// sehingga konsumen wajib memperlakukannya sebagai tidak diketahui —
	// bukan menebak.
	Htf          string    `gorm:"type:varchar(10);not null;default:''" json:"htf"`
	Ltf          string    `gorm:"type:varchar(10);not null;default:''" json:"ltf"`
	Direction    string    `gorm:"type:varchar(10);not null" json:"direction"`
	EntryPrice   float64   `json:"entry_price"`
	StopLoss     float64   `json:"stop_loss"`
	TakeProfit   float64   `json:"take_profit"`
	RiskReward   float64   `json:"risk_reward"`
	Status       string    `gorm:"type:varchar(24);index;not null" json:"status"`
	Summary      string    `gorm:"type:text" json:"summary"`
	NextAction   string    `gorm:"type:text" json:"next_action"`
	PublishedAt  time.Time `json:"published_at"`
	ExpiresAt    time.Time `gorm:"index" json:"expires_at"`
	SupersededBy string    `gorm:"type:varchar(64)" json:"superseded_by"`
	CancelReason string    `gorm:"type:varchar(255)" json:"cancel_reason"`
}

// TableName menetapkan nama tabel.
func (Signal) TableName() string {
	return "signal"
}

// Status posisi. Posisi hanya berpindah dari OPEN ke salah satu status tutup;
// tidak ada transisi lain, dan engine tidak pernah bisa membatalkannya.
const (
	PositionStatusOpen         = "OPEN"
	PositionStatusClosedTP     = "CLOSED_TP"
	PositionStatusClosedSL     = "CLOSED_SL"
	PositionStatusClosedManual = "CLOSED_MANUAL"
	PositionStatusUntracked    = "UNTRACKED"
)

// Position adalah order nyata yang hidup di server broker.
// Posisi tidak pernah batal; hanya OPEN lalu CLOSED dengan alasan.
type Position struct {
	Base
	UserID          string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	BrokerAccountID string     `gorm:"type:varchar(36);index;not null" json:"broker_account_id"`
	SourceSignalID  string     `gorm:"type:varchar(64);index" json:"source_signal_id"`
	Symbol          string     `gorm:"type:varchar(20);index;not null" json:"symbol"`
	StrategyVersion string     `gorm:"type:varchar(20)" json:"strategy_version"`
	Direction       string     `gorm:"type:varchar(10);not null" json:"direction"`
	TicketNumber    string     `gorm:"type:varchar(64);index" json:"ticket_number"`
	LotSize         float64    `json:"lot_size"`
	OpenPrice       float64    `json:"open_price"`
	StopLoss        float64    `json:"stop_loss"`
	TakeProfit      float64    `json:"take_profit"`
	ExitPrice       float64    `json:"exit_price"`
	ProfitLoss      float64    `json:"profit_loss"`
	OutcomeR        float64    `json:"outcome_r"`
	RiskPercent     float64    `json:"risk_percent"`
	Status          string     `gorm:"type:varchar(24);index;not null" json:"status"`
	IsDryRun        bool       `gorm:"not null;default:false" json:"is_dry_run"`
	OpenedAt        time.Time  `json:"opened_at"`
	ClosedAt        *time.Time `json:"closed_at"`
}

// TableName menetapkan nama tabel.
func (Position) TableName() string {
	return "position"
}
