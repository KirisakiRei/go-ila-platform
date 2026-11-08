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

// gatewayStub meniru Transaction Detail API Pakasir.
func gatewayStub(t *testing.T, amount int64, status string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/transactiondetail" {
			t.Errorf("path tidak dikenal: %s", r.URL.Path)
		}
		if r.URL.Query().Get("api_key") == "" {
			t.Error("api_key wajib ikut pada pemeriksaan status")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"transaction": map[string]any{
				"order_id": r.URL.Query().Get("order_id"),
				"amount":   amount,
				"status":   status,
			},
		})
	}))
}

// Webhook yang cocok dengan konfirmasi gateway diterima.
func TestWebhookDikonfirmasiGateway(t *testing.T) {
	server := gatewayStub(t, 666000, "completed")
	defer server.Close()

	gw := NewPakasir(server.URL, "proyek", "kunci", "")
	body := webhookBody("ILA-2026-000001", 666000, "paid")

	if err := gw.VerifyWebhook(t.Context(), nil, body); err != nil {
		t.Fatalf("webhook yang sah harus diterima: %v", err)
	}
}

// Tanpa API Key, webhook tidak bisa diverifikasi dan harus ditolak. Ini
// menggantikan "tanpa kunci rahasia" pada desain tanda tangan lokal.
func TestWebhookTanpaAPIKeyDitolak(t *testing.T) {
	gw := NewPakasir("", "proyek", "", "")
	body := webhookBody("ILA-2026-000001", 666000, "paid")

	if err := gw.VerifyWebhook(t.Context(), nil, body); err != ErrDisabled {
		t.Fatalf("tanpa API Key harus ErrDisabled, dapat %v", err)
	}
}

// Inti perbaikan: webhook palsu yang mengaku lunas ditolak karena gateway
// menyatakan transaksinya belum dibayar.
func TestWebhookPalsuDitolak(t *testing.T) {
	server := gatewayStub(t, 666000, "pending")
	defer server.Close()

	gw := NewPakasir(server.URL, "proyek", "kunci", "")
	body := webhookBody("ILA-2026-000001", 666000, "paid")

	if err := gw.VerifyWebhook(t.Context(), nil, body); err == nil {
		t.Fatal("webhook yang mengaku lunas padahal gateway belum lunas harus ditolak")
	}
}

// Nominal berbeda dari gateway berarti webhook dipalsukan atau tertukar.
func TestWebhookNominalBedaDitolak(t *testing.T) {
	server := gatewayStub(t, 1000, "completed")
	defer server.Close()

	gw := NewPakasir(server.URL, "proyek", "kunci", "")
	body := webhookBody("ILA-2026-000001", 666000, "paid")

	if err := gw.VerifyWebhook(t.Context(), nil, body); err == nil {
		t.Fatal("nominal yang tidak cocok harus ditolak")
	}
}

// Header tanda tangan tetap dihormati bila dipasang di depan gateway.
func TestHeaderTandaTanganBilaAda(t *testing.T) {
	server := gatewayStub(t, 666000, "completed")
	defer server.Close()

	gw := NewPakasir(server.URL, "proyek", "kunci", "rahasia")
	body := webhookBody("ILA-2026-000001", 666000, "paid")

	if err := gw.VerifyWebhook(t.Context(), map[string]string{"X-Pakasir-Signature": "palsu"}, body); err == nil {
		t.Fatal("tanda tangan header yang salah harus ditolak")
	}

	sig := gw.Signature("ILA-2026-000001", 666000, "paid")
	if err := gw.VerifyWebhook(t.Context(), map[string]string{"X-Pakasir-Signature": sig}, body); err != nil {
		t.Fatalf("tanda tangan header yang benar harus diterima: %v", err)
	}
}

// Gateway yang tidak terjangkau membuat webhook ditolak, bukan diterima
// diam-diam.
func TestGatewayTidakTerjangkauDitolak(t *testing.T) {
	gw := NewPakasir("http://127.0.0.1:1", "proyek", "kunci", "")
	body := webhookBody("ILA-2026-000001", 666000, "paid")

	if err := gw.VerifyWebhook(t.Context(), nil, body); err == nil {
		t.Fatal("verifikasi yang gagal tidak boleh dianggap sah")
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
