// Package backtest menjalankan aturan ILA 2.0 pada data historis.
//
// Paket ini sengaja tidak tahu soal HTTP, database, maupun antrean. Ia hanya
// menerima deret candle dan parameter, lalu mengembalikan daftar entry beserta
// hasilnya. Dengan begitu aturan simulasi bisa diuji tanpa menyalakan server,
// dan mesin yang sama bisa dipakai CLI maupun worker.
//
// Satu prinsip yang menjiwai seluruh paket ini: setiap evaluasi hanya boleh
// melihat candle sampai titik waktunya sendiri. Melanggar itu menghasilkan
// backtest yang angkanya bagus tetapi bohong, dan kesalahan seperti itu tidak
// terlihat dari hasilnya.
package backtest

import (
	"time"

	"ila-strategy/indicators"
)

// Outcome adalah cara sebuah posisi berakhir.
type Outcome string

const (
	// OutcomeTakeProfit berarti harga mencapai target lebih dulu.
	OutcomeTakeProfit Outcome = "TP"
	// OutcomeStopLoss berarti harga mencapai batas rugi lebih dulu.
	OutcomeStopLoss Outcome = "SL"
	// OutcomeExpired berarti data berakhir sebelum TP maupun SL tersentuh.
	// Posisi seperti ini masih terbuka saat rentang habis.
	OutcomeExpired Outcome = "EXPIRED"
)

// Arah posisi.
const (
	DirectionBuy  = "BUY"
	DirectionSell = "SELL"
)

// Params adalah masukan satu kali backtest.
//
// Tidak ada parameter RR di sini dengan sengaja. Target ditentukan oleh aturan
// ILA 2.0 sendiri (Draw on Liquidity dengan lantai 1:2), dan menawarkan RR
// sebagai angka bebas hanya akan menghasilkan nol trade karena semua setup
// gagal saringan PassedMinRR.
type Params struct {
	Symbol string
	HTF    string
	LTF    string

	// From dan To membatasi rentang yang dievaluasi. Keduanya inklusif.
	From time.Time
	To   time.Time

	// Equity adalah modal awal, dipakai untuk menghitung kurva ekuitas.
	Equity float64
	// RiskPercent adalah porsi modal yang dipertaruhkan per trade, dalam persen.
	// Nilai 0,5 berarti setengah persen.
	RiskPercent float64

	// MaxTradesPerDay dan MaxLossesPerDay mengikuti aturan live apa adanya.
	// Backtest yang memakai aturan berbeda dari live tidak bisa dipercaya.
	MaxTradesPerDay int
	MaxLossesPerDay int
	// StopAfterWin selalu aktif. Tidak ada jalan mematikannya, sama seperti
	// di runner (lihat runner/rules.go).
	StopAfterWin bool
}

// Entry adalah satu posisi hasil simulasi.
type Entry struct {
	// Seq adalah nomor urut mulai dari 1, dipakai UI untuk menunjuk satu entry.
	Seq int

	Direction string

	// SinyalTime adalah waktu candle LTF yang memicu evaluasi.
	SinyalTime time.Time
	EntryTime  time.Time
	ExitTime   time.Time

	EntryPrice float64
	StopLoss   float64
	TakeProfit float64
	PlannedRR  float64

	Outcome  Outcome
	ActualR  float64
	Duration time.Duration
	// EquityAfter adalah ekuitas setelah trade ini selesai.
	EquityAfter float64

	// Ambiguous berarti satu candle menyentuh TP dan SL sekaligus, sehingga
	// urutannya tidak bisa dipastikan tanpa data tick. Kasus ini dihitung
	// sebagai SL; jumlahnya dilaporkan agar pembaca tahu seberapa besar
	// pengaruhnya terhadap hasil.
	Ambiguous bool

	// Narasi tahapan diambil apa adanya dari engine, bukan ditulis ulang.
	Summary    string
	NextAction string

	// Levels menyimpan koordinat gambar chart: FVG, sweep, CISD.
	Levels LevelSpec
}

// LevelSpec adalah koordinat yang dibutuhkan chart untuk satu entry.
//
// Semua waktu memakai Unix detik UTC karena itu format lightweight-charts.
type LevelSpec struct {
	// FVG adalah zona imbalance yang menjadi acuan entry.
	FVG *FVGSpec
	// Sweep menandai candle yang menyapu likuiditas HTF.
	Sweep *SweepSpec
	// CISD menandai candle LTF yang mengonfirmasi perubahan struktur.
	CISD *CISDSpec
}

// FVGSpec adalah satu zona Fair Value Gap.
type FVGSpec struct {
	Top      float64
	Bottom   float64
	TimeFrom int64
	TimeTo   int64
	Label    string
}

// SweepSpec menandai candle sweep pada timeframe HTF.
type SweepSpec struct {
	Time  int64
	Price float64
	Label string
}

// CISDSpec menandai candle konfirmasi pada timeframe LTF.
type CISDSpec struct {
	Time  int64
	Price float64
	Label string
}

// Result adalah keluaran lengkap satu kali backtest.
type Result struct {
	Entries []Entry

	// Statistik ringkas.
	TotalTrades     int
	Wins            int
	Losses          int
	Expired         int
	AmbiguousExits  int
	SignalsRejected int

	WinRate      float64
	TotalR       float64
	MaxDrawdown  float64
	ProfitFactor float64

	// EquityCurve adalah titik-titik ekuitas untuk digambar di grafik.
	EquityCurve []EquityPoint
}

// EquityPoint adalah satu titik pada kurva ekuitas.
type EquityPoint struct {
	Time   int64
	Equity float64
	// Drawdown adalah penurunan dari puncak tertinggi sejauh ini, sebagai
	// pecahan (0,05 berarti turun 5% dari puncak).
	Drawdown float64
}

// CandleWindow adalah potongan candle di sekitar satu entry, disimpan agar
// chart detail bisa digambar tanpa mengambil ulang data dari sumbernya.
type CandleWindow struct {
	HTF []indicators.Candle
	LTF []indicators.Candle
}
