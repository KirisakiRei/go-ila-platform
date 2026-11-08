package service

import (
	"context"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

// AdminBillingService mengelola paket, kupon, dan kredit dari panel admin.
// Seluruh aksi mengubah uang, jadi setiap perubahan tercatat di audit.
type AdminBillingService interface {
	Plans(ctx context.Context) ([]dto.PublicPlan, error)
	UpdatePlan(ctx context.Context, actor AdminActor, code string, req dto.PlanRequest) error
	SetGuard(ctx context.Context, actor AdminActor, code string, req dto.GuardRequest) error
	Coupons(ctx context.Context, status, q string) ([]dto.CouponItem, error)
	CreateCoupon(ctx context.Context, actor AdminActor, req dto.CouponRequest) (*dto.CouponItem, error)
	UpdateCoupon(ctx context.Context, actor AdminActor, id string, req dto.CouponRequest) error
	DisableCoupon(ctx context.Context, actor AdminActor, id, reason string) error
	CreditSummary(ctx context.Context, actor AdminActor, userID string) (*dto.CreditSummary, error)
	AdjustCredit(ctx context.Context, actor AdminActor, userID string, req dto.CreditRequest) (*dto.CreditSummary, error)

	// Kode kredit: voucher saldo yang ditukar pengguna sendiri.
	CreditCodes(ctx context.Context, status, q string) ([]dto.CreditCodeItem, error)
	CreateCreditCode(ctx context.Context, actor AdminActor, req dto.CreditCodeRequest) (*dto.CreditCodeItem, error)
	DisableCreditCode(ctx context.Context, actor AdminActor, id, reason string) error

	// Invoice: pencatatan pembayaran manual dan refund manual.
	Invoices(ctx context.Context, status, q string, limit int) ([]dto.AdminInvoiceItem, error)
	MarkInvoicePaid(ctx context.Context, actor AdminActor, id string, req dto.ManualPaymentRequest) error
	RefundPayment(ctx context.Context, actor AdminActor, id string, req dto.RefundRequest) error
}

// billingStore adalah kebutuhan kelola uang di sisi admin.
type billingStore interface {
	ListPlans(ctx context.Context) ([]model.Plan, error)
	FindPlan(ctx context.Context, code string) (*model.Plan, error)
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
	FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error)
	CreditBalance(ctx context.Context, userID string) (int64, error)
	ApplyCredit(ctx context.Context, input *repository.ApplyCreditInput) (int64, error)
	CreditHistory(ctx context.Context, userID string, limit int) ([]model.CreditTransaction, error)

	// Voucher saldo.
	ListCreditCodes(ctx context.Context, status, q string, limit int) ([]model.CreditCode, error)
	FindCreditCodeByID(ctx context.Context, id string) (*model.CreditCode, error)
	CreateCreditCode(ctx context.Context, code *model.CreditCode) error
	CountCreditRedemptions(ctx context.Context, codeID string) (int64, error)
	SetCreditCodeActive(ctx context.Context, id string, active bool) error

	// Invoice admin.
	ListInvoicesByStatus(ctx context.Context, status, q string, limit int) ([]model.Invoice, error)
	FindInvoiceByID(ctx context.Context, id string) (*model.Invoice, error)
	UpdateInvoice(ctx context.Context, invoice *model.Invoice) error
	SavePayment(ctx context.Context, payment *model.Payment) (bool, error)
	AddRefund(ctx context.Context, refund *model.Refund) error
	RefundedAmount(ctx context.Context, paymentID string) (int64, error)
}

type adminBillingService struct {
	store    billingStore
	audit    AdminAudit
	subsaver SubscriptionExtender
	gateway  RefundGateway
	now      func() time.Time
}

// RefundGateway meneruskan pengajuan refund ke penyedia pembayaran.
// Penyedia tanpa API refund hanya mencatat; penyedia yang punya API bisa
// benar-benar mengembalikan dana tanpa mengubah pemanggilnya.
type RefundGateway interface {
	Name() string
	RequestRefund(ctx context.Context, transactionID string, amountIDR int64, reason string) error
}

// SubscriptionExtender memberi hak paket setelah pembayaran sah.
// Dipisah dari store agar jalur uang tunggal tetap di BillingService.
type SubscriptionExtender interface {
	ActivatePlan(ctx context.Context, userID, planCode string, periodStart, periodEnd time.Time) error
}

