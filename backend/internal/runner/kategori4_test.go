package runner_test

import (
	"context"
	"math"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/broker"
	"ila/backend/internal/market"
	"ila/backend/internal/model"
	"ila/backend/internal/runner"
	"ila/backend/pkg/cryptox"
)

func sinyalSiapPada(id, symbol string, published time.Time) *enginev1.Signal {
	return &enginev1.Signal{
		SignalId:        id,
		Symbol:          symbol,
		Htf:             "H1",
		Ltf:             "M5",
		Status:          enginev1.SignalStatus_READY_TO_ENTRY,
		Action:          enginev1.SignalAction_SIGNAL_ACTION_PUBLISH,
		Direction:       enginev1.Direction_BUY,
		EntryPrice:      4120.00,
		StopLoss:        4112.00, // jarak SL = 8.00 -> $800/lot di XAUUSD
		TakeProfit:      4136.00, // jarak TP = 16.00 (2R)
		RiskReward:      2.00,
		PublishedAt:     timestamppb.New(published),
		ExpiresAt:       timestamppb.New(published.Add(30 * time.Minute)),
		StrategyVersion: "ila-2.0",
	}
}

func TestRiskModeLotMemakaiLotSizePilihanPengguna(t *testing.T) {
	waktuPasar := time.Date(2026, 11, 17, 15, 0, 0, 0, time.UTC) // Selasa 10:00 NY

	// 1. Mode "lot" memakai LotSize=0.04 apa adanya, bukan menghitung dari RiskPercent.
	storeLot := newFakeStore()
	pubLot := &fakePublisher{}
	storeLot.users["u-lot"] = &model.User{Base: model.Base{ID: "u-lot"}, Name: "Lot", Tier: "plus", Status: "active"}
	storeLot.accounts["acct-lot"] = &model.BrokerAccount{
		Base:        model.Base{ID: "acct-lot"},
		UserID:      "u-lot",
		BrokerType:  "paper",
		Mode:        "auto",
		RiskMode:    model.RiskModeLot,
		RiskPercent: 0.5,
		LotSize:     0.04,
		IsActive:    true,
		IsDryRun:    true,
	}
	storeLot.subs = append(storeLot.subs, model.Subscription{UserID: "u-lot", Symbol: "XAUUSD", IsActive: true})

	svcLot := newService(storeLot, pubLot)
	svcLot.SetClock(func() time.Time { return waktuPasar })
	svcLot.HandleSignal(context.Background(), sinyalSiapPada("sig-lot-1", "XAUUSD", waktuPasar))

	posLot, _ := storeLot.ListOpenPositionsByUser(context.Background(), "u-lot")
	if len(posLot) != 1 {
		t.Fatalf("posisi terbuka mode lot = %d, ingin 1", len(posLot))
	}
	if posLot[0].LotSize != 0.04 {
		t.Fatalf("LotSize mode lot = %v, ingin 0.04", posLot[0].LotSize)
	}

	// 2. Mode "percent" (bawaan) tetap identik seperti semula: $10.000 * 0.5% = $50 / $800 = 0.06 lot.
	storePct := newFakeStore()
	pubPct := &fakePublisher{}
	addUser(storePct, "u-pct", "plus", "auto", true)

	svcPct := newService(storePct, pubPct)
	svcPct.SetClock(func() time.Time { return waktuPasar })
	svcPct.HandleSignal(context.Background(), sinyalSiapPada("sig-pct-1", "XAUUSD", waktuPasar))

	posPct, _ := storePct.ListOpenPositionsByUser(context.Background(), "u-pct")
	if len(posPct) != 1 || posPct[0].LotSize != 0.06 {
		t.Fatalf("posisi mode percent berubah: %+v (ingin 0.06 lot)", posPct)
	}
}

type tinyEquityBroker struct {
	equity float64
}

