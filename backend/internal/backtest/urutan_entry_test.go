package backtest

import (
	"testing"
	"time"

	"ila-strategy/indicators"
)

// Paket tes ini menjaga URUTAN waktu sebuah entry. Bug yang pernah terjadi
// bukan soal aturan entry-nya salah, melainkan soal urutan: posisi dibuka pada
// candle yang belum melahirkan sinyalnya, atau dibuka sebelum candle konfirmasi
// selesai. Kesalahan seperti itu tidak terlihat dari win rate, hanya dari
// memeriksa waktu satu per satu. Karena itu invarian di bawah ini dikunci.
//
// Data yang dipakai sintetis dan sengaja dirancang menghasilkan beberapa entry
// sekaligus: FVG bearish H4 dibentuk tiga candle, disapu candle keempat, lalu
// M15 di dalam jendela eksekusi membentuk deretan berlawanan (bullish) yang
// ditembus close. Dari situ engine harus melahirkan entry SELL yang sah.

// racikCandleLTF membuat candle M15 berurutan mulai dari `mulai`.
func racikCandleLTF(mulai time.Time, daftar [][4]float64) []indicators.Candle {
	candles := make([]indicators.Candle, 0, len(daftar))
	for i, ohlc := range daftar {
		candles = append(candles, indicators.Candle{
			Timestamp: mulai.Add(time.Duration(i) * 15 * time.Minute),
			Open:      ohlc[0],
			High:      ohlc[1],
			Low:       ohlc[2],
			Close:     ohlc[3],
			IsClosed:  true,
		})
	}
	return candles
}

// TestInvarianUrutanEntry memeriksa seluruh entry hasil simulasi terhadap
// aturan urutan yang harus selalu benar, apa pun datanya.
func TestInvarianUrutanEntry(t *testing.T) {
	hasil, jendela, err := jalankanSkenarioILA(t)
	if err != nil {
		t.Fatalf("engine gagal: %v", err)
	}
	if len(hasil.Entries) == 0 {
		t.Fatalf("skenario tes tidak menghasilkan entry; invarian tidak bisa diuji")
	}
	if len(jendela) != len(hasil.Entries) {
		t.Fatalf("jumlah jendela chart (%d) harus sama dengan jumlah entry (%d)",
			len(jendela), len(hasil.Entries))
	}

	for i := range hasil.Entries {
		posisi := &hasil.Entries[i]
		nama := "entry #" + itoa(posisi.Seq)

		// 1. Nomor urut harus naik satu-satu dan waktu entry tidak boleh mundur.
		if posisi.Seq != i+1 {
			t.Errorf("%s: Seq tidak berurutan, dapat %d", nama, posisi.Seq)
		}
		if i > 0 && posisi.EntryTime.Before(hasil.Entries[i-1].EntryTime) {
			t.Errorf("%s: waktu entry mundur dari entry sebelumnya (%v < %v)",
				nama, posisi.EntryTime, hasil.Entries[i-1].EntryTime)
		}

		// 2. Posisi hanya boleh ada setelah sinyalnya lahir.
		if posisi.EntryTime.Before(posisi.SinyalTime) {
			t.Errorf("%s: entry dibuka sebelum sinyalnya (%v < %v)",
				nama, posisi.EntryTime, posisi.SinyalTime)
		}

		// 3. Entry tidak boleh mendahului candle sweep HTF.
		if posisi.Levels.Sweep != nil {
			waktuSweep := time.Unix(posisi.Levels.Sweep.Time, 0).UTC()
			if posisi.EntryTime.Before(waktuSweep) {
				t.Errorf("%s: entry %v mendahului sweep %v", nama, posisi.EntryTime, waktuSweep)
			}
		}

		// 4. Entry tidak boleh mendahului candle CISD, dan candle CISD itulah
		//    yang close-nya menjadi harga masuk.
		if posisi.Levels.CISD != nil {
			waktuCISD := time.Unix(posisi.Levels.CISD.Time, 0).UTC()
			if posisi.EntryTime.Before(waktuCISD) {
				t.Errorf("%s: entry %v mendahului CISD %v", nama, posisi.EntryTime, waktuCISD)
			}
			if !hampirSama(posisi.EntryPrice, posisi.Levels.CISD.Price) {
				t.Errorf("%s: harga entry %.2f harus sama dengan close candle CISD %.2f",
					nama, posisi.EntryPrice, posisi.Levels.CISD.Price)
			}
		}

		// 4b. Entry wajib terjadi selama C3 berjalan (open sampai close C3).
		//     Aturan New ILA 2.0 tidak memperpanjang jendela ke C4.
		if posisi.Levels.Sweep != nil {
			waktuSweep := time.Unix(posisi.Levels.Sweep.Time, 0).UTC()
			// Satu candle H4 = 4 jam: C2 berakhir +4j, C3 berakhir +8j.
			batasJendela := waktuSweep.Add(8 * time.Hour)
			if posisi.EntryTime.After(batasJendela) {
				t.Errorf("%s: entry %v di luar jendela C3 dari C2 %v (batas %v)",
					nama, posisi.EntryTime, waktuSweep, batasJendela)
			}
		}

		// 5. SL dan TP harus di sisi yang benar. SELL: SL di atas, TP di bawah.
		//    BUY: SL di bawah, TP di atas. Tertukar berarti trade rugi berubah
		//    tampak untung tanpa satu pun aturan melanggar.
		switch posisi.Direction {
		case DirectionSell:
			if posisi.StopLoss <= posisi.EntryPrice {
				t.Errorf("%s SELL: SL (%.2f) harus di atas entry (%.2f)",
					nama, posisi.StopLoss, posisi.EntryPrice)
			}
			if posisi.TakeProfit >= posisi.EntryPrice {
				t.Errorf("%s SELL: TP (%.2f) harus di bawah entry (%.2f)",
					nama, posisi.TakeProfit, posisi.EntryPrice)
			}
		case DirectionBuy:
			if posisi.StopLoss >= posisi.EntryPrice {
				t.Errorf("%s BUY: SL (%.2f) harus di bawah entry (%.2f)",
					nama, posisi.StopLoss, posisi.EntryPrice)
			}
			if posisi.TakeProfit <= posisi.EntryPrice {
				t.Errorf("%s BUY: TP (%.2f) harus di atas entry (%.2f)",
					nama, posisi.TakeProfit, posisi.EntryPrice)
			}
		default:
			t.Errorf("%s: arah tidak dikenal %q", nama, posisi.Direction)
		}

		// 6. Exit tidak boleh terjadi sebelum entry dibuka.
		if !posisi.ExitTime.IsZero() && posisi.ExitTime.Before(posisi.EntryTime) {
			t.Errorf("%s: exit %v sebelum entry %v", nama, posisi.ExitTime, posisi.EntryTime)
		}

		// 7. Entry berikutnya tidak boleh dibuka pada candle yang sama atau lebih
		//    awal daripada candle exit entry ini. Candle exit sudah dipakai
		//    menilai posisi lama, jadi tidak menyisakan harga yang jujur.
		if i+1 < len(hasil.Entries) {
			berikut := &hasil.Entries[i+1]
			if !berikut.EntryTime.After(posisi.ExitTime) {
				t.Errorf("%s: entry berikutnya dibuka %v, tidak setelah exit %v",
					nama, berikut.EntryTime, posisi.ExitTime)
			}
		}
	}
}

