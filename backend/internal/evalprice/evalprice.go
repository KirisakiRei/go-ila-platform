// Package evalprice menilai nasib setup yang tidak dieksekusi.
//
// Runner memakai checkpoint persisten untuk memantau forward test tanpa order
// broker. Jalur candle tanpa resolusi dipertahankan untuk kompatibilitas aturan
// penilaian historis; jurnal HTTP tidak menjalankan penilaian harga.
package evalprice

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

	"ila/backend/internal/market"
	"ila/backend/internal/model"
	"ila/backend/internal/timezone"
)

// Outcome adalah hasil penilaian satu setup.
type Outcome struct {
	Status    string
	EntryAt   *time.Time
	At        *time.Time
	Price     float64
	RLower    float64
	Note      string
	CheckedAt *time.Time
	Evidence  string
}

// Input adalah data sinyal yang dibutuhkan untuk menilai.
type Input struct {
	Direction     string
	EntryPrice    float64
	StopLoss      float64
	TakeProfit    float64
	RiskReward    float64
	PublishedAt   time.Time
	ExpiresAt     time.Time
	ResumeEntryAt *time.Time
	ResumeAfter   *time.Time
	CandleSeconds int64
}

// InputDariSinyal menyalin kolom sinyal yang dibutuhkan penilai.
func InputDariSinyal(s *model.Signal) Input {
	return Input{
		Direction:   s.Direction,
		EntryPrice:  s.EntryPrice,
		StopLoss:    s.StopLoss,
		TakeProfit:  s.TakeProfit,
		RiskReward:  s.RiskReward,
		PublishedAt: s.PublishedAt,
		ExpiresAt:   s.ExpiresAt,
	}
}

// FromCandles menilai setup memakai candle yang sudah tersedia.
//
// Nilai balik kedua false berarti hasilnya belum bisa dipastikan: pemanggil
// wajib mencoba lagi nanti, bukan menyimpulkan apa pun.
//
// Urutan candle menentukan pemenang saat TP dan SL tersentuh pada lilin yang
// sama. Kasus itu dilaporkan UNKNOWN, bukan ditebak.
func FromCandles(in Input, candles []market.Candle) (Outcome, bool) {
	if in.CandleSeconds > 0 {
		return forwardTest(in, candles)
	}
	if in.EntryPrice <= 0 || in.ExpiresAt.IsZero() {
		return Outcome{}, false
	}
	buy := strings.EqualFold(in.Direction, "BUY")
	expiresUnix := in.ExpiresAt.Unix()
	sinceUnix := in.PublishedAt.Unix()
	touched := false
	var entryAt *time.Time

	for _, c := range candles {
		if c.Time < sinceUnix {
			continue
		}
		// Candle di luar jam pasar tidak boleh menilai apa pun: pergerakan di
		// sana hanya sisa penutupan dan spread lebar.
		if !timezone.MarketExpectedOpen(time.Unix(c.Time, 0)) {
			continue
		}
		if !touched {
			hitEntry := (buy && c.Low <= in.EntryPrice) || (!buy && c.High >= in.EntryPrice)
			if !hitEntry {
				if c.Time > expiresUnix {
					return Outcome{
						Status: model.EvalNoEntry,
						At:     utcPtr(time.Unix(expiresUnix, 0)),
						Note:   "harga tidak pernah menyentuh level entry",
					}, true
				}
				continue
			}
			touched = true
			entryAt = utcPtr(time.Unix(c.Time, 0))
		}
		hitTP, hitSL := false, false
		if buy {
			hitTP = c.High >= in.TakeProfit
			hitSL = c.Low <= in.StopLoss
		} else {
			hitTP = c.Low <= in.TakeProfit
			hitSL = c.High >= in.StopLoss
		}
		switch {
		case hitTP && hitSL:
			return Outcome{
				Status:  model.EvalUnknown,
				EntryAt: entryAt,
				At:      utcPtr(time.Unix(c.Time, 0)),
				Note:    "TP dan SL tersentuh pada lilin yang sama",
			}, true
		case hitTP:
			return Outcome{
				Status:  model.EvalTP,
				EntryAt: entryAt,
				At:      utcPtr(time.Unix(c.Time, 0)),
				Price:   in.TakeProfit,
				RLower:  in.RiskReward,
			}, true
		case hitSL:
			return Outcome{
				Status:  model.EvalSL,
				EntryAt: entryAt,
				At:      utcPtr(time.Unix(c.Time, 0)),
				Price:   in.StopLoss,
				RLower:  -1,
			}, true
		}
	}

	if touched {
		// Sudah masuk tetapi belum keluar. Dicek sebelum kesimpulan lain agar
		// setup yang masih berjalan tidak dicap "tidak tersentuh".
		return Outcome{Status: model.EvalRunning, EntryAt: entryAt}, true
	}
	if len(candles) > 0 && candles[len(candles)-1].Time > expiresUnix {
		return Outcome{
			Status: model.EvalNoEntry,
			At:     utcPtr(time.Unix(expiresUnix, 0)),
			Note:   "harga tidak pernah menyentuh level entry",
		}, true
	}
	return Outcome{}, false
}

// CandleTerlaluTua melaporkan apakah candle tertua masih lebih baru daripada
// masa berlaku sinyal. Artinya feed sudah tidak memuat waktu setup itu, jadi
// hasilnya tidak akan pernah bisa dihitung.
func CandleTerlaluTua(in Input, candles []market.Candle) bool {
	if len(candles) == 0 || in.PublishedAt.IsZero() {
		return false
	}
	return time.Unix(candles[0].Time, 0).After(in.ExpiresAt)
}

