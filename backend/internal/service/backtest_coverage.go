package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"ila/backend/internal/market"
	"ila/backend/pkg/apperror"
)

// Batas cakupan historis diukur langsung ke TradingView pada 14 November 2026.
//
// Angka ini bukan tebakan: TradingView memang hanya menyimpan data sedalam ini
// untuk tiap resolusi, dan itulah alasan pemilih tanggal di UI dikunci ke
// cakupan yang benar-benar tersedia alih-alih membiarkan pengguna memilih
// rentang yang mustahil dipenuhi.
//
// Isi peta ini hanya cadangan tampilan. Sumber kebenarannya tetap candle
// sungguhan yang diambil sekali per simbol per hari; angka di sini dipakai
// untuk memberi tahu pengguna lebih awal bahwa cakupannya memang terbatas.
var rentangTersediaMenurutResolusi = map[string]time.Duration{
	"D1":  540 * 24 * time.Hour,
	"H4":  1400 * 24 * time.Hour,
	"H1":  320 * 24 * time.Hour,
	"M15": 95 * 24 * time.Hour,
	"M5":  30 * 24 * time.Hour,
	"M1":  7 * 24 * time.Hour,
}

// PasanganTimeframeBacktest adalah satu kombinasi timeframe yang boleh diuji.
//
// Lima pasangan (Monthly s.d. Micro) sesuai tabel time cycle New ILA 2.0.
type PasanganTimeframeBacktest struct {
	HTF string
	LTF string
}

// DaftarPasanganBacktest adalah pilihan yang ditawarkan ke pengguna.
var DaftarPasanganBacktest = []PasanganTimeframeBacktest{
	{HTF: "W", LTF: "H4"},
	{HTF: "D1", LTF: "H1"},
	{HTF: "H4", LTF: "M15"},
	{HTF: "H1", LTF: "M5"},
	{HTF: "M15", LTF: "M1"},
}

// CariPasanganBacktest memeriksa apakah kombinasi timeframe termasuk yang
// didukung, dan mengembalikan bentuk bakunya.
func CariPasanganBacktest(htf, ltf string) (PasanganTimeframeBacktest, bool) {
	htf = strings.ToUpper(strings.TrimSpace(htf))
	ltf = strings.ToUpper(strings.TrimSpace(ltf))
	for _, pasangan := range DaftarPasanganBacktest {
		if pasangan.HTF == htf && pasangan.LTF == ltf {
			return pasangan, true
		}
	}
	return PasanganTimeframeBacktest{}, false
}

// batasCandleBacktest adalah jumlah candle yang diminta per timeframe.
//
// Lima ribu dipilih setelah pengukuran langsung: di atas angka itu TradingView
// mulai mengirim data rusak (high < low) alih-alih memberi galat yang jelas.
const batasCandleBacktest = 5000

// batasWaktuAmbilCandle membatasi satu pengambilan candle.
//
// Panggilan TradingView yang menggantung harus menyerah jauh sebelum batas
// waktu HTTP 30 detik, supaya pengguna menerima keterangan alih-alih koneksi
// yang putus tanpa pesan.
const batasWaktuAmbilCandle = 20 * time.Second

// CakupanTimeframe adalah jendela data yang benar-benar tersedia untuk satu
// timeframe.
type CakupanTimeframe struct {
	Timeframe string `json:"timeframe"`
	// Mulai dan Selesai dalam Unix detik UTC; nol berarti tidak ada data.
	Mulai   int64 `json:"mulai"`
	Selesai int64 `json:"selesai"`
	Jumlah  int   `json:"jumlah"`
}

// CakupanBacktest adalah hasil pemeriksaan data untuk satu simbol.
type CakupanBacktest struct {
	Symbol string `json:"symbol"`
	Feed   string `json:"feed"`
	// Mulai dan Selesai adalah irisan seluruh pasangan timeframe: rentang yang
	// bisa dipilih pengguna tanpa ada timeframe yang kehabisan data.
	Mulai   int64              `json:"mulai"`
	Selesai int64              `json:"selesai"`
	Rentang []CakupanTimeframe `json:"rentang"`
	// DiperiksaPada menandai kapan cakupan ini dibaca dari feed, agar UI bisa
	// menyebutkan bahwa angkanya bukan hasil pengukuran ulang tiap detik.
	DiperiksaPada time.Time `json:"diperiksa_pada"`
}