func (b *tinyEquityBroker) Platform() string                                  { return "tiny" }
func (b *tinyEquityBroker) AccountID() string                                 { return "tiny-1" }
func (b *tinyEquityBroker) Connect(context.Context, broker.Credentials) error { return nil }
func (b *tinyEquityBroker) Connected() bool                                   { return true }
func (b *tinyEquityBroker) Close() error                                      { return nil }
func (b *tinyEquityBroker) Account(context.Context) (broker.AccountSnapshot, error) {
	return broker.AccountSnapshot{AccountID: "tiny-1", Currency: "USD", Balance: b.equity, Equity: b.equity, FreeMargin: b.equity}, nil
}
func (b *tinyEquityBroker) Symbol(_ context.Context, sym string) (broker.SymbolSpec, error) {
	return broker.SymbolSpec{Symbol: sym, Digits: 2, PipSize: 0.01, ValuePerLot: 100, MinLot: 0.01, MaxLot: 50, LotStep: 0.01, ContractSize: 100}, nil
}
func (b *tinyEquityBroker) CurrentPrice(context.Context, string) (float64, error) {
	return 4120.00, nil
}
func (b *tinyEquityBroker) OpenPosition(_ context.Context, req broker.OpenRequest) (broker.OpenResult, error) {
	return broker.OpenResult{Ticket: "T-1", Lots: req.Lots, ExecutedPrice: 4120.00, StopLoss: req.StopLoss, TakeProfit: req.TakeProfit}, nil
}
func (b *tinyEquityBroker) ListPositions(context.Context) ([]broker.PositionState, error) {
	return nil, nil
}
func (b *tinyEquityBroker) ClosedSince(context.Context, int64) ([]broker.ClosedPosition, error) {
	return nil, nil
}

type storeWithCred struct {
	*fakeStore
	cred *model.Credential
}

func (s *storeWithCred) FindCredential(ctx context.Context, accountID string) (*model.Credential, error) {
	if s.cred != nil && s.cred.BrokerAccountID == accountID {
		return s.cred, nil
	}
	return s.fakeStore.FindCredential(ctx, accountID)
}

