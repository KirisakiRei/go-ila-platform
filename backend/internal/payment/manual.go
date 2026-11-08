package payment

import (
	"context"
	"errors"
)

// manual adalah gateway tanpa penyedia daring.
//
// Dipakai saat belum ada penyedia yang dipilih, atau saat penyedia lama
// berhenti beroperasi. Tagihan tetap bisa diterbitkan dan dilunasi lewat
// pencatatan pembayaran manual di panel admin, jadi alur uang tetap utuh
// tanpa bergantung pada satu merek.
type manual struct{}

// NewManual membuat gateway tanpa penyedia daring.
func NewManual() Gateway {
	return manual{}
}

// Name mengembalikan nama gateway untuk disimpan di invoice.
func (manual) Name() string { return ProviderManual }

// CreateTransaction menolak pembuatan tautan bayar daring.
// Penolakan ini disengaja: lebih baik admin mencatat pembayaran daripada
// pengguna menerima tautan yang tidak bisa dibuka.
func (manual) CreateTransaction(_ context.Context, _ string, _ int64) (*Transaction, error) {
	return nil, errors.Join(ErrDisabled, errors.New("tanpa penyedia pembayaran daring; gunakan catat pembayaran manual"))
}

// VerifyWebhook selalu menolak: tidak ada webhook tanpa penyedia.
func (manual) VerifyWebhook(_ context.Context, _ map[string]string, _ []byte) error {
	return ErrDisabled
}

// ParseWebhook tidak pernah dipakai karena VerifyWebhook menolak lebih dulu.
func (manual) ParseWebhook(_ []byte) (*WebhookEvent, error) {
	return nil, ErrDisabled
}

// MapStatus menerjemahkan status yang dikenal gateway lain.
func (manual) MapStatus(raw string) string {
	switch raw {
	case StatusPaid, StatusPending, StatusExpired, StatusCancelled, StatusRefunded:
		return raw
	default:
		return StatusPending
	}
}

// RequestRefund mencatat bahwa refund dilakukan di luar sistem.
func (manual) RequestRefund(_ context.Context, transactionID string, _ int64, _ string) error {
	if transactionID == "" {
		return errors.New("transaksi tidak dikenal")
	}
	return nil
}
