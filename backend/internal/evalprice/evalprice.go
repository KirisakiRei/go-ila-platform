// Package evalprice menilai nasib setup yang tidak dieksekusi.
//
// Satu aturan dipakai dua tempat: Runner saat sinyal berakhir (candle masih
// segar) dan jurnal saat baris lama dibaca. Dengan begitu angka di riwayat
// tidak bergantung pada siapa yang menghitung lebih dulu.
package evalprice

import (
	"strings"
	"time"

	"ila/backend/internal/market"
	"ila/backend/internal/model"
	"ila/backend/internal/timezone"
)

// Outcome adalah hasil penilaian satu setup.
type Outcome struct {
	Status  string
	EntryAt *time.Time
	At      *time.Time
	Price   float64
	RLower  float64
	Note    string
}

// Input adalah data sinyal yang dibutuhkan untuk menilai.
type Input struct {
	Direction   string
	EntryPrice  float64
	StopLoss    float64
	TakeProfit  float64
	RiskReward  float64
	PublishedAt time.Time
	ExpiresAt   time.Time
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

func utcPtr(t time.Time) *time.Time {
	utc := t.UTC()
	return &utc
}