func TestClampedMinBerlebihanDitolakDemiKeamananModalKecil(t *testing.T) {
	waktuPasar := time.Date(2026, 11, 17, 15, 0, 0, 0, time.UTC)

	// Akun $200, risiko 0.5% ($1), SL XAUUSD $8 -> 0.01 lot berisiko $8 (4% ekuitas > batas 2%).
	baseStore := newFakeStore()
	pub := &fakePublisher{}
	baseStore.users["u-kecil"] = &model.User{Base: model.Base{ID: "u-kecil"}, Name: "Kecil", Tier: "plus", Status: "active"}
	baseStore.accounts["acct-kecil"] = &model.BrokerAccount{
		Base:        model.Base{ID: "acct-kecil"},
		UserID:      "u-kecil",
		BrokerType:  "tiny",
		Mode:        "auto",
		RiskMode:    model.RiskModePercent,
		RiskPercent: 0.5,
		IsActive:    true,
		IsDryRun:    false,
	}
	baseStore.subs = append(baseStore.subs, model.Subscription{UserID: "u-kecil", Symbol: "XAUUSD", IsActive: true})

	keyring, err := cryptox.NewKeyring(1, "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", nil)
	if err != nil {
		t.Fatalf("NewKeyring: %v", err)
	}
	rawCred, err := runner.EncodeCredentials(broker.Credentials{Platform: "tiny", AccountID: "acct-kecil", Token: "tok"})
	if err != nil {
		t.Fatalf("EncodeCredentials: %v", err)
	}
	env, err := keyring.Seal(rawCred, []byte("acct-kecil"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	store := &storeWithCred{
		fakeStore: baseStore,
		cred: &model.Credential{
			BrokerAccountID: "acct-kecil",
			UserID:          "u-kecil",
			UserKeyWrapped:  env.UserKeyWrapped,
			UserKeyNonce:    env.UserKeyNonce,
			Ciphertext:      env.Ciphertext,
			Nonce:           env.Nonce,
			KeyVersion:      env.KeyVersion,
		},
	}

	reg := broker.NewRegistry()
	reg.Register("tiny", func(_, _ string) (broker.Client, error) {
		return &tinyEquityBroker{equity: 200}, nil
	})
	svc := runner.NewService(store, reg, pub, discardLogger(), runner.Options{
		Policy:        runner.DefaultDailyPolicy(),
		AutoExecute:   true,
		SlippageGuard: false,
		Clock:         func() time.Time { return waktuPasar },
		Settings:      ila3General(),
	}).WithKeyring(keyring)

	svc.HandleSignal(context.Background(), sinyalSiapPada("sig-kecil", "XAUUSD", waktuPasar))

	pos, _ := store.ListOpenPositionsByUser(context.Background(), "u-kecil")
	if len(pos) != 0 {
		t.Fatalf("akun modal $200 tidak boleh dipaksa membuka 0.01 lot berisiko 4%%: %+v", pos)
	}
	us, err := store.FindUserSignal(context.Background(), "u-kecil", "sig-kecil")
	if err != nil || us.RejectReason != string(runner.RejectLotTooSmall) {
		t.Fatalf("alasan penolakan = %q (err=%v), ingin %q", us.RejectReason, err, runner.RejectLotTooSmall)
	}
}

func TestMonitorDryRunPositionsMenutupSimulasiSaatTP(t *testing.T) {
	waktuBuka := time.Date(2026, 11, 17, 15, 0, 0, 0, time.UTC)
	waktuCek := waktuBuka.Add(15 * time.Minute)
	sekarang := waktuBuka

	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u-sim", "plus", "auto", true)

	reg := broker.NewRegistry()
	reg.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	svc := runner.NewService(store, reg, pub, discardLogger(), runner.Options{
		Policy:        runner.DefaultDailyPolicy(),
		AutoExecute:   true,
		SlippageGuard: false,
		Clock:         func() time.Time { return sekarang },
		Settings:      ila3General(),
		CandleFetcher: func(_ context.Context, symbol, _, timeframe string, _ int) ([]market.Candle, error) {
			if symbol != "XAUUSD" || timeframe != "M5" {
				return nil, nil
			}
			return []market.Candle{
				{
					Time:  waktuBuka.Add(5 * time.Minute).Unix(),
					Open:  4122,
					High:  4138, // menembus TP 4136
					Low:   4120, // tidak menyentuh SL 4112
					Close: 4137,
				},
			}, nil
		},
	})

	svc.HandleSignal(context.Background(), sinyalSiapPada("sig-sim-1", "XAUUSD", waktuBuka))
	terbuka, _ := store.ListOpenPositionsByUser(context.Background(), "u-sim")
	if len(terbuka) != 1 {
		t.Fatalf("posisi simulasi belum terbuka: %d", len(terbuka))
	}

	sekarang = waktuCek
	svc.MonitorDryRunPositions(context.Background())

	terbukaSesudah, _ := store.ListOpenPositionsByUser(context.Background(), "u-sim")
	if len(terbukaSesudah) != 0 {
		t.Fatalf("posisi simulasi masih menggantung OPEN sesudah TP tersentuh: %+v", terbukaSesudah)
	}

	posSim := store.positions[terbuka[0].ID]
	if posSim == nil || posSim.Status != model.PositionStatusClosedTP {
		t.Fatalf("status akhir simulasi = %+v, ingin %q", posSim, model.PositionStatusClosedTP)
	}
	if math.Abs(posSim.OutcomeR-2.0) > 0.05 {
		t.Fatalf("OutcomeR simulasi = %v, ingin ~2.0", posSim.OutcomeR)
	}
}

func TestCancelDiniAtauAsyncMenahanEksekusiPublish(t *testing.T) {
	waktuPasar := time.Date(2026, 11, 17, 15, 0, 0, 0, time.UTC)

	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u-race", "plus", "auto", true)

	svc := newService(store, pub)
	svc.SetClock(func() time.Time { return waktuPasar })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.RunSignalWorkers(ctx)

	// Skenario terburuk: CANCEL tiba lebih dulu daripada PUBLISH untuk signal_id yang sama.
	svc.HandleUpdateAsync(ctx, &enginev1.SignalUpdate{
		SignalId: "sig-race-1",
		Action:   enginev1.SignalAction_SIGNAL_ACTION_CANCEL,
		Reason:   "setup dibatalkan engine",
	})
	svc.HandleSignalAsync(ctx, sinyalSiapPada("sig-race-1", "XAUUSD", waktuPasar))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		sig, err := store.FindSignal(ctx, "sig-race-1")
		if err == nil && sig.Status == model.SignalStatusCancelled {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	pos, _ := store.ListOpenPositionsByUser(ctx, "u-race")
	if len(pos) != 0 {
		t.Fatalf("sinyal yang sudah dibatalkan tidak boleh membuka posisi: %+v", pos)
	}
	sig, err := store.FindSignal(ctx, "sig-race-1")
	if err != nil || sig.Status != model.SignalStatusCancelled {
		t.Fatalf("status sinyal akhir = %+v (err=%v), ingin CANCELLED", sig, err)
	}
}
