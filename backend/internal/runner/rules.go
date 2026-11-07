package runner

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/timezone"
)

// Alasan penolakan eksekusi. Nilainya dipakai di telemetry dan kolom
// signal.reject_reason, jadi teks ini yang dibaca user di dashboard.
type RejectReason string

const (
	RejectNone          RejectReason = ""
	RejectExpired       RejectReason = "Sinyal sudah kedaluwarsa (batas waktu habis)."
	RejectNotReady      RejectReason = "Sinyal belum siap dieksekusi."
	RejectTerminal      RejectReason = "Sinyal sudah berada di status akhir."
	RejectOpenPosition  RejectReason = "Masih ada posisi terbuka. Tunggu sampai ditutup."
	RejectDailyWin      RejectReason = "Target harian tercapai (1 win). Trading dihentikan hari ini."
	RejectDailyQuota    RejectReason = "Kuota trade hari ini sudah habis."
	RejectTwoStrike     RejectReason = "Batas kerugian harian tercapai (2x SL)."
	RejectSlippage      RejectReason = "Harga sudah bergerak terlalu jauh dari level entry."
	RejectEntitlement   RejectReason = "Mode eksekusi ini tidak tersedia pada tier Anda."
	RejectStrategyTier  RejectReason = "Strategi ini belum tersedia untuk tier Anda."
	RejectAccountFrozen RejectReason = "Akun broker sedang dibekukan."
	RejectSuspended     RejectReason = "Akun sedang ditangguhkan karena pembayaran. Entry baru diblokir."
	RejectAccountBanned RejectReason = "Akun diblokir. Hubungi dukungan untuk keterangan lebih lanjut."
	RejectMarketClosed  RejectReason = "Pasar sedang tutup."
	RejectDuplicate     RejectReason = "Sinyal ini sudah pernah dieksekusi untuk akun ini."
	RejectBadSignal     RejectReason = "Data sinyal tidak lengkap atau tidak sah."
)

// Status sinyal dirujuk dari paket model agar hanya ada satu sumber nilai.
const (
	SignalReady      = model.SignalStatusReadyToEntry
	SignalAcked      = model.SignalStatusAcked
	SignalConsumed   = model.SignalStatusConsumed
	SignalExpired    = model.SignalStatusExpired
	SignalCancelled  = model.SignalStatusCancelled
	SignalSuperseded = model.SignalStatusSuperseded
	SignalRejected   = model.SignalStatusRejected
	SignalSkipped    = model.SignalStatusSkipped
)

// IsTerminal menandai sinyal yang sudah tidak bisa dieksekusi.
func IsTerminal(status string) bool {
	return model.IsTerminalSignal(status)
}

// Status posisi dirujuk dari paket model.
const (
	PositionOpen         = model.PositionStatusOpen
	PositionClosedTP     = model.PositionStatusClosedTP
	PositionClosedSL     = model.PositionStatusClosedSL
	PositionClosedManual = model.PositionStatusClosedManual
)

// ErrNoLot menandakan ukuran lot jatuh di bawah minimum broker.
var ErrNoLot = errors.New("ukuran lot di bawah minimum broker")

// DailyStatus adalah ringkasan aktivitas trading satu user dalam satu hari WIB.
type DailyStatus struct {
	Day         string    `json:"day"`
	WinCount    int       `json:"win_count"`
	LossCount   int       `json:"loss_count"`
	TotalTrades int       `json:"total_trades"`
	OpenCount   int       `json:"open_count"`
	LockedAt    time.Time `json:"locked_at,omitempty"`
	LockReason  string    `json:"lock_reason,omitempty"`
}

// DailyPolicy adalah pagar aturan harian yang berlaku untuk satu user.
// Nilai-nilainya sudah dibatasi pagar aman global di plans/14 §6.
type DailyPolicy struct {
	MaxTrades    int
	StopAfterWin bool
	MaxLosses    int
}

// DefaultDailyPolicy mengembalikan kebijakan bawaan sesuai plans/05 §3.
func DefaultDailyPolicy() DailyPolicy {
	return DailyPolicy{MaxTrades: 2, StopAfterWin: true, MaxLosses: 2}
}

