package runner_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/broker"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/pkg/apperror"
)

// ---------------------------------------------------------------------------
// Store in-memory: cukup untuk menguji alur tanpa database.
// ---------------------------------------------------------------------------

type fakeStore struct {
	mu sync.Mutex

	signals     map[string]*model.Signal
	userSignals map[string]*model.UserSignal
	positions   map[string]*model.Position
	executions  map[string]*model.ExecutionRecord
	daily       map[string]*model.DailySummary
	users       map[string]*model.User
	accounts    map[string]*model.BrokerAccount
	subs        []model.Subscription
	events      []model.SignalEvent
	// correlations adalah peta korelasi SMT yang disajikan ke Runner.
	correlations map[string][]string
	// switches menyimpan saklar metode per akun × pair untuk pengujian.
	switches map[string]map[string]bool
	// accountLocks meniru kunci eksekusi per akun pada basis data.
	accountLocks map[string]string
	// timeframes menyimpan saklar timeframe untuk pengujian, berkunci
	// "akun|simbol|metode|htf|ltf".
	timeframes map[string]repository.TimeframeState
	// staleClaims menandai klaim yang dianggap tertinggal pada pengujian.
	staleClaims map[string]struct{}
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		signals:      map[string]*model.Signal{},
		userSignals:  map[string]*model.UserSignal{},
		positions:    map[string]*model.Position{},
		executions:   map[string]*model.ExecutionRecord{},
		daily:        map[string]*model.DailySummary{},
		users:        map[string]*model.User{},
		accounts:     map[string]*model.BrokerAccount{},
		correlations: map[string][]string{},
		switches:     map[string]map[string]bool{},
		accountLocks: map[string]string{},
		timeframes:   map[string]repository.TimeframeState{},
	}
}

func userSignalKey(userID, signalID string) string { return userID + "|" + signalID }
func dailyKey(userID, day string) string           { return userID + "|" + day }

// UpsertSignal meniru repository asli: baris yang sudah ada hanya diperbarui
// pada kolom ringkasan dan masa berlaku. Kolom status sengaja tidak ditimpa
// agar status lokal (ACKED) tidak hilang oleh kiriman ulang engine.
func (s *fakeStore) UpsertSignal(_ context.Context, signal *model.Signal) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ada, ok := s.signals[signal.SignalID]; ok {
		ada.Summary = signal.Summary
		ada.NextAction = signal.NextAction
		ada.ExpiresAt = signal.ExpiresAt
		ada.UpdatedAt = signal.UpdatedAt
		return nil
	}

	copySignal := *signal
	s.signals[signal.SignalID] = &copySignal
	return nil
}

func (s *fakeStore) FindSignal(_ context.Context, signalID string) (*model.Signal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sig, ok := s.signals[signalID]; ok {
		copySignal := *sig
		return &copySignal, nil
	}
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) UpdateSignalStatus(_ context.Context, signalID, status, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sig, ok := s.signals[signalID]; ok {
		sig.Status = status
		sig.CancelReason = reason
	}
	return nil
}

func (s *fakeStore) MarkSuperseded(_ context.Context, signalID, newSignalID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sig, ok := s.signals[signalID]; ok {
		sig.Status = model.SignalStatusSuperseded
		sig.SupersededBy = newSignalID
		sig.CancelReason = reason
	}
	return nil
}

func (s *fakeStore) TouchSignal(_ context.Context, _ string, _ time.Time) error { return nil }

// ExpireStaleSignals meniru penyapu database: hanya READY_TO_ENTRY dan
// OPPORTUNITY yang lewat masa berlaku yang ditutup.
func (s *fakeStore) ExpireStaleSignals(_ context.Context, now time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var jumlah int64
	for _, sig := range s.signals {
		if sig.Status != model.SignalStatusReadyToEntry && sig.Status != model.SignalStatusOpportunity {
			continue
		}
		if now.Before(sig.ExpiresAt) {
			continue
		}
		sig.Status = model.SignalStatusExpired
		jumlah++
	}
	return jumlah, nil
}

func (s *fakeStore) UpsertUserSignal(_ context.Context, us *model.UserSignal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyUs := *us
	s.userSignals[userSignalKey(us.UserID, us.SignalID)] = &copyUs
	return nil
}

