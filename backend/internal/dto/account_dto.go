package dto

import "time"

// CreateAccountRequest mendaftarkan akun broker user.
// Kredensial dikirim sekali, dienkripsi, lalu tidak pernah dikembalikan.
type CreateAccountRequest struct {
	BrokerType  string  `json:"broker_type" binding:"required,oneof=matchtrader ctrader"`
	Label       string  `json:"label" binding:"max=60"`
	Mode        string  `json:"mode" binding:"omitempty,oneof=manual auto"`
	RiskMode    string  `json:"risk_mode" binding:"omitempty,oneof=percent"`
	RiskPercent float64 `json:"risk_percent"`
	IsDryRun    bool    `json:"is_dry_run"`

	// Token hanya untuk Match-Trader; disimpan terenkripsi.
	Token string `json:"token"`
	// BaseURL opsional bila broker memakai domain sendiri.
	BaseURL string `json:"base_url" binding:"omitempty,url"`
}

// UpdateAccountRequest mengubah pengaturan akun tanpa menyentuh kredensial.
type UpdateAccountRequest struct {
	Label       *string  `json:"label" binding:"omitempty,max=60"`
	Mode        *string  `json:"mode" binding:"omitempty,oneof=manual auto"`
	RiskPercent *float64 `json:"risk_percent"`
	IsActive    *bool    `json:"is_active"`
	IsDryRun    *bool    `json:"is_dry_run"`
}

// AccountResponse adalah tampilan akun broker untuk dashboard.
// Tidak pernah memuat token atau password.
type AccountResponse struct {
	ID                string     `json:"id"`
	BrokerType        string     `json:"broker_type"`
	Label             string     `json:"label"`
	ExternalAccountID string     `json:"external_account_id"`
	Mode              string     `json:"mode"`
	RiskPercent       float64    `json:"risk_percent"`
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
type SubscriptionResponse struct {
	ID              string `json:"id"`
	Symbol          string `json:"symbol"`
	Strategy        string `json:"strategy"`
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
