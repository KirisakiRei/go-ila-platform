package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"

	"github.com/google/uuid"
)

const ManualJournalMaxUpload int64 = 5 << 20
const ManualJournalMaxCSV = 1 << 20
const manualJournalCSVRows = 500
const manualJournalReportLimit = 10000

var manualJournalWIB = time.FixedZone("WIB", 7*3600)
var manualDecimal = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
var manualCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

type ManualJournalStore interface {
	Account(context.Context, string, string) (*model.ManualJournalAccount, error)
	Currencies(context.Context, string, []string) ([]string, error)
	BalanceValues(context.Context, string, []string, func(string, string, *string) error) error
	Capital(context.Context, string, string) (*model.ManualJournalCapital, error)
	Entry(context.Context, string, string) (*model.ManualJournalEntry, error)
	Create(context.Context, any) error
	Update(context.Context, string, string, any) error
	Delete(context.Context, string, string, any) error
	Accounts(context.Context, string, dto.ManualJournalFilter) ([]model.ManualJournalAccount, int64, error)
	Capitals(context.Context, string, dto.ManualJournalFilter) ([]model.ManualJournalCapital, int64, error)
	Entries(context.Context, string, dto.ManualJournalFilter, bool) ([]model.ManualJournalEntry, int64, error)
	Import(context.Context, string, string, string, []model.ManualJournalEntry) error
	Screenshots(context.Context, string, string) ([]model.ManualJournalScreenshot, error)
	Screenshot(context.Context, string, string, string) (*model.ManualJournalScreenshot, error)
	ScreenshotTombstones(context.Context, string, string) ([]model.ManualJournalScreenshot, error)
	PurgeScreenshot(context.Context, string, string) error
}
type ManualJournalService struct {
	repo ManualJournalStore
	dir  string
}

