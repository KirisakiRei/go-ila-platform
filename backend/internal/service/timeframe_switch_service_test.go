package service

import (
	"context"
	"testing"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

// fakeTimeframeRepo menyimpan saklar timeframe di memori.
type fakeTimeframeRepo struct {
	rows []model.AccountPairTimeframe
}

func (f *fakeTimeframeRepo) ListByUser(_ context.Context, userID string) ([]model.AccountPairTimeframe, error) {
	var out []model.AccountPairTimeframe
	for _, row := range f.rows {
		if row.UserID == userID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeTimeframeRepo) ListByAccount(_ context.Context, accountID string) ([]model.AccountPairTimeframe, error) {
	var out []model.AccountPairTimeframe
	for _, row := range f.rows {
		if row.BrokerAccountID == accountID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (f *fakeTimeframeRepo) Replace(_ context.Context, accountID, userID, symbol, strategy string, rows []repository.TimeframeSwitch) error {
	for _, row := range rows {
		ketemu := false
		for i := range f.rows {
			ada := f.rows[i]
			if ada.BrokerAccountID == accountID && ada.Symbol == symbol &&
				ada.Strategy == strategy && ada.Htf == row.Htf && ada.Ltf == row.Ltf {
				f.rows[i].IsWatch = row.Watch
				f.rows[i].IsAuto = row.Auto
				ketemu = true
				break
			}
		}
		if !ketemu {
			f.rows = append(f.rows, model.AccountPairTimeframe{
				BrokerAccountID: accountID, UserID: userID, Symbol: symbol,
				Strategy: strategy, Htf: row.Htf, Ltf: row.Ltf,
				IsWatch: row.Watch, IsAuto: row.Auto,
			})
		}
	}
	return nil
}

func (f *fakeTimeframeRepo) Ensure(_ context.Context, rows []model.AccountPairTimeframe) error {
	f.rows = append(f.rows, rows...)
	return nil
}

func (f *fakeTimeframeRepo) WatchedPairs(_ context.Context, _ string) (map[string]map[repository.PairKey]bool, error) {
	return map[string]map[repository.PairKey]bool{}, nil
}

// fakeTimeframeSignals menyajikan pasangan timeframe yang "pernah terlihat".
type fakeTimeframeSignals struct {
	known map[string][]repository.TimeframePairRef
}

func (f *fakeTimeframeSignals) ListKnownTimeframes(context.Context) (map[string][]repository.TimeframePairRef, error) {
	return f.known, nil
}

// fakeTierFeature menjawab fitur paket dari peta sederhana.
type fakeTierFeature struct {
	values map[string]map[string]string
}

func (f *fakeTierFeature) Feature(_ context.Context, tier, key, fallback string) string {
	if row, ok := f.values[tier]; ok {
		if value, ok := row[key]; ok {
			return value
		}
	}
	return fallback
}

// siapkanTimeframe menyusun service dengan satu akun dan dua pasangan timeframe.
func siapkanTimeframe(t *testing.T) (*timeframeSwitchService, *fakeTimeframeRepo, *fakeStrategyRepo) {
	t.Helper()

	lister := &fakeStrategyLister{
		accounts: []model.BrokerAccount{
			{Base: model.Base{ID: "acc-1"}, UserID: "u1", Label: "Utama", BrokerType: "paper", IsActive: true},
		},
	}
	strategyRepo := &fakeStrategyRepo{
		rows: []model.AccountPairStrategy{
			{BrokerAccountID: "acc-1", UserID: "u1", Symbol: "XAUUSD", Strategy: "ila-2.0", IsEnabled: true},
		},
	}
	tfRepo := &fakeTimeframeRepo{}
	signals := &fakeTimeframeSignals{known: map[string][]repository.TimeframePairRef{
		"ila-2.0": {
			{Htf: "W1", Ltf: "H4"},
			{Htf: "H4", Ltf: "M15"},
		},
	}}
	tierFeature := &fakeTierFeature{values: map[string]map[string]string{
		"plus":  {model.FeatureAutoExecute: "true"},
		"basic": {model.FeatureAutoExecute: "false"},
	}}

	tier := "plus"
	svc := NewTimeframeSwitchService(
		lister, strategyRepo, tfRepo, signals, lister,
		func(context.Context, string) (string, error) { return tier, nil },
		tierFeature,
	).(*timeframeSwitchService)

	return svc, tfRepo, strategyRepo
}

func TestTimeframeMatrix_MenampilkanPasanganDariDataNyata(t *testing.T) {
	svc, _, _ := siapkanTimeframe(t)

	matrix, err := svc.Matrix(context.Background(), "u1", "xauusd")
	if err != nil {
		t.Fatalf("Matrix: %v", err)
	}

	if matrix.Symbol != "XAUUSD" {
		t.Fatalf("simbol = %q, ingin XAUUSD", matrix.Symbol)
	}
	if !matrix.AutoAllowed {
		t.Fatal("tier plus seharusnya boleh auto-eksekusi")
	}

	daftar := matrix.Tersedia["ila-2.0"]
	if len(daftar) != 2 {
		t.Fatalf("pasangan tersedia = %d, ingin 2", len(daftar))
	}
	if daftar[0].Label != "W1 · H4" {
		t.Fatalf("label = %q, ingin \"W1 · H4\"", daftar[0].Label)
	}
}

func TestTimeframeMatrix_KosongBilaEngineBelumKirim(t *testing.T) {
	svc, _, _ := siapkanTimeframe(t)
	// Tidak ada sinyal yang membawa timeframe.
	svc.signals = &fakeTimeframeSignals{known: map[string][]repository.TimeframePairRef{}}

	matrix, err := svc.Matrix(context.Background(), "u1", "XAUUSD")
	if err != nil {
		t.Fatalf("Matrix: %v", err)
	}
	if len(matrix.Tersedia) != 0 {
		t.Fatalf("pasangan tersedia = %v, ingin kosong saat belum ada data", matrix.Tersedia)
	}
}

func TestTimeframeSet_MenyimpanSaklar(t *testing.T) {
	svc, repo, _ := siapkanTimeframe(t)

	_, err := svc.Set(context.Background(), "u1", "acc-1", "XAUUSD", dto.UpdateTimeframeSwitchRequest{
		Strategy: "ila-2.0",
		Pairs: []dto.TimeframeSwitchItem{
			{Htf: "W1", Ltf: "H4", Watch: true, Auto: true},
			{Htf: "H4", Ltf: "M15", Watch: false, Auto: false},
		},
	})
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(repo.rows) != 2 {
		t.Fatalf("baris tersimpan = %d, ingin 2", len(repo.rows))
	}
	if !repo.rows[0].IsWatch || !repo.rows[0].IsAuto {
		t.Fatalf("saklar pertama salah: %+v", repo.rows[0])
	}
}

func TestTimeframeSet_MenolakAutoTanpaPantau(t *testing.T) {
	svc, repo, _ := siapkanTimeframe(t)

	_, err := svc.Set(context.Background(), "u1", "acc-1", "XAUUSD", dto.UpdateTimeframeSwitchRequest{
		Strategy: "ila-2.0",
		Pairs: []dto.TimeframeSwitchItem{
			{Htf: "W1", Ltf: "H4", Watch: false, Auto: true},
		},
	})
	if err == nil {
		t.Fatal("auto tanpa pantau harus ditolak")
	}
	if len(repo.rows) != 0 {
		t.Fatalf("tidak boleh ada baris tersimpan saat ditolak, dapat %d", len(repo.rows))
	}
}

func TestTimeframeSet_MenolakAutoUntukTierTanpaHak(t *testing.T) {
	svc, _, _ := siapkanTimeframe(t)
	svc.tierOf = func(context.Context, string) (string, error) { return "basic", nil }

	_, err := svc.Set(context.Background(), "u1", "acc-1", "XAUUSD", dto.UpdateTimeframeSwitchRequest{
		Strategy: "ila-2.0",
		Pairs: []dto.TimeframeSwitchItem{
			{Htf: "W1", Ltf: "H4", Watch: true, Auto: true},
		},
	})
	if err == nil {
		t.Fatal("tier tanpa auto_execute harus ditolak saat meminta auto")
	}
}

func TestTimeframeSet_MenolakMetodeYangTidakAktif(t *testing.T) {
	svc, _, _ := siapkanTimeframe(t)

	// Saklar metode hanya menyalakan ila-2.0.
	_, err := svc.Set(context.Background(), "u1", "acc-1", "XAUUSD", dto.UpdateTimeframeSwitchRequest{
		Strategy: "ila-3.0",
		Pairs: []dto.TimeframeSwitchItem{
			{Htf: "W1", Ltf: "H4", Watch: true},
		},
	})
	if err == nil {
		t.Fatal("metode yang tidak dinyalakan akun harus ditolak")
	}
}