func (s *fakeStore) FindUserSignal(_ context.Context, userID, signalID string) (*model.UserSignal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if us, ok := s.userSignals[userSignalKey(userID, signalID)]; ok {
		copyUs := *us
		return &copyUs, nil
	}
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) FindUserSignalByPosition(_ context.Context, _ string) (*model.UserSignal, error) {
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) RecordSignalEvent(_ context.Context, event *model.SignalEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, *event)
	return nil
}

func (s *fakeStore) CreatePosition(_ context.Context, pos *model.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pos.ID == "" {
		pos.ID = "pos-" + pos.SourceSignalID + "-" + pos.BrokerAccountID
	}
	copyPos := *pos
	s.positions[pos.ID] = &copyPos
	return nil
}

func (s *fakeStore) FindPosition(_ context.Context, id string) (*model.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pos, ok := s.positions[id]; ok {
		copyPos := *pos
		return &copyPos, nil
	}
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) FindOpenPositionBySignal(_ context.Context, userID, signalID string) (*model.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pos := range s.positions {
		if pos.UserID == userID && pos.SourceSignalID == signalID && pos.Status == model.PositionStatusOpen {
			copyPos := *pos
			return &copyPos, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) ListOpenPositions(_ context.Context) ([]model.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Position
	for _, pos := range s.positions {
		if pos.Status == model.PositionStatusOpen {
			out = append(out, *pos)
		}
	}
	return out, nil
}

func (s *fakeStore) ListOpenPositionsByUser(_ context.Context, userID string) ([]model.Position, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.Position
	for _, pos := range s.positions {
		if pos.UserID == userID && pos.Status == model.PositionStatusOpen {
			out = append(out, *pos)
		}
	}
	return out, nil
}

func (s *fakeStore) ClosePosition(_ context.Context, pos *model.Position) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copyPos := *pos
	s.positions[pos.ID] = &copyPos
	return nil
}

func (s *fakeStore) ClaimExecution(_ context.Context, record *model.ExecutionRecord) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.executions[record.IdempotencyKey]; exists {
		return false, nil
	}
	s.executions[record.IdempotencyKey] = record
	return true, nil
}

func (s *fakeStore) CompleteExecution(_ context.Context, _ string, _ string) error { return nil }
func (s *fakeStore) FailExecution(_ context.Context, _ string, _ string) error     { return nil }

// reclaims mencatat kunci klaim yang berhasil diambil alih pada pengujian.
func (s *fakeStore) ReclaimStaleExecution(_ context.Context, key string, _ time.Time) (bool, error) {
	if s.staleClaims == nil {
		return false, nil
	}
	if _, ok := s.staleClaims[key]; !ok {
		return false, nil
	}
	delete(s.staleClaims, key)
	delete(s.executions, key)
	return true, nil
}

func (s *fakeStore) ListStaleExecutions(_ context.Context, _ time.Time) ([]model.ExecutionRecord, error) {
	out := make([]model.ExecutionRecord, 0, len(s.staleClaims))
	for key := range s.staleClaims {
		out = append(out, model.ExecutionRecord{IdempotencyKey: key, Status: "CLAIMED"})
	}
	return out, nil
}

func (s *fakeStore) FindExecution(_ context.Context, _ string) (*model.ExecutionRecord, error) {
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) GetDailySummary(_ context.Context, userID, day string) (*model.DailySummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if summary, ok := s.daily[dailyKey(userID, day)]; ok {
		copySummary := *summary
		return &copySummary, nil
	}
	return &model.DailySummary{UserID: userID, Day: day}, nil
}

func (s *fakeStore) IncrementDailyTrade(_ context.Context, userID, day string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	summary := s.dailyOrCreate(userID, day)
	summary.TotalTrades++
	return nil
}

func (s *fakeStore) RecordDailyOutcome(_ context.Context, userID, day string, won bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	summary := s.dailyOrCreate(userID, day)
	// Total trade dinaikkan saat eksekusi, bukan saat tutup; samakan dengan
	// perilaku repository sungguhan agar tes tidak memberi rasa aman palsu.
	if won {
		summary.WinCount++
	} else {
		summary.LossCount++
	}
	return nil
}

func (s *fakeStore) LockDaily(_ context.Context, userID, day, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	summary := s.dailyOrCreate(userID, day)
	now := time.Now().UTC()
	summary.LockedAt = &now
	summary.LockReason = reason
	return nil
}

func (s *fakeStore) dailyOrCreate(userID, day string) *model.DailySummary {
	key := dailyKey(userID, day)
	if existing, ok := s.daily[key]; ok {
		return existing
	}
	created := &model.DailySummary{UserID: userID, Day: day}
	s.daily[key] = created
	return created
}

