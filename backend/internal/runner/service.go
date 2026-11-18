// Package runner berisi aturan dan orkestrasi Runner: distributor sinyal
// engine ke akun broker milik user.
//
// Batas tanggung jawab (plans/05, plans/03 §2):
//   - Runner tidak pernah menghitung ulang strategi.
//   - Runner tidak pernah menutup posisi berdasarkan feed lokal.
//   - Runner tidak pernah meneruskan kredensial atau hasil trading ke engine.
package runner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/authstate"
	"ila/backend/internal/broker"
	"ila/backend/internal/market"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	ingest "ila/backend/internal/runner/ingest"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/cryptox"
)

// Publisher menerbitkan perubahan status ke Console.
// Interface dipisah supaya Runner bisa diuji tanpa kanal notifikasi.
type Publisher interface {
	PublishSignal(ctx context.Context, userID, signalID, status, reason string)
	PublishTelemetry(ctx context.Context, accountID string)
	// PublishPositionClosed memberi tahu user bahwa posisinya sudah ditutup.
	// Tanpa ini user tidak pernah tahu hasil trading (plans/19 C1).
	PublishPositionClosed(ctx context.Context, pos *model.Position, daily *DailyOutcome)
}

// PositionOpenedPublisher adalah perluasan opsional Publisher untuk memberi tahu
// kanal notifikasi begitu posisi baru tercatat terbuka (otomatis maupun manual).
type PositionOpenedPublisher interface {
	PublishPositionOpened(ctx context.Context, pos *model.Position, signal *model.Signal, isAuto bool)
}

// DailyOutcome menjelaskan sisa kuota setelah sebuah posisi ditutup.
type DailyOutcome struct {
	TotalTrades int
	MaxTrades   int
	Locked      bool
	LockReason  string
}

// Store adalah kebutuhan penyimpanan Runner.
// Dideklarasikan di sisi pemakai agar paket ini tidak terikat implementasi.
type Store interface {
	repository.RunnerRepository
}

// ClaimStaleAfter adalah umur klaim yang dianggap tertinggal. Lima menit jauh
// lebih lama daripada waktu yang dibutuhkan untuk mengirim satu order, sehingga
// eksekusi yang masih berjalan tidak pernah diambil alih.
const ClaimStaleAfter = 5 * time.Minute

// maxAccountsPerUser membatasi akun yang boleh menerima order dari satu sinyal.
// Satu sinyal melahirkan satu order; akun pertama yang lolos evaluasi dipakai.
const maxAccountsPerUser = 2

// Options mengatur perilaku Runner.
type Options struct {
	// Policy adalah pagar aturan harian default.
	Policy DailyPolicy
	// SlippageGuard mematikan pemeriksaan penyimpangan harga bila false.
	SlippageGuard bool
	// DryRunAll memaksa semua eksekusi menjadi simulasi.
	DryRunAll bool
	// AutoExecute menentukan apakah user boleh dieksekusi tanpa konfirmasi
	// bila akunnya ber-mode auto.
	AutoExecute bool
	// MarketOpenCheck menolak eksekusi saat pasar tutup.
	MarketOpenCheck bool
	// Clock bisa diganti saat pengujian.
	Clock func() time.Time
	// PriceTTL membatasi umur cache harga untuk pemeriksaan slippage.
	PriceTTL time.Duration
	// Settings menyimpan ketersediaan strategi; bila nil dipakai default.
	Settings *Settings
	// LegalGate menandai dokumen wajib yang belum disetujui. Bila nil,
	// pemeriksaan dilewati agar Runner tetap berjalan tanpa penanda.
	LegalGate func(ctx context.Context, userID string) bool
	// Flags adalah pengaturan yang bisa diubah admin tanpa deploy ulang.
	// Bila nil, nilai dari Options yang berlaku.
	Flags Flags
	// Mode membaca mode platform (normal/read-only/pemeliharaan).
	// Bila nil, mode dianggap normal.
	Mode ModeSource
	// OnMarketAnomaly melaporkan status pasar yang tidak sesuai jadwal, mis.
	// engine mengaku buka pada Sabtu. Bila nil, temuan hanya dicatat di log.
	OnMarketAnomaly func(anomalies []MarketScheduleAnomaly)
	// PairGate menyaring simbol yang masih diaktifkan admin. Bila nil, semua
	// simbol dianggap aktif sehingga pemasangan lama tetap berjalan.
	PairGate PairGate
	// CorrelationGate menyediakan pasangan instrumen pembanding SMT. Bila nil,
	// engine hanya diminta simbol yang benar-benar dilanggani user.
	CorrelationGate CorrelationGate
	// ValuePerLot mengembalikan nilai uang untuk pergerakan harga 1.0 pada
	// 1 lot, dipakai menghitung hasil dalam R saat posisi ditutup.
	//
	// Tanpa ini, Runner hanya punya jarak harga yang satuannya berbeda antar
	// pair (EURUSD 0,00164 vs XAUUSD 0,00164 jauh beda nilainya), sehingga R
	// tidak bisa dihitung benar. Bila nil, dipakai nilai bawaan per jenis
	// simbol dari DefaultValuePerLot.
	ValuePerLot func(symbol string) float64
	// CandleFetcher reads the bounded feed window for durable forward tests.
	CandleFetcher func(ctx context.Context, symbol, feed, timeframe string, limit int) ([]market.Candle, error)
	// Evaluations owns durable claims and guarded checkpoint writes.
	Evaluations interface {
		Claim(context.Context, time.Time, int) ([]repository.SignalEvalCandidate, error)
		Save(context.Context, repository.SignalEvalCandidate, *model.UserSignalEval) error
	}
}

// PairGate memberi tahu Runner simbol mana yang masih boleh diminta ke engine.
// Implementasi disarankan menyimpan cache agar tidak memukul basis data pada
// setiap perubahan kebutuhan simbol.
type PairGate interface {
	// EnabledSymbols mengembalikan himpunan simbol kanonik yang masih aktif.
	EnabledSymbols(ctx context.Context) (map[string]bool, error)
}

// CorrelationGate memberi tahu Runner instrumen pembanding SMT per simbol.
//
// Runner hanya memakainya sebagai data: cara membandingkan lilin sepenuhnya
// milik engine. Tujuan gate ini adalah memastikan engine menerima feed untuk
// instrumen pembanding, tanpa membuat user harus melangganinya.
type CorrelationGate interface {
	Correlations(ctx context.Context) (map[string][]string, error)
}

// Service adalah inti Runner. Aman dipakai dari beberapa goroutine.
type Service struct {
	store   Store
	brokers *broker.Registry
	pub     Publisher
	logger  *slog.Logger
	opts    Options

	mu     sync.Mutex
	market enginev1.MarketState
	// marketAt adalah waktu status pasar terakhir DITERIMA dari engine.
	//
	// Kata "diterima", bukan "berubah": Runner perlu tahu kapan terakhir
	// mendengar kabar, bukan kapan nilainya terakhir berbeda. Tanpa ini ia
	// memegang status lama tanpa batas waktu -- kasus nyata 16 Nov 2026:
	// Runner memakai CLOSED dari snapshot Sabtu 14 Nov sampai Senin siang,
	// sehingga semua setup ditahan dengan alasan "Pasar sedang tutup".
	marketAt time.Time
	// engineVers, acks, dan syncStates dipisah per strategi karena satu Runner
	// melayani lebih dari satu engine (ila-2.0 dan ila-3.0) sekaligus. Bila
	// state-nya digabung, penyelarasan satu engine akan memakai daftar simbol
	// engine lain sehingga keduanya saling mencabut langganan.
	engineVers map[string]string
	acks       map[string]AckSender
	syncStates map[string]*strategySyncState
	clients    map[string]broker.Client
	keyring    *cryptox.Keyring
	// schedule mengawasi kecocokan status pasar dengan jadwal NY.
	schedule *ScheduleWatch
	// accounts menyimpan snapshot akun sebentar agar satu eksekusi tidak
	// membaca equity dua kali ke broker.
	accounts *accountSnapshotCache
	// antrean memisahkan pembacaan stream engine dari pemrosesan sinyal,
	// sehingga sinyal siap entry tidak menunggu status WAIT_* di depannya.
	antrean *signalQueue

	// signalLockMu menyerikan PUBLISH dan CANCEL/EXPIRE/SUPERSEDE untuk signal_id
	// yang sama, serta mengingat pembatalan dini agar tidak pernah dibalik urutan.
	signalLockMu  sync.Mutex
	signalLocks   map[string]*sync.Mutex
	recentCancels map[string]earlySignalCancel
}

type earlySignalCancel struct {
	status     string
	reason     string
	superseded string
	recordedAt time.Time
}

// NewService membuat Runner.
func NewService(store Store, brokers *broker.Registry, pub Publisher, logger *slog.Logger, opts Options) *Service {
	if opts.Clock == nil {
		opts.Clock = func() time.Time { return time.Now().UTC() }
	}
	if opts.PriceTTL <= 0 {
		opts.PriceTTL = 30 * time.Second
	}
	if opts.Settings == nil {
		opts.Settings = LoadSettings()
	}
	opts.Policy = opts.Policy.Clamp()

	return &Service{
		store:         store,
		brokers:       brokers,
		pub:           pub,
		logger:        logger,
		opts:          opts,
		market:        enginev1.MarketState_MARKET_OPEN,
		clients:       make(map[string]broker.Client),
		schedule:      NewScheduleWatch(10*time.Minute, 10*time.Minute),
		accounts:      newAccountSnapshotCache(opts.Clock),
		antrean:       newSignalQueue(0),
		signalLocks:   make(map[string]*sync.Mutex),
		recentCancels: make(map[string]earlySignalCancel),
	}
}

func (s *Service) lockSignalOp(signalID string) func() {
	if signalID == "" {
		return func() {}
	}
	s.signalLockMu.Lock()
	if len(s.signalLocks) > 2048 {
		// Bersihkan entri tidak terkunci secara oportunistis agar peta tetap kecil.
		for k, m := range s.signalLocks {
			if k != signalID && m.TryLock() {
				m.Unlock()
				delete(s.signalLocks, k)
			}
		}
	}
	m, ok := s.signalLocks[signalID]
	if !ok {
		m = &sync.Mutex{}
		s.signalLocks[signalID] = m
	}
	s.signalLockMu.Unlock()

	m.Lock()
	return m.Unlock
}

func (s *Service) rememberEarlyCancel(signalID, status, reason, supersededBy string) {
	now := s.opts.Clock()
	s.signalLockMu.Lock()
	defer s.signalLockMu.Unlock()
	for k, v := range s.recentCancels {
		if now.Sub(v.recordedAt) > 10*time.Minute {
			delete(s.recentCancels, k)
		}
	}
	if len(s.recentCancels) >= 1024 {
		for k := range s.recentCancels {
			delete(s.recentCancels, k)
			break
		}
	}
	s.recentCancels[signalID] = earlySignalCancel{
		status:     status,
		reason:     reason,
		superseded: supersededBy,
		recordedAt: now,
	}
}

func (s *Service) takeEarlyCancel(signalID string) (earlySignalCancel, bool) {
	now := s.opts.Clock()
	s.signalLockMu.Lock()
	defer s.signalLockMu.Unlock()
	item, ok := s.recentCancels[signalID]
	if !ok {
		return earlySignalCancel{}, false
	}
	delete(s.recentCancels, signalID)
	if now.Sub(item.recordedAt) > 10*time.Minute {
		return earlySignalCancel{}, false
	}
	return item, true
}

