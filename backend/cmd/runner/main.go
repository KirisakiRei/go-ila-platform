package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/alert"
	"ila/backend/internal/broker"
	"ila/backend/internal/broker/ctrader"
	"ila/backend/internal/broker/matchtrader"
	"ila/backend/internal/config"
	"ila/backend/internal/engineclient"
	"ila/backend/internal/logkit"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/runner/notify"
	"ila/backend/internal/settings"
	"ila/backend/internal/telegram"
	"ila/backend/pkg/cryptox"
)

// batas percobaan koneksi ulang ke engine.
const (
	minBackoff = 1 * time.Second
	maxBackoff = 30 * time.Second
	// Jeda penyelarasan langganan: cukup cepat merespons langganan baru,
	// cukup jarang agar tidak membebani database.
	symbolSyncInterval = 2 * time.Minute
	// Sesi yang bertahan selama ini dianggap sehat, bukan kegagalan berulang.
	backoffResetAfter = 30 * time.Second
	// Jeda minimum antar penulisan status engine dari heartbeat.
	engineStateWriteInterval = 30 * time.Second
)

func main() {
	_ = godotenv.Load()

	hasilLog := logkit.Buat(logkit.Opsi{
		Dir:        os.Getenv("LOG_DIR"),
		Layanan:    "runner",
		Level:      slogLevel(),
		JugaStdout: true,
		HariSimpan: envInt("LOG_HARI", 30),
		BatasBytes: int64(envInt("LOG_BATAS_MB", 1024)) * 1024 * 1024,
	})
	defer hasilLog.Tutup()

	logger := hasilLog.Logger
	slog.SetDefault(logger)
	if hasilLog.Rawat != nil {
		if err := hasilLog.Rawat(); err != nil {
			logger.Warn("perawatan log gagal", "evt", "log.rawat.gagal", "error", err.Error())
		}
	}

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

	// Flag & mode dibaca ulang tiap evaluasi dengan cache 10 detik, sehingga
	// rem darurat dari panel admin berlaku tanpa restart (plans/11 I2).
	flags := settings.NewService(repository.NewSettingsRepository(db), envFlagReader(cfg))
	// Ketersediaan ILA 3.0 dibaca dari katalog paket, bukan env, agar panel
	// admin dan Runner menampilkan tahap yang sama (plans/15 §6.1).
	features := repository.NewTierFeatureReader(db)
	modeSource := runnerModeSource{store: repository.NewSystemRepository(db)}

	// --- Kanal notifikasi -----------------------------------------------------
	tgClient := telegram.NewClient(telegram.Config{
		BotToken: cfg.TelegramBotToken,
		Timeout:  cfg.TelegramTimeout,
	}, logger)

	notifier := notify.New(store, repository.NewNotificationRepository(db), tgClient, logger, 2)
	defer notifier.Close()

	// Peringatan gangguan platform ke admin; anti-banjir 15 menit per pemicu.
	alerts := alert.New(tgClient, cfg.TelegramAdminChatID, logger)

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
	// Eksekusi nyata lewat access token akun, tanpa menyimpan password.
	registry.Register("matchtrader", func(accountID, _ string) (broker.Client, error) {
		return matchtrader.New(accountID), nil
	})
	registry.Register("ctrader", func(accountID, _ string) (broker.Client, error) {
		return ctrader.New(accountID, cfg.CTraderClientID, cfg.CTraderClientSecret), nil
	})

	// --- Runner ---------------------------------------------------------------
	runner.SetDailyTimezone(cfg.DailyTimezone)
	// Tahap strategi dipakai untuk gerbang ComputeRequirements. Nilainya harus
	// sama dengan yang dipakai engineTargets, jadi keduanya dibaca dari katalog
	// paket yang sama.
	strategySettings := runner.NewSettings(cfg.ILA3Availability)
	strategySettings.Reload(ctx, features)

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
		// Entry baru ditahan bila dokumen wajib belum disetujui.
		LegalGate: repository.NewLegalGate(db),
		Flags:     flags,
		Mode:      modeSource,
		// Engine yang mengaku buka di akhir pekan hampir pasti salah zona
		// waktu; ini satu-satunya cara mengetahuinya tanpa menunggu keluhan.
		OnMarketAnomaly: func(anomalies []runner.MarketScheduleAnomaly) {
			details := make([]string, 0, len(anomalies))
			for _, anomaly := range anomalies {
				details = append(details, string(anomaly))
			}
			alerts.Fire(ctx, alert.MarketScheduleAnomaly, strings.Join(details, "; "))
		},
		// PairGate memastikan simbol yang dinonaktifkan admin tidak pernah
		// diminta ke engine, tanpa memutus posisi yang sudah terbuka.
		PairGate: runner.NewCachedPairGate(store, time.Minute),
		// CorrelationGate meneruskan pasangan SMT (mis. XAUUSD → XAGUSD) agar
		// engine ikut menarik lilin instrumen pembanding.
		CorrelationGate: runner.NewCachedCorrelationGate(store, time.Minute),
		// Tanpa ini gerbang strategi memakai env saja dan seluruh permintaan
		// ILA 3.0 tersaring habis walau engine-nya sudah berjalan.
		Settings: strategySettings,
	}

	svc := runner.NewService(store, registry, publisher, logger, opts)

	if cfg.MasterKey != "" {
		keyring, keyErr := cryptox.NewKeyring(cfg.MasterKeyVersion, cfg.MasterKey, cfg.LegacyMasterKeys())
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
	// Sisa sinyal kedaluwarsa dari proses sebelumnya dibersihkan lebih dulu
	// agar radar tidak menampilkan baris mati sejak awal.
	svc.ExpireStaleSignals(ctx)

	var wg sync.WaitGroup

	// Pemantau posisi: selalu jalan walau engine belum tersambung, karena
	// posisi user tetap butuh pelaporan.
	wg.Add(1)
	go func() {
		defer wg.Done()
		runMonitor(ctx, svc, logger)
	}()

	// Telemetry saldo akun: cukup sekali per menit (plans/05 par.9).
	wg.Add(1)
	go func() {
		defer wg.Done()
		runTelemetry(ctx, svc, logger)
	}()

	// Koneksi engine: setiap strategi punya proses sendiri (plans/15 par.1.1).
	targets := engineTargets(cfg, features)
	for _, target := range targets {
		wg.Add(1)
		go func(target engineTarget) {
			defer wg.Done()
			runEngineLoop(ctx, target, svc, store, logger)
		}(target)
	}
	if len(targets) == 0 {
		logger.Warn("tidak ada alamat engine; runner berjalan tanpa sinyal")
	}

	// Penerima log Engine: satu tempat untuk membaca log dua VPS.
	logSrv, err := jalankanPenerimaLog(
		os.Getenv("LOG_GRPC_ADDR"),
		os.Getenv("LOG_DIR"),
		os.Getenv("LOG_MTLS_CERT"),
		os.Getenv("LOG_MTLS_KEY"),
		os.Getenv("LOG_MTLS_CA"),
		logger,
	)
	if err != nil {
		logger.Error("penerima log gagal disiapkan", "evt", "log.server.gagal", "error", err.Error())
		os.Exit(1)
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

	if logSrv != nil {
		logSrv.GracefulStop()
	}

	cancel()
	wg.Wait()
	svc.CloseClients()

	logger.Info("runner berhenti")
}

// ila3Running memutuskan apakah proses engine ILA 3.0 boleh dijalankan.
// Nilai env hanya cadangan bila katalog paket belum punya barisnya.
func ila3Running(cfg *config.Config, features runner.ILA3StatusReader) bool {
	stage := runner.NewSettings(cfg.ILA3Availability)
	stage.Reload(context.Background(), features)
	return stage.ILA3Running()
}

// runTelemetry memperbarui saldo akun broker secara berkala.
// Jalan pertama langsung agar dashboard tidak kosong setelah restart.
func runTelemetry(ctx context.Context, svc *runner.Service, logger *slog.Logger) {
	svc.RefreshTelemetry(ctx)

	ticker := time.NewTicker(runner.TelemetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			svc.RefreshTelemetry(ctx)
		}
	}
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
			// Sinyal yang TTL-nya habis tidak akan pernah menerima status dari
			// engine; penyapu ini yang menutupnya.
			svc.ExpireStaleSignals(ctx)
		}
	}
}

