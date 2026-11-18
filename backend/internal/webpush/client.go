package webpush

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

var (
	// ErrSubscriptionExpired menandakan langganan push sudah dicabut atau
	// kedaluwarsa di sisi browser/OS (HTTP 404 atau 410), sehingga barisnya
	// wajib dihapus dari database.
	ErrSubscriptionExpired = errors.New("langganan Web Push sudah kedaluwarsa atau dicabut")
)

// Subscription memuat alamat tujuan dan kunci enkripsi perangkat pengguna.
type Subscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// SendOptions mengatur metadata pengiriman RFC 8030.
type SendOptions struct {
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string
	TTLSeconds      int
	Urgency         string // very-low | low | normal | high
	Topic           string
	// SkipHostAllowlist HANYA dipakai pada pengujian lokal httptest TLS.
	SkipHostAllowlist bool
}

// Client mengirim pesan Web Push melalui koneksi HTTP/2 atau HTTP/1.1 yang dibatasi.
type Client struct {
	httpClient *http.Client
}

// NewClient membuat pengirim Web Push dengan batas waktu dan pool koneksi hemat memori.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        8,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 8 * time.Second,
		ForceAttemptHTTP2:   true,
	}
	return &Client{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// WithHTTPClient mengganti http.Client internal (dipakai untuk pengujian).
func (c *Client) WithHTTPClient(hc *http.Client) *Client {
	if hc != nil {
		c.httpClient = hc
	}
	return c
}

// Send mengenkripsi payload dan mengirimkannya ke Push Service perangkat.
func (c *Client) Send(ctx context.Context, sub Subscription, payload []byte, opts SendOptions) (int, error) {
	if !opts.SkipHostAllowlist {
		if err := ValidateEndpoint(sub.Endpoint); err != nil {
			return 0, err
		}
	}

	body, err := EncryptPayload(payload, sub.P256dh, sub.Auth)
	if err != nil {
		return 0, err
	}

	authHeader, err := BuildVAPIDHeader(
		sub.Endpoint,
		opts.VAPIDSubject,
		opts.VAPIDPublicKey,
		opts.VAPIDPrivateKey,
		time.Now().UTC(),
	)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("menyusun request push: %w", err)
	}

	ttl := opts.TTLSeconds
	if ttl <= 0 {
		ttl = 3600
	}
	urgency := opts.Urgency
	if urgency == "" {
		urgency = "high"
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("TTL", strconv.Itoa(ttl))
	req.Header.Set("Urgency", urgency)
	if opts.Topic != "" && len(opts.Topic) <= 32 {
		req.Header.Set("Topic", opts.Topic)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("mengirim ke %s: %w", RedactEndpoint(sub.Endpoint), err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return resp.StatusCode, ErrSubscriptionExpired
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("gateway %s menolak dengan HTTP %d", RedactEndpoint(sub.Endpoint), resp.StatusCode)
	}

	return resp.StatusCode, nil
}
