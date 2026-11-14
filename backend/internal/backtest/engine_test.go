package backtest

import (
	"testing"
	"time"

	"ila-strategy/indicators"
)

// buatCandle membuat candle tutup secara berurutan tiap jam.
func buatCandle(mulai time.Time, i int, open, high, low, close float64) indicators.Candle {
	return indicators.Candle{
		Timestamp: mulai.Add(time.Duration(i) * time.Hour),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		IsClosed:  true,
	}
}

// TestNilaiExitTPDanSL memastikan arah penetapan hasil benar untuk BUY dan SELL.
func TestNilaiExitTPDanSL(t *testing.T) {
	kasus := []struct {
		nama      string
		posisi    Entry
		candle    indicators.Candle
		harapExit bool
		harapOut  Outcome
	}{
		{
			nama:      "BUY kena TP",
			posisi:    Entry{Direction: DirectionBuy, EntryPrice: 100, StopLoss: 95, TakeProfit: 110},
			candle:    indicators.Candle{High: 111, Low: 99},
			harapExit: true,
			harapOut:  OutcomeTakeProfit,
		},
		{
			nama:      "BUY kena SL",
			posisi:    Entry{Direction: DirectionBuy, EntryPrice: 100, StopLoss: 95, TakeProfit: 110},
			candle:    indicators.Candle{High: 101, Low: 94},
			harapExit: true,
			harapOut:  OutcomeStopLoss,
		},
		{
			nama:      "SELL kena TP",
			posisi:    Entry{Direction: DirectionSell, EntryPrice: 100, StopLoss: 105, TakeProfit: 90},
			candle:    indicators.Candle{High: 101, Low: 89},
			harapExit: true,
			harapOut:  OutcomeTakeProfit,
		},
		{
			nama:      "SELL kena SL",
			posisi:    Entry{Direction: DirectionSell, EntryPrice: 100, StopLoss: 105, TakeProfit: 90},
			candle:    indicators.Candle{High: 106, Low: 99},
			harapExit: true,
			harapOut:  OutcomeStopLoss,
		},
		{
			nama:      "belum tersentuh",
			posisi:    Entry{Direction: DirectionBuy, EntryPrice: 100, StopLoss: 95, TakeProfit: 110},
			candle:    indicators.Candle{High: 102, Low: 98},
			harapExit: false,
		},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			selesai, _, outcome, _ := nilaiExit(&k.posisi, k.candle)
			if selesai != k.harapExit {
				t.Fatalf("selesai: dapat %v, harap %v", selesai, k.harapExit)
			}
			if k.harapExit && outcome != k.harapOut {
				t.Fatalf("outcome: dapat %v, harap %v", outcome, k.harapOut)
			}
		})
	}
}

// TestCandleAmbiguDianggapSL adalah janji paling penting di paket ini.
//
// Saat satu candle menyentuh TP dan SL sekaligus, urutannya tidak bisa
// diketahui tanpa data tick. Memilih SL membuat hasil cenderung meremehkan,
// dan itu disengaja: backtest yang terlalu optimis membuat orang memakai
// metode yang sebenarnya tidak layak.
func TestCandleAmbiguDianggapSL(t *testing.T) {
	p := Entry{Direction: DirectionBuy, EntryPrice: 100, StopLoss: 95, TakeProfit: 110}
	candle := indicators.Candle{High: 111, Low: 94} // menyentuh keduanya

	selesai, harga, outcome, ambigu := nilaiExit(&p, candle)
	if !selesai {
		t.Fatal("seharusnya selesai")
	}
	if outcome != OutcomeStopLoss {
		t.Fatalf("outcome: dapat %v, harap SL", outcome)
	}
	if !ambigu {
		t.Fatal("kasus ini harus ditandai ambigu agar bisa dilaporkan")
	}
	if harga != p.StopLoss {
		t.Fatalf("harga exit: dapat %v, harap %v", harga, p.StopLoss)
	}
}

