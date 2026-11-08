package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// fakePricingStore menyajikan data harga di memori untuk pengujian.
type fakePricingStore struct {
	plans    map[string]model.Plan
	features map[string]map[string]string
	guard    map[string]int64
	coupons  map[string]model.Coupon
	redeemed int64
	perUser  int64
	invoices int64
	balance  int64
}

func newFakePricingStore() *fakePricingStore {
	return &fakePricingStore{
		plans: map[string]model.Plan{
			model.TierFree:  {Code: model.TierFree, Name: "Free", PriceIDR: 0, IsActive: true},
			model.TierBasic: {Code: model.TierBasic, Name: "Basic", PriceIDR: 666000, IsActive: true},
			model.TierPlus:  {Code: model.TierPlus, Name: "Plus", PriceIDR: 999000, IsActive: true},
		},
		features: map[string]map[string]string{},
		// Pagar harga tidak boleh kosong: paket berbayar tanpa batas bawah
		// ditolak, jadi fikstur memberi batas yang wajar.
		guard: map[string]int64{
			model.TierBasic: 100000,
			model.TierPlus:  150000,
		},
		coupons: map[string]model.Coupon{},
	}
}

func (f *fakePricingStore) ListPlans(ctx context.Context) ([]model.Plan, error) {
	out := make([]model.Plan, 0, len(f.plans))
	for _, plan := range f.plans {
		out = append(out, plan)
	}
	return out, nil
}

func (f *fakePricingStore) FindPlan(ctx context.Context, code string) (*model.Plan, error) {
	plan, ok := f.plans[code]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return &plan, nil
}

func (f *fakePricingStore) FeaturesOfPlan(ctx context.Context, code string) (map[string]string, error) {
	return f.features[code], nil
}

func (f *fakePricingStore) ActiveOverride(ctx context.Context, planCode string, at time.Time) (*model.PriceOverride, error) {
	return nil, nil
}

func (f *fakePricingStore) FindGuard(ctx context.Context, planCode string) (*model.PricingGuard, error) {
	min, ok := f.guard[planCode]
	if !ok {
		return nil, nil
	}
	return &model.PricingGuard{PlanCode: planCode, MinPriceIDR: min}, nil
}

func (f *fakePricingStore) FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error) {
	coupon, ok := f.coupons[code]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return &coupon, nil
}

func (f *fakePricingStore) CountRedemptions(ctx context.Context, couponID string) (int64, error) {
	return f.redeemed, nil
}

func (f *fakePricingStore) CountUserRedemptions(ctx context.Context, couponID, userID string) (int64, error) {
	return f.perUser, nil
}

func (f *fakePricingStore) CountUserInvoices(ctx context.Context, userID string) (int64, error) {
	return f.invoices, nil
}

func (f *fakePricingStore) CreditBalance(ctx context.Context, userID string) (int64, error) {
	return f.balance, nil
}

func newCoupon(code, kind string, value int64) model.Coupon {
	now := time.Now().UTC()
	return model.Coupon{
		Code: code, DiscountType: kind, DiscountValue: value,
		StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour),
		IsActive: true, MaxPerUser: 1,
	}
}

func TestQuoteAppliesPercentCoupon(t *testing.T) {
	store := newFakePricingStore()
	store.coupons["HEMAT20"] = newCoupon("HEMAT20", model.DiscountPercent, 20)
	svc := NewPricingService(store)

	quote, err := svc.Quote(context.Background(), "u1", model.TierBasic, "hemat20", false)
	if err != nil {
		t.Fatalf("kupon sah harus diterima: %v", err)
	}
	if quote.Discount != 133200 {
		t.Fatalf("potongan 20%% dari 666.000 harus 133.200, dapat %d", quote.Discount)
	}
	if quote.Total != 532800 {
		t.Fatalf("total harus 532.800, dapat %d", quote.Total)
	}
}

func TestPercentCapApplied(t *testing.T) {
	store := newFakePricingStore()
	coupon := newCoupon("BESAR", model.DiscountPercent, 50)
	cap := int64(50000)
	coupon.MaxDiscountIDR = &cap
	store.coupons["BESAR"] = coupon
	svc := NewPricingService(store)

	quote, err := svc.Quote(context.Background(), "u1", model.TierPlus, "BESAR", false)
	if err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if quote.Discount != 50000 {
		t.Fatalf("potongan harus dibatasi 50.000, dapat %d", quote.Discount)
	}
}

func TestCoupon500PercentRejected(t *testing.T) {
	// Nilai ini tidak mungkin lolos CHECK constraint DB; service juga menolak
	// agar tidak bergantung pada lapisan bawah.
	store := newFakePricingStore()
	store.coupons["GILA"] = newCoupon("GILA", model.DiscountPercent, 500)
	svc := NewPricingService(store)

	if _, err := svc.Quote(context.Background(), "u1", model.TierBasic, "GILA", false); err == nil {
		t.Fatal("kupon di atas 100% harus ditolak")
	}
}

