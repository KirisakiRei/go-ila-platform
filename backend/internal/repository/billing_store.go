package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// BillingStore adalah seluruh data yang dibutuhkan alur uang: invoice,
// pembayaran, langganan, dan voucher saldo. Satu nilai untuk admin dan
// pengguna agar tidak ada dua jalur yang bisa berbeda hasilnya.
type BillingStore interface {
	FindPlan(ctx context.Context, code string) (*model.Plan, error)
	CreditBalance(ctx context.Context, userID string) (int64, error)
	ApplyCredit(ctx context.Context, input *ApplyCreditInput) (int64, error)
	NextInvoiceNumber(ctx context.Context, at time.Time) (string, error)
	CreateInvoice(ctx context.Context, invoice *model.Invoice) error
	FindInvoiceByNumber(ctx context.Context, number string) (*model.Invoice, error)
	FindInvoiceByID(ctx context.Context, id string) (*model.Invoice, error)
	UpdateInvoice(ctx context.Context, invoice *model.Invoice) error
	ListInvoices(ctx context.Context, userID string, limit int) ([]model.Invoice, error)
	SavePayment(ctx context.Context, payment *model.Payment) (bool, error)
	SaveRedemption(ctx context.Context, redemption *model.CouponRedemption) error
	ActiveSubscription(ctx context.Context, userID string) (*model.SubscriptionRecord, error)
	UpsertSubscription(ctx context.Context, sub *model.SubscriptionRecord) error
	ExtendTier(ctx context.Context, userID, planCode string, start, end time.Time) error
	DowngradeExpired(ctx context.Context, at time.Time) ([]string, error)
	ExpireOverdue(ctx context.Context, at time.Time) (int64, error)
	StartGrace(ctx context.Context, at time.Time) ([]string, error)
	AddRefund(ctx context.Context, refund *model.Refund) error
	FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error)

	ListPlans(ctx context.Context) ([]model.Plan, error)
	ActiveOverride(ctx context.Context, planCode string, at time.Time) (*model.PriceOverride, error)
	SavePlan(ctx context.Context, plan *model.Plan, features map[string]string) error
	FeaturesOfPlan(ctx context.Context, code string) (map[string]string, error)
	SaveGuard(ctx context.Context, guard *model.PricingGuard) error
	FindGuard(ctx context.Context, code string) (*model.PricingGuard, error)
	ListCoupons(ctx context.Context, status, q string) ([]model.Coupon, error)
	SaveCoupon(ctx context.Context, coupon *model.Coupon) error
	FindCouponByID(ctx context.Context, id string) (*model.Coupon, error)
	CountRedemptions(ctx context.Context, couponID string) (int64, error)
	CountUserRedemptions(ctx context.Context, couponID, userID string) (int64, error)
	CountUserInvoices(ctx context.Context, userID string) (int64, error)
	CreditHistory(ctx context.Context, userID string, limit int) ([]model.CreditTransaction, error)

	ListInvoicesByStatus(ctx context.Context, status, q string, limit int) ([]model.Invoice, error)
	RefundedAmount(ctx context.Context, paymentID string) (int64, error)

	Redeem(ctx context.Context, userID, code string, now time.Time) (int64, error)
	ListCreditCodes(ctx context.Context, status, q string, limit int) ([]model.CreditCode, error)
	FindCreditCodeByID(ctx context.Context, id string) (*model.CreditCode, error)
	CreateCreditCode(ctx context.Context, code *model.CreditCode) error
	CountCreditRedemptions(ctx context.Context, codeID string) (int64, error)
	SetCreditCodeActive(ctx context.Context, id string, active bool) error
}

// billingStore adalah satu-satunya implementasi alur uang.
// Dua bidang digabung (billing & pricing) agar laporan dan hak paket membaca
// angka yang sama.
type billingStore struct {
	*billingRepository
	pricing *pricingRepository
	codes   *creditCodeRepository
}

// NewBillingStore merangkai seluruh data uang.
func NewBillingStore(db *gorm.DB) *billingStore {
	return &billingStore{
		billingRepository: NewBillingRepository(db),
		pricing:           NewPricingRepository(db),
		codes:             NewCreditCodeRepository(db),
	}
}

// ActiveOverride membaca kampanye harga yang sedang berlaku.
func (s *billingStore) ActiveOverride(ctx context.Context, planCode string, at time.Time) (*model.PriceOverride, error) {
	return s.pricing.ActiveOverride(ctx, planCode, at)
}

