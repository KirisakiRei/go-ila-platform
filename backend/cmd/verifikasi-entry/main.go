// Command verifikasi-entry memeriksa satu entry secara manual dari candle mentah.
//
// Alat ini ada untuk memastikan mesin backtest tidak salah menilai exit. Ia
// mengambil candle sungguhan mulai dari waktu masuk lalu mencetak open/high/low
// tiap candle, sehingga hasil simulasi bisa dibandingkan langsung dengan data.
//
// Contoh:
//
//	go run ./cmd/verifikasi-entry -ltf H1 -masuk "2026-05-20 01:00" \
//	  -entry 4481.48 -sl 4508.87 -tp 4418.50 -arah SELL
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"ila/backend/internal/market"
)

const (
	batasCandle       = 5000
	candleDitampilkan = 40
)

func main() {
	symbol := flag.String("symbol", "XAUUSD", "simbol")
	ltf := flag.String("ltf", "H1", "timeframe entry")
	feed := flag.String("feed", "FOREXCOM", "sumber candle")
	masuk := flag.String("masuk", "", "waktu masuk, format 2006-01-02 15:04 (UTC)")
	entry := flag.Float64("entry", 0, "harga entry")
	sl := flag.Float64("sl", 0, "stop loss")
	tp := flag.Float64("tp", 0, "take profit")
	arah := flag.String("arah", "BUY", "arah: BUY atau SELL")
	flag.Parse()

	if err := jalankan(*symbol, *ltf, *feed, *masuk, *entry, *sl, *tp, *arah); err != nil {
		fmt.Fprintf(os.Stderr, "gagal: %v\n", err)
		os.Exit(1)
	}
}

func jalankan(symbol, ltf, feed, masuk string, entry, sl, tp float64, arah string) error {
	waktuMasuk, err := time.Parse("2006-01-02 15:04", masuk)
	if err != nil {
		return fmt.Errorf("format waktu masuk: %w", err)
	}

	ctx, batal := context.WithTimeout(context.Background(), 90*time.Second)
	defer batal()

	candles, err := market.NewTradingViewProvider().FetchCandles(ctx, symbol, feed, ltf, batasCandle)
	if err != nil {
		return err
	}

	fmt.Printf("%s %s  entry %.2f  SL %.2f  TP %.2f  arah %s\n", symbol, ltf, entry, sl, tp, arah)
	fmt.Printf("Mulai dari: %s\n\n", waktuMasuk.Format("2006-01-02 15:04"))
	fmt.Printf("%-20s %-10s %-10s %-10s %-10s  %s\n", "waktu", "open", "high", "low", "close", "catatan")

	jumlah := 0
	for _, c := range candles {
		waktu := time.Unix(c.Time, 0).UTC()
		if waktu.Before(waktuMasuk) {
			continue
		}
		if jumlah >= candleDitampilkan {
			break
		}
		jumlah++

		kenaTP, kenaSL := false, false
		if arah == "BUY" {
			kenaTP = c.High >= tp
			kenaSL = c.Low <= sl
		} else {
			kenaTP = c.Low <= tp
			kenaSL = c.High >= sl
		}

		catatan := ""
		switch {
		case kenaTP && kenaSL:
			catatan = "AMBIGU: TP dan SL tersentuh -> dihitung SL"
		case kenaSL:
			catatan = "SL"
		case kenaTP:
			catatan = "TP"
		}

		fmt.Printf("%-20s %-10.2f %-10.2f %-10.2f %-10.2f  %s\n",
			waktu.Format("2006-01-02 15:04"), c.Open, c.High, c.Low, c.Close, catatan)

		if catatan != "" {
			return nil
		}
	}

	fmt.Printf("\nTidak tersentuh dalam %d candle pertama\n", jumlah)
	return nil
}
