package ingest

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	enginev1 "ila/backend/gen/engine/v1"
	"ila/backend/internal/model"
)

func validSignal() *enginev1.Signal {
	published := time.Date(2026, 11, 6, 10, 15, 0, 0, time.UTC)
	return &enginev1.Signal{
		SignalId:        "sig-20261106-1015-a3f",
		Symbol:          "XAUUSD",
		Htf:             "H4",
		Ltf:             "M15",
		Status:          enginev1.SignalStatus_READY_TO_ENTRY,
		Action:          enginev1.SignalAction_SIGNAL_ACTION_PUBLISH,
		Direction:       enginev1.Direction_BUY,
		EntryPrice:      4120.50,
		StopLoss:        4112.30,
		TakeProfit:      4149.20,
		RiskReward:      2.30,
		PublishedAt:     timestamppb.New(published),
		ExpiresAt:       timestamppb.New(published.Add(45 * time.Minute)),
		Summary:         "Setup sesuai rencana.",
		StrategyVersion: "ila-2.0",
		EngineVersion:   "a41c167",
	}
}

func TestConvertSignal(t *testing.T) {
	got, ok := ConvertSignal(validSignal())
	if !ok {
		t.Fatal("sinyal valid seharusnya diterima")
	}

	if got.SignalID != "sig-20261106-1015-a3f" {
		t.Fatalf("SignalID = %q", got.SignalID)
	}
	if got.Direction != "BUY" {
		t.Fatalf("Direction = %q", got.Direction)
	}
	if got.Status != model.SignalStatusReadyToEntry {
		t.Fatalf("Status = %q", got.Status)
	}
	if got.StrategyVersion != "ila-2.0" {
		t.Fatalf("StrategyVersion = %q", got.StrategyVersion)
	}
	if got.ExpiresAt.Sub(got.PublishedAt) != 45*time.Minute {
		t.Fatal("TTL tidak terjaga")
	}
}

func TestConvertSignalRejectsIncomplete(t *testing.T) {
	cases := map[string]*enginev1.Signal{
		"nil":              nil,
		"tanpa id":         {Symbol: "XAUUSD", Direction: enginev1.Direction_BUY},
		"tanpa simbol":     {SignalId: "sig-1", Direction: enginev1.Direction_BUY},
		"tanpa arah":       {SignalId: "sig-1", Symbol: "XAUUSD"},
		"arah tak dikenal": {SignalId: "sig-1", Symbol: "XAUUSD", Direction: enginev1.Direction_DIRECTION_UNSPECIFIED},
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, ok := ConvertSignal(input); ok {
				t.Fatal("sinyal tidak lengkap seharusnya ditolak")
			}
		})
	}
}

// Sinyal tanpa expires_at tidak boleh mendapat TTL tak terbatas.
func TestConvertSignalWithoutExpiryBecomesImmediatelyStale(t *testing.T) {
	input := validSignal()
	input.ExpiresAt = nil

	got, ok := ConvertSignal(input)
	if !ok {
		t.Fatal("sinyal masih bisa dipetakan")
	}
	if !got.ExpiresAt.Equal(got.PublishedAt) {
		t.Fatalf("ExpiresAt = %v, ingin sama dengan PublishedAt", got.ExpiresAt)
	}
}

func TestConvertSignalSellDirection(t *testing.T) {
	input := validSignal()
	input.Direction = enginev1.Direction_SELL

	got, ok := ConvertSignal(input)
	if !ok {
		t.Fatal("sinyal SELL seharusnya diterima")
	}
	if got.Direction != "SELL" {
		t.Fatalf("Direction = %q", got.Direction)
	}
}

func TestUpdateStatus(t *testing.T) {
	cases := map[enginev1.SignalAction]string{
		enginev1.SignalAction_SIGNAL_ACTION_CANCEL:    model.SignalStatusCancelled,
		enginev1.SignalAction_SIGNAL_ACTION_EXPIRE:    model.SignalStatusExpired,
		enginev1.SignalAction_SIGNAL_ACTION_SUPERSEDE: model.SignalStatusSuperseded,
		enginev1.SignalAction_SIGNAL_ACTION_PUBLISH:   "",
	}

	for action, want := range cases {
		if got := UpdateStatus(action); got != want {
			t.Fatalf("UpdateStatus(%v) = %q, ingin %q", action, got, want)
		}
	}
}

func TestMarketOpen(t *testing.T) {
	if !MarketOpen(enginev1.MarketState_MARKET_OPEN) {
		t.Fatal("MARKET_OPEN harus true")
	}
	if MarketOpen(enginev1.MarketState_MARKET_CLOSED) {
		t.Fatal("MARKET_CLOSED harus false")
	}
}

// Sinyal ILA 3.0 harus diterima. Sebelum pemetaan ini, status ILA 3.0 tidak
// dikenal sehingga sinyalnya dibuang tanpa jejak.
func TestConvertSignalILA3(t *testing.T) {
	cases := []struct {
		name   string
		status enginev1.SignalStatus
		want   string
	}{
		{"entry 1 reversal", enginev1.SignalStatus_READY_ENTRY_1, model.SignalStatusReadyToEntry},
		{"entry 2 retrace", enginev1.SignalStatus_READY_ENTRY_2, model.SignalStatusReadyToEntry},
		{"menunggu reversal", enginev1.SignalStatus_WAIT_ILA3_MTF_REVERSAL, "WAIT_ILA3_MTF_REVERSAL"},
		{"menunggu fvg retrace", enginev1.SignalStatus_WAIT_ILA3_MTF_FVG_RETR, "WAIT_ILA3_MTF_FVG_RETR"},
		{"target tercapai", enginev1.SignalStatus_ILA3_TARGET_REACHED, "ILA3_TARGET_REACHED"},
	}

	for _, c := range cases {
		in := validSignal()
		in.Status = c.status
		in.StrategyVersion = "ila-3.0"
		in.SequenceState = c.want

		got, ok := ConvertSignal(in)
		if !ok {
			t.Fatalf("%s: sinyal ILA 3.0 ditolak", c.name)
		}
		if got.Status != c.want {
			t.Errorf("%s: status = %q, ingin %q", c.name, got.Status, c.want)
		}
	}
}

// Dua titik entry ILA 3.0 harus bisa dieksekusi, bukan hanya tampil.
func TestILA3ReadyBolehDieksekusi(t *testing.T) {
	for _, st := range []enginev1.SignalStatus{
		enginev1.SignalStatus_READY_ENTRY_1,
		enginev1.SignalStatus_READY_ENTRY_2,
	} {
		got, ok := ConvertSignal(validSignalWithStatus(st))
		if !ok || got.Status != model.SignalStatusReadyToEntry {
			t.Fatalf("status %v harus menjadi READY_TO_ENTRY", st)
		}
	}
}

func validSignalWithStatus(st enginev1.SignalStatus) *enginev1.Signal {
	in := validSignal()
	in.Status = st
	return in
}
