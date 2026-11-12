package matchtrader

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"ila/backend/internal/broker"
)

// transport membungkus satu base URL dengan bearer token akun.
// Token hanya hidup di memori dan tidak pernah dicetak ke log.
type transport struct {
	baseURL string
	token   string
	http    *http.Client
}

// httpTransport adalah pengaturan koneksi bersama seluruh akun Match-Trader.
//
// Koneksi di-pool dan handshake TLS dipakai ulang. Tanpa ini, setiap panggilan
// (cek akun, baca kutipan, kirim order) bisa membayar handshake baru 200-400 ms
// ke server broker di luar negeri - ongkos yang langsung mengurangi kecepatan
// entry, padahal datanya sendiri kecil.
var httpTransport = &http.Transport{
	Proxy:               http.ProxyFromEnvironment,
	MaxIdleConns:        64,
	MaxIdleConnsPerHost: 8,
	// Rapatkan koneksi idle agar tetap hangat tanpa menahan terlalu lama
	// sumber daya di VPS kecil.
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ForceAttemptHTTP2:     true,
}

func newTransport(baseURL, token string) *transport {
	return &transport{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout:   15 * time.Second,
			Transport: httpTransport,
		},
	}
}

// get memanggil endpoint GET dan men-decode body ke out.
func (t *transport) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL+path, nil)
	if err != nil {
		return err
	}
	t.setHeaders(req)
	return t.do(req, out)
}

// postJSON memanggil endpoint POST dengan body JSON.
func (t *transport) postJSON(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	t.setHeaders(req)
	return t.do(req, out)
}

func (t *transport) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Accept", "application/json")
}

// do mengeksekusi request dan menerjemahkan status error ke error domain.
func (t *transport) do(req *http.Request, out any) error {
	resp, err := t.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", broker.ErrNotConnected, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: HTTP %d", broker.ErrInvalidCredentials, resp.StatusCode)
	case resp.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: %s", broker.ErrPositionNotFound, pathOf(req))
	case resp.StatusCode >= 400:
		return translateStatus(resp.StatusCode, body)
	}

	if out == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

// translateStatus memetakan penolakan broker ke error yang bisa diputuskan runner.
func translateStatus(status int, body []byte) error {
	var env apiError
	_ = json.Unmarshal(body, &env)

	detail := env.Detail
	if detail == "" {
		detail = env.Title
	}
	if detail == "" {
		detail = string(body)
	}

	switch {
	case status == http.StatusUnprocessableEntity || status == http.StatusBadRequest:
		// Pesan margin dari broker tidak punya kode tetap; cek kata kunci.
		if containsFold(detail, "margin") || containsFold(detail, "not enough") {
			return fmt.Errorf("%w: %s", broker.ErrInsufficientMargin, detail)
		}
		return fmt.Errorf("%w: %s", broker.ErrUnsupported, detail)
	default:
		return fmt.Errorf("match-trader HTTP %d: %s", status, detail)
	}
}

func pathOf(req *http.Request) string {
	if req.URL == nil {
		return ""
	}
	return req.URL.Path
}

// containsFold mengecek substring tanpa memedulikan kapitalisasi.
func containsFold(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if equalFoldASCII(s[i:i+len(sub)], sub) {
			return true
		}
	}
	return false
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

var errEmptyToken = errors.New("token match-trader kosong")