// NewAdminBillingService merangkai kelola uang.
func NewAdminBillingService(store billingStore, recorder AdminAudit, subs SubscriptionExtender, gateway RefundGateway) AdminBillingService {
	return &adminBillingService{
		store: store, audit: recorder, subsaver: subs, gateway: gateway,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *adminBillingService) Plans(ctx context.Context) ([]dto.PublicPlan, error) {
	return NewPricingService(s.store).Plans(ctx)
}

func (s *adminBillingService) UpdatePlan(ctx context.Context, actor AdminActor, code string, req dto.PlanRequest) error {
	if err := ValidateReason(audit.PlanPriceChange, req.Reason); err != nil {
		return err
	}
	plan, err := s.store.FindPlan(ctx, code)
	if err != nil {
		return err
	}

	// Harga tidak boleh turun di bawah pagar: satu salah ketik tidak boleh
	// menjual paket berbayar hampir gratis (plans/14 §6).
	if guard, err := s.store.FindGuard(ctx, code); err == nil && guard != nil {
		if req.PriceIDR < guard.MinPriceIDR {
			return validation("harga tidak boleh di bawah batas bawah paket ini")
		}
	}

	before := map[string]any{"price_idr": plan.PriceIDR, "name": plan.Name, "is_active": plan.IsActive}

	updated := *plan
	updated.Name = strings.TrimSpace(req.Name)
	updated.PriceIDR = req.PriceIDR
	updated.Description = strings.TrimSpace(req.Description)
	if req.IsActive != nil {
		updated.IsActive = *req.IsActive
	}
	if err := s.store.SavePlan(ctx, &updated, req.Features); err != nil {
		return err
	}

	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.PlanPriceChange,
		TargetType: "plan", TargetID: code, Reason: req.Reason, RequestID: actor.RequestID,
		Before: before,
		After:  map[string]any{"price_idr": req.PriceIDR, "name": updated.Name, "is_active": updated.IsActive},
	})
}

func (s *adminBillingService) SetGuard(ctx context.Context, actor AdminActor, code string, req dto.GuardRequest) error {
	if err := ValidateReason(audit.PricingGuardChange, req.Reason); err != nil {
		return err
	}
	before := int64(0)
	if current, err := s.store.FindGuard(ctx, code); err == nil && current != nil {
		before = current.MinPriceIDR
	}
	guard := &model.PricingGuard{PlanCode: code, MinPriceIDR: req.MinPriceIDR, UpdatedBy: actor.ID}
	if err := s.store.SaveGuard(ctx, guard); err != nil {
		return err
	}
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.PricingGuardChange,
		TargetType: "plan", TargetID: code, Reason: req.Reason, RequestID: actor.RequestID,
		Before: map[string]any{"min_price_idr": before},
		After:  map[string]any{"min_price_idr": req.MinPriceIDR},
	})
}

func (s *adminBillingService) Coupons(ctx context.Context, status, q string) ([]dto.CouponItem, error) {
	rows, err := s.store.ListCoupons(ctx, status, strings.TrimSpace(q))
	if err != nil {
		return nil, err
	}
	now := s.now()
	out := make([]dto.CouponItem, 0, len(rows))
	for _, row := range rows {
		used, err := s.store.CountRedemptions(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		item := toCouponItem(row, used, now)
		if status != "" && item.Status != status {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *adminBillingService) CreateCoupon(ctx context.Context, actor AdminActor, req dto.CouponRequest) (*dto.CouponItem, error) {
	coupon, err := s.buildCoupon(req)
	if err != nil {
		return nil, err
	}
	coupon.ID = ""
	coupon.CreatedBy = actor.ID
	coupon.CreatedAt = s.now()
	if err := s.store.SaveCoupon(ctx, &coupon); err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.CouponCreate,
		TargetType: "coupon", TargetID: coupon.ID,
		After:     map[string]any{"code": coupon.Code, "type": coupon.DiscountType, "value": coupon.DiscountValue},
		RequestID: actor.RequestID,
	}); err != nil {
		return nil, err
	}
	item := toCouponItem(coupon, 0, s.now())
	return &item, nil
}

func (s *adminBillingService) UpdateCoupon(ctx context.Context, actor AdminActor, id string, req dto.CouponRequest) error {
	existing, err := s.store.FindCouponByID(ctx, id)
	if err != nil {
		return err
	}
	used, err := s.store.CountRedemptions(ctx, id)
	if err != nil {
		return err
	}
	// Nilai kupon yang sudah dipakai tidak diubah: laporan akan menyesatkan.
	if used > 0 && (existing.DiscountType != req.DiscountType || existing.DiscountValue != req.DiscountValue) {
		return validation("nilai kupon tidak dapat diubah setelah dipakai; buat kupon baru")
	}

	updated, err := s.buildCoupon(req)
	if err != nil {
		return err
	}
	updated.ID = existing.ID
	updated.CreatedBy = existing.CreatedBy
	updated.CreatedAt = existing.CreatedAt
	if err := s.store.SaveCoupon(ctx, &updated); err != nil {
		return err
	}
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.CouponUpdate,
		TargetType: "coupon", TargetID: id, RequestID: actor.RequestID,
		Before: map[string]any{"value": existing.DiscountValue, "active": existing.IsActive},
		After:  map[string]any{"value": updated.DiscountValue, "active": updated.IsActive},
	})
}

func (s *adminBillingService) DisableCoupon(ctx context.Context, actor AdminActor, id, reason string) error {
	existing, err := s.store.FindCouponByID(ctx, id)
	if err != nil {
		return err
	}
	existing.IsActive = false
	if err := s.store.SaveCoupon(ctx, existing); err != nil {
		return err
	}
	// Kupon dihentikan, bukan dihapus: riwayat pemakaian harus utuh.
	return s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.CouponDisable,
		TargetType: "coupon", TargetID: id, Reason: reason, RequestID: actor.RequestID,
		Before: map[string]any{"is_active": true},
		After:  map[string]any{"is_active": false},
	})
}

