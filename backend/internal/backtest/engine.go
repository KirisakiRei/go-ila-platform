package backtest

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"ila-strategy/ila"
	"ila-strategy/indicators"
)

// MinimalCandleHTF dan MinimalCandleLTF adalah syarat data yang sama dengan
// yang dipakai evaluator ILA 2.0. Angkanya sepuluh, bukan lima: dengan data
// lebih pendek, evaluator menolak dan backtest akan tampak "tidak ada sinyal"
// padahal sebenarnya datanya kurang.
const (
	MinimalCandleHTF = 10
	MinimalCandleLTF = 10
)

// candlePemanasan adalah jumlah candle yang disisakan SEBELUM rentang yang
// diminta. Evaluasi butuh riwayat untuk mengenali struktur; tanpa pemanasan,
// evaluasi di hari-hari awal rentang akan selalu kosong karena tidak punya
// konteks apa pun.
const candlePemanasan = 100

// candleJendelaChart adalah jumlah candle sebelum entry dan sesudah exit yang
// ikut disimpan untuk kebutuhan gambar chart.
const candleJendelaChart = 20

// candleJendelaChartHTF adalah jumlah candle sesudah exit khusus untuk timeframe
// besar.
//
// Satu candle D1 berjarak sehari; dua puluh candle D1 berarti dua puluh hari
// sesudah exit. Itu terlalu panjang, dan lebih buruk lagi bila seluruh rentang
// entry..exit sudah lebih pendek dari sepuluh candle HTF — chart HTF lalu
// didominasi ruang kosong sesudah posisi ditutup. Sepuluh candle cukup untuk
// melihat reaksi harga tepat setelah exit.
const candleJendelaChartHTF = 10

// Engine menjalankan simulasi pada deret candle yang sudah disiapkan.
//
// Engine tidak mengambil data sendiri: pemanggil yang menyediakan candle.
// Pemisahan ini membuat simulasi bisa diuji dengan candle buatan.
type Engine struct {
	evaluator *ila.Evaluator
}

// NewEngine menyiapkan mesin backtest.
func NewEngine() *Engine {
	return &Engine{evaluator: ila.NewEvaluator()}
}