func (s *fakeStore) ListBrokerAccountsByUser(_ context.Context, userID string) ([]model.BrokerAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.BrokerAccount
	for _, account := range s.accounts {
		if account.UserID == userID {
			out = append(out, *account)
		}
	}
	return out, nil
}

func (s *fakeStore) FindBrokerAccount(_ context.Context, accountID, userID string) (*model.BrokerAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if account, ok := s.accounts[accountID]; ok && account.UserID == userID {
		copyAccount := *account
		return &copyAccount, nil
	}
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) FindUserByID(_ context.Context, userID string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if user, ok := s.users[userID]; ok {
		copyUser := *user
		return &copyUser, nil
	}
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) ListActiveBrokerAccounts(_ context.Context) ([]model.BrokerAccount, error) {
	return nil, nil
}

func (s *fakeStore) UpdateBrokerAccount(_ context.Context, _ *model.BrokerAccount) error { return nil }
func (s *fakeStore) UpsertAccountState(_ context.Context, _ *model.BrokerAccountState) error {
	return nil
}

func (s *fakeStore) FindCredential(_ context.Context, _ string) (*model.Credential, error) {
	return nil, apperror.ErrNotFound
}

func (s *fakeStore) ListActiveSubscriptions(_ context.Context) ([]model.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]model.Subscription(nil), s.subs...), nil
}

func (s *fakeStore) ListUsersBySymbol(_ context.Context, symbol string) ([]model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []model.User
	for _, sub := range s.subs {
		if !sub.IsActive || sub.Symbol != symbol {
			continue
		}
		if user, ok := s.users[sub.UserID]; ok {
			out = append(out, *user)
		}
	}
	return out, nil
}

// ListAccountsBySymbol meniru kueri per akun: langganan tanpa akun berlaku
// untuk semua akun aktif user; yang punya akun hanya untuk akun itu.
func (s *fakeStore) ListAccountsBySymbol(_ context.Context, symbol string) ([]model.SubscriptionAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []model.SubscriptionAccount
	seen := make(map[string]bool)
	for _, sub := range s.subs {
		if !sub.IsActive || sub.Symbol != symbol {
			continue
		}
		user, ok := s.users[sub.UserID]
		if !ok {
			continue
		}
		for _, acc := range s.accounts {
			if acc.UserID != sub.UserID || !acc.IsActive {
				continue
			}
			if sub.BrokerAccountID != nil && *sub.BrokerAccountID != acc.ID {
				continue
			}
			if seen[acc.ID] {
				continue
			}
			seen[acc.ID] = true
			out = append(out, model.SubscriptionAccount{User: *user, Account: *acc})
		}
	}
	return out, nil
}

// EnabledAccountStrategies mengembalikan saklar metode yang menyala untuk satu
// simbol, dalam bentuk akun -> himpunan metode.
func (s *fakeStore) EnabledAccountStrategies(_ context.Context, symbol string) (map[string]map[string]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]map[string]bool{}
	prefix := symbol + "|"
	for key, metode := range s.switches {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		accountID := strings.TrimPrefix(key, prefix)
		out[accountID] = metode
	}
	return out, nil
}

// ClaimAccountLock meniru primary key akun: hanya satu sinyal yang boleh
// memegang kunci pada satu waktu.
func (s *fakeStore) ClaimAccountLock(_ context.Context, accountID, signalID string, _ time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ada := s.accountLocks[accountID]; ada {
		return false, nil
	}
	s.accountLocks[accountID] = signalID
	return true, nil
}

func (s *fakeStore) ReleaseAccountLock(_ context.Context, accountID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.accountLocks, accountID)
	return nil
}

func (s *fakeStore) ReleaseStaleAccountLock(_ context.Context, _ time.Time) error {
	// Fake store tidak menyimpan waktu klaim, jadi tidak ada yang kedaluwarsa.
	return nil
}

// ConfiguredAccounts melaporkan akun yang punya baris saklar untuk simbol ini,
// termasuk yang saklarnya mati.
func (s *fakeStore) ConfiguredAccounts(_ context.Context, symbol string) (map[string]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]bool{}
	prefix := symbol + "|"
	for key := range s.switches {
		if strings.HasPrefix(key, prefix) {
			out[strings.TrimPrefix(key, prefix)] = true
		}
	}
	return out, nil
}