func NewManualJournalService(repo ManualJournalStore, privateUploadDir string) *ManualJournalService {
	return &ManualJournalService{repo: repo, dir: privateUploadDir}
}
func manualBad(message string) error { return fmt.Errorf("%w: %s", apperror.ErrBadRequest, message) }
func manualIdentity(user string) error {
	if user == "" {
		return apperror.ErrUnauthorized
	}
	return nil
}
func manualID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return manualBad("invalid ID")
	}
	return nil
}
func manualDate(s string) error {
	if len(s) != 10 {
		return manualBad("date must be YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return manualBad("invalid date")
	}
	return nil
}

// Decimal strings are validated before exact rational arithmetic; no float money crosses this boundary.
func manualNumber(s string, scale int, negative bool) (*big.Rat, error) {
	if len(s) > 32 || !manualDecimal.MatchString(s) {
		return nil, manualBad("decimal string required")
	}
	if p := strings.IndexByte(s, '.'); p >= 0 && len(s)-p-1 > scale {
		return nil, manualBad(fmt.Sprintf("maximum %d decimal places", scale))
	}
	n, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, manualBad("invalid decimal")
	}
	if !negative && n.Sign() < 0 {
		return nil, manualBad("negative amount not allowed")
	}
	if new(big.Rat).Abs(n).Cmp(big.NewRat(1000000000, 1)) > 0 {
		return nil, manualBad("amount exceeds 1000000000")
	}
	return n, nil
}
func manualMoney(s string, negative bool) (string, error) {
	n, err := manualNumber(s, 4, negative)
	if err != nil {
		return "", err
	}
	return n.FloatString(4), nil
}
func manualStrings(values []string, maxLen int) error {
	if len(values) > 30 {
		return manualBad("maximum 30 labels/URLs")
	}
	seen := map[string]bool{}
	for _, v := range values {
		if err := manualText(v); err != nil {
			return err
		}
		if strings.TrimSpace(v) == "" || len(v) > maxLen || seen[v] {
			return manualBad("empty, duplicate or oversized label/URL")
		}
		seen[v] = true
	}
	return nil
}
func manualText(values ...string) error {
	for _, s := range values {
		if len(s) > 10000 || strings.ContainsRune(s, 0) {
			return manualBad("text is too long or contains NUL")
		}
	}
	return nil
}
func (s *ManualJournalService) Filter(ctx context.Context, user string, f dto.ManualJournalFilter) (dto.ManualJournalFilter, error) {
	if err := manualIdentity(user); err != nil {
		return f, err
	}
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PerPage == 0 {
		f.PerPage = 20
	}
	if f.Page < 1 || f.Page > 1000000 || f.PerPage < 1 || f.PerPage > 100 {
		return f, manualBad("invalid pagination")
	}
	if len(f.Search) > 100 || len(f.Tag) > 100 || len(f.Emotion) > 100 || len(f.Mistake) > 100 || len(f.Asset) > 100 || len(f.Setup) > 100 || len(f.Session) > 100 {
		return f, manualBad("filter too long")
	}
	if f.From != "" {
		if err := manualDate(f.From); err != nil {
			return f, err
		}
	}
	if f.To != "" {
		if err := manualDate(f.To); err != nil {
			return f, err
		}
	}
	if f.From != "" && f.To != "" && f.From > f.To {
		return f, manualBad("from exceeds to")
	}
	if f.Status != "" && f.Status != "draft" && f.Status != "open" && f.Status != "closed" {
		return f, manualBad("invalid status")
	}
	if f.Side != "" && f.Side != "long" && f.Side != "short" {
		return f, manualBad("invalid side")
	}
	if f.AccountID != "" {
		if _, err := s.GetAccount(ctx, user, f.AccountID); err != nil {
			return f, err
		}
	}
	return f, nil
}
func manualMeta(f dto.ManualJournalFilter, n int64) dto.ManualJournalMeta {
	return dto.ManualJournalMeta{CurrentPage: f.Page, PerPage: f.PerPage, TotalItems: n, TotalPages: int((n + int64(f.PerPage) - 1) / int64(f.PerPage))}
}
func (s *ManualJournalService) accountBalances(ctx context.Context, user string, accounts []model.ManualJournalAccount) ([]dto.ManualJournalAccountResponse, error) {
	out := make([]dto.ManualJournalAccountResponse, len(accounts))
	ids := make([]string, 0, len(accounts))
	indices := map[string]int{}
	totals := make([]*big.Rat, len(accounts))
	for i, a := range accounts {
		n, err := manualRat(a.InitialCapital)
		if err != nil {
			return nil, err
		}
		totals[i] = n
		indices[a.ID] = i
		ids = append(ids, a.ID)
		out[i] = dto.ManualJournalAccountResponse(a)
		out[i].BalanceComplete = true
	}
	err := s.repo.BalanceValues(ctx, user, ids, func(id, kind string, amount *string) error {
		i, ok := indices[id]
		if !ok {
			return fmt.Errorf("unexpected account in balance inputs")
		}
		if amount == nil {
			if kind != "pnl" {
				return fmt.Errorf("missing capital amount")
			}
			out[i].UnknownClosedPNL++
			out[i].BalanceComplete = false
			return nil
		}
		n, err := manualRat(*amount)
		if err != nil {
			return err
		}
		switch kind {
		case "withdrawal":
			totals[i].Sub(totals[i], n)
		case "deposit", "pnl":
			totals[i].Add(totals[i], n)
		default:
			return fmt.Errorf("invalid capital kind")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].BalanceComplete {
			v := totals[i].FloatString(4)
			out[i].CurrentRealizedBalance = &v
		}
	}
	return out, nil
}

func (s *ManualJournalService) GetAccount(ctx context.Context, user, id string) (*dto.ManualJournalAccountResponse, error) {
	if err := manualIdentity(user); err != nil {
		return nil, err
	}
	if err := manualID(id); err != nil {
		return nil, err
	}
	v, err := s.repo.Account(ctx, user, id)
	if err != nil {
		return nil, err
	}
	out, err := s.accountBalances(ctx, user, []model.ManualJournalAccount{*v})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}
func (s *ManualJournalService) ListAccounts(ctx context.Context, user string, f dto.ManualJournalFilter) ([]dto.ManualJournalAccountResponse, dto.ManualJournalMeta, error) {
	f, err := s.Filter(ctx, user, f)
	if err != nil {
		return nil, dto.ManualJournalMeta{}, err
	}
	rows, n, err := s.repo.Accounts(ctx, user, f)
	if err != nil {
		return nil, manualMeta(f, n), err
	}
	out, err := s.accountBalances(ctx, user, rows)
	return out, manualMeta(f, n), err
}
func (s *ManualJournalService) SaveAccount(ctx context.Context, user, id string, in dto.ManualJournalAccountInput) (*dto.ManualJournalAccountResponse, error) {
	if err := manualIdentity(user); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 || len(in.Broker) > 100 || !manualCurrency.MatchString(in.Currency) {
		return nil, manualBad("name, broker or currency invalid")
	}
	if err := manualText(in.Name, in.Broker, in.Notes); err != nil {
		return nil, err
	}
	if err := manualDate(in.InitialDate); err != nil {
		return nil, err
	}
	if err := manualStrings(in.Checklist, 200); err != nil {
		return nil, err
	}
	amount, err := manualMoney(in.InitialCapital, false)
	if err != nil {
		return nil, err
	}
	v := model.ManualJournalAccount{UserID: user, Name: in.Name, Broker: in.Broker, Currency: in.Currency, InitialCapital: amount, InitialDate: in.InitialDate, Notes: in.Notes, Archived: in.Archived, Checklist: in.Checklist}
	if id != "" {
		old, err := s.GetAccount(ctx, user, id)
		if err != nil {
			return nil, err
		}
		if old.Currency != in.Currency {
			return nil, manualBad("account currency is immutable; create another account")
		}
		v.Base = old.Base
		err = s.repo.Update(ctx, user, id, &v)
		if err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.Create(ctx, &v); err != nil {
			return nil, err
		}
	}
	out, err := s.accountBalances(ctx, user, []model.ManualJournalAccount{v})
	if err != nil {
		return nil, err
	}
	return &out[0], nil
}
func (s *ManualJournalService) ArchiveAccount(ctx context.Context, user, id string) error {
	v, err := s.GetAccount(ctx, user, id)
	if err != nil {
		return err
	}
	row := model.ManualJournalAccount(*v)
	row.Archived = true
	return s.repo.Update(ctx, user, id, &row)
}
func (s *ManualJournalService) writableAccount(ctx context.Context, user, id string) (*dto.ManualJournalAccountResponse, error) {
	v, err := s.GetAccount(ctx, user, id)
	if err != nil {
		return nil, err
	}
	if v.Archived {
		return nil, fmt.Errorf("%w: account is archived", apperror.ErrConflict)
	}
	return v, nil
}
func (s *ManualJournalService) GetCapital(ctx context.Context, user, id string) (*dto.ManualJournalCapitalResponse, error) {
	if err := manualIdentity(user); err != nil {
		return nil, err
	}
	if err := manualID(id); err != nil {
		return nil, err
	}
	v, err := s.repo.Capital(ctx, user, id)
	if err != nil {
		return nil, err
	}
	out := dto.ManualJournalCapitalResponse(*v)
	return &out, nil
}
func (s *ManualJournalService) ListCapital(ctx context.Context, user string, f dto.ManualJournalFilter) ([]dto.ManualJournalCapitalResponse, dto.ManualJournalMeta, error) {
	f, err := s.Filter(ctx, user, f)
	if err != nil {
		return nil, dto.ManualJournalMeta{}, err
	}
	rows, n, err := s.repo.Capitals(ctx, user, f)
	out := make([]dto.ManualJournalCapitalResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, dto.ManualJournalCapitalResponse(v))
	}
	return out, manualMeta(f, n), err
}
func (s *ManualJournalService) SaveCapital(ctx context.Context, user, id string, in dto.ManualJournalCapitalInput) (*dto.ManualJournalCapitalResponse, error) {
	if _, err := s.writableAccount(ctx, user, in.AccountID); err != nil {
		return nil, err
	}
	if in.Kind != "deposit" && in.Kind != "withdrawal" {
		return nil, manualBad("invalid capital kind")
	}
	if err := manualDate(in.Date); err != nil {
		return nil, err
	}
	if err := manualText(in.Notes); err != nil {
		return nil, err
	}
	n, err := manualNumber(in.Amount, 4, false)
	if err != nil {
		return nil, err
	}
	if n.Sign() == 0 {
		return nil, manualBad("amount must be positive")
	}
	v := model.ManualJournalCapital{UserID: user, AccountID: in.AccountID, Kind: in.Kind, Amount: n.FloatString(4), Date: in.Date, Notes: in.Notes}
	if id != "" {
		old, err := s.GetCapital(ctx, user, id)
		if err != nil {
			return nil, err
		}
		if old.AccountID != in.AccountID {
			return nil, manualBad("account cannot change")
		}
		v.Base = old.Base
		err = s.repo.Update(ctx, user, id, &v)
		if err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.Create(ctx, &v); err != nil {
			return nil, err
		}
	}
	out := dto.ManualJournalCapitalResponse(v)
	return &out, nil
}
func (s *ManualJournalService) DeleteCapital(ctx context.Context, user, id string) error {
	v, err := s.GetCapital(ctx, user, id)
	if err != nil {
		return err
	}
	if _, err = s.writableAccount(ctx, user, v.AccountID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, user, id, &model.ManualJournalCapital{})
}
func (s *ManualJournalService) GetEntry(ctx context.Context, user, id string) (*dto.ManualJournalEntryResponse, error) {
	if err := manualIdentity(user); err != nil {
		return nil, err
	}
	if err := manualID(id); err != nil {
		return nil, err
	}
	v, err := s.repo.Entry(ctx, user, id)
	if err != nil {
		return nil, err
	}
	out := dto.ManualJournalEntryResponse(*v)
	return &out, nil
}
func (s *ManualJournalService) ListEntries(ctx context.Context, user string, f dto.ManualJournalFilter) ([]dto.ManualJournalEntryResponse, dto.ManualJournalMeta, error) {
	f, err := s.Filter(ctx, user, f)
	if err != nil {
		return nil, dto.ManualJournalMeta{}, err
	}
	rows, n, err := s.repo.Entries(ctx, user, f, false)
	out := make([]dto.ManualJournalEntryResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, dto.ManualJournalEntryResponse(v))
	}
	return out, manualMeta(f, n), err
}
func (s *ManualJournalService) validateEntry(ctx context.Context, user string, in dto.ManualJournalEntryInput, old *dto.ManualJournalEntryResponse, importSnapshot bool) (model.ManualJournalEntry, error) {
	var v model.ManualJournalEntry
	a, err := s.writableAccount(ctx, user, in.AccountID)
	if err != nil {
		return v, err
	}
	if old != nil && old.AccountID != in.AccountID {
		return v, manualBad("entry account cannot change")
	}
	if in.Status != "draft" && in.Status != "open" && in.Status != "closed" {
		return v, manualBad("invalid status")
	}
	if in.Side != "" && in.Side != "long" && in.Side != "short" {
		return v, manualBad("invalid side")
	}
	if len(in.Asset) > 100 || len(in.SizeUnit) > 40 || len(in.Setup) > 100 || len(in.Session) > 100 {
		return v, manualBad("entry field too long")
	}
	if err := manualText(in.Asset, in.SizeUnit, in.Setup, in.Session, in.Notes); err != nil {
		return v, err
	}
	if in.Status != "draft" && (strings.TrimSpace(in.Asset) == "" || in.Side == "" || in.OpenedAt == nil || in.EntryPrice == nil || in.Size == nil || strings.TrimSpace(in.SizeUnit) == "") {
		return v, manualBad("open/closed trade requires asset, side, opened_at, entry_price, size, size_unit")
	}
	if in.Status == "closed" && (in.ClosedAt == nil || in.NetPNL == nil) {
		return v, manualBad("closed trade requires closed_at and net_pnl")
	}
	if in.Status != "closed" && (in.ClosedAt != nil || in.NetPNL != nil) {
		return v, manualBad("only closed trades may carry realized pnl or closed_at")
	}
	for _, t := range []*time.Time{in.OpenedAt, in.ClosedAt} {
		if t != nil && (t.Year() < 1900 || t.Year() > 2200) {
			return v, manualBad("timestamp outside 1900..2200")
		}
	}
	if in.ClosedAt != nil && (in.OpenedAt == nil || in.ClosedAt.Before(*in.OpenedAt)) {
		return v, manualBad("closed_at precedes opened_at")
	}
	for _, p := range []*string{in.EntryPrice, in.StopLoss, in.TakeProfit, in.ExitPrice, in.Size} {
		if p != nil {
			n, err := manualNumber(*p, 8, false)
			if err != nil {
				return v, err
			}
			if n.Sign() == 0 {
				return v, manualBad("price/size must be positive")
			}
		}
	}
	for i, p := range []*string{in.NetPNL, in.RiskAmount, in.CapitalSnapshot} {
		if p != nil {
			if _, err := manualNumber(*p, 4, i == 0); err != nil {
				return v, err
			}
		}
	}
	for _, list := range [][]string{in.Tags, in.Emotions, in.Mistakes} {
		if err := manualStrings(list, 100); err != nil {
			return v, err
		}
	}
	if err := manualStrings(in.TradingViewURLs, 2048); err != nil {
		return v, err
	}
	for _, raw := range in.TradingViewURLs {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || !(u.Hostname() == "tradingview.com" || strings.HasSuffix(u.Hostname(), ".tradingview.com")) {
			return v, manualBad("only HTTPS TradingView URLs allowed")
		}
	}
	labels := a.Checklist
	if old != nil {
		labels = make([]string, 0, len(old.Checklist))
		for _, c := range old.Checklist {
			labels = append(labels, c.Label)
		}
	}
	inputLabels := make([]string, 0, len(in.Checklist))
	for _, c := range in.Checklist {
		inputLabels = append(inputLabels, c.Label)
	}
	if err := manualStrings(inputLabels, 200); err != nil {
		return v, err
	}

	// CSV carries historical snapshots, not the current account template.
	if importSnapshot {
		labels = inputLabels
	}
	checked := map[string]bool{}
	for _, c := range in.Checklist {
		if _, ok := checked[c.Label]; ok {
			return v, manualBad("duplicate checklist label")
		}
		found := false
		for _, label := range labels {
			if label == c.Label {
				found = true
				break
			}
		}
		if !found {
			return v, manualBad("unknown checklist label")
		}
		checked[c.Label] = c.Checked
	}
	snapshot := make([]model.ManualJournalCheck, 0, len(labels))
	for _, label := range labels {
		snapshot = append(snapshot, model.ManualJournalCheck{Label: label, Checked: checked[label]})
	}
	v = model.ManualJournalEntry{UserID: user, AccountID: in.AccountID, Status: in.Status, Asset: in.Asset, Side: in.Side, OpenedAt: in.OpenedAt, ClosedAt: in.ClosedAt, EntryPrice: in.EntryPrice, StopLoss: in.StopLoss, TakeProfit: in.TakeProfit, ExitPrice: in.ExitPrice, Size: in.Size, SizeUnit: in.SizeUnit, NetPNL: in.NetPNL, RiskAmount: in.RiskAmount, CapitalSnapshot: in.CapitalSnapshot, Setup: in.Setup, Tags: in.Tags, Emotions: in.Emotions, Mistakes: in.Mistakes, Session: in.Session, Checklist: snapshot, Notes: in.Notes, TradingViewURLs: in.TradingViewURLs}
	if old != nil {
		v.Base = old.Base
	}
	return v, nil
}
func (s *ManualJournalService) SaveEntry(ctx context.Context, user, id string, in dto.ManualJournalEntryInput) (*dto.ManualJournalEntryResponse, error) {
	var old *dto.ManualJournalEntryResponse
	if id != "" {
		var err error
		old, err = s.GetEntry(ctx, user, id)
		if err != nil {
			return nil, err
		}
	}
	v, err := s.validateEntry(ctx, user, in, old, false)
	if err != nil {
		return nil, err
	}
	if id == "" {
		err = s.repo.Create(ctx, &v)
	} else {
		err = s.repo.Update(ctx, user, id, &v)
	}
	if err != nil {
		return nil, err
	}
	out := dto.ManualJournalEntryResponse(v)
	return &out, nil
}
func (s *ManualJournalService) DeleteEntry(ctx context.Context, user, id string) error {
	v, err := s.GetEntry(ctx, user, id)
	if err != nil {
		return err
	}
	if _, err = s.writableAccount(ctx, user, v.AccountID); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, user, id, &model.ManualJournalEntry{}); err != nil {
		return err
	}
	return s.cleanupScreenshots(ctx, user, id)
}