// Run menjalankan simulasi penuh dan mengembalikan hasil beserta jendela candle
// per entry untuk digambar.
//
// htfCandles dan ltfCandles harus sudah urut naik menurut waktu dan sudah
// mencakup rentang yang diminta beserta pemanasannya.
func (e *Engine) Run(params Params, htfCandles, ltfCandles []indicators.Candle) (*Result, []CandleWindow, error) {
	paramsBersih, err := validasiParams(params)
	if err != nil {
		return nil, nil, err
	}
	// Pakai parameter yang sudah dibersihkan, bukan salinan mentah dari
	// pemanggil: tanpa ini pagar harian akan menerima angka nol.
	params = paramsBersih
	if len(htfCandles) == 0 || len(ltfCandles) == 0 {
		return nil, nil, fmt.Errorf("candle kosong: htf=%d ltf=%d", len(htfCandles), len(ltfCandles))
	}

	penjaga := aturanHarian{
		maksTrade:    params.MaxTradesPerDay,
		maksLoss:     params.MaxLossesPerDay,
		stopAfterWin: params.StopAfterWin,
	}

	hasil := &Result{}
	gambar := make([]CandleWindow, 0, 32)

	// Posisi yang sedang berjalan. Backtest ini mengikuti aturan live: satu
	// posisi pada satu waktu untuk satu simbol.
	var posisiAktif *Entry
	// perPosisiSelesai menyimpan indeks LTF tempat posisi terakhir selesai.
	// Entry baru tidak boleh dibuka pada candle yang sama dengan exit, karena
	// candle itu sudah dipakai menilai exit dan tidak menyisakan harga yang
	// jujur untuk masuk.
	perPosisiSelesai := -1
	// sinyalTerpakai menyimpan sidik jari setup yang sudah pernah dibuka.
	//
	// Evaluator mengembalikan status yang sama berulang kali selama strukturnya
	// belum berubah. Tanpa penjagaan ini, satu setup yang sama akan dibuka
	// berkali-kali dengan harga identik — dan itu bukan tiga trade, melainkan
	// satu trade yang dihitung tiga kali. Hasil backtest seperti itu tidak
	// sekadar salah, ia mengarang keuntungan.
	sinyalTerpakai := make(map[string]bool)

	modal := params.Equity
	if modal <= 0 {
		modal = 10_000
	}

	durHTF := durasiTimeframeBacktest(params.HTF)
	durLTF := durasiTimeframeBacktest(params.LTF)
	htfBuf := make([]indicators.Candle, 0, len(htfCandles))

	for iLTF := range ltfCandles {
		candleLTF := ltfCandles[iLTF]

		// Lewati candle yang belum tutup: high/low-nya belum final sehingga
		// bisa berubah, dan memakainya berarti menilai sesuatu yang belum
		// terjadi.
		if !candleLTF.IsClosed {
			continue
		}

		// 1. Nilai posisi yang sedang berjalan lebih dulu, memakai candle ini.
		if posisiAktif != nil && iLTF > perPosisiSelesai+1 {
			if selesai, exitPrice, outcome, ambiguous := nilaiExit(posisiAktif, candleLTF); selesai {
				tutupPosisi(posisiAktif, candleLTF.Timestamp, exitPrice, outcome, ambiguous, params, &modal)
				hasil.Entries = append(hasil.Entries, *posisiAktif)
				// Jendela candle dibuat di sini, saat exit sudah diketahui, supaya
				// chart memuat seluruh perjalanan posisi. Membuatnya saat entry
				// dibuka akan memotong trade panjang sebelum target atau batas
				// rugi tercapai, sehingga penanda exit jatuh di luar chart.
				gambar = append(gambar, jendelaChart(posisiAktif, htfCandles, ltfCandles))
				penjaga.catat(posisiAktif.Outcome, candleLTF.Timestamp)
				perPosisiSelesai = iLTF
				posisiAktif = nil
			}
		}

		// 2. Hanya cari entry baru di dalam rentang yang diminta.
		if candleLTF.Timestamp.Before(params.From) || candleLTF.Timestamp.After(params.To) {
			continue
		}
		if posisiAktif != nil {
			continue
		}
		// Jangan buka posisi baru pada candle tempat posisi sebelumnya selesai.
		if iLTF <= perPosisiSelesai {
			continue
		}

		tanggal := candleLTF.Timestamp
		if !penjaga.bolehBuka(tanggal) {
			continue
		}

		// 3. Potong data sampai titik waktu ini saja.
		//
		// Inilah pengaman anti look-ahead. Candle HTF yang belum selesai pada
		// penutupan candle LTF ini dibentuk hanya dari pergerakan LTF yang sudah
		// lewat dan ditandai IsClosed=false persis seperti di engine live.
		ltfSampai := ltfCandles[:iLTF+1]
		htfSampai := susunHTFSampai(htfCandles, ltfSampai, candleLTF.Timestamp, durHTF, durLTF, &htfBuf)

		if len(htfSampai) < MinimalCandleHTF || len(ltfSampai) < MinimalCandleLTF {
			continue
		}

		setup := e.evaluator.EvaluateILA(htfSampai, ltfSampai, params.HTF, params.LTF, params.Symbol)
		if setup == nil {
			continue
		}
		if setup.Status != "READY_TO_ENTRY" {
			continue
		}
		if !setup.PassedMinRR {
			// Ditolak saringan RR minimum. Dihitung agar pembaca tahu berapa
			// banyak setup yang tidak lolos, bukan disembunyikan.
			hasil.SignalsRejected++
			continue
		}
		arah := arahDariBias(setup.Bias)
		if arah == "" {
			continue
		}
		if setup.EntryPrice <= 0 || setup.StopLoss <= 0 || setup.TakeProfit <= 0 {
			continue
		}

		// Tolak setup yang identik dengan yang sudah pernah dibuka. Struktur
		// yang sama masih terbaca sampai ada candle yang membatalkannya, jadi
		// tanpa ini satu setup bisa menghasilkan banyak entry palsu.
		sidikJari := sidikJariSetup(setup.Bias, setup.EntryPrice, setup.StopLoss, setup.TakeProfit)
		if sinyalTerpakai[sidikJari] {
			continue
		}
		sinyalTerpakai[sidikJari] = true

		entry := Entry{
			Seq:        len(hasil.Entries) + 1,
			Direction:  arah,
			SinyalTime: candleLTF.Timestamp,
			EntryTime:  candleLTF.Timestamp,
			EntryPrice: setup.EntryPrice,
			StopLoss:   setup.StopLoss,
			TakeProfit: setup.TakeProfit,
			PlannedRR:  setup.RiskReward,
			Summary:    setup.Summary,
			NextAction: setup.NextAction,
			Levels:     levelDariSetup(setup),
		}
		posisiAktif = &entry
	}

	// Posisi yang belum selesai saat data habis tetap dicatat sebagai EXPIRED
	// agar tidak ada entry yang hilang dari hitungan.
	if posisiAktif != nil {
		hargaAkhir := ltfCandles[len(ltfCandles)-1].Close
		tutupPosisi(posisiAktif, ltfCandles[len(ltfCandles)-1].Timestamp, hargaAkhir,
			OutcomeExpired, false, params, &modal)
		hasil.Entries = append(hasil.Entries, *posisiAktif)
		gambar = append(gambar, jendelaChart(posisiAktif, htfCandles, ltfCandles))
	}

	hitungStatistik(hasil, params.Equity)
	return hasil, gambar, nil
}

