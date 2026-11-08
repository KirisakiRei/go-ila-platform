// Package runnerclient memanggil Runner dari backend-api.
//
// Runner tidak membuka port publik (plans/09 §5): hanya backend-api yang
// menjangkau alamatnya di jaringan container. Dipakai HTTP internal karena
// pemanggilnya satu proses, jadi latensi gRPC tidak memberi keuntungan.
package runnerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client adalah klien ke API internal Runner.
type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

// New membuat klien Runner.
func New(baseURL, secret string) *Client {
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

// Enabled menandai alamat Runner tersedia.
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != ""
}

type approveRequest struct {
	UserID    string `json:"user_id"`
	SignalID  string `json:"signal_id"`
	AccountID string `json:"account_id"`
}

type approveResponse struct {
	RejectReason string `json:"reject_reason"`
}

// Approve meminta Runner mengeksekusi sinyal atas nama user.
func (c *Client) Approve(ctx context.Context, userID, signalID, accountID string) (string, error) {
	var resp approveResponse

	err := c.post(ctx, "/internal/approve", approveRequest{
		UserID:    userID,
		SignalID:  signalID,
		AccountID: accountID,
	}, &resp)
	if err != nil {
		return "", err
	}

	return resp.RejectReason, nil
}

type skipRequest struct {
	UserID   string `json:"user_id"`
	SignalID string `json:"signal_id"`
}

// Skip meminta Runner menandai sinyal dilewati.
func (c *Client) Skip(ctx context.Context, userID, signalID string) error {
	return c.post(ctx, "/internal/skip", skipRequest{UserID: userID, SignalID: signalID}, nil)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("menyusun permintaan runner: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("membuat permintaan runner: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.secret != "" {
		req.Header.Set("X-Internal-Secret", c.secret)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("menghubungi runner: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("runner menolak permintaan: %s", resp.Status)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("membaca balasan runner: %w", err)
	}
	return nil
}

type revokeRequest struct {
	AccountID string `json:"account_id"`
}

// RevokeAccount meminta Runner memutus sesi broker akun yang dicabut.
// Kegagalan di sini tidak membatalkan pencabutan kredensial: sesi akan
// berakhir sendiri saat Runner restart.
func (c *Client) RevokeAccount(ctx context.Context, accountID string) error {
	return c.post(ctx, "/internal/account/revoke", revokeRequest{AccountID: accountID}, nil)
}

// ReloadAccount memaksa Runner membuka ulang sesi broker.
//
// Dipakai setelah kredensial diganti di dashboard: tanpa ini Runner tetap
// memakai sesi lama yang tokennya sudah tidak berlaku, sehingga akun sehat
// terlihat mati sampai Runner restart (plans/12 §3.4).
func (c *Client) ReloadAccount(ctx context.Context, accountID string) error {
	return c.post(ctx, "/internal/account/reload", revokeRequest{AccountID: accountID}, nil)
}