// SetMarketState mencatat status pasar terakhir dari engine.
func (s *Service) SetMarketState(state enginev1.MarketState) {
	s.mu.Lock()
	changed := s.market != state
	s.market = state
	s.marketAt = s.opts.Clock()
	s.mu.Unlock()

	// Perubahan status pasar jarang; hanya dicatat saat benar-benar berubah
	// agar Console bisa menampilkannya tanpa bertanya ke Engine.
	if changed {
		s.persistMarketState(state)
	}

	s.checkSchedule(state)
}

// checkSchedule membandingkan status pasar engine dengan jadwal NY.
//
// Engine yang salah zona waktu akan mengaku buka di akhir pekan. Tanpa
// pemeriksaan ini, kekeliruan itu tidak pernah terlihat sampai ada yang
// menyadari tidak ada sinyal selama berhari-hari (plans/22 §3).
func (s *Service) checkSchedule(state enginev1.MarketState) {
	if s.schedule == nil {
		return
	}
	now := s.opts.Clock()

	// Status MARKET_UNKNOWN tidak dibandingkan: engine belum tahu apa pun.
	if state == enginev1.MarketState_MARKET_STATE_UNSPECIFIED {
		if s.schedule.Stale(now) {
			s.logger.Warn("status pasar belum pernah diterima dari engine")
		}
		return
	}

	anomalies := s.schedule.Observe(state == enginev1.MarketState_MARKET_OPEN, now)
	if len(anomalies) == 0 {
		return
	}
	for _, anomaly := range anomalies {
		s.logger.Warn("status pasar tidak sesuai jadwal", "anomali", string(anomaly))
	}
	if s.opts.OnMarketAnomaly != nil {
		s.opts.OnMarketAnomaly(anomalies)
	}
}

type marketStateUpdater interface {
	UpdateEngineMarketState(ctx context.Context, state string, markedAt time.Time) error
}

// persistMarketState menyimpan status pasar pada engine yang terdaftar tanpa
// menimpa kolom connected atau last_seen_at.
func (s *Service) persistMarketState(state enginev1.MarketState) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := s.opts.Clock()
	label := marketStateLabel(state)

	if updater, ok := s.store.(marketStateUpdater); ok {
		if err := updater.UpdateEngineMarketState(ctx, label, now); err != nil {
			s.logger.Warn("gagal menyimpan status pasar", "error", err.Error())
		}
		return
	}

	for _, strategy := range []string{"ila-2.0", "ila-3.0"} {
		record := &model.EngineState{
			ID:          "engine:" + strategy,
			Strategy:    strategy,
			MarketState: label,
			MarkedAt:    &now,
			UpdatedAt:   now,
		}
		if err := s.store.UpsertEngineState(ctx, record); err != nil {
			s.logger.Warn("gagal menyimpan status pasar", "error", err.Error())
		}
	}
}

// marketStateLabel mengubah enum engine menjadi label pendek untuk dashboard.
func marketStateLabel(state enginev1.MarketState) string {
	switch state {
	case enginev1.MarketState_MARKET_OPEN:
		return "OPEN"
	case enginev1.MarketState_MARKET_CLOSED:
		return "CLOSED"
	case enginev1.MarketState_MARKET_ROLLOVER_BLACKOUT:
		return "ROLLOVER"
	default:
		return "UNKNOWN"
	}
}

// SetEngineVersion mencatat build engine yang sedang terhubung.
func (s *Service) SetEngineVersion(strategy, version string) {
	s.mu.Lock()
	if s.engineVers == nil {
		s.engineVers = make(map[string]string, 2)
	}
	s.engineVers[strategy] = version
	s.mu.Unlock()
}

// EngineVersion mengembalikan versi engine terakhir yang dilaporkan untuk satu
// strategi. Dipakai Runner saat mencatat status koneksi agar kolom
// engine_version tidak kosong di panel kesehatan.
func (s *Service) EngineVersion(strategy string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.engineVers[strategy]
}

// MarketOpen melaporkan apakah pasar boleh dieksekusi saat ini.
//
// Diekspor agar console dan tes memakai keputusan yang sama persis dengan
// jalur eksekusi, bukan salinan logika yang bisa berbeda.
func (s *Service) MarketOpen() bool {
	return s.marketOpen()
}

// marketStatusMaxAge adalah batas umur status pasar yang masih dipercaya.
//
// Engine mengirim status pada setiap denyut (30 detik), jadi status yang lebih
// tua dari ini berarti kabar dari engine sudah berhenti -- misalnya stream
// tergantung tanpa terdeteksi putus. Dalam keadaan itu jadwal pasar lokal yang
// menentukan, bukan nilai lama yang bisa sama sekali tidak relevan.
const marketStatusMaxAge = 3 * time.Minute

// marketOpen memutuskan apakah pasar boleh dieksekusi saat ini.
//
// Sumber utamanya status dari engine. Bila status itu sudah terlalu tua,
// keputusan jatuh ke jadwal pasar lokal (ExpectedOpen, berbasis jam New York)
// alih-alih memakai nilai basi.
//
// Sebelumnya fungsi ini hanya membaca nilai terakhir tanpa memeriksa umurnya,
// sehingga koneksi engine yang bertahan lama membuat Runner menahan semua
// sinyal dengan alasan "Pasar sedang tutup" padahal pasar sudah buka.
func (s *Service) marketOpen() bool {
	return s.marketOpenAt(s.opts.Clock())
}

// marketOpenAt menguji status pasar pada satu waktu tertentu.
//
// Dipakai saat menilai sinyal: alasannya dibaca user sebagai keterangan tentang
// jam sinyal itu, bukan tentang jam pemeriksaan. Memakai jam sekarang membuat
// sinyal yang terbit saat sesi buka tertulis "Pasar sedang tutup" hanya karena
// diperiksa setelah sesi berganti (kasus USTEC 16 Nov 2026).
func (s *Service) marketOpenAt(t time.Time) bool {
	s.mu.Lock()
	state, at := s.market, s.marketAt
	s.mu.Unlock()
	if at.IsZero() || s.opts.Clock().Sub(at) > marketStatusMaxAge {
		if s.logger != nil {
			s.logger.Warn("status pasar dari engine basi, memakai jadwal lokal",
				"umur", s.opts.Clock().Sub(at).String(), "dari_engine", state.String())
		}
		return ExpectedOpen(t)
	}
	return ingest.MarketOpen(state)
}

// ---------------------------------------------------------------------------
// Ingest sinyal
// ---------------------------------------------------------------------------

// HandleSignal memproses sinyal baru dari engine untuk seluruh user yang
// memantau simbolnya. Setiap user dinilai terpisah: satu sinyal bisa menjadi
// posisi bagi user A, tertahan bagi user B, dan terlewat bagi user C.
// HandleSignal memproses satu sinyal sampai selesai pada goroutine pemanggil.
//
// Dipakai pengujian dan pemanggil yang memang menginginkan hasil langsung.
// Jalur produksi memakai HandleSignalAsync agar pembacaan stream tidak menahan
// sinyal berikutnya.
func (s *Service) HandleSignal(ctx context.Context, in *enginev1.Signal) {
	signal := s.siapkanSinyal(ctx, in)
	if signal == nil {
		return
	}
	s.prosesSinyalMasuk(ctx, signal)
}

// HandleSignalAsync menerima sinyal dari stream engine tanpa menahan pembaca.
//
// Callback stream gRPC dipanggil sinkron, sehingga pekerjaan berat di sana
// menahan seluruh sinyal di belakangnya. Data produksi menunjukkannya: sinyal
// yang rilis bersamaan tiba antara 3,6 detik sampai 20,7 detik setelah candle
// tutup, dan sinyal siap entry selalu paling terakhir karena engine mengirim
// status WAIT_* lebih dulu.
//
// Antrean berprioritas membalik urutan itu: sinyal siap entry didahulukan,
// sedangkan status WAIT_* menyusul di belakang.
func (s *Service) HandleSignalAsync(ctx context.Context, in *enginev1.Signal) {
	if action := in.GetAction(); action != enginev1.SignalAction_SIGNAL_ACTION_PUBLISH &&
		action != enginev1.SignalAction_SIGNAL_ACTION_UNSPECIFIED {
		s.HandleUpdateAsync(ctx, &enginev1.SignalUpdate{
			SignalId:  in.GetSignalId(),
			Action:    action,
			Reason:    alasanPenarikan(action),
			UpdatedAt: in.GetPublishedAt(),
		})
		return
	}

	signal := s.siapkanSinyal(ctx, in)
	if signal == nil {
		return
	}

	prioritas := PrioritasMenunggu
	if signal.Status == SignalReady {
		prioritas = PrioritasSiap
	}

	// Salinan dipakai pekerja antrean: payload gRPC tidak boleh dirujuk lagi
	// setelah callback kembali.
	salinan := *signal
	if !s.antrean.push(prioritas, func() { s.prosesSinyalMasuk(ctx, &salinan) }) {
		siap, tunggu, dibuang := s.antrean.stats()
		s.logger.Warn("antrean sinyal penuh, sinyal dilepas",
			"evt", "signal.antrean.penuh",
			"signal_id", signal.SignalID, "status", signal.Status,
			"siap", siap, "tunggu", tunggu, "dibuang", dibuang)
	}
}

// HandleUpdateAsync mengantrekan penarikan sinyal (CANCEL / EXPIRE / SUPERSEDE)
// lewat antrean prioritas yang sama dengan PUBLISH agar pembaruan tidak pernah
// melompati sinyal yang masih mengantre untuk ditulis ke database.
func (s *Service) HandleUpdateAsync(ctx context.Context, in *enginev1.SignalUpdate) {
	if in == nil || in.GetSignalId() == "" {
		return
	}
	// Catat pembatalan dini secara sinkron saat paket tiba dari stream supaya
	// pekerja paralel yang sedang mengambil PUBLISH sinyal yang sama langsung
	// menahan eksekusi meskipun jadwal goroutine mendahulukan pekerja PUBLISH.
	if status := ingest.UpdateStatus(in.GetAction()); status != "" {
		if _, err := s.store.FindSignal(ctx, in.GetSignalId()); err != nil {
			s.rememberEarlyCancel(in.GetSignalId(), status, in.GetReason(), in.GetSupersededBy())
		}
	}
	salinan := &enginev1.SignalUpdate{
		SignalId:     in.GetSignalId(),
		Action:       in.GetAction(),
		Reason:       in.GetReason(),
		SupersededBy: in.GetSupersededBy(),
		UpdatedAt:    in.GetUpdatedAt(),
	}
	if !s.antrean.push(PrioritasSiap, func() { s.HandleUpdate(ctx, salinan) }) {
		// Bila antrean penuh, jalankan langsung agar penarikan tidak pernah hilang.
		s.HandleUpdate(ctx, salinan)
	}
}

