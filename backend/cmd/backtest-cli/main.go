// Command backtest-cli menjalankan satu backtest dari terminal.
//
// Alat ini ada untuk memverifikasi mesin sebelum ada UI. Angka dari sini
// dibandingkan dengan pembacaan chart manual: bila tidak cocok, tidak ada
// gunanya membangun halaman apa pun di atasnya.
//
// Contoh:
//
//	go run ./cmd/backtest-cli -symbol XAUUSD -htf D1 -ltf H1 -bulan 3
//	go run ./cmd/backtest-cli -symbol XAUUSD -htf W -ltf H4 -bulan 6 -equity 10000 -risk 0.5
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"ila-strategy/ila"
	"ila-strategy/indicators"
	"ila/backend/internal/backtest"
	"ila/backend/internal/market"
	"ila/backend/internal/timezone"
)

// batasCandle adalah jumlah candle yang diminta per timeframe.
//
// Lima ribu dipilih setelah pengukuran langsung: TradingView memberi sampai
// sekitar enam ribu, tetapi di atas itu data yang dikirim mulai rusak
// (high < low) alih-alih memberi galat yang jelas. Lima ribu aman.
const batasCandle = 5000

func main() {
	symbol := flag.String("symbol", "XAUUSD", "simbol, mis. XAUUSD")
	htf := flag.String("htf", "D1", "timeframe HTF, mis. D1 atau W")
	ltf := flag.String("ltf", "H1", "timeframe LTF, mis. H1 atau H4")
	feed := flag.String("feed", "FOREXCOM", "sumber candle di TradingView")
	bulan := flag.Int("bulan", 3, "panjang rentang ke belakang, dalam bulan")
	equity := flag.Float64("equity", 10000, "modal awal")
	risk := flag.Float64("risk", 0.5, "risiko per trade dalam persen")
	maxTrade := flag.Int("max-trade", 2, "maksimum trade per hari (aturan live)")
	maxLoss := flag.Int("max-loss", 2, "maksimum loss per hari (aturan live)")
	flag.Parse()

	if err := jalankan(*symbol, *htf, *ltf, *feed, *bulan, *equity, *risk, *maxTrade, *maxLoss); err != nil {
		fmt.Fprintf(os.Stderr, "gagal: %v\n", err)
		os.Exit(1)
	}
}

func jalankan(symbol, htf, ltf, feed string, bulan int, equity, risk float64, maxTrade, maxLoss int) error {
	timezone.SetDaily("Asia/Jakarta")

	fmt.Printf("Mengambil candle %s %s->%s dari %s ...\n", symbol, htf, ltf, feed)
	penyedia := market.NewTradingViewProvider()

	ctx, batal := context.WithTimeout(context.Background(), 90*time.Second)
	defer batal()

	candleHTF, err := penyedia.FetchCandles(ctx, symbol, feed, htf, batasCandle)
	if err != nil {
		return fmt.Errorf("mengambil candle %s: %w", htf, err)
	}
	candleLTF, err := penyedia.FetchCandles(ctx, symbol, feed, ltf, batasCandle)
	if err != nil {
		return fmt.Errorf("mengambil candle %s: %w", ltf, err)
	}

	fmt.Printf("  %s: %d candle (%s s/d %s)\n", htf, len(candleHTF),
		waktu(candleHTF[0].Time), waktu(candleHTF[len(candleHTF)-1].Time))
	fmt.Printf("  %s: %d candle (%s s/d %s)\n", ltf, len(candleLTF),
		waktu(candleLTF[0].Time), waktu(candleLTF[len(candleLTF)-1].Time))

	// Rentang efektif adalah irisan kedua timeframe: evaluasi butuh keduanya,
	// jadi yang paling pendek yang menentukan.
	awalTersedia := candleHTF[0].Time
	if candleLTF[0].Time > awalTersedia {
		awalTersedia = candleLTF[0].Time
	}
	akhirTersedia := candleLTF[len(candleLTF)-1].Time

	diminta := time.Now().AddDate(0, -bulan, 0)
	awal := diminta.Unix()
	if awal < awalTersedia {
		awal = awalTersedia
	}

	if awal >= akhirTersedia {
		return fmt.Errorf("data tidak mencukupi: tersedia mulai %s", waktu(awalTersedia))
	}

	params := backtest.Params{
		Symbol:          symbol,
		HTF:             htf,
		LTF:             ltf,
		From:            time.Unix(awal, 0).UTC(),
		To:              time.Unix(akhirTersedia, 0).UTC(),
		Equity:          equity,
		RiskPercent:     risk,
		MaxTradesPerDay: maxTrade,
		MaxLossesPerDay: maxLoss,
		StopAfterWin:    true,
	}

	fmt.Printf("\nRentang efektif: %s s/d %s (%.0f hari)\n",
		waktu(awal), waktu(akhirTersedia),
		(float64(akhirTersedia-awal) / 86400))

	mesin := backtest.NewEngine()
	hasil, jendela, err := mesin.Run(params, keIndikator(candleHTF), keIndikator(candleLTF))
	if err != nil {
		return err
	}

	cetakHasil(hasil, jendela, htf, ltf)
	return nil
}