// TimeframeSwitches meniru pembacaan saklar timeframe dari basis data.
func (s *fakeStore) TimeframeSwitches(_ context.Context, symbol string) (map[string]map[string]map[repository.TimeframeKey]repository.TimeframeState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := map[string]map[string]map[repository.TimeframeKey]repository.TimeframeState{}
	for key, state := range s.timeframes {
		parts := strings.Split(key, "|")
		if len(parts) != 5 {
			continue
		}
		accountID, sym, strategy, htf, ltf := parts[0], parts[1], parts[2], parts[3], parts[4]
		if sym != symbol {
			continue
		}
		if out[accountID] == nil {
			out[accountID] = map[string]map[repository.TimeframeKey]repository.TimeframeState{}
		}
		if out[accountID][strategy] == nil {
			out[accountID][strategy] = map[repository.TimeframeKey]repository.TimeframeState{}
		}
		out[accountID][strategy][repository.TimeframeKey{Htf: htf, Ltf: ltf}] = state
	}
	return out, nil
}

// EnabledPairSymbols mengembalikan seluruh simbol yang dikenal; fake store
// tidak punya katalog, jadi semua simbol dianggap aktif.
func (s *fakeStore) EnabledPairSymbols(_ context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := make(map[string]bool)
	var out []string
	for _, sub := range s.subs {
		if !seen[sub.Symbol] {
			seen[sub.Symbol] = true
			out = append(out, sub.Symbol)
		}
	}
	return out, nil
}

// CorrelationMap mengembalikan peta korelasi SMT yang dipasang tes.
func (s *fakeStore) CorrelationMap(_ context.Context) (map[string][]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]string, len(s.correlations))
	for symbol, list := range s.correlations {
		out[symbol] = append([]string(nil), list...)
	}
	return out, nil
}

// PairFeeds mengembalikan peta feed kosong: pengujian Runner tidak bergantung
// pada feed broker tertentu.
func (s *fakeStore) PairFeeds(context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *fakeStore) UpsertEngineState(_ context.Context, _ *model.EngineState) error { return nil }

func (s *fakeStore) ListEngines(_ context.Context) ([]model.EngineState, error) { return nil, nil }

func (s *fakeStore) FindDailySummary(_ context.Context, userID, day string) (*model.DailySummary, error) {
	return s.GetDailySummary(context.Background(), userID, day)
}

func (s *fakeStore) CreateApprovalToken(_ context.Context, _ *model.ApprovalToken) error { return nil }

// Pastikan fake memenuhi kontrak.
var _ repository.RunnerRepository = (*fakeStore)(nil)

// ---------------------------------------------------------------------------
// Publisher perekam
// ---------------------------------------------------------------------------

type recordedSignal struct {
	UserID   string
	SignalID string
	Status   string
	Reason   string
}

type fakePublisher struct {
	mu      sync.Mutex
	signals []recordedSignal
	closed  []recordedClose
}

type recordedClose struct {
	positionID string
	daily      *runner.DailyOutcome
}

func (p *fakePublisher) PublishSignal(_ context.Context, userID, signalID, status, reason string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.signals = append(p.signals, recordedSignal{userID, signalID, status, reason})
}

func (p *fakePublisher) PublishTelemetry(_ context.Context, _ string) {}

func (p *fakePublisher) PublishPositionClosed(_ context.Context, pos *model.Position, daily *runner.DailyOutcome) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pos == nil {
		return
	}
	p.closed = append(p.closed, recordedClose{positionID: pos.ID, daily: daily})
}

func (p *fakePublisher) last() (recordedSignal, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.signals) == 0 {
		return recordedSignal{}, false
	}
	return p.signals[len(p.signals)-1], true
}

// ---------------------------------------------------------------------------
// Bantuan penyusunan fixture
// ---------------------------------------------------------------------------

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newService(store *fakeStore, pub *fakePublisher) *runner.Service {
	// ILA 3.0 dianggap sudah umum agar perilaku tier bisa diuji.
	return newServiceWithSettings(store, pub, ila3General())
}

// newServiceWithSettings memungkinkan tes memilih tahap ketersediaan strategi.
func newServiceWithSettings(store *fakeStore, pub *fakePublisher, settings *runner.Settings) *runner.Service {
	registry := broker.NewRegistry()
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	return runner.NewService(store, registry, pub, discardLogger(), runner.Options{
		Policy:          runner.DefaultDailyPolicy(),
		SlippageGuard:   true,
		AutoExecute:     true,
		MarketOpenCheck: false,
		Settings:        settings,
	})
}

