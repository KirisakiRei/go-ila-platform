package service

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
)

func TestSignalsDeduplikasiKohortDanWariskanPosisi(t *testing.T) {
	now := time.Date(2026, 11, 18, 7, 0, 0, 0, time.UTC)

	// Setup USTEC identik terbit dua kali (05:35 dieksekusi OPEN, 06:00 REJECTED/EXPIRED).
	t1 := time.Date(2026, 11, 18, 5, 35, 5, 0, time.UTC)
	t2 := time.Date(2026, 11, 18, 6, 0, 5, 0, time.UTC)

	sig1 := model.Signal{
		SignalID:        "sig-USTEC-11",
		Symbol:          "USTEC",
		StrategyVersion: "ila-2.0",
		Htf:             "H1",
		Ltf:             "M5",
		Direction:       "SELL",
		EntryPrice:      27146.0,
		StopLoss:        27190.3,
		TakeProfit:      27044.11,
		RiskReward:      2.3,
		Status:          model.SignalStatusExpired,
		PublishedAt:     t1,
		ExpiresAt:       t1.Add(5 * time.Minute),
	}
	sig2 := sig1
	sig2.SignalID = "sig-USTEC-12"
	sig2.PublishedAt = t2
	sig2.ExpiresAt = t2.Add(5 * time.Minute)
	sig2.SetupEvalStatus = model.EvalRunning

	// Empat terbitan ulang XAUUSD identik dalam jendela setup yang sama, tanpa eksekusi, sudah kena TP.
	tpAt := time.Date(2026, 11, 17, 20, 15, 0, 0, time.UTC)
	var xauRows []model.Signal
	for i, menit := range []int{0, 30, 60, 90} {
		pub := time.Date(2026, 11, 17, 15, menit, 0, 0, time.UTC)
		xauRows = append(xauRows, model.Signal{
			SignalID:        "sig-XAU-" + string(rune('1'+i)),
			Symbol:          "XAUUSD",
			StrategyVersion: "ila-2.0",
			Htf:             "H4",
			Ltf:             "M15",
			Direction:       "SELL",
			EntryPrice:      3877.72,
			StopLoss:        3894.865,
			TakeProfit:      3843.43,
			RiskReward:      2.0,
			Status:          model.SignalStatusExpired,
			PublishedAt:     pub,
			ExpiresAt:       pub.Add(15 * time.Minute),
			SetupEvalStatus: model.EvalTP,
			SetupEvalAt:     &tpAt,
			SetupEvalRLower: 2.0,
		})
	}

	// Baris radar tidak boleh ikut digabung atau hilang.
	radar := model.Signal{
		Base:            model.Base{UpdatedAt: now.Add(-time.Minute)},
		SignalID:        "status-USTEC-H1-M5",
		Symbol:          "USTEC",
		StrategyVersion: "ila-2.0",
		Htf:             "H1",
		Ltf:             "M5",
		Direction:       "SELL",
		Status:          model.SignalStatusWaitHTFSweep,
		PublishedAt:     now.Add(-48 * time.Hour),
		ExpiresAt:       now.Add(time.Hour),
	}

	riwayat := append([]model.Signal{sig2, sig1}, xauRows...)

	users := newFakeUserRepo()
	users.users["u1@contoh.id"] = newTestUser("plus")

	positions := &fakePositionLister{
		positions: []model.Position{
			{
				Base:            model.Base{ID: "pos-open-1"},
				UserID:          userIDTest,
				SourceSignalID:  "sig-USTEC-11",
				Symbol:          "USTEC",
				StrategyVersion: "ila-2.0",
				Direction:       "SELL",
				OpenPrice:       27139.96,
				StopLoss:        27190.3,
				TakeProfit:      27044.11,
				Status:          model.PositionStatusOpen,
				OpenedAt:        t1.Add(19 * time.Second),
			},
		},
	}

	userSignals := &fakeUserSignalReader{
		status: map[string]*model.UserSignal{
			"sig-USTEC-11": {SignalID: "sig-USTEC-11", UserID: userIDTest, Status: model.SignalStatusConsumed, PositionID: "pos-open-1"},
			"sig-USTEC-12": {SignalID: "sig-USTEC-12", UserID: userIDTest, Status: model.SignalStatusRejected, RejectReason: "Posisi masih berjalan"},
		},
	}

	svc := NewTradingService(users, &fakeSignalLister{signals: riwayat, radar: []model.Signal{radar}}, positions, userSignals)
	svc.clock = func() time.Time { return now }

	got, err := svc.Signals(context.Background(), userIDTest, 50)
	if err != nil {
		t.Fatalf("Signals gagal: %v", err)
	}

	// Harus menghasilkan tepat 3 item: 1 radar + 1 setup USTEC + 1 setup XAUUSD (bukan 1 + 2 + 4 = 7).
	if len(got) != 3 {
		t.Fatalf("jumlah sinyal terdeduplikasi = %d, ingin 3: %+v", len(got), got)
	}

	if got[0].SignalID != radar.SignalID {
		t.Fatalf("radar harus tetap di indeks 0, dapat %q", got[0].SignalID)
	}

	ustec := got[1]
	if ustec.Symbol != "USTEC" || !ustec.IsExecuted || ustec.PositionStatus != model.PositionStatusOpen {
		t.Fatalf("setup USTEC harus mewarisi posisi OPEN: %+v", ustec)
	}
	if ustec.EvalStatus != model.EvalRunning {
		t.Fatalf("setup USTEC harus tetap mewarisi eval_status RUNNING, dapat %q", ustec.EvalStatus)
	}

	xau := got[2]
	if xau.Symbol != "XAUUSD" || xau.EvalStatus != model.EvalTP || xau.EvalRLower != 2.0 {
		t.Fatalf("setup XAUUSD harus berstatus TP +2R: %+v", xau)
	}
}

func TestSignalsSiklusBaruSetelahTPSelesaiTidakDigabung(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 11, 18, 6, 0, 0, 0, time.UTC)
	tpDone := time.Date(2026, 11, 18, 7, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

	s1 := newSignal("sig-awal", t1, t1.Add(15*time.Minute))
	s1.Status = model.SignalStatusExpired
	s1.SetupEvalStatus = model.EvalTP
	s1.SetupEvalAt = &tpDone
	s1.SetupEvalRLower = 3.0

	s2 := newSignal("sig-siklus-baru", t2, t2.Add(15*time.Minute))
	s2.Status = model.SignalStatusExpired
	s2.SetupEvalStatus = model.EvalSL
	s2.SetupEvalRLower = -1.0

	svc := newTradingServiceFor(t, newTestUser("plus"), []model.Signal{s2, s1}, now)
	got, err := svc.Signals(context.Background(), userIDTest, 50)
	if err != nil {
		t.Fatalf("Signals gagal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("setup baru setelah TP selesai harus tampil terpisah (2 kartu), dapat %d", len(got))
	}
}
