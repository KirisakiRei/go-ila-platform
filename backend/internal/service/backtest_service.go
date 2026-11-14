package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"ila-strategy/ila"
	"ila-strategy/indicators"

	"ila/backend/internal/backtest"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// Batas yang mengikat seluruh fitur backtest.
//
// Angka-angka ini bukan setelan tampilan: rentang maksimum, kuota per tier, dan
// jumlah candle semuanya memutuskan berapa memori yang dipakai satu job. VPS
// platform hanya punya sekitar 900 MB, jadi satu backtest yang lepas kendali
// cukup untuk mematikan seluruh situs.
const (
	// BatasRentangHariSemua melindungi mesin dari rentang enam bulan pada
	// pasangan timeframe pendek, yang bisa berarti puluhan ribu candle LTF.
	BatasRentangHariMaks = 190

	// KuotaHarian per tier. Admin tidak dibatasi.
	KuotaHarianFree  = 1
	KuotaHarianBasic = 3
	KuotaHarianPlus  = 20

	// BatasUmurHariFree, Basic, Plus membatasi panjang rentang yang boleh
	// dipilih. Semakin pendek timeframe, semakin pendek datanya di feed, jadi
	// batas ini mengikuti apa yang benar-benar tersedia.
	BatasBulanFree  = 1
	BatasBulanBasic = 3
	BatasBulanPlus  = 6

	// MaksJobAktifPerUser menjaga satu pengguna tidak memenuhi antrean.
	MaksJobAktifPerUser = 1
	// MaksJobBersamaanPerAdmin memberi ruang lebih bagi admin yang menguji.
	MaksJobBersamaanPerAdmin = 3

	// kerjaWorkerBerhenti menentukan berapa lama worker menunggu sebelum
	// memeriksa ulang antrean kosong.
	kerjaWorkerBerhenti = 3 * time.Second

	// batasDurasiSatuJob menghentikan simulasi yang menggantung.
	batasDurasiSatuJob = 4 * time.Minute
)

// Pengaturan bawaan bila pengguna tidak mengubah modal dan risiko.
//
// Nilainya sama dengan yang dipakai runner live, supaya hasil backtest bisa
// dibandingkan apel dengan apel.
const (
	ekuitasBawaan      = 10000.0
	risikoBawaanPersen = 0.5
)

// backtestQuotaStore adalah kebutuhan layanan terhadap penyimpanan kuota.
type backtestQuotaStore interface {
	KlaimKuota(ctx context.Context, userID, tanggal string, batas int) error
	SisaKuota(ctx context.Context, userID, tanggal string, batas int) (int, error)
	KembalikanKuota(ctx context.Context, userID, tanggal string) error
}

// BacktestService mengatur pembuatan, penjalanan, dan pembacaan backtest.
type BacktestService struct {
	runs     *repository.BacktestRepository
	users    userFinder
	coverage *BacktestCoverageService
	provider ProviderCandleBacktest
	features TierFeatureReader
	logger   *slog.Logger
	zonaHari *time.Location
	// antrean mengirim ID run baru ke worker. Buffernya kecil: bila penuh,
	// worker akan tetap menemukannya lewat pemindaian berkala, jadi tidak ada
	// job yang hilang karena buffer sesak.
	antrean chan string
	// pemicuAntrean dipasang worker saat mulai berjalan. Selama nilainya kosong
	// (mis. saat uji unit), job baru cukup menunggu pemindaian berkala.
	pemicuAntrean chan struct{}
}

// BacktestServiceDeps mengumpulkan dependensi layanan backtest.
type BacktestServiceDeps struct {
	Runs     *repository.BacktestRepository
	Users    userFinder
	Coverage *BacktestCoverageService
	Provider ProviderCandleBacktest
	Features TierFeatureReader
	Logger   *slog.Logger
}

// NewBacktestService merangkai layanan backtest.
func NewBacktestService(deps BacktestServiceDeps) *BacktestService {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &BacktestService{
		runs:     deps.Runs,
		users:    deps.Users,
		coverage: deps.Coverage,
		provider: deps.Provider,
		features: deps.Features,
		logger:   logger,
		zonaHari: waktuHarian(),
		antrean:  make(chan string, 64),
	}
}

// backtestHak mengumpulkan batas yang berlaku untuk seorang pengguna.
type backtestHak struct {
	Tier        string
	Admin       bool
	KuotaHarian int
	BatasBulan  int
	MaksAktif   int
}