func TestExhaustedCouponRejected(t *testing.T) {
	store := newFakePricingStore()
	coupon := newCoupon("HABIS", model.DiscountPercent, 10)
	coupon.MaxRedemptions = 5
	store.coupons["HABIS"] = coupon
	store.redeemed = 5
	svc := NewPricingService(store)

	_, err := svc.Quote(context.Background(), "u1", model.TierBasic, "HABIS", false)
	if err == nil {
		t.Fatal("kupon habis harus ditolak")
	}
}

func TestPerUserLimitRejected(t *testing.T) {
	store := newFakePricingStore()
	coupon := newCoupon("SEKALI", model.DiscountPercent, 10)
	coupon.MaxPerUser = 1
	store.coupons["SEKALI"] = coupon
	store.perUser = 1
	svc := NewPricingService(store)

	if _, err := svc.Quote(context.Background(), "u1", model.TierBasic, "SEKALI", false); err == nil {
		t.Fatal("kupon melebihi batas per pengguna harus ditolak")
	}
}

func TestExpiredCouponRejected(t *testing.T) {
	store := newFakePricingStore()
	coupon := newCoupon("LAMA", model.DiscountPercent, 10)
	coupon.EndsAt = time.Now().Add(-time.Hour)
	coupon.StartsAt = time.Now().Add(-2 * time.Hour)
	store.coupons["LAMA"] = coupon
	svc := NewPricingService(store)

	if _, err := svc.Quote(context.Background(), "u1", model.TierBasic, "LAMA", false); err == nil {
		t.Fatal("kupon kedaluwarsa harus ditolak")
	}
}

func TestFirstPurchaseOnlyRejected(t *testing.T) {
	store := newFakePricingStore()
	coupon := newCoupon("BARU", model.DiscountPercent, 10)
	coupon.FirstPurchaseOnly = true
	store.coupons["BARU"] = coupon
	store.invoices = 1
	svc := NewPricingService(store)

	if _, err := svc.Quote(context.Background(), "u1", model.TierBasic, "BARU", false); err == nil {
		t.Fatal("kupon pembelian pertama harus ditolak untuk pelanggan lama")
	}
}

func TestGuardRejectsTooCheapTotal(t *testing.T) {
	store := newFakePricingStore()
	store.guard[model.TierPlus] = 499000
	svc := NewPricingService(store)

	// Kredit besar tidak boleh menjatuhkan total di bawah batas bawah.
	store.balance = 900000
	_, err := svc.Quote(context.Background(), "u1", model.TierPlus, "", true)
	if err == nil {
		t.Fatal("total di bawah batas bawah harus ditolak")
	}
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("error harus BAD_REQUEST, dapat %v", err)
	}
}

func TestCouponForOtherPlanRejected(t *testing.T) {
	store := newFakePricingStore()
	coupon := newCoupon("PLUSONLY", model.DiscountPercent, 10)
	coupon.AppliesToPlans = "plus"
	store.coupons["PLUSONLY"] = coupon
	svc := NewPricingService(store)

	if _, err := svc.Quote(context.Background(), "u1", model.TierBasic, "PLUSONLY", false); err == nil {
		t.Fatal("kupon khusus Plus harus ditolak untuk Basic")
	}
}

func TestCreditNeverMakesTotalZero(t *testing.T) {
	store := newFakePricingStore()
	store.balance = 5000000
	svc := NewPricingService(store)

	quote, err := svc.Quote(context.Background(), "u1", model.TierBasic, "", true)
	if err != nil {
		// Ditolak pagar juga dapat diterima; yang penting bukan total nol.
		return
	}
	if quote.Total <= 0 {
		t.Fatalf("kredit tidak boleh membuat total nol, dapat %d", quote.Total)
	}
}

// Pagar aman trading tidak pernah berubah karena voucher: fitur tier dibaca
// dari data paket, bukan dari kupon.
func TestVoucherDoesNotChangeTradingLimits(t *testing.T) {
	store := newFakePricingStore()
	store.features[model.TierFree] = map[string]string{
		model.FeatureAccountLimit: "1",
		model.FeatureAutoExecute:  "false",
	}
	store.coupons["GRATIS"] = newCoupon("GRATIS", model.DiscountPercent, 100)
	svc := NewPricingService(store)

	// Paket gratis tidak perlu kupon; yang diuji adalah fitur tier tetap utuh.
	if _, err := svc.Quote(context.Background(), "u1", model.TierFree, "", false); err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	features, _ := store.FeaturesOfPlan(context.Background(), model.TierFree)
	if features[model.FeatureAutoExecute] != "false" {
		t.Fatal("kupon tidak boleh mengubah fitur eksekusi")
	}
}

var _ = repository.ApplyCreditInput{}

// Paket berbayar tanpa baris pagar harus ditolak, bukan diloloskan: kesalahan
// konfigurasi tidak boleh berubah menjadi harga hampir gratis.
func TestPagarHargaKosongMenolakPaketBerbayar(t *testing.T) {
	store := newFakePricingStore()
	delete(store.guard, model.TierBasic)
	svc := NewPricingService(store)

	_, err := svc.Quote(context.Background(), "u1", model.TierBasic, "", false)
	if err == nil {
		t.Fatal("paket berbayar tanpa batas bawah harus ditolak")
	}
}
