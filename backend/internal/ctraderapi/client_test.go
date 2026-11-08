package ctraderapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildAuthURL(t *testing.T) {
	c := NewClient("client-123", "secret-456", "https://app.test/cb")
	url := c.BuildAuthURL("trading")
	expected := "https://id.ctrader.com/oauth/v2/auth?client_id=client-123&redirect_uri=https%3A%2F%2Fapp.test%2Fcb&scope=trading"
	if url != expected {
		t.Errorf("got %s, want %s", url, expected)
	}
}

func TestExchangeToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("grant_type") != "authorization_code" {
			t.Errorf("unexpected grant_type: %s", q.Get("grant_type"))
		}
		if q.Get("code") != "test-code" {
			t.Errorf("unexpected code: %s", q.Get("code"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"accessToken": "acc-token-123",
			"tokenType": "bearer",
			"expiresIn": 2592000,
			"refreshToken": "ref-token-456"
		}`))
	}))
	defer server.Close()

	c := NewClient("client-123", "secret-456", "https://app.test/cb")
	c.tokenURL = server.URL

	resp, err := c.ExchangeToken(context.Background(), "test-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "acc-token-123" {
		t.Errorf("got accessToken %s, want acc-token-123", resp.AccessToken)
	}
	if resp.RefreshToken != "ref-token-456" {
		t.Errorf("got refreshToken %s, want ref-token-456", resp.RefreshToken)
	}
}

func TestGetAccounts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("token") != "test-token" {
			t.Errorf("unexpected token query: %s", r.URL.Query().Get("token"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{
				"accountId": 49366554,
				"traderLogin": 10131354,
				"brokerTitle": "IC Markets",
				"depositCurrency": "USD",
				"live": false
			}
		]`))
	}))
	defer server.Close()

	c := NewClient("client-123", "secret-456", "https://app.test/cb")
	c.accountsURL = server.URL

	accounts, err := c.GetAccounts(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(accounts) != 1 {
		t.Fatalf("got %d accounts, want 1", len(accounts))
	}
	if accounts[0].AccountID != 49366554 {
		t.Errorf("got accountId %d, want 49366554", accounts[0].AccountID)
	}
	if accounts[0].TraderLogin != 10131354 {
		t.Errorf("got traderLogin %d, want 10131354", accounts[0].TraderLogin)
	}
	if accounts[0].Live {
		t.Errorf("expected live to be false")
	}
}
