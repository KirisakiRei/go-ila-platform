// Package payment membungkus gateway pembayaran agar kode lain tidak tahu
// merek gateway (plans/08 §6). Adapter yang tersedia: Pakasir.
package payment

import (
	"context"
	"errors"
	"time"
)

// Status internal. Nama gateway dipetakan ke nilai ini lebih dulu.
const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusExpired   = "expired"
	StatusCancelled = "cancelled"
	StatusRefunded  = "refunded"
)

// Transaction adalah hasil pembuatan tagihan di gateway.
type Transaction struct {
	Reference  string
	PaymentURL string
	ExpiresAt  time.Time
}

// WebhookEvent adalah isi webhook yang sudah diurai.
type WebhookEvent struct {
	TransactionID string
	InvoiceNumber string
	AmountIDR     int64
	Status        string
	PaidAt        time.Time
}

// Gateway adalah antarmuka penyedia pembayaran. Implementasinya bisa diganti
// tanpa menyentuh service billing.
type Gateway interface {
	Name() string
	// CreateTransaction membuat tagihan dan mengembalikan URL pembayaran.
	CreateTransaction(ctx context.Context, invoiceNumber string, amountIDR int64) (*Transaction, error)
	// VerifyWebhook memastikan permintaan benar berasal dari gateway.
	VerifyWebhook(header map[string]string, body []byte) error
	// ParseWebhook mengurai isi webhook menjadi kejadian yang bisa diproses.
	ParseWebhook(body []byte) (*WebhookEvent, error)
	// MapStatus memetakan status gateway ke status internal.
	MapStatus(raw string) string
	// RequestRefund mengajukan pengembalian dana secara manual.
	RequestRefund(ctx context.Context, transactionID string, amountIDR int64, reason string) error
}

// ErrDisabled menandai gateway yang belum dikonfigurasi.
var ErrDisabled = errors.New("gateway pembayaran belum dikonfigurasi")
