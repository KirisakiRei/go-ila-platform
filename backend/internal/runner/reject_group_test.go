package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// TestRejectGroupLengkap memastikan setiap alasan penolakan punya grup.
// Alasan yang lupa dipetakan akan jatuh ke GroupBatal diam-diam dan membuat
// user diberi tahu setup berakhir padahal masih berjalan.
func TestRejectGroupLengkap(t *testing.T) {
	kasus := map[runner.RejectReason]runner.RejectGroup{
		runner.RejectNone:             runner.GroupNone,
		runner.RejectExpired:          runner.GroupBatal,
		runner.RejectWaitingNext:      runner.GroupBatal,
		runner.RejectNotReady:         runner.GroupBatal,
		runner.RejectTerminal:         runner.GroupBatal,
		runner.RejectBadSignal:        runner.GroupBatal,
		runner.RejectSlippage:         runner.GroupHarga,
		runner.RejectDailyQuota:       runner.GroupKuota,
		runner.RejectDailyWin:         runner.GroupKuota,
		runner.RejectTwoStrike:        runner.GroupKuota,
		runner.RejectOpenPosition:     runner.GroupPosisi,
		runner.RejectLegalPending:     runner.GroupAkun,
		runner.RejectAccountFrozen:    runner.GroupAkun,
		runner.RejectSuspended:        runner.GroupAkun,
		runner.RejectAccountBanned:    runner.GroupAkun,
		runner.RejectDuplicate:        runner.GroupAkun,
		runner.RejectEntitlement:      runner.GroupTerkunci,
		runner.RejectStrategyTier:     runner.GroupTerkunci,
		runner.RejectStrategyDisabled: runner.GroupTerkunci,
		runner.RejectMarketClosed:     runner.GroupPasar,
		runner.RejectMaintenance:      runner.GroupSistem,
		runner.RejectReadOnly:         runner.GroupSistem,
	}

	for reason, ingin := range kasus {
		if got := runner.RejectGroupOf(reason); got != ingin {
			t.Errorf("RejectGroupOf(%q) = %q, ingin %q", reason, got, ingin)
		}
	}
}

// TestRejectTerkunciTidakDikirim memastikan penahanan karena tier atau metode
// tidak pernah menjadi pesan Telegram.
func TestRejectTerkunciTidakDikirim(t *testing.T) {
	for _, reason := range []runner.RejectReason{
		runner.RejectEntitlement,
		runner.RejectStrategyTier,
		runner.RejectStrategyDisabled,
	} {
		if runner.RejectTerkirimKeTelegram(reason) {
			t.Errorf("%q seharusnya tidak dikirim ke Telegram", reason)
		}
	}

	for _, reason := range []runner.RejectReason{
		runner.RejectDailyQuota,
		runner.RejectSlippage,
		runner.RejectMarketClosed,
	} {
		if !runner.RejectTerkirimKeTelegram(reason) {
			t.Errorf("%q seharusnya tetap dikirim ke Telegram", reason)
		}
	}
}

// TestOverrideKuotaHanyaUntukPagarHarian memastikan jalur manual hanya boleh
// menembus batas harian. Pagar lain harus tetap menolak, karena melewatinya
// membahayakan akun atau melanggar aturan platform.
func TestOverrideKuotaHanyaUntukPagarHarian(t *testing.T) {
	boleh := []runner.RejectReason{
		runner.RejectDailyQuota,
		runner.RejectDailyWin,
		runner.RejectTwoStrike,
		runner.RejectOpenPosition,
	}
	for _, reason := range boleh {
		if !runner.OverrideKuotaManual(reason) {
			t.Errorf("%q seharusnya boleh dilewati manual", reason)
		}
	}

	tetapTolak := []runner.RejectReason{
		runner.RejectAccountBanned,
		runner.RejectAccountFrozen,
		runner.RejectSuspended,
		runner.RejectLegalPending,
		runner.RejectSlippage,
		runner.RejectExpired,
		runner.RejectMarketClosed,
		runner.RejectEntitlement,
	}
	for _, reason := range tetapTolak {
		if runner.OverrideKuotaManual(reason) {
			t.Errorf("%q tidak boleh dilewati manual", reason)
		}
	}
}

