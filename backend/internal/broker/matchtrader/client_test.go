package matchtrader

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ila/backend/internal/broker"
)

func TestMapSymbol(t *testing.T) {
	cases := map[string]string{
		"USTEC": "US100", "NAS100": "US100", "NDX": "US100", "US100": "US100",
		"GOLD": "XAUUSD", "XAUUSD": "XAUUSD", "xau/usd": "XAUUSD",
		"EURUSD": "EURUSD", "eur/usd": "EURUSD",
	}
	for in, want := range cases {
		if got := mapSymbol(in); got != want {
			t.Errorf("mapSymbol(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestValuePerLot(t *testing.T) {
	// Nilai terverifikasi dari PnL nyata akun demo.
	if got := valuePerLot(symbolInfo{Multiplier: 20}); got != 20 {
		t.Errorf("US100 valuePerLot=%v, want 20", got)
	}
	if got := valuePerLot(symbolInfo{Multiplier: 100}); got != 100 {
		t.Errorf("XAUUSD valuePerLot=%v, want 100", got)
	}
	if got := valuePerLot(symbolInfo{ContractSize: 100000}); got != 100000 {
		t.Errorf("EURUSD valuePerLot=%v, want 100000", got)
	}
}

func TestRoundToStep(t *testing.T) {
	if got := roundToStep(0.07, 0.01, 2); got != 0.07 {
		t.Errorf("got %v, want 0.07", got)
	}
	if got := roundToStep(0.067, 0.01, 2); got != 0.06 {
		t.Errorf("got %v, want 0.06", got)
	}
	if got := roundToStep(1.234, 0.1, 1); got != 1.2 {
		t.Errorf("got %v, want 1.2", got)
	}
}

func TestCloseReason(t *testing.T) {
	cases := map[string]broker.CloseReason{
		"CLOSE_REASON_TAKE_PROFIT": broker.CloseReasonTP,
		"CLOSE_REASON_STOP_LOSS":   broker.CloseReasonSL,
		"CLOSE_REASON_CLOSE":       broker.CloseReasonManual,
		"WHATEVER":                 broker.CloseReasonOther,
	}
	for in, want := range cases {
		if got := closeReason(in); got != want {
			t.Errorf("closeReason(%q)=%q, want %q", in, got, want)
		}
	}
}

// newTestServer menyiapkan broker tiruan dengan bentuk response asli.
func newTestServer(t *testing.T, openPrice float64, capture *openPositionRequest) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/account-summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"balance":10610.33,"equity":11100.27,"freeFunds":8937.58,"profit":483.52}`))
	})

	mux.HandleFunc("/v1/symbols", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"symbols":[
			{"symbol":"US100","alias":"US100","multiplier":20,"contractSize":1,"pricePrecision":2,"sizeOfOnePoint":0.01,"volumeMin":0.01,"volumeMax":1000000,"volumeStep":0.01,"volumePrecision":2}
		]}`))
	})

	mux.HandleFunc("/v1/quotes", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"quotes":[{"symbol":"US100","alias":"US100","bid":26971.18,"ask":26973.17}]}`))
	})

	mux.HandleFunc("/v1/positions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body openPositionRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			if capture != nil {
				*capture = body
			}
			// Sengaja tanpa openPrice untuk menguji fallback.
			_, _ = w.Write([]byte(`{"id":"W525654572930137","symbol":"US100","side":"SELL","volume":0.4,"openPrice":0}`))
			return
		}
		_, _ = w.Write([]byte(`{"positions":[{"id":"W525654572930137","symbol":"US100","alias":"US100","side":"SELL","volume":0.4,"openPrice":` +
			jsonFloat(openPrice) + `,"currentPrice":26973.17,"slPrice":27151.35,"tpPrice":26792.64,"profit":483.52,"netProfit":489.94,"openTime":1793997975079}]}`))
	})

	mux.HandleFunc("/v1/closed-positions", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"closedPositions":[{"id":"W1","symbol":"XAUUSD","side":"BUY","volume":0.06,"openPrice":4102.89,"closePrice":4098.21,"profit":-28.08,"netProfit":-28.08,"closeReason":"CLOSE_REASON_STOP_LOSS","openTime":1793911011813,"closeTime":1793911595287}]}`))
	})

	return httptest.NewServer(mux)
}

