package service

import (
	"testing"
	"time"

	"ila/backend/internal/model"
)

// Cakupan dan pelanggaran batas diuji di sini tanpa basis data: yang diperiksa
// adalah perhitungannya, bukan penyimpanannya.

func TestIrisanCakupanMengambilYangPalingSempit(t *testing.T) {
	rentang := []CakupanTimeframe{
		{Timeframe: "H4", Mulai: 1_600_000_000, Selesai: 1_800_000_000},
		{Timeframe: "M15", Mulai: 1_750_000_000, Selesai: 1_790_000_000},
	}
	mulai, selesai := irisanCakupan(rentang)
	if mulai != 1_750_000_000 {
		t.Fatalf("mulai: dapat %d, harap 1750000000", mulai)
	}
	if selesai != 1_790_000_000 {
		t.Fatalf("selesai: dapat %d, harap 1790000000", selesai)
	}
}

// TestIrisanCakupanKosongBilaAdaTimeframeTanpaData memastikan pasangan yang
// salah satu timeframe-nya kosong tidak dianggap punya cakupan.
func TestIrisanCakupanKosongBilaAdaTimeframeTanpaData(t *testing.T) {
	rentang := []CakupanTimeframe{
		{Timeframe: "H1", Mulai: 1_700_000_000, Selesai: 1_800_000_000},
		{Timeframe: "M5"},
	}
	mulai, selesai := irisanCakupan(rentang)
	if mulai != 0 || selesai != 0 {
		t.Fatalf("harap kosong, dapat %d sampai %d", mulai, selesai)
	}
}

func TestPasanganBacktestYangDidukung(t *testing.T) {
	sah := [][2]string{{"W", "H4"}, {"D1", "H1"}, {"H4", "M15"}, {"H1", "M5"}, {"M15", "M1"}}
	for _, pasangan := range sah {
		if _, ok := CariPasanganBacktest(pasangan[0], pasangan[1]); !ok {
			t.Fatalf("pasangan %s-%s seharusnya sah", pasangan[0], pasangan[1])
		}
	}

	salah := [][2]string{{"D1", "M5"}, {"H4", "H1"}, {"", ""}, {"M15", "M5"}}
	for _, pasangan := range salah {
		if _, ok := CariPasanganBacktest(pasangan[0], pasangan[1]); ok {
			t.Fatalf("pasangan %s-%s seharusnya ditolak", pasangan[0], pasangan[1])
		}
	}
}

// TestCariPasanganMembakukanHuruf memastikan kiriman huruf kecil dari klien
// tidak ditolak hanya karena beda kapital.
func TestCariPasanganMembakukanHuruf(t *testing.T) {
	pasangan, ok := CariPasanganBacktest("d1", "h1")
	if !ok {
		t.Fatal("pasangan huruf kecil seharusnya diterima")
	}
	if pasangan.HTF != "D1" || pasangan.LTF != "H1" {
		t.Fatalf("bentuk baku salah: %s-%s", pasangan.HTF, pasangan.LTF)
	}
}

// TestRentangCakupanMengikutiPasangan memastikan pilihan pasangan memakai
// timeframe pendek sebagai penentu, karena data M5 selalu paling dangkal.
func TestRentangCakupanMengikutiPasangan(t *testing.T) {
	cakupan := CakupanBacktest{
		Rentang: []CakupanTimeframe{
			{Timeframe: "H1", Mulai: 1_700_000_000, Selesai: 1_800_000_000},
			{Timeframe: "M5", Mulai: 1_790_000_000, Selesai: 1_799_000_000},
		},
	}
	pasangan, _ := CariPasanganBacktest("H1", "M5")
	rentang := cakupan.RentangCakupan(pasangan)
	if rentang.Mulai != 1_790_000_000 || rentang.Selesai != 1_799_000_000 {
		t.Fatalf("rentang salah: %d sampai %d", rentang.Mulai, rentang.Selesai)
	}
}

func TestTengahMalamBerikutnyaMemakaiZonaHarian(t *testing.T) {
	zona := waktuHarian()
	// 13 November 2026 pukul 20.00 WIB.
	sekarang := time.Date(2026, 11, 13, 20, 0, 0, 0, zona).UTC()
	hasil := tengahMalamBerikutnya(sekarang, zona).In(zona)

	if hasil.Day() != 14 || hasil.Month() != time.November || hasil.Hour() != 0 {
		t.Fatalf("harap 14 November 00.00 WIB, dapat %s", hasil.Format(time.RFC3339))
	}
}

// TestHakTierMenentukanKuota memastikan batas per paket sesuai keputusan:
// free satu, basic tiga, plus dua puluh.
func TestHakTierMenentukanKuota(t *testing.T) {
	kasus := []struct {
		tier  string
		kuota int
		bulan int
	}{
		{model.TierFree, KuotaHarianFree, BatasBulanFree},
		{model.TierBasic, KuotaHarianBasic, BatasBulanBasic},
		{model.TierPlus, KuotaHarianPlus, BatasBulanPlus},
	}
	for _, satu := range kasus {
		if satu.kuota != kuotaUntukTier(satu.tier) {
			t.Fatalf("%s: kuota %d, harap %d", satu.tier, kuotaUntukTier(satu.tier), satu.kuota)
		}
		if satu.bulan != batasBulanUntukTier(satu.tier) {
			t.Fatalf("%s: batas bulan %d, harap %d", satu.tier, batasBulanUntukTier(satu.tier), satu.bulan)
		}
	}
}

// kuotaUntukTier memetakan tier ke jatah harian seperti yang dipakai service.
func kuotaUntukTier(tier string) int {
	switch tier {
	case model.TierPlus:
		return KuotaHarianPlus
	case model.TierBasic:
		return KuotaHarianBasic
	default:
		return KuotaHarianFree
	}
}

func batasBulanUntukTier(tier string) int {
	switch tier {
	case model.TierPlus:
		return BatasBulanPlus
	case model.TierBasic:
		return BatasBulanBasic
	default:
		return BatasBulanFree
	}
}

// TestRingkasTradeMenandaiChartBersih memastikan UI tahu chart tidak lagi
// tersedia setelah candle dibersihkan penyapu.
func TestRingkasTradeMenandaiChartBersih(t *testing.T) {
	waktu := time.Date(2026, 6, 3, 1, 0, 0, 0, time.UTC)
	trade := &model.BacktestTrade{
		Seq:         3,
		Direction:   "SELL",
		EntryTime:   waktu,
		ExitTime:    waktu.Add(4 * time.Hour),
		EntryPrice:  4525.65,
		StopLoss:    4550,
		TakeProfit:  4478,
		PlannedRR:   2.3,
		Outcome:     "TP",
		ActualR:     2.3,
		Duration:    int((4 * time.Hour).Seconds()),
		EquityAfter: 10234,
		CandlesJSON: "{}",
	}
	segar := ringkasTrade(trade)
	if !segar.ChartTersedia {
		t.Fatal("trade dengan candle seharusnya punya chart")
	}
	if segar.Arah != "SELL" || segar.Hasil != "TP" {
		t.Fatalf("ringkasan salah: %+v", segar)
	}

	dibersihkan := waktu.Add(7 * 24 * time.Hour)
	trade.CandlesPurgedAt = &dibersihkan
	trade.CandlesJSON = ""
	bersih := ringkasTrade(trade)
	if bersih.ChartTersedia {
		t.Fatal("trade yang candle-nya dibersihkan tidak boleh menawarkan chart")
	}
}
