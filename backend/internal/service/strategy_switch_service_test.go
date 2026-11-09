package service

import (
	"context"
	"testing"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
)

// fakeStrategyRepo menyimpan saklar metode di memori untuk pengujian.
type fakeStrategyRepo struct {
	rows []model.AccountPairStrategy
}

func (f *fakeStrategyRepo) ListByUser(_ context.Context, userID string) ([]model.AccountPairStrategy, error) {
	var out []model.AccountPairStrategy
	for _, row := range f.rows {
		if row.UserID == userID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeStrategyRepo) ListByAccount(_ context.Context, accountID string) ([]model.AccountPairStrategy, error) {
	var out []model.AccountPairStrategy
	for _, row := range f.rows {
		if row.BrokerAccountID == accountID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeStrategyRepo) Replace(_ context.Context, accountID, userID, symbol string, ucapan map[string]bool) error {
	for strategy, enabled := range ucapan {
		found := false
		for i := range f.rows {
			row := f.rows[i]
			if row.BrokerAccountID == accountID && row.Symbol == symbol && row.Strategy == strategy {
				f.rows[i].IsEnabled = enabled
				found = true
				break
			}
		}
		if !found {
			f.rows = append(f.rows, model.AccountPairStrategy{
				BrokerAccountID: accountID,
				UserID:          userID,
				Symbol:          symbol,
				Strategy:        strategy,
				IsEnabled:       enabled,
			})
		}
	}
	return nil
}

func (f *fakeStrategyRepo) Ensure(_ context.Context, rows []model.AccountPairStrategy) error {
	for _, row := range rows {
		ada := false
		for _, existing := range f.rows {
			if existing.BrokerAccountID == row.BrokerAccountID &&
				existing.Symbol == row.Symbol &&
				existing.Strategy == row.Strategy {
				ada = true
				break
			}
		}
		if !ada {
			f.rows = append(f.rows, row)
		}
	}
	return nil
}

// fakeStrategyLister melayani kebutuhan daftar akun dan hak metode.
type fakeStrategyLister struct {
	accounts []model.BrokerAccount
	allowed  map[string]bool
}

func (f *fakeStrategyLister) ListByUser(_ context.Context, userID string) ([]model.BrokerAccount, error) {
	var out []model.BrokerAccount
	for _, account := range f.accounts {
		if account.UserID == userID {
			out = append(out, account)
		}
	}
	return out, nil
}

func (f *fakeStrategyLister) FindByID(_ context.Context, accountID string) (*model.BrokerAccount, error) {
	for i := range f.accounts {
		if f.accounts[i].ID == accountID {
			copy := f.accounts[i]
			return &copy, nil
		}
	}
	return nil, nil
}

func (f *fakeStrategyLister) Strategies(_ context.Context, _ string) ([]dto.StrategyResponse, error) {
	out := make([]dto.StrategyResponse, 0, len(f.allowed))
	for version, allowed := range f.allowed {
		out = append(out, dto.StrategyResponse{
			Version:   version,
			Available: allowed,
			Allowed:   allowed,
		})
	}
	return out, nil
}

func TestStrategySwitch_MatrixMenyembunyikanIla3TanpaHak(t *testing.T) {
	repo := &fakeStrategyRepo{}
	lister := &fakeStrategyLister{
		accounts: []model.BrokerAccount{
			{Base: model.Base{ID: "acc-1"}, UserID: "u1", Label: "Utama", BrokerType: "paper", IsActive: true},
		},
		allowed: map[string]bool{"ila-2.0": true, "ila-3.0": false},
	}

	svc := NewStrategySwitchService(lister, repo, nil, lister)
	matrix, err := svc.Matrix(context.Background(), "u1", "xauusd")
	if err != nil {
		t.Fatalf("Matrix: %v", err)
	}

	if matrix.Symbol != "XAUUSD" {
		t.Fatalf("simbol = %q, ingin XAUUSD (huruf besar)", matrix.Symbol)
	}
	if len(matrix.Akun) != 1 {
		t.Fatalf("akun = %d, ingin 1", len(matrix.Akun))
	}
	akun := matrix.Akun[0]
	if akun.Allowed["ila-3.0"] {
		t.Fatal("ILA 3.0 tidak boleh ditandai berhak untuk tier ini")
	}
	if akun.Allowed["ila-2.0"] != true {
		t.Fatal("ILA 2.0 seharusnya selalu berhak")
	}
	if !akun.Strategies["ila-2.0"] {
		t.Fatal("ILA 2.0 seharusnya menyala secara bawaan")
	}
}

func TestStrategySwitch_SetMenolakIla3TanpaHak(t *testing.T) {
	repo := &fakeStrategyRepo{}
	lister := &fakeStrategyLister{
		accounts: []model.BrokerAccount{
			{Base: model.Base{ID: "acc-1"}, UserID: "u1", BrokerType: "paper", IsActive: true},
		},
		allowed: map[string]bool{"ila-2.0": true, "ila-3.0": false},
	}

	svc := NewStrategySwitchService(lister, repo, nil, lister)
	_, err := svc.Set(context.Background(), "u1", "acc-1", "XAUUSD",
		dto.UpdateStrategySwitchRequest{Strategies: map[string]bool{"ila-3.0": true}})
	if err == nil {
		t.Fatal("menyalakan ILA 3.0 tanpa hak harus ditolak")
	}
	if len(repo.rows) != 0 {
		t.Fatalf("saklar tidak boleh tersimpan saat ditolak, dapat %d baris", len(repo.rows))
	}
}

func TestStrategySwitch_SetMenyimpanSaklar(t *testing.T) {
	repo := &fakeStrategyRepo{}
	lister := &fakeStrategyLister{
		accounts: []model.BrokerAccount{
			{Base: model.Base{ID: "acc-1"}, UserID: "u1", BrokerType: "paper", IsActive: true},
		},
		allowed: map[string]bool{"ila-2.0": true, "ila-3.0": true},
	}

	svc := NewStrategySwitchService(lister, repo, nil, lister)
	matrix, err := svc.Set(context.Background(), "u1", "acc-1", "XAUUSD",
		dto.UpdateStrategySwitchRequest{Strategies: map[string]bool{"ila-2.0": false}})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if matrix.Akun[0].Strategies["ila-2.0"] {
		t.Fatal("saklar ILA 2.0 seharusnya mati setelah diubah")
	}
}

func TestStrategySwitch_SetMenolakAkunMilikOrangLain(t *testing.T) {
	repo := &fakeStrategyRepo{}
	lister := &fakeStrategyLister{
		accounts: []model.BrokerAccount{
			{Base: model.Base{ID: "acc-1"}, UserID: "u2", BrokerType: "paper", IsActive: true},
		},
		allowed: map[string]bool{"ila-2.0": true},
	}

	svc := NewStrategySwitchService(lister, repo, nil, lister)
	_, err := svc.Set(context.Background(), "u1", "acc-1", "XAUUSD",
		dto.UpdateStrategySwitchRequest{Strategies: map[string]bool{"ila-2.0": false}})
	if err == nil {
		t.Fatal("akun milik user lain harus ditolak")
	}
}

func TestStrategySwitch_EnsureDefaultsHanyaUntukYangBerhak(t *testing.T) {
	repo := &fakeStrategyRepo{}
	lister := &fakeStrategyLister{
		accounts: []model.BrokerAccount{
			{Base: model.Base{ID: "acc-1"}, UserID: "u1", BrokerType: "paper", IsActive: true},
		},
		allowed: map[string]bool{"ila-2.0": true, "ila-3.0": false},
	}

	svc := NewStrategySwitchService(lister, repo, nil, lister)
	if err := svc.EnsureDefaults(context.Background(), "u1", "XAUUSD"); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	if len(repo.rows) != 1 {
		t.Fatalf("baris saklar = %d, ingin 1 (hanya ILA 2.0)", len(repo.rows))
	}
	if repo.rows[0].Strategy != "ila-2.0" || !repo.rows[0].IsEnabled {
		t.Fatalf("saklar bawaan salah: %+v", repo.rows[0])
	}

	// Idempoten: pemanggilan kedua tidak menambah baris.
	if err := svc.EnsureDefaults(context.Background(), "u1", "XAUUSD"); err != nil {
		t.Fatalf("EnsureDefaults kedua: %v", err)
	}
	if len(repo.rows) != 1 {
		t.Fatalf("baris saklar = %d setelah panggilan kedua, ingin tetap 1", len(repo.rows))
	}
}