func jsonFloat(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func connectTest(t *testing.T, srv *httptest.Server, token string) *Client {
	t.Helper()
	c := New("acc-1")
	err := c.Connect(context.Background(), broker.Credentials{
		Platform: "matchtrader", AccountID: "acc-1", Token: token,
		Extra: map[string]string{"base_url": srv.URL},
	})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	return c
}

func TestConnectInvalidToken(t *testing.T) {
	srv := newTestServer(t, 27033.61, nil)
	defer srv.Close()

	c := New("acc-1")
	err := c.Connect(context.Background(), broker.Credentials{
		Token: "bad", Extra: map[string]string{"base_url": srv.URL},
	})
	if err == nil {
		t.Fatal("token salah seharusnya ditolak")
	}
	if c.Connected() {
		t.Fatal("sesi seharusnya tidak hidup")
	}
}

func TestConnectEmptyToken(t *testing.T) {
	c := New("acc-1")
	if err := c.Connect(context.Background(), broker.Credentials{}); err == nil {
		t.Fatal("token kosong seharusnya ditolak")
	}
}

func TestAccountAndSymbol(t *testing.T) {
	srv := newTestServer(t, 27033.61, nil)
	defer srv.Close()
	c := connectTest(t, srv, "good")
	defer c.Close()

	acc, err := c.Account(context.Background())
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	if acc.Equity != 11100.27 || acc.FreeMargin != 8937.58 {
		t.Errorf("snapshot tidak sesuai: %+v", acc)
	}

	spec, err := c.Symbol(context.Background(), "USTEC")
	if err != nil {
		t.Fatalf("symbol: %v", err)
	}
	if spec.ValuePerLot != 20 || spec.Digits != 2 || spec.LotStep != 0.01 {
		t.Errorf("spec tidak sesuai: %+v", spec)
	}
}

// TestOpenPositionUsesRealFillPrice menjaga bug lama: harga isi harus dari
// broker, bukan dari permintaan.
func TestOpenPositionUsesRealFillPrice(t *testing.T) {
	var captured openPositionRequest
	srv := newTestServer(t, 27035.5, &captured)
	defer srv.Close()
	c := connectTest(t, srv, "good")
	defer c.Close()

	res, err := c.OpenPosition(context.Background(), broker.OpenRequest{
		Symbol: "USTEC", Side: "sell", Lots: 0.4,
		StopLoss: 27151.345, TakeProfit: 26792.641, Comment: "ILA H4->M15",
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if res.Ticket != "W525654572930137" {
		t.Errorf("ticket=%q", res.Ticket)
	}
	if res.ExecutedPrice != 27035.5 {
		t.Errorf("ExecutedPrice=%v, harus dari broker (27035.5)", res.ExecutedPrice)
	}
	if captured.Side != "SELL" {
		t.Errorf("side harus dinormalisasi jadi SELL, dapat %q", captured.Side)
	}
	if captured.SLPrice == nil || *captured.SLPrice != 27151.35 {
		t.Errorf("SL harus dibulatkan ke 2 digit: %v", captured.SLPrice)
	}
	if captured.TPPrice == nil || *captured.TPPrice != 26792.64 {
		t.Errorf("TP harus dibulatkan ke 2 digit: %v", captured.TPPrice)
	}
}

func TestCurrentPriceMid(t *testing.T) {
	srv := newTestServer(t, 27033.61, nil)
	defer srv.Close()
	c := connectTest(t, srv, "good")
	defer c.Close()

	got, err := c.CurrentPrice(context.Background(), "US100")
	if err != nil {
		t.Fatalf("price: %v", err)
	}
	want := (26971.18 + 26973.17) / 2
	if got != want {
		t.Errorf("price=%v, want %v", got, want)
	}
}

func TestListPositionsAndClosed(t *testing.T) {
	srv := newTestServer(t, 27033.61, nil)
	defer srv.Close()
	c := connectTest(t, srv, "good")
	defer c.Close()

	open, err := c.ListPositions(context.Background())
	if err != nil {
		t.Fatalf("positions: %v", err)
	}
	if len(open) != 1 || open[0].Side != "SELL" || open[0].Profit != 489.94 {
		t.Errorf("posisi tidak sesuai: %+v", open)
	}

	closed, err := c.ClosedSince(context.Background(), 0)
	if err != nil {
		t.Fatalf("closed: %v", err)
	}
	if len(closed) != 1 || closed[0].Reason != broker.CloseReasonSL {
		t.Errorf("closed tidak sesuai: %+v", closed)
	}
	if closed[0].ClosedAt != 1793911595 {
		t.Errorf("ClosedAt harus detik, dapat %d", closed[0].ClosedAt)
	}
}

func TestSymbolUnavailable(t *testing.T) {
	srv := newTestServer(t, 27033.61, nil)
	defer srv.Close()
	c := connectTest(t, srv, "good")
	defer c.Close()

	if _, err := c.Symbol(context.Background(), "GBPJPY"); err == nil {
		t.Fatal("simbol tidak tersedia seharusnya error")
	}
}

func TestTruncateComment(t *testing.T) {
	long := ""
	for i := 0; i < 100; i++ {
		long += "a"
	}
	if got := len(truncateComment(long)); got != 64 {
		t.Errorf("panjang komentar=%d, want 64", got)
	}
}