func (s *ManualJournalService) report(ctx context.Context, user string, f dto.ManualJournalFilter) ([]model.ManualJournalEntry, error) {
	f, err := s.Filter(ctx, user, f)
	if err != nil {
		return nil, err
	}
	rows, _, err := s.repo.Entries(ctx, user, f, true)
	if err != nil {
		return nil, err
	}
	if len(rows) > manualJournalReportLimit {
		return nil, manualBad("report exceeds 10000 trades; narrow filters")
	}
	return rows, nil
}
func manualRat(s string) (*big.Rat, error) {
	n, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid persisted journal decimal")
	}
	return n, nil
}
func manualGroupAdd(groups map[string]dto.ManualJournalGroup, key string, pnl *big.Rat) error {
	g := groups[key]
	g.Trades++
	if pnl.Sign() > 0 {
		g.Wins++
	}
	n := new(big.Rat)
	if g.NetPNL != "" {
		var err error
		n, err = manualRat(g.NetPNL)
		if err != nil {
			return err
		}
	}
	g.NetPNL = n.Add(n, pnl).FloatString(4)
	groups[key] = g
	return nil
}
func manualCalculate(rows []model.ManualJournalEntry) (dto.ManualJournalStats, error) {
	out := dto.ManualJournalStats{WinRate: "0.0000", AverageWin: "0.0000", AverageLoss: "0.0000", NetPNL: "0.0000", MaxDrawdown: "0.0000", Curve: []dto.ManualJournalPoint{}, Calendar: map[string]dto.ManualJournalGroup{}, Breakdowns: map[string]map[string]dto.ManualJournalGroup{}}
	for _, dim := range []string{"asset", "setup", "session", "tag", "emotion", "mistake", "side"} {
		out.Breakdowns[dim] = map[string]dto.ManualJournalGroup{}
	}
	realized := make([]model.ManualJournalEntry, 0, len(rows))
	for _, row := range rows {
		if row.Status == "closed" && row.NetPNL != nil {
			if row.ClosedAt == nil {
				return out, fmt.Errorf("closed journal entry missing timestamp")
			}
			realized = append(realized, row)
		}
	}
	sort.SliceStable(realized, func(i, j int) bool {
		if realized[i].ClosedAt.Equal(*realized[j].ClosedAt) {
			return realized[i].ID < realized[j].ID
		}
		return realized[i].ClosedAt.Before(*realized[j].ClosedAt)
	})
	cumulative, peak, maxDD, positive, negative := new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat), new(big.Rat)
	for _, row := range realized {
		pnl, err := manualRat(*row.NetPNL)
		if err != nil {
			return out, err
		}
		out.Trades++
		switch pnl.Sign() {
		case 1:
			out.Wins++
			positive.Add(positive, pnl)
		case -1:
			out.Losses++
			negative.Sub(negative, pnl)
		default:
			out.Breakeven++
		}
		cumulative.Add(cumulative, pnl)
		if cumulative.Cmp(peak) > 0 {
			peak.Set(cumulative)
		}
		dd := new(big.Rat).Sub(peak, cumulative)
		if dd.Cmp(maxDD) > 0 {
			maxDD.Set(dd)
		}
		date := row.ClosedAt.In(manualJournalWIB).Format("2006-01-02")
		out.Curve = append(out.Curve, dto.ManualJournalPoint{ID: row.ID, Date: date, PNL: pnl.FloatString(4), Cumulative: cumulative.FloatString(4), Drawdown: dd.FloatString(4)})
		if err := manualGroupAdd(out.Calendar, date, pnl); err != nil {
			return out, err
		}
		for dim, labels := range map[string][]string{"asset": {row.Asset}, "setup": {row.Setup}, "session": {row.Session}, "side": {row.Side}, "tag": row.Tags, "emotion": row.Emotions, "mistake": row.Mistakes} {
			for _, label := range labels {
				if err := manualGroupAdd(out.Breakdowns[dim], label, pnl); err != nil {
					return out, err
				}
			}
		}
	}
	out.NetPNL = cumulative.FloatString(4)
	out.MaxDrawdown = maxDD.FloatString(4)
	if decided := out.Wins + out.Losses; decided > 0 {
		out.WinRate = big.NewRat(int64(out.Wins*100), int64(decided)).FloatString(4)
	}
	if out.Wins > 0 {
		out.AverageWin = new(big.Rat).Quo(positive, big.NewRat(int64(out.Wins), 1)).FloatString(4)
	}
	if out.Losses > 0 {
		out.AverageLoss = new(big.Rat).Quo(negative, big.NewRat(int64(out.Losses), 1)).FloatString(4)
		v := new(big.Rat).Quo(positive, negative).FloatString(4)
		out.ProfitFactor = &v
	}
	return out, nil
}
func (s *ManualJournalService) Stats(ctx context.Context, user string, f dto.ManualJournalFilter) (*dto.ManualJournalStats, error) {
	rows, err := s.report(ctx, user, f)
	if err != nil {
		return nil, err
	}
	currency := ""
	seen := map[string]bool{}
	ids := []string{}
	for _, row := range rows {
		if row.Status == "closed" && row.NetPNL != nil && !seen[row.AccountID] {
			seen[row.AccountID] = true
			ids = append(ids, row.AccountID)
		}
	}
	currencies, err := s.repo.Currencies(ctx, user, ids)
	if err != nil {
		return nil, err
	}
	if len(currencies) > 1 {
		return nil, manualBad("mixed currencies; select one account/currency")
	}
	if len(currencies) == 1 {
		currency = currencies[0]
	}
	if currency == "" && f.AccountID != "" {
		a, err := s.GetAccount(ctx, user, f.AccountID)
		if err != nil {
			return nil, err
		}
		currency = a.Currency
	}
	out, err := manualCalculate(rows)
	if err != nil {
		return nil, err
	}
	out.Currency = currency
	return &out, nil
}
func (s *ManualJournalService) Insights(ctx context.Context, user string, f dto.ManualJournalFilter) ([]dto.ManualJournalInsight, error) {
	stats, err := s.Stats(ctx, user, f)
	if err != nil {
		return nil, err
	}
	out := []dto.ManualJournalInsight{}
	if stats.Trades < 20 {
		out = append(out, dto.ManualJournalInsight{Code: "low_sample", Message: "Fewer than 20 realized trades; avoid strong conclusions.", Count: stats.Trades})
	}
	net, err := manualRat(stats.NetPNL)
	if err != nil {
		return nil, err
	}
	if net.Sign() < 0 {
		out = append(out, dto.ManualJournalInsight{Code: "negative_expectancy", Message: "Realized average PNL is negative.", Count: stats.Trades})
	}
	// Curve is chronological realized-only data; breakeven ends the current losing streak.
	streak := 0
	for i := len(stats.Curve) - 1; i >= 0; i-- {
		n, err := manualRat(stats.Curve[i].PNL)
		if err != nil {
			return nil, err
		}
		if n.Sign() >= 0 {
			break
		}
		streak++
	}
	if streak >= 3 {
		out = append(out, dto.ManualJournalInsight{Code: "losing_streak", Message: "The latest realized trades end with consecutive losses; review risk before the next trade.", Count: streak})
	}
	for _, dim := range []string{"mistake", "setup", "session"} {
		keys := make([]string, 0, len(stats.Breakdowns[dim]))
		for key := range stats.Breakdowns[dim] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			g := stats.Breakdowns[dim][key]
			n, err := manualRat(g.NetPNL)
			if err != nil {
				return nil, err
			}
			if key != "" && g.Trades >= 3 && (dim == "mistake" || n.Sign() < 0) {
				out = append(out, dto.ManualJournalInsight{Code: "review_" + dim, Message: fmt.Sprintf("Review %s: %s", dim, key), Count: g.Trades})
			}
		}
	}
	return out, nil
}