// validasiParams memastikan masukan masuk akal sebelum simulasi dimulai.
//
// Fungsi ini mengembalikan parameter yang sudah dibersihkan, bukan mengubah
// salinannya. Go menyalin struct saat dikirim sebagai nilai, sehingga perubahan
// di sini tidak akan terlihat pemanggil bila memakai bentuk tanpa kembalian.
func validasiParams(p Params) (Params, error) {
	if p.Symbol == "" {
		return p, fmt.Errorf("simbol wajib diisi")
	}
	if p.HTF == "" || p.LTF == "" {
		return p, fmt.Errorf("pasangan timeframe wajib diisi")
	}
	if p.From.IsZero() || p.To.IsZero() {
		return p, fmt.Errorf("rentang waktu wajib diisi")
	}
	if !p.From.Before(p.To) {
		return p, fmt.Errorf("waktu mulai harus lebih awal dari waktu akhir")
	}
	if p.Equity <= 0 {
		return p, fmt.Errorf("modal harus lebih besar dari nol")
	}
	if p.RiskPercent <= 0 || p.RiskPercent > 100 {
		return p, fmt.Errorf("risiko per trade harus di antara 0 dan 100 persen")
	}

	// Pagar harian mengikuti aturan live: 1 sampai 3 trade, 1 sampai 3 loss,
	// dan berhenti setelah menang selalu aktif.
	if p.MaxTradesPerDay < 1 {
		p.MaxTradesPerDay = 1
	}
	if p.MaxTradesPerDay > 3 {
		p.MaxTradesPerDay = 3
	}
	if p.MaxLossesPerDay < 1 {
		p.MaxLossesPerDay = 1
	}
	if p.MaxLossesPerDay > 3 {
		p.MaxLossesPerDay = 3
	}
	p.StopAfterWin = true

	return p, nil
}

// sidikJariSetup membuat kunci pembanding sebuah setup.
//
// Dipakai untuk memastikan satu struktur pasar hanya menghasilkan satu entry.
// Harga entry, SL, dan TP dipakai sebagai penanda karena ketiganya berubah
// begitu struktur yang mendasarinya berubah.
func sidikJariSetup(bias string, entry, stopLoss, takeProfit float64) string {
	return fmt.Sprintf("%s|%.10f|%.10f|%.10f", bias, entry, stopLoss, takeProfit)
}

