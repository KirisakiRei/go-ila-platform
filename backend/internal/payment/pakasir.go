package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Pakasir adalah adapter gateway pembayaran lokal.
//
// Tanda tangan webhook memakai HMAC-SHA256 atas "<order_id>.<amount>.<status>"
// dengan kunci rahasia. Bila kunci kosong, gateway dianggap belum siap dan
// seluruh operasi ditolak, bukan dilewati.
type Pakasir struct {
	BaseURL    string
	Slug       string
	APIKey     string
	WebhookKey string
	HTTP       *http.Client
	Now        func() time.Time
}

// NewPakasir menyusun adapter dengan nilai bawaan yang aman.
func NewPakasir(baseURL, slug, apiKey, webhookKey string) *Pakasir {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://pakasir.com"
	}
	return &Pakasir{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Slug:       slug,
		APIKey:     apiKey,
		WebhookKey: webhookKey,
		HTTP:       &http.Client{Timeout: 15 * time.Second},
		Now:        func() time.Time { return time.Now().UTC() },
	}
}

// Name mengembalikan nama gateway.
func (p *Pakasir) Name() string {
	return "pakasir"
}

// Ready menandai apakah kunci sudah diisi.
func (p *Pakasir) Ready() bool {
	return strings.TrimSpace(p.Slug) != "" && strings.TrimSpace(p.APIKey) != ""
}

// CreateTransaction meminta tagihan baru ke Pakasir.
func (p *Pakasir) CreateTransaction(ctx context.Context, invoiceNumber string, amountIDR int64) (*Transaction, error) {
	if !p.Ready() {
		return nil, ErrDisabled
	}

	form := url.Values{}
	form.Set("project", p.Slug)
	form.Set("amount", strconv.FormatInt(amountIDR, 10))
	form.Set("order_id", invoiceNumber)
	form.Set("api_key", p.APIKey)

	endpoint := p.BaseURL + "/api/transactions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("menyusun permintaan pembayaran: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("menghubungi gateway pembayaran: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gateway menolak tagihan (status %d)", resp.StatusCode)
	}

	var payload struct {
		Payment struct {
			OrderID    string `json:"order_id"`
			Amount     int64  `json:"amount"`
			TotalFee   int64  `json:"total_fee"`
			PaymentURL string `json:"payment_url"`
			ExpiredAt  string `json:"expired_at"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("membaca balasan gateway: %w", err)
	}
	if payload.Payment.PaymentURL == "" {
		return nil, errors.New("gateway tidak mengembalikan URL pembayaran")
	}

	expiresAt := p.Now().Add(24 * time.Hour)
	if payload.Payment.ExpiredAt != "" {
		if parsed, err := time.Parse(time.RFC3339, payload.Payment.ExpiredAt); err == nil {
			expiresAt = parsed
		}
	}

	return &Transaction{
		Reference:  payload.Payment.OrderID,
		PaymentURL: payload.Payment.PaymentURL,
		ExpiresAt:  expiresAt,
	}, nil
}

// Signature menghitung tanda tangan webhook dari bagian yang disepakati.
func (p *Pakasir) Signature(orderID string, amount int64, status string) string {
	message := orderID + "." + strconv.FormatInt(amount, 10) + "." + status
	mac := hmac.New(sha256.New, []byte(p.WebhookKey))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhook memastikan webhook benar berasal dari Pakasir.
//
// Pakasir TIDAK mengirim header tanda tangan apa pun (lihat panduan resmi
// bagian D). Karena itu isi body tidak boleh dipercaya sendiri: setiap webhook
// diverifikasi balik ke Transaction Detail API dengan API Key. Hanya status
// yang dikonfirmasi gateway yang boleh dipakai menaikkan langganan.
//
// Tanpa API Key, gateway dianggap belum siap dan seluruh webhook ditolak.
func (p *Pakasir) VerifyWebhook(ctx context.Context, header map[string]string, body []byte) error {
	if strings.TrimSpace(p.APIKey) == "" || strings.TrimSpace(p.Slug) == "" {
		return ErrDisabled
	}

	event, err := p.ParseWebhook(body)
	if err != nil {
		return err
	}

	// Header opsional tetap dihormati bila suatu saat dipasang di depan
	// gateway: hanya dipakai untuk menolak lebih awal, bukan sebagai bukti.
	if signature := lookupHeader(header, "x-pakasir-signature"); signature != "" {
		expected := p.Signature(event.InvoiceNumber, event.AmountIDR, rawStatus(body))
		if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
			return errors.New("tanda tangan webhook tidak sah")
		}
	}

	verified, err := p.TransactionDetail(ctx, event.InvoiceNumber, event.AmountIDR)
	if err != nil {
		return fmt.Errorf("memverifikasi pembayaran ke gateway: %w", err)
	}
	// Nominal dan status harus sama dengan yang diklaim; selisih sekecil apa
	// pun membuat webhook ditolak.
	if verified.AmountIDR != event.AmountIDR {
		return errors.New("nominal webhook tidak cocok dengan gateway")
	}
	if p.MapStatus(verified.RawStatus) != event.Status {
		return errors.New("status webhook tidak cocok dengan gateway")
	}
	return nil
}

// TransactionDetail membaca status sebenarnya dari gateway.
func (p *Pakasir) TransactionDetail(ctx context.Context, orderID string, amountIDR int64) (*VerifiedTransaction, error) {
	query := url.Values{}
	query.Set("project", p.Slug)
	query.Set("amount", strconv.FormatInt(amountIDR, 10))
	query.Set("order_id", orderID)
	query.Set("api_key", p.APIKey)

	endpoint := p.BaseURL + "/api/transactiondetail?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("menyusun pemeriksaan transaksi: %w", err)
	}

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("menghubungi gateway pembayaran: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gateway menolak pemeriksaan (status %d)", resp.StatusCode)
	}

	var payload struct {
		Transaction struct {
			OrderID string `json:"order_id"`
			Amount  int64  `json:"amount"`
			Status  string `json:"status"`
		} `json:"transaction"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("membaca balasan gateway: %w", err)
	}
	if payload.Transaction.OrderID == "" {
		return nil, errors.New("transaksi tidak ditemukan di gateway")
	}

	return &VerifiedTransaction{
		OrderID:   payload.Transaction.OrderID,
		AmountIDR: payload.Transaction.Amount,
		RawStatus: payload.Transaction.Status,
	}, nil
}

