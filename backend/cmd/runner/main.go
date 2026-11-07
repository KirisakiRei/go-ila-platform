package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "github.com/KirisakiRei/go-ila-platform/backend/gen/engine/v1"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/broker"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/config"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/engineclient"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/repository"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/runner"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/runner/notify"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/telegram"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/cryptox"
)

// batas percobaan koneksi ulang ke engine.
const (
	minBackoff = 1 * time.Second
	maxBackoff = 30 * time.Second
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel()}))
	slog.SetDefault(logger)

	cfg, err := config.Load("runner")
	if err != nil {
		logger.Error("konfigurasi tidak sah", "error", err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		logger.Error("database tidak siap", "error", err.Error())
		os.Exit(1)
	}

	store := repository.NewRunnerRepository(db)

	// --- Kanal notifikasi -----------------------------------------------------
	tgClient := telegram.NewClient(telegram.Config{
		BotToken: cfg.TelegramBotToken,
		Timeout:  cfg.TelegramTimeout,
	}, logger)

	notifier := notify.New(store, repository.NewNotificationRepository(db), tgClient, logger, 2)
	defer notifier.Close()

	publisher := &runnerPublisher{
		store:    store,
		notifier: notifier,
		baseURL:  cfg.PublicURL,
		logger:   logger,
	}

	// --- Broker ---------------------------------------------------------------
	registry := broker.NewRegistry()
	// Simulasi tersedia untuk semua akun dry-run dan seluruh tier.
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	// --- Runner ---------------------------------------------------------------
	opts := runner.Options{
		Policy: runner.DailyPolicy{
			MaxTrades:    cfg.MaxDailyTrades,
			StopAfterWin: cfg.StopAfterWin,
			MaxLosses:    cfg.MaxDailyLosses,
		}.Clamp(),
		SlippageGuard:   true,
		DryRunAll:       cfg.DryRunAll,
		AutoExecute:     cfg.AutoExecute,
		MarketOpenCheck: true,
	}

	svc := runner.NewService(store, registry, publisher, logger, opts)

	if cfg.MasterKey != "" {
		keyring, keyErr := cryptox.NewKeyring(cfg.MasterKeyVersion, cfg.MasterKey, nil)
		if keyErr != nil {
			logger.Error("master key tidak sah", "error", keyErr.Error())
			os.Exit(1)
		}
		svc.WithKeyring(keyring)
	} else if !cfg.DryRunAll {
		logger.Warn("MASTER_KEY belum diset; kredensial broker tidak bisa dibuka")
	}

	logger.Info("runner mulai",
		"env", cfg.AppEnv,
		"engine_addr", cfg.EngineGRPCAddr,
		"daily_tz", cfg.DailyTimezone,
		"max_daily_trades", opts.Policy.MaxTrades,
		"auto_execute", opts.AutoExecute,
		"dry_run_all", opts.DryRunAll,
	)

	// Rekonsiliasi jalan sebelum sinyal baru diterima, agar selisih posisi
	// ditemukan saat sistem masih tenang.
	svc.Reconcile(ctx)

	var wg sync.WaitGroup

	// Pemantau posisi: selalu jalan walau engine belum tersambung, karena
	// posisi user tetap butuh pelaporan.
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMonitor(ctx, svc, logger)
	}()

	// Koneksi engine.
	if cfg.EngineGRPCAddr != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runEngineLoop(ctx, cfg, svc, store, registry, logger)
		}()
	} else {
		logger.Warn("ENGINE_GRPC_ADDR kosong; runner berjalan tanpa sinyal engine")
	}

	// Server internal: hanya backend-api yang memanggilnya.
	internalSrv := newInternalServer(svc, cfg.InternalSecret, logger)
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("runner HTTP internal berjalan", "addr", internalSrv.Addr)
		if err := internalSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server internal berhenti tak terduga", "error", err.Error())
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("menghentikan runner")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()
	if err := internalSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("gagal menghentikan server internal", "error", err.Error())
	}

	cancel()
	wg.Wait()
	svc.CloseClients()

	logger.Info("runner berhenti")
}

// runMonitor memeriksa posisi terbuka secara berkala.
func runMonitor(ctx context.Context, svc *runner.Service, logger *slog.Logger) {
	ticker := time.NewTicker(runner.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			svc.MonitorPositions(ctx)
		}
	}
}