// ila3General menyalakan ILA 3.0 sampai tahap umum untuk pengujian.
func ila3General() *runner.Settings {
	return runner.NewSettings(runner.ILA3General)
}

func readySignal(symbol string, ttl time.Duration) *enginev1.Signal {
	published := time.Now().UTC()
	return &enginev1.Signal{
		SignalId:        "sig-" + symbol,
		Symbol:          symbol,
		Status:          enginev1.SignalStatus_READY_TO_ENTRY,
		Action:          enginev1.SignalAction_SIGNAL_ACTION_PUBLISH,
		Direction:       enginev1.Direction_BUY,
		EntryPrice:      4120.50,
		StopLoss:        4112.30,
		TakeProfit:      4149.20,
		RiskReward:      2.30,
		PublishedAt:     timestamppb.New(published),
		ExpiresAt:       timestamppb.New(published.Add(ttl)),
		StrategyVersion: "ila-2.0",
	}
}

// tambahAkun membuat user + akun aktif tanpa langganan, lalu menyalakan saklar
// metode sesuai daftar. Dipakai tes ComputeRequirements yang kini membaca
// metode dari saklar, bukan dari kolom langganan.
func tambahAkun(store *fakeStore, id, tier, symbol string, metode ...string) string {
	accountID := "acct-" + id
	store.users[id] = &model.User{Base: model.Base{ID: id}, Name: id, Tier: tier, Status: "active"}
	store.accounts[accountID] = &model.BrokerAccount{
		Base:        model.Base{ID: accountID},
		UserID:      id,
		BrokerType:  "paper",
		Mode:        "manual",
		IsActive:    true,
		IsDryRun:    true,
		RiskPercent: 0.5,
	}
	saklar := make(map[string]bool, len(metode))
	for _, m := range metode {
		saklar[m] = true
	}
	store.switches[symbol+"|"+accountID] = saklar
	return accountID
}

// addUser membuat user beserta akun dan langganan XAUUSD.
func addUser(store *fakeStore, id, tier, mode string, dryRun bool) {
	store.users[id] = &model.User{Base: model.Base{ID: id}, Name: id, Tier: tier, Status: "active"}
	store.accounts["acct-"+id] = &model.BrokerAccount{
		Base:        model.Base{ID: "acct-" + id},
		UserID:      id,
		BrokerType:  "paper",
		Mode:        mode,
		IsActive:    true,
		IsDryRun:    dryRun,
		RiskPercent: 0.5,
	}
	store.subs = append(store.subs, model.Subscription{
		UserID: id, Symbol: "XAUUSD", IsActive: true,
	})
}

// ---------------------------------------------------------------------------
// Tes alur
// ---------------------------------------------------------------------------

func TestAutoModeOpensPosition(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin 1", len(positions))
	}

	pos := positions[0]
	if pos.SourceSignalID != "sig-XAUUSD" {
		t.Fatalf("SourceSignalID = %q", pos.SourceSignalID)
	}
	if pos.LotSize <= 0 {
		t.Fatalf("LotSize = %v, ingin positif", pos.LotSize)
	}
	if pos.StopLoss != 4112.30 || pos.TakeProfit != 4149.20 {
		t.Fatalf("SL/TP tidak sesuai sinyal: %v/%v", pos.StopLoss, pos.TakeProfit)
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.Status != model.SignalStatusConsumed {
		t.Fatalf("status user = %q, ingin CONSUMED", us.Status)
	}

	// Kuota harian bertambah begitu order hidup.
	summary, _ := store.GetDailySummary(context.Background(), "u1", runner.DayKey(time.Now()))
	if summary.TotalTrades != 1 {
		t.Fatalf("TotalTrades = %d, ingin 1", summary.TotalTrades)
	}
}

func TestManualModeDoesNotExecute(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatalf("mode manual tidak boleh membuka posisi, dapat %d", len(positions))
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.Status != model.SignalStatusReadyToEntry {
		t.Fatalf("status user = %q, ingin READY_TO_ENTRY", us.Status)
	}

	last, ok := pub.last()
	if !ok || last.Status != model.SignalStatusReadyToEntry {
		t.Fatalf("notifikasi = %+v, ingin READY_TO_ENTRY", last)
	}
}

// Tier basic tidak boleh memakai mode auto walau akunnya disetel auto.
func TestAutoModeBlockedForBasicTier(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "auto", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatalf("basic tidak boleh auto, dapat %d posisi", len(positions))
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.Status != model.SignalStatusRejected {
		t.Fatalf("status user = %q, ingin REJECTED", us.Status)
	}
}