func cetakHasil(h *backtest.Result, jendela []backtest.CandleWindow, htf, ltf string) {
	fmt.Printf("\n%s\n", garis())
	fmt.Printf("HASIL BACKTEST  %s  (%s)\n", ila.Version, htf+"->"+ltf)
	fmt.Printf("%s\n", garis())

	if h.TotalTrades == 0 {
		fmt.Println("Tidak ada entry pada rentang ini.")
		if h.SignalsRejected > 0 {
			fmt.Printf("(%d setup ditolak karena RR di bawah minimum)\n", h.SignalsRejected)
		}
		return
	}

	fmt.Printf("Total trade      : %d\n", h.TotalTrades)
	fmt.Printf("Menang / kalah   : %d / %d\n", h.Wins, h.Losses)
	if h.Expired > 0 {
		fmt.Printf("Masih terbuka    : %d\n", h.Expired)
	}
	fmt.Printf("Win rate         : %.1f%%\n", h.WinRate*100)
	fmt.Printf("Total R          : %+.2f\n", h.TotalR)
	fmt.Printf("Max drawdown     : %.2f%%\n", h.MaxDrawdown*100)
	if h.ProfitFactor > 0 {
		fmt.Printf("Profit factor    : %.2f\n", h.ProfitFactor)
	}
	fmt.Printf("Sinyal ditolak   : %d (RR di bawah minimum)\n", h.SignalsRejected)
	if h.AmbiguousExits > 0 {
		fmt.Printf("Candle ambigu    : %d (TP dan SL tersentuh bersamaan, dihitung SL)\n", h.AmbiguousExits)
	}

	fmt.Printf("\n%-3s %-20s %-5s %-11s %-11s %-11s %-6s %-6s\n",
		"#", "Masuk", "Arah", "Entry", "SL", "TP", "Hasil", "R")
	for i, e := range h.Entries {
		if i >= 40 {
			fmt.Printf("... dan %d entry lainnya\n", len(h.Entries)-40)
			break
		}
		tanda := ""
		if e.Ambiguous {
			tanda = "*"
		}
		fmt.Printf("%-3d %-20s %-5s %-11.2f %-11.2f %-11.2f %-6s %+6.2f%s\n",
			e.Seq, e.EntryTime.Format("2006-01-02 15:04"), e.Direction,
			e.EntryPrice, e.StopLoss, e.TakeProfit, e.Outcome, e.ActualR, tanda)
	}

	if len(jendela) != len(h.Entries) {
		fmt.Printf("\nPERINGATAN: %d jendela candle untuk %d entry\n", len(jendela), len(h.Entries))
	}
}

func keIndikator(candles []market.Candle) []indicators.Candle {
	hasil := make([]indicators.Candle, len(candles))
	for i, c := range candles {
		hasil[i] = indicators.Candle{
			Timestamp: time.Unix(c.Time, 0).UTC(),
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
			IsClosed:  true,
		}
	}
	sort.SliceStable(hasil, func(i, j int) bool {
		return hasil[i].Timestamp.Before(hasil[j].Timestamp)
	})
	return hasil
}

func waktu(unix int64) string {
	return time.Unix(unix, 0).UTC().Format("2006-01-02 15:04")
}

func garis() string {
	return "------------------------------------------------------------------"
}
