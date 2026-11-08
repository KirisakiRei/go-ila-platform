package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// Batas total risiko seluruh akun aktif. Nilai dipakai sebagai fallback bila
// katalog paket tidak menyediakan feature max_total_risk.
const (
	// Total risk default free sengaja nol: free tidak boleh punya akun nyata.
	defaultMaxTotalRiskFree  = 0.0
	defaultMaxTotalRiskBasic = 1.0
	defaultMaxTotalRiskPlus  = 2.0
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
