package service

import (
	"ila/backend/internal/evalprice"
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

// Kasus nyata 16 Nov 2026: EURUSD H1/M5 terbit Minggu 20:10 NY dengan masa
// berlaku 5 menit. Candle yang lahir di luar sesi (mis. Sabtu pagi NY) hanya
// sisa penutupan, jadi tidak boleh ikut menilai TP/SL.
func TestEvaluateCandlesMengabaikanCandleLuarSesi(t *testing.T) {
	// Sabtu 14 Nov 2026 07:00 NY = 12:00 UTC. Pasar tutup penuh hari Sabtu.
	tutup := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	sejak := tutup.Add(-10 * time.Minute)
	signal := &model.Signal{
		SignalID:   "sig-uji-tutup",
		Symbol:     "EURUSD",
		Direction:  "SELL",
		EntryPrice: 1.10218,
		StopLoss:   1.102395,
		TakeProfit: 1.10175,
		RiskReward: 2,
		ExpiresAt:  sejak.Add(5 * time.Minute),
	}

	// Candle Sabtu menyentuh entry dan TP sekaligus.
	hasil, ok := evaluateCandles(signal, []market.Candle{
		lilin(tutup.Unix(), 1.1025, 1.1017),
	}, sejak)

	// Candle itu boleh membuat setup dinilai NO_ENTRY (memang tak ada harga sah
	// yang menyentuh entry), tetapi tidak boleh menghasilkan TP/SL.
	if ok && (hasil.Status == model.EvalTP || hasil.Status == model.EvalSL) {
		t.Fatalf("candle luar sesi menghasilkan %+v; TP/SL tidak boleh muncul dari sesi tutup", hasil)
	}
}

// Legacy calculation is test-only; production journal/evidence never evaluates candles.
type evalOutcome struct {
	Status string
	// EntryAt adalah waktu harga pertama menyentuh level entry. Diisi juga untuk
	// hasil RUNNING sehingga durasi bisa dihitung sejak masuk, bukan sejak terbit.
	EntryAt *time.Time
	At      *time.Time
	Price   float64
	RLower  float64
	Note    string
}

func evaluateCandles(signal *model.Signal, candles []market.Candle, since time.Time) (evalOutcome, bool) {
	in := evalprice.InputDariSinyal(signal)
	// Titik mulai penilaian ditentukan pemanggil: baris user_signal ditulis
	// ulang setiap status berubah, jadi waktu terbit bisa jatuh setelah harga
	// bergerak.
	in.PublishedAt = since
	out, ok := evalprice.FromCandles(in, candles)
	if !ok {
		return evalOutcome{}, false
	}
	return evalOutcome{
		Status:  out.Status,
		EntryAt: out.EntryAt,
		At:      out.At,
		Price:   out.Price,
		RLower:  out.RLower,
		Note:    out.Note,
	}, true
}
