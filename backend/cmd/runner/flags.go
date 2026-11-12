package main

import (
	"context"

	"ila/backend/internal/config"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/settings"
)

// envFlagReader menjadi nilai bawaan flag ketika belum ada override di
// database.
//
// Logikanya tidak lagi ditulis di sini melainkan dipakai bersama backend-api
// lewat settings.EnvFromConfig. Duplikasi sebelumnya membuat dua saklar yang
// bawaannya NYALA tidak terbaca di salah satu sisi, sehingga setelan seolah
// diabaikan.
func envFlagReader(cfg *config.Config) func(string) string {
	return settings.EnvFromConfig(cfg)
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
