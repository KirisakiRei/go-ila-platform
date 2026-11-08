package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/payment"
	"ila/backend/internal/repository"
)

// fakeBillingStore menyimpan invoice dan langganan di memori untuk pengujian.
type fakeBillingStore struct {
	invoices    map[string]*model.Invoice
	payments    map[string]bool
	sub         *model.SubscriptionRecord
	tierChanges []tierChange
	redemptions int
}

type tierChange struct {
	userID string
	plan   string
	start  time.Time
	end    time.Time
}

func newFakeBillingStore() *fakeBillingStore {
	return &fakeBillingStore{
		invoices: map[string]*model.Invoice{},
		payments: map[string]bool{},
	}
}

func (f *fakeBillingStore) FindPlan(ctx context.Context, code string) (*model.Plan, error) {
	return &model.Plan{Code: code, Name: code, PriceIDR: 666000, IsActive: true}, nil
}

func (f *fakeBillingStore) CreditBalance(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}

func (f *fakeBillingStore) ApplyCredit(ctx context.Context, in *repository.ApplyCreditInput) (int64, error) {
	return 0, nil
}

func (f *fakeBillingStore) NextInvoiceNumber(ctx context.Context, at time.Time) (string, error) {
	return "ILA-2026-00000" + string(rune('1'+len(f.invoices))), nil
}

func (f *fakeBillingStore) CreateInvoice(ctx context.Context, inv *model.Invoice) error {
	inv.ID = inv.Number
	f.invoices[inv.Number] = inv
	return nil
}

func (f *fakeBillingStore) FindInvoiceByNumber(ctx context.Context, number string) (*model.Invoice, error) {
	inv, ok := f.invoices[number]
	if !ok {
		return nil, errors.New("tidak ditemukan")
	}
	return inv, nil
}

func (f *fakeBillingStore) FindInvoiceByID(ctx context.Context, id string) (*model.Invoice, error) {
	for _, inv := range f.invoices {
		if inv.ID == id {
			return inv, nil
		}
	}
	return nil, errors.New("tidak ditemukan")
}

func (f *fakeBillingStore) UpdateInvoice(ctx context.Context, inv *model.Invoice) error {
	f.invoices[inv.Number] = inv
	return nil
}

func (f *fakeBillingStore) ListInvoices(ctx context.Context, userID string, limit int) ([]model.Invoice, error) {
	return nil, nil
}

func (f *fakeBillingStore) SavePayment(ctx context.Context, row *model.Payment) (bool, error) {
	key := row.Gateway + ":" + row.GatewayTxnID
	if f.payments[key] {
		// Duplikat: selesai tanpa efek, sama seperti indeks unik di database.
		return false, nil
	}
	f.payments[key] = true
	return true, nil
}

func (f *fakeBillingStore) SaveRedemption(ctx context.Context, r *model.CouponRedemption) error {
	f.redemptions++
	return nil
}

func (f *fakeBillingStore) ActiveSubscription(ctx context.Context, userID string) (*model.SubscriptionRecord, error) {
	return f.sub, nil
}

func (f *fakeBillingStore) UpsertSubscription(ctx context.Context, sub *model.SubscriptionRecord) error {
	f.sub = sub
	return nil
}

func (f *fakeBillingStore) ExtendTier(ctx context.Context, userID, plan string, start, end time.Time) error {
	f.tierChanges = append(f.tierChanges, tierChange{userID: userID, plan: plan, start: start, end: end})
	return nil
}

func (f *fakeBillingStore) DowngradeExpired(ctx context.Context, at time.Time) ([]string, error) {
	return nil, nil
}

func (f *fakeBillingStore) StartGrace(ctx context.Context, at time.Time) ([]string, error) {
	return nil, nil
}

func (f *fakeBillingStore) AddRefund(ctx context.Context, r *model.Refund) error { return nil }

// fakeCouponLookup selalu menemukan kupon yang diminta.
type fakeCouponLookup struct{}

func (fakeCouponLookup) FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error) {
	return &model.Coupon{Base: model.Base{ID: "coupon-1"}, Code: code}, nil
}

// verifiedUsers memenuhi syarat email terverifikasi, karena checkout menolak
// akun yang alamatnya belum terbukti.
type verifiedUsers struct{}

func (verifiedUsers) FindByID(_ context.Context, id string) (*model.User, error) {
	now := time.Now().UTC()
	return &model.User{Base: model.Base{ID: id}, EmailVerifiedAt: &now}, nil
}

func newTestBilling(store *fakeBillingStore, gw payment.Gateway) BillingService {
	pricing := NewPricingService(newFakePricingStore())
	return NewBillingService(verifiedUsers{}, store, pricing, fakeCouponLookup{}, gw, &fakeAdminAudit{}, nil)
}

func paidWebhook(t *testing.T, gw *payment.Pakasir, orderID string, amount int64, status string) (map[string]string, []byte) {
	t.Helper()
	body := []byte(`{"order_id":"` + orderID + `","amount":` + itoa(amount) + `,"status":"` + status + `"}`)
	sig := gw.Signature(orderID, amount, status)
	return map[string]string{"x-pakasir-signature": sig}, body
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	return out
}

