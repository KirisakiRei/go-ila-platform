package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/payment"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// BillingService mengurus checkout, webhook, dan masa tenggang.
//
// Aturan yang tidak boleh dilanggar: halaman sukses TIDAK menaikkan tier.
// Hanya webhook bertanda tangan sah yang boleh mengubah langganan.
type BillingService interface {
	Checkout(ctx context.Context, userID string, req dto.CheckoutRequest) (*dto.CheckoutResult, error)
	HandleWebhook(ctx context.Context, header map[string]string, body []byte) error
	Invoices(ctx context.Context, userID string) ([]dto.InvoiceItem, error)
	Subscription(ctx context.Context, userID string) (*dto.SubscriptionItem, error)
	RunGraceSweep(ctx context.Context) (int, error)
	RedeemCredit(ctx context.Context, userID string, code string) (*dto.CreditSummary, error)
	// ActivatePlan dipakai admin saat mencatat pembayaran manual: hak paket
	// diberikan lewat jalur yang sama dengan webhook agar periodenya konsisten.
	ActivatePlan(ctx context.Context, userID, planCode string, periodStart, periodEnd time.Time) error
}

// BillingStore adalah data yang dibutuhkan billing.
type BillingStore interface {
	FindPlan(ctx context.Context, code string) (*model.Plan, error)
	CreditBalance(ctx context.Context, userID string) (int64, error)
	ApplyCredit(ctx context.Context, input *repository.ApplyCreditInput) (int64, error)
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
}

// creditRedeemer menukar kode kredit menjadi saldo.
type creditRedeemer interface {
	Redeem(ctx context.Context, userID, code string, now time.Time) (int64, error)
}

type billingService struct {
	users   BillingUserReader
	store   BillingStore
	codes   creditRedeemer
	pricing PricingService
	coupons couponLookup
	gateway payment.Gateway
	audit   AdminAudit
	now     func() time.Time
}

// couponLookup membaca id kupon berdasarkan kode untuk dicatat pemakaiannya.
type couponLookup interface {
	FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error)
}

// NewBillingService merangkai billing.
// BillingUserReader membaca akun untuk memeriksa verifikasi email.
type BillingUserReader interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
}