// jalankanSkenarioILA menyusun data H4 dan M15 sintetis yang menghasilkan
// minimal satu entry, lalu menjalankan engine penuh.
func jalankanSkenarioILA(t *testing.T) (*Result, []CandleWindow, error) {
	t.Helper()
	htf, ltf, params := dataSkenario(t)
	engine := NewEngine()
	return engine.Run(params, htf, ltf)
}

// dataSkenario mengembalikan candle H4, M15, dan parameter yang dipakai tes.
func dataSkenario(t *testing.T) ([]indicators.Candle, []indicators.Candle, Params) {
	t.Helper()

	// Awal deret. Enam candle pertama adalah pemanasan agar evaluator punya
	// konteks struktur; candle berikutnya yang membentuk formasi FVG dan sweep.
	awal := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// H4: tiga candle (indeks 6-8) membentuk FVG bearish karena High candle 3
	// berada di bawah Low candle 1. Candle indeks 9 menyapu ke atas masuk ke
	// zona itu lalu ditolak close di bawahnya: itulah sweep yang sah.
	htf := []indicators.Candle{
		{Timestamp: awal.Add(-24 * time.Hour), Open: 4150, High: 4160, Low: 4100, Close: 4110, IsClosed: true},
		{Timestamp: awal.Add(-20 * time.Hour), Open: 4110, High: 4130, Low: 4090, Close: 4100, IsClosed: true},
		{Timestamp: awal.Add(-16 * time.Hour), Open: 4100, High: 4170, Low: 4090, Close: 4150, IsClosed: true},
		{Timestamp: awal.Add(-12 * time.Hour), Open: 4150, High: 4190, Low: 4140, Close: 4180, IsClosed: true},
		{Timestamp: awal.Add(-8 * time.Hour), Open: 4180, High: 4230, Low: 4170, Close: 4210, IsClosed: true},
		{Timestamp: awal.Add(-4 * time.Hour), Open: 4210, High: 4250, Low: 4200, Close: 4230, IsClosed: true},
		{Timestamp: awal, Open: 4230, High: 4260, Low: 4240, Close: 4250, IsClosed: true},                     // C1 FVG
		{Timestamp: awal.Add(4 * time.Hour), Open: 4250, High: 4245, Low: 4150, Close: 4155, IsClosed: true},  // C2 FVG
		{Timestamp: awal.Add(8 * time.Hour), Open: 4155, High: 4212, Low: 4150, Close: 4160, IsClosed: true},  // C3 FVG
		{Timestamp: awal.Add(12 * time.Hour), Open: 4160, High: 4230, Low: 4155, Close: 4180, IsClosed: true}, // C2 closure
		{Timestamp: awal.Add(16 * time.Hour), Open: 4180, High: 4210, Low: 4160, Close: 4190, IsClosed: true}, // C3
		{Timestamp: awal.Add(20 * time.Hour), Open: 4190, High: 4205, Low: 4150, Close: 4165, IsClosed: true}, // C4
		{Timestamp: awal.Add(24 * time.Hour), Open: 4165, High: 4180, Low: 4120, Close: 4130, IsClosed: true},
		{Timestamp: awal.Add(28 * time.Hour), Open: 4130, High: 4140, Low: 4080, Close: 4090, IsClosed: true},
		{Timestamp: awal.Add(32 * time.Hour), Open: 4090, High: 4100, Low: 4040, Close: 4050, IsClosed: true},
	}

	// M15 dimulai 1 jam sebelum C3 (awal+16h) agar terbentuk Swing High ERL LTF
	// (4215 di idx 1) sebelum rangkaian bullish di dalam C3 menyapunya dengan wick
	// (Kondisi 1 New ILA 2.0) lalu ditembus close body (-CISD).
	mulaiLTF := awal.Add(15 * time.Hour)
	daftar := make([][4]float64, 0, 96)
	// Pembentukan Swing High ERL LTF = 4215 (candle kiri, tengah, kanan di idx 0..2).
	daftar = append(daftar,
		[4]float64{4195, 4200, 4190, 4192},
		[4]float64{4192, 4215, 4191, 4210},
		[4]float64{4210, 4212, 4190, 4192},
		[4]float64{4192, 4195, 4180, 4182},
	)
	// Fase naik di dalam C3: rangkaian bullish mulai 4182, wick 4218 menyentuh
	// Swing High 4215 tanpa close di atasnya (Kondisi 1).
	daftar = append(daftar,
		[4]float64{4182, 4196, 4180, 4195},
		[4]float64{4195, 4210, 4193, 4208},
		[4]float64{4208, 4218, 4205, 4213},
	)
	// Fase turun: close menembus body deretan bullish (level 4182), memicu
	// konfirmasi CISD sell.
	for i := 0; i < 24; i++ {
		basis := 4214.0 - float64(i)*14
		daftar = append(daftar, [4]float64{basis, basis + 4, basis - 12, basis - 10})
	}
	// Fase lanjut turun ke arah DOL agar TP dan SL sama-sama teruji.
	for i := 0; i < 48; i++ {
		basis := 3916.0 - float64(i)*2
		daftar = append(daftar, [4]float64{basis, basis + 3, basis - 3, basis - 2})
	}
	ltf := racikCandleLTF(mulaiLTF, daftar)

	params := Params{
		Symbol:          "XAUUSD",
		HTF:             "H4",
		LTF:             "M15",
		From:            mulaiLTF,
		To:              ltf[len(ltf)-1].Timestamp,
		Equity:          10_000,
		RiskPercent:     0.5,
		MaxTradesPerDay: 3,
		MaxLossesPerDay: 2,
		StopAfterWin:    true,
	}

	return htf, ltf, params
}

// hampirSama membandingkan harga dengan toleransi kecil untuk pembulatan float.
func hampirSama(a, b float64) bool {
	selisih := a - b
	if selisih < 0 {
		selisih = -selisih
	}
	return selisih < 0.005
}

// itoa menghindari impor strconv hanya untuk satu tempat.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digit := make([]byte, 0, 4)
	for n > 0 {
		digit = append([]byte{byte('0' + n%10)}, digit...)
		n /= 10
	}
	return string(digit)
}