func (s *ManualJournalService) ListScreenshots(ctx context.Context, user, entry string) ([]dto.ManualJournalScreenshotResponse, error) {
	if _, err := s.GetEntry(ctx, user, entry); err != nil {
		return nil, err
	}
	rows, err := s.repo.Screenshots(ctx, user, entry)
	out := make([]dto.ManualJournalScreenshotResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.ManualJournalScreenshotResponse(r))
	}
	return out, err
}
func (s *ManualJournalService) UploadScreenshot(ctx context.Context, user, entry, name string, reader io.Reader) (*dto.ManualJournalScreenshotResponse, error) {
	e, err := s.GetEntry(ctx, user, entry)
	if err != nil {
		return nil, err
	}
	if _, err = s.writableAccount(ctx, user, e.AccountID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.dir) == "" {
		return nil, apperror.ErrServiceUnavailable
	}
	rows, err := s.repo.Screenshots(ctx, user, entry)
	if err != nil {
		return nil, err
	}
	if len(rows) >= 100 {
		return nil, manualBad("maximum 100 screenshots per trade")
	}
	b, err := io.ReadAll(io.LimitReader(reader, ManualJournalMaxUpload+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > ManualJournalMaxUpload {
		return nil, manualBad("screenshot exceeds 5 MiB")
	}
	mime := http.DetectContentType(b)
	ext := ""
	switch mime {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	default:
		return nil, manualBad("screenshot must be PNG, JPEG, GIF or WebP")
	}
	folder := filepath.Join(s.dir, entry)
	if err := os.MkdirAll(folder, 0700); err != nil {
		return nil, err
	}
	stored := uuid.NewString() + ext
	path := filepath.Join(folder, stored)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	_, writeErr := f.Write(b)
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return nil, errors.Join(err, os.Remove(path))
	}
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if len(name) > 160 {
		name = name[:160]
	}
	row := model.ManualJournalScreenshot{UserID: user, EntryID: entry, FileName: name, MimeType: mime, SizeBytes: int64(len(b)), StoredAs: stored}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, errors.Join(err, os.Remove(path))
	}
	out := dto.ManualJournalScreenshotResponse(row)
	return &out, nil
}

