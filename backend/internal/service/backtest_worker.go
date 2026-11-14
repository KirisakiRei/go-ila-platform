package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"
	"time"

	"ila-strategy/indicators"

	"ila/backend/internal/backtest"
	"ila/backend/internal/market"
	"ila/backend/internal/model"
)

// jumlahWorkerBacktest menentukan berapa job yang boleh berjalan serentak.
//
// Satu. Bukan karena ragu soal CPU, tetapi karena satu job sudah memegang
// deret candle dua timeframe (sampai 5000 baris masing-masing) di memori.
// Menjalankan beberapa sekaligus di VPS 900 MB berarti risiko container lain
// ikut dimatikan.
const jumlahWorkerBacktest = 1

// jedaPindaiAntrean adalah selang pemindaian basis data bila tidak ada
// pemberitahuan lewat kanal. Ini jaring pengaman untuk job yang dibuat proses
// lain atau kanal yang penuh.
const jedaPindaiAntrean = 15 * time.Second

// RunBacktestWorker menjalankan antrean backtest sampai konteks dibatalkan.
//
// Pola hidupnya sama dengan pengirim surel: satu goroutine, berhenti saat
// konteks induk selesai, tidak membutuhkan container baru.
func RunBacktestWorker(ctx context.Context, svc *BacktestService) {
	logger := svc.logger
	logger.Info("worker backtest aktif", "jumlah", jumlahWorkerBacktest)

	pemicu := make(chan struct{}, 1)
	svc.pemicuAntrean = pemicu

	// Teruskan notifikasi dari Buat ke loop pemindaian.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-svc.antrean:
				select {
				case pemicu <- struct{}{}:
				default:
				}
			}
		}
	}()

	penanda := time.NewTicker(jedaPindaiAntrean)
	defer penanda.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker backtest berhenti")
			return
		case <-pemicu:
		case <-penanda.C:
		}
		svc.prosesAntrean(ctx)
	}
}

// prosesAntrean mengerjakan job yang menunggu sampai antrean kosong.
func (s *BacktestService) prosesAntrean(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		daftar, err := s.runs.AmbilAntrean(ctx, jumlahWorkerBacktest)
		if err != nil {
			s.logger.Error("membaca antrean backtest", "error", err.Error())
			return
		}
		if len(daftar) == 0 {
			return
		}

		for i := range daftar {
			if err := s.jalankanSatu(ctx, &daftar[i]); err != nil {
				s.logger.Error("backtest gagal", "run", daftar[i].ID, "error", err.Error())
			}
		}
	}
}

