package model

import "time"

// RejectReason panjang maksimal yang ditampilkan di dashboard.
const RejectReasonMaxLen = 255

// SignalEvent menyimpan jejak peralihan status sinyal per user.
// Sinyal global boleh batal; catatan per user wajib tetap ada agar audit jelas.
type SignalEvent struct {
	ID         string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID     string    `gorm:"type:varchar(36);index:idx_signal_event_user,priority:1;not null" json:"user_id"`
	SignalID   string    `gorm:"type:varchar(64);index;not null" json:"signal_id"`
	FromStatus string    `gorm:"type:varchar(24)" json:"from_status"`
	ToStatus   string    `gorm:"type:varchar(24);not null" json:"to_status"`
	Reason     string    `gorm:"type:varchar(255)" json:"reason"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (SignalEvent) TableName() string {
	return "signal_event"
}

// UserSignal adalah status sinyal untuk satu user.
// Satu sinyal engine bisa berbeda nasibnya per user: user A sudah entry,
// user B masih menunggu, user C menolak karena kuota harian.
type UserSignal struct {
	Base
	UserID          string     `gorm:"type:varchar(36);index:idx_user_signal_unique,unique,priority:1;not null" json:"user_id"`
	SignalID        string     `gorm:"type:varchar(64);index:idx_user_signal_unique,unique,priority:2;not null" json:"signal_id"`
	BrokerAccountID string     `gorm:"type:varchar(36);index" json:"broker_account_id"`
	Status          string     `gorm:"type:varchar(24);index;not null" json:"status"`
	RejectReason    string     `gorm:"type:varchar(255)" json:"reject_reason"`
	NotBefore       *time.Time `gorm:"index" json:"not_before"`
	DecidedAt       *time.Time `json:"decided_at"`
	PositionID      string     `gorm:"type:varchar(36)" json:"position_id"`
}

// TableName menetapkan nama tabel.
func (UserSignal) TableName() string {
	return "user_signal"
}

// ExecutionRecord adalah kunci idempotensi eksekusi.
// Kunci = signal_id + akun broker; satu kombinasi hanya boleh lahir satu order.
type ExecutionRecord struct {
	ID              string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID          string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	SignalID        string     `gorm:"type:varchar(64);index:idx_execution_key,unique,priority:1;not null" json:"signal_id"`
	BrokerAccountID string     `gorm:"type:varchar(36);index:idx_execution_key,unique,priority:2;not null" json:"broker_account_id"`
	PositionID      string     `gorm:"type:varchar(36)" json:"position_id"`
	IdempotencyKey  string     `gorm:"type:varchar(120);uniqueIndex;not null" json:"idempotency_key"`
	Status          string     `gorm:"type:varchar(24);not null" json:"status"`
	FailureReason   string     `gorm:"type:varchar(255)" json:"failure_reason"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at"`
}

// TableName menetapkan nama tabel.
func (ExecutionRecord) TableName() string {
	return "execution_record"
}

// BrokerAccountState menampung telemetry koneksi dan saldo akun broker.
// Ditulis Runner, dibaca backend-api untuk dashboard.
type BrokerAccountState struct {
	BrokerAccountID string     `gorm:"primaryKey;type:varchar(36)" json:"broker_account_id"`
	UserID          string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Connected       bool       `gorm:"not null;default:false" json:"connected"`
	Balance         float64    `json:"balance"`
	Equity          float64    `json:"equity"`
	FreeMargin      float64    `json:"free_margin"`
	Currency        string     `gorm:"type:varchar(8)" json:"currency"`
	LastError       string     `gorm:"type:varchar(255)" json:"last_error"`
	LastSyncedAt    *time.Time `json:"last_synced_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (BrokerAccountState) TableName() string {
	return "broker_account_state"
}

// DailySummary adalah kuota harian per user per hari WIB.
// Diperbarui Runner setiap ada perubahan posisi.
type DailySummary struct {
	UserID      string     `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	Day         string     `gorm:"type:varchar(10);primaryKey" json:"day"`
	WinCount    int        `gorm:"not null;default:0" json:"win_count"`
	LossCount   int        `gorm:"not null;default:0" json:"loss_count"`
	TotalTrades int        `gorm:"not null;default:0" json:"total_trades"`
	LockedAt    *time.Time `json:"locked_at"`
	LockReason  string     `gorm:"type:varchar(255)" json:"lock_reason"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (DailySummary) TableName() string {
	return "daily_summary"
}

// EngineState mencatat status koneksi Runner ke engine, dipakai health check.
type EngineState struct {
	ID            string     `gorm:"primaryKey;type:varchar(40)" json:"id"`
	Connected     bool       `gorm:"not null;default:false" json:"connected"`
	EngineVersion string     `gorm:"type:varchar(40)" json:"engine_version"`
	Strategy      string     `gorm:"type:varchar(20)" json:"strategy"`
	LastError     string     `gorm:"type:varchar(255)" json:"last_error"`
	LastSeenAt    *time.Time `json:"last_seen_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (EngineState) TableName() string {
	return "engine_state"
}