type ManualJournalFile struct {
	File *os.File
	Name string
	MIME string
	Size int64
}

func (s *ManualJournalService) screenshot(ctx context.Context, user, entry, id string) (*model.ManualJournalScreenshot, error) {
	if _, err := s.GetEntry(ctx, user, entry); err != nil {
		return nil, err
	}
	if err := manualID(id); err != nil {
		return nil, err
	}
	return s.repo.Screenshot(ctx, user, entry, id)
}
func (s *ManualJournalService) screenshotPath(row *model.ManualJournalScreenshot) (string, error) {
	if s.dir == "" {
		return "", apperror.ErrServiceUnavailable
	}
	if manualID(row.EntryID) != nil || row.StoredAs == "" || row.StoredAs == "." || row.StoredAs == ".." || strings.ContainsAny(row.StoredAs, "/\\:\x00") || filepath.Base(row.StoredAs) != row.StoredAs {
		return "", fmt.Errorf("invalid screenshot storage key")
	}
	return filepath.Join(s.dir, row.EntryID, row.StoredAs), nil
}
func (s *ManualJournalService) OpenScreenshot(ctx context.Context, user, entry, id string) (*ManualJournalFile, error) {
	row, err := s.screenshot(ctx, user, entry, id)
	if err != nil {
		return nil, err
	}
	path, err := s.screenshotPath(row)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, apperror.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ManualJournalFile{File: f, Name: row.FileName, MIME: row.MimeType, Size: row.SizeBytes}, nil
}
func (s *ManualJournalService) DeleteScreenshot(ctx context.Context, user, entry, id string) error {
	row, err := s.screenshot(ctx, user, entry, id)
	if err != nil {
		return err
	}
	e, err := s.GetEntry(ctx, user, entry)
	if err != nil {
		return err
	}
	if _, err = s.writableAccount(ctx, user, e.AccountID); err != nil {
		return err
	}
	// Revoke access durably before unlinking, retaining metadata for cleanup retries.
	if err = s.repo.Delete(ctx, user, id, &model.ManualJournalScreenshot{}); err != nil {
		return err
	}
	return s.cleanupScreenshot(ctx, row)
}

