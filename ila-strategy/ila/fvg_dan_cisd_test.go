package ila

import (
	"testing"
	"time"

	"ila-strategy/indicators"
)

// Berkas ini mengunci dua aturan yang sebelumnya membuat backtest mengeksekusi
// zona yang sudah tidak relevan:
//
//  1. CISD hanya lahir dari deretan candle yang benar-benar ada. Dulu ada blok
//     cadangan yang meminjam candle dari Candle 4, sehingga candle ekstrem yang
//     tidak pernah bagian dari deretan mana pun bisa menjadi "penembus".
//  2. FVG acuan harus yang paling muda, masih dijangkau harga, dan tidak lebih
//     tua dari empat candle HTF.

// TestCISDTidakTerbentukTanpaDeretan mengunci kasus USTEC 9 Okt 2026 21:00 NY:
// Candle 5 baru berisi satu candle bearish, jadi tidak ada deretan bullish sama
// sekali dan CISD sell tidak boleh terbentuk.
//
// Sebelum diperbaiki, blok cadangan menelusuri mundur melewati Candle 5 masuk
// ke Candle 4, mengambil candle bullish di sana, lalu menjadikan candle bearish
// di Candle 5 sebagai penembusnya. Lahirlah entry dengan SL hanya 8,7 poin.
func TestCISDTidakTerbentukTanpaDeretan(t *testing.T) {
	eval := NewEvaluator()
	awalCandle5 := time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC)

	sweepCandle := indicators.Candle{
		Timestamp: time.Date(2026, 10, 8, 21, 0, 0, 0, time.UTC),
		Open:      30781.10, High: 30840.10, Low: 30721.40, Close: 30760.60, IsClosed: true,
	}

	ltfCandles := []indicators.Candle{
		// Candle 4: deretan turun menuju titik terendah.
		{Timestamp: awalCandle5.Add(-60 * time.Minute), Open: 30764.10, High: 30767.50, Low: 30736.10, Close: 30750.90, IsClosed: true},
		{Timestamp: awalCandle5.Add(-45 * time.Minute), Open: 30750.90, High: 30770.50, Low: 30721.40, Close: 30736.00, IsClosed: true},
		{Timestamp: awalCandle5.Add(-30 * time.Minute), Open: 30736.00, High: 30764.60, Low: 30732.50, Close: 30764.10, IsClosed: true},
		{Timestamp: awalCandle5.Add(-15 * time.Minute), Open: 30764.10, High: 30797.50, Low: 30763.10, Close: 30796.20, IsClosed: true},
		// Candle 5: satu-satunya candle, dan ia bearish.
		{Timestamp: awalCandle5, Open: 30796.20, High: 30801.90, Low: 30756.70, Close: 30760.60, IsClosed: true},
		{Timestamp: awalCandle5.Add(15 * time.Minute), Open: 30760.60, High: 30764.70, Low: 30743.70, Close: 30756.00, IsClosed: true},
	}

	cisd, invalidated := eval.DetectDynamicCISD(ltfCandles, "BEARISH", awalCandle5, sweepCandle)
	if invalidated {
		t.Fatalf("setup tidak boleh divalidasi batal pada kasus ini")
	}
	if cisd != nil && cisd.Detected {
		t.Fatalf("CISD sell tidak boleh terbentuk tanpa deretan bullish, dapat level %.2f SL %.2f",
			cisd.CISDLevel, cisd.ExtremeSL)
	}
}

// TestCISDTetapTerbentukDariDeretanSah memastikan penghapusan blok cadangan
// tidak ikut membuang CISD yang benar-benar punya deretan bersambung.
func TestCISDTetapTerbentukDariDeretanSah(t *testing.T) {
	eval := NewEvaluator()
	awalCandle5 := time.Date(2026, 11, 11, 6, 0, 0, 0, time.UTC)

	sweepCandle := indicators.Candle{
		Timestamp: awalCandle5.Add(-4 * time.Hour),
		Open:      4050, High: 4060, Low: 4030, Close: 4045, IsClosed: true,
	}

	// Deretan bullish bersambung menuju puncak, lalu ditembus close.
	ltfCandles := []indicators.Candle{
		{Timestamp: awalCandle5.Add(-30 * time.Minute), Open: 4051.00, High: 4053.00, Low: 4050.50, Close: 4052.50, IsClosed: true},
		{Timestamp: awalCandle5.Add(-15 * time.Minute), Open: 4052.50, High: 4055.00, Low: 4052.00, Close: 4054.50, IsClosed: true},
		{Timestamp: awalCandle5, Open: 4054.50, High: 4057.00, Low: 4054.00, Close: 4056.50, IsClosed: true},
		{Timestamp: awalCandle5.Add(15 * time.Minute), Open: 4056.50, High: 4056.80, Low: 4048.00, Close: 4049.00, IsClosed: true},
	}

	cisd, invalidated := eval.DetectDynamicCISD(ltfCandles, "BEARISH", awalCandle5, sweepCandle)
	if invalidated {
		t.Fatalf("tidak boleh divalidasi batal")
	}
	if cisd == nil || !cisd.Detected {
		t.Fatalf("CISD sell dengan deretan bullish sah harus terbentuk")
	}
	// Origin adalah open candle pembuka deretan bullish menuju puncak. Pada data
	// ini deretannya bersambung tanpa jeda, jadi origin-nya candle paling awal.
	if cisd.CISDLevel != 4051.00 {
		t.Errorf("origin CISD = %.2f; ingin 4051.00 (awal deretan bullish)", cisd.CISDLevel)
	}
	if cisd.EntryPrice != 4049.00 {
		t.Errorf("harga masuk = %.2f; ingin 4049.00 (close candle break)", cisd.EntryPrice)
	}
}