func (s *adminBillingService) CreditSummary(ctx context.Context, actor AdminActor, userID string) (*dto.CreditSummary, error) {
	balance, err := s.store.CreditBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.CreditHistory(ctx, userID, 100)
	if err != nil {
		return nil, err
	}
	history := make([]dto.CreditEntryItem, 0, len(rows))
	for _, row := range rows {
		history = append(history, dto.CreditEntryItem{
			ID: row.ID, DeltaIDR: row.DeltaIDR, BalanceAfter: row.BalanceAfter,
			Kind: row.Kind, Reason: row.Reason, CreatedAt: dto.NewTimePair(row.CreatedAt),
		})
	}
	return &dto.CreditSummary{UserID: userID, Balance: balance, History: history}, nil
}

func (s *adminBillingService) AdjustCredit(ctx context.Context, actor AdminActor, userID string, req dto.CreditRequest) (*dto.CreditSummary, error) {
	if err := ValidateReason(audit.CreditAdjust, req.Reason); err != nil {
		return nil, err
	}
	// Memberi kredit ke akun sendiri berarti menambah hak bayar sendiri.
	if actor.ID == userID {
		return nil, validation("tidak dapat menyesuaikan kredit akun sendiri")
	}
	if req.AmountIDR == 0 {
		return nil, validation("jumlah kredit tidak boleh nol")
	}

	actorID := actor.ID
	if _, err := s.store.ApplyCredit(ctx, &repository.ApplyCreditInput{
		UserID: userID, DeltaIDR: req.AmountIDR, Kind: model.CreditKindManual,
		Reason: req.Reason, ActorID: &actorID,
	}); err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.CreditAdjust,
		TargetType: "user", TargetID: userID, Reason: req.Reason, RequestID: actor.RequestID,
		After: map[string]any{"delta_idr": req.AmountIDR},
	}); err != nil {
		return nil, err
	}
	return s.CreditSummary(ctx, actor, userID)
}

// buildCoupon memvalidasi masukan kupon dan menyusun modelnya.
func (s *adminBillingService) buildCoupon(req dto.CouponRequest) (model.Coupon, error) {
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		return model.Coupon{}, validation("kode kupon wajib diisi")
	}
	// Kupon persen di atas 100% akan membuat total negatif.
	if req.DiscountType == model.DiscountPercent && req.DiscountValue > 100 {
		return model.Coupon{}, validation("potongan persen tidak boleh melebihi 100")
	}
	if req.DiscountType != model.DiscountPercent && req.DiscountType != model.DiscountFixed {
		return model.Coupon{}, validation("jenis potongan tidak dikenal")
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		return model.Coupon{}, validation("tanggal mulai harus format RFC3339")
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		return model.Coupon{}, validation("tanggal akhir harus format RFC3339")
	}
	if !endsAt.After(startsAt) {
		return model.Coupon{}, validation("tanggal akhir harus setelah tanggal mulai")
	}

	maxPerUser := req.MaxPerUser
	if maxPerUser <= 0 {
		maxPerUser = 1
	}

	return model.Coupon{
		Code:              code,
		Description:       strings.TrimSpace(req.Description),
		DiscountType:      req.DiscountType,
		DiscountValue:     req.DiscountValue,
		MaxDiscountIDR:    req.MaxDiscountIDR,
		MinAmountIDR:      req.MinAmountIDR,
		AppliesToPlans:    strings.TrimSpace(req.AppliesToPlans),
		FirstPurchaseOnly: req.FirstPurchaseOnly,
		StartsAt:          startsAt.UTC(),
		EndsAt:            endsAt.UTC(),
		MaxRedemptions:    req.MaxRedemptions,
		MaxPerUser:        maxPerUser,
		IsActive:          true,
	}, nil
}

// toCouponItem menentukan status kupon dari waktu dan pemakaiannya.
func toCouponItem(row model.Coupon, used int64, now time.Time) dto.CouponItem {
	status := "active"
	switch {
	case !row.IsActive:
		status = "disabled"
	case now.After(row.EndsAt):
		status = "expired"
	case now.Before(row.StartsAt):
		status = "scheduled"
	case row.MaxRedemptions > 0 && used >= int64(row.MaxRedemptions):
		status = "exhausted"
	}

	return dto.CouponItem{
		ID: row.ID, Code: row.Code, Description: row.Description,
		DiscountType: row.DiscountType, DiscountValue: row.DiscountValue,
		MaxDiscountIDR: row.MaxDiscountIDR, MinAmountIDR: row.MinAmountIDR,
		AppliesToPlans: row.AppliesToPlans, FirstOnly: row.FirstPurchaseOnly,
		StartsAt: dto.NewTimePair(row.StartsAt), EndsAt: dto.NewTimePair(row.EndsAt),
		MaxRedemptions: row.MaxRedemptions, MaxPerUser: row.MaxPerUser,
		IsActive: row.IsActive, Redeemed: used, Status: status,
	}
}