// siapkanSinyal mengubah payload engine menjadi sinyal yang bisa diproses.
//
// Mengembalikan nil bila sinyal bukan penerbitan (penarikan ditangani langsung)
// atau bila isinya tidak lengkap.
func (s *Service) siapkanSinyal(ctx context.Context, in *enginev1.Signal) *model.Signal {
	// Penarikan sinyal (CANCEL / EXPIRE / SUPERSEDE) tetap langsung: isinya
	// kecil dan harus segera terlihat di dashboard.
	if action := in.GetAction(); action != enginev1.SignalAction_SIGNAL_ACTION_PUBLISH &&
		action != enginev1.SignalAction_SIGNAL_ACTION_UNSPECIFIED {
		s.HandleUpdate(ctx, &enginev1.SignalUpdate{
			SignalId:  in.GetSignalId(),
			Action:    action,
			Reason:    alasanPenarikan(action),
			UpdatedAt: in.GetPublishedAt(),
		})
		return nil
	}

	signal, ok := ingest.ConvertSignal(in)
	if !ok {
		s.logger.Warn("sinyal engine tidak lengkap, diabaikan",
			"signal_id", in.GetSignalId())
		return nil
	}

	return signal
}

// prosesSinyalMasuk menyimpan sinyal lalu menyebarkannya ke akun.
func (s *Service) prosesSinyalMasuk(ctx context.Context, signal *model.Signal) {
	unlock := s.lockSignalOp(signal.SignalID)
	defer unlock()

	// Sinyal yang lahir saat pasar tutup tidak dicatat sama sekali.
	//
	// Kasus nyata 16 Nov 2026: tiga setup terbit pada Minggu malam NY dengan
	// masa berlaku 5-15 menit, lalu dinilai TP/SL memakai candle di luar sesi.
	// Baris seperti itu tidak bisa dieksekusi siapa pun dan hanya mengotori
	// jurnal, jadi dibuang di gerbang ini sebelum ada yang tersimpan.
	if !ExpectedOpen(signal.PublishedAt) {
		s.logger.Info("sinyal terbit saat pasar tutup, diabaikan",
			"evt", "signal.skip.closed",
			"signal_id", signal.SignalID,
			"symbol", signal.Symbol,
			"published_at", signal.PublishedAt)
		return
	}

	existing, findErr := s.store.FindSignal(ctx, signal.SignalID)
	if findErr == nil && IsTerminal(existing.Status) {
		// Sinyal terminal tidak boleh dihidupkan kembali oleh kiriman ulang.
		s.logger.Info("sinyal sudah final, dilewati",
			"evt", "signal.skip.final",
			"signal_id", signal.SignalID, "status", existing.Status)
		return
	}

	// Baris status radar memakai identitas tetap per slot, sedangkan UpsertSignal
	// sengaja tidak menimpa kolom status agar status lokal (ACKED) tidak hilang.
	// Tanpa pembaruan khusus ini status menunggu macet di nilai pertama
	// selamanya, sehingga radar tampak tidak pernah bergerak.
	if findErr == nil && model.IsWaitingSignalStatus(existing.Status) && existing.Status != signal.Status {
		if err := s.store.UpdateSignalStatus(ctx, signal.SignalID, signal.Status, ""); err != nil {
			s.logger.Warn("gagal memperbarui status menunggu",
				"evt", "signal.status.gagal",
				"signal_id", signal.SignalID, "error", err.Error())
		}
	}

	// Engine menerbitkan ulang setup yang sama dengan signal_id baru setiap siklus.
	// Bila sinyal dengan ID baru ini murni pengulangan setup yang masih hidup (dan
	// tidak ada posisi terbuka yang perlu menerima log penolakan), tahan sebelum
	// UpsertSignal agar tabel signal tidak dikotori baris kembar.
	if signal.Status == SignalReady && findErr != nil {
		if asal, ok := s.penerbitanUlang(ctx, signal); ok {
			s.ackDuplikat(ctx, signal)
			s.logger.Info("penerbitan ulang setup ditahan",
				"evt", "signal.duplikat.ditahan",
				"signal_id", signal.SignalID,
				"signal_asal", asal.SignalID,
				"symbol", signal.Symbol,
				"htf", signal.Htf, "ltf", signal.Ltf,
				"status_asal", asal.Status)
			return
		}
	}

	if err := s.store.UpsertSignal(ctx, signal); err != nil {
		s.logger.Error("gagal menyimpan sinyal",
			"evt", "signal.simpan.gagal",
			"signal_id", signal.SignalID, "error", err.Error())
		return
	}

	// Bila CANCEL/EXPIRE/SUPERSEDE sempat tiba lebih dulu sebelum PUBLISH ini
	// tersimpan ke database, terapkan penarikan itu sekarang dan batalkan eksekusi.
	if early, canceled := s.takeEarlyCancel(signal.SignalID); canceled {
		if early.status == SignalSuperseded {
			_ = s.store.MarkSuperseded(ctx, signal.SignalID, early.superseded, early.reason)
		} else {
			_ = s.store.UpdateSignalStatus(ctx, signal.SignalID, early.status, early.reason)
		}
		s.logger.Info("sinyal sudah dibatalkan lebih dulu sebelum publish selesai, eksekusi ditahan",
			"evt", "signal.batal.dini",
			"signal_id", signal.SignalID, "status", early.status)
		return
	}

	s.logger.Info("sinyal diterima",
		"evt", "signal.terima",
		"signal_id", signal.SignalID,
		"symbol", signal.Symbol,
		"status", signal.Status,
		"strategy", signal.StrategyVersion,
		"rr", signal.RiskReward)

	// Sinyal status menengah hanya untuk dashboard; tidak ada ACK.
	if signal.Status != SignalReady {
		s.fanOutIntermediate(ctx, signal)
		return
	}

	// Engine menerbitkan ulang setup yang sama dengan signal_id baru setiap
	// siklus. Penahanan hanya berlaku bila tidak ada posisi terbuka pada setup
	// itu: saat posisi sudah terbuka, penolakan "posisi masih berjalan" beserta
	// notifikasinya tetap harus lahir agar user tahu setup sejenis muncul.
	if asal, ok := s.penerbitanUlang(ctx, signal); ok {
		s.ackDuplikat(ctx, signal)
		s.logger.Info("penerbitan ulang setup ditahan",
			"evt", "signal.duplikat.ditahan",
			"signal_id", signal.SignalID,
			"signal_asal", asal.SignalID,
			"symbol", signal.Symbol,
			"htf", signal.Htf, "ltf", signal.Ltf,
			"status_asal", asal.Status)
		return
	}

	if err := s.ackSignal(ctx, signal.StrategyVersion, signal.SignalID); err != nil {
		// ACK gagal bukan alasan membuang sinyal: engine boleh mengulang.
		s.logger.Warn("ACK sinyal gagal", "evt", "signal.ack.gagal",
			"signal_id", signal.SignalID, "error", err.Error())
	}
	s.fanOutReady(ctx, signal)
}

// penerbitanUlang melaporkan apakah sinyal ini penerbitan ulang setup yang
// masih hidup, beserta sinyal asalnya.
//
// Yang dianggap penerbitan ulang: entry, SL, dan TP sama persis dengan sinyal
// lain yang statusnya belum final. Bila setup lama sudah berakhir (EXPIRED,
// CANCELLED, SUPERSEDED), terbitan baru diloloskan karena struktur harganya
// bisa masih valid setelah jeda.
//
// Posisi terbuka pada setup yang sama tidak ikut ditahan: penolakan "posisi
// masih berjalan" tetap harus lahir sebagai catatan dan notifikasi.
//
// Masa berlaku yang lebih panjang dari keduanya dipertahankan agar setup tidak
// mati lebih cepat hanya karena terbitan pengulang membawa masa berlaku pendek.
func (s *Service) penerbitanUlang(ctx context.Context, signal *model.Signal) (*model.Signal, bool) {
	if signal == nil || signal.EntryPrice <= 0 || signal.StopLoss <= 0 || signal.TakeProfit <= 0 {
		return nil, false
	}
	asal, err := s.store.FindLiveDuplicate(ctx, signal)
	if err != nil {
		// Termasuk "tidak ada duplikat": sinyal diteruskan seperti biasa.
		return nil, false
	}
	users, err := s.store.ListUsersBySymbol(ctx, signal.Symbol)
	if err != nil {
		// Tanpa daftar pemantau, pengecekan posisi terbuka tidak bisa dilakukan;
		// sinyal diteruskan agar tidak ada pemantau yang kehilangan kabar.
		return nil, false
	}
	for _, user := range users {
		if _, posErr := s.store.FindOpenPositionBySignal(ctx, user.ID, asal.SignalID); posErr == nil {
			return nil, false
		}
	}
	if signal.ExpiresAt.After(asal.ExpiresAt) {
		if err := s.store.PerpanjangMasaBerlaku(ctx, asal.SignalID, signal.ExpiresAt); err != nil {
			s.logger.Warn("gagal memperpanjang masa berlaku sinyal",
				"evt", "signal.duplikat.perpanjang.gagal",
				"signal_id", asal.SignalID, "error", err.Error())
		}
	}
	return asal, true
}

// ackDuplikat memberi tahu engine bahwa penerbitan ulang sudah diterima, supaya
// pengiriman berikutnya tidak diulang terus-menerus.
func (s *Service) ackDuplikat(ctx context.Context, signal *model.Signal) {
	if err := s.ackSignal(ctx, signal.StrategyVersion, signal.SignalID); err != nil {
		s.logger.Warn("ACK sinyal duplikat gagal", "evt", "signal.ack.gagal",
			"signal_id", signal.SignalID, "error", err.Error())
	}
}

// fanOutIntermediate menyebarkan status menengah ke pemantau simbol.
func (s *Service) fanOutIntermediate(ctx context.Context, signal *model.Signal) {
	users, err := s.store.ListUsersBySymbol(ctx, signal.Symbol)
	if err != nil {
		s.logger.Error("gagal membaca pemantau simbol", "error", err.Error())
		return
	}
	for _, user := range users {
		if !s.userWantsSignal(ctx, user.ID, signal) {
			continue
		}
		// Kemajuan radar datang berkali-kali untuk slot yang sama. Baris user yang
		// sudah punya posisi atau sudah berstatus final tidak boleh disentuh lagi:
		// menimpanya akan mengembalikan sinyal yang sudah dieksekusi ke keadaan
		// menunggu, sehingga radar meminta entry ulang padahal posisinya terbuka.
		if _, posErr := s.store.FindOpenPositionBySignal(ctx, user.ID, signal.SignalID); posErr == nil {
			continue
		}
		existing, findErr := s.store.FindUserSignal(ctx, user.ID, signal.SignalID)
		if findErr != nil && !errors.Is(findErr, apperror.ErrNotFound) {
			continue
		}
		if existing != nil && IsTerminal(existing.Status) {
			continue
		}
		// Kiriman dengan status yang sama tidak dicatat ulang dan tidak
		// dinotifikasi lagi; hanya perpindahan tahap, termasuk mundur ke tahap
		// sebelumnya, yang layak dikabari.
		if !s.recordUserSignal(ctx, user.ID, signal.SignalID, signal.Status, "", nil) {
			continue
		}
		s.pub.PublishSignal(ctx, user.ID, signal.SignalID, signal.Status, "")
	}
}

