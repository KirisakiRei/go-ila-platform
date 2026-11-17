package service

import (
	"context"
	"time"

	"ila/backend/internal/evalprice"
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
	// EntryAt adalah waktu harga pertama menyentuh level entry. Diisi juga untuk
	// hasil RUNNING sehingga durasi bisa dihitung sejak masuk, bukan sejak terbit.
	EntryAt *time.Time
	At      *time.Time
	Price   float64
	RLower  float64
	Note    string
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
	in := evalprice.InputDariSinyal(signal)
	if evalprice.CandleTerlaluTua(in, candles) {
		// Feed tidak lagi memuat masa berlaku setup ini. Menyimpulkan apa pun
		// akan mengarang; baris ditandai supaya tidak terbaca "tidak terjadi
		// apa-apa" selamanya.
		out := evalprice.TanpaCandle()
		return evalOutcome{Status: out.Status, Note: out.Note}, true
	}
	return evaluateCandles(signal, candles, since)
}

// evaluateCandles memisahkan perhitungan dari pengambilan data agar bisa diuji
// dengan candle buatan. Aturannya milik paket evalprice supaya jurnal dan
// runner tidak pernah menghasilkan angka berbeda untuk setup yang sama.
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