// hakUntuk menghitung batas dari tier efektif, bukan dari kolom mentah.
func (s *BacktestService) hakUntuk(ctx context.Context, userID string) (backtestHak, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return backtestHak{}, err
	}
	if user == nil {
		return backtestHak{}, apperror.ErrNotFound
	}
	if user.Role == "admin" || user.Role == "superadmin" {
		return backtestHak{
			Tier:        model.TierPlus,
			Admin:       true,
			KuotaHarian: 0, // 0 berarti tanpa batas
			BatasBulan:  BatasBulanPlus,
			MaksAktif:   MaksJobBersamaanPerAdmin,
		}, nil
	}

	tier := effectiveTier(user, time.Now().UTC())
	switch tier {
	case model.TierPlus:
		return backtestHak{Tier: tier, KuotaHarian: KuotaHarianPlus, BatasBulan: BatasBulanPlus, MaksAktif: MaksJobAktifPerUser}, nil
	case model.TierBasic:
		return backtestHak{Tier: tier, KuotaHarian: KuotaHarianBasic, BatasBulan: BatasBulanBasic, MaksAktif: MaksJobAktifPerUser}, nil
	default:
		return backtestHak{Tier: model.TierFree, KuotaHarian: KuotaHarianFree, BatasBulan: BatasBulanFree, MaksAktif: MaksJobAktifPerUser}, nil
	}
}

// Cakupan mengembalikan cakupan data historis satu simbol untuk pengguna.
func (s *BacktestService) Cakupan(ctx context.Context, userID, symbol, feed string, periksaUlang bool) (dto.BacktestCoverageResponse, error) {
	if _, err := s.hakUntuk(ctx, userID); err != nil {
		return dto.BacktestCoverageResponse{}, err
	}

	cakupan, err := s.coverage.Ambil(ctx, symbol, feed, periksaUlang)
	if err != nil {
		return dto.BacktestCoverageResponse{}, err
	}

	balasan := dto.BacktestCoverageResponse{
		Symbol:        cakupan.Symbol,
		Feed:          cakupan.Feed,
		Mulai:         cakupan.Mulai,
		Selesai:       cakupan.Selesai,
		DiperiksaPada: cakupan.DiperiksaPada.Unix(),
	}
	for _, satu := range cakupan.Rentang {
		balasan.Rentang = append(balasan.Rentang, dto.BacktestTimeframeCoverageDTO{
			Timeframe: satu.Timeframe,
			Mulai:     satu.Mulai,
			Selesai:   satu.Selesai,
			Jumlah:    satu.Jumlah,
		})
	}
	for _, pasangan := range DaftarPasanganBacktest {
		rentang := cakupan.RentangCakupan(pasangan)
		balasan.Pasangan = append(balasan.Pasangan, dto.BacktestPairOptionDTO{
			HTF:      pasangan.HTF,
			LTF:      pasangan.LTF,
			Label:    pasangan.HTF + " ke " + pasangan.LTF,
			Mulai:    rentang.Mulai,
			Selesai:  rentang.Selesai,
			Tersedia: rentang.Mulai > 0 && rentang.Selesai > rentang.Mulai,
		})
	}
	return balasan, nil
}

// Kuota mengembalikan sisa jatah hari ini beserta batas rentang tier.
func (s *BacktestService) Kuota(ctx context.Context, userID string) (dto.BacktestQuotaResponse, error) {
	hak, err := s.hakUntuk(ctx, userID)
	if err != nil {
		return dto.BacktestQuotaResponse{}, err
	}

	tanggal := time.Now().In(s.zonaHari).Format("2006-01-02")
	balasan := dto.BacktestQuotaResponse{
		Tier:          hak.Tier,
		BatasHarian:   hak.KuotaHarian,
		MaksHari:      BatasRentangHariMaks,
		RentangMaksRb: hak.BatasBulan,
	}

	if hak.Admin {
		balasan.Sisa = -1 // negatif berarti tanpa batas
		balasan.BisaJalankan = true
		return balasan, nil
	}

	sisa, err := s.runs.SisaKuota(ctx, userID, tanggal, hak.KuotaHarian)
	if err != nil {
		return dto.BacktestQuotaResponse{}, err
	}
	balasan.Sisa = sisa
	balasan.Terpakai = hak.KuotaHarian - sisa
	balasan.BisaJalankan = sisa > 0
	if !balasan.BisaJalankan {
		balasan.Alasan = "Jatah backtest hari ini sudah habis. Sisa jatah kembali besok."
	}
	return balasan, nil
}