// userWantsSignal menentukan apakah user menginginkan notifikasi atau pencatatan
// untuk sinyal ini berdasarkan hak tier, saklar metode per akun, dan saklar timeframe.
func (s *Service) userWantsSignal(ctx context.Context, userID string, signal *model.Signal) bool {
	user, err := s.store.FindUserByID(ctx, userID)
	if err != nil || user == nil {
		return false
	}

	now := s.opts.Clock()
	tier := EffectiveTier(user.Tier, user.TrialExpiresAt, user.TierExpiresAt, now)

	// User berhak atas strategi ini?
	if !StrategyAllowed(tier, signal.StrategyVersion) {
		return false
	}
	if signal.StrategyVersion == "ila-3.0" && !s.opts.Settings.StrategySelectable("ila-3.0") {
		return false
	}

	accounts, err := s.store.ListBrokerAccountsByUser(ctx, userID)
	if err != nil || len(accounts) == 0 {
		// Pengguna tanpa akun broker tetap menerima sinyal sesuai langganan simbolnya.
		return true
	}

	// Jika pengguna memiliki akun broker, periksa apakah ada minimal satu akun aktif
	// yang mengizinkan metode dan memantau timeframe sinyal ini.
	for _, acc := range accounts {
		if !acc.IsActive || acc.IsFrozen {
			continue
		}
		if !s.akunMengizinkanMetode(ctx, acc.ID, signal.Symbol, signal.StrategyVersion) {
			continue
		}
		tindakan := s.putuskanTimeframe(ctx, acc.ID, signal.Symbol, signal.StrategyVersion, signal.Htf, signal.Ltf)
		if tindakan != TimeframeAbaikan {
			return true
		}
	}

	return false
}

// FanOutConcurrency membatasi berapa akun diproses bersamaan untuk satu sinyal.
//
// Sebelumnya akun diproses satu per satu. Setiap akun memanggil broker beberapa
// kali (cek akun, baca kutipan, hitung lot, kirim order) sehingga akun terakhir
// menunggu seluruh antrean di depannya. Data produksi menunjukkan sinyal siap
// entry tiba 20 detik setelah candle tutup - terlalu lama untuk setup M5.
//
// Batas 8 dipilih agar tetap jauh di bawah batas koneksi broker dan memori VPS
// kecil, sementara akun-akun pertama tidak lagi menunggu yang terakhir.
const FanOutConcurrency = 8

func (s *Service) fanOutReady(ctx context.Context, signal *model.Signal) {
	// Sinyal disebar ke setiap akun yang mencentang simbol ini. Satu user dapat
	// menyumbang lebih dari satu akun karena pilihan pair kini per akun.
	pairs, err := s.store.ListAccountsBySymbol(ctx, signal.Symbol)
	if err == nil {
		mulai := s.opts.Clock()
		s.prosesParalel(ctx, signal, pairs)
		s.logger.Info("penyebaran sinyal selesai",
			"evt", "signal.fanout",
			"signal_id", signal.SignalID,
			"symbol", signal.Symbol,
			"akun", len(pairs),
			"durasi_ms", s.opts.Clock().Sub(mulai).Milliseconds())
		return
	}

	// Jalur cadangan: bila kueri per akun gagal, pakai jalur per user yang lama
	// agar sinyal tetap tersebar dan tidak hilang saat basis data bermasalah.
	s.logger.Warn("membaca akun per simbol gagal; memakai jalur per user", "error", err.Error())
	users, userErr := s.store.ListUsersBySymbol(ctx, signal.Symbol)
	if userErr != nil {
		s.logger.Error("gagal membaca pemantau simbol", "error", userErr.Error())
		return
	}
	for _, user := range users {
		if procErr := s.processSignalForUser(ctx, &user, signal); procErr != nil {
			s.logger.Error("gagal memproses sinyal untuk user",
				"user_id", user.ID, "signal_id", signal.SignalID, "error", procErr.Error())
		}
	}
}

// prosesParalel menjalankan pemrosesan sinyal untuk beberapa akun sekaligus.
//
// Kendali yang dijaga:
//   - Batas FanOutConcurrency, supaya lonjakan akun tidak membuka puluhan
//     koneksi broker sekaligus.
//   - Kegagalan satu akun tidak menghentikan akun lain: trading user lain tidak
//     boleh ikut berhenti karena satu kredensial bermasalah.
//   - Pembatalan mengikuti ctx, sehingga saat Runner berhenti tidak ada
//     goroutine yang tertinggal menahan shutdown.
func (s *Service) prosesParalel(ctx context.Context, signal *model.Signal, pairs []model.SubscriptionAccount) {
	if len(pairs) == 0 {
		return
	}

	batas := FanOutConcurrency
	if len(pairs) < batas {
		batas = len(pairs)
	}

	var wg sync.WaitGroup
	antre := make(chan struct{}, batas)

	for i := range pairs {
		user := pairs[i].User
		account := pairs[i].Account

		antre <- struct{}{}
		wg.Add(1)

		go func(u *model.User, a *model.BrokerAccount) {
			defer wg.Done()
			defer func() { <-antre }()

			if ctx.Err() != nil {
				return
			}
			if procErr := s.processSignalForAccount(ctx, u, a, signal); procErr != nil {
				s.logger.Error("gagal memproses sinyal untuk akun",
					"user_id", u.ID, "account_id", a.ID,
					"signal_id", signal.SignalID, "error", procErr.Error())
			}
		}(&user, &account)
	}

	wg.Wait()
}

// processSignalForAccount menjalankan urutan pemeriksaan untuk SATU akun.
//
// Ini jalur utama fan-out: setiap akun yang mencentang simbol menerima
// evaluasinya sendiri, tanpa terpengaruh akun lain milik user yang sama.
func (s *Service) processSignalForAccount(ctx context.Context, user *model.User, account *model.BrokerAccount, signal *model.Signal) error {
	now := s.opts.Clock()
	tier := EffectiveTier(user.Tier, user.TrialExpiresAt, user.TierExpiresAt, now)

	// [1] Sinyal sah?
	if reason := s.validateSignal(ctx, signal, now); reason != RejectNone {
		return s.rejectAccount(ctx, user.ID, account.ID, signal.SignalID, reason)
	}

	// [2] User berhak atas strategi ini?
	if !StrategyAllowed(tier, signal.StrategyVersion) {
		return s.rejectAccount(ctx, user.ID, account.ID, signal.SignalID, RejectStrategyTier)
	}
	if signal.StrategyVersion == "ila-3.0" && !s.opts.Settings.StrategySelectable("ila-3.0") {
		return s.rejectAccount(ctx, user.ID, account.ID, signal.SignalID, RejectStrategyTier)
	}
	// [2b] Akun ini mematikan metode tersebut untuk pair ini?
	if !s.akunMengizinkanMetode(ctx, account.ID, signal.Symbol, signal.StrategyVersion) {
		return s.rejectAccount(ctx, user.ID, account.ID, signal.SignalID, RejectStrategyDisabled)
	}

	// [2c] Pasangan timeframe: dieksekusi otomatis, hanya ditampilkan sebagai
	// peluang, atau tidak relevan bagi akun ini.
	//
	// "Peluang" sengaja BUKAN penolakan: user tetap menerima alert dan bisa
	// memutuskan sendiri lewat tombol eksekusi.
	tindakanTimeframe := s.putuskanTimeframe(ctx, account.ID, signal.Symbol,
		signal.StrategyVersion, signal.Htf, signal.Ltf)
	if tindakanTimeframe == TimeframeAbaikan {
		return nil
	}

	// [3] Akun ini sendiri boleh menerima order?
	if !account.IsActive || account.IsFrozen {
		return s.rejectAccount(ctx, user.ID, account.ID, signal.SignalID, RejectAccountFrozen)
	}

	// [3b] Kunci akun selama eksekusi berlangsung.
	//
	// Pagar "satu posisi terbuka" pada evaluate hanya melihat posisi yang SUDAH
	// tercatat. Dua sinyal berbeda untuk akun yang sama — misalnya dua pasangan
	// timeframe pada pair yang sama — bisa melewati pemeriksaan itu bersamaan
	// lalu mengeksekusi keduanya. Kunci ini menutup celah tersebut.
	//
	// Kunci hanya dipakai pada mode yang benar-benar mengeksekusi; mode manual
	// hanya mencatat kandidat sehingga tidak perlu dikunci.
	otomatis := account.Mode == "auto" && s.autoExecute(ctx) && tindakanTimeframe != TimeframePeluang
	if otomatis {
		release, dikunci, lockErr := s.lockAccount(ctx, account.ID, signal.SignalID, now)
		if lockErr != nil {
			return lockErr
		}
		if !dikunci {
			// Akun sedang mengeksekusi sinyal lain. Ini bukan kegagalan: sinyal
			// ini memang tidak boleh membuka posisi kedua.
			s.logger.Info("akun sedang mengeksekusi sinyal lain",
				"user_id", user.ID, "account_id", account.ID, "signal_id", signal.SignalID)
			return s.rejectAccount(ctx, user.ID, account.ID, signal.SignalID, RejectOpenPosition)
		}
		defer release()
	}

	reason, evalErr := s.evaluate(ctx, user, account, signal, tier, now)
	if evalErr != nil {
		return evalErr
	}
	if reason != RejectNone {
		s.logger.Info("sinyal tertahan",
			"user_id", user.ID, "signal_id", signal.SignalID,
			"account_id", account.ID, "reason", string(reason))
		return s.rejectAccount(ctx, user.ID, account.ID, signal.SignalID, reason)
	}

	if otomatis {
		return s.execute(ctx, user, account, signal, true)
	}

	// Peluang: pasangan timeframe ini tidak diizinkan auto, tetapi user tetap
	// diberi tahu. Dicatat dengan status tersendiri agar tidak tertukar dengan
	// penolakan, dan agar bisa dieksekusi manual lewat tombol konfirmasi.
	if tindakanTimeframe == TimeframePeluang {
		s.recordUserSignal(ctx, user.ID, signal.SignalID, model.SignalStatusOpportunity,
			pesanPeluang(signal), &account.ID)
		s.pub.PublishSignal(ctx, user.ID, signal.SignalID, model.SignalStatusOpportunity, pesanPeluang(signal))
		return nil
	}

	// Mode manual: catat siap dengan akun ini sebagai kandidat konfirmasi.
	s.recordUserSignal(ctx, user.ID, signal.SignalID, SignalReady, "", &account.ID)
	s.pub.PublishSignal(ctx, user.ID, signal.SignalID, SignalReady, "")
	return nil
}

// pesanPeluang menjelaskan mengapa sebuah setup tidak dieksekusi otomatis.
//
// Pesan ini dibaca user, jadi menyebutkan pasangan timeframe-nya dan langkah
// berikutnya, bukan sekadar "tidak dieksekusi".
func pesanPeluang(signal *model.Signal) string {
	pasangan := signal.Htf
	if signal.Ltf != "" {
		pasangan = signal.Htf + "-" + signal.Ltf
	}
	if pasangan == "" {
		return "Setup tersedia untuk dipantau. Eksekusi otomatis tidak aktif untuk akun ini; buka dashboard untuk memutuskan."
	}
	return "Setup pasangan " + pasangan + " tersedia untuk dipantau. Eksekusi otomatis tidak aktif untuk akun ini; buka dashboard untuk memutuskan."
}

