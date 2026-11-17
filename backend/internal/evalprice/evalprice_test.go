package evalprice

import (
	"testing"
	"time"

	"ila/backend/internal/market"
	"ila/backend/internal/model"
)

// sinyalSELL menyusun setup SELL dengan level yang jelas.
func sinyalSELL(terbit time.Time) Input {
	return Input{
		Direction:   "SELL",
		EntryPrice:  101,
		StopLoss:    102,
		TakeProfit:  99,
		RiskReward:  2,
		PublishedAt: terbit,
		ExpiresAt:   terbit.Add(time.Hour),
	}
}

func lilin(unix int64, high, low float64) market.Candle {
	return market.Candle{Time: unix, High: high, Low: low}
}

// Kasus yang diminta: apakah TP atau SL duluan, beserta jam masuk dan keluar.
// Terbit Senin 15:00 UTC (pasar buka), entry tersentuh lalu TP.
func TestMenentukanTPBesertasJam(t *testing.T) {
	terbit := time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)
	in := sinyalSELL(terbit)

	out, ok := FromCandles(in, []market.Candle{
		lilin(terbit.Unix()+3600, 101.2, 100.8),  // menyentuh entry
		lilin(terbit.Unix()+7200, 100.9, 98.9),   // menembus TP 99
	})
	if !ok || out.Status != model.EvalTP {
		t.Fatalf("hasil = %+v ok=%v, ingin TP", out, ok)
	}
	if out.EntryAt == nil || out.EntryAt.Unix() != terbit.Unix()+3600 {
		t.Fatalf("jam entry = %v, ingin candle pertama yang menyentuh entry", out.EntryAt)
	}
	if out.At == nil || out.At.Unix() != terbit.Unix()+7200 {
		t.Fatalf("jam exit = %v, ingin candle yang menembus TP", out.At)
	}
	if out.RLower != 2 {
		t.Fatalf("R = %v, ingin 2", out.RLower)
	}
}

// Sebaliknya: SL yang kena lebih dulu.
func TestMenentukanSL(t *testing.T) {
	terbit := time.Date(2026, 11, 17, 15, 0, 0, 0, time.UTC)
	in := sinyalSELL(terbit)

	out, ok := FromCandles(in, []market.Candle{
		lilin(terbit.Unix()+3600, 101.1, 100.9),
		lilin(terbit.Unix()+7200, 102.5, 100.5),
	})
	if !ok || out.Status != model.EvalSL {
		t.Fatalf("hasil = %+v ok=%v, ingin SL", out, ok)
	}
	if out.RLower != -1 {
		t.Fatalf("R = %v, ingin -1", out.RLower)
	}
}

// Setup yang baru masuk tetapi belum keluar harus berstatus berjalan, bukan
// dinyatakan tidak tersentuh.
func TestMasihBerjalan(t *testing.T) {
	terbit := time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)
	in := sinyalSELL(terbit)
	out, ok := FromCandles(in, []market.Candle{
		lilin(terbit.Unix()+3600, 101.1, 100.9),
	})
	if !ok || out.Status != model.EvalRunning {
		t.Fatalf("hasil = %+v ok=%v, ingin RUNNING", out, ok)
	}
	if out.EntryAt == nil {
		t.Fatal("jam entry wajib terisi saat posisi berjalan")
	}
}

// Candle yang sudah jauh lebih baru dari sinyal berarti feed tidak lagi
// memuat waktunya; hasilnya dicatat tanpa data, bukan dikarang.
func TestCandleTerlaluTua(t *testing.T) {
	terbit := time.Date(2026, 11, 10, 15, 0, 0, 0, time.UTC)
	in := sinyalSELL(terbit)
	candles := []market.Candle{lilin(time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC).Unix(), 105, 100)}

	if !CandleTerlaluTua(in, candles) {
		t.Fatal("candle 16 Nov harus dianggap di luar jangkauan sinyal 10 Nov")
	}
	out := TanpaCandle()
	if out.Status != model.EvalUnknown || out.Note == "" {
		t.Fatalf("hasil = %+v, ingin UNKNOWN dengan keterangan", out)
	}
}

// Candle di luar jam pasar tidak boleh menilai apa pun.
func TestCandleLuarSesiDiabaikan(t *testing.T) {
	// Sabtu 14 Nov 2026 12:00 UTC: pasar tutup.
	sabtu := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	in := sinyalSELL(sabtu.Add(-10 * time.Minute))
	if _, ok := FromCandles(in, []market.Candle{lilin(sabtu.Unix(), 102.5, 98.5)}); ok {
		t.Fatal("candle hari Sabtu tidak boleh menghasilkan TP/SL")
	}
}

// Setup yang sudah tersentuh tidak boleh macet berjalan: ketika candle
// sesudahnya tersedia, hasilnya wajib menjadi final tanpa menunggu siapa pun.
func TestBerjalanMenjadiFinalSaatCandleBertambah(t *testing.T) {
	terbit := time.Date(2026, 11, 16, 15, 0, 0, 0, time.UTC)
	in := sinyalSELL(terbit)
	masuk := lilin(terbit.Unix()+3600, 101.2, 100.6)

	out, ok := FromCandles(in, []market.Candle{masuk})
	if !ok || out.Status != model.EvalRunning {
		t.Fatalf("hasil = %+v ok=%v, ingin RUNNING", out, ok)
	}

	out, ok = FromCandles(in, []market.Candle{
		masuk,
		lilin(terbit.Unix()+7200, 102.1, 100.5),
	})
	if !ok || out.Status != model.EvalSL {
		t.Fatalf("hasil = %+v ok=%v, ingin SL", out, ok)
	}
}