// Sinyal kedaluwarsa tidak boleh dieksekusi walau baru diterima.
func TestExpiredSignalIsRejected(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", -time.Minute))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatal("sinyal kedaluwarsa tidak boleh menghasilkan posisi")
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.Status != model.SignalStatusRejected {
		t.Fatalf("status = %q, ingin REJECTED", us.Status)
	}
	if us.RejectReason != string(runner.RejectExpired) {
		t.Fatalf("alasan = %q, ingin kedaluwarsa", us.RejectReason)
	}
}

// Sinyal yang sama tidak boleh melahirkan dua posisi.
func TestDuplicateSignalIsNotExecutedTwice(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newService(store, pub)
	signal := readySignal("XAUUSD", time.Hour)

	svc.HandleSignal(context.Background(), signal)
	svc.HandleSignal(context.Background(), signal)

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi = %d, ingin 1 (idempotensi)", len(positions))
	}
}

// Sinyal ditahan saat masih ada posisi terbuka, tetapi tetap dinotifikasi.
func TestSignalHeldWhenPositionOpen(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	second := readySignal("XAUUSD", time.Hour)
	second.SignalId = "sig-kedua"
	svc.HandleSignal(context.Background(), second)

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi = %d, ingin tetap 1", len(positions))
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-kedua")
	if us.RejectReason != string(runner.RejectOpenPosition) {
		t.Fatalf("alasan = %q, ingin posisi terbuka", us.RejectReason)
	}

	last, _ := pub.last()
	if last.Status != model.SignalStatusRejected {
		t.Fatalf("notifikasi = %q, ingin REJECTED", last.Status)
	}
}

// Setelah satu win, hari dikunci dan entry berikutnya ditahan.
func TestStopAfterWin(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	day := runner.DayKey(time.Now())
	_ = store.RecordDailyOutcome(context.Background(), "u1", day, true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatal("setelah win tidak boleh ada entry baru")
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.RejectReason != string(runner.RejectDailyWin) {
		t.Fatalf("alasan = %q, ingin target harian", us.RejectReason)
	}
}

// Dua SL menutup hari.
func TestTwoStrikeOut(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	day := runner.DayKey(time.Now())
	_ = store.RecordDailyOutcome(context.Background(), "u1", day, false)
	_ = store.RecordDailyOutcome(context.Background(), "u1", day, false)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.RejectReason != string(runner.RejectTwoStrike) {
		t.Fatalf("alasan = %q, ingin 2x SL", us.RejectReason)
	}
}

// Kuota dua trade menutup hari.
func TestDailyQuotaExhausted(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	day := runner.DayKey(time.Now())
	_ = store.IncrementDailyTrade(context.Background(), "u1", day)
	_ = store.IncrementDailyTrade(context.Background(), "u1", day)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.RejectReason != string(runner.RejectDailyQuota) {
		t.Fatalf("alasan = %q, ingin kuota habis", us.RejectReason)
	}
}

// User Free tidak punya akun nyata: sinyal tampil, tidak ada order.
func TestFreeTierNeverExecutes(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "free", "auto", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatal("tier free tidak boleh mengeksekusi")
	}
}

// Masa tenggang memblokir entry baru.
func TestGracePeriodBlocksEntry(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	past := time.Now().Add(-time.Hour)
	store.users["u1"].GraceUntil = &past

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatal("masa tenggang harus memblokir entry")
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.RejectReason != string(runner.RejectSuspended) {
		t.Fatalf("alasan = %q, ingin suspended", us.RejectReason)
	}
}

// Akun dibekukan tidak boleh menerima order.
func TestFrozenAccountSkipped(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	store.accounts["acct-u1"].IsFrozen = true

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatal("akun beku tidak boleh mengeksekusi")
	}
}