// runEngineLoop menjaga koneksi ke engine tetap hidup.
//
// Pola reconnect mengikuti plans/05 §8: setelah terhubung, minta snapshot,
// lalu berlangganan ulang untuk semua simbol kebutuhan. Sinyal lama yang
// terlewat TIDAK dieksekusi karena pemeriksaan TTL berjalan di jalur sinyal.
func runEngineLoop(
	ctx context.Context,
	cfg *config.Config,
	svc *runner.Service,
	store repository.RunnerRepository,
	registry *broker.Registry,
	logger *slog.Logger,
) {
	backoff := minBackoff
	stateID := engineStateID(cfg.EngineStrategy)

	for ctx.Err() == nil {
		err := streamOnce(ctx, cfg, svc, store, logger)
		recordEngineState(ctx, store, stateID, cfg.EngineStrategy, err, logger)

		if ctx.Err() != nil {
			return
		}

		logger.Warn("koneksi engine terputus, mencoba ulang",
			"backoff", backoff.String(), "error", errorText(err))

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// streamOnce membuka satu sesi ke engine sampai terputus.
func streamOnce(
	ctx context.Context,
	cfg *config.Config,
	svc *runner.Service,
	store repository.RunnerRepository,
	logger *slog.Logger,
) error {
	conn, err := engineclient.NewConnection(engineclient.Config{
		Addr:               cfg.EngineGRPCAddr,
		Strategy:           cfg.EngineStrategy,
		SubscriberID:       cfg.RunnerSubscriber,
		CertFile:           cfg.EngineMTLSCert,
		KeyFile:            cfg.EngineMTLSKey,
		CAFile:             cfg.EngineMTLSCA,
		ServerNameOverride: cfg.EngineServerName,
		ConnectTimeout:     10 * time.Second,
	})
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	return engineclient.Run(ctx, conn,
		&enginev1.SubscribeRequest{
			SubscriberId:      cfg.RunnerSubscriber,
			Symbols:           subscribedSymbols(ctx, cfg, svc),
			WantStatusUpdates: false,
		},
		engineclient.Events{
			OnConnected: func(client enginev1.EngineServiceClient) {
				logger.Info("terhubung ke engine", "addr", cfg.EngineGRPCAddr)
				applySnapshot(ctx, client, svc, cfg, logger)
				svc.SetAckSender(ackSender(client, cfg.RunnerSubscriber))
			},
			OnSignal: func(signal *enginev1.Signal) {
				svc.SetEngineVersion(signal.GetEngineVersion())
				svc.HandleSignal(ctx, signal)
			},
			OnUpdate: func(update *enginev1.SignalUpdate) {
				svc.HandleUpdate(ctx, update)
			},
			OnMarketState: func(state *enginev1.MarketStateEvent) {
				svc.SetMarketState(state.GetState())
				logger.Info("status pasar", "state", state.GetState().String(), "detail", state.GetDetail())
			},
			OnHeartbeat: func(hb *enginev1.Heartbeat) {
				svc.SetEngineVersion(hb.GetEngineVersion())
			},
			OnDisconnected: func(err error) {
				logger.Warn("stream engine berakhir", "error", errorText(err))
			},
		},
	)
}

// applySnapshot memulihkan sinyal yang masih berlaku setelah reconnect (D19).
func applySnapshot(ctx context.Context, client enginev1.EngineServiceClient, svc *runner.Service, cfg *config.Config, logger *slog.Logger) {
	snapCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	snapshot, err := client.Snapshot(snapCtx, &enginev1.SnapshotRequest{
		SubscriberId: cfg.RunnerSubscriber,
	})
	if err != nil {
		logger.Warn("snapshot engine gagal", "error", err.Error())
		return
	}

	svc.SetEngineVersion(snapshot.GetEngineVersion())
	if state := snapshot.GetMarketState(); state != nil {
		svc.SetMarketState(state.GetState())
	}

	logger.Info("snapshot diterima", "signals", len(snapshot.GetSignals()))

	// Sinyal dari snapshot melewati jalur yang sama dengan sinyal live:
	// TTL tetap diperiksa, sehingga sinyal kedaluwarsa tidak dieksekusi.
	for _, signal := range snapshot.GetSignals() {
		svc.HandleSignal(ctx, signal)
	}
}

// ackSender mengirim ACK ke engine. Payload tidak memuat hasil trading.
func ackSender(client enginev1.EngineServiceClient, subscriberID string) runner.AckSender {
	return func(ctx context.Context, signalID string) error {
		ackCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		resp, err := client.AckSignal(ackCtx, &enginev1.AckRequest{
			SignalId:     signalID,
			SubscriberId: subscriberID,
			ReceivedAt:   timestamppbNow(),
		})
		if err != nil {
			return err
		}
		if !resp.GetOk() {
			return errors.New(resp.GetMessage())
		}
		return nil
	}
}

// subscribedSymbols menghitung simbol yang perlu dipantau saat koneksi dibuka.
func subscribedSymbols(ctx context.Context, cfg *config.Config, svc *runner.Service) []string {
	reqs, err := svc.ComputeRequirements(ctx, []runner.StrategyLocation{
		{Strategy: cfg.EngineStrategy, Addr: cfg.EngineGRPCAddr},
	})
	if err != nil {
		return nil
	}
	return reqs[cfg.EngineStrategy]
}

// recordEngineState mencatat status koneksi agar dashboard bisa menampilkan
// apakah engine sedang terjangkau.
func recordEngineState(ctx context.Context, store repository.RunnerRepository, id, strategy string, cause error, logger *slog.Logger) {
	now := time.Now().UTC()

	state := &model.EngineState{
		ID:         id,
		Strategy:   strategy,
		Connected:  cause == nil,
		LastError:  truncateText(errorText(cause), 255),
		LastSeenAt: &now,
	}

	if err := store.UpsertEngineState(ctx, state); err != nil {
		logger.Error("gagal mencatat status engine", "error", err.Error())
	}
}

func truncateText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

func timestamppbNow() *timestamppb.Timestamp {
	return timestamppb.New(time.Now().UTC())
}

func engineStateID(strategy string) string {
	return "engine:" + strategy
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func slogLevel() slog.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