// Clamp menahan nilai dari konfigurasi agar tidak melampaui pagar aman.
// Tier memberi keleluasaan, bukan kemampuan menonaktifkan pengaman.
func (p DailyPolicy) Clamp() DailyPolicy {
	if p.MaxTrades < 1 {
		p.MaxTrades = 1
	}
	if p.MaxTrades > 3 {
		p.MaxTrades = 3
	}
	if p.MaxLosses < 1 {
		p.MaxLosses = 1
	}
	if p.MaxLosses > 3 {
		p.MaxLosses = 3
	}
	// Berhenti setelah win wajib aktif; tidak ada jalan mematikannya.
	p.StopAfterWin = true
	return p
}

// EvaluateDaily menentukan apakah user boleh membuka posisi baru.
// Urutan pemeriksaan mengikuti plans/05 §3.2, dari yang paling murah.
func EvaluateDaily(d DailyStatus, p DailyPolicy) RejectReason {
	if d.OpenCount > 0 {
		return RejectOpenPosition
	}
	if p.StopAfterWin && d.WinCount >= 1 {
		return RejectDailyWin
	}
	if d.TotalTrades >= p.MaxTrades {
		return RejectDailyQuota
	}
	if d.LossCount >= p.MaxLosses {
		return RejectTwoStrike
	}
	return RejectNone
}

// WIB dan NY diteruskan dari paket timezone agar pemakai lama tetap berjalan.
// Rumah sebenarnya ada di internal/timezone supaya lapisan lain bisa memakainya.
var WIB = timezone.WIB

// SetDailyTimezone mengganti zona hari akuntansi dari konfigurasi.
// Nama zona tak dikenal diabaikan agar Runner tetap memakai WIB.
func SetDailyTimezone(name string) {
	timezone.SetDaily(name)
	WIB = timezone.WIB
}

// DayKey mengubah waktu menjadi kunci hari WIB berformat YYYY-MM-DD.
func DayKey(t time.Time) string {
	return t.In(WIB).Format("2006-01-02")
}

// DayStart mengembalikan 00:00 WIB pada hari yang memuat t.
func DayStart(t time.Time) time.Time {
	local := t.In(WIB)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, WIB)
}

// DayEnd mengembalikan 00:00 WIB pada hari berikutnya.
func DayEnd(t time.Time) time.Time {
	return DayStart(t).AddDate(0, 0, 1)
}

// slippageTolerance mengembalikan batas penyimpangan harga per simbol.
// Simbol tak dikenal memakai toleransi konservatif berbasis jarak SL.
func slippageTolerance(symbol string) (float64, bool) {
	switch strings.ToUpper(symbol) {
	case "XAUUSD":
		return 0.50, true
	case "USTEC", "NAS100", "US100":
		return 5.0, true
	case "EURUSD", "GBPUSD", "AUDUSD", "NZDUSD", "USDCAD", "USDCHF", "USDJPY":
		return 0.00010, true
	default:
		return 0, false
	}
}

// SlippageCheck menjaga kualitas eksekusi agar RR yang diproyeksikan tidak
// hancur oleh pergerakan harga selama sinyal menempuh jaringan.
//
//	batas = max(toleransi simbol, 10% jarak SL)
func SlippageCheck(symbol string, entryPrice, currentPrice, stopLoss float64) (bool, float64, float64) {
	tolerance, known := slippageTolerance(symbol)
	if !known {
		tolerance = 0
	}

	slDistance := math.Abs(entryPrice - stopLoss)
	limit := tolerance
	if tenth := slDistance * 0.10; tenth > limit {
		limit = tenth
	}
	if limit <= 0 {
		// Tanpa toleransi dan tanpa jarak SL yang masuk akal, jangan menebak.
		return false, math.Abs(currentPrice - entryPrice), limit
	}

	drift := math.Abs(currentPrice - entryPrice)
	return drift <= limit, drift, limit
}

// LotCalculation adalah hasil perhitungan ukuran posisi.
type LotCalculation struct {
	Lots         float64 `json:"lots"`
	RiskAmount   float64 `json:"risk_amount"`
	RiskPercent  float64 `json:"risk_percent"`
	StopDistance float64 `json:"stop_distance"`
	ValuePerLot  float64 `json:"value_per_lot"`
	ClampedMin   bool    `json:"clamped_min"`
	ClampedMax   bool    `json:"clamped_max"`
}

// LotSpec menampung parameter kontrak simbol yang dibutuhkan perhitungan lot.
type LotSpec struct {
	ValuePerLotPerPrice float64
	MinLot              float64
	MaxLot              float64
	LotStep             float64
}

