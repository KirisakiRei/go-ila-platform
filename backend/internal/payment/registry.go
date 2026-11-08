package payment

import (
	"fmt"
	"strings"

	"ila/backend/internal/config"
)

// Nama adapter yang dikenali. Nilai ini dipakai di PAYMENT_PROVIDER.
const (
	ProviderPakasir = "pakasir"
	// ProviderManual menonaktifkan pembayaran daring: tagihan tetap bisa
	// diterbitkan, lalu dilunasi lewat "catat pembayaran manual" di panel admin.
	// Ini juga jalur aman bila penyedia lama berhenti beroperasi.
	ProviderManual = "manual"
)

// NewFromConfig memilih adapter gateway berdasarkan PAYMENT_PROVIDER.
//
// Service billing tidak tahu merek gateway: mengganti penyedia cukup dengan
// menambah adapter baru di paket ini dan mendaftarkannya di sini. Nilai yang
// tidak dikenal mengembalikan galat; pemanggil memutuskan jatuh ke manual.
func NewFromConfig(cfg *config.Config) (Gateway, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.PaymentProvider)) {
	case "", ProviderPakasir:
		return NewPakasir(cfg.PakasirBaseURL, cfg.PakasirSlug, cfg.PakasirAPIKey, cfg.PakasirWebhookKey), nil
	case ProviderManual:
		return NewManual(), nil
	default:
		return nil, fmt.Errorf("PAYMENT_PROVIDER %q belum dikenal", cfg.PaymentProvider)
	}
}
