package service

import (
	"context"
	"testing"

	"ila/backend/internal/dto"
)

// fakeBrokerSpec memberi spesifikasi broker tiruan untuk menguji lapisan C.
type fakeBrokerSpec struct {
	spec SpecBroker
	ok   bool
}

func (f fakeBrokerSpec) SymbolSpec(_ context.Context, _, _, _ string) (SpecBroker, bool) {
	return f.spec, f.ok
}

func TestSimulasiRisiko_ModePersenXAUUSD(t *testing.T) {
	// XAUUSD: nilai per lot 100, berhenti 8.2, ekuitas 10.000, risiko 0.5%.
	// lots = (10000 x 0.5%) / (8.2 x 100) = 50 / 820 = 0.0609 -> 0.06
	sim := NewRiskSimulator(nil, nil, nil)
	res, err := sim.SimulateRisk(context.Background(), "user", dto.RiskSimRequest{
		Symbol: "XAUUSD", EntryPrice: 4120.5, StopLoss: 4112.3,
		Equity: 10000, Mode: "percent", RiskPercent: 0.5,
	})
	if err != nil {
		t.Fatalf("simulasi: %v", err)
	}
	if res.ValuePerLot != 100 {
		t.Errorf("nilai per lot = %v, ingin 100", res.ValuePerLot)
	}
	if res.Hasil.Lots != 0.06 {
		t.Errorf("lot = %v, ingin 0.06", res.Hasil.Lots)
	}
	if res.Hasil.RiskAmount != 49.2 {
		t.Errorf("risiko nominal = %v, ingin 49.2", res.Hasil.RiskAmount)
	}
	if len(res.Tabel) != 5 {
		t.Fatalf("tabel persen harus 5 baris, dapat %d", len(res.Tabel))
	}
	if res.Tabel[0].Warning {
		t.Error("baris 1% tidak boleh berperingatan")
	}
	if !res.Tabel[4].Warning {
		t.Error("baris 5% harus berperingatan")
	}
}

func TestSimulasiRisiko_ModeLot(t *testing.T) {
	// 0.50 lot XAUUSD, berhenti 8.2 -> risiko 410, pada ekuitas 10.000 = 4.1%.
	sim := NewRiskSimulator(nil, nil, nil)
	res, err := sim.SimulateRisk(context.Background(), "user", dto.RiskSimRequest{
		Symbol: "XAUUSD", EntryPrice: 4120.5, StopLoss: 4112.3,
		Equity: 10000, Mode: "lot", LotSize: 0.5,
	})
	if err != nil {
		t.Fatalf("simulasi: %v", err)
	}
	if res.Hasil.Lots != 0.5 {
		t.Errorf("lot = %v, ingin 0.5", res.Hasil.Lots)
	}
	if res.Hasil.RiskAmount != 410 {
		t.Errorf("risiko nominal = %v, ingin 410", res.Hasil.RiskAmount)
	}
	if res.Hasil.RiskPercent != 4.1 {
		t.Errorf("persen efektif = %v, ingin 4.1", res.Hasil.RiskPercent)
	}
	if !res.Hasil.Warning {
		t.Error("4.1% harus berperingatan")
	}
}

func TestSimulasiRisiko_LotDibatasiMaksimum(t *testing.T) {
	sim := NewRiskSimulator(nil, nil, nil)
	res, err := sim.SimulateRisk(context.Background(), "user", dto.RiskSimRequest{
		Symbol: "EURUSD", EntryPrice: 1.0850, StopLoss: 1.0800,
		Equity: 10000, Mode: "lot", LotSize: 50,
	})
	if err != nil {
		t.Fatalf("simulasi: %v", err)
	}
	if res.Hasil.Lots != 10 {
		t.Errorf("lot harus dipagari 10, dapat %v", res.Hasil.Lots)
	}
}

func TestSimulasiRisiko_MenolakStopSamaDenganEntry(t *testing.T) {
	sim := NewRiskSimulator(nil, nil, nil)
	if _, err := sim.SimulateRisk(context.Background(), "user", dto.RiskSimRequest{
		Symbol: "XAUUSD", EntryPrice: 4120, StopLoss: 4120, Equity: 10000,
	}); err == nil {
		t.Fatal("stop sama dengan entry harus ditolak")
	}
}

func TestSimulasiRisiko_MenolakTanpaEkuitas(t *testing.T) {
	sim := NewRiskSimulator(nil, nil, nil)
	if _, err := sim.SimulateRisk(context.Background(), "user", dto.RiskSimRequest{
		Symbol: "XAUUSD", EntryPrice: 4120, StopLoss: 4112,
	}); err == nil {
		t.Fatal("tanpa ekuitas harus ditolak")
	}
}

func TestSimulasiRisiko_SpecBrokerMenimpaKatalog(t *testing.T) {
	sim := NewRiskSimulator(nil, nil, fakeBrokerSpec{
		ok:   true,
		spec: SpecBroker{ValuePerLot: 10, MinLot: 0.01, MaxLot: 10, LotStep: 0.01},
	})
	res, err := sim.SimulateRisk(context.Background(), "user", dto.RiskSimRequest{
		AccountID: "akun-1", Symbol: "XAUUSD", EntryPrice: 4120, StopLoss: 4112,
		Equity: 10000, Mode: "percent", RiskPercent: 1,
	})
	if err != nil {
		t.Fatalf("simulasi: %v", err)
	}
	if res.SumberNilai != "broker" {
		t.Errorf("sumber = %q, ingin broker", res.SumberNilai)
	}
	if res.ValuePerLot != 10 {
		t.Errorf("nilai per lot = %v, ingin 10", res.ValuePerLot)
	}
}

func TestSimulasiRisiko_TabelLotBerisiNilaiBatas(t *testing.T) {
	sim := NewRiskSimulator(nil, nil, nil)
	res, err := sim.SimulateRisk(context.Background(), "user", dto.RiskSimRequest{
		Symbol: "EURUSD", EntryPrice: 1.0850, StopLoss: 1.0800,
		Equity: 10000, Mode: "percent", RiskPercent: 0.5,
	})
	if err != nil {
		t.Fatalf("simulasi: %v", err)
	}
	if len(res.TabelLot) == 0 {
		t.Fatal("tabel lot tidak boleh kosong")
	}
	if res.TabelLot[0].Lots != 0.01 {
		t.Errorf("lot pertama = %v, ingin 0.01", res.TabelLot[0].Lots)
	}
}
