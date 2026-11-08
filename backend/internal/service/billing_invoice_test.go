package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// Tagihan yang lewat tenggat harus ditutup, jika tidak daftar dan laporan
// terus menghitung tagihan yang sudah mati.
func TestSweeperMenutupInvoiceKedaluwarsa(t *testing.T) {
	store := newFakeBillingStore()
	svc := NewBillingService(nil, store, nil, nil, nil, &fakeAdminAudit{}, nil)

	now := time.Now().UTC()
	store.invoices["ILA-1"] = &model.Invoice{
		ID: "inv-1", Number: "ILA-1", UserID: "u1", PlanCode: model.TierBasic,
		TotalIDR: 1000, Status: model.InvoicePending, ExpiresAt: now.Add(-time.Hour),
	}

	if _, err := svc.RunGraceSweep(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if got := store.invoices["ILA-1"].Status; got != model.InvoiceExpired {
		t.Fatalf("status = %q, ingin %q", got, model.InvoiceExpired)
	}
}

// Webhook status batal tidak boleh mengubah invoice yang sudah lunas:
// konfirmasi terlambat tidak boleh membatalkan hak yang sudah diberikan.
func TestWebhookBatalTidakMembatalkanInvoiceLunas(t *testing.T) {
	store := newFakeBillingStore()
	paidAt := time.Now().UTC()
	store.invoices["ILA-1"] = &model.Invoice{
		ID: "inv-1", Number: "ILA-1", UserID: "u1",
		Status: model.InvoicePaid, PaidAt: &paidAt,
	}

	svc := newFakeBillingService(store)
	if err := svc.closeInvoice(context.Background(), store.invoices["ILA-1"], "cancelled"); err != nil {
		t.Fatalf("closeInvoice: %v", err)
	}
	if got := store.invoices["ILA-1"].Status; got != model.InvoicePaid {
		t.Fatalf("status = %q, ingin tetap paid", got)
	}
}

// Invoice pending yang dibatalkan gateway harus ikut berubah.
func TestWebhookBatalMenutupInvoicePending(t *testing.T) {
	store := newFakeBillingStore()
	store.invoices["ILA-1"] = &model.Invoice{
		ID: "inv-1", Number: "ILA-1", UserID: "u1", Status: model.InvoicePending,
	}

	svc := newFakeBillingService(store)
	if err := svc.closeInvoice(context.Background(), store.invoices["ILA-1"], "cancelled"); err != nil {
		t.Fatalf("closeInvoice: %v", err)
	}
	if got := store.invoices["ILA-1"].Status; got != model.InvoiceCancelled {
		t.Fatalf("status = %q, ingin cancelled", got)
	}
}

// Pelunasan manual pada tagihan kedaluwarsa ditolak: periode yang tercatat
// di tagihan lama sudah tidak berlaku, jadi harus terbit tagihan baru.
func TestCatatBayarDitolakBilaInvoiceKedaluwarsa(t *testing.T) {
	store := newFakeBillingStore()
	now := time.Now().UTC()
	store.invoices["ILA-1"] = &model.Invoice{
		ID: "inv-1", Number: "ILA-1", UserID: "u1", PlanCode: model.TierBasic,
		TotalIDR: 666000, Status: model.InvoiceExpired, ExpiresAt: now.Add(-time.Hour),
	}

	// Uji ini hanya menyentuh baca invoice + tulis status.
	svc := &adminBillingService{
		store: invoiceStore{fake: store}, audit: &fakeAdminAudit{},
		now: func() time.Time { return now },
	}
	err := svc.MarkInvoicePaid(context.Background(), AdminActor{ID: "a1"}, "inv-1",
		dto.ManualPaymentRequest{AmountIDR: 666000, Note: "transfer bank terverifikasi"})
	if err == nil {
		t.Fatal("pelunasan invoice kedaluwarsa seharusnya ditolak")
	}
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("galat = %v, ingin CONFLICT", err)
	}
	if got := store.invoices["ILA-1"].Status; got != model.InvoiceExpired {
		t.Fatalf("status = %q, ingin tetap expired", got)
	}
}

// invoiceStore menyediakan bagian interface yang dipakai jalur pelunasan.
// Embed interface membuat sisanya kompilasi tanpa stub berulang.
type invoiceStore struct {
	billingStore
	fake *fakeBillingStore
}

func (s invoiceStore) FindInvoiceByID(ctx context.Context, id string) (*model.Invoice, error) {
	return s.fake.FindInvoiceByID(ctx, id)
}

func (s invoiceStore) UpdateInvoice(ctx context.Context, inv *model.Invoice) error {
	return s.fake.UpdateInvoice(ctx, inv)
}

func (s invoiceStore) SavePayment(ctx context.Context, row *model.Payment) (bool, error) {
	return s.fake.SavePayment(ctx, row)
}