// processSignalForUser menjalankan seluruh urutan pemeriksaan plans/14 §5.2
// lalu memutuskan: tahan, tunggu konfirmasi, atau eksekusi.
func (s *Service) processSignalForUser(ctx context.Context, user *model.User, signal *model.Signal) error {
	now := s.opts.Clock()

	tier := EffectiveTier(user.Tier, user.TrialExpiresAt, user.TierExpiresAt, now)

	// [1] Sinyal sah?
	if reason := s.validateSignal(ctx, signal, now); reason != RejectNone {
		return s.reject(ctx, user.ID, signal.SignalID, reason)
	}

	// [2] User berhak atas strategi ini?
	if !StrategyAllowed(tier, signal.StrategyVersion) {
		return s.reject(ctx, user.ID, signal.SignalID, RejectStrategyTier)
	}
	// Strategi yang belum dirilis tidak boleh menghasilkan posisi.
	if signal.StrategyVersion == "ila-3.0" && !s.opts.Settings.StrategySelectable("ila-3.0") {
		return s.reject(ctx, user.ID, signal.SignalID, RejectStrategyTier)
	}

	accounts, err := s.executableAccounts(ctx, user, tier, now)
	if err != nil {
		return err
	}
	if len(accounts) == 0 {
		// Tidak ada akun yang boleh mengeksekusi: tetap tampil di dashboard
		// tanpa order. Untuk Free ini normal, bukan kegagalan.
		s.recordUserSignal(ctx, user.ID, signal.SignalID, SignalReady, "", nil)
		s.pub.PublishSignal(ctx, user.ID, signal.SignalID, SignalReady, string(RejectEntitlement))
		return nil
	}

	// Satu sinyal hanya menjadi satu order. Akun pertama yang memenuhi syarat
	// dipakai, sisanya menunggu sinyal berikutnya.
	for _, account := range accounts {
		reason, evalErr := s.evaluate(ctx, user, &account, signal, tier, now)
		if evalErr != nil {
			s.logger.Error("evaluasi akun gagal",
				"user_id", user.ID, "account_id", account.ID, "error", evalErr.Error())
			continue
		}
		if reason != RejectNone {
			s.logger.Info("sinyal tertahan",
				"user_id", user.ID, "signal_id", signal.SignalID,
				"account_id", account.ID, "reason", string(reason))
			return s.reject(ctx, user.ID, signal.SignalID, reason)
		}

		// Semua syarat lolos.
		//
		// Mode auto mengirim order tanpa konfirmasi. Untuk akun dry-run,
		// "order" itu tetap dicatat sebagai posisi simulasi sehingga aturan
		// harian dan riwayat tetap berjalan persis seperti akun nyata.
		if account.Mode == "auto" && s.autoExecute(ctx) {
			return s.execute(ctx, user, &account, signal, true)
		}

		// Mode manual: tampilkan sebagai siap, tunggu konfirmasi user.
		s.recordUserSignal(ctx, user.ID, signal.SignalID, SignalReady, "", &account.ID)
		s.pub.PublishSignal(ctx, user.ID, signal.SignalID, SignalReady, "")
		return nil
	}

	// Semua akun gagal dievaluasi karena masalah teknis, bukan aturan trading.
	return s.reject(ctx, user.ID, signal.SignalID, RejectAccountFrozen)
}

// validateSignal memeriksa kelengkapan dan masa berlaku sinyal.
func (s *Service) validateSignal(ctx context.Context, signal *model.Signal, now time.Time) RejectReason {
	return s.validateSignalMode(ctx, signal, now, false)
}

// MaxManualTTLFactor adalah kelipatan masa berlaku yang masih boleh dipakai
// untuk eksekusi manual.
//
// Sinyal M5 berlaku 5 menit; dengan faktor 4 user masih punya 20 menit untuk
// memutuskan, asalkan harga masih di dalam range entry. Angka ini menjaga dua
// hal sekaligus: setup yang sudah basi tidak dieksekusi, tetapi keputusan user
// tidak dibunuh oleh jam dinding ketika harganya belum bergerak.
const MaxManualTTLFactor = 4

// validateSignalMode menjalankan pemeriksaan kelayakan sinyal.
//
// manual=false (eksekusi otomatis) memakai aturan ketat: begitu masa berlaku
// habis, sinyal tidak boleh dieksekusi lagi karena order otomatis harus
// mengikuti harga saat setup terbit.
//
// manual=true (tombol konfirmasi user) memperlakukan masa berlaku sebagai
// petunjuk, bukan dinding: sinyal masih boleh dieksekusi selama belum melewati
// batas keras MaxManualTTLFactor kali masa berlakunya. Kelayakan harga tetap
// diperiksa terpisah oleh gerbang slippage, sehingga setup yang harganya sudah
// jauh tetap ditolak - dan pesan yang dilihat user menyebut harga, bukan jam.
func (s *Service) validateSignalMode(ctx context.Context, signal *model.Signal, now time.Time, manual bool) RejectReason {
	// Mode platform diperiksa paling awal: saat pemeliharaan, sinyal baru
	// dilewati (bukan ditunda) karena harga sudah berubah. Posisi berjalan
	// tidak pernah disentuh di sini (plans/18 §4.3).
	switch s.maintenanceMode(ctx) {
	case ModeMaintenance:
		return RejectMaintenance
	case ModeReadOnly:
		return RejectReadOnly
	}
	if signal.EntryPrice <= 0 || signal.StopLoss <= 0 || signal.TakeProfit <= 0 {
		return RejectBadSignal
	}
	if !now.Before(signal.ExpiresAt) {
		// Sinyal yang belum pernah siap bukan "gagal dieksekusi": engine hanya
		// mengirim ulang status menunggu pada slot yang sama, sehingga barisnya
		// memang basi. Bedakan agar notifikasi tidak berbunyi "terlambat" pada
		// setup yang belum pernah menawarkan entry.
		if menungguStatus(signal.Status) {
			return RejectWaitingNext
		}
		// Eksekusi manual masih boleh selama belum melewati batas keras.
		// Sinyal yang sudah tidak menawarkan entry (dibatalkan, digantikan,
		// sudah dipakai) tetap ditolak di tempat lain, bukan di sini.
		if !manual || !manualStillAllowed(signal, now) {
			return RejectExpired
		}
	}
	if s.marketOpenRequired(ctx) && !s.marketOpenAt(now) {
		return RejectMarketClosed
	}
	return RejectNone
}

// manualStillAllowed menandai sinyal yang sudah lewat masa berlaku tetapi masih
// boleh dieksekusi manual karena belum melewati batas keras.
//
// Sinyal tanpa PublishedAt (nol) selalu ditolak: selisihnya menjadi durasi
// raksasa dan akan membuka eksekusi tanpa batas nyata.
func manualStillAllowed(signal *model.Signal, now time.Time) bool {
	if signal.PublishedAt.IsZero() || signal.ExpiresAt.IsZero() {
		return false
	}
	ttl := signal.ExpiresAt.Sub(signal.PublishedAt)
	if ttl <= 0 {
		// Baris tanpa rentang waktu yang masuk akal tidak diberi perpanjangan:
		// menebak batas lebih berbahaya daripada menolak.
		return false
	}
	return now.Before(signal.ExpiresAt.Add(ttl * (MaxManualTTLFactor - 1)))
}

// menungguStatus menandai sinyal yang masih dalam tahap menunggu konfirmasi
// (belum pernah menawarkan entry). Dipakai untuk membedakan "setup kedaluwarsa"
// dari "setup masih menunggu tahap berikutnya".
func menungguStatus(status string) bool {
	return strings.HasPrefix(status, "WAIT_")
}

// executableAccounts menyaring akun yang boleh menerima order baru.
func (s *Service) executableAccounts(ctx context.Context, user *model.User, tier string, now time.Time) ([]model.BrokerAccount, error) {
	accounts, err := s.store.ListBrokerAccountsByUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	limit := MaxAccounts(tier)
	if limit > maxAccountsPerUser {
		limit = maxAccountsPerUser
	}

	// Dua entri yang menunjuk akun broker sama akan mengeksekusi sinyal yang
	// sama dua kali. Entri kembar dibuang di sini (plans/12 §3.4).
	seen := make(map[string]struct{}, limit)

	out := make([]model.BrokerAccount, 0, limit)
	for _, account := range accounts {
		if !account.IsActive || account.IsFrozen {
			continue
		}
		if key := AccountIdentity(account); key != "" {
			if _, dup := seen[key]; dup {
				s.logger.Warn("akun broker kembar diabaikan",
					"user_id", user.ID, "account_id", account.ID, "identity", key)
				continue
			}
			seen[key] = struct{}{}
		}
		if len(out) >= limit {
			break
		}
		out = append(out, account)
	}
	return out, nil
}

// AccountIdentity mengembalikan penanda unik akun di sisi broker.
// Akun tanpa penanda tidak pernah digabung: lebih baik dianggap berbeda.
func AccountIdentity(account model.BrokerAccount) string {
	if account.ExternalAccountID == "" {
		return ""
	}
	return account.BrokerType + ":" + account.ExternalAccountID
}

// evaluate menjalankan pemeriksaan 3-5 (plans/14 §5.2) pada satu akun.
func (s *Service) evaluate(ctx context.Context, user *model.User, account *model.BrokerAccount, signal *model.Signal, tier string, now time.Time) (RejectReason, error) {
	// Ban berlaku seketika: akun ter-ban tidak boleh dieksekusi sama sekali,
	// walau sesinya masih hidup di perangkat user (plans/19 C3).
	if user.Status == authstate.StatusBanned {
		return RejectAccountBanned, nil
	}

	// Akun yang sudah dihapus tidak pernah boleh trading. Sisa subscription
	// atau akun broker yang tertinggal tidak boleh menghidupkan kembali
	// eksekusi atas nama akun yang sudah tidak ada (plans/19 C3).
	if user.Status == model.UserStatusDeleted || user.DeletedAtUser != nil {
		return RejectAccountBanned, nil
	}

	// Dokumen wajib harus disetujui sebelum eksekusi. Posisi yang sudah
	// berjalan tidak disentuh: yang diblokir hanya entry baru (plans/09).
	if s.opts.LegalGate != nil && s.legalGateActive(ctx) && s.opts.LegalGate(ctx, user.ID) {
		return RejectLegalPending, nil
	}

	// Penurunan tier tidak menyentuh posisi hidup, tetapi mematikan mode auto.
	if !ModeAllowed(tier, account.Mode) {
		return RejectEntitlement, nil
	}

	// Masa tenggang dan suspend: hanya entry baru yang diblokir.
	if WithinGrace(user.GraceUntil, now) || user.SuspendedAt != nil {
		return RejectSuspended, nil
	}

	// [3] Broker harus bisa dihubungi; dry-run tidak butuh koneksi.
	//
	// Snapshot ini juga dipakai calculateLot saat menghitung ukuran posisi,
	// sehingga satu eksekusi hanya sekali membaca akun ke broker.
	if !s.mustDryRun(ctx, &userSession{User: user}, account) {
		if _, err := s.accountSnapshot(ctx, account); err != nil {
			s.markBrokerDown(ctx, account, err)
			return RejectAccountFrozen, nil
		}
	}

	// [4] Aturan harian milik Runner.
	day := DayKey(now)
	summary, err := s.store.GetDailySummary(ctx, user.ID, day)
	if err != nil {
		return RejectNone, err
	}

	openPositions, err := s.store.ListOpenPositionsByUser(ctx, user.ID)
	if err != nil {
		return RejectNone, err
	}

	// Posisi terbuka dihitung per akun: satu akun tidak boleh membuka posisi baru
	// bila akun itu sendiri masih memiliki posisi terbuka. Akun lain milik user
	// yang sama (pada tier Plus) berhak mengeksekusi sinyal secara independen.
	openCount := 0
	for _, p := range openPositions {
		if p.BrokerAccountID == "" || p.BrokerAccountID == account.ID {
			openCount++
		}
	}

	daily := DailyStatus{
		Day:         summary.Day,
		WinCount:    summary.WinCount,
		LossCount:   summary.LossCount,
		TotalTrades: summary.TotalTrades,
		OpenCount:   openCount,
	}
	if reason := EvaluateDaily(daily, s.policy(ctx)); reason != RejectNone {
		return reason, nil
	}

	// [5] Slippage guard, kecuali simulasi.
	if s.opts.SlippageGuard && !s.mustDryRun(ctx, &userSession{User: user}, account) {
		if reason := s.checkSlippage(ctx, account, signal); reason != RejectNone {
			return reason, nil
		}
	}

	return RejectNone, nil
}

