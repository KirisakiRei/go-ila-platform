//go:build live

package matchtrader

import (
	"context"
	"os"
	"testing"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/broker"
)

// TestLiveAccount menguji koneksi nyata; jalankan dengan -tags live dan MT_TOKEN.
func TestLiveAccount(t *testing.T) {
	token := os.Getenv("MT_TOKEN")
	if token == "" {
		t.Skip("MT_TOKEN tidak diisi")
	}

	c := New("live-test")
	ctx := context.Background()
	if err := c.Connect(ctx, broker.Credentials{Token: token}); err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer c.Close()

	acc, err := c.Account(ctx)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	t.Logf("akun: balance=%.2f equity=%.2f free=%.2f", acc.Balance, acc.Equity, acc.FreeMargin)

	for _, sym := range []string{"XAUUSD", "EURUSD", "USTEC"} {
		spec, err := c.Symbol(ctx, sym)
		if err != nil {
			t.Errorf("symbol %s: %v", sym, err)
			continue
		}
		price, perr := c.CurrentPrice(ctx, sym)
		if perr != nil {
			t.Errorf("price %s: %v", sym, perr)
			continue
		}
		t.Logf("%s: digits=%d valuePerLot=%.2f minLot=%v step=%v harga=%.5f",
			spec.Symbol, spec.Digits, spec.ValuePerLot, spec.MinLot, spec.LotStep, price)
	}

	open, err := c.ListPositions(ctx)
	if err != nil {
		t.Fatalf("positions: %v", err)
	}
	for _, p := range open {
		t.Logf("posisi terbuka: %s %s %s %.2f lot open=%.5f sl=%.5f tp=%.5f pnl=%.2f",
			p.Ticket, p.Symbol, p.Side, p.Lots, p.OpenPrice, p.StopLoss, p.TakeProfit, p.Profit)
	}

	closed, err := c.ClosedSince(ctx, 0)
	if err != nil {
		t.Fatalf("closed: %v", err)
	}
	t.Logf("posisi tertutup 7 hari: %d", len(closed))
	for _, p := range closed {
		t.Logf("  %s exit=%.5f pnl=%.2f reason=%s", p.Ticket, p.ExitPrice, p.Profit, p.Reason)
	}
}