// RentangCakupan mengembalikan irisan cakupan untuk satu pasangan timeframe.
func (c CakupanBacktest) RentangCakupan(pasangan PasanganTimeframeBacktest) CakupanTimeframe {
	var hasil CakupanTimeframe
	for _, satu := range c.Rentang {
		switch satu.Timeframe {
		case pasangan.HTF:
			hasil.Mulai, hasil.Selesai = satu.Mulai, satu.Selesai
		case pasangan.LTF:
			// Timeframe pendek selalu lebih sempit dalam kasus ILA 2.0, tetapi
			// irisan tetap dihitung supaya aturan ini tidak bergantung urutan.
			if satu.Mulai > hasil.Mulai {
				hasil.Mulai = satu.Mulai
			}
			if hasil.Selesai == 0 || (satu.Selesai > 0 && satu.Selesai < hasil.Selesai) {
				hasil.Selesai = satu.Selesai
			}
		}
	}
	return hasil
}

// ProviderCandleBacktest adalah kebutuhan layanan cakupan terhadap sumber data.
//
// Dipisah sebagai antarmuka agar cakupan bisa diuji tanpa menyentuh jaringan.
type ProviderCandleBacktest interface {
	FetchCandles(ctx context.Context, symbol, feed, timeframe string, limit int) ([]market.Candle, error)
}

// BacktestCoverageService membaca dan menyimpan cakupan data historis.
//
// Cakupan disimpan di memori dengan kunci "simbol|feed|tanggal WIB". Biaya satu
// pembacaan adalah delapan koneksi ke feed (empat pasangan timeframe, dua
// timeframe masing-masing), jadi membacanya ulang untuk setiap pengguna akan
// membebani sumber yang sama yang dipakai chart dashboard. Dengan cache ini,
// semua pengguna yang membuka pasangan sama pada hari yang sama berbagi satu
// hasil.
type BacktestCoverageService struct {
	provider ProviderCandleBacktest
	clock    func() time.Time
	zonaHari *time.Location

	mu    sync.Mutex
	cache map[string]cakupanTersimpan
	// lock mengunci per kunci cache, bukan global, supaya pengguna yang membuka
	// simbol berbeda tidak saling menunggu.
	lock map[string]*sync.Mutex
}

type cakupanTersimpan struct {
	cakupan CakupanBacktest
	// kedaluwarsa jatuh pada tengah malam WIB berikutnya: data harian berubah
	// saat pergantian hari, bukan pada jam yang sama tiap 24 jam.
	kedaluwarsa time.Time
}

// NewBacktestCoverageService merangkai layanan cakupan.
func NewBacktestCoverageService(provider ProviderCandleBacktest) *BacktestCoverageService {
	return &BacktestCoverageService{
		provider: provider,
		clock:    func() time.Time { return time.Now().UTC() },
		zonaHari: waktuHarian(),
		cache:    make(map[string]cakupanTersimpan),
		lock:     make(map[string]*sync.Mutex),
	}
}

// waktuHarian mengembalikan zona harian yang dipakai seluruh platform.
func waktuHarian() *time.Location {
	if zona, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return zona
	}
	return time.FixedZone("WIB", 7*3600)
}

// Ambil mengembalikan cakupan satu simbol, memakai cache bila masih berlaku.
//
// force melewati cache; dipakai saat pengguna menekan "periksa ulang" karena
// data bisa berubah lebih cepat dari perkiraan (mis. feed baru menambah bar).
func (s *BacktestCoverageService) Ambil(ctx context.Context, symbol, feed string, force bool) (CakupanBacktest, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	feed = strings.ToUpper(strings.TrimSpace(feed))
	if symbol == "" {
		return CakupanBacktest{}, fmt.Errorf("%w: simbol wajib diisi", apperror.ErrBadRequest)
	}
	if feed == "" {
		feed = market.DefaultFeed(symbol)
	}

	sekarang := s.clock()
	kunci := symbol + "|" + feed + "|" + sekarang.In(s.zonaHari).Format("2006-01-02")

	perKunci := s.kunciUntuk(kunci)
	perKunci.Lock()
	defer perKunci.Unlock()

	if !force {
		s.mu.Lock()
		tersimpan, ada := s.cache[kunci]
		s.mu.Unlock()
		if ada && sekarang.Before(tersimpan.kedaluwarsa) {
			return tersimpan.cakupan, nil
		}
	}

	cakupan, err := s.bacaDariFeed(ctx, symbol, feed)
	if err != nil {
		return CakupanBacktest{}, err
	}

	s.mu.Lock()
	s.cache[kunci] = cakupanTersimpan{
		cakupan:     cakupan,
		kedaluwarsa: tengahMalamBerikutnya(sekarang, s.zonaHari),
	}
	s.mu.Unlock()

	return cakupan, nil
}

