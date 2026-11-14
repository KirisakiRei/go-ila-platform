package model

import "time"

// Status perjalanan satu backtest.
//
// Antrean dipakai karena satu run bisa memakan puluhan detik. Bila dikerjakan
// langsung di dalam permintaan HTTP, koneksi akan menunggu dan pengguna melihat
// halaman yang menggantung — padahal prosesnya baik-baik saja.
const (
	BacktestStatusQueued  = "queued"
	BacktestStatusRunning = "running"
	BacktestStatusDone    = "done"
	BacktestStatusFailed  = "failed"
)

// BacktestRun adalah satu kali backtest dijalankan.
//
// Statistik disimpan sebagai kolom, bukan sebagai satu blok JSON, supaya riwayat
// bisa diurutkan dan disaring langsung oleh database tanpa membaca seluruh isi.
type BacktestRun struct {
	Base

	UserID string `gorm:"type:varchar(36);index;not null" json:"user_id"`

	Symbol string `gorm:"type:varchar(20);not null" json:"symbol"`
	HTF    string `gorm:"type:varchar(10);not null" json:"htf"`
	LTF    string `gorm:"type:varchar(10);not null" json:"ltf"`

	// RangeFrom dan RangeTo adalah rentang yang benar-benar dievaluasi, sudah
	// dipangkas ke cakupan data yang tersedia. Ini bukan rentang yang diminta
	// pengguna, karena keduanya sering berbeda.
	RangeFrom time.Time `gorm:"not null" json:"range_from"`
	RangeTo   time.Time `gorm:"not null" json:"range_to"`

	Equity      float64 `gorm:"type:decimal(18,2);not null" json:"equity"`
	RiskPercent float64 `gorm:"type:decimal(6,3);not null" json:"risk_percent"`

	// Identitas aturan yang dipakai. Tanpa ini, hasil lama tidak bisa
	// dibandingkan dengan hasil baru setelah aturan berubah.
	Strategy    string `gorm:"type:varchar(20);not null" json:"strategy"`
	StrategyVer string `gorm:"type:varchar(40);not null" json:"strategy_version"`
	CodeCommit  string `gorm:"type:varchar(40)" json:"code_commit"`

	Status   string `gorm:"type:varchar(20);index;not null" json:"status"`
	ErrorMsg string `gorm:"type:text" json:"error_message,omitempty"`

	TotalTrades     int `gorm:"not null;default:0" json:"total_trades"`
	Wins            int `gorm:"not null;default:0" json:"wins"`
	Losses          int `gorm:"not null;default:0" json:"losses"`
	Expired         int `gorm:"not null;default:0" json:"expired"`
	AmbiguousExits  int `gorm:"not null;default:0" json:"ambiguous_exits"`
	SignalsRejected int `gorm:"not null;default:0" json:"signals_rejected"`

	WinRate      float64 `gorm:"type:decimal(6,4);not null;default:0" json:"win_rate"`
	TotalR       float64 `gorm:"type:decimal(12,4);not null;default:0" json:"total_r"`
	MaxDrawdown  float64 `gorm:"type:decimal(6,4);not null;default:0" json:"max_drawdown"`
	ProfitFactor float64 `gorm:"type:decimal(10,4);not null;default:0" json:"profit_factor"`

	// EquityCurveJSON menyimpan titik-titik kurva ekuitas. Ukurannya kecil
	// (satu titik per trade), jadi aman disimpan bersama barisnya.
	EquityCurveJSON string `gorm:"type:text" json:"-"`

	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

// TableName menetapkan nama tabel secara eksplisit.
func (BacktestRun) TableName() string { return "backtest_runs" }

// BacktestTrade adalah satu entry di dalam sebuah run.
type BacktestTrade struct {
	Base

	RunID string `gorm:"type:varchar(36);index;not null" json:"run_id"`
	// Seq adalah nomor urut mulai dari satu, dipakai UI untuk menunjuk entry.
	Seq int `gorm:"not null" json:"seq"`

	Direction string    `gorm:"type:varchar(5);not null" json:"direction"`
	EntryTime time.Time `gorm:"index" json:"entry_time"`
	ExitTime  time.Time `json:"exit_time"`

	EntryPrice float64 `gorm:"type:decimal(16,5);not null" json:"entry_price"`
	StopLoss   float64 `gorm:"type:decimal(16,5);not null" json:"stop_loss"`
	TakeProfit float64 `gorm:"type:decimal(16,5);not null" json:"take_profit"`
	PlannedRR  float64 `gorm:"type:decimal(8,4);not null" json:"planned_rr"`

	Outcome  string  `gorm:"type:varchar(10);not null" json:"outcome"`
	ActualR  float64 `gorm:"type:decimal(10,4);not null" json:"actual_r"`
	Duration int     `gorm:"not null;default:0" json:"duration_seconds"`

	EquityAfter float64 `gorm:"type:decimal(18,2);not null" json:"equity_after"`
	// Ambiguous menandai candle yang menyentuh TP dan SL sekaligus.
	Ambiguous bool `gorm:"not null;default:false" json:"ambiguous"`

	Summary    string `gorm:"type:text" json:"summary"`
	NextAction string `gorm:"type:text" json:"next_action"`

	// CandlesJSON menyimpan potongan candle di sekitar entry untuk menggambar
	// chart. Isinya berat, jadi dihapus oleh sweeper setelah beberapa hari;
	// statistiknya tetap tersimpan.
	CandlesJSON string `gorm:"type:text" json:"-"`
	// LevelsJSON menyimpan koordinat FVG, sweep, dan CISD.
	LevelsJSON string `gorm:"type:text" json:"-"`
	// CandlesPurgedAt menandai kapan candle dihapus, agar UI bisa menjelaskan
	// kenapa chart tidak tersedia lagi alih-alih menampilkan kotak kosong.
	CandlesPurgedAt *time.Time `json:"candles_purged_at,omitempty"`
}

// TableName menetapkan nama tabel secara eksplisit.
func (BacktestTrade) TableName() string { return "backtest_trades" }

// BacktestQuota mencatat pemakaian kuota harian per pengguna.
//
// Kuota dihitung per hari waktu lokal (WIB), bukan per 24 jam bergulir, supaya
// batasnya bisa diprediksi pengguna.
type BacktestQuota struct {
	UserID string `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	// Tanggal disimpan sebagai string YYYY-MM-DD agar perbandingan tetap
	// sederhana di semua basis data.
	Tanggal   string    `gorm:"type:varchar(10);primaryKey" json:"tanggal"`
	Terpakai  int       `gorm:"not null;default:0" json:"terpakai"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel secara eksplisit.
func (BacktestQuota) TableName() string { return "backtest_quotas" }
