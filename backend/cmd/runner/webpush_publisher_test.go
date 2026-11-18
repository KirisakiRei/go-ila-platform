package main

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/runner/pushnotify"
	"ila/backend/internal/service"
	"ila/backend/internal/webpush"
)

func TestWebPushPublisherStrictEventFiltering(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Signal{},
		&model.Position{},
		&model.ApprovalToken{},
		&model.NotificationLog{},
		&model.TelegramMessage{},
		&model.UserSignal{},
		&model.SignalEvent{},
		&model.WebPushVAPID{},
		&model.WebPushSubscription{},
	); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	var receivedCount atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		receivedCount.Add(1)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	pushRepo := repository.NewWebPushRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	wpClient := webpush.NewClient(5 * time.Second).WithHTTPClient(srv.Client())
	wpSvc := service.NewWebPushService(service.WebPushServiceDeps{
		Repo:          pushRepo,
		Client:        wpClient,
		Subject:       "mailto:test@rekavia.com",
		SkipHostCheck: true,
	})

	uaPriv, _ := ecdh.P256().GenerateKey(rand.Reader)
	authSecret := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, authSecret)

	ctx := context.Background()
	userID := "user-push-1"
	_, err = wpSvc.Subscribe(ctx, userID, "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)", dto.PushSubscribeRequest{
		Endpoint: srv.URL + "/push/dev1",
		Keys: dto.PushKeysPayload{
			P256dh: base64.RawURLEncoding.EncodeToString(uaPriv.PublicKey().Bytes()),
			Auth:   base64.RawURLEncoding.EncodeToString(authSecret),
		},
	})
	if err != nil {
		t.Fatalf("Subscribe gagal: %v", err)
	}

	sig := &model.Signal{
		SignalID:        "sig-xau-1",
		Symbol:          "XAUUSD",
		StrategyVersion: "ila-2.0",
		Direction:       "BUY",
		Htf:             "H4",
		Ltf:             "M15",
		EntryPrice:      2650.50,
		StopLoss:        2640.00,
		TakeProfit:      2671.50,
		RiskReward:      2.0,
		Status:          model.SignalStatusReadyToEntry,
		PublishedAt:     time.Now().UTC(),
		ExpiresAt:       time.Now().UTC().Add(5 * time.Minute),
	}
	if err := db.Create(sig).Error; err != nil {
		t.Fatalf("create signal: %v", err)
	}

	pushWorker := pushnotify.New(pushRepo, notifRepo, wpSvc, wpClient, slog.Default(), 1).
		WithSkipHostAllowlist(true)

	pub := &runnerPublisher{
		store:   repository.NewRunnerRepository(db),
		push:    pushWorker,
		baseURL: "https://rekavia.com",
		logger:  slog.Default(),
		jejak:   repository.NewTelegramMessageRepository(db),
	}

	// 1. Tahap radar & pembatalan TIDAK BOLEH memicu Web Push sama sekali!
	ignoredStatuses := []string{
		model.SignalStatusWaitHTFSweep,
		model.SignalStatusWaitHTFCandleClose,
		model.SignalStatusWaitLTFCISD,
		model.SignalStatusRejected,
		model.SignalStatusCancelled,
		model.SignalStatusExpired,
		model.SignalStatusSuperseded,
		model.SignalStatusSkipped,
	}
	for _, st := range ignoredStatuses {
		pub.PublishSignal(ctx, userID, sig.SignalID, st, "")
	}
	// Ready dengan RejectEntitlement (mis. tier Free) juga TIDAK BOLEH memicu Web Push!
	pub.PublishSignal(ctx, userID, sig.SignalID, model.SignalStatusReadyToEntry, string(runner.RejectEntitlement))

	// 2. READY_TO_ENTRY sah -> +1 Web Push
	pub.PublishSignal(ctx, userID, sig.SignalID, model.SignalStatusReadyToEntry, "")

	// 3. Posisi baru dibuka (Auto) -> +1 Web Push
	posOpen := &model.Position{
		Base:       model.Base{ID: "pos-1"},
		UserID:     userID,
		Symbol:     "XAUUSD",
		Direction:  "BUY",
		LotSize:    0.10,
		OpenPrice:  2650.55,
		StopLoss:   2640.00,
		TakeProfit: 2671.50,
		Status:     model.PositionStatusOpen,
	}
	pub.PublishPositionOpened(ctx, posOpen, sig, true)

	// 4. Posisi ditutup TP, SL, dan Manual -> masing-masing +1 Web Push (UNTRACKED diabaikan!)
	pub.PublishPositionClosed(ctx, &model.Position{
		Base: model.Base{ID: "pos-tp"}, UserID: userID, Symbol: "XAUUSD", Status: model.PositionStatusClosedTP,
		ProfitLoss: 210, OutcomeR: 2.0, ExitPrice: 2671.50,
	}, nil)
	pub.PublishPositionClosed(ctx, &model.Position{
		Base: model.Base{ID: "pos-sl"}, UserID: userID, Symbol: "XAUUSD", Status: model.PositionStatusClosedSL,
		ProfitLoss: -105, OutcomeR: -1.0, ExitPrice: 2640.00,
	}, nil)
	pub.PublishPositionClosed(ctx, &model.Position{
		Base: model.Base{ID: "pos-manual"}, UserID: userID, Symbol: "XAUUSD", Status: model.PositionStatusClosedManual,
		ProfitLoss: 45, OutcomeR: 0.4, ExitPrice: 2655.00,
	}, nil)
	pub.PublishPositionClosed(ctx, &model.Position{
		Base: model.Base{ID: "pos-untracked"}, UserID: userID, Symbol: "XAUUSD", Status: "UNTRACKED",
	}, nil)

	pushWorker.Close()

	if got := receivedCount.Load(); got != 5 {
		t.Fatalf("jumlah Web Push terkirim = %d, ingin tepat 5 (1 ready + 1 open + 1 TP + 1 SL + 1 manual)", got)
	}
}