// checkSlippage menolak eksekusi bila harga sudah menjauh dari level entry.
func (s *Service) checkSlippage(ctx context.Context, account *model.BrokerAccount, signal *model.Signal) RejectReason {
	client, err := s.client(ctx, account)
	if err != nil {
		return RejectAccountFrozen
	}

	price, spread, err := s.quoteForSlippage(ctx, client, signal.Symbol)
	if err != nil || price <= 0 {
		// Harga tidak terbaca: menolak lebih aman daripada menebak.
		s.logger.Warn("harga acuan tidak tersedia",
			"symbol", signal.Symbol, "error", errText(err))
		return RejectSlippage
	}

	decision := EffectiveSlippageLimit(signal.Symbol, signal.EntryPrice, signal.StopLoss,
		LimitOptions{Spread: spread})
	decision.Drift = math.Abs(price - signal.EntryPrice)

	if !decision.OK() {
		s.logger.Warn("slippage melewati batas",
			"evt", "slippage.tolak",
			"symbol", signal.Symbol, "signal_id", signal.SignalID,
			"drift", decision.Drift, "limit", decision.Limit,
			"basis", decision.Basis, "spread", decision.Spread,
			"sl_distance", decision.SLDistance)
		return RejectSlippage
	}

	// Keputusan dibuat per pair dan per akun, sehingga sumber batas yang
	// berlaku dicatat: tanpa ini, sulit menjelaskan mengapa satu akun lolos
	// sementara akun lain ditolak pada sinyal yang sama.
	s.logger.Info("slippage lolos",
		"evt", "slippage.lolos",
		"symbol", signal.Symbol, "signal_id", signal.SignalID,
		"drift", decision.Drift, "limit", decision.Limit,
		"basis", decision.Basis, "spread", decision.Spread,
		"sl_distance", decision.SLDistance)
	return RejectNone
}

// quoteForSlippage membaca harga acuan beserta spread-nya.
//
// Spread dipakai agar toleransi menyesuaikan akun: akun dengan spread lebar
// wajar bergerak lebih jauh dari level entry, sehingga batas tetap wajar tanpa
// harus dilonggarkan untuk semua orang. Adapter yang belum melaporkan bid/ask
// mengembalikan spread nol, dan perhitungan jatuh ke tabel pair + persentase SL
// persis seperti sebelumnya.
func (s *Service) quoteForSlippage(ctx context.Context, client broker.Client, symbol string) (float64, float64, error) {
	if quoted, ok := client.(broker.QuoteSource); ok {
		q, err := quoted.Quote(ctx, symbol)
		if err != nil {
			return 0, 0, err
		}
		if q.Mid() > 0 {
			return q.Mid(), q.Spread(), nil
		}
	}

	price, err := client.CurrentPrice(ctx, symbol)
	if err != nil {
		return 0, 0, err
	}
	return price, 0, nil
}

// ---------------------------------------------------------------------------
// Eksekusi
// ---------------------------------------------------------------------------

// execute membuka posisi nyata (atau simulasi) untuk satu sinyal dan satu akun.
func (s *Service) execute(ctx context.Context, user *model.User, account *model.BrokerAccount, signal *model.Signal, isAuto bool) error {
	key := IdempotencyKey(signal.SignalID, account.ID)

	record := &model.ExecutionRecord{
		UserID:          user.ID,
		SignalID:        signal.SignalID,
		BrokerAccountID: account.ID,
		IdempotencyKey:  key,
		Status:          "CLAIMED",
		CreatedAt:       s.opts.Clock(),
	}

	claimed, err := s.store.ClaimExecution(ctx, record)
	if err != nil {
		return err
	}
	if !claimed {
		// Kunci sudah ada. Bila klaim itu tertinggal karena proses berhenti
		// sebelum order terkirim, ambil alih agar sinyal tidak hilang diam-diam.
		retaken, err := s.store.ReclaimStaleExecution(ctx, key, s.opts.Clock().Add(-ClaimStaleAfter))
		if err == nil && retaken {
			s.logger.Warn("klaim eksekusi kedaluwarsa diambil alih",
				"signal_id", signal.SignalID, "account_id", account.ID)
			record.ID = ""
			if claimed, err = s.store.ClaimExecution(ctx, record); err != nil {
				return err
			}
			if !claimed {
				return s.reject(ctx, user.ID, signal.SignalID, RejectDuplicate)
			}
		} else {
			// Sinyal ini pernah diproses untuk akun ini.
			s.logger.Warn("eksekusi ganda dicegah",
				"signal_id", signal.SignalID, "account_id", account.ID)
			return s.reject(ctx, user.ID, signal.SignalID, RejectDuplicate)
		}
	}

	dryRun := s.mustDryRun(ctx, &userSession{User: user}, account)
	now := s.opts.Clock()

	position := &model.Position{
		UserID:          user.ID,
		BrokerAccountID: account.ID,
		SourceSignalID:  signal.SignalID,
		Symbol:          signal.Symbol,
		StrategyVersion: signal.StrategyVersion,
		Direction:       signal.Direction,
		StopLoss:        signal.StopLoss,
		TakeProfit:      signal.TakeProfit,
		IsDryRun:        dryRun,
		Status:          PositionOpen,
		OpenedAt:        now,
	}

	// Langkah 4: hitung lot dari equity live (plans/05 §6.2).
	calc, err := s.calculateLot(ctx, user, account, signal, dryRun)
	if err != nil {
		s.failExecution(ctx, key, err)
		if errors.Is(err, ErrNoLot) {
			return s.reject(ctx, user.ID, signal.SignalID, RejectLotTooSmall)
		}
		return s.reject(ctx, user.ID, signal.SignalID, RejectAccountFrozen)
	}
	position.LotSize = calc.Lots
	position.RiskPercent = calc.RiskPercent
	position.OpenPrice = signal.EntryPrice

	if dryRun {
		position.TicketNumber = "SIM-" + key
		position.Status = PositionOpen
	} else {
		client, clientErr := s.client(ctx, account)
		if clientErr != nil {
			s.failExecution(ctx, key, clientErr)
			return s.reject(ctx, user.ID, signal.SignalID, RejectAccountFrozen)
		}

		// Langkah 5-6: kirim order dengan SL/TP server-side.
		result, openErr := client.OpenPosition(ctx, broker.OpenRequest{
			Symbol:     signal.Symbol,
			Side:       signal.Direction,
			Lots:       calc.Lots,
			StopLoss:   signal.StopLoss,
			TakeProfit: signal.TakeProfit,
			Comment:    signal.SignalID,
			ClientID:   key,
		})
		if openErr != nil {
			s.failExecution(ctx, key, openErr)
			s.logger.Error("order ditolak broker",
				"signal_id", signal.SignalID, "account_id", account.ID,
				"error", openErr.Error())

			// Timeout tidak berarti order tidak masuk. Periksa dulu: tanpa ini,
			// sinyal berikutnya bisa membuka posisi kedua di akun yang sama.
			if recovered, recErr := s.findOpenOrder(ctx, client, signal.SignalID); recErr == nil && recovered != nil {
				position.TicketNumber = recovered.Ticket
				if recovered.OpenPrice > 0 {
					position.OpenPrice = recovered.OpenPrice
				}
				if recovered.Lots > 0 {
					position.LotSize = recovered.Lots
				}
				s.logger.Warn("order ternyata sudah masuk saat koneksi terputus",
					"signal_id", signal.SignalID, "account_id", account.ID,
					"ticket", recovered.Ticket)
			} else if recErr != nil {
				s.logger.Error("gagal memeriksa order yang mungkin sudah masuk",
					"signal_id", signal.SignalID, "error", recErr.Error())
				return s.reject(ctx, user.ID, signal.SignalID, RejectAccountFrozen)
			} else {
				return s.reject(ctx, user.ID, signal.SignalID, RejectAccountFrozen)
			}
		} else {
			position.TicketNumber = result.Ticket
			if result.ExecutedPrice > 0 {
				position.OpenPrice = result.ExecutedPrice
			}
			if result.Lots > 0 {
				position.LotSize = result.Lots
			}
		}
	}

	// Langkah 7: catat posisi.
	if err := s.store.CreatePosition(ctx, position); err != nil {
		s.failExecution(ctx, key, err)
		return err
	}

	// Langkah 8: kuota harian bertambah begitu order hidup.
	if err := s.store.IncrementDailyTrade(ctx, user.ID, DayKey(now)); err != nil {
		s.logger.Error("gagal menambah kuota harian", "error", err.Error())
	}

	// Sinyal menjadi CONSUMED bagi user ini.
	s.recordUserSignal(ctx, user.ID, signal.SignalID, SignalConsumed, "", &account.ID)
	if err := s.updateUserSignalPosition(ctx, user.ID, signal.SignalID, position.ID); err != nil {
		s.logger.Error("gagal menautkan posisi ke sinyal", "error", err.Error())
	}

	if err := s.store.CompleteExecution(ctx, key, position.ID); err != nil {
		s.logger.Error("gagal menandai eksekusi selesai", "error", err.Error())
	}

	s.logger.Info("posisi dibuka",
		"user_id", user.ID,
		"signal_id", signal.SignalID,
		"account_id", account.ID,
		"ticket", position.TicketNumber,
		"lots", position.LotSize,
		"dry_run", dryRun)

	s.pub.PublishSignal(ctx, user.ID, signal.SignalID, SignalConsumed, "")
	if op, ok := s.pub.(PositionOpenedPublisher); ok {
		op.PublishPositionOpened(ctx, position, signal, isAuto)
	}
	s.pub.PublishTelemetry(ctx, account.ID)
	return nil
}

// findOpenOrder mencari posisi yang sudah terbuka dengan komentar sinyal tertentu.
// Dipakai saat pengiriman order gagal di tengah jalan: timeout bukan bukti
// bahwa broker tidak menerima order, dan menebak bisa menghasilkan dua posisi.
func (s *Service) findOpenOrder(ctx context.Context, client broker.Client, signalID string) (*broker.PositionState, error) {
	positions, err := client.ListPositions(ctx)
	if err != nil {
		return nil, err
	}
	for i := range positions {
		if positions[i].Comment == signalID {
			return &positions[i], nil
		}
	}
	return nil, nil
}

