package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// Nilai hasil dalam R harus memakai risiko dalam UANG, bukan jarak harga.
//
// Bug nyata di produksi: posisi EURUSD 0,66 lot, entry 1,10821, SL 1,10985,
// exit 1,10506, profit 207,90 USD tercatat 192.073,17R. Angka itu muncul karena
// profit (USD) dibagi lot x jarak harga (0,0010824) — besaran tanpa satuan yang
// tidak sebanding.
//
// Hasil sebenarnya: risiko 0,66 x 0,00164 x 100.000 = 108,24 USD, sehingga
// R = 207,90 / 108,24 = 1,92.
func TestOutcomeRMenggunakanRisikoDalamUang(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	svc.SetClock(func() time.Time { return now })

	position := &model.Position{
		Base:         model.Base{ID: "pos-rr"},
		UserID:       "u-rr",
		Symbol:       "EURUSD",
		Status:       model.PositionStatusOpen,
		TicketNumber: "T-RR",
		OpenPrice:    1.10821,
		StopLoss:     1.10985,
		TakeProfit:   1.10506,
		LotSize:      0.66,
		OpenedAt:     now.Add(-time.Hour),
	}

	svc.ApplyClose(context.Background(), position, model.PositionStatusClosedTP, 1.10506, 207.90)

	if position.OutcomeR < 1.9 || position.OutcomeR > 1.94 {
		t.Fatalf("outcome R = %v, ingin sekitar 1.92", position.OutcomeR)
	}
}

// Perilaku yang sama harus berlaku untuk pair bernilai kontrak berbeda.
func TestOutcomeRUntukEmas(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	svc.SetClock(func() time.Time { return now })

	// Emas: 1 lot = 100 ons. Risiko 0,10 lot x 4,00 = 40 USD.
	position := &model.Position{
		Base:         model.Base{ID: "pos-xau"},
		UserID:       "u-xau",
		Symbol:       "XAUUSD",
		Status:       model.PositionStatusOpen,
		TicketNumber: "T-XAU",
		OpenPrice:    4050.00,
		StopLoss:     4046.00,
		LotSize:      0.10,
		OpenedAt:     now.Add(-time.Hour),
	}

	// Profit 80 USD berarti menang 2R.
	svc.ApplyClose(context.Background(), position, model.PositionStatusClosedTP, 4058.00, 80)

	if position.OutcomeR < 1.99 || position.OutcomeR > 2.01 {
		t.Fatalf("outcome R emas = %v, ingin 2.00", position.OutcomeR)
	}
}

// Posisi tanpa SL tidak punya risiko terukur: R harus dibiarkan nol, bukan
// dibagi dengan angka yang mendekati nol sehingga meledak.
func TestOutcomeRTanpaStopLossTidakDihitung(t *testing.T) {
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	svc := newService(store, &fakePublisher{})
	svc.SetClock(func() time.Time { return now })

	position := &model.Position{
		Base:         model.Base{ID: "pos-nosl"},
		UserID:       "u-nosl",
		Symbol:       "EURUSD",
		Status:       model.PositionStatusOpen,
		TicketNumber: "T-NOSL",
		OpenPrice:    1.1,
		StopLoss:     0,
		LotSize:      1,
		OpenedAt:     now.Add(-time.Hour),
	}

	svc.ApplyClose(context.Background(), position, model.PositionStatusClosedManual, 1.11, 100)

	if position.OutcomeR != 0 {
		t.Fatalf("outcome R tanpa SL = %v, ingin 0", position.OutcomeR)
	}
}

func TestDefaultValuePerLot(t *testing.T) {
	kasus := map[string]float64{
		"EURUSD": 100000,
		"USDJPY": 1000,
		"XAUUSD": 100,
		"GOLD":   100,
		"XAGUSD": 5000,
		"USTEC":  1,
		"NAS100": 1,
		"BTCUSD": 1,
		"USOIL":  1000,
	}
	for simbol, ingin := range kasus {
		if hasil := runner.DefaultValuePerLot(simbol); hasil != ingin {
			t.Errorf("DefaultValuePerLot(%q) = %v, ingin %v", simbol, hasil, ingin)
		}
	}
}
