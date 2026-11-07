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
	"sync"
	"time"

	enginev1 "github.com/KirisakiRei/go-ila-platform/backend/gen/engine/v1"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/broker"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/repository"
	ingest "github.com/KirisakiRei/go-ila-platform/backend/internal/runner/ingest"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/cryptox"
)

// Publisher menerbitkan perubahan status ke Console.
// Interface dipisah supaya Runner bisa diuji tanpa kanal notifikasi.
type Publisher interface {
	PublishSignal(ctx context.Context, userID, signalID, status, reason string)
	PublishTelemetry(ctx context.Context, accountID string)
}

// Store adalah kebutuhan penyimpanan Runner.
// Dideklarasikan di sisi pemakai agar paket ini tidak terikat implementasi.
type Store interface {
	repository.RunnerRepository
}

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
	clients   map[string]broker.Client
	keyring   *cryptox.Keyring
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
		store:   store,
		brokers: brokers,
		pub:     pub,
		logger:  logger,
		opts:    opts,
		market:  enginev1.MarketState_MARKET_OPEN,
		clients: make(map[string]broker.Client),
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
			"signal_id", signal.SignalID, "status", existing.Status)
		return
	}

	if err := s.store.UpsertSignal(ctx, signal); err != nil {
		s.logger.Error("gagal menyimpan sinyal",
			"signal_id", signal.SignalID, "error", err.Error())
		return
	}

	s.logger.Info("sinyal diterima",
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
		s.logger.Warn("ACK sinyal gagal", "signal_id", signal.SignalID, "error", err.Error())
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
	users, err := s.store.ListUsersBySymbol(ctx, signal.Symbol)
	if err != nil {
		s.logger.Error("gagal membaca pemantau simbol", "error", err.Error())
		return
	}

	for _, user := range users {
		if err := s.processSignalForUser(ctx, &user, signal); err != nil {
			s.logger.Error("gagal memproses sinyal untuk user",
				"user_id", user.ID, "signal_id", signal.SignalID, "error", err.Error())
		}
	}
}

// processSignalForUser menjalankan seluruh urutan pemeriksaan plans/14 §5.2
// lalu memutuskan: tahan, tunggu konfirmasi, atau eksekusi.
func (s *Service) processSignalForUser(ctx context.Context, user *model.User, signal *model.Signal) error {
	now := s.opts.Clock()

	tier := EffectiveTier(user.Tier, user.TrialExpiresAt, user.TierExpiresAt, now)

	// [1] Sinyal sah?
	if reason := s.validateSignal(signal, now); reason != RejectNone {
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
		if account.Mode == "auto" && s.opts.AutoExecute {
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
func (s *Service) validateSignal(signal *model.Signal, now time.Time) RejectReason {
	if signal.EntryPrice <= 0 || signal.StopLoss <= 0 || signal.TakeProfit <= 0 {
		return RejectBadSignal
	}
	if !now.Before(signal.ExpiresAt) {
		return RejectExpired
	}
	if s.opts.MarketOpenCheck && !s.marketOpen() {
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
	if limit > 2 {
		limit = 2
	}

	out := make([]model.BrokerAccount, 0, limit)
	for _, account := range accounts {
		if !account.IsActive || account.IsFrozen {
			continue
		}
		if len(out) >= limit {
			break
		}
		out = append(out, account)
	}
	return out, nil
}

// evaluate menjalankan pemeriksaan 3-5 (plans/14 §5.2) pada satu akun.
func (s *Service) evaluate(ctx context.Context, user *model.User, account *model.BrokerAccount, signal *model.Signal, tier string, now time.Time) (RejectReason, error) {
	// Penurunan tier tidak menyentuh posisi hidup, tetapi mematikan mode auto.
	if !ModeAllowed(tier, account.Mode) {
		return RejectEntitlement, nil
	}

	// Masa tenggang dan suspend: hanya entry baru yang diblokir.
	if WithinGrace(user.GraceUntil, now) || user.SuspendedAt != nil {
		return RejectSuspended, nil
	}

	// [3] Broker harus bisa dihubungi; dry-run tidak butuh koneksi.
	if !s.mustDryRun(&userSession{User: user}, account) {
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

	daily := DailyStatus{
		Day:         summary.Day,
		WinCount:    summary.WinCount,
		LossCount:   summary.LossCount,
		TotalTrades: summary.TotalTrades,
		OpenCount:   len(openPositions),
	}
	if reason := EvaluateDaily(daily, s.opts.Policy); reason != RejectNone {
		return reason, nil
	}

	// [5] Slippage guard, kecuali simulasi.
	if s.opts.SlippageGuard && !s.mustDryRun(&userSession{User: user}, account) {
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
		// Kunci sudah ada: sinyal ini pernah diproses untuk akun ini.
		s.logger.Warn("eksekusi ganda dicegah",
			"signal_id", signal.SignalID, "account_id", account.ID)
		return s.reject(ctx, user.ID, signal.SignalID, RejectDuplicate)
	}

	dryRun := s.mustDryRun(&userSession{User: user}, account)
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
			return s.reject(ctx, user.ID, signal.SignalID, RejectAccountFrozen)
		}

		position.TicketNumber = result.Ticket
		if result.ExecutedPrice > 0 {
			position.OpenPrice = result.ExecutedPrice
		}
		if result.Lots > 0 {
			position.LotSize = result.Lots
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

	if reason := s.validateSignal(signal, now); reason != RejectNone {
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

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