// CANCELLED tidak boleh menyentuh posisi yang sudah terbuka.
func TestCancelDoesNotTouchExistingPosition(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positionsBefore, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positionsBefore) != 1 {
		t.Fatal("posisi belum terbuka")
	}

	svc.HandleUpdate(context.Background(), &enginev1.SignalUpdate{
		SignalId: "sig-XAUUSD",
		Action:   enginev1.SignalAction_SIGNAL_ACTION_CANCEL,
		Reason:   "struktur batal",
	})

	positionsAfter, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positionsAfter) != 1 {
		t.Fatal("posisi terbuka tidak boleh dibatalkan")
	}
	if positionsAfter[0].Status != model.PositionStatusOpen {
		t.Fatalf("status posisi = %q, ingin OPEN", positionsAfter[0].Status)
	}

	// Sinyal global tetap batal, karena itu status usulan, bukan fakta.
	sig, _ := store.FindSignal(context.Background(), "sig-XAUUSD")
	if sig.Status != model.SignalStatusCancelled {
		t.Fatalf("status sinyal = %q, ingin CANCELLED", sig.Status)
	}
}

// CANCELLED menandai batal untuk user yang belum entry.
func TestCancelMarksPendingSignal(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	svc.HandleUpdate(context.Background(), &enginev1.SignalUpdate{
		SignalId: "sig-XAUUSD",
		Action:   enginev1.SignalAction_SIGNAL_ACTION_CANCEL,
		Reason:   "struktur batal",
	})

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.Status != model.SignalStatusCancelled {
		t.Fatalf("status user = %q, ingin CANCELLED", us.Status)
	}
}

// Status terminal tidak boleh dihidupkan kembali.
func TestTerminalSignalNotRevived(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)

	svc := newService(store, pub)
	signal := readySignal("XAUUSD", time.Hour)
	svc.HandleSignal(context.Background(), signal)

	svc.HandleUpdate(context.Background(), &enginev1.SignalUpdate{
		SignalId: "sig-XAUUSD",
		Action:   enginev1.SignalAction_SIGNAL_ACTION_EXPIRE,
		Reason:   "lewat TTL",
	})

	svc.HandleSignal(context.Background(), signal)

	sig, _ := store.FindSignal(context.Background(), "sig-XAUUSD")
	if sig.Status != model.SignalStatusExpired {
		t.Fatalf("status sinyal = %q, ingin tetap EXPIRED", sig.Status)
	}
}

// Approve mengeksekusi sinyal pending milik user.
func TestApproveExecutesPendingSignal(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	reason, err := svc.Approve(context.Background(), "u1", "sig-XAUUSD", "acct-u1")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if reason != runner.RejectNone {
		t.Fatalf("Approve ditolak: %q", reason)
	}

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi = %d, ingin 1", len(positions))
	}
}

// Approve atas sinyal kedaluwarsa ditolak.
func TestApproveRejectsExpiredSignal(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", -time.Minute))

	reason, err := svc.Approve(context.Background(), "u1", "sig-XAUUSD", "acct-u1")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if reason != runner.RejectExpired {
		t.Fatalf("alasan = %q, ingin kedaluwarsa", reason)
	}
}

// Approve tidak boleh memakai akun milik user lain.
func TestApproveRejectsForeignAccount(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)
	addUser(store, "u2", "basic", "manual", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	reason, err := svc.Approve(context.Background(), "u1", "sig-XAUUSD", "acct-u2")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if reason != runner.RejectAccountFrozen {
		t.Fatalf("alasan = %q, ingin akun tidak sah", reason)
	}
}

// Skip tidak memakai kuota harian.
func TestSkipDoesNotConsumeQuota(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	if err := svc.Skip(context.Background(), "u1", "sig-XAUUSD"); err != nil {
		t.Fatalf("Skip: %v", err)
	}

	summary, _ := store.GetDailySummary(context.Background(), "u1", runner.DayKey(time.Now()))
	if summary.TotalTrades != 0 {
		t.Fatalf("TotalTrades = %d, ingin 0", summary.TotalTrades)
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.Status != model.SignalStatusSkipped {
		t.Fatalf("status = %q, ingin SKIPPED", us.Status)
	}
}

// Satu sinyal untuk banyak user: masing-masing dinilai terpisah.
func TestSignalFannedOutPerUser(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	addUser(store, "u1", "plus", "auto", true)
	store.users["u2"] = &model.User{Base: model.Base{ID: "u2"}, Name: "u2", Tier: "basic", Status: "active"}
	store.accounts["acct-u2"] = &model.BrokerAccount{
		Base: model.Base{ID: "acct-u2"}, UserID: "u2", BrokerType: "paper",
		Mode: "manual", IsActive: true, IsDryRun: true, RiskPercent: 0.5,
	}
	store.subs = append(store.subs, model.Subscription{
		UserID: "u2", Symbol: "XAUUSD", IsActive: true,
	})

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	// User A dieksekusi otomatis.
	a, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if a.Status != model.SignalStatusConsumed {
		t.Fatalf("u1 = %q, ingin CONSUMED", a.Status)
	}

	// User B menunggu konfirmasi.
	b, _ := store.FindUserSignal(context.Background(), "u2", "sig-XAUUSD")
	if b.Status != model.SignalStatusReadyToEntry {
		t.Fatalf("u2 = %q, ingin READY_TO_ENTRY", b.Status)
	}
}

// Sinyal untuk simbol yang tidak dipantau tidak menghasilkan apa pun.
func TestSignalForUnmonitoredSymbolIgnoredPerUser(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newService(store, pub)
	signal := readySignal("EURUSD", time.Hour)
	svc.HandleSignal(context.Background(), signal)

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatal("simbol tidak dipantau tidak boleh dieksekusi")
	}
}

