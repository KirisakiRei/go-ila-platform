package broker

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// PaperClient adalah adapter simulasi (dry-run).
//
// Dipakai untuk tier Free, trial, dan uji coba produksi: menghitung dan
// mencatat seolah order, tanpa menyentuh broker mana pun (plans/14 §3.1).
type PaperClient struct {
	accountID string
	platform  string

	mu        sync.Mutex
	connected bool
	balance   float64
	equity    float64
	positions map[string]PositionState
	closed    []ClosedPosition
}

// NewPaperClient membuat adapter simulasi untuk satu akun.
func NewPaperClient(accountID, platform string) *PaperClient {
	return &PaperClient{
		accountID: accountID,
		platform:  platform,
		balance:   10_000,
		equity:    10_000,
		positions: make(map[string]PositionState),
	}
}

// Platform mengembalikan nama platform.
func (c *PaperClient) Platform() string { return c.platform }

// AccountID mengembalikan identitas akun.
func (c *PaperClient) AccountID() string { return c.accountID }

// Connect menandai sesi simulasi siap.
func (c *PaperClient) Connect(_ context.Context, _ Credentials) error {
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()
	return nil
}

// Connected menandai sesi hidup.
func (c *PaperClient) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// Close mengakhiri sesi simulasi.
func (c *PaperClient) Close() error {
	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()
	return nil
}

// Account mengembalikan saldo simulasi.
func (c *PaperClient) Account(_ context.Context) (AccountSnapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return AccountSnapshot{
		AccountID:  c.accountID,
		Currency:   "USD",
		Balance:    c.balance,
		Equity:     c.equity,
		FreeMargin: c.equity,
		Leverage:   100,
	}, nil
}

// Symbol mengembalikan spesifikasi teoretis per simbol.
func (c *PaperClient) Symbol(_ context.Context, symbol string) (SymbolSpec, error) {
	switch symbol {
	case "XAUUSD":
		return SymbolSpec{
			Symbol: symbol, Digits: 2, PipSize: 0.01, ValuePerLot: 100,
			MinLot: 0.01, MaxLot: 50, LotStep: 0.01, ContractSize: 100,
		}, nil
	case "USTEC", "US100", "NAS100":
		return SymbolSpec{
			Symbol: symbol, Digits: 2, PipSize: 1, ValuePerLot: 1,
			MinLot: 0.01, MaxLot: 100, LotStep: 0.01, ContractSize: 1,
		}, nil
	default:
		return SymbolSpec{
			Symbol: symbol, Digits: 5, PipSize: 0.0001, ValuePerLot: 100_000,
			MinLot: 0.01, MaxLot: 100, LotStep: 0.01, ContractSize: 100_000,
		}, nil
	}
}

// Quote mengembalikan harga simulasi dua arah.
//
// Simulasi tidak punya spread sungguhan, jadi Bid dan Ask dibuat sama. Akibatnya
// spread nol dan toleransi slippage kembali ke tabel pair + persentase SL,
// sehingga simulasi tidak ditolak oleh spread karangan.
func (c *PaperClient) Quote(_ context.Context, symbol string) (Quote, error) {
	price := referencePrice(symbol)
	return Quote{Bid: price, Ask: price}, nil
}

// CurrentPrice mengembalikan harga simulasi berbasis harga acuan simbol.
// Nilainya sengaja stabil agar simulasi tidak menolak karena slippage palsu.
func (c *PaperClient) CurrentPrice(_ context.Context, symbol string) (float64, error) {
	return referencePrice(symbol), nil
}

// OpenPosition mencatat posisi simulasi tanpa mengirim order.
func (c *PaperClient) OpenPosition(_ context.Context, req OpenRequest) (OpenResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return OpenResult{}, ErrNotConnected
	}

	ticket := fmt.Sprintf("SIM-%d-%04d", time.Now().UnixNano(), rand.Intn(9999))
	price := referencePrice(req.Symbol)

	c.positions[ticket] = PositionState{
		Ticket:     ticket,
		Symbol:     req.Symbol,
		Side:       req.Side,
		Lots:       req.Lots,
		OpenPrice:  price,
		StopLoss:   req.StopLoss,
		TakeProfit: req.TakeProfit,
	}

	return OpenResult{
		Ticket:        ticket,
		ExecutedPrice: price,
		Lots:          req.Lots,
		StopLoss:      req.StopLoss,
		TakeProfit:    req.TakeProfit,
	}, nil
}

// ListPositions mengembalikan posisi simulasi yang hidup.
func (c *PaperClient) ListPositions(_ context.Context) ([]PositionState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]PositionState, 0, len(c.positions))
	for _, pos := range c.positions {
		out = append(out, pos)
	}
	return out, nil
}

// ClosedSince mengembalikan riwayat penutupan simulasi.
func (c *PaperClient) ClosedSince(_ context.Context, since int64) ([]ClosedPosition, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]ClosedPosition, 0, len(c.closed))
	for _, item := range c.closed {
		if item.ClosedAt >= since {
			out = append(out, item)
		}
	}
	return out, nil
}

// referencePrice adalah harga acuan tetap per simbol untuk simulasi.
func referencePrice(symbol string) float64 {
	switch symbol {
	case "XAUUSD":
		return 4120.50
	case "USTEC", "US100", "NAS100":
		return 21450
	case "EURUSD":
		return 1.08500
	case "GBPUSD":
		return 1.27200
	default:
		return 1.00000
	}
}