// bacaDariFeed mengambil candle terlama setiap timeframe yang dibutuhkan.
//
// Yang dibaca sengaja hanya satu kali per timeframe meskipun ada beberapa
// pasangan yang memakainya (H4 dipakai W→H4 dan H4→M15): satu permintaan
// melayani keduanya.
func (s *BacktestCoverageService) bacaDariFeed(ctx context.Context, symbol, feed string) (CakupanBacktest, error) {
	timeframes := timeframeUnik()
	hasil := CakupanBacktest{Symbol: symbol, Feed: feed, DiperiksaPada: s.clock()}

	for _, timeframe := range timeframes {
		candles, err := s.ambilCandle(ctx, symbol, feed, timeframe)
		if err != nil {
			// Satu timeframe gagal bukan alasan menggagalkan seluruh pemeriksaan:
			// pasangan lain mungkin tetap bisa dipakai. Kombinasi yang
			// timeframe-nya kosong akan ditolak saat dijalankan.
			hasil.Rentang = append(hasil.Rentang, CakupanTimeframe{Timeframe: timeframe})
			continue
		}
		hasil.Rentang = append(hasil.Rentang, CakupanTimeframe{
			Timeframe: timeframe,
			Mulai:     candles[0].Time,
			Selesai:   candles[len(candles)-1].Time,
			Jumlah:    len(candles),
		})
	}

	sort.SliceStable(hasil.Rentang, func(i, j int) bool {
		return hasil.Rentang[i].Timeframe < hasil.Rentang[j].Timeframe
	})

	hasil.Mulai, hasil.Selesai = irisanCakupan(hasil.Rentang)
	if hasil.Mulai == 0 || hasil.Selesai == 0 || hasil.Mulai >= hasil.Selesai {
		return CakupanBacktest{}, fmt.Errorf("%w: data historis untuk %s belum tersedia di feed %s", apperror.ErrServiceUnavailable, symbol, feed)
	}
	return hasil, nil
}

func (s *BacktestCoverageService) ambilCandle(ctx context.Context, symbol, feed, timeframe string) ([]market.Candle, error) {
	ctxAmbil, batal := context.WithTimeout(ctx, batasWaktuAmbilCandle)
	defer batal()

	candles, err := s.provider.FetchCandles(ctxAmbil, symbol, feed, timeframe, batasCandleBacktest)
	if err != nil {
		return nil, err
	}
	if len(candles) == 0 {
		return nil, fmt.Errorf("feed tidak mengirim candle %s untuk %s", timeframe, symbol)
	}
	sort.SliceStable(candles, func(i, j int) bool { return candles[i].Time < candles[j].Time })
	return candles, nil
}

func (s *BacktestCoverageService) kunciUntuk(kunci string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	perKunci, ada := s.lock[kunci]
	if !ada {
		perKunci = &sync.Mutex{}
		s.lock[kunci] = perKunci
	}
	return perKunci
}

// timeframeUnik mengumpulkan seluruh timeframe yang dipakai pasangan backtest.
func timeframeUnik() []string {
	terlihat := make(map[string]bool, 2*len(DaftarPasanganBacktest))
	daftar := make([]string, 0, 2*len(DaftarPasanganBacktest))
	for _, pasangan := range DaftarPasanganBacktest {
		for _, timeframe := range []string{pasangan.HTF, pasangan.LTF} {
			if !terlihat[timeframe] {
				terlihat[timeframe] = true
				daftar = append(daftar, timeframe)
			}
		}
	}
	return daftar
}

// irisanCakupan mengambil irisan seluruh timeframe pada daftar.
func irisanCakupan(rentang []CakupanTimeframe) (int64, int64) {
	var mulai, selesai int64
	for _, satu := range rentang {
		if satu.Mulai == 0 || satu.Selesai == 0 {
			// Ada timeframe tanpa data: irisan tidak bisa ditentukan.
			return 0, 0
		}
		if mulai == 0 || satu.Mulai > mulai {
			mulai = satu.Mulai
		}
		if selesai == 0 || satu.Selesai < selesai {
			selesai = satu.Selesai
		}
	}
	return mulai, selesai
}

// tengahMalamBerikutnya menghitung kapan cache cakupan harus kedaluwarsa.
func tengahMalamBerikutnya(sekarang time.Time, zona *time.Location) time.Time {
	lokal := sekarang.In(zona)
	besok := time.Date(lokal.Year(), lokal.Month(), lokal.Day(), 0, 0, 0, 0, zona).AddDate(0, 0, 1)
	return besok.UTC()
}