// LotRequest adalah masukan perhitungan lot untuk satu eksekusi.
type LotRequest struct {
	Equity      float64
	RiskPercent float64
	EntryPrice  float64
	StopLoss    float64
	Spec        LotSpec
}

// Pagar risiko global; tidak ada tier yang boleh melampauinya.
const (
	MinRiskPercent = 0.10
	MaxRiskPercent = 1.00
)

// CalculateLot menghitung ukuran posisi dari risiko persen equity.
//
//	risiko($)   = equity * riskPercent / 100
//	nilai/lot   = jarak SL * valuePerLotPerPrice
//	lot         = risiko($) / nilai/lot
func CalculateLot(req LotRequest) (LotCalculation, error) {
	if req.Equity <= 0 {
		return LotCalculation{}, fmt.Errorf("%w: equity tidak tersedia", ErrNoLot)
	}
	if req.Spec.ValuePerLotPerPrice <= 0 {
		return LotCalculation{}, fmt.Errorf("%w: nilai per lot tidak diketahui", ErrNoLot)
	}

	riskPercent := req.RiskPercent
	if riskPercent < MinRiskPercent {
		riskPercent = MinRiskPercent
	}
	if riskPercent > MaxRiskPercent {
		riskPercent = MaxRiskPercent
	}

	stopDistance := math.Abs(req.EntryPrice - req.StopLoss)
	if stopDistance <= 0 {
		return LotCalculation{}, fmt.Errorf("%w: jarak stop loss nol", ErrNoLot)
	}

	riskAmount := req.Equity * riskPercent / 100
	valuePerLot := stopDistance * req.Spec.ValuePerLotPerPrice

	calc := LotCalculation{
		RiskPercent:  riskPercent,
		RiskAmount:   riskAmount,
		StopDistance: stopDistance,
		ValuePerLot:  valuePerLot,
	}

	lots := riskAmount / valuePerLot

	step := req.Spec.LotStep
	if step <= 0 {
		step = 0.01
	}
	// Bulatkan ke bawah agar risiko tidak pernah melebihi yang dipilih user.
	lots = math.Floor(lots/step) * step

	if req.Spec.MinLot > 0 && lots < req.Spec.MinLot {
		lots = req.Spec.MinLot
		calc.ClampedMin = true
	}
	if req.Spec.MaxLot > 0 && lots > req.Spec.MaxLot {
		lots = req.Spec.MaxLot
		calc.ClampedMax = true
	}

	if lots <= 0 {
		return LotCalculation{}, ErrNoLot
	}

	calc.Lots = math.Round(lots*100) / 100
	return calc, nil
}

// EffectiveTier mengembalikan tier yang berlaku saat ini.
// Trial yang masih berjalan diperlakukan setara Plus.
func EffectiveTier(tier string, trialExpiresAt, tierExpiresAt *time.Time, now time.Time) string {
	if trialExpiresAt != nil && now.Before(*trialExpiresAt) {
		return "plus"
	}
	if tierExpiresAt != nil && now.After(*tierExpiresAt) {
		return "free"
	}
	return tier
}

// WithinGrace menandai akun yang masih boleh masuk & melihat data
// tetapi tidak boleh membuka posisi baru (plans/14 §5.4).
func WithinGrace(graceUntil *time.Time, now time.Time) bool {
	return graceUntil != nil && now.After(*graceUntil)
}

// ModeAllowed menentukan apakah mode eksekusi boleh dipakai pada tier ini.
func ModeAllowed(tier, mode string) bool {
	if mode == "auto" {
		return tier == "plus"
	}
	return true
}

// StrategyAllowed menentukan apakah strategi boleh dipakai pada tier ini.
// Ketersediaan tahap ILA 3.0 diperiksa terpisah di Settings.
func StrategyAllowed(tier, strategy string) bool {
	if strategy == "ila-3.0" {
		return tier == "plus"
	}
	return true
}

// StrategyKnown menandai nilai strategy_version yang dikenali platform.
func StrategyKnown(strategy string) bool {
	return strategy == "ila-2.0" || strategy == "ila-3.0"
}

// MaxAccounts mengembalikan batas jumlah akun broker per tier.
func MaxAccounts(tier string) int {
	switch tier {
	case "plus":
		return 2
	case "basic":
		return 1
	default:
		return 0
	}
}