// ParseWebhook mengurai isi webhook menjadi kejadian internal.
func (p *Pakasir) ParseWebhook(body []byte) (*WebhookEvent, error) {
	var payload struct {
		OrderID     string `json:"order_id"`
		Amount      int64  `json:"amount"`
		Status      string `json:"status"`
		CompletedAt string `json:"completed_at"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("membaca webhook: %w", err)
	}
	if payload.OrderID == "" {
		return nil, errors.New("webhook tanpa order_id")
	}

	paidAt := p.Now()
	if payload.CompletedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, payload.CompletedAt); err == nil {
			paidAt = parsed
		}
	}

	return &WebhookEvent{
		TransactionID: payload.OrderID,
		InvoiceNumber: payload.OrderID,
		AmountIDR:     payload.Amount,
		Status:        p.MapStatus(payload.Status),
		PaidAt:        paidAt,
	}, nil
}

// MapStatus memetakan status gateway ke status internal.
// Status tak dikenal dianggap pending, BUKAN paid: lebih baik tidak naik tier
// daripada naik karena salah tafsir (plans/08 §6.2).
func (p *Pakasir) MapStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "paid", "success", "settled", "completed":
		return StatusPaid
	case "expired":
		return StatusExpired
	case "failed", "cancelled", "canceled":
		return StatusCancelled
	case "refunded":
		return StatusRefunded
	default:
		return StatusPending
	}
}

// RequestRefund mencatat pengajuan refund. Pakasir tidak punya API refund
// otomatis, jadi ini hanya penanda bahwa refund dilakukan manual.
func (p *Pakasir) RequestRefund(ctx context.Context, transactionID string, amountIDR int64, reason string) error {
	if transactionID == "" {
		return errors.New("transaksi tidak dikenal")
	}
	return nil
}

// lookupHeader mencari header tanpa peduli besar-kecil huruf.
func lookupHeader(header map[string]string, key string) string {
	for name, value := range header {
		if strings.EqualFold(name, key) {
			return value
		}
	}
	return ""
}

// rawStatus mengambil status mentah dari body webhook untuk verifikasi.
func rawStatus(body []byte) string {
	var payload struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(body, &payload)
	return payload.Status
}
