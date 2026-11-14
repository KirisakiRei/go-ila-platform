package backtest

import (
	"testing"
	"time"

	"ila-strategy/indicators"
)

// TestJendelaHTFMemuatCandleExit menjaga panel timeframe acuan tetap akurat.
//
// Panel HTF harus memperlihatkan target atau batas rugi benar-benar tersentuh.
// Bila candle HTF berhenti sebelum exit, garis exit di chart menggantung di
// luar candle terakhir dan pembaca menyimpulkan harga tidak pernah sampai ke
// sana. Berbeda dengan LTF, kasus ini belum pernah diuji.
func TestJendelaHTFMemuatCandleExit(t *testing.T) {
	hari := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	harian := make([]indicators.Candle, 30)
	for i := range harian {
		harian[i] = indicators.Candle{
			Timestamp: hari.AddDate(0, 0, i),
			Open:      100, High: 101, Low: 99, Close: 100,
			IsClosed: true,
		}
	}

	// Empat jam per candle; posisi dibuka hari ke-10 dan ditutup hari ke-14.
	jam := make([]indicators.Candle, 400)
	for i := range jam {
		jam[i] = indicators.Candle{
			Timestamp: hari.Add(time.Duration(i) * 4 * time.Hour),
			Open:      100, High: 101, Low: 99, Close: 100,
			IsClosed: true,
		}
	}

	posisi := &Entry{
		Seq:        1,
		Direction:  DirectionBuy,
		EntryTime:  harian[10].Timestamp.Add(2 * time.Hour),
		ExitTime:   harian[14].Timestamp.Add(6 * time.Hour),
		EntryPrice: 100,
		StopLoss:   95,
		TakeProfit: 110,
		Outcome:    OutcomeTakeProfit,
	}

	jendela := jendelaChart(posisi, harian, jam)
	if len(jendela.HTF) == 0 {
		t.Fatal("jendela HTF kosong")
	}

	// Candle harian hari exit (hari ke-14) wajib ikut: candle itulah yang
	// menyentuh target, dan tanpa itu chart HTF bercerita lain.
	candleExit := harian[14].Timestamp
	var terakhir time.Time
	var adaExit bool
	for _, candle := range jendela.HTF {
		if candle.Timestamp.Equal(candleExit) {
			adaExit = true
		}
		terakhir = candle.Timestamp
	}
	if !adaExit {
		t.Fatalf("candle HTF pada hari exit (%s) tidak tersimpan; panel acuan berhenti di %s",
			candleExit.Format(time.RFC3339), terakhir.Format(time.RFC3339))
	}

	// Panel HTF tidak boleh memanjang terlalu jauh sesudah exit: sepuluh candle
	// harian sesudahnya masih wajar, lebih dari itu hanya ruang kosong.
	const batasSesudah = candleJendelaChartHTF
	hariTerakhir := (int(terakhir.Sub(hari).Hours() / 24)) // hari ke-berapa
	if hariTerakhir > 14+batasSesudah {
		t.Fatalf("jendela HTF memanjang sampai hari ke-%d, melewati batas %d hari sesudah exit",
			hariTerakhir, batasSesudah)
	}
}