// Webhook bertanda tangan salah tidak boleh diproses sama sekali.
func TestWebhookWrongSignatureRejected(t *testing.T) {
	store := newFakeBillingStore()
	gw := payment.NewPakasir("", "p", "k", "rahasia")
	svc := newTestBilling(store, gw)

	body := []byte(`{"order_id":"ILA-2026-000001","amount":666000,"status":"paid"}`)
	err := svc.HandleWebhook(context.Background(), map[string]string{"x-pakasir-signature": "palsu"}, body)
	if err == nil {
		t.Fatal("tanda tangan salah harus ditolak")
	}
	if len(store.tierChanges) != 0 {
		t.Fatal("tier tidak boleh berubah saat tanda tangan salah")
	}
}

// Webhook duplikat tidak menambah periode dua kali.
func TestWebhookDuplicateDoesNotExtendTwice(t *testing.T) {
	store := newFakeBillingStore()
	gw := payment.NewPakasir("", "p", "k", "rahasia")
	svc := newTestBilling(store, gw)

	// Invoice sudah terbit dan menunggu pembayaran.
	store.invoices["ILA-2026-000001"] = &model.Invoice{
		ID: "ILA-2026-000001", Number: "ILA-2026-000001", UserID: "u1",
		PlanCode: model.TierBasic, TotalIDR: 666000, Status: model.InvoicePending,
	}

	header, body := paidWebhook(t, gw, "ILA-2026-000001", 666000, "paid")
	if err := svc.HandleWebhook(context.Background(), header, body); err != nil {
		t.Fatalf("webhook sah harus berhasil: %v", err)
	}
	if len(store.tierChanges) != 1 {
		t.Fatalf("perpanjangan harus sekali, dapat %d", len(store.tierChanges))
	}

	firstEnd := store.tierChanges[0].end

	if err := svc.HandleWebhook(context.Background(), header, body); err != nil {
		t.Fatalf("webhook duplikat tidak boleh gagal: %v", err)
	}
	if len(store.tierChanges) != 1 {
		t.Fatalf("webhook duplikat tidak boleh menambah periode, dapat %d", len(store.tierChanges))
	}
	if !store.tierChanges[0].end.Equal(firstEnd) {
		t.Fatal("akhir periode tidak boleh bergeser karena duplikat")
	}
}

// Status tak dikenal tidak menaikkan tier.
func TestUnknownWebhookStatusStaysPending(t *testing.T) {
	store := newFakeBillingStore()
	gw := payment.NewPakasir("", "p", "k", "rahasia")
	svc := newTestBilling(store, gw)

	store.invoices["ILA-2026-000001"] = &model.Invoice{
		ID: "ILA-2026-000001", Number: "ILA-2026-000001", UserID: "u1",
		PlanCode: model.TierBasic, TotalIDR: 666000, Status: model.InvoicePending,
	}

	header, body := paidWebhook(t, gw, "ILA-2026-000001", 666000, "settlement_pending")
	if err := svc.HandleWebhook(context.Background(), header, body); err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if len(store.tierChanges) != 0 {
		t.Fatal("status tak dikenal tidak boleh menaikkan tier")
	}
	if store.invoices["ILA-2026-000001"].Status == model.InvoicePaid {
		t.Fatal("invoice tidak boleh bertanda lunas untuk status tak dikenal")
	}
}

// Kurang bayar tidak menaikkan tier.
func TestUnderpaymentRejected(t *testing.T) {
	store := newFakeBillingStore()
	gw := payment.NewPakasir("", "p", "k", "rahasia")
	svc := newTestBilling(store, gw)

	store.invoices["ILA-2026-000001"] = &model.Invoice{
		ID: "ILA-2026-000001", Number: "ILA-2026-000001", UserID: "u1",
		PlanCode: model.TierBasic, TotalIDR: 666000, Status: model.InvoicePending,
	}

	header, body := paidWebhook(t, gw, "ILA-2026-000001", 1000, "paid")
	if err := svc.HandleWebhook(context.Background(), header, body); err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if len(store.tierChanges) != 0 {
		t.Fatal("kurang bayar tidak boleh menaikkan tier")
	}
}

// Halaman sukses tidak punya jalur untuk mengubah tier: hanya webhook yang bisa.
func TestSuccessPageCannotActivate(t *testing.T) {
	store := newFakeBillingStore()
	store.invoices["ILA-2026-000001"] = &model.Invoice{
		Number: "ILA-2026-000001", UserID: "u1", PlanCode: model.TierBasic,
		TotalIDR: 666000, Status: model.InvoicePending,
	}
	svc := newTestBilling(store, payment.NewPakasir("", "p", "k", "rahasia"))

	// Antarmuka BillingService tidak menyediakan metode aktivasi manual.
	if _, err := svc.Subscription(context.Background(), "u1"); err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if len(store.tierChanges) != 0 {
		t.Fatal("membaca halaman tidak boleh mengubah tier")
	}
}
