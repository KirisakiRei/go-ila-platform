// Package matchtrader mengimplementasikan broker.Client untuk Match-Trader
// Trader API v1. Autentikasi memakai access token milik user, sehingga
// runner tidak perlu menyimpan password akun trading.
package matchtrader

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"ila/backend/internal/broker"
)

// BaseURLDefault adalah domain default Trader API Match-Trader.
const BaseURLDefault = "https://trader-api.match-trader.com"

// Jam cadangan bila broker tidak mengirim waktu tutup posisi.
const closedLookbackSeconds = 7 * 24 * 3600

// Client adalah satu akun Match-Trader milik user.
type Client struct {
	accountID string

	mu      sync.Mutex
	conn    *transport
	symbols map[string]symbolInfo
	cred    broker.Credentials
}

// New membuat klien untuk satu akun; sesi dibuka lewat Connect.
func New(accountID string) *Client {
	return &Client{accountID: accountID, symbols: make(map[string]symbolInfo)}
}

// Platform mengembalikan nama platform.
func (c *Client) Platform() string { return "matchtrader" }

// AccountID mengembalikan identitas akun.
func (c *Client) AccountID() string { return c.accountID }

// Connect membuka sesi memakai access token yang sudah didekripsi.
// Base URL bisa ditimpa per akun lewat Extra["base_url"].
func (c *Client) Connect(ctx context.Context, creds broker.Credentials) error {
	token := strings.TrimSpace(creds.Token)
	if token == "" {
		return fmt.Errorf("%w: %v", broker.ErrInvalidCredentials, errEmptyToken)
	}

	base := BaseURLDefault
	if v := strings.TrimSpace(creds.Extra["base_url"]); v != "" {
		base = v
	}
	base = strings.TrimRight(base, "/")

	c.mu.Lock()
	c.conn = newTransport(base, token)
	c.cred = broker.Credentials{Platform: creds.Platform, AccountID: creds.AccountID}
	c.symbols = make(map[string]symbolInfo)
	c.mu.Unlock()

	// Verifikasi sesi sekali agar token salah langsung ketahuan.
	if _, err := c.Account(ctx); err != nil {
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		return err
	}
	return nil
}

// Connected menandai sesi hidup.
func (c *Client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// Close membuang token dari memori.
func (c *Client) Close() error {
	c.mu.Lock()
	c.conn = nil
	c.cred = broker.Credentials{}
	c.symbols = make(map[string]symbolInfo)
	c.mu.Unlock()
	return nil
}

func (c *Client) transport() (*transport, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil, broker.ErrNotConnected
	}
	return c.conn, nil
}

// Account membaca saldo dan ekuitas akun.
func (c *Client) Account(ctx context.Context) (broker.AccountSnapshot, error) {
	t, err := c.transport()
	if err != nil {
		return broker.AccountSnapshot{}, err
	}

	var sum accountSummary
	if err := t.get(ctx, "/v1/account-summary", &sum); err != nil {
		return broker.AccountSnapshot{}, err
	}

	return broker.AccountSnapshot{
		AccountID:  c.accountID,
		Currency:   "USD",
		Balance:    sum.Balance,
		Equity:     sum.Equity,
		FreeMargin: sum.FreeFunds,
	}, nil
}

// Symbol mengembalikan spesifikasi simbol dari broker, bukan tebakan.
func (c *Client) Symbol(ctx context.Context, symbol string) (broker.SymbolSpec, error) {
	info, err := c.symbolInfo(ctx, symbol)
	if err != nil {
		return broker.SymbolSpec{}, err
	}
	return toSpec(info), nil
}

// CurrentPrice mengembalikan harga tengah bid/ask sebagai acuan slippage.
func (c *Client) CurrentPrice(ctx context.Context, symbol string) (float64, error) {
	t, err := c.transport()
	if err != nil {
		return 0, err
	}

	brokerSymbol := mapSymbol(symbol)
	path := "/v1/quotes?symbol=" + url.QueryEscape(brokerSymbol)

	var out quotesResponse
	if err := t.get(ctx, path, &out); err != nil {
		return 0, err
	}
	if len(out.Quotes) == 0 {
		return 0, fmt.Errorf("%w: kutipan %s kosong", broker.ErrUnsupported, brokerSymbol)
	}

	q := out.Quotes[0]
	if q.Ask <= 0 || q.Bid <= 0 {
		return 0, fmt.Errorf("%w: kutipan %s tidak sah", broker.ErrUnsupported, brokerSymbol)
	}
	return (q.Ask + q.Bid) / 2, nil
}

