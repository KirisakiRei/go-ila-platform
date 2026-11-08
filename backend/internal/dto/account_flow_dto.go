package dto

import "time"

// ForgotPasswordRequest meminta tautan reset tanpa membocorkan status email.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=150"`
}

// ResetPasswordRequest menukar token menjadi kata sandi baru.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required,max=200"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

// VerifyEmailRequest memakai tautan verifikasi.
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required,max=200"`
}

// OnboardingState adalah posisi terakhir alur mulai bagi pengguna baru.
type OnboardingState struct {
	Step             int             `json:"step"`
	Done             bool            `json:"done"`
	TotalSteps       int             `json:"total_steps"`
	Steps            []string        `json:"steps"`
	StepDone         map[string]bool `json:"step_done"`
	CompletedCount   int             `json:"completed_count"`
	HasBroker        bool            `json:"has_broker"`
	Verified         bool            `json:"email_verified"`
	TelegramOK       bool            `json:"telegram_linked"`
	StrategySelected bool            `json:"strategy_selected"`
	RiskConfigured   bool            `json:"risk_configured"`
	NextAction       string          `json:"next_action"`
}

// OnboardingRequest mencatat langkah yang sudah diselesaikan.
type OnboardingRequest struct {
	Step int  `json:"step" binding:"min=0,max=5"`
	Done bool `json:"done"`
}

// SubscriptionBanner menjelaskan keadaan langganan pada dashboard.
// `state` bernilai: normal, ending, grace, expired.
type SubscriptionBanner struct {
	State     string     `json:"state"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	DaysLeft  int        `json:"days_left"`
	Level     string     `json:"level"`
}