// TestApproveMenembusKuotaHarian memastikan user tetap bisa eksekusi manual
// setelah kuota otomatis habis. Batas harian hanya menghentikan sistem, bukan
// mencabut keputusan pemilik akun.
func TestApproveMenembusKuotaHarian(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)

	// Kuota otomatis sudah penuh: dua trade selesai hari ini.
	store.daily[dailyKey("u1", runner.DayKey(time.Now()))] = &model.DailySummary{
		UserID:      "u1",
		Day:         runner.DayKey(time.Now()),
		TotalTrades: 2,
		LossCount:   2,
	}

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	reason, err := svc.Approve(context.Background(), "u1", "sig-XAUUSD", "acct-u1")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if reason != runner.RejectNone {
		t.Fatalf("eksekusi manual seharusnya lolos walau kuota habis, ditolak: %q", reason)
	}

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi terbuka = %d, ingin 1", len(positions))
	}
}

// TestApproveMenembusPosisiTerbuka memastikan posisi terbuka tetap menahan
// mode auto, tetapi user tetap boleh mengeksekusi sinyal lain secara manual;
// sinyal yang sama tetap tidak boleh dibuka dua kali (idempotency).
func TestApproveMenembusPosisiTerbuka(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	// Sudah ada satu posisi terbuka di pair lain pada akun yang sama.
	store.positions["pos-existing"] = &model.Position{
		Base:            model.Base{ID: "pos-existing"},
		UserID:          "u1",
		BrokerAccountID: "acct-u1",
		SourceSignalID:  "sig-USTEC",
		Symbol:          "USTEC",
		Direction:       "BUY",
		Status:          runner.PositionOpen,
		OpenedAt:        time.Now().Add(-30 * time.Minute),
	}

	svc := newService(store, pub)
	sig := readySignal("XAUUSD", time.Hour)
	svc.HandleSignal(context.Background(), sig)

	// Mode auto harus tetap menahan dengan RejectOpenPosition.
	us, err := store.FindUserSignal(context.Background(), "u1", sig.SignalId)
	if err != nil {
		t.Fatalf("FindUserSignal: %v", err)
	}
	if us.Status != runner.SignalRejected || us.RejectReason != string(runner.RejectOpenPosition) {
		t.Fatalf("mode auto harus menahan posisi terbuka, got status=%q reason=%q", us.Status, us.RejectReason)
	}
	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("mode auto tidak boleh menambah posisi, got %d", len(positions))
	}

	// Eksekusi manual oleh user harus lolos dan membuka posisi kedua.
	reason, err := svc.Approve(context.Background(), "u1", sig.SignalId, "acct-u1")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if reason != runner.RejectNone {
		t.Fatalf("eksekusi manual seharusnya lolos walau ada posisi terbuka, ditolak: %q", reason)
	}

	positions, _ = store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 2 {
		t.Fatalf("posisi terbuka = %d, ingin 2", len(positions))
	}

	// Eksekusi manual kedua untuk sinyal yang SAMA harus tetap ditolak.
	reason2, err := svc.Approve(context.Background(), "u1", sig.SignalId, "acct-u1")
	if err != nil {
		t.Fatalf("Approve ulang: %v", err)
	}
	if reason2 == runner.RejectNone {
		t.Fatal("sinyal yang sama tidak boleh dieksekusi dua kali")
	}
}

// TestApproveTetapMenolakAkunDibanned memastikan override kuota tidak ikut
// membuka pagar akun. Ban berlaku mutlak.
func TestApproveTetapMenolakAkunDibanned(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "basic", "manual", true)
	store.users["u1"].Status = "banned"

	store.daily[dailyKey("u1", runner.DayKey(time.Now()))] = &model.DailySummary{
		UserID: "u1", Day: runner.DayKey(time.Now()), TotalTrades: 5,
	}

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	reason, err := svc.Approve(context.Background(), "u1", "sig-XAUUSD", "acct-u1")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if reason == runner.RejectNone {
		t.Fatal("akun dibanned tidak boleh lolos walau lewat eksekusi manual")
	}

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatalf("posisi terbuka = %d, ingin 0 untuk akun dibanned", len(positions))
	}
}
