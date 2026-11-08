package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"ila/backend/internal/alert"
	"ila/backend/internal/config"
	"ila/backend/internal/email"
	"ila/backend/internal/logkit"
	"ila/backend/internal/repository"
	"ila/backend/internal/router"
	"ila/backend/internal/runner"
	"ila/backend/internal/service"
	"ila/backend/internal/telegram"
)

func main() {
	_ = godotenv.Load()

	hasilLog := logkit.Buat(logkit.Opsi{
		Dir:        os.Getenv("LOG_DIR"),
		Layanan:    "backend-api",
		Level:      slogLevel(os.Getenv("LOG_LEVEL")),
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

	cfg, err := config.Load("backend-api")
	if err != nil {
		logger.Error("konfigurasi tidak sah", "error", err.Error())
		os.Exit(1)
	}

	// Zona hari akuntansi harus sama dengan yang dipakai Runner.
	runner.SetDailyTimezone(cfg.DailyTimezone)

	ctx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	db, err := config.NewDatabase(ctx, cfg)
	if err != nil {
		logger.Error("database tidak siap", "error", err.Error())
		os.Exit(1)
	}

	// Antrean surel dikirim dari proses ini: satu worker cukup untuk jumlah
	// pelanggan saat ini dan tidak menambah container baru (plans/17 §2.2).
	mailCfg := email.Config{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort,
		Username: cfg.SMTPUser, Password: cfg.SMTPPass, From: cfg.SMTPFrom,
	}
	if mailCfg.Enabled() {
		worker := email.NewWorker(repository.NewEmailRepository(db), email.NewSMTPSender(mailCfg), logger)
		go worker.Run(ctx, 30*time.Second)
		logger.Info("pengirim surel aktif", "host", mailCfg.Host)
	} else {
		// Tanpa SMTP, pesan tetap masuk antrean sehingga tidak ada yang hilang.
		logger.Warn("SMTP belum dikonfigurasi; surel menumpuk di antrean")
	}

	// Tautan surel memakai alamat publik yang sama dengan halaman web.
	email.PublicBaseURL = cfg.PublicURL

	// Peringatan admin: gangguan terdeteksi ≤2 menit, anti-banjir 15 menit
	// per pemicu agar ponsel admin tidak bergetar terus (plans/18 §2).
	tgClient := telegram.NewClient(telegram.Config{
		BotToken: cfg.TelegramBotToken, Timeout: cfg.TelegramTimeout,
	}, logger)
	notifier := alert.New(tgClient, cfg.TelegramAdminChatID, logger)
	// Penyapu langganan: menurunkan tier yang masa tenggangnya habis. Tanpa
	// ini user kedaluwarsa tetap mendapat akses Plus selamanya.
	sweeper := service.NewBillingSweeper(router.Billing(cfg, db, logger), logger)
	go sweeper.Run(ctx, time.Hour)

	watchdog := service.NewWatchdog(router.SystemOps(cfg, db, logger), notifier, logger).
		WithDiskProbe(diskUsage)
	go watchdog.Run(ctx, service.WatchInterval)
	if notifier.Enabled() {
		logger.Info("peringatan admin aktif", "penerima", cfg.TelegramAdminChatID)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router.NewBackendAPI(cfg, db, logger),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("backend-api berjalan", "port", cfg.HTTPPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server HTTP berhenti tak terduga", "error", err.Error())
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("menghentikan backend-api")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("gagal menghentikan server dengan rapi", "error", err.Error())
		os.Exit(1)
	}

	logger.Info("backend-api berhenti")
}