// TanpaCandle adalah hasil untuk sinyal yang candle-nya sudah di luar jangkauan
// feed. Hasil tidak boleh dikarang: pelanggan diberi tahu datanya tak tersedia
// agar angka kosong tidak terbaca sebagai "tidak terjadi apa-apa".
func TanpaCandle() Outcome {
	return Outcome{
		Status: model.EvalUnknown,
		Note:   "data candle tidak tersedia karena sinyal sudah terlalu lama",
	}
}

// forwardTest models a theoretical fill at the published signal level. It does
// not claim broker execution or intrabar precision. CheckedAt is the end of the
// last fully observed candle, so a restart never replays an already checked bar.
func forwardTest(in Input, candles []market.Candle) (Outcome, bool) {
	buy := strings.EqualFold(in.Direction, "BUY")
	if (!buy && !strings.EqualFold(in.Direction, "SELL")) || in.PublishedAt.IsZero() || in.EntryPrice <= 0 || in.StopLoss <= 0 || in.TakeProfit <= 0 || math.IsNaN(in.EntryPrice+in.StopLoss+in.TakeProfit) || math.IsInf(in.EntryPrice+in.StopLoss+in.TakeProfit, 0) {
		return Outcome{}, false
	}
	if (buy && !(in.StopLoss < in.EntryPrice && in.EntryPrice < in.TakeProfit)) || (!buy && !(in.TakeProfit < in.EntryPrice && in.EntryPrice < in.StopLoss)) {
		return Outcome{}, false
	}
	out := Outcome{Status: model.EvalRunning, EntryAt: utcPtr(in.PublishedAt), CheckedAt: in.ResumeAfter}
	if in.ResumeEntryAt != nil {
		out.EntryAt = utcPtr(*in.ResumeEntryAt)
	}
	unknown := func(note string) (Outcome, bool) {
		out.Status, out.Note = model.EvalUnknown, note
		out.Evidence = `{"method":"signal_forward_v1","resolution":"candle","incomplete":true}`
		return out, true
	}
	if in.ResumeEntryAt != nil && in.ResumeAfter == nil {
		return unknown("Simulasi lama tidak memiliki checkpoint harga; hasil tidak dapat diteruskan memakai aturan entry saat publikasi.")
	}
	if !timezone.MarketExpectedOpen(*out.EntryAt) {
		return unknown("Sinyal terbit di luar sesi pasar; entry simulasi tidak dapat diverifikasi.")
	}
	bars := append([]market.Candle(nil), candles...)
	sort.Slice(bars, func(i, j int) bool { return bars[i].Time < bars[j].Time })
	cursor := in.PublishedAt.Unix()
	if in.ResumeAfter != nil {
		cursor = in.ResumeAfter.Unix()
	}
	for _, c := range bars {
		end := c.Time + in.CandleSeconds
		if end <= cursor {
			continue
		}
		if !timezone.MarketExpectedOpen(time.Unix(c.Time, 0)) {
			continue
		}
		if c.Time < cursor {
			// A partial publication candle is safe only when neither barrier
			// appears anywhere in its range. Otherwise the touch may predate entry.
			safe := c.Low > 0 && c.High >= c.Low && !math.IsNaN(c.High+c.Low) && !math.IsInf(c.High+c.Low, 0)
			if buy {
				safe = safe && c.High < in.TakeProfit && c.Low > in.StopLoss
			} else {
				safe = safe && c.Low > in.TakeProfit && c.High < in.StopLoss
			}
			if !safe {
				return unknown("Candle mencakup waktu sebelum sinyal terbit; urutan harga setelah entry tidak dapat dipastikan.")
			}
			out.CheckedAt = utcPtr(time.Unix(end, 0))
			cursor = end
			continue
		}
		// Weekend gaps are expected. Missing bars during an open session are
		// not: a skipped interval could have hit the opposite barrier first.
		for missing := cursor; missing < c.Time; missing += in.CandleSeconds {
			if timezone.MarketExpectedOpen(time.Unix(missing, 0)) {
				return unknown("Data candle tidak mencakup seluruh periode pemantauan; TP/SL lebih dahulu tidak dapat dipastikan.")
			}
		}
		if c.Low <= 0 || c.High < c.Low || math.IsNaN(c.High+c.Low) || math.IsInf(c.High+c.Low, 0) {
			return unknown("Data candle tidak valid; hasil tidak dapat dipastikan.")
		}
		out.CheckedAt = utcPtr(time.Unix(end, 0))
		hitTP, hitSL := c.Low <= in.TakeProfit, c.High >= in.StopLoss
		if buy {
			hitTP, hitSL = c.High >= in.TakeProfit, c.Low <= in.StopLoss
		}
		data, err := json.Marshal(struct {
			Method  string        `json:"method"`
			Seconds int64         `json:"candle_seconds"`
			Candle  market.Candle `json:"candle"`
		}{"signal_forward_v1", in.CandleSeconds, c})
		if err != nil {
			return unknown("Bukti harga tidak dapat disimpan.")
		}
		out.Evidence = string(data)
		if hitTP && hitSL {
			return unknown("TP dan SL tersentuh pada candle yang sama; urutannya tidak dapat dipastikan.")
		}
		if hitTP || hitSL {
			out.At = utcPtr(time.Unix(c.Time, 0))
			if hitTP {
				out.Status, out.Price = model.EvalTP, in.TakeProfit
				out.RLower = math.Abs(in.TakeProfit-in.EntryPrice) / math.Abs(in.EntryPrice-in.StopLoss)
			} else {
				out.Status, out.Price, out.RLower = model.EvalSL, in.StopLoss, -1
			}
			return out, true
		}
		cursor = end
	}
	if out.CheckedAt == nil {
		return Outcome{}, false
	}
	return out, true
}

func utcPtr(t time.Time) *time.Time {
	utc := t.UTC()
	return &utc
}
