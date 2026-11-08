package payment_test

import (
	"context"
	"errors"
	"testing"

	"ila/backend/internal/config"
	"ila/backend/internal/payment"
)

// Pemilih gateway harus menolak nama yang tidak dikenal agar salah ketik
// konfigurasi berhenti saat start, bukan saat pengguna menekan "bayar".
func TestNewFromConfigMenolakProviderAsing(t *testing.T) {
	cfg := &config.Config{PaymentProvider: "midtrans"}
	if _, err := payment.NewFromConfig(cfg); err == nil {
		t.Fatal("provider tak dikenal seharusnya ditolak")
	}
}

// Provider kosong memakai Pakasir agar konfigurasi lama tetap berjalan.
func TestNewFromConfigDefaultPakasir(t *testing.T) {
	cfg := &config.Config{PaymentProvider: ""}
	gateway, err := payment.NewFromConfig(cfg)
	if err != nil {
		t.Fatalf("provider kosong: %v", err)
	}
	if gateway.Name() != payment.ProviderPakasir {
		t.Fatalf("gateway = %q, ingin pakasir", gateway.Name())
	}
}

// Mode manual dipakai saat belum ada penyedia: tagihan daring ditolak, tetapi
// refund tetap bisa dicatat karena dilakukan di luar sistem.
func TestManualMenolakPembayaranDaring(t *testing.T) {
	gateway, err := payment.NewFromConfig(&config.Config{PaymentProvider: payment.ProviderManual})
	if err != nil {
		t.Fatalf("provider manual: %v", err)
	}
	if gateway.Name() != payment.ProviderManual {
		t.Fatalf("gateway = %q", gateway.Name())
	}

	if _, err := gateway.CreateTransaction(context.Background(), "ILA-1", 1000); !errors.Is(err, payment.ErrDisabled) {
		t.Fatalf("CreateTransaction = %v, ingin ErrDisabled", err)
	}
	if err := gateway.VerifyWebhook(context.Background(), nil, []byte("{}")); !errors.Is(err, payment.ErrDisabled) {
		t.Fatalf("VerifyWebhook = %v, ingin ErrDisabled", err)
	}
	if err := gateway.RequestRefund(context.Background(), "trx-1", 1000, "uji"); err != nil {
		t.Fatalf("refund manual harus boleh: %v", err)
	}
}
