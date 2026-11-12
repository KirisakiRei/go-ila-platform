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
	RejectNone RejectReason = ""
	// Dua alasan berbeda perlu dibedakan agar notifikasi bisa menjelaskan
	// keadaan sebenarnya: "lewat masa berlaku" hanya terjadi pada sinyal yang
	// sempat READY, sedangkan "menunggu setup berikutnya" terjadi pada sinyal
	// yang memang belum pernah siap sehingga wajar bila baris lamanya basi.
	RejectExpired      RejectReason = "Sinyal sudah kedaluwarsa (batas waktu habis)."
	RejectWaitingNext  RejectReason = "Setup belum siap dan sudah menunggu lewat satu lilin. Radar menunggu setup berikutnya."
	RejectNotReady     RejectReason = "Sinyal belum siap dieksekusi."
	RejectTerminal     RejectReason = "Sinyal sudah berada di status akhir."
	RejectOpenPosition RejectReason = "Masih ada posisi terbuka. Tunggu sampai ditutup."
	RejectDailyWin     RejectReason = "Target harian tercapai (1 win). Trading dihentikan hari ini."
	RejectDailyQuota   RejectReason = "Kuota trade hari ini sudah habis."
	RejectTwoStrike    RejectReason = "Batas kerugian harian tercapai (2x SL)."
	RejectSlippage     RejectReason = "Harga sudah bergerak terlalu jauh dari level entry."
	RejectEntitlement  RejectReason = "Mode eksekusi ini tidak tersedia pada tier Anda."
	RejectStrategyTier RejectReason = "Strategi ini belum tersedia untuk tier Anda."
	// Metode ini memang dimatikan pemilik akun, bukan soal tier.
	RejectStrategyDisabled RejectReason = "Metode analisis ini sedang dimatikan untuk akun tersebut. Nyalakan di halaman Sinyal bila ingin dipantau."
	// Pesan penolakan dibaca pengguna, jadi menyebut langkah berikutnya.
	RejectLegalPending  RejectReason = "Ada dokumen wajib yang belum Anda setujui. Buka dashboard untuk menyetujuinya; setup tetap dipantau."
	RejectAccountFrozen RejectReason = "Eksekusi di akun broker Anda sedang dibekukan. Hubungi dukungan untuk membukanya; setup dan notifikasi tetap berjalan."
	RejectSuspended     RejectReason = "Akun sedang ditangguhkan karena pembayaran. Entry baru diblokir."
	RejectAccountBanned RejectReason = "Akun diblokir. Hubungi dukungan untuk keterangan lebih lanjut."
	RejectMarketClosed  RejectReason = "Pasar sedang tutup."
	// Mode platform menahan entry baru tanpa menyentuh posisi berjalan.
	RejectMaintenance RejectReason = "Sistem sedang dalam pemeliharaan. Setup tetap tampil, entry baru ditahan sementara."
	RejectReadOnly    RejectReason = "Mode hanya-baca aktif. Setup tetap tampil, entry baru ditahan sementara."
	RejectDuplicate   RejectReason = "Sinyal ini sudah pernah dieksekusi untuk akun ini."
	RejectBadSignal   RejectReason = "Data sinyal tidak lengkap atau tidak sah."
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

// slippageTolerance mengembalikan batas penyimpangan harga dasar per simbol.
//
// Nilai ini adalah toleransi MINIMUM. Batas efektif selalu dihitung ulang oleh
// EffectiveSlippageLimit, yang ikut mempertimbangkan spread akun dan jarak SL.
//
// Satuan mengikuti harga instrumen apa adanya (bukan pip), karena broker
// berbeda-beda menuliskan desimal. Contoh: EURUSD 10 pip ditulis 0.00100.
//
// Rujukan angka mengikuti data sinyal nyata (jarak SL rata-rata per pair):
//   - XAUUSD  SL ~$5.00  -> 0.75 (75 pips)  ~15% SL
//   - XAUEUR  SL ~EUR16  -> 1.50            ~9% SL
//   - XAGUSD  SL ~$0.28  -> 0.15            ~54% SL
//   - EURUSD  SL ~16 pip -> 10 pip          ~63% SL (dipilih: gagal entry
//     jauh lebih merugikan daripada meleset 10 pip pada RR 2.5)
//   - USTEC   SL ~110    -> 40              ~36% SL
func slippageTolerance(symbol string) (float64, bool) {
	switch strings.ToUpper(symbol) {
	case "XAUUSD":
		return 0.75, true
	case "XAUEUR":
		return 1.50, true
	case "XAGUSD":
		return 0.15, true
	case "EURUSD", "GBPUSD", "AUDUSD", "NZDUSD", "USDCAD", "USDCHF":
		return 0.00100, true
	case "USDJPY":
		return 0.10, true
	case "USTEC", "NAS100", "US100":
		return 40, true
	case "US30", "DJ30", "WS30":
		return 30, true
	case "US500", "SPX500", "US500.CASH":
		return 15, true
	case "BTCUSD":
		return 50, true
	default:
		return 0, false
	}
}

