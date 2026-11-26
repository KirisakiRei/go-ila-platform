package dto

import (
	"ila/backend/internal/model"
	"time"
)

type ManualJournalAccountInput struct {
	Name           string   `json:"name"`
	Broker         string   `json:"broker"`
	Currency       string   `json:"currency"`
	InitialCapital string   `json:"initial_capital"`
	InitialDate    string   `json:"initial_date"`
	Notes          string   `json:"notes"`
	Archived       bool     `json:"archived"`
	Checklist      []string `json:"checklist"`
}
type ManualJournalCapitalInput struct {
	AccountID string `json:"account_id"`
	Kind      string `json:"kind"`
	Amount    string `json:"amount"`
	Date      string `json:"date"`
	Notes     string `json:"notes"`
}
type ManualJournalEntryInput struct {
	AccountID       string                     `json:"account_id"`
	Status          string                     `json:"status"`
	Asset           string                     `json:"asset"`
	Side            string                     `json:"side"`
	OpenedAt        *time.Time                 `json:"opened_at"`
	ClosedAt        *time.Time                 `json:"closed_at"`
	EntryPrice      *string                    `json:"entry_price"`
	StopLoss        *string                    `json:"stop_loss"`
	TakeProfit      *string                    `json:"take_profit"`
	ExitPrice       *string                    `json:"exit_price"`
	Size            *string                    `json:"size"`
	SizeUnit        string                     `json:"size_unit"`
	NetPNL          *string                    `json:"net_pnl"`
	RiskAmount      *string                    `json:"risk_amount"`
	CapitalSnapshot *string                    `json:"capital_snapshot"`
	Setup           string                     `json:"setup"`
	Tags            []string                   `json:"tags"`
	Emotions        []string                   `json:"emotions"`
	Mistakes        []string                   `json:"mistakes"`
	Session         string                     `json:"session"`
	Checklist       []model.ManualJournalCheck `json:"checklist"`
	Notes           string                     `json:"notes"`
	TradingViewURLs []string                   `json:"tradingview_urls"`
}

// Responses use dedicated named types; persistence metadata stays internal except public Base fields.
type ManualJournalAccountResponse model.ManualJournalAccount
type ManualJournalCapitalResponse model.ManualJournalCapital
type ManualJournalEntryResponse model.ManualJournalEntry
type ManualJournalScreenshotResponse model.ManualJournalScreenshot

type ManualJournalFilter = model.ManualJournalFilter

type ManualJournalMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}
type ManualJournalGroup struct {
	Trades int    `json:"trades"`
	Wins   int    `json:"wins"`
	NetPNL string `json:"net_pnl"`
}
type ManualJournalPoint struct {
	ID         string `json:"id"`
	Date       string `json:"date"`
	PNL        string `json:"pnl"`
	Cumulative string `json:"cumulative"`
	Drawdown   string `json:"drawdown"`
}
type ManualJournalStats struct {
	Currency     string                                   `json:"currency"`
	Trades       int                                      `json:"trades"`
	Wins         int                                      `json:"wins"`
	Losses       int                                      `json:"losses"`
	Breakeven    int                                      `json:"breakeven"`
	WinRate      string                                   `json:"win_rate"`
	ProfitFactor *string                                  `json:"profit_factor"`
	AverageWin   string                                   `json:"average_win"`
	AverageLoss  string                                   `json:"average_loss"`
	NetPNL       string                                   `json:"net_pnl"`
	MaxDrawdown  string                                   `json:"max_drawdown"`
	Curve        []ManualJournalPoint                     `json:"curve"`
	Calendar     map[string]ManualJournalGroup            `json:"calendar"`
	Breakdowns   map[string]map[string]ManualJournalGroup `json:"breakdowns"`
}
type ManualJournalInsight struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}
type ManualJournalCSVInput struct {
	CSV string `json:"csv"`
}
type ManualJournalConfirmInput struct {
	CSV   string `json:"csv"`
	Token string `json:"token"`
}
type ManualJournalPreview struct {
	Rows      []ManualJournalEntryInput `json:"rows"`
	Count     int                       `json:"count"`
	Token     string                    `json:"token"`
	ExpiresAt time.Time                 `json:"expires_at"`
}