// calculateLot mengambil equity dan spesifikasi simbol dari broker.
// Mode simulasi memakai asumsi konservatif yang tetap bisa diuji.
func (s *Service) calculateLot(ctx context.Context, user *model.User, account *model.BrokerAccount, signal *model.Signal, dryRun bool) (LotCalculation, error) {
	riskPercent := account.RiskPercent
	if riskPercent <= 0 {
		riskPercent = 0.5
	}

	// Nilai cadangan dipakai juga untuk simulasi: satu sumber supaya angka di
	// lot dan di R tidak pernah berbeda.
	spec := broker.SymbolSpec{ValuePerLot: s.valuePerLotRunner(signal.Symbol), MinLot: 0.01, MaxLot: 100, LotStep: 0.01}
	equity := 10_000.0 // asumsi simulasi saat akun nyata tidak bisa dibaca

	if !dryRun {
		snapshot, err := s.accountSnapshot(ctx, account)
		if err != nil {
			return LotCalculation{}, fmt.Errorf("membaca akun: %w", err)
		}
		equity = snapshot.Equity

		client, err := s.client(ctx, account)
		if err != nil {
			return LotCalculation{}, err
		}
		if cur := strings.ToUpper(strings.TrimSpace(snapshot.Currency)); cur != "" && cur != "USD" {
			equity = convertAccountEquityToUSD(ctx, client, equity, cur)
		}
		if brokerSpec, specErr := client.Symbol(ctx, signal.Symbol); specErr == nil {
			spec = brokerSpec
		}
	}

	_ = user
	lotReq := LotRequest{
		Equity:      equity,
		RiskPercent: riskPercent,
		EntryPrice:  signal.EntryPrice,
		StopLoss:    signal.StopLoss,
		Spec: LotSpec{
			ValuePerLotPerPrice: spec.ValuePerLot,
			MinLot:              spec.MinLot,
			MaxLot:              spec.MaxLot,
			LotStep:             spec.LotStep,
		},
	}

	if account.RiskMode == model.RiskModeLot {
		return CalculateFixedLot(account.LotSize, lotReq)
	}

	calc, err := CalculateLot(lotReq)
	if err != nil {
		return LotCalculation{}, err
	}
	// Cegah pembulatan paksa ke MinLot bila lot minimum menyebabkan risiko aktual
	// membengkak melewati 2x batas risiko akun sekaligus di atas 2% ekuitas.
	if calc.ClampedMin && equity > 0 && calc.ValuePerLot > 0 {
		actualRiskPct := (calc.Lots * calc.ValuePerLot) / equity * 100
		safeCeiling := math.Max(MaxRiskPercent*2, riskPercent*2)
		if actualRiskPct > safeCeiling+1e-9 {
			return LotCalculation{}, fmt.Errorf("%w: risiko lot minimum %.2f%% melampaui batas aman %.2f%%", ErrNoLot, actualRiskPct, safeCeiling)
		}
	}
	return calc, nil
}

func convertAccountEquityToUSD(ctx context.Context, client broker.Client, equity float64, currency string) float64 {
	if equity <= 0 || currency == "" || currency == "USD" {
		return equity
	}
	if currency == "USC" {
		return equity / 100
	}
	if rate, err := client.CurrentPrice(ctx, currency+"USD"); err == nil && rate > 0 {
		return equity * rate
	}
	if rate, err := client.CurrentPrice(ctx, "USD"+currency); err == nil && rate > 0 {
		return equity / rate
	}
	return equity
}

// ---------------------------------------------------------------------------
// Pembaruan sinyal (CANCEL / EXPIRE / SUPERSEDE)
// ---------------------------------------------------------------------------

// alasanPenarikan memberi keterangan baku untuk penarikan sinyal.
//
// Payload Signal tidak membawa kolom Reason (itu milik SignalUpdate), jadi
// alasannya diturunkan dari jenis aksi agar riwayat tidak kosong.
func alasanPenarikan(action enginev1.SignalAction) string {
	switch action {
	case enginev1.SignalAction_SIGNAL_ACTION_EXPIRE:
		return "masa berlaku habis"
	case enginev1.SignalAction_SIGNAL_ACTION_CANCEL:
		return "setup dibatalkan engine"
	case enginev1.SignalAction_SIGNAL_ACTION_SUPERSEDE:
		return "digantikan sinyal baru"
	default:
		return ""
	}
}

// HandleUpdate menerapkan pembaruan status sinyal dari engine.
//
// Aturan mutlak plans/07 §4.1: bila user sudah memiliki posisi dari sinyal itu,
// pembaruan DIABAIKAN dan user tidak diberi tahu "batal".
func (s *Service) HandleUpdate(ctx context.Context, in *enginev1.SignalUpdate) {
	if in.GetSignalId() == "" {
		return
	}

	status := ingest.UpdateStatus(in.GetAction())
	if status == "" {
		return
	}

	unlock := s.lockSignalOp(in.GetSignalId())
	defer unlock()

	signal, err := s.store.FindSignal(ctx, in.GetSignalId())
	if err != nil {
		// Simpan jejak pembatalan dini agar bila PUBLISH sinyal yang sama masih
		// berada di belakangnya, eksekusi tidak akan pernah dilanjutkan.
		s.rememberEarlyCancel(in.GetSignalId(), status, in.GetReason(), in.GetSupersededBy())
		s.logger.Warn("pembaruan untuk sinyal tak dikenal",
			"signal_id", in.GetSignalId())
		return
	}
	// Sinyal sudah ada di database: bersihkan jejak dini bila sempat tercatat.
	_, _ = s.takeEarlyCancel(in.GetSignalId())

	if IsTerminal(signal.Status) {
		return
	}

	if status == SignalSuperseded {
		if err := s.store.MarkSuperseded(ctx, signal.SignalID, in.GetSupersededBy(), in.GetReason()); err != nil {
			s.logger.Error("gagal menandai sinyal digantikan", "error", err.Error())
			return
		}
	} else {
		if err := s.store.UpdateSignalStatus(ctx, signal.SignalID, status, in.GetReason()); err != nil {
			s.logger.Error("gagal memperbarui sinyal", "error", err.Error())
			return
		}
	}

	users, err := s.store.ListUsersBySymbol(ctx, signal.Symbol)
	if err != nil {
		s.logger.Error("gagal membaca pemantau simbol", "error", err.Error())
		return
	}

	for _, user := range users {
		// Sudah punya posisi? Posisi tidak pernah batal; jangan ganggu.
		if _, posErr := s.store.FindOpenPositionBySignal(ctx, user.ID, signal.SignalID); posErr == nil {
			s.logger.Info("pembaruan diabaikan karena posisi sudah ada",
				"user_id", user.ID, "signal_id", signal.SignalID, "action", status)
			continue
		}

		if !s.userWantsSignal(ctx, user.ID, signal) {
			continue
		}

		existing, findErr := s.store.FindUserSignal(ctx, user.ID, signal.SignalID)
		if findErr != nil && !errors.Is(findErr, apperror.ErrNotFound) {
			continue
		}
		if existing != nil && IsTerminal(existing.Status) {
			continue
		}

		s.recordUserSignal(ctx, user.ID, signal.SignalID, status, in.GetReason(), nil)
		s.pub.PublishSignal(ctx, user.ID, signal.SignalID, status, in.GetReason())
	}

	s.logger.Info("sinyal diperbarui",
		"signal_id", signal.SignalID, "status", status, "reason", in.GetReason())
}

// ---------------------------------------------------------------------------
// Aksi user
// ---------------------------------------------------------------------------

// Approve mengeksekusi sinyal atas nama user setelah konfirmasi di browser.
// Seluruh syarat diperiksa ulang: keadaan bisa berubah antara render dan klik.
func (s *Service) Approve(ctx context.Context, userID, signalID, accountID string) (RejectReason, error) {
	now := s.opts.Clock()

	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return RejectNone, err
	}

	signal, err := s.store.FindSignal(ctx, signalID)
	if err != nil {
		return RejectNone, err
	}

	// Jalur manual: user sudah melihat setupnya di dashboard dan memutuskan
	// sekarang. Masa berlaku tidak boleh membatalkannya selama harga masih di
	// dalam range entry (diperiksa gerbang slippage di evaluate).
	if reason := s.validateSignalMode(ctx, signal, now, true); reason != RejectNone {
		return reason, nil
	}

	us, findErr := s.store.FindUserSignal(ctx, userID, signalID)
	if findErr != nil && !errors.Is(findErr, apperror.ErrNotFound) {
		return RejectNone, findErr
	}
	if us != nil && IsTerminal(us.Status) {
		// Sinyal yang tertahan kuota bukan alasan menutup pintu manual:
		// sistem sudah bilang tidak, user masih boleh memutuskan.
		if !(us.Status == SignalRejected && OverrideKuotaManual(RejectReason(us.RejectReason))) {
			return RejectNotReady, nil
		}
	}

	account, err := s.store.FindBrokerAccount(ctx, accountID, userID)
	if err != nil {
		return RejectAccountFrozen, nil
	}

	tier := EffectiveTier(user.Tier, user.TrialExpiresAt, user.TierExpiresAt, now)

	// Kunci akun juga berlaku di jalur manual: dua klik konfirmasi yang tiba
	// bersamaan tidak boleh melahirkan dua order. Tanpa ini, celah yang sama
	// dengan jalur otomatis tetap terbuka lewat tombol eksekusi.
	release, dikunci, lockErr := s.lockAccount(ctx, account.ID, signal.SignalID, now)
	if lockErr != nil {
		return RejectNone, lockErr
	}
	if !dikunci {
		s.logger.Info("akun sedang mengeksekusi sinyal lain",
			"user_id", userID, "account_id", account.ID, "signal_id", signal.SignalID)
		return RejectOpenPosition, nil
	}
	defer release()

	reason, evalErr := s.evaluate(ctx, user, account, signal, tier, now)
	if evalErr != nil {
		return RejectNone, evalErr
	}
	if reason != RejectNone {
		// Pagar harian bisa dilewati di jalur manual, pagar lain tidak.
		// Keputusan menembus kuota dicatat agar bisa diaudit nanti.
		if !OverrideKuotaManual(reason) {
			return reason, nil
		}
		s.logger.Info("kuota harian dilewati lewat eksekusi manual",
			"evt", "signal.override.kuota",
			"user_id", userID, "signal_id", signal.SignalID,
			"account_id", account.ID, "alasan", string(reason))
	}
	return RejectNone, s.execute(ctx, user, account, signal, false)
}

// Skip menandai sinyal dilewati user. Kuota harian tidak terpakai.
func (s *Service) Skip(ctx context.Context, userID, signalID string) error {
	s.recordUserSignal(ctx, userID, signalID, SignalSkipped, "dilewati user", nil)
	s.pub.PublishSignal(ctx, userID, signalID, SignalSkipped, "")
	return nil
}

// ---------------------------------------------------------------------------
// Bantuan internal
// ---------------------------------------------------------------------------

func (s *Service) ackSignal(ctx context.Context, strategy, signalID string) error {
	s.mu.Lock()
	ack := s.acks[strategy]
	s.mu.Unlock()

	if ack == nil {
		return nil
	}
	return ack(ctx, signalID)
}

// AckSender dipasang setelah stream engine tersambung.
type AckSender func(ctx context.Context, signalID string) error

// SymbolSync mengirim selisih simbol terakhir ke engine.
// Hanya selisih yang dikirim agar engine tidak menerima beban penuh tiap menit.
type SymbolSync func(ctx context.Context, add, remove []string) error

// strategySyncState menyimpan penyelaras langganan dan sidik terakhir untuk
// satu strategi.
type strategySyncState struct {
	sync SymbolSync
	// generation bertambah setiap stream tersambung ulang. Penulisan sidik dari
	// sesi lama diabaikan agar tidak menandai terkirim pada koneksi baru.
	generation  uint64
	symbolsSent map[string]struct{}
	// pairsSent adalah sidik pasangan timeframe terakhir yang dikirim.
	// Perubahan saklar timeframe tidak mengubah daftar simbol, jadi tanpa
	// pembanding ini perubahan saklar tidak pernah sampai ke engine.
	pairsSent string
}