// ListPlans membaca seluruh paket untuk panel admin.
func (s *billingStore) ListPlans(ctx context.Context) ([]model.Plan, error) {
	return s.pricing.ListPlans(ctx)
}

// SavePlan menyimpan harga dan fitur paket.
func (s *billingStore) SavePlan(ctx context.Context, plan *model.Plan, features map[string]string) error {
	return s.pricing.SavePlan(ctx, plan, features)
}

// FeaturesOfPlan membaca fitur satu paket.
func (s *billingStore) FeaturesOfPlan(ctx context.Context, code string) (map[string]string, error) {
	return s.pricing.FeaturesOfPlan(ctx, code)
}

// FindGuard membaca batas bawah harga paket.
func (s *billingStore) FindGuard(ctx context.Context, code string) (*model.PricingGuard, error) {
	return s.pricing.FindGuard(ctx, code)
}

// SaveGuard menyimpan batas bawah harga paket.
func (s *billingStore) SaveGuard(ctx context.Context, guard *model.PricingGuard) error {
	return s.pricing.SaveGuard(ctx, guard)
}

// ListCoupons membaca kupon untuk panel admin.
func (s *billingStore) ListCoupons(ctx context.Context, status, q string) ([]model.Coupon, error) {
	return s.pricing.ListCoupons(ctx, status, q)
}

// SaveCoupon menyimpan kupon baru maupun perubahan.
func (s *billingStore) SaveCoupon(ctx context.Context, coupon *model.Coupon) error {
	return s.pricing.SaveCoupon(ctx, coupon)
}

// FindCouponByID membaca satu kupon.
func (s *billingStore) FindCouponByID(ctx context.Context, id string) (*model.Coupon, error) {
	return s.pricing.FindCouponByID(ctx, id)
}

// CountRedemptions menghitung pemakaian kupon.
func (s *billingStore) CountRedemptions(ctx context.Context, couponID string) (int64, error) {
	return s.pricing.CountRedemptions(ctx, couponID)
}

// CountUserRedemptions menghitung pemakaian kupon oleh satu pengguna.
func (s *billingStore) CountUserRedemptions(ctx context.Context, couponID, userID string) (int64, error) {
	return s.pricing.CountUserRedemptions(ctx, couponID, userID)
}

// CountUserInvoices menghitung invoice lunas pengguna.
func (s *billingStore) CountUserInvoices(ctx context.Context, userID string) (int64, error) {
	return s.pricing.CountUserInvoices(ctx, userID)
}

// CreditHistory membaca buku besar kredit pengguna.
func (s *billingStore) CreditHistory(ctx context.Context, userID string, limit int) ([]model.CreditTransaction, error) {
	return s.pricing.CreditHistory(ctx, userID, limit)
}

// Redeem menukar voucher saldo menjadi kredit.
func (s *billingStore) Redeem(ctx context.Context, userID, code string, now time.Time) (int64, error) {
	return s.codes.Redeem(ctx, userID, code, now)
}

// ListCreditCodes membaca daftar voucher saldo.
func (s *billingStore) ListCreditCodes(ctx context.Context, status, q string, limit int) ([]model.CreditCode, error) {
	return s.codes.ListCreditCodes(ctx, status, q, limit)
}

// FindCreditCodeByID membaca satu voucher saldo.
func (s *billingStore) FindCreditCodeByID(ctx context.Context, id string) (*model.CreditCode, error) {
	return s.codes.FindByID(ctx, id)
}

// CreateCreditCode menyimpan voucher saldo baru.
func (s *billingStore) CreateCreditCode(ctx context.Context, code *model.CreditCode) error {
	return s.codes.Create(ctx, code)
}

// CountCreditRedemptions menghitung pemakaian satu voucher.
func (s *billingStore) CountCreditRedemptions(ctx context.Context, codeID string) (int64, error) {
	return s.codes.CountRedemptions(ctx, codeID)
}

// SetCreditCodeActive mengubah status aktif voucher.
func (s *billingStore) SetCreditCodeActive(ctx context.Context, id string, active bool) error {
	return s.codes.SetActive(ctx, id, active)
}

// ApplyCredit menulis saldo dan buku besar lewat satu jalur.
func (s *billingStore) ApplyCredit(ctx context.Context, input *ApplyCreditInput) (int64, error) {
	return s.pricing.ApplyCredit(ctx, input)
}

// FindCouponByCode membaca kupon untuk dicatat pemakaiannya.
func (s *billingStore) FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error) {
	return s.billingRepository.FindCouponByCode(ctx, code)
}