// fvgUji menyusun FVG lengkap dengan waktu candle pembentuknya.
func fvgUji(arah FVGType, bottom, top float64, c1, c2, c3 int, t time.Time) FVG {
	return FVG{
		Type:         arah,
		Bottom:       bottom,
		Top:          top,
		Candle1Index: c1,
		Candle2Index: c2,
		Candle3Index: c3,
		Candle1Time:  t,
		Candle2Time:  t,
		Candle3Time:  t,
	}
}

// TestFVGUsangTidakDipilihAcuan mengunci kasus USTEC: FVG dari 21 Sep tidak
// boleh lagi jadi acuan pada 9 Okt hanya karena belum termitigasi.
func TestFVGUsangTidakDipilihAcuan(t *testing.T) {
	eval := NewEvaluator()
	awal := time.Date(2026, 9, 21, 17, 0, 0, 0, time.UTC)

	// Rangkaian candle: FVG bullish lahir di awal, lalu harga naik jauh dan
	// tidak pernah kembali ke zona itu.
	candles := []indicators.Candle{
		{Timestamp: awal.Add(-12 * time.Hour), Open: 29900, High: 29930, Low: 29890, Close: 29920, IsClosed: true},
		{Timestamp: awal.Add(-8 * time.Hour), Open: 29920, High: 29987, Low: 29930, Close: 29937, IsClosed: true},
		{Timestamp: awal.Add(-4 * time.Hour), Open: 29937, High: 30405, Low: 29914, Close: 30382, IsClosed: true},
		{Timestamp: awal, Open: 30382, High: 30555, Low: 30366, Close: 30474, IsClosed: true},
	}
	// Tambah 20 candle setelahnya yang semuanya jauh di atas zona FVG.
	for i := 0; i < 20; i++ {
		harga := 30900.0 + float64(i)*10
		candles = append(candles, indicators.Candle{
			Timestamp: awal.Add(time.Duration(i+1) * 4 * time.Hour),
			Open:      harga, High: harga + 40, Low: harga - 20, Close: harga + 30, IsClosed: true,
		})
	}

	sweep := eval.DetectHTFIlaSweep(candles)
	if sweep == nil {
		return // tidak ada acuan sama sekali: juga berarti FVG usang tidak dipakai
	}
	if sweep.FVG != nil && sweep.FVG.Candle3Index <= 3 {
		t.Fatalf("FVG umur 20 candle masih dipakai sebagai acuan; seharusnya ditolak")
	}
}

// TestFVGTerpilihYangTermuda mengunci pemilihan acuan: begitu FVG terbaru
// termitigasi, acuan harus jatuh ke FVG aktif yang paling muda, bukan yang
// kebetulan paling akhir di daftar.
func TestFVGTerpilihYangTermuda(t *testing.T) {
	eval := NewEvaluator()
	awal := time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)

	// Susunan asli XAUUSD H4 12 Nov: candle 02:00 dan 06:00 membentuk FVG bearish
	// besar (Low candle 02:00 = 3924.74 di atas High candle 06:00 = 3866.33),
	// lalu 10:00 dan 14:00 membentuk FVG bullish kecil 3886.93-3888.47, dan
	// 18:00 serta 22:00 menyapu zona bearish.
	candles := []indicators.Candle{
		{Timestamp: awal.Add(-16 * time.Hour), Open: 3965.88, High: 3972.51, Low: 3924.74, Close: 3954.01, IsClosed: true},
		{Timestamp: awal.Add(-12 * time.Hour), Open: 3954.01, High: 3955.51, Low: 3866.33, Close: 3870.07, IsClosed: true},
		{Timestamp: awal.Add(-8 * time.Hour), Open: 3870.07, High: 3887.64, Low: 3852.64, Close: 3882.88, IsClosed: true},
		{Timestamp: awal.Add(-4 * time.Hour), Open: 3882.88, High: 3886.93, Low: 3872.16, Close: 3877.36, IsClosed: true},
		{Timestamp: awal, Open: 3877.36, High: 3916.74, Low: 3875.20, Close: 3915.86, IsClosed: true},
		{Timestamp: awal.Add(4 * time.Hour), Open: 3915.86, High: 3923.20, Low: 3888.47, Close: 3900.61, IsClosed: true},
	}

	sweep := eval.DetectHTFIlaSweep(candles)
	if sweep == nil || !sweep.Detected {
		t.Fatalf("sweep bearish pada FVG 3887.64-3924.74 harus tetap sah")
	}
	if sweep.Direction != "BEARISH" {
		t.Fatalf("arah sweep = %s; ingin BEARISH", sweep.Direction)
	}
	if sweep.FVG == nil || sweep.FVG.Top != 3924.74 {
		t.Errorf("acuan harus FVG bearish 3924.74, dapat %+v", sweep.FVG)
	}
}