// Buat memvalidasi permintaan, memotong kuota, lalu memasukkan job ke antrean.
//
// Kuota dipotong SEBELUM job dibuat, dan dikembalikan bila penyimpanan gagal:
// urutan ini mencegah pengguna menembus batas dengan menembak banyak permintaan
// sekaligus sementara setiap permintaan terlihat belum tercatat.
func (s *BacktestService) Buat(ctx context.Context, userID string, req dto.BacktestCreateRequest) (dto.BacktestRunResponse, error) {
	hak, err := s.hakUntuk(ctx, userID)
	if err != nil {
		return dto.BacktestRunResponse{}, err
	}

	pasangan, ok := CariPasanganBacktest(req.HTF, req.LTF)
	if !ok {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: pasangan timeframe %s ke %s tidak didukung", apperror.ErrBadRequest, req.HTF, req.LTF)
	}

	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: pasangan simbol wajib dipilih", apperror.ErrBadRequest)
	}

	if req.Dari <= 0 || req.Sampai <= 0 {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: rentang tanggal belum lengkap", apperror.ErrBadRequest)
	}
	if req.Sampai <= req.Dari {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: tanggal akhir harus setelah tanggal mulai", apperror.ErrBadRequest)
	}

	dari := time.Unix(req.Dari, 0).UTC()
	sampai := time.Unix(req.Sampai, 0).UTC()

	if hemat := time.Now().UTC().AddDate(0, 0, 2); sampai.After(hemat) {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: tanggal akhir tidak boleh melewati hari ini", apperror.ErrBadRequest)
	}

	// Batas tier dipotong lagi ke batas mutlak: tier Plus memang enam bulan,
	// tetapi mesin tidak pernah mau menerima rentang yang lebih panjang dari
	// yang bisa ditanggung memori.
	batasBulan := hak.BatasBulan
	if batasBulan > BatasBulanPlus {
		batasBulan = BatasBulanPlus
	}
	// Toleransi satu hari, bukan kelonggaran: tanggal akhir boleh sampai
	// esok, sehingga awal rentang enam bulan lalu jatuh sedikit lebih tua
	// daripada AddDate(0,-6,0). Tanpa toleransi ini, pengguna yang memilih
	// "enam bulan terakhir" persis ditolak karena selisih beberapa detik —
	// dan pesan "hanya enam bulan terakhir" terbaca seperti salah sistem.
	batasAwal := time.Now().UTC().AddDate(0, -batasBulan, -1)
	if dari.Before(batasAwal) {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: paket %s hanya bisa menguji %d bulan terakhir", apperror.ErrBadRequest, hak.Tier, batasBulan)
	}
	if sampai.Sub(dari) > BatasRentangHariMaks*24*time.Hour {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: rentang paling panjang %d hari", apperror.ErrBadRequest, BatasRentangHariMaks)
	}

	// Satu job aktif per pengguna. Antrean yang menumpuk untuk satu akun akan
	// membuat pengguna lain menunggu tanpa alasan.
	aktif, err := s.runs.JumlahBerjalan(ctx, userID)
	if err != nil {
		return dto.BacktestRunResponse{}, err
	}
	if aktif >= int64(hak.MaksAktif) {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: masih ada backtest yang berjalan, tunggu sampai selesai", apperror.ErrConflict)
	}

	equity := req.Equity
	if equity <= 0 {
		equity = ekuitasBawaan
	}
	risk := req.Risk
	if risk <= 0 {
		risk = risikoBawaanPersen
	}
	if risk > 5 {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: risiko per trade paling besar 5%%", apperror.ErrBadRequest)
	}
	if equity < 100 || equity > 1_000_000_000 {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: modal harus antara 100 dan 1.000.000.000", apperror.ErrBadRequest)
	}

	// Cakupan diperiksa sekali lagi sebelum job dibuat. Pemilih tanggal di UI
	// sudah dikunci ke cakupan, tetapi permintaan bisa datang tanpa lewat UI;
	// tanpa pemeriksaan ini feed akan dipanggil dengan rentang mustahil dan
	// pengguna menerima galat tanpa penjelasan.
	cakupan, err := s.coverage.Ambil(ctx, symbol, "", false)
	if err != nil {
		return dto.BacktestRunResponse{}, err
	}
	rentang := cakupan.RentangCakupan(pasangan)
	if rentang.Mulai == 0 || rentang.Selesai == 0 {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: data %s untuk %s belum tersedia", apperror.ErrServiceUnavailable, pasangan.LTF, symbol)
	}
	if dari.Unix() < rentang.Mulai {
		return dto.BacktestRunResponse{}, fmt.Errorf("%w: data %s dimulai %s, tanggal mulai terlalu awal",
			apperror.ErrBadRequest, pasangan.LTF,
			time.Unix(rentang.Mulai, 0).In(s.zonaHari).Format("2 Jan 2006"))
	}
	if sampai.Unix() > rentang.Selesai {
		sampai = time.Unix(rentang.Selesai, 0).UTC()
	}

	tanggal := time.Now().In(s.zonaHari).Format("2006-01-02")
	if !hak.Admin {
		if err := s.runs.KlaimKuota(ctx, userID, tanggal, hak.KuotaHarian); err != nil {
			if err == repository.ErrKuotaHabis {
				return dto.BacktestRunResponse{}, fmt.Errorf("%w: jatah backtest hari ini sudah habis", apperror.ErrTooManyRequests)
			}
			return dto.BacktestRunResponse{}, err
		}
	}

	run := &model.BacktestRun{
		UserID:      userID,
		Symbol:      symbol,
		HTF:         pasangan.HTF,
		LTF:         pasangan.LTF,
		RangeFrom:   dari,
		RangeTo:     sampai,
		Equity:      equity,
		RiskPercent: risk,
		Strategy:    "ila-2.0",
		StrategyVer: ila.Version,
		Status:      model.BacktestStatusQueued,
	}
	if err := s.runs.Buat(ctx, run); err != nil {
		// Job tidak pernah lahir, jadi jatah yang terlanjur dipotong harus
		// dikembalikan. Kegagalan pengembalian dicatat, tidak menggagalkan
		// balasan: pengguna lebih perlu tahu job-nya tidak dibuat.
		if !hak.Admin {
			if kembalikanErr := s.runs.KembalikanKuota(ctx, userID, tanggal); kembalikanErr != nil {
				s.logger.Error("gagal mengembalikan kuota backtest", "error", kembalikanErr.Error(), "user", userID)
			}
		}
		return dto.BacktestRunResponse{}, err
	}

	select {
	case s.antrean <- run.ID:
	default:
		// Buffer penuh: worker akan menemukannya lewat pemindaian berkala.
	}

	return s.ringkas(run, backtest.Result{}), nil
}