// engineTarget adalah satu proses engine yang harus dijangkau Runner.
type engineTarget struct {
	Strategy           string
	Addr               string
	CertFile           string
	KeyFile            string
	CAFile             string
	ServerNameOverride string
}

// engineTargets menyusun daftar engine yang aktif.
// ILA 3.0 hanya ikut bila tahapnya mengizinkan engine berjalan.
func engineTargets(cfg *config.Config, features runner.ILA3StatusReader) []engineTarget {
	out := make([]engineTarget, 0, 2)

	if cfg.EngineGRPCAddr != "" {
		out = append(out, engineTarget{
			Strategy:           cfg.EngineStrategy,
			Addr:               cfg.EngineGRPCAddr,
			CertFile:           cfg.EngineMTLSCert,
			KeyFile:            cfg.EngineMTLSKey,
			CAFile:             cfg.EngineMTLSCA,
			ServerNameOverride: cfg.EngineServerName,
		})
	}

	if cfg.Engine3GRPCAddr != "" && ila3Running(cfg, features) {
		out = append(out, engineTarget{
			Strategy:           "ila-3.0",
			Addr:               cfg.Engine3GRPCAddr,
			CertFile:           cfg.Engine3MTLSCert,
			KeyFile:            cfg.Engine3MTLSKey,
			CAFile:             cfg.Engine3MTLSCA,
			ServerNameOverride: cfg.Engine3ServerName,
		})
	}

	return out
}

