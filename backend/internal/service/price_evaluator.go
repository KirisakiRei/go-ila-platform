package service

import (
	"context"
	"strings"
	"time"

	"ila/backend/internal/market"
	"ila/backend/internal/model"
)

// evalCandleLimit membatasi candle yang diambil. Cukup untuk menutup setup
// harian tanpa memuat riwayat panjang pada setiap pembacaan jurnal.
const evalCandleLimit = 500

// evalTimeout membatasi satu pengambilan candle. Feed yang lambat tidak boleh
// menahan pembacaan jurnal lebih lama daripada ini.
const evalTimeout = 15 * time.Second

// priceEvaluator menilai nasib hipotetis sebuah setup yang tidak dieksekusi:
// apakah harga menyentuh take profit lebih dulu, stop loss lebih dulu, atau
// tidak pernah menyentuh level entry sampai sinyal kedaluwarsa.
type priceEvaluator struct {
	candles market.Provider
	sink    EvalSink
}

// EvalSink menyimpan hasil hipotetis. Dipisahkan dari evaluator agar service
// tidak bergantung pada detail penyimpanan.
type EvalSink interface {
	SaveSignalEval(ctx context.Context, userID, signalID string, out *model.UserSignalEval) error
}

// evalOutcome adalah hasil penilaian satu setup.
type evalOutcome struct {
	Status string
	At     *time.Time
	Price  float64
	RLower float64
	Note   string
}

// evaluate menilai satu sinyal. Nilai balik kedua false berarti data harga
// belum memadai; pemanggil wajib membiarkan baris apa adanya.
func (e *priceEvaluator) evaluate(ctx context.Context, signal *model.Signal, since time.Time) (evalOutcome, bool) {
	if e == nil || signal == nil || signal.Ltf == "" || signal.EntryPrice <= 0 {
		return evalOutcome{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, evalTimeout)
	defer cancel()
	candles, err := e.candles.FetchCandles(ctx, signal.Symbol, signal.FeedSource, signal.Ltf, evalCandleLimit)
	if err != nil || len(candles) == 0 {
		return evalOutcome{}, false
	}
	return evaluateCandles(signal, candles, since)
}

// evaluateCandles memisahkan perhitungan dari pengambilan data agar bisa diuji
// dengan candle buatan.
//
// Urutan candle menentukan pemenang saat TP dan SL tersentuh pada lilin yang
// sama. Kasus itu dilaporkan UNKNOWN, bukan ditebak.
func evaluateCandles(signal *model.Signal, candles []market.Candle, since time.Time) (evalOutcome, bool) {
	buy := strings.EqualFold(signal.Direction, "BUY")
	sinceUnix := since.Unix()
	expiresUnix := signal.ExpiresAt.Unix()

	touched := false
	for _, c := range candles {
		if c.Time < sinceUnix {
			continue
		}
		if !touched {
			hitEntry := (buy && c.Low <= signal.EntryPrice) || (!buy && c.High >= signal.EntryPrice)
			if !hitEntry {
				// Harga tidak pernah menyentuh level entry sebelum kedaluwarsa:
				// setup tidak pernah berjalan, jadi TP/SL sesudahnya tidak
				// relevan untuk dinilai.
				if c.Time > expiresUnix {
					return evalOutcome{
						Status: model.EvalNoEntry,
						At:     utcPtr(time.Unix(expiresUnix, 0)),
						Note:   "harga tidak pernah menyentuh level entry",
					}, true
				}
				continue
			}
			touched = true
		}

		hitTP, hitSL := false, false
		if buy {
			hitTP = c.High >= signal.TakeProfit
			hitSL = c.Low <= signal.StopLoss
		} else {
			hitTP = c.Low <= signal.TakeProfit
			hitSL = c.High >= signal.StopLoss
		}

		switch {
		case hitTP && hitSL:
			return evalOutcome{
				Status: model.EvalUnknown,
				At:     utcPtr(time.Unix(c.Time, 0)),
				Note:   "TP dan SL tersentuh pada lilin yang sama",
			}, true
		case hitTP:
			return evalOutcome{
				Status: model.EvalTP,
				At:     utcPtr(time.Unix(c.Time, 0)),
				Price:  signal.TakeProfit,
				RLower: signal.RiskReward,
			}, true
		case hitSL:
			return evalOutcome{
				Status: model.EvalSL,
				At:     utcPtr(time.Unix(c.Time, 0)),
				Price:  signal.StopLoss,
				RLower: -1,
			}, true
		}
	}

	if touched {
		// Setup sudah tersentuh tetapi belum selesai: biarkan baris menunggu.
		return evalOutcome{}, false
	}
	if len(candles) > 0 && candles[len(candles)-1].Time > expiresUnix {
		return evalOutcome{
			Status: model.EvalNoEntry,
			At:     utcPtr(time.Unix(expiresUnix, 0)),
			Note:   "harga tidak pernah menyentuh level entry",
		}, true
	}
	return evalOutcome{}, false
}

func utcPtr(t time.Time) *time.Time {
	utc := t.UTC()
	return &utc
}