// jalankanSatu mengeksekusi satu run dari awal sampai hasil tersimpan.
func (s *BacktestService) jalankanSatu(ctx context.Context, run *model.BacktestRun) (errKembali error) {
	defer func() {
		// Satu data yang aneh tidak boleh menjatuhkan seluruh proses: worker ini
		// berjalan di dalam backend-api, jadi panic di sini berarti situs mati.
		if r := recover(); r != nil {
			s.logger.Error("backtest panic", "run", run.ID, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			errKembali = fmt.Errorf("panic: %v", r)
			s.tandaiGagal(ctx, run.ID, "Terjadi kesalahan saat menjalankan simulasi.")
		}
	}()

	ctxJob, batal := context.WithTimeout(ctx, batasDurasiSatuJob)
	defer batal()

	if err := s.runs.PerbaruiStatus(ctxJob, run.ID, model.BacktestStatusRunning, ""); err != nil {
		return err
	}
	mulai := time.Now()
	s.logger.Info("backtest mulai", "run", run.ID, "symbol", run.Symbol, "htf", run.HTF, "ltf", run.LTF)

	candleHTF, candleLTF, err := s.ambilCandleRun(ctxJob, run)
	if err != nil {
		s.logger.Warn("backtest gagal mengambil data", "run", run.ID, "error", err.Error())
		s.tandaiGagal(ctxJob, run.ID, err.Error())
		return nil
	}

	params := backtest.Params{
		Symbol:      run.Symbol,
		HTF:         run.HTF,
		LTF:         run.LTF,
		From:        run.RangeFrom,
		To:          run.RangeTo,
		Equity:      run.Equity,
		RiskPercent: run.RiskPercent,
		// Aturan harian dikunci ke nilai live: backtest yang memakai aturan
		// berbeda dari runner tidak bisa dipakai untuk menilai metode.
		MaxTradesPerDay: aturanHarianLive.MaksTrade,
		MaxLossesPerDay: aturanHarianLive.MaksLoss,
		StopAfterWin:    true,
	}

	hasil, jendela, err := backtest.NewEngine().Run(params, candleHTF, candleLTF)
	if err != nil {
		s.logger.Warn("backtest gagal dijalankan", "run", run.ID, "error", err.Error())
		s.tandaiGagal(ctxJob, run.ID, err.Error())
		return nil
	}

	run.Status = model.BacktestStatusDone
	run.TotalTrades = hasil.TotalTrades
	run.Wins = hasil.Wins
	run.Losses = hasil.Losses
	run.Expired = hasil.Expired
	run.AmbiguousExits = hasil.AmbiguousExits
	run.SignalsRejected = hasil.SignalsRejected
	run.WinRate = hasil.WinRate
	run.TotalR = hasil.TotalR
	run.MaxDrawdown = hasil.MaxDrawdown
	run.ProfitFactor = hasil.ProfitFactor
	selesai := time.Now()
	run.FinishedAt = &selesai

	if kurva, err := json.Marshal(hasil.EquityCurve); err == nil {
		run.EquityCurveJSON = string(kurva)
	}

	trades := s.susunTrade(run, hasil, jendela)
	if err := s.runs.SimpanHasil(ctxJob, run, trades); err != nil {
		s.logger.Error("menyimpan hasil backtest", "run", run.ID, "error", err.Error())
		s.tandaiGagal(ctxJob, run.ID, "Hasil simulasi gagal disimpan.")
		return nil
	}

	s.logger.Info("backtest selesai", "run", run.ID,
		"trade", hasil.TotalTrades, "durasi_detik", int(time.Since(mulai).Seconds()))
	return nil
}

// pengaturanAturanHarian mengikuti runner live.
//
// Disalin sebagai nilai, bukan dibaca dari konfigurasi runner: backtest harus
// tetap bisa dijalankan meski runner sedang mati, dan angkanya memang sengaja
// dikunci (lihat plans/26 §2.5).
var aturanHarianLive = struct {
	MaksTrade int
	MaksLoss  int
}{MaksTrade: 2, MaksLoss: 2}

// ambilCandleRun mengambil candle kedua timeframe dari feed.
//
// Pemanasan ditambahkan di depan: evaluator butuh riwayat untuk mengenali
// struktur, dan tanpa itu hari-hari awal rentang akan selalu kosong.
func (s *BacktestService) ambilCandleRun(ctx context.Context, run *model.BacktestRun) ([]indicators.Candle, []indicators.Candle, error) {
	ctxAmbil, batal := context.WithTimeout(ctx, 60*time.Second)
	defer batal()

	candleHTF, err := s.provider.FetchCandles(ctxAmbil, run.Symbol, "", run.HTF, batasCandleBacktest)
	if err != nil {
		return nil, nil, fmt.Errorf("mengambil candle %s: %w", run.HTF, err)
	}
	candleLTF, err := s.provider.FetchCandles(ctxAmbil, run.Symbol, "", run.LTF, batasCandleBacktest)
	if err != nil {
		return nil, nil, fmt.Errorf("mengambil candle %s: %w", run.LTF, err)
	}
	if len(candleHTF) < backtest.MinimalCandleHTF || len(candleLTF) < backtest.MinimalCandleLTF {
		return nil, nil, fmt.Errorf("data tidak cukup: %s %d candle, %s %d candle",
			run.HTF, len(candleHTF), run.LTF, len(candleLTF))
	}
	return ubahKeIndikator(candleHTF), ubahKeIndikator(candleLTF), nil
}

func (s *BacktestService) susunTrade(run *model.BacktestRun, hasil *backtest.Result, jendela []backtest.CandleWindow) []model.BacktestTrade {
	trades := make([]model.BacktestTrade, 0, len(hasil.Entries))
	for i := range hasil.Entries {
		entry := hasil.Entries[i]

		var candleJSON, levelJSON string
		if i < len(jendela) {
			if bahan, err := json.Marshal(jendela[i]); err == nil {
				candleJSON = ringkasJendela(string(bahan))
			}
			if bahan, err := json.Marshal(entry.Levels); err == nil {
				levelJSON = string(bahan)
			}
		}

		trades = append(trades, model.BacktestTrade{
			RunID:       run.ID,
			Seq:         entry.Seq,
			Direction:   entry.Direction,
			EntryTime:   entry.EntryTime,
			ExitTime:    entry.ExitTime,
			EntryPrice:  entry.EntryPrice,
			StopLoss:    entry.StopLoss,
			TakeProfit:  entry.TakeProfit,
			PlannedRR:   entry.PlannedRR,
			Outcome:     string(entry.Outcome),
			ActualR:     entry.ActualR,
			Duration:    int(entry.Duration.Seconds()),
			EquityAfter: entry.EquityAfter,
			Ambiguous:   entry.Ambiguous,
			Summary:     entry.Summary,
			NextAction:  entry.NextAction,
			CandlesJSON: candleJSON,
			LevelsJSON:  levelJSON,
		})
	}
	return trades
}

// ringkasJendela memastikan candle yang disimpan hanya berisi empat puluh
// lilin di sekitar entry (dua puluh kiri, dua puluh kanan).
//
// Batas ini yang membuat tabel trade tetap ringan: tanpa pemotongan, satu run
// dengan lima puluh entry bisa menyimpan puluhan megabyte teks.
func ringkasJendela(bahan string) string {
	var jendela struct {
		HTF []indicators.Candle `json:"htf"`
		LTF []indicators.Candle `json:"ltf"`
	}
	if err := json.Unmarshal([]byte(bahan), &jendela); err != nil {
		return ""
	}
	batasi := func(candles []indicators.Candle) []indicators.Candle {
		const maksimal = 80
		if len(candles) <= maksimal {
			return candles
		}
		return candles[len(candles)-maksimal:]
	}
	jendela.HTF = batasi(jendela.HTF)
	jendela.LTF = batasi(jendela.LTF)

	hasil, err := json.Marshal(jendela)
	if err != nil {
		return ""
	}
	return string(hasil)
}

func (s *BacktestService) tandaiGagal(ctx context.Context, runID, pesan string) {
	// Konteks job bisa sudah dibatalkan saat gagal (mis. timeout), jadi status
	// gagal ditulis lewat konteks baru yang pendek agar tetap tercatat.
	ctxCatat, batal := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer batal()
	if err := s.runs.PerbaruiStatus(ctxCatat, runID, model.BacktestStatusFailed, pesan); err != nil {
		s.logger.Error("menandai backtest gagal", "run", runID, "error", err.Error())
	}
}

// SweeperBacktestMembersihkan menjalankan pembersihan berkala.
//
// Aturan umurnya disusun supaya data yang berat hilang lebih dulu: candle dan
// level dibuang setelah tujuh hari, sementara statistik run tetap disimpan lama
// agar riwayat bisa dibaca.
func (s *BacktestService) Sweeper(ctx context.Context, logger *slog.Logger) {
	const (
		umurCandleHari  = 7
		umurGagalJam    = 24
		umurSelesaiHari = 90
		umurKuotaHari   = 30
	)

	sekarang := time.Now().UTC()

	if jumlah, err := s.runs.BersihkanCandleLama(ctx, sekarang.AddDate(0, 0, -umurCandleHari)); err != nil {
		logger.Warn("membersihkan candle backtest", "error", err.Error())
	} else if jumlah > 0 {
		logger.Info("candle backtest dibersihkan", "trade", jumlah)
	}

	if jumlah, err := s.runs.HapusRunGagal(ctx, sekarang.Add(-umurGagalJam*time.Hour)); err != nil {
		logger.Warn("menghapus backtest gagal", "error", err.Error())
	} else if jumlah > 0 {
		logger.Info("backtest gagal dihapus", "run", jumlah)
	}

	if jumlah, err := s.runs.HapusRunLama(ctx, sekarang.AddDate(0, 0, -umurSelesaiHari)); err != nil {
		logger.Warn("menghapus backtest lama", "error", err.Error())
	} else if jumlah > 0 {
		logger.Info("backtest lama dihapus", "run", jumlah)
	}

	if jumlah, err := s.runs.HapusKuotaLama(ctx, sekarang.AddDate(0, 0, -umurKuotaHari)); err != nil {
		logger.Warn("menghapus kuota backtest lama", "error", err.Error())
	} else if jumlah > 0 {
		logger.Info("baris kuota backtest dihapus", "baris", jumlah)
	}
}

// RunBacktestSweeper menjalankan penyapu setiap enam jam.
func RunBacktestSweeper(ctx context.Context, svc *BacktestService, logger *slog.Logger) {
	penanda := time.NewTicker(6 * time.Hour)
	defer penanda.Stop()
	for {
		svc.Sweeper(ctx, logger)
		select {
		case <-ctx.Done():
			return
		case <-penanda.C:
		}
	}
}

// ubahKeIndikator menyeragamkan candle feed menjadi bentuk yang dipakai
// evaluator, sekaligus memastikan urut naik.
func ubahKeIndikator(candles []market.Candle) []indicators.Candle {
	hasil := make([]indicators.Candle, len(candles))
	for i, candle := range candles {
		hasil[i] = indicators.Candle{
			Timestamp: time.Unix(candle.Time, 0).UTC(),
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
			IsClosed:  true,
		}
	}
	sort.SliceStable(hasil, func(i, j int) bool {
		return hasil[i].Timestamp.Before(hasil[j].Timestamp)
	})
	return hasil
}
