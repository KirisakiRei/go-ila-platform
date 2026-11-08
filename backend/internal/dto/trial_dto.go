package dto

import "time"

// TrialResponse melaporkan status uji coba 7 hari.
// DryRunOnly selalu true: trial tidak pernah mengeksekusi order nyata.
type TrialResponse struct {
	Used           bool       `json:"used"`
	Active         bool       `json:"active"`
	DurationDays   int        `json:"duration_days"`
	DaysRemaining  int        `json:"days_remaining"`
	DryRunOnly     bool       `json:"dry_run_only"`
	TrialStartedAt *time.Time `json:"trial_started_at,omitempty"`
	TrialExpiresAt *time.Time `json:"trial_expires_at,omitempty"`
	// Tier selama trial selalu Basic; ditampilkan agar user tahu batasnya.
	Tier string `json:"tier"`
	// AccountLimit adalah jumlah akun broker yang boleh dihubungkan.
	AccountLimit int `json:"account_limit"`
}