// arahDariBias menyelaraskan bias struktur dengan arah order. Bias netral
// tidak menghasilkan arah, sehingga tidak ada entry.
func arahDariBias(bias string) string {
	switch bias {
	case "BULLISH":
		return DirectionBuy
	case "BEARISH":
		return DirectionSell
	default:
		return ""
	}
}

// potongSampai mengambil candle yang waktunya tidak melewati batas.
//
// Candle HTF yang sedang berjalan (belum tutup) pada saat candle LTF tertentu
// belum boleh dipakai, karena high/low-nya masih bisa berubah. Karena itu
// batasnya bukan "lebih awal dari sekarang", melainkan "sudah tutup sebelum
// sekarang".
func potongSampai(candles []indicators.Candle, batas time.Time) []indicators.Candle {
	idx := sort.Search(len(candles), func(i int) bool {
		return !candles[i].Timestamp.Before(batas)
	})
	return candles[:idx]
}

// susunHTFSampai menyusun deret candle HTF yang terlihat pada penutupan satu
// candle LTF tanpa membocorkan high/low/close masa depan dari bar HTF berjalan.
func susunHTFSampai(
	htfCandles []indicators.Candle,
	ltfSampai []indicators.Candle,
	waktuBukaLTF time.Time,
	durHTF, durLTF time.Duration,
	buf *[]indicators.Candle,
) []indicators.Candle {
	if durHTF <= 0 || durLTF <= 0 || durHTF <= durLTF {
		return potongSampai(htfCandles, waktuBukaLTF)
	}

	waktuEvaluasi := waktuBukaLTF.Add(durLTF)
	idx := sort.Search(len(htfCandles), func(i int) bool {
		return !htfCandles[i].Timestamp.Before(waktuEvaluasi)
	})
	if idx == 0 {
		return nil
	}

	terakhir := htfCandles[idx-1]
	// Bila candle HTF terakhir sudah resmi tutup pada waktu evaluasi ini,
	// pakai irisan asli tanpa alokasi baru.
	if !terakhir.Timestamp.Add(durHTF).After(waktuEvaluasi) {
		return htfCandles[:idx]
	}

	// Candle HTF terakhir masih berjalan: bentuk OHLC-nya murni dari candle LTF
	// yang sudah lewat di dalam rentang tersebut dengan IsClosed=false.
	berjalan := bentukCandleHTFBerjalan(terakhir, ltfSampai)
	out := (*buf)[:0]
	out = append(out, htfCandles[:idx-1]...)
	out = append(out, berjalan)
	*buf = out
	return out
}

func bentukCandleHTFBerjalan(acuan indicators.Candle, ltfSampai []indicators.Candle) indicators.Candle {
	mulai := sort.Search(len(ltfSampai), func(i int) bool {
		return !ltfSampai[i].Timestamp.Before(acuan.Timestamp)
	})
	if mulai >= len(ltfSampai) {
		return indicators.Candle{
			Timestamp: acuan.Timestamp,
			Open:      acuan.Open,
			High:      acuan.Open,
			Low:       acuan.Open,
			Close:     acuan.Open,
			IsClosed:  false,
		}
	}

	pertama := ltfSampai[mulai]
	c := indicators.Candle{
		Timestamp: acuan.Timestamp,
		Open:      pertama.Open,
		High:      pertama.High,
		Low:       pertama.Low,
		Close:     pertama.Close,
		Volume:    pertama.Volume,
		IsClosed:  false,
	}
	for i := mulai + 1; i < len(ltfSampai); i++ {
		b := ltfSampai[i]
		if b.High > c.High {
			c.High = b.High
		}
		if b.Low < c.Low {
			c.Low = b.Low
		}
		c.Close = b.Close
		c.Volume += b.Volume
	}
	return c
}