// OpenPosition memasang order market beserta SL/TP di server broker.
func (c *Client) OpenPosition(ctx context.Context, req broker.OpenRequest) (broker.OpenResult, error) {
	t, err := c.transport()
	if err != nil {
		return broker.OpenResult{}, err
	}
	if req.Lots <= 0 {
		return broker.OpenResult{}, fmt.Errorf("%w: lot %v", broker.ErrUnsupported, req.Lots)
	}

	info, err := c.symbolInfo(ctx, req.Symbol)
	if err != nil {
		return broker.OpenResult{}, err
	}

	body := openPositionRequest{
		Symbol:  info.Symbol,
		Side:    normalizeSide(req.Side),
		Volume:  roundToStep(req.Lots, info.VolumeStep, info.VolumePrecision),
		Comment: truncateComment(req.Comment),
	}

	if req.StopLoss > 0 {
		v := roundTo(req.StopLoss, info.PricePrecision)
		body.SLPrice = &v
	}
	if req.TakeProfit > 0 {
		v := roundTo(req.TakeProfit, info.PricePrecision)
		body.TPPrice = &v
	}

	var out position
	if err := t.postJSON(ctx, "/v1/positions", body, &out); err != nil {
		return broker.OpenResult{}, err
	}
	if out.ID == "" {
		return broker.OpenResult{}, fmt.Errorf("match-trader tidak mengembalikan id posisi")
	}

	result := broker.OpenResult{
		Ticket:        out.ID,
		Lots:          body.Volume,
		StopLoss:      req.StopLoss,
		TakeProfit:    req.TakeProfit,
		ExecutedPrice: out.OpenPrice,
	}

	// Harga isi asli wajib akurat: bila kosong, tarik dari daftar posisi.
	if result.ExecutedPrice <= 0 {
		if filled, ok := c.findOpenPrice(ctx, out.ID); ok {
			result.ExecutedPrice = filled
		}
	}
	return result, nil
}

// ListPositions membaca seluruh posisi yang masih hidup.
func (c *Client) ListPositions(ctx context.Context) ([]broker.PositionState, error) {
	t, err := c.transport()
	if err != nil {
		return nil, err
	}

	var out positionsResponse
	if err := t.get(ctx, "/v1/positions", &out); err != nil {
		return nil, err
	}

	list := make([]broker.PositionState, 0, len(out.Positions))
	for _, p := range out.Positions {
		list = append(list, broker.PositionState{
			Ticket:     p.ID,
			Symbol:     displaySymbol(p.Symbol, p.Alias),
			Side:       normalizeSide(p.Side),
			Lots:       p.Volume,
			OpenPrice:  p.OpenPrice,
			StopLoss:   deref(p.SLPrice),
			TakeProfit: deref(p.TPPrice),
			Profit:     profitOf(p.Profit, p.NetProfit),
		})
	}
	return list, nil
}

// ClosedSince membaca posisi tertutup sejak unix detik tertentu.
func (c *Client) ClosedSince(ctx context.Context, since int64) ([]broker.ClosedPosition, error) {
	t, err := c.transport()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	from := since
	if from <= 0 {
		from = now.Unix() - closedLookbackSeconds
	}

	path := fmt.Sprintf("/v1/closed-positions?from=%d&to=%d",
		from*1000, now.UnixMilli())

	var out closedPositionsResponse
	if err := t.get(ctx, path, &out); err != nil {
		return nil, err
	}

	list := make([]broker.ClosedPosition, 0, len(out.ClosedPositions))
	for _, p := range out.ClosedPositions {
		list = append(list, broker.ClosedPosition{
			Ticket:    p.ID,
			ExitPrice: p.ClosePrice,
			Profit:    profitOf(p.Profit, p.NetProfit),
			Reason:    closeReason(p.CloseReason),
			ClosedAt:  p.CloseTime / 1000,
		})
	}
	return list, nil
}

// findOpenPrice mengambil harga isi posisi dari daftar posisi.
func (c *Client) findOpenPrice(ctx context.Context, ticket string) (float64, bool) {
	positions, err := c.ListPositions(ctx)
	if err != nil {
		return 0, false
	}
	for _, p := range positions {
		if p.Ticket == ticket && p.OpenPrice > 0 {
			return p.OpenPrice, true
		}
	}
	return 0, false
}