func NewBillingService(users BillingUserReader, store BillingStore, pricing PricingService, coupons couponLookup, gateway payment.Gateway, recorder AdminAudit, codes creditRedeemer) BillingService {
	return &billingService{
		users: users,
		store: store, pricing: pricing, coupons: coupons, gateway: gateway, audit: recorder, codes: codes,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// Checkout membuat invoice dan meminta URL pembayaran.
func (s *billingService) Checkout(ctx context.Context, userID string, req dto.CheckoutRequest) (*dto.CheckoutResult, error) {
	// Pembayaran dikunci sampai email terbukti: tanpa itu, akun hasil salah
	// ketik akan menerima invoice yang tidak bisa dipulihkan (plans/17 §2.5).
	if s.users != nil {
		user, err := s.users.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if user.EmailVerifiedAt == nil {
			return nil, ErrEmailNotVerified
		}
	}

	quote, err := s.pricing.Quote(ctx, userID, req.PlanCode, req.CouponCode, req.UseCredit)
	if err != nil {
		return nil, err
	}

	now := s.now()
	// Periode dihitung sekali di sini dan dipakai lagi saat pembayaran tuntas,
	// sehingga angka yang tertera di invoice sama dengan hak yang diberikan.
	// Perpanjangan menyambung periode berjalan, bukan menimpanya.
	periodStart := now
	if sub, subErr := s.store.ActiveSubscription(ctx, userID); subErr == nil && sub != nil && sub.CurrentPeriodEnd.After(now) {
		periodStart = sub.CurrentPeriodEnd
	}
	periodEnd := periodStart.AddDate(0, 1, 0)
	expiresAt := now.Add(24 * time.Hour)

	number, err := s.store.NextInvoiceNumber(ctx, now)
	if err != nil {
		return nil, err
	}

	invoice := &model.Invoice{
		Number: number, UserID: userID, PlanCode: quote.PlanCode,
		AmountIDR: quote.Subtotal, DiscountIDR: quote.Discount, CreditIDR: quote.CreditUsed,
		TotalIDR: quote.Total, Status: model.InvoicePending, CouponCode: quote.CouponCode,
		PeriodStart: periodStart, PeriodEnd: periodEnd, ExpiresAt: expiresAt,
		Gateway: s.gateway.Name(), CreatedAt: now, UpdatedAt: now,
	}

	// Tagihan gratis (total 0) langsung lunas tanpa gateway.
	paymentURL := ""
	if quote.Total == 0 {
		invoice.Status = model.InvoicePaid
		invoice.PaidAt = &now
	} else {
		txn, err := s.gateway.CreateTransaction(ctx, number, quote.Total)
		if err != nil {
			// Pesan ke pengguna tetap umum, tetapi penyebabnya harus ikut
			// terbawa: tanpa itu kegagalan gateway tidak bisa ditelusuri
			// dari log (bentuk balasan gateway bisa berubah).
			return nil, errors.Join(
				apperror.ErrServiceUnavailable,
				errors.New("gateway pembayaran sedang tidak tersedia"),
				err,
			)
		}
		invoice.GatewayRef = txn.Reference
		invoice.ExpiresAt = txn.ExpiresAt
		paymentURL = txn.PaymentURL
	}

	if err := s.store.CreateInvoice(ctx, invoice); err != nil {
		return nil, err
	}

	// Kupon dicatat saat invoice terbit dengan nilai yang sudah dibekukan.
	if quote.CouponCode != "" {
		coupon, err := s.coupons.FindCouponByCode(ctx, quote.CouponCode)
		if err != nil {
			return nil, err
		}
		if err := s.store.SaveRedemption(ctx, &model.CouponRedemption{
			CouponID: coupon.ID, UserID: userID, InvoiceID: invoice.ID,
			DiscountIDR: quote.Discount, RedeemedAt: now,
		}); err != nil {
			return nil, err
		}
	}

	if quote.CreditUsed > 0 {
		invoiceID := invoice.ID
		if _, err := s.store.ApplyCredit(ctx, &repository.ApplyCreditInput{
			UserID: userID, DeltaIDR: -quote.CreditUsed, Kind: model.CreditKindUsed,
			InvoiceID: &invoiceID, Reason: "pemakaian kredit pada invoice " + number,
		}); err != nil {
			return nil, err
		}
	}

	if quote.Total == 0 {
		if err := s.activate(ctx, invoice, now); err != nil {
			return nil, err
		}
	}

	return &dto.CheckoutResult{
		InvoiceNumber: number,
		TotalIDR:      quote.Total,
		PaymentURL:    paymentURL,
		ExpiresAt:     dto.NewTimePair(invoice.ExpiresAt),
		Status:        invoice.Status,
	}, nil
}

// HandleWebhook memproses webhook dengan tujuh langkah berurutan (plans/08 §5.1).
func (s *billingService) HandleWebhook(ctx context.Context, header map[string]string, body []byte) error {
	// 1. Verifikasi tanda tangan lebih dulu: tidak ada pemrosesan sebelum sah.
	if err := s.gateway.VerifyWebhook(ctx, header, body); err != nil {
		if errors.Is(err, payment.ErrDisabled) {
			return errors.Join(apperror.ErrServiceUnavailable, errors.New("gateway belum dikonfigurasi"))
		}
		return errors.Join(apperror.ErrUnauthorized, errors.New("tanda tangan webhook tidak sah"))
	}

	// 2. Urai isi webhook.
	event, err := s.gateway.ParseWebhook(body)
	if err != nil {
		return errors.Join(apperror.ErrBadRequest, err)
	}

	// 3. Temukan invoice.
	invoice, err := s.store.FindInvoiceByNumber(ctx, event.InvoiceNumber)
	if err != nil {
		return err
	}

	// 4. Catat pembayaran. Duplikat tidak menambah periode dua kali.
	isNew, err := s.store.SavePayment(ctx, &model.Payment{
		InvoiceID: invoice.ID, Gateway: s.gateway.Name(),
		GatewayTxnID: event.TransactionID, AmountIDR: event.AmountIDR,
		Status: event.Status, SignatureValid: true,
		RawPayload: truncateRaw(body), ReceivedAt: s.now(),
	})
	if err != nil {
		return err
	}
	if !isNew {
		// Webhook yang sama sudah diproses: selesai tanpa efek.
		return nil
	}

	// 5. Status yang tidak dikenal tidak pernah dianggap lunas.
	// Status mati (kedaluwarsa/batal) dicerminkan ke invoice agar daftar
	// tagihan tidak bertentangan dengan kenyataan di gateway.
	switch event.Status {
	case payment.StatusPaid:
		// lanjut ke pemeriksaan nominal
	case payment.StatusExpired, payment.StatusCancelled:
		return s.closeInvoice(ctx, invoice, event.Status)
	default:
		return nil
	}

	// 6. Nominal harus cukup; kurang bayar tidak menaikkan tier.
	if event.AmountIDR < invoice.TotalIDR {
		return nil
	}

	// 7. Tandai lunas dan aktifkan langganan dalam satu alur.
	return s.activate(ctx, invoice, s.now())
}

// closeInvoice mencatat invoice yang mati di sisi gateway.
// Hanya invoice pending yang berubah: invoice lunas tidak boleh dibatalkan
// oleh webhook yang datang terlambat.
func (s *billingService) closeInvoice(ctx context.Context, invoice *model.Invoice, status string) error {
	if invoice.Status != model.InvoicePending {
		return nil
	}
	target := model.InvoiceCancelled
	if status == payment.StatusExpired {
		target = model.InvoiceExpired
	}
	invoice.Status = target
	invoice.UpdatedAt = s.now()
	if err := s.store.UpdateInvoice(ctx, invoice); err != nil {
		return err
	}
	return s.audit.Record(ctx, AuditEntry{
		ActorType: "system", Action: audit.InvoiceVoid,
		Reason:     "gateway melaporkan tagihan " + status,
		TargetType: "invoice", TargetID: invoice.ID,
		After: map[string]any{"number": invoice.Number, "status": target},
	})
}

// activate menandai invoice lunas dan memperpanjang langganan.
func (s *billingService) activate(ctx context.Context, invoice *model.Invoice, now time.Time) error {
	if invoice.Status == model.InvoicePaid {
		return nil
	}

	// Periode sudah dibekukan saat invoice terbit. Menghitung ulang di sini
	// akan membuat hak berbeda dari yang tertulis di invoice, terutama bila
	// pembayaran datang beberapa jam kemudian.
	start := invoice.PeriodStart
	if start.IsZero() {
		start = now
	}
	end := invoice.PeriodEnd
	if !end.After(start) {
		end = start.AddDate(0, 1, 0)
	}

	invoice.Status = model.InvoicePaid
	invoice.PaidAt = &now
	invoice.PeriodStart = start
	invoice.PeriodEnd = end
	invoice.UpdatedAt = now
	if err := s.store.UpdateInvoice(ctx, invoice); err != nil {
		return err
	}

	if err := s.store.ExtendTier(ctx, invoice.UserID, invoice.PlanCode, start, end); err != nil {
		return err
	}
	if err := s.store.UpsertSubscription(ctx, &model.SubscriptionRecord{
		UserID: invoice.UserID, PlanCode: invoice.PlanCode, Status: model.SubStatusActive,
		CurrentPeriodStart: start, CurrentPeriodEnd: end,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		return err
	}

	return s.audit.Record(ctx, AuditEntry{
		ActorType: "system", Action: audit.InvoicePaid,
		TargetType: "invoice", TargetID: invoice.ID,
		After: map[string]any{
			"number": invoice.Number, "total_idr": invoice.TotalIDR,
			"plan": invoice.PlanCode, "user_id": invoice.UserID,
		},
	})
}

// RedeemCredit menukar kode kredit. Penukaran ganda ditolak repository, bukan
// hanya diperiksa di sini, agar dua permintaan bersamaan tidak lolos.
func (s *billingService) RedeemCredit(ctx context.Context, userID string, code string) (*dto.CreditSummary, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if normalized == "" {
		return nil, validation("kode kredit wajib diisi")
	}
	if s.codes == nil {
		return nil, errors.Join(apperror.ErrServiceUnavailable, errors.New("kode kredit belum tersedia"))
	}
	if _, err := s.codes.Redeem(ctx, userID, normalized, s.now()); err != nil {
		return nil, err
	}

	balance, err := s.store.CreditBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.CreditSummary{UserID: userID, Balance: balance}, nil
}

// ActivatePlan memberi hak paket untuk periode yang sudah ditentukan.
// Dipakai admin setelah pembayaran manual dicatat; tidak ada pembayaran palsu
// yang dibuat, hanya rekaman langganan yang sama bentuknya dengan jalur webhook.
func (s *billingService) ActivatePlan(ctx context.Context, userID, planCode string, periodStart, periodEnd time.Time) error {
	start := periodStart
	if start.IsZero() {
		start = s.now()
	}
	end := periodEnd
	if !end.After(start) {
		end = start.AddDate(0, 1, 0)
	}
	if err := s.store.ExtendTier(ctx, userID, planCode, start, end); err != nil {
		return err
	}
	now := s.now()
	return s.store.UpsertSubscription(ctx, &model.SubscriptionRecord{
		UserID: userID, PlanCode: planCode, Status: model.SubStatusActive,
		CurrentPeriodStart: start, CurrentPeriodEnd: end,
		CreatedAt: now, UpdatedAt: now,
	})
}

// Invoices menampilkan tagihan milik pengguna.
func (s *billingService) Invoices(ctx context.Context, userID string) ([]dto.InvoiceItem, error) {
	rows, err := s.store.ListInvoices(ctx, userID, 50)
	if err != nil {
		return nil, err
	}
	items := make([]dto.InvoiceItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toInvoiceItem(row))
	}
	return items, nil
}

// Subscription menampilkan langganan aktif pengguna.
func (s *billingService) Subscription(ctx context.Context, userID string) (*dto.SubscriptionItem, error) {
	sub, err := s.store.ActiveSubscription(ctx, userID)
	if err != nil || sub == nil {
		return nil, nil
	}
	return &dto.SubscriptionItem{
		PlanCode:    sub.PlanCode,
		Status:      sub.Status,
		PeriodStart: dto.NewTimePair(sub.CurrentPeriodStart),
		PeriodEnd:   dto.NewTimePair(sub.CurrentPeriodEnd),
		CancelAtEnd: sub.CancelAtPeriodEnd,
	}, nil
}

// RunGraceSweep menjalankan pekerjaan harian uang: tagihan lewat tenggat
// ditutup lebih dulu, lalu langganan masuk masa tenggang dan yang habis
// diturunkan. Tagihan kedaluwarsa tidak mengubah hak paket.
// Posisi berjalan TIDAK disentuh: yang berubah hanya akses entry baru.
func (s *billingService) RunGraceSweep(ctx context.Context) (int, error) {
	now := s.now()

	expired, err := s.store.ExpireOverdue(ctx, now)
	if err != nil {
		return 0, err
	}
	started, err := s.store.StartGrace(ctx, now)
	if err != nil {
		return 0, err
	}
	downgraded, err := s.store.DowngradeExpired(ctx, now)
	if err != nil {
		return 0, err
	}
	return int(expired) + len(started) + len(downgraded), nil
}

func toInvoiceItem(row model.Invoice) dto.InvoiceItem {
	return dto.InvoiceItem{
		ID: row.ID, Number: row.Number, PlanCode: row.PlanCode,
		AmountIDR: row.AmountIDR, DiscountIDR: row.DiscountIDR, CreditIDR: row.CreditIDR,
		TotalIDR: row.TotalIDR, Status: row.Status, CouponCode: row.CouponCode,
		CreatedAt: dto.NewTimePair(row.CreatedAt), PaidAt: dto.NewTimePairPtr(row.PaidAt),
		ExpiresAt: dto.NewTimePair(row.ExpiresAt),
	}
}

func truncateRaw(body []byte) string {
	const max = 4000
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max])
}

// BillingSweeper menjalankan pekerjaan harian masa tenggang secara berkala.
//
// Tanpa ini, tier kedaluwarsa tidak pernah turun: user tetap Plus selamanya
// sampai ada admin yang mengubahnya manual (plans/08 §4).
type BillingSweeper struct {
	service BillingService
	logger  *slog.Logger
}

// NewBillingSweeper merangkai penyapu langganan.
func NewBillingSweeper(svc BillingService, logger *slog.Logger) *BillingSweeper {
	return &BillingSweeper{service: svc, logger: logger}
}

// Run menyapu sekali saat mulai, lalu setiap interval sampai konteks selesai.
func (b *BillingSweeper) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Hour
	}
	b.sweep(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.sweep(ctx)
		}
	}
}

func (b *BillingSweeper) sweep(ctx context.Context) {
	affected, err := b.service.RunGraceSweep(ctx)
	if err != nil {
		b.logger.Error("sapu masa tenggang gagal", "error", err.Error())
		return
	}
	if affected > 0 {
		b.logger.Info("masa tenggang diperbarui", "changed", affected)
	}
}
