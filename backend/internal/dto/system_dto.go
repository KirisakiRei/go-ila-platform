package dto

import "time"

// SystemStatus adalah keadaan platform yang ditampilkan di header dashboard
// (plans/06 C9). Semua nilainya berasal dari laporan Runner, bukan tebakan.
type SystemStatus struct {
	Market     MarketStatus   `json:"market"`
	Engines    []EngineStatus `json:"engines"`
	Brokers    BrokerStatus   `json:"brokers"`
	Daily      DailyResponse  `json:"daily"`
	ServerTime time.Time      `json:"server_time"`
	// ServerTimePair menyajikan waktu server dalam WIB dan NY sekaligus.
	ServerTimePair TimePair `json:"server_time_pair"`
	DailyTimezone  string   `json:"daily_timezone"`
}

// MarketStatus adalah status sesi pasar menurut Engine.
type MarketStatus struct {
	State     string     `json:"state"` // OPEN | CLOSED | ROLLOVER | UNKNOWN
	Label     string     `json:"label"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// EngineStatus adalah kondisi satu proses engine.
type EngineStatus struct {
	Strategy      string     `json:"strategy"`
	Connected     bool       `json:"connected"`
	EngineVersion string     `json:"engine_version,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	LastSeenAt    *time.Time `json:"last_seen_at,omitempty"`
}

// BrokerStatus merangkum kesehatan akun broker user.
type BrokerStatus struct {
	Total     int  `json:"total"`
	Connected int  `json:"connected"`
	AnyError  bool `json:"any_error"`
}

// DailyResponse adalah ringkasan kuota hari ini dalam zona WIB.
type DailyResponse struct {
	Day string `json:"day"`
	// DayPair adalah tengah malam hari akuntansi dalam WIB dan NY.
	DayPair     TimePair   `json:"day_pair"`
	WinCount    int        `json:"win_count"`
	LossCount   int        `json:"loss_count"`
	TotalTrades int        `json:"total_trades"`
	MaxTrades   int        `json:"max_trades"`
	OpenCount   int        `json:"open_count"`
	Locked      bool       `json:"locked"`
	LockReason  string     `json:"lock_reason,omitempty"`
	LockedAt    *time.Time `json:"locked_at,omitempty"`
}