// symbolInfo mengambil kontrak simbol, dengan cache per sesi.
func (c *Client) symbolInfo(ctx context.Context, symbol string) (symbolInfo, error) {
	want := mapSymbol(symbol)

	c.mu.Lock()
	cached, ok := c.symbols[want]
	c.mu.Unlock()
	if ok {
		return cached, nil
	}

	t, err := c.transport()
	if err != nil {
		return symbolInfo{}, err
	}

	var out symbolsResponse
	if err := t.get(ctx, "/v1/symbols", &out); err != nil {
		return symbolInfo{}, err
	}

	found := make(map[string]symbolInfo, len(out.Symbols))
	for _, s := range out.Symbols {
		found[strings.ToUpper(s.Symbol)] = s
	}

	c.mu.Lock()
	c.symbols = found
	c.mu.Unlock()

	info, ok := found[want]
	if !ok {
		return symbolInfo{}, fmt.Errorf("%w: simbol %s tidak tersedia", broker.ErrUnsupported, want)
	}
	return info, nil
}

// toSpec menerjemahkan kontrak broker ke SymbolSpec internal.
func toSpec(s symbolInfo) broker.SymbolSpec {
	return broker.SymbolSpec{
		Symbol:       displaySymbol(s.Symbol, s.Alias),
		Digits:       s.PricePrecision,
		PipSize:      s.SizeOfOnePoint,
		ValuePerLot:  valuePerLot(s),
		MinLot:       s.VolumeMin,
		MaxLot:       s.VolumeMax,
		LotStep:      s.VolumeStep,
		ContractSize: s.ContractSize,
	}
}

// valuePerLot menghitung nilai uang per 1.0 pergerakan harga untuk 1 lot.
// CFD memakai multiplier (terverifikasi: US100=20, XAUUSD=100) dan forex
// memakai contractSize dalam mata uang kutipan.
func valuePerLot(s symbolInfo) float64 {
	if s.Multiplier > 0 {
		return s.Multiplier
	}
	return s.ContractSize
}

// mapSymbol menyelaraskan nama simbol engine dengan nama di broker.
func mapSymbol(sym string) string {
	s := strings.ToUpper(strings.TrimSpace(sym))
	s = strings.ReplaceAll(s, "/", "")
	switch s {
	case "USTEC", "NAS100", "NDX", "US100":
		return "US100"
	case "GOLD":
		return "XAUUSD"
	default:
		return s
	}
}

func displaySymbol(symbol, alias string) string {
	if strings.TrimSpace(alias) != "" {
		return strings.ToUpper(alias)
	}
	return strings.ToUpper(symbol)
}

func normalizeSide(side string) string {
	if strings.EqualFold(strings.TrimSpace(side), "SELL") {
		return "SELL"
	}
	return "BUY"
}

func closeReason(raw string) broker.CloseReason {
	switch raw {
	case "CLOSE_REASON_TAKE_PROFIT":
		return broker.CloseReasonTP
	case "CLOSE_REASON_STOP_LOSS":
		return broker.CloseReasonSL
	case "CLOSE_REASON_CLOSE", "CLOSE_REASON_MANUAL":
		return broker.CloseReasonManual
	default:
		return broker.CloseReasonOther
	}
}

// profitOf lebih memilih netProfit bila broker mengirimnya.
func profitOf(gross, net float64) float64 {
	if net != 0 {
		return net
	}
	return gross
}

func deref(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func roundTo(v float64, digits int) float64 {
	pow := math.Pow10(digits)
	return math.Round(v*pow) / pow
}

// roundToStep membulatkan lot ke kelipatan volumeStep dan menghormati presisi.
func roundToStep(lots, step float64, precision int) float64 {
	if step > 0 {
		steps := math.Floor(lots/step + 1e-9)
		lots = steps * step
	}
	return roundTo(lots, precision)
}

// truncateComment membatasi komentar agar tidak ditolak broker.
func truncateComment(s string) string {
	const maxLen = 64
	r := []rune(strings.TrimSpace(s))
	if len(r) > maxLen {
		return string(r[:maxLen])
	}
	return string(r)
}

// FormatLot membantu log tanpa membocorkan kredensial.
func FormatLot(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

var _ broker.Client = (*Client)(nil)
