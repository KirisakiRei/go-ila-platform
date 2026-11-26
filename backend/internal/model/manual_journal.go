package model

import "time"

type ManualJournalCheck struct {
	Label   string `json:"label"`
	Checked bool   `json:"checked"`
}

type ManualJournalAccount struct {
	Base
	UserID         string   `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Name           string   `json:"name"`
	Broker         string   `json:"broker"`
	Currency       string   `gorm:"type:varchar(3);not null" json:"currency"`
	InitialCapital string   `gorm:"type:varchar(32);not null" json:"initial_capital"`
	InitialDate    string   `gorm:"type:varchar(10);not null" json:"initial_date"`
	Notes          string   `json:"notes"`
	Archived       bool     `json:"archived"`
	Checklist      []string `gorm:"serializer:json;type:text" json:"checklist"`
	// Computed response fields are never persisted; preserves DTO/model conversion for existing consumers.
	CurrentRealizedBalance *string `gorm:"-" json:"current_realized_balance"`
	BalanceComplete        bool    `gorm:"-" json:"balance_complete"`
	UnknownClosedPNL       int     `gorm:"-" json:"unknown_closed_pnl"`
}

func (ManualJournalAccount) TableName() string { return "manual_journal_accounts" }

type ManualJournalCapital struct {
	Base
	UserID    string `gorm:"type:varchar(36);not null;index" json:"user_id"`
	AccountID string `gorm:"type:varchar(36);not null;index" json:"account_id"`
	Kind      string `json:"kind"`
	Amount    string `gorm:"type:varchar(32);not null" json:"amount"`
	Date      string `gorm:"type:varchar(10);not null;index" json:"date"`
	Notes     string `json:"notes"`
}

func (ManualJournalCapital) TableName() string { return "manual_journal_capital" }

type ManualJournalEntry struct {
	Base
	UserID          string               `gorm:"type:varchar(36);not null;index" json:"user_id"`
	AccountID       string               `gorm:"type:varchar(36);not null;index" json:"account_id"`
	Status          string               `gorm:"index" json:"status"`
	Asset           string               `json:"asset"`
	Side            string               `json:"side"`
	OpenedAt        *time.Time           `json:"opened_at"`
	ClosedAt        *time.Time           `gorm:"index" json:"closed_at"`
	EntryPrice      *string              `gorm:"type:varchar(32)" json:"entry_price"`
	StopLoss        *string              `gorm:"type:varchar(32)" json:"stop_loss"`
	TakeProfit      *string              `gorm:"type:varchar(32)" json:"take_profit"`
	ExitPrice       *string              `gorm:"type:varchar(32)" json:"exit_price"`
	Size            *string              `gorm:"type:varchar(32)" json:"size"`
	SizeUnit        string               `json:"size_unit"`
	NetPNL          *string              `gorm:"type:varchar(32)" json:"net_pnl"`
	RiskAmount      *string              `gorm:"type:varchar(32)" json:"risk_amount"`
	CapitalSnapshot *string              `gorm:"type:varchar(32)" json:"capital_snapshot"`
	Setup           string               `json:"setup"`
	Tags            []string             `gorm:"serializer:json;type:text" json:"tags"`
	Emotions        []string             `gorm:"serializer:json;type:text" json:"emotions"`
	Mistakes        []string             `gorm:"serializer:json;type:text" json:"mistakes"`
	Session         string               `json:"session"`
	Checklist       []ManualJournalCheck `gorm:"serializer:json;type:text" json:"checklist"`
	Notes           string               `json:"notes"`
	TradingViewURLs []string             `gorm:"serializer:json;type:text" json:"tradingview_urls"`
}

func (ManualJournalEntry) TableName() string { return "manual_journal_entries" }

type ManualJournalScreenshot struct {
	Base
	UserID    string `gorm:"type:varchar(36);not null;index" json:"-"`
	EntryID   string `gorm:"type:varchar(36);not null;index" json:"entry_id"`
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	StoredAs  string `json:"-"`
}

func (ManualJournalScreenshot) TableName() string { return "manual_journal_screenshots" }

// Import tokens persist across processes; conditional consumption and inserts share one transaction.
type ManualJournalImportToken struct {
	Base
	UserID      string     `gorm:"type:varchar(36);not null;index" json:"-"`
	PayloadHash string     `gorm:"type:varchar(64);not null" json:"-"`
	ExpiresAt   time.Time  `gorm:"not null;index" json:"expires_at"`
	ConsumedAt  *time.Time `json:"-"`
}

func (ManualJournalImportToken) TableName() string { return "manual_journal_import_tokens" }

type ManualJournalFilter struct {
	Page      int    `form:"page"`
	PerPage   int    `form:"per_page"`
	AccountID string `form:"account_id"`
	Search    string `form:"search"`
	From      string `form:"from"`
	To        string `form:"to"`
	Status    string `form:"status"`
	Asset     string `form:"asset"`
	Side      string `form:"side"`
	Tag       string `form:"tag"`
	Emotion   string `form:"emotion"`
	Mistake   string `form:"mistake"`
	Session   string `form:"session"`
	Setup     string `form:"setup"`
	Archived  *bool  `form:"archived"`
}
