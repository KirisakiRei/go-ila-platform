package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/config"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
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
	_ = db

	logger.Info("runner siap",
		"env", cfg.AppEnv,
		"engine_addr", cfg.EngineGRPCAddr,
		"daily_tz", cfg.DailyTimezone,
		"max_daily_trades", cfg.MaxDailyTrades,
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("runner berhenti")
}
