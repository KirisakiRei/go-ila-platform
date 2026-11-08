package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// PricingService menghitung harga akhir: subtotal → kupon → kredit → pagar.
// Seluruh aturan diskon ada di sini agar invoice, tampilan, dan laporan
// memakai perhitungan yang sama (plans/14 §3.2).
type PricingService interface {
	Quote(ctx context.Context, userID, planCode, couponCode string, useCredit bool) (*dto.PriceQuote, error)
	Plans(ctx context.Context) ([]dto.PublicPlan, error)
}

// PricingStore adalah data yang dibutuhkan perhitungan harga.
type PricingStore interface {
	ListPlans(ctx context.Context) ([]model.Plan, error)
	FindPlan(ctx context.Context, code string) (*model.Plan, error)
	FeaturesOfPlan(ctx context.Context, code string) (map[string]string, error)
	ActiveOverride(ctx context.Context, planCode string, at time.Time) (*model.PriceOverride, error)
	FindGuard(ctx context.Context, planCode string) (*model.PricingGuard, error)
	FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error)
	CountRedemptions(ctx context.Context, couponID string) (int64, error)
	CountUserRedemptions(ctx context.Context, couponID, userID string) (int64, error)
	CountUserInvoices(ctx context.Context, userID string) (int64, error)
	CreditBalance(ctx context.Context, userID string) (int64, error)
}

type pricingService struct {
	store PricingStore
	now   func() time.Time
}

// NewPricingService merangkai perhitungan harga.
func NewPricingService(store PricingStore) PricingService {
	return &pricingService{store: store, now: func() time.Time { return time.Now().UTC() }}
}

// Quote menghitung harga satu paket untuk satu pengguna.
func (s *pricingService) Quote(ctx context.Context, userID, planCode, couponCode string, useCredit bool) (*dto.PriceQuote, error) {
	plan, err := s.store.FindPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}
	if !plan.IsActive {
		return nil, validation("paket ini sedang tidak dijual")
	}

	now := s.now()
	subtotal := plan.PriceIDR
	overrideLabel := ""
	if override, err := s.store.ActiveOverride(ctx, plan.Code, now); err == nil && override != nil {
		subtotal = override.PriceIDR
		overrideLabel = override.Label
	}

	quote := &dto.PriceQuote{
		PlanCode: plan.Code,
		PlanName: plan.Name,
		Subtotal: subtotal,
		Total:    subtotal,
		Promo:    overrideLabel,
	}

	if strings.TrimSpace(couponCode) != "" {
		discount, err := s.evaluateCoupon(ctx, userID, plan, subtotal, couponCode, now)
		if err != nil {
			return nil, err
		}
		quote.CouponCode = discount.Code
		quote.CouponLabel = discount.Label
		quote.Discount = discount.Amount
	}

	if useCredit {
		balance, err := s.store.CreditBalance(ctx, userID)
		if err != nil {
			balance = 0
		}
		usable := balance
		remaining := subtotal - quote.Discount
		if usable > remaining-1 {
			// Kredit tidak boleh membuat total nol: total Rp 0 hanya sah bila
			// paketnya memang gratis.
			usable = remaining - 1
		}
		if usable < 0 {
			usable = 0
		}
		quote.CreditUsed = usable
	}

	quote.Total = subtotal - quote.Discount - quote.CreditUsed

	// Pagar terakhir: total tidak boleh di bawah batas yang ditetapkan admin.
	//
	// Pagar ini bersikap tertutup: paket berbayar tanpa baris pagar ditolak
	// daripada diloloskan. Paket baru yang belum punya batas bawah berarti
	// admin belum memutuskan harga minimumnya (plans/14 §6).
	guard, guardErr := s.store.FindGuard(ctx, plan.Code)
	if guardErr != nil {
		return nil, guardErr
	}
	if guard == nil {
		if plan.PriceIDR > 0 || subtotal > 0 {
			return nil, validation("batas harga minimum paket ini belum ditetapkan admin")
		}
	} else if quote.Total < guard.MinPriceIDR {
		return nil, validation("total setelah diskon berada di bawah batas harga yang diizinkan")
	}
	if quote.Total < 0 {
		return nil, validation("total tidak boleh negatif")
	}
	return quote, nil
}

// couponResult adalah hasil perhitungan satu kupon.
type couponResult struct {
	Code   string
	Label  string
	Amount int64
}

