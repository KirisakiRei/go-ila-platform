package ctraderapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultAuthURL     = "https://id.ctrader.com/oauth/v2/auth"
	DefaultTokenURL    = "https://openapi.ctraderapi.com/apps/token"
	DefaultAccountsURL = "https://openapi.ctraderapi.com/apps/accounts"
)

type TokenResponse struct {
	AccessToken      string `json:"accessToken"`
	TokenType        string `json:"tokenType"`
	ExpiresIn        int    `json:"expiresIn"`
	RefreshToken     string `json:"refreshToken"`
	ErrorCode        string `json:"errorCode,omitempty"`
	ErrorDescription string `json:"errorDescription,omitempty"`
}

type AccountInfo struct {
	AccountID       int64  `json:"accountId"`
	TraderLogin     int64  `json:"traderLogin"`
	BrokerTitle     string `json:"brokerTitle"`
	DepositCurrency string `json:"depositCurrency"`
	Live            bool   `json:"live"`
}

type Client struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	redirectURI  string
	authURL      string
	tokenURL     string
	accountsURL  string
}

func NewClient(clientID, clientSecret, redirectURI string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		authURL:      DefaultAuthURL,
		tokenURL:     DefaultTokenURL,
		accountsURL:  DefaultAccountsURL,
	}
}

// BuildAuthURL membuat tautan otorisasi OAuth cTrader Spotware.
func (c *Client) BuildAuthURL(scope string) string {
	if scope == "" {
		scope = "trading"
	}
	v := url.Values{}
	v.Set("client_id", c.clientID)
	v.Set("redirect_uri", c.redirectURI)
	v.Set("scope", scope)
	return fmt.Sprintf("%s?%s", c.authURL, v.Encode())
}

// ExchangeToken menukar authorization code menjadi access token dan refresh token.
func (c *Client) ExchangeToken(ctx context.Context, code string) (*TokenResponse, error) {
	if c.clientID == "" || c.clientSecret == "" {
		return nil, errors.New("ctrader client_id atau client_secret belum dikonfigurasi")
	}

	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", strings.TrimSpace(code))
	params.Set("redirect_uri", c.redirectURI)
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)

	reqURL := fmt.Sprintf("%s?%s", c.tokenURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("membuat request token: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("menghubungi server token ctrader: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("membaca respons token: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("membedah respons token (status %d): %w", resp.StatusCode, err)
	}

	if tokenResp.ErrorCode != "" {
		return nil, fmt.Errorf("ctrader token error [%s]: %s", tokenResp.ErrorCode, tokenResp.ErrorDescription)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ctrader token gagal dengan HTTP %d: %s", resp.StatusCode, string(body))
	}
	if tokenResp.AccessToken == "" {
		return nil, errors.New("ctrader token tidak memuat access_token")
	}

	return &tokenResp, nil
}

// RefreshToken memperbarui access token menggunakan refresh token.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if c.clientID == "" || c.clientSecret == "" {
		return nil, errors.New("ctrader client_id atau client_secret belum dikonfigurasi")
	}

	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", strings.TrimSpace(refreshToken))
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)

	reqURL := fmt.Sprintf("%s?%s", c.tokenURL, params.Encode())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("membuat request refresh token: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("menghubungi server refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("membaca respons refresh token: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("membedah respons refresh token (status %d): %w", resp.StatusCode, err)
	}

	if tokenResp.ErrorCode != "" {
		return nil, fmt.Errorf("ctrader refresh token error [%s]: %s", tokenResp.ErrorCode, tokenResp.ErrorDescription)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ctrader refresh token gagal dengan HTTP %d: %s", resp.StatusCode, string(body))
	}
	if tokenResp.AccessToken == "" {
		return nil, errors.New("ctrader refresh tidak memuat access_token baru")
	}

	return &tokenResp, nil
}

// GetAccounts mengambil seluruh daftar akun trading cTrader milik user yang diotorisasi.
func (c *Client) GetAccounts(ctx context.Context, accessToken string) ([]AccountInfo, error) {
	reqURL := fmt.Sprintf("%s?token=%s", c.accountsURL, url.QueryEscape(strings.TrimSpace(accessToken)))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("membuat request akun ctrader: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("menghubungi server akun ctrader: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("membaca respons akun ctrader: %w", err)
	}

	// Periksa kemungkinan error object JSON
	var errResp struct {
		ErrorCode        string `json:"errorCode"`
		ErrorDescription string `json:"errorDescription"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.ErrorCode != "" {
		return nil, fmt.Errorf("ctrader accounts error [%s]: %s", errResp.ErrorCode, errResp.ErrorDescription)
	}

	var accounts []AccountInfo
	if err := json.Unmarshal(body, &accounts); err != nil {
		return nil, fmt.Errorf("membedah daftar akun ctrader (status %d): %w", resp.StatusCode, err)
	}

	return accounts, nil
}