// TestFVGDijangkauHarga menguji syarat sentuhan zona secara langsung.
func TestFVGDijangkauHarga(t *testing.T) {
	awal := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	bearish := fvgUji(FVGBearish, 3887.64, 3924.74, 0, 1, 2, awal)

	kasus := []struct {
		nama    string
		candles []indicators.Candle
		harap   bool
	}{
		{
			// Zona bearish ada di 3887.64-3924.74; harga harus NAIK masuk zona.
			// Candle yang seluruh rentangnya di bawah 3887.64 belum menyentuhnya.
			nama: "harga masih jauh di bawah zona",
			candles: []indicators.Candle{
				{Timestamp: awal.Add(4 * time.Hour), Open: 3800, High: 3820, Low: 3780, Close: 3810, IsClosed: true},
				{Timestamp: awal.Add(8 * time.Hour), Open: 3810, High: 3870, Low: 3800, Close: 3860, IsClosed: true},
			},
			harap: false,
		},
		{
			nama: "harga naik menyentuh zona",
			candles: []indicators.Candle{
				{Timestamp: awal.Add(4 * time.Hour), Open: 3800, High: 3890, Low: 3780, Close: 3885, IsClosed: true},
			},
			harap: true,
		},
	}

	for _, k := range kasus {
		if dapat := fvgMasihDijangkauHarga(bearish, k.candles); dapat != k.harap {
			t.Errorf("%s: dapat %v, ingin %v", k.nama, dapat, k.harap)
		}
	}
}

// TestBatasUmurFVGMengunciNilai mengunci angka batas umur FVG beserta alasannya.
//
// Nilai ini bukan preferensi: ia hasil pengukuran pada run XAUUSD, USTEC, dan
// EURUSD memakai feed asli. Batas empat terbukti membuang entry EURUSD 12 Nov
// 2026 09:45 NY yang berakhir TP karena FVG acuannya berumur lima candle.
// Mengubah angka ini wajib disertai pengukuran ulang ketiga pasangan.
func TestBatasUmurFVGMengunciNilai(t *testing.T) {
	if umurMaksimalFVG != 6 {
		t.Fatalf("umurMaksimalFVG = %d; ingin 6 (lihat komentar di atas)", umurMaksimalFVG)
	}
}

// TestFVGLewatBatasUmurDitolak memastikan FVG yang lebih tua dari batas tidak
// lagi dipakai sebagai acuan walau harganya masih menyentuh zonanya.
func TestFVGLewatBatasUmurDitolak(t *testing.T) {
	eval := NewEvaluator()
	awal := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	// FVG bearish 3887.64-3924.74 di indeks 1..3.
	candles := []indicators.Candle{
		{Timestamp: awal, Open: 3965.88, High: 3972.51, Low: 3924.74, Close: 3954.01, IsClosed: true},
		{Timestamp: awal.Add(4 * time.Hour), Open: 3954.01, High: 3955.51, Low: 3887.00, Close: 3890.00, IsClosed: true},
		{Timestamp: awal.Add(8 * time.Hour), Open: 3890.00, High: 3887.64, Low: 3852.64, Close: 3882.88, IsClosed: true},
	}

	// Tambah candle berikutnya yang tetap berada di dalam zona, supaya FVG
	// tidak termitigasi dan yang diuji benar-benar batas umurnya.
	for i := 0; i < 14; i++ {
		titik := awal.Add(time.Duration(12+i*4) * time.Hour)
		candles = append(candles, indicators.Candle{
			Timestamp: titik, Open: 3900, High: 3910, Low: 3892, Close: 3905, IsClosed: true,
		})
	}

	// Dengan 14 candle tambahan, FVG pasti sudah berumur jauh melewati batas,
	// sehingga tidak boleh ada acuan.
	sweep := eval.DetectHTFIlaSweep(candles)
	if sweep != nil && sweep.FVG != nil {
		t.Fatalf("FVG berumur %d masih dipakai; seharusnya ditolak karena batas %d",
			len(candles)-1-sweep.FVG.Candle3Index, umurMaksimalFVG)
	}
}
