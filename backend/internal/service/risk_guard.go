package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// Batas lot untuk mode risiko "lot".
//
// 0.01 adalah lot terkecil yang diterima hampir semua broker retail; 10 adalah
// pagar kewajaran platform agar satu klik tidak membuka posisi raksasa.
const (
	MinLotSize = 0.01
	MaxLotSize = 10.00
)

// validasiLot memastikan ukuran lot sah dan kelipatan 0,01.
func validasiLot(value float64) (float64, error) {
	if value < MinLotSize-1e-9 || value > MaxLotSize+1e-9 {
		return 0, fmt.Errorf("%w: ukuran lot harus %.2f sampai %.2f",
			apperror.ErrBadRequest, MinLotSize, MaxLotSize)
	}
	bulat := math.Round(value*100) / 100
	if math.Abs(bulat-value) > 1e-9 {
		return 0, fmt.Errorf("%w: ukuran lot harus kelipatan 0,01", apperror.ErrBadRequest)
	}
	return bulat, nil
}

// resolveRiskSettings menormalkan pasangan mode risiko dan ukuran lot.
//
// Aturan:
//   - mode kosong dianggap "percent" agar data lama tetap sah.
//   - mode "lot" wajib punya lot sah (0,01–10, kelipatan 0,01); kosong berarti 0,01.
//   - mode "percent" tetap menyimpan lot terakhir untuk dipakai saat berpindah mode.
func (s *accountService) resolveRiskSettings(rawMode string, rawLot, _ float64, _ string) (string, float64, error) {
	mode := strings.TrimSpace(rawMode)
	if mode == "" {
		mode = model.RiskModePercent
	}
	if mode != model.RiskModePercent && mode != model.RiskModeLot {
		return "", 0, fmt.Errorf("%w: mode risiko harus percent atau lot", apperror.ErrBadRequest)
	}

	lot := rawLot
	if lot == 0 {
		lot = MinLotSize
	}
	lot, err := validasiLot(lot)
	if err != nil {
		return "", 0, err
	}

	return mode, lot, nil
}

// Batas total risiko seluruh akun aktif. Nilai dipakai sebagai fallback bila
// katalog paket tidak menyediakan feature max_total_risk.
const (
	// Total risk default free sengaja nol: free tidak boleh punya akun nyata.
	defaultMaxTotalRiskFree  = 0.0
	defaultMaxTotalRiskBasic = 5.0
	defaultMaxTotalRiskPlus  = 5.0
)

// maxRiskPerTradeByTier mengembalikan batas risk per trade menurut tier.
// Free tidak boleh membuka akun nyata, sehingga batasnya nol.
func maxRiskPerTradeByTier(tier string) float64 {
	switch tier {
	case model.TierPlus, model.TierBasic:
		return MaxRiskPercent
	default:
		return 0
	}
}

// defaultMaxTotalRisk adalah fallback bila fitur paket tidak tersedia.
func defaultMaxTotalRisk(tier string) float64 {
	switch tier {
	case model.TierPlus:
		return defaultMaxTotalRiskPlus
	case model.TierBasic:
		return defaultMaxTotalRiskBasic
	default:
		return defaultMaxTotalRiskFree
	}
}

// validateRiskForTier memastikan nilai risiko per trade sah untuk tier.
//
// Batas atas mengikuti tier; batas bawah tetap konstanta global agar user
// tidak bisa memilih risiko nyaris nol yang membuat perhitungan lot tidak stabil.
func (s *accountService) validateRiskForTier(ctx context.Context, tier string, risk float64) (float64, error) {
	value, err := clampRisk(risk)
	if err != nil {
		return 0, err
	}
	maxAllowed := maxRiskPerTradeByTier(tier)
	if maxAllowed == 0 {
		return 0, fmt.Errorf("%w: tier %s tidak boleh menghubungkan akun nyata", apperror.ErrForbidden, tier)
	}
	if value > maxAllowed {
		return 0, fmt.Errorf("%w: risiko per trade untuk tier %s maksimal %.2f%%", apperror.ErrForbidden, tier, maxAllowed)
	}
	return value, nil
}

// validateTotalRisk memastikan penjumlahan risiko seluruh akun aktif (termasuk
// nilai baru yang akan dipakai) tidak melewati batas total tier.
//
// accounts adalah akun yang sudah ada sebelum perubahan. candidateRisk dan
// candidateAccountID menggambarkan nilai baru; akun yang sedang diubah tidak
// dihitung dua kali.
func (s *accountService) validateTotalRisk(ctx context.Context, tier string, accounts []model.BrokerAccount, candidateRisk float64, candidateAccountID string) error {
	limit := s.tierMaxTotalRisk(ctx, tier)

	total := 0.0
	for _, acc := range accounts {
		if !acc.IsActive {
			continue
		}
		if acc.ID == candidateAccountID {
			continue
		}
		if acc.RiskMode == model.RiskModeLot {
			// Akun bermode lot tidak dijumlahkan di pagar persen: risiko
			// sesungguhnya baru diketahui saat harga dan SL sinyal ada. Akun
			// seperti ini dijaga oleh batas 0,01–10 lot dan peringatan di UI.
			continue
		}
		total += acc.RiskPercent
	}
	total += candidateRisk

	// Toleransi pembulatan titik mengambang (mis. 0.10+0.20 = 0.30000000000000004).
	if total > limit+1e-9 {
		return fmt.Errorf("%w: total risiko semua akun (%.2f%%) melewati batas tier %s (%.2f%%)",
			apperror.ErrForbidden, total, tier, limit)
	}
	return nil
}

// tierMaxTotalRisk membaca batas total risiko dari katalog paket, dengan
// fallback bila feature tidak ada.
func (s *accountService) tierMaxTotalRisk(ctx context.Context, tier string) float64 {
	fallback := defaultMaxTotalRisk(tier)
	if s.features == nil {
		return fallback
	}
	raw := s.features.Feature(ctx, tier, model.FeatureMaxTotalRisk, formatPercent(fallback))
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

// formatPercent menampilkan angka tanpa nol berlebih, mis. 1 -> "1", 0.5 -> "0.5".
func formatPercent(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
