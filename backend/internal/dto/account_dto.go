package dto

import "time"

// CreateAccountRequest mendaftarkan akun broker user.
// Kredensial dikirim sekali, dienkripsi, lalu tidak pernah dikembalikan.
type CreateAccountRequest struct {
	BrokerType  string  `json:"broker_type" binding:"required,oneof=matchtrader ctrader"`
	Label       string  `json:"label" binding:"max=60"`
	Mode        string  `json:"mode" binding:"omitempty,oneof=manual auto"`
	RiskMode    string  `json:"risk_mode" binding:"omitempty,oneof=percent lot"`
	RiskPercent float64 `json:"risk_percent"`
	// LotSize dipakai bila RiskMode=lot. Kosong berarti bawaan 0.01.
	LotSize  float64 `json:"lot_size"`
	IsDryRun bool    `json:"is_dry_run"`

	// Token hanya untuk Match-Trader; disimpan terenkripsi.
	Token string `json:"token"`
	// BaseURL opsional bila broker memakai domain sendiri.
	BaseURL string `json:"base_url" binding:"omitempty,url"`
}

// UpdateAccountRequest mengubah pengaturan akun tanpa menyentuh kredensial.
type UpdateAccountRequest struct {
	Label       *string  `json:"label" binding:"omitempty,max=60"`
	Mode        *string  `json:"mode" binding:"omitempty,oneof=manual auto"`
	RiskMode    *string  `json:"risk_mode" binding:"omitempty,oneof=percent lot"`
	RiskPercent *float64 `json:"risk_percent"`
	// LotSize dipakai bila RiskMode=lot.
	LotSize  *float64 `json:"lot_size"`
	IsActive *bool    `json:"is_active"`
	IsDryRun *bool    `json:"is_dry_run"`
	// Token bersifat opsional: diisi hanya saat kredensial broker berganti
	// (mis. token MatchTrader kedaluwarsa). Kosong berarti token lama dipakai.
	Token   *string `json:"token" binding:"omitempty,max=4000"`
	BaseURL *string `json:"base_url" binding:"omitempty,max=200"`
}

// AccountResponse adalah tampilan akun broker untuk dashboard.
// Tidak pernah memuat token atau password.
type AccountResponse struct {
	ID                string     `json:"id"`
	BrokerType        string     `json:"broker_type"`
	Label             string     `json:"label"`
	ExternalAccountID string     `json:"external_account_id"`
	Mode              string     `json:"mode"`
	RiskMode          string     `json:"risk_mode"`
	RiskPercent       float64    `json:"risk_percent"`
	LotSize           float64    `json:"lot_size"`
	IsActive          bool       `json:"is_active"`
	IsFrozen          bool       `json:"is_frozen"`
	IsDryRun          bool       `json:"is_dry_run"`
	HasCredential     bool       `json:"has_credential"`
	LastVerifiedAt    *time.Time `json:"last_verified_at"`
	LastError         string     `json:"last_error"`

	// Telemetry terakhir dari Runner.
	Connected    bool       `json:"connected"`
	Balance      float64    `json:"balance"`
	Equity       float64    `json:"equity"`
	FreeMargin   float64    `json:"free_margin"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
}

// CreateSubscriptionRequest mengaktifkan satu pasangan (symbol, strategy).
type CreateSubscriptionRequest struct {
	Symbol          string `json:"symbol" binding:"required,min=3,max=20"`
	Strategy        string `json:"strategy" binding:"omitempty,oneof=ila-2.0 ila-3.0"`
	BrokerAccountID string `json:"broker_account_id"`
}

// SubscriptionResponse adalah tampilan langganan simbol user.
//
// Metode analisis tidak lagi dilaporkan di sini: satu pair dipantau seluruh
// metode, dan pilihan per akun dibaca lewat matriks saklar
// GET /pairs/:symbol/strategies.
type SubscriptionResponse struct {
	ID              string `json:"id"`
	Symbol          string `json:"symbol"`
	BrokerAccountID string `json:"broker_account_id,omitempty"`
	IsActive        bool   `json:"is_active"`
}

// StrategyResponse menjelaskan ketersediaan satu strategi bagi user.
type StrategyResponse struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Available   bool   `json:"available"`
	Allowed     bool   `json:"allowed"`
	Selected    bool   `json:"selected"`
	Description string `json:"description"`
}

// CTraderAuthURLResponse mengembalikan URL otorisasi cTrader OAuth.
type CTraderAuthURLResponse struct {
	AuthURL string `json:"auth_url"`
}

// CTraderCallbackRequest memproses authorization code dari cTrader.
type CTraderCallbackRequest struct {
	Code        string  `json:"code" binding:"required"`
	Mode        string  `json:"mode" binding:"omitempty,oneof=manual auto"`
	RiskMode    string  `json:"risk_mode" binding:"omitempty,oneof=percent lot"`
	RiskPercent float64 `json:"risk_percent"`
	LotSize     float64 `json:"lot_size"`
}

// CTraderCallbackResponse mengembalikan ringkasan akun cTrader yang terhubung.
type CTraderCallbackResponse struct {
	TotalConnected int               `json:"total_connected"`
	Accounts       []AccountResponse `json:"accounts"`
}