// slippageSLFraction adalah porsi jarak SL yang boleh termakan slippage.
//
// Dinaikkan dari 10% ke 15% karena data produksi menunjukkan harga sering
// bergerak 3-6% jarak SL hanya dalam hitungan detik antara sinyal terbit dan
// order terkirim. Dengan 15%, RR terencana 2.5 turun paling jauh ke ~2.15 -
// masih sehat, jauh lebih baik daripada kehilangan entry sama sekali.
const slippageSLFraction = 0.15

// SlippageDecision adalah hasil perhitungan batas slippage beserta asalnya.
//
// Basis dicatat agar keputusan bisa diaudit: ketika sebuah entry ditolak,
// operator bisa melihat sumber batas mana yang berlaku (tabel pair, spread
// akun, atau persentase SL) tanpa menebak.
type SlippageDecision struct {
	// Symbol adalah simbol yang diperiksa.
	Symbol string
	// Drift adalah besar pergerakan harga dari level entry ke harga sekarang.
	Drift float64
	// Limit adalah batas efektif yang berlaku.
	Limit float64
	// Basis menandai sumber batas yang dipakai: "pair", "spread", atau "sl".
	Basis string
	// Spread adalah spread akun saat pemeriksaan; nol bila tidak diketahui.
	Spread float64
	// SLDistance adalah jarak entry ke stop loss.
	SLDistance float64
}

// slippageEpsilon adalah kelonggaran relatif pada perbandingan drift.
//
// Aritmetika float64 membuat selisih harga yang "seharusnya" tepat sama dengan
// batas kadang sedikit lebih besar (mis. 1.08600-1.08500 = 0.001000000...09).
// Tanpa kelonggaran ini, harga yang persis di batas ditolak hanya karena galat
// representasi, sehingga pedagang melihat penolakan yang tidak bisa dijelaskan.
const slippageEpsilon = 1e-9

// OK menandai apakah drift masih dalam batas. Batas bersifat inklusif.
func (d SlippageDecision) OK() bool {
	if d.Limit <= 0 {
		// Tanpa batas yang masuk akal, jangan menebak: tolak.
		return false
	}
	return d.Drift <= d.Limit*(1+slippageEpsilon)
}

// LimitOptions adalah masukan opsional untuk EffectiveSlippageLimit.
//
// Zero value berarti "tanpa spread diketahui", sehingga perilakunya sama
// dengan sebelum fitur spread akun ada. Ini menjaga pemanggil lama tetap sah.
type LimitOptions struct {
	// Spread adalah spread aktual akun untuk simbol ini (ask - bid).
	Spread float64
}

// EffectiveSlippageLimit menghitung batas slippage yang berlaku.
//
//	batas = max(toleransi_pair, faktor_spread x spread_akun, 15% jarak SL)
//
// Setiap komponen menjawab masalah berbeda:
//   - toleransi_pair : perbedaan karakter instrumen (XAUUSD vs EURUSD).
//   - spread_akun    : akun dengan spread lebar wajar bergerak lebih jauh,
//     karena harga yang tersedia memang sudah lebih jauh dari level.
//   - 15% jarak SL   : adaptasi terhadap volatilitas setup itu sendiri.
//
// spreadFactor sengaja 2x spread: satu spread sudah termakan saat masuk, satu
// lagi memberi ruang untuk pergerakan selama order menempuh jaringan.
const spreadFactor = 2.0

func EffectiveSlippageLimit(symbol string, entryPrice, stopLoss float64, opts LimitOptions) SlippageDecision {
	tolerance, _ := slippageTolerance(symbol)

	slDistance := math.Abs(entryPrice - stopLoss)
	fromSL := slDistance * slippageSLFraction

	limit := tolerance
	basis := "pair"
	if tolerance == 0 {
		basis = "sl"
	}

	if fromSL > limit {
		limit = fromSL
		basis = "sl"
	}

	if spread := math.Abs(opts.Spread); spread > 0 {
		if fromSpread := spread * spreadFactor; fromSpread > limit {
			limit = fromSpread
			basis = "spread"
		}
	}

	return SlippageDecision{
		Symbol:     symbol,
		Limit:      limit,
		Basis:      basis,
		Spread:     math.Abs(opts.Spread),
		SLDistance: slDistance,
	}
}

// SlippageCheck memeriksa apakah harga saat ini masih layak dieksekusi.
//
// Dipertahankan dengan tanda tangan lama agar pemanggil dan pengujian yang
// sudah ada tetap sah. Implementasinya kini memakai EffectiveSlippageLimit
// dengan spread tidak diketahui; pemanggil yang punya data bid/ask sebaiknya
// memakai EffectiveSlippageLimit langsung agar spread akun ikut dihitung.
func SlippageCheck(symbol string, entryPrice, currentPrice, stopLoss float64) (bool, float64, float64) {
	decision := EffectiveSlippageLimit(symbol, entryPrice, stopLoss, LimitOptions{})
	decision.Drift = math.Abs(currentPrice - entryPrice)
	return decision.OK(), decision.Drift, decision.Limit
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
// Trial yang masih berjalan diperlakukan setara Basic.
func EffectiveTier(tier string, trialExpiresAt, tierExpiresAt *time.Time, now time.Time) string {
	if trialExpiresAt != nil && now.Before(*trialExpiresAt) {
		return model.TierTrial
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
