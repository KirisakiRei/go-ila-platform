package main

import (
	"context"
	"strconv"

	"ila/backend/internal/config"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/settings"
)

// envFlagReader menjadi nilai bawaan flag ketika belum ada override di
// database. Nilainya sama dengan yang dibaca backend-api agar keduanya
// menampilkan angka yang sama (plans/11 I2).
func envFlagReader(cfg *config.Config) func(string) string {
	return func(key string) string {
		switch key {
		case settings.KeyAutoExecute:
			return strconv.FormatBool(cfg.AutoExecute)
		case settings.KeyDryRunAll:
			return strconv.FormatBool(cfg.DryRunAll)
		case settings.KeyMaxDailyTrades:
			return strconv.Itoa(cfg.MaxDailyTrades)
		case settings.KeyMaxDailyLosses:
			return strconv.Itoa(cfg.MaxDailyLosses)
		case settings.KeyStopAfterWin:
			return strconv.FormatBool(cfg.StopAfterWin)
		case settings.KeyMarketOpenOnly:
			return "true"
		case settings.KeyLegalGate:
			return "true"
		}
		return ""
	}
}

// runnerModeSource membaca mode platform dari tabel system_state.
type runnerModeSource struct {
	store *repository.SystemRepository
}

// State mengembalikan mode yang berlaku; baris kosong berarti normal.
func (r runnerModeSource) State(ctx context.Context) (*runner.ModedState, error) {
	row, err := r.store.State(ctx)
	if err != nil {
		return nil, err
	}
	return &runner.ModedState{Mode: row.Mode}, nil
}