// runEngineLoop menjaga koneksi ke satu engine tetap hidup.
//
// Pola reconnect mengikuti plans/05 §8: setelah terhubung, minta snapshot,
// lalu berlangganan ulang untuk semua simbol kebutuhan. Sinyal lama yang
// terlewat TIDAK dieksekusi karena pemeriksaan TTL berjalan di jalur sinyal.
func runEngineLoop(
	ctx context.Context,
	target engineTarget,
	svc *runner.Service,
	store repository.RunnerRepository,
	logger *slog.Logger,
) {
	backoff := minBackoff
	stateID := engineStateID(target.Strategy)

	for ctx.Err() == nil {
		mulai := time.Now()
		err := streamOnce(ctx, target, svc, store, logger)
		recordEngineState(ctx, store, stateID, target.Strategy, svc.EngineVersion(target.Strategy), err, logger)

		// Sesi yang bertahan cukup lama dianggap sehat: jeda antar percobaan
		// dikembalikan ke awal agar koneksi berikutnya tidak menunggu lama.
		if time.Since(mulai) >= backoffResetAfter {
			backoff = minBackoff
		}

		if ctx.Err() != nil {
			return
		}

		logger.Warn("koneksi engine terputus, mencoba ulang",
			"strategy", target.Strategy,
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
	target engineTarget,
	svc *runner.Service,
	store repository.RunnerRepository,
	logger *slog.Logger,
) error {
	conn, err := engineclient.NewConnection(engineclient.Config{
		Addr:               target.Addr,
		Strategy:           target.Strategy,
		SubscriberID:       subscriberFor(target.Strategy),
		CertFile:           target.CertFile,
		KeyFile:            target.KeyFile,
		CAFile:             target.CAFile,
		ServerNameOverride: target.ServerNameOverride,
		ConnectTimeout:     10 * time.Second,
	})
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	subscriberID := subscriberFor(target.Strategy)

	// Penulisan status engine dibatasi: heartbeat datang jauh lebih sering
	// daripada kebutuhan dashboard, sedangkan setiap tulis adalah UPDATE tabel.
	var stateMu sync.Mutex
	var lastStateWrite time.Time
	catatState := func(version string, cause error) {
		stateMu.Lock()
		if cause == nil && !lastStateWrite.IsZero() && time.Since(lastStateWrite) < engineStateWriteInterval {
			stateMu.Unlock()
			return
		}
		lastStateWrite = time.Now()
		stateMu.Unlock()
		recordEngineState(ctx, store, engineStateID(target.Strategy), target.Strategy, version, cause, logger)
	}

	// Trace khusus stream: satu koneksi = satu rangkaian log yang bisa dicari.
	streamCtx := engineclient.DenganTrace(ctx, logkit.Trace())

	// Daftar simbol dihitung sekali: dipakai untuk langganan dan peta feed agar
	// keduanya tidak pernah berbeda isi.
	symbolsAwal := subscribedSymbols(ctx, target, svc)

	return engineclient.Run(streamCtx, conn,
		&enginev1.SubscribeRequest{
			SubscriberId:      subscriberID,
			Symbols:           symbolsAwal,
			Pairs:             timeframePairs(ctx, target, svc, logger),
			SymbolPairs:       symbolPairs(ctx, target, svc, symbolsAwal, logger),
			SymbolFeeds:       pairFeeds(ctx, store, symbolsAwal, logger),
			WantStatusUpdates: true,
		},
		engineclient.Events{
			OnConnected: func(client enginev1.EngineServiceClient, connCtx context.Context) {
				logger.Info("terhubung ke engine",
					"strategy", target.Strategy, "addr", target.Addr)
				// Catat SEBELUM apa pun yang bisa gagal: dashboard harus tahu
				// engine hidup sejak koneksi terbuka, bukan baru setelah
				// stream berakhir. Sebelumnya status hanya ditulis saat putus,
				// sehingga yang tersimpan selalu "tidak terhubung" padahal
				// engine sedang berjalan.
				recordEngineState(ctx, store, engineStateID(target.Strategy), target.Strategy, svc.EngineVersion(target.Strategy), nil, logger)
				applySnapshot(ctx, client, svc, subscriberID, target.Strategy, logger)
				svc.SetAckSender(target.Strategy, ackSender(client, subscriberID))
				svc.SetSymbolSync(target.Strategy, symbolSync(client, subscriberID, target, svc, store, logger))
				// Selaraskan segera: antara hitung kebutuhan dan koneksi
				// terbuka, langganan pengguna bisa sudah berubah.
				syncNow(ctx, target, svc, logger)
				go syncLoop(connCtx, target, svc, logger)
			},
			OnSignal: func(signal *enginev1.Signal) {
				svc.SetEngineVersion(target.Strategy, signal.GetEngineVersion())
				svc.HandleSignal(ctx, signal)
			},
			OnUpdate: func(update *enginev1.SignalUpdate) {
				svc.HandleUpdate(ctx, update)
			},
			OnMarketState: func(state *enginev1.MarketStateEvent) {
				svc.SetMarketState(state.GetState())
				logger.Info("status pasar",
					"strategy", target.Strategy,
					"state", state.GetState().String(), "detail", state.GetDetail())
			},
			OnHeartbeat: func(hb *enginev1.Heartbeat) {
				svc.SetEngineVersion(target.Strategy, hb.GetEngineVersion())
				// Heartbeat adalah satu-satunya bukti engine masih hidup saat
				// tidak ada sinyal. Tanpa penulisan ini last_seen_at hanya
				// berubah ketika koneksi dibuka atau ditutup, sehingga panel
				// kesehatan menandai engine sehat sebagai stale.
				catatState(hb.GetEngineVersion(), nil)
			},
			OnDisconnected: func(err error) {
				logger.Warn("stream engine berakhir",
					"strategy", target.Strategy, "error", errorText(err))
			},
		},
	)
}

// symbolSync mengirim langganan terbaru ke engine.
//
// Subscribe di engine MENGGANTI seluruh daftar, bukan menambah. Karena itu
// setiap penyelarasan mengirim daftar simbol lengkap bersama pasangan
// timeframe-nya; mengirim hanya "add" akan menghapus simbol lain tanpa
// terlihat.
//
// Pasangan ikut dikirim setiap kali agar perubahan saklar timeframe di
// dashboard langsung berlaku, tanpa menunggu stream terputus dan tersambung
// ulang.
func symbolSync(client enginev1.EngineServiceClient, subscriberID string, target engineTarget, svc *runner.Service, store repository.RunnerRepository, logger *slog.Logger) runner.SymbolSync {
	return func(ctx context.Context, _, _ []string) error {
		syncCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		syncCtx = engineclient.DenganTrace(syncCtx, logkit.Trace())

		symbols := subscribedSymbols(syncCtx, target, svc)
		if len(symbols) == 0 {
			// Tidak ada yang perlu dipantau: engine dibiarkan seperti
			// sebelumnya agar tidak ada langganan menggantung.
			return nil
		}

		resp, err := client.Subscribe(syncCtx, &enginev1.SubscribeRequest{
			SubscriberId:      subscriberID,
			Symbols:           symbols,
			Pairs:             timeframePairs(syncCtx, target, svc, logger),
			SymbolPairs:       symbolPairs(syncCtx, target, svc, symbols, logger),
			SymbolFeeds:       pairFeeds(syncCtx, store, symbols, logger),
			WantStatusUpdates: true,
		})
		if err != nil {
			return fmt.Errorf("menyelaraskan langganan engine: %w", err)
		}
		if !resp.GetOk() {
			return errors.New(resp.GetMessage())
		}
		return nil
	}
}

// syncNow menghitung kebutuhan terkini lalu menyelaraskannya dengan engine.
//
// Gagal di sini bukan alasan memutus stream: langganan lama masih berjalan dan
// percobaan berikutnya akan mengulang dengan daftar terbaru.
func syncNow(ctx context.Context, target engineTarget, svc *runner.Service, logger *slog.Logger) {
	desired := subscribedSymbols(ctx, target, svc)
	if err := svc.SyncSubscriptions(ctx, target.Strategy, desired); err != nil {
		logger.Warn("gagal menyelaraskan langganan engine",
			"strategy", target.Strategy, "error", errorText(err))
	}
}

// syncLoop menyelaraskan langganan secara berkala selama stream hidup.
//
// Tanpa ini, pengguna yang baru berlangganan simbol tidak akan dipantau sampai
// Runner dijalankan ulang (plans/05 §8).
func syncLoop(ctx context.Context, target engineTarget, svc *runner.Service, logger *slog.Logger) {
	ticker := time.NewTicker(symbolSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncNow(ctx, target, svc, logger)
		}
	}
}

// applySnapshot memulihkan sinyal yang masih berlaku setelah reconnect (D19).
func applySnapshot(
	ctx context.Context,
	client enginev1.EngineServiceClient,
	svc *runner.Service,
	subscriberID, strategy string,
	logger *slog.Logger,
) {
	snapCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	snapshot, err := client.Snapshot(snapCtx, &enginev1.SnapshotRequest{
		SubscriberId: subscriberID,
	})
	if err != nil {
		logger.Warn("snapshot engine gagal", "error", err.Error())
		return
	}

	svc.SetEngineVersion(strategy, snapshot.GetEngineVersion())
	if state := snapshot.GetMarketState(); state != nil {
		svc.SetMarketState(state.GetState())
	}

	logger.Info("snapshot diterima",
		"strategy", strategy, "signals", len(snapshot.GetSignals()))

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
func subscribedSymbols(ctx context.Context, target engineTarget, svc *runner.Service) []string {
	reqs, err := svc.ComputeRequirements(ctx, []runner.StrategyLocation{
		{Strategy: target.Strategy, Addr: target.Addr},
	})
	if err != nil {
		return nil
	}
	return reqs[target.Strategy]
}

// timeframePairs menyusun pasangan timeframe yang perlu dihitung engine untuk
// satu strategi.
//
// Isinya gabungan pasangan yang dipantau siapa pun. Bila fitur timeframe belum
// dinyalakan atau belum ada yang mengaturnya, hasilnya kosong sehingga engine
// menerima perilaku lama: hanya daftar simbol.
// timeframePairs menyusun pasangan timeframe gabungan yang perlu dihitung
// engine untuk sebuah strategi.
//
// Deprecated: daftar ini tidak memuat identitas simbol, sehingga switch satu
// simbol bocor ke simbol lain. Fungsi ini dipertahankan HANYA sebagai jalur
// kompatibilitas untuk engine lama yang belum mengenal symbol_pairs.
func timeframePairs(ctx context.Context, target engineTarget, svc *runner.Service, logger *slog.Logger) []*enginev1.Pair {
	symbols := subscribedSymbols(ctx, target, svc)
	if len(symbols) == 0 {
		return nil
	}

	unik := make(map[string]*enginev1.Pair)
	for _, symbol := range symbols {
		reqs, err := svc.ComputeTimeframeRequirements(ctx, symbol)
		if err != nil {
			logger.Warn("gagal menyusun pasangan timeframe",
				"strategy", target.Strategy, "symbol", symbol, "error", errorText(err))
			continue
		}
		for _, pair := range reqs[target.Strategy] {
			key := pair.Htf + "|" + pair.Ltf
			unik[key] = &enginev1.Pair{Htf: pair.Htf, Ltf: pair.Ltf}
		}
	}
	if len(unik) == 0 {
		return nil
	}

	// Urutan stabil: permintaan yang sama tidak boleh menghasilkan susunan
	// berbeda setiap kali stream tersambung.
	keys := make([]string, 0, len(unik))
	for key := range unik {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]*enginev1.Pair, 0, len(keys))
	for _, key := range keys {
		out = append(out, unik[key])
	}
	return out
}

// symbolPairs menyusun kebijakan pasangan timeframe PER SIMBOL untuk satu
// strategi. Inilah jalur yang mencegah switch satu simbol mematikan simbol
// lain.
//
// Simbol tanpa kebijakan sengaja tidak dimasukkan: engine memperlakukannya
// sebagai "pakai daftar bawaan", yang memang perilaku yang diinginkan.
func symbolPairs(ctx context.Context, target engineTarget, svc *runner.Service, symbols []string, logger *slog.Logger) map[string]*enginev1.SymbolPairs {
	if len(symbols) == 0 {
		return nil
	}

	out := make(map[string]*enginev1.SymbolPairs, len(symbols))
	for _, symbol := range symbols {
		kebijakan, err := svc.ComputeSymbolPairs(ctx, symbol)
		if err != nil {
			logger.Warn("gagal menyusun pasangan timeframe per simbol",
				"strategy", target.Strategy, "symbol", symbol, "error", errorText(err))
			continue
		}
		policy, ada := kebijakan[target.Strategy]
		if !ada || !policy.Restricted {
			// Belum diatur: biarkan engine memakai daftar bawaannya.
			continue
		}
		pairs := make([]*enginev1.Pair, 0, len(policy.Pairs))
		for _, p := range policy.Pairs {
			pairs = append(pairs, &enginev1.Pair{Htf: p.Htf, Ltf: p.Ltf})
		}
		out[symbol] = &enginev1.SymbolPairs{Pairs: pairs, Restricted: true}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// subscriberFor membentuk identitas langganan per strategi, agar dua engine
// tidak saling mengira sebagai pelanggan yang sama.
func subscriberFor(strategy string) string {
	return "runner-" + strategy
}

// pairFeeds menyusun peta simbol ke feed candle pilihan admin untuk simbol yang
// benar-benar dipantau. Simbol di luar daftar langganan tidak perlu dikirim.
func pairFeeds(ctx context.Context, store repository.RunnerRepository, symbols []string, logger *slog.Logger) map[string]string {
	if len(symbols) == 0 {
		return nil
	}

	all, err := store.PairFeeds(ctx)
	if err != nil {
		// Gagal membaca feed bukan alasan menolak langganan: engine tetap
		// memakai pemetaan bawaannya, dan penyelarasan berikutnya mengulang.
		logger.Warn("gagal membaca feed pair", "error", errorText(err))
		return nil
	}

	out := make(map[string]string, len(symbols))
	for _, symbol := range symbols {
		if feed, ok := all[strings.ToUpper(symbol)]; ok {
			out[symbol] = feed
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// recordEngineState mencatat status koneksi agar dashboard bisa menampilkan
// apakah engine sedang terjangkau.
func recordEngineState(ctx context.Context, store repository.RunnerRepository, id, strategy, version string, cause error, logger *slog.Logger) {
	now := time.Now().UTC()

	state := &model.EngineState{
		ID:            id,
		Strategy:      strategy,
		EngineVersion: version,
		Connected:     cause == nil,
		LastError:     truncateText(errorText(cause), 255),
		LastSeenAt:    &now,
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

func envInt(key string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
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
