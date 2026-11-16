package service

import (
	"testing"
	"time"

	"ila/backend/internal/market"
	"ila/backend/internal/model"
)

func sinyalUji(arah string) *model.Signal {
	mulai := time.Unix(1_000_000, 0).UTC()
	return &model.Signal{
		SignalID:   "sig-uji",
		Symbol:     "XAUUSD",
		Direction:  arah,
		EntryPrice: 4120,
		StopLoss:   4112,
		TakeProfit: 4149,
		RiskReward: 2.3,
		ExpiresAt:  mulai.Add(4 * time.Hour),
	}
}

func lilin(unix int64, high, low float64) market.Candle {
	return market.Candle{Time: unix, High: high, Low: low}
}

func TestEvaluateCandlesBuyTP(t *testing.T) {
	signal := sinyalUji("BUY")
	sejak := time.Unix(1_000_000, 0)

	hasil, ok := evaluateCandles(signal, []market.Candle{
		lilin(1_000_000, 4125, 4118),
		lilin(1_000_300, 4152, 4124),
	}, sejak)

	if !ok || hasil.Status != model.EvalTP {
		t.Fatalf("hasil = %+v ok=%v, ingin TP", hasil, ok)
	}
	if hasil.Price != signal.TakeProfit || hasil.RLower != signal.RiskReward {
		t.Fatalf("harga/R = %v/%v, ingin %v/%v", hasil.Price, hasil.RLower, signal.TakeProfit, signal.RiskReward)
	}
}

func TestEvaluateCandlesBuySL(t *testing.T) {
	signal := sinyalUji("BUY")
	sejak := time.Unix(1_000_000, 0)

	hasil, ok := evaluateCandles(signal, []market.Candle{
		lilin(1_000_000, 4125, 4110),
	}, sejak)

	if !ok || hasil.Status != model.EvalSL || hasil.RLower != -1 {
		t.Fatalf("hasil = %+v ok=%v, ingin SL dengan R -1", hasil, ok)
	}
}

// Satu lilin yang menyentuh TP dan SL sekaligus tidak bisa dinilai urutannya.
func TestEvaluateCandlesTPdanSLSamaLilin(t *testing.T) {
	signal := sinyalUji("BUY")
	sejak := time.Unix(1_000_000, 0)

	hasil, ok := evaluateCandles(signal, []market.Candle{
		lilin(1_000_000, 4152, 4110),
	}, sejak)

	if !ok || hasil.Status != model.EvalUnknown {
		t.Fatalf("hasil = %+v ok=%v, ingin UNKNOWN", hasil, ok)
	}
}

// Entry tidak tersentuh sampai kedaluwarsa: setup tidak pernah berjalan.
func TestEvaluateCandlesNoEntry(t *testing.T) {
	signal := sinyalUji("BUY")
	sejak := time.Unix(1_000_000, 0)

	hasil, ok := evaluateCandles(signal, []market.Candle{
		lilin(1_000_000, 4150, 4140),
		lilin(1_020_000, 4150, 4140),
	}, sejak)

	if !ok || hasil.Status != model.EvalNoEntry {
		t.Fatalf("hasil = %+v ok=%v, ingin NO_ENTRY", hasil, ok)
	}
}

// Entry tersentuh tetapi TP/SL belum: setup sedang berjalan, dan jam entry
// harus dilaporkan agar durasi bisa dihitung sejak masuk.
func TestEvaluateCandlesMasihBerjalan(t *testing.T) {
	signal := sinyalUji("BUY")
	sejak := time.Unix(1_000_000, 0)

	hasil, ok := evaluateCandles(signal, []market.Candle{
		lilin(1_000_000, 4125, 4116),
		lilin(1_000_300, 4127, 4118),
	}, sejak)

	if !ok || hasil.Status != model.EvalRunning {
		t.Fatalf("hasil = %+v ok=%v, ingin RUNNING", hasil, ok)
	}
	if hasil.EntryAt == nil || hasil.EntryAt.Unix() != 1_000_000 {
		t.Fatalf("jam entry = %v, ingin lilin pertama yang menyentuh entry", hasil.EntryAt)
	}
	if hasil.At != nil {
		t.Fatalf("waktu keluar = %v, ingin kosong selama belum TP/SL", hasil.At)
	}
}

// Durasi dihitung dari jam masuk ke jam keluar.
func TestEvaluateCandlesDurasi(t *testing.T) {
	signal := sinyalUji("BUY")
	sejak := time.Unix(1_000_000, 0)

	hasil, ok := evaluateCandles(signal, []market.Candle{
		lilin(1_000_000, 4125, 4118),
		lilin(1_003_600, 4152, 4124),
	}, sejak)

	if !ok || hasil.EntryAt == nil || hasil.At == nil {
		t.Fatalf("hasil = %+v ok=%v, ingin TP dengan jam masuk dan keluar", hasil, ok)
	}
	if durasi := hasil.At.Sub(*hasil.EntryAt); durasi != time.Hour {
		t.Fatalf("durasi = %v, ingin 1 jam", durasi)
	}
}

// Sinyal SELL memakai sisi harga yang berlawanan.
func TestEvaluateCandlesSellSL(t *testing.T) {
	signal := sinyalUji("SELL")
	signal.EntryPrice, signal.StopLoss, signal.TakeProfit = 4120, 4128, 4091
	sejak := time.Unix(1_000_000, 0)

	hasil, ok := evaluateCandles(signal, []market.Candle{
		lilin(1_000_000, 4130, 4115),
	}, sejak)

	if !ok || hasil.Status != model.EvalSL {
		t.Fatalf("hasil = %+v ok=%v, ingin SL", hasil, ok)
	}
}
