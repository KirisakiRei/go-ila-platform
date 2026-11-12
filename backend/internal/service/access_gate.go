package service

import (
	"context"
	"errors"

	"ila/backend/internal/settings"
	"ila/backend/pkg/apperror"
)

// AccessGate memutuskan apakah pintu masuk mandiri masih dibuka.
//
// Selama platform berjalan mode undangan, tiga pintu ditutup sekaligus:
// pendaftaran mandiri, uji coba, dan pembayaran. Ketiganya adalah jalan masuk
// yang sama menuju akun berbayar, jadi satu gerbang, bukan tiga saklar.
type AccessGate interface {
	InviteOnly(ctx context.Context) bool
}

// ErrInviteOnly adalah penolakan seragam untuk seluruh pintu yang ditutup
// selama mode undangan.
var ErrInviteOnly = errors.Join(
	apperror.ErrForbidden,
	errors.New("Platform sedang invite only. Akses hanya lewat undangan admin."),
)

// billingGateSetter dipenuhi *billingService.
type billingGateSetter interface{ SetAccessGate(AccessGate) }

// SetBillingAccessGate memasang gerbang ke pembaca tagihan.
//
// Pemasangan lewat antarmuka kecil, bukan parameter konstruktor, agar
// pemanggil lama (termasuk cmd/sweeper dan pengujian) tidak perlu diubah.
func SetBillingAccessGate(b BillingService, gate AccessGate) {
	if setter, ok := b.(billingGateSetter); ok {
		setter.SetAccessGate(gate)
	}
}

// trialGateSetter dipenuhi *trialService.
type trialGateSetter interface{ SetAccessGate(AccessGate) }

// SetTrialAccessGate memasang gerbang ke layanan uji coba.
func SetTrialAccessGate(t TrialService, gate AccessGate) {
	if setter, ok := t.(trialGateSetter); ok {
		setter.SetAccessGate(gate)
	}
}

// FlagAccessGate membaca gerbang dari flag sistem.
type FlagAccessGate struct {
	flags *settings.Service
}

// NewFlagAccessGate merangkai gerbang dari pembaca flag.
//
// flags boleh nil: tanpa pembaca, gerbang menganggap pintu tertutup, sama
// seperti kegagalan membaca flag.
func NewFlagAccessGate(flags *settings.Service) *FlagAccessGate {
	return &FlagAccessGate{flags: flags}
}

// InviteOnly melaporkan apakah pendaftaran mandiri ditutup.
//
// Kegagalan membaca flag menghasilkan true (tertutup). Menutup pintu yang
// seharusnya terbuka hanya menunda satu pendaftaran; membuka pintu yang
// seharusnya tertutup mengotori data secara permanen.
func (g *FlagAccessGate) InviteOnly(ctx context.Context) bool {
	if g == nil || g.flags == nil {
		return true
	}
	return g.flags.Bool(ctx, settings.KeyInviteOnly, true)
}