// ComputeRequirements menerjemahkan user menjadi kebutuhan (symbol, strategy).
func TestComputeRequirements(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	addUser(store, "u1", "basic", "manual", true)
	addUser(store, "u2", "basic", "manual", true)
	// u3 Plus memantau EURUSD dengan ILA 3.0 menyala.
	tambahAkun(store, "u3", "plus", "EURUSD", "ila-2.0", "ila-3.0")
	store.subs = append(store.subs, model.Subscription{
		UserID: "u3", Symbol: "EURUSD", IsActive: true,
	})

	svc := newService(store, pub)
	reqs, err := svc.ComputeRequirements(context.Background(), []runner.StrategyLocation{
		{Strategy: "ila-2.0", Addr: "ila2:9090"},
		{Strategy: "ila-3.0", Addr: "ila3:9090"},
	})
	if err != nil {
		t.Fatalf("ComputeRequirements: %v", err)
	}

	// Dua user Basic memantau XAUUSD tetap menghasilkan satu entri, sementara
	// u3 Plus menambah EURUSD ke kedua metode yang dinyalakannya.
	if got := reqs["ila-2.0"]; strings.Join(got, ",") != "EURUSD,XAUUSD" {
		t.Fatalf("ila-2.0 = %v, ingin [EURUSD XAUUSD]", got)
	}
	if got := reqs["ila-3.0"]; len(got) != 1 || got[0] != "EURUSD" {
		t.Fatalf("ila-3.0 = %v, ingin [EURUSD]", got)
	}
}

// User Basic tidak boleh memicu perhitungan ILA 3.0 walau saklarnya diminta
// menyala. Hak tier tetap menang atas pilihan akun.
func TestComputeRequirementsBlocksIla3ForBasic(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}

	addUser(store, "u1", "basic", "manual", true)
	// Akun Basic mencoba menyalakan ILA 3.0 untuk XAUUSD.
	store.switches["XAUUSD|acct-u1"] = map[string]bool{"ila-2.0": true, "ila-3.0": true}

	svc := newService(store, pub)
	reqs, err := svc.ComputeRequirements(context.Background(), []runner.StrategyLocation{
		{Strategy: "ila-3.0", Addr: "ila3:9090"},
	})
	if err != nil {
		t.Fatalf("ComputeRequirements: %v", err)
	}
	if _, ok := reqs["ila-3.0"]; ok {
		t.Fatal("basic tidak boleh memicu ILA 3.0")
	}
}

// ---------------------------------------------------------------------------
// Klaim eksekusi
// ---------------------------------------------------------------------------

// TestKlaimTertinggalDiambilAlih menjaga sinyal tidak hilang diam-diam ketika
// proses berhenti setelah klaim tetapi sebelum order terkirim.
func TestKlaimTertinggalDiambilAlih(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	signal := readySignal("XAUUSD", time.Hour)
	key := runner.IdempotencyKey(signal.SignalId, "acct-u1")

	// Klaim lama tertinggal; tanpa masa tenggang, sinyal ini akan dilewati.
	store.staleClaims = map[string]struct{}{key: {}}

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), signal)

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin 1 setelah klaim diambil alih", len(positions))
	}
}

// TestKlaimBaruTidakDiambilAlih memastikan eksekusi ganda tetap dicegah.
func TestKlaimBaruTidakDiambilAlih(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	signal := readySignal("XAUUSD", time.Hour)
	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), signal)

	// Kiriman ulang sinyal yang sama tidak boleh membuka posisi kedua.
	store.staleClaims = nil
	svc.HandleSignal(context.Background(), signal)

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin 1 karena kunci idempotensi", len(positions))
	}
}