// syncStateLocked mengembalikan state sinkronisasi satu strategi.
// Wajib dipanggil dengan s.mu terkunci.
func (s *Service) syncStateLocked(strategy string) *strategySyncState {
	if s.syncStates == nil {
		s.syncStates = make(map[string]*strategySyncState, 2)
	}
	state := s.syncStates[strategy]
	if state == nil {
		state = &strategySyncState{}
		s.syncStates[strategy] = state
	}
	return state
}

// SetSymbolSync memasang penyelaras langganan dan mengosongkan daftar terkirim.
// Dipanggil saat stream terhubung: engine bisa sudah lupa setelah putus,
// sehingga seluruh kebutuhan dikirim ulang.
func (s *Service) SetSymbolSync(strategy string, sync SymbolSync) {
	s.mu.Lock()
	state := s.syncStateLocked(strategy)
	state.sync = sync
	state.generation++
	state.symbolsSent = nil
	// Engine bisa sudah lupa pasangan setelah putus, sehingga sidik lama harus
	// dibuang agar penyelarasan berikutnya mengirim ulang.
	state.pairsSent = ""
	s.mu.Unlock()
}

// SyncSubscriptions membandingkan kebutuhan engine saat ini dengan yang sudah
// dikirim, lalu menyelaraskannya.
//
// Tanpa ini, simbol hanya dihitung sekali saat koneksi dibuka: user yang baru
// berlangganan harus menunggu Runner restart. Fungsi ini dipanggil berkala.
//
// Pasangan timeframe ikut diperiksa: mengubah saklar di dashboard tidak
// mengubah daftar simbol, jadi tanpa perbandingan ini engine akan terus
// memakai pasangan lama sampai stream terputus.
func (s *Service) SyncSubscriptions(ctx context.Context, strategy string, desired []string) error {
	return s.syncDenganSidik(ctx, strategy, desired, s.hitungPasangan(ctx, strategy, desired))
}

// syncDenganSidik menjalankan penyelarasan dengan sidik pasangan yang sudah
// dihitung. Dipisahkan agar bisa diuji tanpa basis data.
func (s *Service) syncDenganSidik(ctx context.Context, strategy string, desired []string, sidik string) error {
	s.mu.Lock()
	state := s.syncStateLocked(strategy)
	sync := state.sync
	generasi := state.generation
	sent := state.symbolsSent
	pasanganLama := state.pairsSent
	s.mu.Unlock()

	if sync == nil {
		return nil
	}

	add, remove := diffSymbols(sent, desired)
	daftarBerubah := len(add) > 0 || len(remove) > 0
	pasanganBerubah := sidik != pasanganLama

	if !daftarBerubah && !pasanganBerubah {
		return nil
	}

	if err := sync(ctx, add, remove); err != nil {
		// Daftar terkirim dan sidik tidak diperbarui: percobaan berikutnya
		// mengulang. Menyimpan sidik sebelum sukses akan membuat perubahan
		// pasangan hilang selamanya bila satu percobaan gagal.
		return err
	}

	s.mu.Lock()
	if state.generation == generasi {
		state.symbolsSent = toSymbolSet(desired)
		state.pairsSent = sidik
	}
	s.mu.Unlock()

	if daftarBerubah {
		s.logger.Info("langganan engine diselaraskan",
			"strategy", strategy,
			"ditambah", add, "dicabut", remove, "total", len(desired))
	} else {
		s.logger.Info("pasangan timeframe engine diselaraskan",
			"strategy", strategy, "simbol", len(desired))
	}
	return nil
}

// hitungPasangan menyusun sidik pasangan timeframe dari seluruh simbol.
func (s *Service) hitungPasangan(ctx context.Context, strategy string, symbols []string) string {
	semua := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		// Fingerprint HARUS memuat simbol. Tanpa itu, mengubah switch satu
		// simbol tidak mengubah sidik, sehingga sinkronisasi ke engine tidak
		// pernah terpicu dan perubahan di UI tidak sampai ke engine.
		kebijakan, err := s.ComputeSymbolPairs(ctx, symbol)
		if err != nil {
			continue
		}
		policy, ada := kebijakan[strategy]
		if !ada {
			// Simbol tanpa kebijakan untuk metode ini: engine memakai daftar
			// bawaan. Tetap dicatat agar perpindahan dari/dan ke kondisi ini
			// ikut memicu sinkronisasi.
			semua = append(semua, symbol+":"+strategy+":bawaan")
			continue
		}
		if policy.Restricted {
			semua = append(semua, symbol+":"+strategy+":restricted")
		}
		for _, p := range policy.Pairs {
			semua = append(semua, symbol+":"+strategy+":"+p.Htf+"/"+p.Ltf)
		}
	}
	sort.Strings(semua)

	var b strings.Builder
	var terakhir string
	for _, item := range semua {
		if item == terakhir {
			continue
		}
		b.WriteString(item)
		b.WriteByte('\n')
		terakhir = item
	}
	return b.String()
}

// SymbolsSent mengembalikan salinan simbol yang sudah dikirim ke engine untuk
// satu strategi.
func (s *Service) SymbolsSent(strategy string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.syncStates[strategy]
	if state == nil {
		return nil
	}
	out := make([]string, 0, len(state.symbolsSent))
	for sym := range state.symbolsSent {
		out = append(out, sym)
	}
	return out
}

func toSymbolSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, it := range items {
		if it != "" {
			out[it] = struct{}{}
		}
	}
	return out
}

// diffSymbols menghitung tambah dan cabut antara kondisi terkirim dan kebutuhan.
func diffSymbols(sent map[string]struct{}, desired []string) (add, remove []string) {
	wanted := toSymbolSet(desired)
	for sym := range wanted {
		if _, ok := sent[sym]; !ok {
			add = append(add, sym)
		}
	}
	for sym := range sent {
		if _, ok := wanted[sym]; !ok {
			remove = append(remove, sym)
		}
	}
	sort.Strings(add)
	sort.Strings(remove)
	return add, remove
}

// SetAckSender memasang pengirim ACK untuk satu strategi. Dipanggil saat stream
// terhubung. ACK harus dikembalikan ke engine asal sinyal, bukan ke engine lain.
func (s *Service) SetAckSender(strategy string, sender AckSender) {
	s.mu.Lock()
	if s.acks == nil {
		s.acks = make(map[string]AckSender, 2)
	}
	s.acks[strategy] = sender
	s.mu.Unlock()
}

// recordUserSignal menyimpan status sinyal untuk satu user dan mencatat
// peralihannya.
//
// Peristiwa hanya dicatat saat status benar-benar berubah. Engine mengirim
// kemajuan radar berulang dengan status yang sama; mencatatnya sebagai
// peristiwa membuat tabel jejak penuh baris identik dan menyamarkan kapan
// setup sebenarnya berpindah tahap. Pemanggil tidak perlu memeriksa perubahan
// sendiri.
//
// Nilai balik kedua menunjukkan apakah status berubah, sehingga pemanggil tahu
// kapan notifikasi masih perlu dikirim.
func (s *Service) recordUserSignal(ctx context.Context, userID, signalID, status, reason string, accountID *string) bool {
	now := s.opts.Clock()

	// Status tersimpan dibaca untuk mengetahui tahap asal dan mendeteksi kiriman
	// berulang. Kegagalan baca diperlakukan sebagai "belum ada": status tetap
	// disimpan, hanya jejak asalnya yang kosong.
	asal := ""
	alasanLama := ""
	if lama, err := s.store.FindUserSignal(ctx, userID, signalID); err == nil && lama != nil {
		asal = lama.Status
		alasanLama = lama.RejectReason
	}
	// Alasan sesaat tidak boleh menempel setelah nasib sinyal berakhir.
	reason = alasanTerbaru(alasanLama, reason)
	us := &model.UserSignal{
		UserID:       userID,
		SignalID:     signalID,
		Status:       status,
		RejectReason: truncate(reason, model.RejectReasonMaxLen),
		DecidedAt:    &now,
	}
	if accountID != nil {
		us.BrokerAccountID = *accountID
	}
	if err := s.store.UpsertUserSignal(ctx, us); err != nil {
		s.logger.Error("gagal menyimpan status sinyal user",
			"user_id", userID, "signal_id", signalID, "error", err.Error())
		return false
	}
	if asal == status {
		return false
	}

	event := &model.SignalEvent{
		UserID:     userID,
		SignalID:   signalID,
		FromStatus: asal,
		ToStatus:   status,
		Reason:     truncate(reason, model.RejectReasonMaxLen),
		CreatedAt:  now,
	}
	// Slot diisi supaya kunci notifikasi bisa menyatukan tahap awal (baris
	// status) dan tahap eksekusi (sinyal baru) yang memakai signal_id berbeda.
	// Kegagalan baca dibiarkan: event tetap tercatat, hanya tanpa slot.
	if sig, err := s.store.FindSignal(ctx, signalID); err == nil && sig != nil {
		event.Symbol, event.Htf, event.Ltf, event.Direction =
			sig.Symbol, sig.Htf, sig.Ltf, sig.Direction
	}
	if err := s.store.RecordSignalEvent(ctx, event); err != nil {
		s.logger.Error("gagal mencatat peristiwa sinyal", "error", err.Error())
	}
	return true
}

func (s *Service) updateUserSignalPosition(ctx context.Context, userID, signalID, positionID string) error {
	return s.store.SetUserSignalPosition(ctx, userID, signalID, positionID)
}

// reject menahan eksekusi.
//
// Sinyal TETAP dikirim ke Console (plans/05 §3): user berhak melihat setup
// yang muncul walau sistem memutuskan tidak mengeksekusinya.
func (s *Service) reject(ctx context.Context, userID, signalID string, reason RejectReason) error {
	s.recordUserSignal(ctx, userID, signalID, SignalRejected, string(reason), nil)
	s.pub.PublishSignal(ctx, userID, signalID, SignalRejected, string(reason))

	s.logger.Info("sinyal ditahan",
		"user_id", userID, "signal_id", signalID, "reason", string(reason))
	return nil
}

// rejectAccount mencatat penolakan untuk satu akun tertentu sehingga dashboard
// dapat menjelaskan akun mana yang tertahan dan alasannya.
func (s *Service) rejectAccount(ctx context.Context, userID, accountID, signalID string, reason RejectReason) error {
	s.recordUserSignal(ctx, userID, signalID, SignalRejected, string(reason), &accountID)
	s.pub.PublishSignal(ctx, userID, signalID, SignalRejected, string(reason))

	s.logger.Info("sinyal ditahan untuk akun",
		"user_id", userID, "account_id", accountID, "signal_id", signalID, "reason", string(reason))
	return nil
}

func (s *Service) failExecution(ctx context.Context, key string, cause error) {
	if err := s.store.FailExecution(ctx, key, truncate(errText(cause), model.RejectReasonMaxLen)); err != nil {
		s.logger.Error("gagal menandai eksekusi gagal", "error", err.Error())
	}
}

func (s *Service) markBrokerDown(ctx context.Context, account *model.BrokerAccount, cause error) {
	now := s.opts.Clock()
	state := &model.BrokerAccountState{
		BrokerAccountID: account.ID,
		UserID:          account.UserID,
		Connected:       false,
		LastError:       truncate(errText(cause), 255),
		LastSyncedAt:    &now,
	}
	if err := s.store.UpsertAccountState(ctx, state); err != nil {
		s.logger.Error("gagal menyimpan status akun", "error", err.Error())
	}
	s.pub.PublishTelemetry(ctx, account.ID)
}

func (s *Service) loadUser(ctx context.Context, userID string) (*model.User, error) {
	return s.store.FindUserByID(ctx, userID)
}