// CleanupScreenshots processes one bounded batch, including attachments of deleted entries.
func (s *ManualJournalService) CleanupScreenshots(ctx context.Context) error {
	return s.cleanupScreenshots(ctx, "", "")
}

func (s *ManualJournalService) cleanupScreenshots(ctx context.Context, user, entry string) error {
	rows, err := s.repo.ScreenshotTombstones(ctx, user, entry)
	if err != nil {
		return err
	}
	for i := range rows {
		err = errors.Join(err, s.cleanupScreenshot(ctx, &rows[i]))
	}
	return err
}

func (s *ManualJournalService) cleanupScreenshot(ctx context.Context, row *model.ManualJournalScreenshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.screenshotPath(row)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return s.repo.PurgeScreenshot(ctx, row.UserID, row.ID)
}

var manualCSVHeader = []string{"account_id", "status", "asset", "side", "opened_at", "closed_at", "entry_price", "stop_loss", "take_profit", "exit_price", "size", "size_unit", "net_pnl", "risk_amount", "capital_snapshot", "setup", "tags", "emotions", "mistakes", "session", "checklist", "notes", "tradingview_urls"}

func manualFormula(s string) bool {
	t := strings.TrimLeft(s, " \t\r\n")
	return strings.ContainsAny(s[:len(s)-len(t)], "\t\r\n") || (t != "" && strings.ContainsRune("=+-@", rune(t[0])))
}
func manualCSVSafe(s string) string {
	if manualFormula(s) || strings.HasPrefix(s, "'") {
		return "'" + s
	}
	return s
}
func manualCSVUnescape(s string) string {
	if strings.HasPrefix(s, "'") && (manualFormula(s[1:]) || strings.HasPrefix(s[1:], "'")) {
		return s[1:]
	}
	return s
}
func manualEntryInput(row model.ManualJournalEntry) (dto.ManualJournalEntryInput, error) {
	// Same field names/types; JSON keeps the CSV DTO independent of persistence metadata.
	b, err := json.Marshal(row)
	if err != nil {
		return dto.ManualJournalEntryInput{}, err
	}
	var in dto.ManualJournalEntryInput
	err = json.Unmarshal(b, &in)
	return in, err
}
func (s *ManualJournalService) parseCSV(ctx context.Context, user, text string) ([]dto.ManualJournalEntryInput, []model.ManualJournalEntry, error) {
	if err := manualIdentity(user); err != nil {
		return nil, nil, err
	}
	if len(text) > ManualJournalMaxCSV {
		return nil, nil, manualBad("CSV exceeds 1 MiB")
	}
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(text, "\uFEFF")))
	r.FieldsPerRecord = len(manualCSVHeader)
	header, err := r.Read()
	if err != nil {
		return nil, nil, manualBad("CSV header required")
	}
	for i, h := range header {
		if h != manualCSVHeader[i] {
			return nil, nil, manualBad("CSV header must match export format")
		}
	}
	inputs := []dto.ManualJournalEntryInput{}
	rows := []model.ManualJournalEntry{}
	for line := 2; ; line++ {
		cells, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, manualBad(fmt.Sprintf("CSV row %d: %v", line, err))
		}
		if len(rows) >= manualJournalCSVRows {
			return nil, nil, manualBad("CSV maximum 500 rows")
		}
		obj := map[string]any{}
		for i, cell := range cells {
			cell = manualCSVUnescape(cell)
			key := header[i]
			switch key {
			case "tags", "emotions", "mistakes", "tradingview_urls", "checklist":
				if cell == "" {
					cell = "[]"
				}
				var v any
				if err := json.Unmarshal([]byte(cell), &v); err != nil {
					return nil, nil, manualBad(fmt.Sprintf("CSV row %d: invalid %s JSON", line, key))
				}
				obj[key] = v
			case "opened_at", "closed_at", "entry_price", "stop_loss", "take_profit", "exit_price", "size", "net_pnl", "risk_amount", "capital_snapshot":
				if cell != "" {
					obj[key] = cell
				}
			default:
				obj[key] = cell
			}
		}
		b, err := json.Marshal(obj)
		if err != nil {
			return nil, nil, err
		}
		var in dto.ManualJournalEntryInput
		if err := json.Unmarshal(b, &in); err != nil {
			return nil, nil, manualBad(fmt.Sprintf("CSV row %d: %v", line, err))
		}
		v, err := s.validateEntry(ctx, user, in, nil, true)
		if err != nil {
			return nil, nil, fmt.Errorf("CSV row %d: %w", line, err)
		}
		normalized, err := manualEntryInput(v)
		if err != nil {
			return nil, nil, err
		}
		inputs = append(inputs, normalized)
		rows = append(rows, v)
	}
	if len(rows) == 0 {
		return nil, nil, manualBad("CSV contains no entries")
	}
	return inputs, rows, nil
}
func (s *ManualJournalService) PreviewCSV(ctx context.Context, user, text string) (*dto.ManualJournalPreview, error) {
	inputs, _, err := s.parseCSV(ctx, user, text)
	if err != nil {
		return nil, err
	}
	token := model.ManualJournalImportToken{
		Base: model.Base{ID: uuid.NewString()}, UserID: user, PayloadHash: manualCSVHash(text), ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
	}
	if err := s.repo.Create(ctx, &token); err != nil {
		return nil, err
	}
	return &dto.ManualJournalPreview{Rows: inputs, Count: len(inputs), Token: token.ID, ExpiresAt: token.ExpiresAt}, nil
}
func manualCSVHash(text string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(text))) }
func (s *ManualJournalService) ConfirmCSV(ctx context.Context, user, text, token string) (int, error) {
	if err := manualIdentity(user); err != nil {
		return 0, err
	}
	if err := manualID(token); err != nil {
		return 0, manualBad("valid preview token required")
	}
	_, rows, err := s.parseCSV(ctx, user, text)
	if err != nil {
		return 0, err
	}
	if err := s.repo.Import(ctx, user, token, manualCSVHash(text), rows); err != nil {
		return 0, err
	}
	return len(rows), nil
}
func (s *ManualJournalService) ExportCSV(ctx context.Context, user string, f dto.ManualJournalFilter) ([]byte, error) {
	rows, err := s.report(ctx, user, f)
	if err != nil {
		return nil, err
	}
	if len(rows) > manualJournalCSVRows {
		return nil, manualBad("CSV export exceeds 500 rows; narrow filters")
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(manualCSVHeader); err != nil {
		return nil, err
	}
	for _, row := range rows {
		in, err := manualEntryInput(row)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(b, &obj); err != nil {
			return nil, err
		}
		cells := make([]string, len(manualCSVHeader))
		for i, key := range manualCSVHeader {
			raw := obj[key]
			if string(raw) == "null" {
				continue
			}
			var text string
			if err := json.Unmarshal(raw, &text); err == nil {
				cells[i] = manualCSVSafe(text)
			} else {
				cells[i] = manualCSVSafe(string(raw))
			}
		}
		if err := w.Write(cells); err != nil {
			return nil, err
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return nil, err
		}
		if buf.Len() > ManualJournalMaxCSV {
			return nil, manualBad("CSV export exceeds 1 MiB; narrow filters")
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