// evaluateCoupon menjalankan urutan pemeriksaan sesuai plans/14 §4.3.
func (s *pricingService) evaluateCoupon(ctx context.Context, userID string, plan *model.Plan, subtotal int64, code string, now time.Time) (*couponResult, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	coupon, err := s.store.FindCouponByCode(ctx, normalized)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, validation("kode kupon tidak valid")
		}
		return nil, err
	}
	if !coupon.IsActive {
		return nil, validation("kode kupon tidak valid")
	}
	if now.Before(coupon.StartsAt) {
		return nil, validation("kode kupon belum berlaku")
	}
	if now.After(coupon.EndsAt) {
		return nil, validation("kode kupon sudah kedaluwarsa")
	}
	if !planAllowed(coupon.AppliesToPlans, plan.Code) {
		return nil, validation("kode kupon tidak berlaku untuk paket ini")
	}

	if coupon.MaxRedemptions > 0 {
		total, err := s.store.CountRedemptions(ctx, coupon.ID)
		if err != nil {
			return nil, err
		}
		if total >= int64(coupon.MaxRedemptions) {
			return nil, validation("kode kupon sudah habis")
		}
	}

	if coupon.MaxPerUser > 0 {
		used, err := s.store.CountUserRedemptions(ctx, coupon.ID, userID)
		if err != nil {
			return nil, err
		}
		if used >= int64(coupon.MaxPerUser) {
			return nil, validation("anda sudah memakai kode kupon ini")
		}
	}

	if coupon.FirstPurchaseOnly {
		invoices, err := s.store.CountUserInvoices(ctx, userID)
		if err != nil {
			return nil, err
		}
		if invoices > 0 {
			return nil, validation("kode kupon khusus pembelian pertama")
		}
	}

	if subtotal < coupon.MinAmountIDR {
		return nil, validation("minimal transaksi belum terpenuhi untuk kode ini")
	}

	// Kupon di atas 100% akan membuat total negatif. Ditolak di service juga,
	// tidak cukup mengandalkan CHECK constraint di database.
	if coupon.DiscountType == model.DiscountPercent && coupon.DiscountValue > 100 {
		return nil, validation("kode kupon tidak valid")
	}

	amount := int64(0)
	switch coupon.DiscountType {
	case model.DiscountPercent:
		amount = subtotal * coupon.DiscountValue / 100
		if coupon.MaxDiscountIDR != nil && amount > *coupon.MaxDiscountIDR {
			amount = *coupon.MaxDiscountIDR
		}
	case model.DiscountFixed:
		amount = coupon.DiscountValue
	default:
		return nil, validation("jenis potongan kupon tidak dikenal")
	}

	// Potongan tidak boleh menghabiskan seluruh subtotal.
	if amount >= subtotal {
		amount = subtotal - 1
	}
	if amount < 0 {
		amount = 0
	}

	return &couponResult{Code: coupon.Code, Label: coupon.Description, Amount: amount}, nil
}

// planAllowed memeriksa daftar paket; kosong berarti semua paket.
func planAllowed(list, planCode string) bool {
	list = strings.TrimSpace(list)
	if list == "" {
		return true
	}
	for _, item := range strings.Split(list, ",") {
		if strings.EqualFold(strings.TrimSpace(item), planCode) {
			return true
		}
	}
	return false
}

// Plans menampilkan paket publik beserta fiturnya.
func (s *pricingService) Plans(ctx context.Context) ([]dto.PublicPlan, error) {
	plans, err := s.store.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()

	out := make([]dto.PublicPlan, 0, len(plans))
	for _, plan := range plans {
		features, err := s.store.FeaturesOfPlan(ctx, plan.Code)
		if err != nil {
			return nil, err
		}
		price := plan.PriceIDR
		label := ""
		if override, err := s.store.ActiveOverride(ctx, plan.Code, now); err == nil && override != nil {
			price = override.PriceIDR
			label = override.Label
		}
		out = append(out, dto.PublicPlan{
			Code:        plan.Code,
			Name:        plan.Name,
			Description: plan.Description,
			PriceIDR:    price,
			Interval:    plan.Interval,
			Promo:       label,
			Features:    features,
			// ILA 3.0 masih penelitian: tampil sebagai coming soon dan mati.
			ILA3ComingSoon: strings.EqualFold(features[model.FeatureILA3Status], "coming_soon"),
			Active:         plan.IsActive,
		})
	}
	return out, nil
}