func durasiTimeframeBacktest(tf string) time.Duration {
	tf = strings.ToUpper(strings.TrimSpace(tf))
	if tf == "D" || tf == "W" {
		tf += "1"
	}
	if len(tf) < 2 {
		return 0
	}
	unit, digits := tf[0], tf[1:]
	if unit >= '0' && unit <= '9' {
		unit, digits = tf[len(tf)-1], tf[:len(tf)-1]
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || n <= 0 || n > 10080 {
		return 0
	}
	switch unit {
	case 'M':
		return time.Duration(n) * time.Minute
	case 'H':
		return time.Duration(n) * time.Hour
	case 'D':
		return time.Duration(n) * 24 * time.Hour
	case 'W':
		return time.Duration(n) * 7 * 24 * time.Hour
	default:
		return 0
	}
}

// potongJendela mengambil sejumlah candle di kiri dan kanan sebuah titik waktu.

// jendelaChart memotong candle di sekitar satu posisi untuk kebutuhan gambar.
//
// Jendelanya membentang dari beberapa candle sebelum entry sampai beberapa
// candle sesudah exit, bukan hanya di sekitar entry. Tanpa rentang penuh,
// trade yang berjalan berhari-hari kehilangan candle saat target atau batas
// rugi tersentuh; chart lalu tampak seperti posisi yang menggantung tanpa
// akhir, padahal mesin sudah menutupnya.
//
// HTF dan LTF dipotong pada TITIK WAKTU yang sama (entry sampai exit), sehingga
// level yang digambar di kedua panel jatuh pada momen yang sama. Sebelum
// diperbaiki, candle HTF dipotong hingga candle LTF terakhir yang dievaluasi,
// bukan hingga exit; pada trade yang ditutup kemudian, target atau batas rugi
// tidak pernah tersentuh di panel HTF dan garis exit tampak keluar dari chart.
func jendelaChart(posisi *Entry, htfCandles, ltfCandles []indicators.Candle) CandleWindow {
	akhir := posisi.ExitTime
	if akhir.IsZero() || akhir.Before(posisi.EntryTime) {
		akhir = posisi.EntryTime
	}
	return CandleWindow{
		HTF: potongRentang(htfCandles, posisi.EntryTime, akhir, candleJendelaChart, candleJendelaChartHTF),
		LTF: potongRentang(ltfCandles, posisi.EntryTime, akhir, candleJendelaChart, candleJendelaChart),
	}
}

// potongRentang mengambil candle dari beberapa butir sebelum mulai sampai
// beberapa butir sesudah selesai.
func potongRentang(candles []indicators.Candle, mulai, selesai time.Time, sebelum, sesudah int) []indicators.Candle {
	titikMulai := sort.Search(len(candles), func(i int) bool {
		return !candles[i].Timestamp.Before(mulai)
	})
	// Search mengembalikan posisi pertama yang >= mulai; satu butir sebelumnya
	// ikut diambil karena candle sweep sering tepat di detik yang dicari.
	awal := titikMulai - sebelum
	if awal < 0 {
		awal = 0
	}

	titikSelesai := sort.Search(len(candles), func(i int) bool {
		return candles[i].Timestamp.After(selesai)
	})
	akhir := titikSelesai + sesudah
	if akhir > len(candles) {
		akhir = len(candles)
	}

	if awal >= akhir {
		return nil
	}
	hasil := make([]indicators.Candle, akhir-awal)
	copy(hasil, candles[awal:akhir])
	return hasil
}

func potongJendela(candles []indicators.Candle, titik time.Time, jendela int) []indicators.Candle {
	posisi := sort.Search(len(candles), func(i int) bool {
		return !candles[i].Timestamp.Before(titik)
	})
	mulai := posisi - jendela
	if mulai < 0 {
		mulai = 0
	}
	akhir := posisi + jendela + 1
	if akhir > len(candles) {
		akhir = len(candles)
	}
	if mulai >= akhir {
		return nil
	}
	hasil := make([]indicators.Candle, akhir-mulai)
	copy(hasil, candles[mulai:akhir])
	return hasil
}

// nilaiExit memeriksa apakah satu candle menutup posisi.
//
// Aturan saat satu candle menyentuh TP dan SL sekaligus: SL didahulukan. Kita
// tidak bisa tahu mana yang benar-benar terjadi lebih dulu tanpa data tick;
// memilih TP berarti berharap, memilih SL berarti jujur. Backtest dipakai untuk
// memutuskan apakah metode layak dipakai uang nyata, sehingga anggapan yang
// terlalu optimis justru yang paling berbahaya. Jumlah kasus ini dilaporkan
// supaya pembaca tahu seberapa besar pengaruhnya.
func nilaiExit(posisi *Entry, candle indicators.Candle) (selesai bool, harga float64, outcome Outcome, ambiguous bool) {
	kenaTP := false
	kenaSL := false

	switch posisi.Direction {
	case DirectionBuy:
		kenaTP = candle.High >= posisi.TakeProfit
		kenaSL = candle.Low <= posisi.StopLoss
	case DirectionSell:
		kenaTP = candle.Low <= posisi.TakeProfit
		kenaSL = candle.High >= posisi.StopLoss
	}

	if kenaTP && kenaSL {
		return true, posisi.StopLoss, OutcomeStopLoss, true
	}
	if kenaSL {
		return true, posisi.StopLoss, OutcomeStopLoss, false
	}
	if kenaTP {
		return true, posisi.TakeProfit, OutcomeTakeProfit, false
	}
	return false, 0, "", false
}

// tutupPosisi mengisi hasil akhir sebuah entry dan memperbarui modal.
func tutupPosisi(posisi *Entry, waktuTutup time.Time, harga float64, outcome Outcome, ambiguous bool, params Params, modal *float64) {
	posisi.ExitTime = waktuTutup
	posisi.Outcome = outcome
	posisi.Ambiguous = ambiguous
	posisi.Duration = waktuTutup.Sub(posisi.EntryTime)

	switch outcome {
	case OutcomeTakeProfit:
		posisi.ActualR = posisi.PlannedRR
	case OutcomeStopLoss:
		posisi.ActualR = -1
	case OutcomeExpired:
		// Belum tersentuh: R dihitung dari harga terakhir supaya tetap jujur
		// daripada dibulatkan menjadi nol.
		risiko := math.Abs(posisi.EntryPrice - posisi.StopLoss)
		if risiko > 0 {
			selisih := harga - posisi.EntryPrice
			if posisi.Direction == DirectionSell {
				selisih = -selisih
			}
			posisi.ActualR = selisih / risiko
		}
	}

	// Perubahan modal memakai risiko nominal yang dikunci saat entry dibuka,
	// bukan persentase dari modal yang sedang berjalan.
	risikoNominal := *modal * params.RiskPercent / 100
	*modal += risikoNominal * posisi.ActualR
	posisi.EquityAfter = *modal
}

// levelDariSetup menerjemahkan hasil evaluasi menjadi koordinat gambar.
func levelDariSetup(setup *ila.ILASetup) LevelSpec {
	spek := LevelSpec{}

	if setup.HTFSweep != nil {
		if setup.HTFSweep.FVG != nil {
			fvg := setup.HTFSweep.FVG
			spek.FVG = &FVGSpec{
				Top:      fvg.Top,
				Bottom:   fvg.Bottom,
				TimeFrom: fvg.Candle1Time.Unix(),
				TimeTo:   fvg.Candle3Time.Unix(),
				Label:    string(fvg.Type),
			}
		}
		if !setup.HTFSweep.SweepTime.IsZero() {
			spek.Sweep = &SweepSpec{
				Time:  setup.HTFSweep.SweepTime.Unix(),
				Price: setup.HTFSweep.StopLoss,
				Label: "sweep",
			}
		}
	}

	if setup.CISD != nil && setup.CISD.Detected {
		// Yang ditandai adalah candle BREAK, bukan RefCandle. RefCandle adalah
		// lilin acuan yang ditembus (bisa berada jauh sebelum jendela eksekusi),
		// sedangkan BreakCandle adalah candle yang menembusnya by body dan
		// close-nya menjadi harga masuk. Memakai RefCandle membuat penanda CISD
		// meleset ke harga acuan yang tidak pernah ditransaksikan.
		waktu := setup.CISD.BreakCandle.Timestamp
		if !waktu.IsZero() {
			spek.CISD = &CISDSpec{
				Time:  waktu.Unix(),
				Price: setup.CISD.BreakCandle.Close,
				Label: "CISD",
			}
		}
	}

	return spek
}

// aturanHarian menegakkan pagar yang sama dengan runner live.
//
// Backtest yang memakai aturan berbeda dari live akan menghasilkan angka yang
// tidak bisa dibandingkan dengan kenyataan, jadi pagar ini disalin apa adanya
// termasuk aturan berhenti setelah menang yang tidak bisa dimatikan.
type aturanHarian struct {
	maksTrade    int
	maksLoss     int
	stopAfterWin bool

	hariAktif  string
	tradeHari  int
	lossHari   int
	menangHari bool
}

// bolehBuka menjawab apakah posisi baru boleh dibuka pada waktu tertentu.
func (a *aturanHarian) bolehBuka(waktu time.Time) bool {
	kunci := waktu.Format("2006-01-02")
	if kunci != a.hariAktif {
		a.hariAktif = kunci
		a.tradeHari = 0
		a.lossHari = 0
		a.menangHari = false
	}
	if a.tradeHari >= a.maksTrade {
		return false
	}
	if a.lossHari >= a.maksLoss {
		return false
	}
	if a.stopAfterWin && a.menangHari {
		return false
	}
	return true
}

// catat memperbarui hitungan harian setelah sebuah posisi selesai.
func (a *aturanHarian) catat(outcome Outcome, waktuTutup time.Time) {
	kunci := waktuTutup.Format("2006-01-02")
	if kunci != a.hariAktif {
		a.hariAktif = kunci
		a.tradeHari = 0
		a.lossHari = 0
		a.menangHari = false
	}
	a.tradeHari++
	switch outcome {
	case OutcomeStopLoss:
		a.lossHari++
	case OutcomeTakeProfit:
		a.menangHari = true
	}
}

// hitungStatistik mengisi ringkasan dari daftar entry.
func hitungStatistik(hasil *Result, modalAwal float64) {
	hasil.TotalTrades = len(hasil.Entries)

	var totalUntung, totalRugi float64
	puncak := modalAwal
	if puncak <= 0 {
		puncak = 1
	}

	for _, entry := range hasil.Entries {
		switch entry.Outcome {
		case OutcomeTakeProfit:
			hasil.Wins++
			totalUntung += entry.ActualR
		case OutcomeStopLoss:
			hasil.Losses++
			totalRugi += math.Abs(entry.ActualR)
		case OutcomeExpired:
			hasil.Expired++
			if entry.ActualR > 0 {
				totalUntung += entry.ActualR
			} else {
				totalRugi += math.Abs(entry.ActualR)
			}
		}
		if entry.Ambiguous {
			hasil.AmbiguousExits++
		}
		hasil.TotalR += entry.ActualR

		if entry.EquityAfter > puncak {
			puncak = entry.EquityAfter
		}
		turun := 0.0
		if puncak > 0 {
			turun = (puncak - entry.EquityAfter) / puncak
		}
		if turun > hasil.MaxDrawdown {
			hasil.MaxDrawdown = turun
		}
		hasil.EquityCurve = append(hasil.EquityCurve, EquityPoint{
			Time:     entry.ExitTime.Unix(),
			Equity:   entry.EquityAfter,
			Drawdown: turun,
		})
	}

	if hasil.TotalTrades > 0 {
		hasil.WinRate = float64(hasil.Wins) / float64(hasil.TotalTrades)
	}
	if totalRugi > 0 {
		hasil.ProfitFactor = totalUntung / totalRugi
	}
}
