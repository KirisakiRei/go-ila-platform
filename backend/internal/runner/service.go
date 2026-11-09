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
	"sort"
	"sync"
	"time"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/authstate"
	"ila/backend/internal/broker"
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

	mu        sync.Mutex
	market    enginev1.MarketState
	engineVer string
	ack       AckSender
	// symbolSync dan symbolsSent menjaga agar engine hanya memantau simbol
	// yang benar-benar dibutuhkan pengguna saat ini.
	symbolSync  SymbolSync
	symbolsSent map[string]struct{}
	clients     map[string]broker.Client
	keyring     *cryptox.Keyring
	// schedule mengawasi kecocokan status pasar dengan jadwal NY.
	schedule *ScheduleWatch
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
		store:    store,
		brokers:  brokers,
		pub:      pub,
		logger:   logger,
		opts:     opts,
		market:   enginev1.MarketState_MARKET_OPEN,
		clients:  make(map[string]broker.Client),
		schedule: NewScheduleWatch(10*time.Minute, 10*time.Minute),
	}
}

// SetMarketState mencatat status pasar terakhir dari engine.
func (s *Service) SetMarketState(state enginev1.MarketState) {
	s.mu.Lock()
	changed := s.market != state
	s.market = state
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

// persistMarketState menyimpan status pasar pada setiap engine yang dikenal.
func (s *Service) persistMarketState(state enginev1.MarketState) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := s.opts.Clock()
	label := marketStateLabel(state)

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
func (s *Service) SetEngineVersion(version string) {
	s.mu.Lock()
	s.engineVer = version
	s.mu.Unlock()
}

func (s *Service) marketOpen() bool {
	s.mu.Lock()
	state := s.market
	s.mu.Unlock()
	return ingest.MarketOpen(state)
}

// ---------------------------------------------------------------------------
// Ingest sinyal
// ---------------------------------------------------------------------------

// HandleSignal memproses sinyal baru dari engine untuk seluruh user yang
// memantau simbolnya. Setiap user dinilai terpisah: satu sinyal bisa menjadi
// posisi bagi user A, tertahan bagi user B, dan terlewat bagi user C.
func (s *Service) HandleSignal(ctx context.Context, in *enginev1.Signal) {
	signal, ok := ingest.ConvertSignal(in)
	if !ok {
		s.logger.Warn("sinyal engine tidak lengkap, diabaikan",
			"signal_id", in.GetSignalId())
		return
	}

	if existing, err := s.store.FindSignal(ctx, signal.SignalID); err == nil && IsTerminal(existing.Status) {
		// Sinyal terminal tidak boleh dihidupkan kembali oleh kiriman ulang.
		s.logger.Info("sinyal sudah final, dilewati",
			"evt", "signal.skip.final",
			"signal_id", signal.SignalID, "status", existing.Status)
		return
	}

	if err := s.store.UpsertSignal(ctx, signal); err != nil {
		s.logger.Error("gagal menyimpan sinyal",
			"evt", "signal.simpan.gagal",
			"signal_id", signal.SignalID, "error", err.Error())
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

	if err := s.ackSignal(ctx, signal.SignalID); err != nil {
		// ACK gagal bukan alasan membuang sinyal: engine boleh mengulang.
		s.logger.Warn("ACK sinyal gagal", "evt", "signal.ack.gagal",
			"signal_id", signal.SignalID, "error", err.Error())
	}

	s.fanOutReady(ctx, signal)
}

func (s *Service) fanOutIntermediate(ctx context.Context, signal *model.Signal) {
	users, err := s.store.ListUsersBySymbol(ctx, signal.Symbol)
	if err != nil {
		s.logger.Error("gagal membaca pemantau simbol", "error", err.Error())
		return
	}

	for _, user := range users {
		s.recordUserSignal(ctx, user.ID, signal.SignalID, signal.Status, "", nil)
		s.pub.PublishSignal(ctx, user.ID, signal.SignalID, signal.Status, "")
	}
}

func (s *Service) fanOutReady(ctx context.Context, signal *model.Signal) {
	// Sinyal disebar ke setiap akun yang mencentang simbol ini. Satu user dapat
	// menyumbang lebih dari satu akun karena pilihan pair kini per akun.
	pairs, err := s.store.ListAccountsBySymbol(ctx, signal.Symbol)
	if err == nil {
		for i := range pairs {
			user := pairs[i].User
			account := pairs[i].Account
			if procErr := s.processSignalForAccount(ctx, &user, &account, signal); procErr != nil {
				s.logger.Error("gagal memproses sinyal untuk akun",
					"user_id", user.ID, "account_id", account.ID,
					"signal_id", signal.SignalID, "error", procErr.Error())
			}
		}
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
		return s.execute(ctx, user, account, signal)
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
			return s.execute(ctx, user, &account, signal)
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
		return RejectExpired
	}
	if s.marketOpenRequired(ctx) && !s.marketOpen() {
		return RejectMarketClosed
	}
	return RejectNone
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
	if !s.mustDryRun(ctx, &userSession{User: user}, account) {
		client, err := s.client(ctx, account)
		if err != nil {
			return RejectAccountFrozen, nil
		}
		if _, err := client.Account(ctx); err != nil {
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

	price, err := client.CurrentPrice(ctx, signal.Symbol)
	if err != nil || price <= 0 {
		// Harga tidak terbaca: menolak lebih aman daripada menebak.
		s.logger.Warn("harga acuan tidak tersedia",
			"symbol", signal.Symbol, "error", errText(err))
		return RejectSlippage
	}

	ok, drift, limit := SlippageCheck(signal.Symbol, signal.EntryPrice, price, signal.StopLoss)
	if !ok {
		s.logger.Warn("slippage melewati batas",
			"symbol", signal.Symbol, "signal_id", signal.SignalID,
			"drift", drift, "limit", limit)
		return RejectSlippage
	}
	return RejectNone
}

// ---------------------------------------------------------------------------
// Eksekusi
// ---------------------------------------------------------------------------

// execute membuka posisi nyata (atau simulasi) untuk satu sinyal dan satu akun.
func (s *Service) execute(ctx context.Context, user *model.User, account *model.BrokerAccount, signal *model.Signal) error {
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

	spec := broker.SymbolSpec{ValuePerLot: valuePerLotFallback(signal.Symbol), MinLot: 0.01, MaxLot: 100, LotStep: 0.01}
	equity := 10_000.0 // asumsi simulasi saat akun nyata tidak bisa dibaca

	if !dryRun {
		client, err := s.client(ctx, account)
		if err != nil {
			return LotCalculation{}, err
		}

		snapshot, err := client.Account(ctx)
		if err != nil {
			return LotCalculation{}, fmt.Errorf("membaca akun: %w", err)
		}
		equity = snapshot.Equity

		if brokerSpec, specErr := client.Symbol(ctx, signal.Symbol); specErr == nil {
			spec = brokerSpec
		}
	}

	_ = user
	return CalculateLot(LotRequest{
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
	})
}

// valuePerLotFallback adalah nilai per lot teoretis bila broker tidak
// mengirim spesifikasi simbol. Dipakai hanya untuk simulasi.
func valuePerLotFallback(symbol string) float64 {
	switch symbol {
	case "XAUUSD":
		return 100 // 1 lot = 100 oz
	case "USTEC", "US100", "NAS100":
		return 1
	default:
		return 100_000 // kontrak forex standar
	}
}

// ---------------------------------------------------------------------------
// Pembaruan sinyal (CANCEL / EXPIRE / SUPERSEDE)
// ---------------------------------------------------------------------------

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

	signal, err := s.store.FindSignal(ctx, in.GetSignalId())
	if err != nil {
		s.logger.Warn("pembaruan untuk sinyal tak dikenal",
			"signal_id", in.GetSignalId())
		return
	}

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

	if reason := s.validateSignal(ctx, signal, now); reason != RejectNone {
		return reason, nil
	}

	us, findErr := s.store.FindUserSignal(ctx, userID, signalID)
	if findErr != nil && !errors.Is(findErr, apperror.ErrNotFound) {
		return RejectNone, findErr
	}
	if us != nil && IsTerminal(us.Status) {
		return RejectNotReady, nil
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
		return reason, nil
	}

	return RejectNone, s.execute(ctx, user, account, signal)
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

func (s *Service) ackSignal(ctx context.Context, signalID string) error {
	s.mu.Lock()
	ack := s.ack
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

// SetSymbolSync memasang penyelaras langganan dan mengosongkan daftar terkirim.
// Dipanggil saat stream terhubung: engine bisa sudah lupa setelah putus,
// sehingga seluruh kebutuhan dikirim ulang.
func (s *Service) SetSymbolSync(sync SymbolSync) {
	s.mu.Lock()
	s.symbolSync = sync
	s.symbolsSent = nil
	s.mu.Unlock()
}

// SyncSubscriptions membandingkan kebutuhan engine saat ini dengan yang sudah
// dikirim, lalu mengirim selisihnya lewat Subscribe/Unsubscribe.
//
// Tanpa ini, simbol hanya dihitung sekali saat koneksi dibuka: user yang baru
// berlangganan harus menunggu Runner restart. Fungsi ini dipanggil berkala.
func (s *Service) SyncSubscriptions(ctx context.Context, desired []string) error {
	s.mu.Lock()
	sync := s.symbolSync
	sent := s.symbolsSent
	s.mu.Unlock()

	if sync == nil {
		return nil
	}

	add, remove := diffSymbols(sent, desired)
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}

	if err := sync(ctx, add, remove); err != nil {
		// Daftar terkirim tidak diperbarui: percobaan berikutnya mengulang.
		return err
	}

	s.mu.Lock()
	s.symbolsSent = toSymbolSet(desired)
	s.mu.Unlock()

	s.logger.Info("langganan engine diselaraskan",
		"ditambah", add, "dicabut", remove, "total", len(desired))
	return nil
}

// SymbolsSent mengembalikan salinan simbol yang sudah dikirim ke engine.
func (s *Service) SymbolsSent() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.symbolsSent))
	for sym := range s.symbolsSent {
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

// SetAckSender memasang pengirim ACK. Dipanggil saat stream terhubung.
func (s *Service) SetAckSender(sender AckSender) {
	s.mu.Lock()
	s.ack = sender
	s.mu.Unlock()
}

func (s *Service) recordUserSignal(ctx context.Context, userID, signalID, status, reason string, accountID *string) {
	now := s.opts.Clock()

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
		return
	}

	event := &model.SignalEvent{
		UserID:    userID,
		SignalID:  signalID,
		ToStatus:  status,
		Reason:    truncate(reason, model.RejectReasonMaxLen),
		CreatedAt: now,
	}
	if err := s.store.RecordSignalEvent(ctx, event); err != nil {
		s.logger.Error("gagal mencatat peristiwa sinyal", "error", err.Error())
	}
}

func (s *Service) updateUserSignalPosition(ctx context.Context, userID, signalID, positionID string) error {
	us, err := s.store.FindUserSignal(ctx, userID, signalID)
	if err != nil {
		return err
	}
	us.PositionID = positionID
	return s.store.UpsertUserSignal(ctx, us)
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