// TestPotongSampaiTidakMembocorkanMasaDepan memastikan pengaman anti
// look-ahead benar-benar memotong data.
func TestPotongSampaiTidakMembocorkanMasaDepan(t *testing.T) {
	mulai := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := []indicators.Candle{
		buatCandle(mulai, 0, 1, 1, 1, 1),
		buatCandle(mulai, 1, 1, 1, 1, 1),
		buatCandle(mulai, 2, 1, 1, 1, 1),
		buatCandle(mulai, 3, 1, 1, 1, 1),
	}

	batas := mulai.Add(2 * time.Hour)
	hasil := potongSampai(candles, batas)
	if len(hasil) != 2 {
		t.Fatalf("dapat %d candle, harap 2", len(hasil))
	}
	for _, c := range hasil {
		if !c.Timestamp.Before(batas) {
			t.Fatalf("candle %v bocor melewati batas %v", c.Timestamp, batas)
		}
	}
}

// TestAturanHarianMengikutiLive memastikan pagar harian sama dengan runner.
func TestAturanHarianMengikutiLive(t *testing.T) {
	hari := time.Date(2026, 6, 3, 8, 0, 0, 0, time.UTC)
	penjaga := aturanHarian{maksTrade: 2, maksLoss: 2, stopAfterWin: true}

	if !penjaga.bolehBuka(hari) {
		t.Fatal("trade pertama seharusnya boleh")
	}
	penjaga.catat(OutcomeTakeProfit, hari)

	// Berhenti setelah menang: trade kedua di hari yang sama ditolak.
	if penjaga.bolehBuka(hari) {
		t.Fatal("setelah menang seharusnya berhenti untuk hari itu")
	}

	// Hari berikutnya boleh lagi.
	besok := hari.Add(24 * time.Hour)
	if !penjaga.bolehBuka(besok) {
		t.Fatal("hari berikutnya seharusnya boleh")
	}
}

// TestAturanHarianBerhentiSetelahDuaLoss memastikan pagar loss ditegakkan.
func TestAturanHarianBerhentiSetelahDuaLoss(t *testing.T) {
	hari := time.Date(2026, 6, 3, 8, 0, 0, 0, time.UTC)
	penjaga := aturanHarian{maksTrade: 3, maksLoss: 2, stopAfterWin: true}

	penjaga.catat(OutcomeStopLoss, hari)
	penjaga.catat(OutcomeStopLoss, hari)

	if penjaga.bolehBuka(hari) {
		t.Fatal("setelah dua loss seharusnya berhenti untuk hari itu")
	}
}

// TestTutupPosisiMenghitungModal memastikan modal bergerak sesuai R.
//
// Perhatikan bahwa risiko nominal dihitung dari modal yang BERJALAN, bukan dari
// modal awal. Setelah satu trade menang, 1% dari modal berikutnya ikut naik —
// begitulah cara kerja risiko persentase di dunia nyata, dan begitulah runner
// live menghitungnya.
func TestTutupPosisiMenghitungModal(t *testing.T) {
	params := Params{Equity: 10_000, RiskPercent: 1}
	modal := 10_000.0
	waktu := time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)

	menang := Entry{Direction: DirectionBuy, EntryPrice: 100, StopLoss: 95, TakeProfit: 110, PlannedRR: 2, EntryTime: waktu}
	tutupPosisi(&menang, waktu.Add(time.Hour), 110, OutcomeTakeProfit, false, params, &modal)
	// Risiko 1% dari 10.000 = 100; menang 2R = +200.
	if modal != 10_200 {
		t.Fatalf("modal setelah TP: dapat %v, harap 10200", modal)
	}

	kalah := Entry{Direction: DirectionBuy, EntryPrice: 100, StopLoss: 95, TakeProfit: 110, PlannedRR: 2, EntryTime: waktu}
	tutupPosisi(&kalah, waktu.Add(2*time.Hour), 95, OutcomeStopLoss, false, params, &modal)
	// Risiko 1% dari 10.200 = 102; kalah 1R = -102.
	if modal != 10_098 {
		t.Fatalf("modal setelah SL: dapat %v, harap 10098", modal)
	}
}
