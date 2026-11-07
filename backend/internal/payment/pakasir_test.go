package payment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func webhookBody(orderID string, amount int64, status string) []byte {
	payload := map[string]any{
		"order_id":     orderID,
		"amount":       amount,
		"status":       status,
		"completed_at": "2026-11-05T10:00:00Z",
	}
	raw, _ := json.Marshal(payload)
	return raw
}

func TestWebhookSignatureValid(t *testing.T) {
	gw := NewPakasir("", "proyek", "kunci", "rahasia")
	body := webhookBody("ILA-2026-000001", 666000, "paid")
	sig := gw.Signature("ILA-2026-000001", 666000, "paid")

	if err := gw.VerifyWebhook(map[string]string{"X-Pakasir-Signature": sig}, body); err != nil {
		t.Fatalf("tanda tangan sah harus diterima: %v", err)
	}
}

func TestWebhookSignatureWrong(t *testing.T) {
	gw := NewPakasir("", "proyek", "kunci", "rahasia")
	body := webhookBody("ILA-2026-000001", 666000, "paid")

	err := gw.VerifyWebhook(map[string]string{"X-Pakasir-Signature": "palsu"}, body)
	if err == nil {
		t.Fatal("tanda tangan salah harus ditolak")
	}
}

func TestWebhookTamperedAmountRejected(t *testing.T) {
	gw := NewPakasir("", "proyek", "kunci", "rahasia")
	original := webhookBody("ILA-2026-000001", 666000, "paid")
	sig := gw.Signature("ILA-2026-000001", 666000, "paid")

	// Nominal diubah setelah tanda tangan dibuat.
	tampered := webhookBody("ILA-2026-000001", 1000, "paid")
	if err := gw.VerifyWebhook(map[string]string{"x-pakasir-signature": sig}, tampered); err == nil {
		t.Fatal("nominal yang diubah harus menggagalkan verifikasi")
	}
	_ = original
}

func TestWebhookMissingSecretDisabled(t *testing.T) {
	gw := NewPakasir("", "proyek", "kunci", "")
	body := webhookBody("ILA-2026-000001", 666000, "paid")

	err := gw.VerifyWebhook(map[string]string{"X-Pakasir-Signature": "apa saja"}, body)
	if err != ErrDisabled {
		t.Fatalf("tanpa kunci webhook harus ErrDisabled, dapat %v", err)
	}
}

func TestUnknownStatusIsPending(t *testing.T) {
	gw := NewPakasir("", "proyek", "kunci", "rahasia")

	// Status tak dikenal tidak boleh menjadi paid: lebih baik tidak naik tier.
	for _, raw := range []string{"", "unknown", "settlement_pending", "PAID?"} {
		if got := gw.MapStatus(raw); got == StatusPaid {
			t.Fatalf("status %q tidak boleh menjadi paid", raw)
		}
	}
	if gw.MapStatus("paid") != StatusPaid {
		t.Fatal("status paid harus dikenali")
	}
}

func TestCreateTransactionWithoutKeys(t *testing.T) {
	gw := NewPakasir("", "", "", "")
	if _, err := gw.CreateTransaction(t.Context(), "ILA-2026-000001", 1000); err != ErrDisabled {
		t.Fatalf("gateway tanpa kunci harus menolak, dapat %v", err)
	}
}

func TestCreateTransactionParsesURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("order_id") != "ILA-2026-000001" {
			t.Errorf("order_id harus diteruskan, dapat %q", r.FormValue("order_id"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"payment": map[string]any{
				"order_id":    "ILA-2026-000001",
				"payment_url": "https://pakasir.com/p/abc",
				"expired_at":  "2026-11-06T10:00:00Z",
			},
		})
	}))
	defer server.Close()

	gw := NewPakasir(server.URL, "proyek", "kunci", "rahasia")
	txn, err := gw.CreateTransaction(t.Context(), "ILA-2026-000001", 666000)
	if err != nil {
		t.Fatalf("tidak boleh gagal: %v", err)
	}
	if txn.PaymentURL == "" {
		t.Fatal("URL pembayaran harus terisi")
	}
}