// Riwayat mengembalikan daftar run milik pengguna.
func (s *BacktestService) Riwayat(ctx context.Context, userID string, limit, offset int) ([]dto.BacktestRunResponse, int64, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	daftar, total, err := s.runs.Daftar(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	hasil := make([]dto.BacktestRunResponse, 0, len(daftar))
	for i := range daftar {
		hasil = append(hasil, s.ringkas(&daftar[i], backtest.Result{}))
	}
	return hasil, total, nil
}

// Ambil mengembalikan satu run beserta kurva ekuitasnya.
func (s *BacktestService) Ambil(ctx context.Context, userID, id string) (dto.BacktestRunDetailResponse, error) {
	run, err := s.runs.Ambil(ctx, id)
	if err != nil {
		return dto.BacktestRunDetailResponse{}, err
	}
	// Run milik orang lain dijawab "tidak ditemukan", bukan "terlarang":
	// membedakannya akan memberi tahu penyerang bahwa ID itu ada.
	if run == nil || run.UserID != userID {
		return dto.BacktestRunDetailResponse{}, apperror.ErrNotFound
	}

	balasan := dto.BacktestRunDetailResponse{BacktestRunResponse: s.ringkas(run, backtest.Result{})}
	if run.EquityCurveJSON != "" {
		var kurva []backtest.EquityPoint
		if err := json.Unmarshal([]byte(run.EquityCurveJSON), &kurva); err == nil {
			for _, titik := range kurva {
				balasan.KurvaEkuitas = append(balasan.KurvaEkuitas, dto.BacktestEquityPointDTO{
					Waktu:    titik.Time,
					Ekuitas:  titik.Equity,
					Drawdown: titik.Drawdown,
				})
			}
		}
	}
	return balasan, nil
}

// TradeDaftar mengembalikan daftar entry sebuah run.
func (s *BacktestService) TradeDaftar(ctx context.Context, userID, id string, limit, offset int) ([]dto.BacktestTradeDTO, int64, error) {
	if err := s.pastikanMilik(ctx, userID, id); err != nil {
		return nil, 0, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	trades, total, err := s.runs.TradeDaftar(ctx, id, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	hasil := make([]dto.BacktestTradeDTO, 0, len(trades))
	for i := range trades {
		hasil = append(hasil, ringkasTrade(&trades[i]))
	}
	return hasil, total, nil
}

// TradeAmbil mengembalikan satu entry beserta candle dan levelnya untuk chart.
func (s *BacktestService) TradeAmbil(ctx context.Context, userID, id string, seq int) (dto.BacktestTradeDetailResponse, error) {
	if err := s.pastikanMilik(ctx, userID, id); err != nil {
		return dto.BacktestTradeDetailResponse{}, err
	}

	run, err := s.runs.AmbilMilik(ctx, id, userID)
	if err != nil {
		return dto.BacktestTradeDetailResponse{}, err
	}
	if run == nil {
		return dto.BacktestTradeDetailResponse{}, apperror.ErrNotFound
	}

	trade, err := s.runs.TradeAmbil(ctx, id, seq)
	if err != nil {
		return dto.BacktestTradeDetailResponse{}, err
	}
	if trade == nil {
		return dto.BacktestTradeDetailResponse{}, apperror.ErrNotFound
	}

	balasan := dto.BacktestTradeDetailResponse{
		BacktestTradeDTO: ringkasTrade(trade),
		HTF:              run.HTF,
		LTF:              run.LTF,
	}

	if trade.CandlesPurgedAt != nil {
		balasan.DibersihkanPada = trade.CandlesPurgedAt.Unix()
		return balasan, nil
	}
	if trade.CandlesJSON == "" {
		return balasan, nil
	}

	var jendela struct {
		HTF []indicators.Candle `json:"htf"`
		LTF []indicators.Candle `json:"ltf"`
	}
	if err := json.Unmarshal([]byte(trade.CandlesJSON), &jendela); err != nil {
		s.logger.Warn("candle trade tidak terbaca", "error", err.Error(), "run", id, "seq", seq)
		return balasan, nil
	}
	balasan.CandleHTF = ubahCandle(jendela.HTF)
	balasan.CandleLTF = ubahCandle(jendela.LTF)

	if trade.LevelsJSON != "" {
		var level backtest.LevelSpec
		if err := json.Unmarshal([]byte(trade.LevelsJSON), &level); err == nil {
			if level.FVG != nil {
				balasan.ZonaFVG = &dto.BacktestZoneDTO{
					Top: level.FVG.Top, Bottom: level.FVG.Bottom,
					Dari: level.FVG.TimeFrom, Sampai: level.FVG.TimeTo, Label: level.FVG.Label,
				}
			}
			if level.Sweep != nil {
				balasan.Sweep = &dto.BacktestMarkerDTO{Waktu: level.Sweep.Time, Harga: level.Sweep.Price, Label: level.Sweep.Label}
			}
			if level.CISD != nil {
				balasan.CISD = &dto.BacktestMarkerDTO{Waktu: level.CISD.Time, Harga: level.CISD.Price, Label: level.CISD.Label}
			}
		}
	}

	return balasan, nil
}

// Hapus membuang satu run milik pengguna.
func (s *BacktestService) Hapus(ctx context.Context, userID, id string) error {
	if err := s.pastikanMilik(ctx, userID, id); err != nil {
		return err
	}
	return s.runs.Hapus(ctx, id, userID)
}

func (s *BacktestService) pastikanMilik(ctx context.Context, userID, id string) error {
	run, err := s.runs.Ambil(ctx, id)
	if err != nil {
		return err
	}
	if run == nil || run.UserID != userID {
		return apperror.ErrNotFound
	}
	return nil
}

// ringkas mengubah baris run menjadi balasan API.
func (s *BacktestService) ringkas(run *model.BacktestRun, _ backtest.Result) dto.BacktestRunResponse {
	balasan := dto.BacktestRunResponse{
		ID:         run.ID,
		Symbol:     run.Symbol,
		HTF:        run.HTF,
		LTF:        run.LTF,
		Status:     run.Status,
		Error:      run.ErrorMsg,
		Dari:       run.RangeFrom.Unix(),
		Sampai:     run.RangeTo.Unix(),
		Equity:     run.Equity,
		Risk:       run.RiskPercent,
		DibuatPada: run.CreatedAt.Unix(),
		Strategi:   run.Strategy,
		Versi:      run.StrategyVer,
		KodeCommit: run.CodeCommit,
		Statistik: dto.BacktestStatsDTO{
			TotalTrade:    run.TotalTrades,
			Menang:        run.Wins,
			Kalah:         run.Losses,
			Kedaluwarsa:   run.Expired,
			CandleAmbigu:  run.AmbiguousExits,
			SinyalDitolak: run.SignalsRejected,
			WinRate:       run.WinRate,
			TotalR:        run.TotalR,
			MaksDrawdown:  run.MaxDrawdown,
			ProfitFactor:  run.ProfitFactor,
			EkuitasMulai:  run.Equity,
			// Ekuitas akhir adalah titik terakhir kurva. Tanpa ini UI menerima
			// nol dan menampilkannya sebagai modal habis — padahal nol berarti
			// "belum dihitung", bukan "bangkrut".
			EkuitasAkhir: ekuitasTerakhir(run),
		},
	}
	if run.StartedAt != nil {
		balasan.MulaiPada = run.StartedAt.Unix()
	}
	if run.FinishedAt != nil {
		balasan.SelesaiPada = run.FinishedAt.Unix()
	}
	return balasan
}

func ringkasTrade(trade *model.BacktestTrade) dto.BacktestTradeDTO {
	return dto.BacktestTradeDTO{
		Seq:            trade.Seq,
		Arah:           trade.Direction,
		WaktuMasuk:     trade.EntryTime.Unix(),
		WaktuKeluar:    trade.ExitTime.Unix(),
		Entry:          trade.EntryPrice,
		StopLoss:       trade.StopLoss,
		TakeProfit:     trade.TakeProfit,
		RRRencana:      trade.PlannedRR,
		Hasil:          trade.Outcome,
		R:              trade.ActualR,
		DurasiDetik:    trade.Duration,
		EkuitasAkhir:   trade.EquityAfter,
		Ambigu:         trade.Ambiguous,
		Ringkasan:      trade.Summary,
		LangkahBerikut: trade.NextAction,
		ChartTersedia:  trade.CandlesPurgedAt == nil && trade.CandlesJSON != "",
	}
}

func ubahCandle(candles []indicators.Candle) []dto.BacktestCandleDTO {
	hasil := make([]dto.BacktestCandleDTO, 0, len(candles))
	for _, candle := range candles {
		hasil = append(hasil, dto.BacktestCandleDTO{
			Time:   candle.Timestamp.Unix(),
			Open:   candle.Open,
			High:   candle.High,
			Low:    candle.Low,
			Close:  candle.Close,
			Volume: candle.Volume,
		})
	}
	return hasil
}

// parseAngka membaca angka dari fitur paket dengan cadangan.
func parseAngka(nilai string, cadangan int) int {
	angka, err := strconv.Atoi(strings.TrimSpace(nilai))
	if err != nil || angka <= 0 {
		return cadangan
	}
	return angka
}

// parsinganDipakai menandai parseAngka disimpan untuk pembacaan kuota dari
// katalog paket pada pengembangan berikutnya.
var _ = parseAngka

// ekuitasTerakhir membaca titik terakhir kurva ekuitas.
//
// Kurva disimpan sebagai JSON di baris run, bukan sebagai kolom angka: satu
// titik per trade sudah cukup untuk digambar, dan kolom tambahan hanya
// menduplikasi nilai yang bisa dihitung dari baris trade terakhir. Bila kurva
// kosong (run tanpa trade), modal awal dipakai agar UI tidak menampilkan nol.
func ekuitasTerakhir(run *model.BacktestRun) float64 {
	if run.EquityCurveJSON == "" {
		return run.Equity
	}
	var kurva []backtest.EquityPoint
	if err := json.Unmarshal([]byte(run.EquityCurveJSON), &kurva); err != nil || len(kurva) == 0 {
		return run.Equity
	}
	return kurva[len(kurva)-1].Equity
}
