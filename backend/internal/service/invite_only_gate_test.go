package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/payment"
	"ila/backend/pkg/apperror"
)

// gateTetap adalah AccessGate dengan jawaban tetap.
type gateTetap bool

func (g gateTetap) InviteOnly(context.Context) bool { return bool(g) }

// TestCheckoutDitolakSaatModeUndangan: selama invite only tidak boleh ada
// tagihan baru, dan penolakan terjadi SEBELUM data apa pun dibaca.
func TestCheckoutDitolakSaatModeUndangan(t *testing.T) {
	store := newFakeBillingStore()
	svc := newTestBilling(store, payment.NewManual())
	SetBillingAccessGate(svc, gateTetap(true))

	_, err := svc.Checkout(context.Background(), "user-1", dto.CheckoutRequest{PlanCode: "basic"})

	if !errors.Is(err, ErrInviteOnly) {
		t.Fatalf("error = %v; ingin ErrInviteOnly", err)
	}
	if status, _ := apperror.MapHTTPStatus(err); status != 403 {
		t.Fatalf("status = %d; ingin 403", status)
	}
	if len(store.invoices) != 0 {
		t.Fatalf("invoice tetap dibuat padahal mode undangan menutup pembayaran: %d", len(store.invoices))
	}
}

// TestRedeemKreditDitolakSaatModeUndangan: kredit hanya berguna untuk membayar.
func TestRedeemKreditDitolakSaatModeUndangan(t *testing.T) {
	svc := newTestBilling(newFakeBillingStore(), payment.NewManual())
	SetBillingAccessGate(svc, gateTetap(true))

	if _, err := svc.RedeemCredit(context.Background(), "user-1", "HEMAT10"); !errors.Is(err, ErrInviteOnly) {
		t.Fatalf("error = %v; ingin ErrInviteOnly", err)
	}
}

// TestGerbangLenientSaatDibuka memastikan checkpoint lama tidak berubah bila
// admin membuka kembali platform.
func TestGerbangLenientSaatDibuka(t *testing.T) {
	svc := newTestBilling(newFakeBillingStore(), payment.NewManual())
	SetBillingAccessGate(svc, gateTetap(false))

	_, err := svc.Checkout(context.Background(), "user-1", dto.CheckoutRequest{PlanCode: "basic"})
	if errors.Is(err, ErrInviteOnly) {
		t.Fatal("gerbang terbuka tetapi checkout tetap ditolak sebagai invite only")
	}
}

// trialUsersEmailTerbukti memasok akun yang lolos syarat lama uji coba, sehingga
// yang diuji benar-benar gerbang undangan dan bukan penolakan verifikasi email.
type trialUsersEmailTerbukti struct{}

func (trialUsersEmailTerbukti) Create(context.Context, *model.User) error { return nil }
func (trialUsersEmailTerbukti) FindByID(_ context.Context, id string) (*model.User, error) {
	now := time.Now().UTC()
	return &model.User{Base: model.Base{ID: id}, EmailVerifiedAt: &now}, nil
}
func (trialUsersEmailTerbukti) FindByEmail(context.Context, string) (*model.User, error) {
	return nil, nil
}
func (trialUsersEmailTerbukti) FindByUsername(context.Context, string) (*model.User, error) {
	return nil, nil
}
func (trialUsersEmailTerbukti) FindByIdentifier(context.Context, string) (*model.User, error) {
	return nil, nil
}
func (trialUsersEmailTerbukti) FindByTelegramChatID(context.Context, string) (*model.User, error) {
	return nil, nil
}
func (trialUsersEmailTerbukti) UpdateLastLogin(context.Context, string, time.Time) error { return nil }
func (trialUsersEmailTerbukti) Update(context.Context, *model.User) error                { return nil }
func (trialUsersEmailTerbukti) RevokeSessions(context.Context, string) error             { return nil }
func (trialUsersEmailTerbukti) UpdatePassword(context.Context, string, string) error     { return nil }

// auditDiam adalah AuditRepository yang tidak menulis apa pun.
type auditDiam struct{}

func (auditDiam) Record(context.Context, *model.AuditLog) error { return nil }

// TestTrialDitolakSaatModeUndangan: uji coba adalah pintu masuk jalur berbayar,
// jadi ikut ditutup.
func TestTrialDitolakSaatModeUndangan(t *testing.T) {
	svc := NewTrialService(trialUsersEmailTerbukti{}, auditDiam{})
	SetTrialAccessGate(svc, gateTetap(true))

	_, err := svc.Start(context.Background(), "user-1")

	if !errors.Is(err, ErrInviteOnly) {
		t.Fatalf("error = %v; ingin ErrInviteOnly", err)
	}
}

// TestTrialBerjalanSaatGerbangTerbuka menjaga alur lama tetap utuh.
func TestTrialBerjalanSaatGerbangTerbuka(t *testing.T) {
	svc := NewTrialService(trialUsersEmailTerbukti{}, auditDiam{})
	SetTrialAccessGate(svc, gateTetap(false))

	if _, err := svc.Start(context.Background(), "user-1"); err != nil {
		t.Fatalf("uji coba ditolak padahal gerbang terbuka: %v", err)
	}
}
