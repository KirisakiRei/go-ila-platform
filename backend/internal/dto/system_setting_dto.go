package dto

import "time"

// SystemSettingRequest adalah permintaan ubah atau kembalikan satu flag.
// `confirm` wajib berisi nama flag persis, agar perubahan berisiko tidak
// terjadi karena salah klik (plans/11 I4).
type SystemSettingRequest struct {
	Value   string `json:"value"`
	Reason  string `json:"reason" binding:"required,min=10,max=500"`
	Confirm string `json:"confirm" binding:"required,max=60"`
}

// SystemOverview merangkum kondisi platform untuk halaman admin.
type SystemOverview struct {
	Mode           MaintenanceState `json:"mode"`
	Settings       []SettingRow     `json:"settings"`
	Engines        []EngineOverview `json:"engines"`
	Runner         RunnerOverview   `json:"runner"`
	BrokerHealth   BrokerHealth     `json:"broker_health"`
	Announcements  int              `json:"announcements"`
	LastEngineSeen *time.Time       `json:"last_engine_seen,omitempty"`
	ServerTime     time.Time        `json:"server_time"`
	ServerTimePair TimePair         `json:"server_time_pair"`
}

// SettingRow adalah satu flag pada tabel pengaturan.
type SettingRow struct {
	Key       string     `json:"key"`
	Label     string     `json:"label"`
	Help      string     `json:"help"`
	Kind      string     `json:"kind"`
	Value     string     `json:"value"`
	Source    string     `json:"source"`
	Danger    bool       `json:"danger"`
	Min       int        `json:"min,omitempty"`
	Max       int        `json:"max,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	UpdatedBy string     `json:"updated_by,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// EngineOverview adalah ringkasan satu engine.
type EngineOverview struct {
	Strategy      string     `json:"strategy"`
	Connected     bool       `json:"connected"`
	EngineVersion string     `json:"engine_version,omitempty"`
	MarketState   string     `json:"market_state,omitempty"`
	LastSeenAt    *time.Time `json:"last_seen_at,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	Stale         bool       `json:"stale"`
}

// RunnerOverview merangkum laporan terakhir Runner.
type RunnerOverview struct {
	Connected bool       `json:"connected"`
	Version   string     `json:"version,omitempty"`
	LastSeen  *time.Time `json:"last_seen,omitempty"`
	Stale     bool       `json:"stale"`
}

// BrokerHealth merangkum akun broker yang sedang bermasalah.
type BrokerHealth struct {
	Total      int `json:"total"`
	Connected  int `json:"connected"`
	ErrorCount int `json:"error_count"`
	Frozen     int `json:"frozen"`
}

// MaintenanceState adalah mode platform yang berlaku.
type MaintenanceState struct {
	Mode        string     `json:"mode"`
	Label       string     `json:"label"`
	Reason      string     `json:"reason"`
	ExpectedEnd *time.Time `json:"expected_end,omitempty"`
	ChangedAt   time.Time  `json:"changed_at"`
	ChangedBy   string     `json:"changed_by,omitempty"`
}

// MaintenanceRequest adalah permintaan ubah mode platform.
// Konfirmasi untuk mode selain normal wajib mengetik "MAINTENANCE".
type MaintenanceRequest struct {
	Mode        string `json:"mode" binding:"required,oneof=normal read-only maintenance"`
	Reason      string `json:"reason" binding:"required,min=10,max=255"`
	ExpectedEnd string `json:"expected_end,omitempty"`
	Confirm     string `json:"confirm,omitempty"`
}

// PublicStatus adalah status publik untuk ditampilkan di halaman masuk dan
// dashboard pelanggan. Tidak memuat detail internal apa pun.
type PublicStatus struct {
	Mode        string     `json:"mode"`
	Message     string     `json:"message"`
	ExpectedEnd *time.Time `json:"expected_end,omitempty"`
	ServerTime  time.Time  `json:"server_time"`
	TimePair    TimePair   `json:"time_pair"`
}
