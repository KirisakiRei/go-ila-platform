package market

import (
	"context"
	"testing"
)

// TestMapSymbolMemakaiFeedAdmin memastikan prefiks feed pilihan admin dipakai,
// sedangkan feed kosong jatuh ke pemetaan bawaan.
func TestMapSymbolMemakaiFeedAdmin(t *testing.T) {
	cases := []struct {
		symbol string
		feed   string
		want   string
	}{
		{"NAS100", "ICMARKETS", "ICMARKETS:NAS100"},
		{"xauusd", "forexcom", "FOREXCOM:XAUUSD"},
		{"EUR/USD", "OANDA", "OANDA:EURUSD"},
		{"BTCUSD", "", "BINANCE:BTCUSD"},
		{"XAUUSD", "", "FOREXCOM:XAUUSD"},
	}

	for _, c := range cases {
		if got := MapSymbol(c.symbol, c.feed); got != c.want {
			t.Errorf("MapSymbol(%q, %q) = %q, ingin %q", c.symbol, c.feed, got, c.want)
		}
	}
}

func TestMapResolutionMencakupTimeframeILA(t *testing.T) {
	cases := map[string]string{
		"M5":  "5",
		"M15": "15",
		"H1":  "60",
		"H4":  "240",
		"D1":  "1D",
		"W":   "1W",
	}

	for tf, want := range cases {
		if got := MapResolution(tf); got != want {
			t.Errorf("MapResolution(%q) = %q, ingin %q", tf, got, want)
		}
	}

	if got := MapResolution("X9"); got != "" {
		t.Errorf("MapResolution(X9) = %q, ingin kosong", got)
	}
}

// TestParseBarsTerurutDanUnik memastikan payload TradingView diubah menjadi
// candle menaik tanpa waktu ganda, karena lightweight-charts menolak keduanya.
func TestParseBarsTerurutDanUnik(t *testing.T) {
	raw := map[string]any{
		"m": "timescale_update",
		"p": []any{
			"cs_1",
			map[string]any{
				"sds_1": map[string]any{
					"s": []any{
						map[string]any{"v": []any{float64(300), 2.0, 3.0, 1.5, 2.5, 10.0}},
						map[string]any{"v": []any{float64(100), 1.0, 1.5, 0.5, 1.2, 5.0}},
						// Waktu ganda: nilai terakhir yang dipakai.
						map[string]any{"v": []any{float64(100), 1.0, 1.6, 0.5, 1.3, 6.0}},
						map[string]any{"v": []any{float64(200), 1.5, 2.0, 1.0, 1.8, 7.0}},
						// Baris rusak harus dilewati, bukan menggagalkan semuanya.
						map[string]any{"v": []any{float64(0), 0.0, 0.0, 0.0, 0.0}},
					},
				},
			},
		},
	}

	candles, ok := parseBars(raw)
	if !ok {
		t.Fatal("parseBars menolak payload sah")
	}
	if len(candles) != 3 {
		t.Fatalf("jumlah candle = %d, ingin 3", len(candles))
	}

	wantTimes := []int64{100, 200, 300}
	for i, want := range wantTimes {
		if candles[i].Time != want {
			t.Errorf("candle[%d].Time = %d, ingin %d", i, candles[i].Time, want)
		}
	}
	if candles[0].Close != 1.3 {
		t.Errorf("nilai ganda seharusnya yang terakhir: close = %v, ingin 1.3", candles[0].Close)
	}
}

// TestParseBarsPayloadKosong memastikan payload tanpa baris tidak dipakai.
func TestParseBarsPayloadKosong(t *testing.T) {
	if _, ok := parseBars(map[string]any{"m": "du", "p": []any{}}); ok {
		t.Error("payload tanpa data seharusnya ditolak")
	}
}

// TestServiceCacheMenahanPermintaanBerulang memastikan cache mencegah
// permintaan ke feed untuk permintaan yang sama dalam masa berlaku.
func TestServiceCacheMenahanPermintaanBerulang(t *testing.T) {
	provider := &hitungProvider{candles: []Candle{{Time: 1, Open: 1, High: 2, Low: 1, Close: 2}}}
	svc := NewService(provider, 0)

	for i := 0; i < 3; i++ {
		if _, _, err := svc.Candles(t.Context(), "XAUUSD", "", "H4", 100); err != nil {
			t.Fatalf("Candles: %v", err)
		}
	}

	if provider.calls != 1 {
		t.Errorf("provider dipanggil %d kali, ingin 1 (cache)", provider.calls)
	}
}

type hitungProvider struct {
	calls   int
	candles []Candle
	err     error
}

func (p *hitungProvider) FetchCandles(_ context.Context, _, _, _ string, _ int) ([]Candle, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return p.candles, nil
}
