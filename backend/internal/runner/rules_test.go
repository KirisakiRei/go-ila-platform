package runner

import (
	"errors"
	"testing"
	"time"
)

func TestEvaluateDaily(t *testing.T) {
	policy := DefaultDailyPolicy().Clamp()

	cases := []struct {
		name   string
		status DailyStatus
		want   RejectReason
	}{
		{
			name:   "hari bersih boleh entry",
			status: DailyStatus{},
			want:   RejectNone,
		},
		{
			name:   "posisi open menahan entry",
			status: DailyStatus{OpenCount: 1},
			want:   RejectOpenPosition,
		},
		{
			name:   "posisi open mengalahkan aturan lain",
			status: DailyStatus{OpenCount: 1, WinCount: 1, TotalTrades: 2},
			want:   RejectOpenPosition,
		},
		{
			name:   "satu win menghentikan hari",
			status: DailyStatus{WinCount: 1, TotalTrades: 1},
			want:   RejectDailyWin,
		},
		{
			name:   "win mengalahkan kuota",
			status: DailyStatus{WinCount: 1, TotalTrades: 2},
			want:   RejectDailyWin,
		},
		{
			name:   "kuota dua trade habis",
			status: DailyStatus{TotalTrades: 2, LossCount: 2},
			want:   RejectDailyQuota,
		},
		{
			name:   "dua SL menghentikan hari",
			status: DailyStatus{TotalTrades: 1, LossCount: 2},
			want:   RejectTwoStrike,
		},
		{
			name:   "satu loss masih boleh",
			status: DailyStatus{TotalTrades: 1, LossCount: 1},
			want:   RejectNone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EvaluateDaily(tc.status, policy); got != tc.want {
				t.Fatalf("EvaluateDaily(%+v) = %q, ingin %q", tc.status, got, tc.want)
			}
		})
	}
}

// Stop-after-win tidak boleh bisa dimatikan siapa pun.
func TestDailyPolicyClampForcesStopAfterWin(t *testing.T) {
	p := DailyPolicy{MaxTrades: 99, StopAfterWin: false, MaxLosses: 0}.Clamp()

	if !p.StopAfterWin {
		t.Fatal("StopAfterWin harus tetap aktif")
	}
	if p.MaxTrades != 3 {
		t.Fatalf("MaxTrades = %d, ingin dibatasi ke 3", p.MaxTrades)
	}
	if p.MaxLosses != 1 {
		t.Fatalf("MaxLosses = %d, ingin dinaikkan ke minimal 1", p.MaxLosses)
	}
}

func TestDayKeyUsesWIB(t *testing.T) {
	// 18:00 UTC tanggal 5 = 01:00 WIB tanggal 6.
	utc := time.Date(2026, 11, 5, 18, 0, 0, 0, time.UTC)
	if got := DayKey(utc); got != "2026-11-06" {
		t.Fatalf("DayKey = %q, ingin 2026-11-06", got)
	}

	// 16:59 UTC tanggal 5 masih tanggal 5 di WIB.
	before := time.Date(2026, 11, 5, 16, 59, 0, 0, time.UTC)
	if got := DayKey(before); got != "2026-11-05" {
		t.Fatalf("DayKey = %q, ingin 2026-11-05", got)
	}
}

func TestDayStartIsMidnightWIB(t *testing.T) {
	target := time.Date(2026, 11, 5, 20, 30, 0, 0, time.UTC)
	start := DayStart(target)

	if start.Hour() != 0 || start.Minute() != 0 {
		t.Fatalf("DayStart = %v, ingin tengah malam WIB", start)
	}
	// 00:00 WIB = 17:00 UTC hari sebelumnya.
	if start.UTC().Hour() != 17 {
		t.Fatalf("DayStart UTC hour = %d, ingin 17", start.UTC().Hour())
	}
	if end := DayEnd(target); !end.Equal(start.AddDate(0, 0, 1)) {
		t.Fatalf("DayEnd tidak sama dengan awal hari berikutnya")
	}
}

func TestSlippageCheckUsesSymbolTolerance(t *testing.T) {
	// XAUUSD toleransi 0.50; jarak SL 8.20 memberi 10% = 0.82.
	ok, drift, limit := SlippageCheck("XAUUSD", 4120.50, 4120.90, 4112.30)
	if !ok {
		t.Fatalf("drift %v seharusnya masih dalam batas %v", drift, limit)
	}
	if limit < 0.8 || limit > 0.83 {
		t.Fatalf("limit = %v, ingin sekitar 0.82", limit)
	}

	ok, _, _ = SlippageCheck("XAUUSD", 4120.50, 4122.00, 4112.30)
	if ok {
		t.Fatal("pergerakan 1.5 seharusnya ditolak")
	}
}

func TestSlippageCheckForexTolerance(t *testing.T) {
	// EURUSD pip 0.0001; jarak SL 0.0050 memberi 10% = 0.00050.
	ok, _, limit := SlippageCheck("EURUSD", 1.08500, 1.08530, 1.08000)
	if !ok {
		t.Fatalf("drift kecil seharusnya lolos, limit %v", limit)
	}

	ok, _, _ = SlippageCheck("EURUSD", 1.08500, 1.08700, 1.08000)
	if ok {
		t.Fatal("pergerakan 20 pip seharusnya ditolak")
	}
}

func TestSlippageRejectsUnknownSymbolWithoutStop(t *testing.T) {
	// Tanpa toleransi dan tanpa jarak SL, sistem menolak alih-alih menebak.
	ok, _, _ := SlippageCheck("UNKNOWN", 100, 100.1, 100)
	if ok {
		t.Fatal("simbol tanpa spesifikasi dan tanpa SL harus ditolak")
	}
}

func TestCalculateLotBasic(t *testing.T) {
	calc, err := CalculateLot(LotRequest{
		Equity:      10_000,
		RiskPercent: 0.5,
		EntryPrice:  4120.50,
		StopLoss:    4112.30,
		Spec: LotSpec{
			ValuePerLotPerPrice: 100,
			MinLot:              0.01,
			MaxLot:              50,
			LotStep:             0.01,
		},
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}

	// risiko $50, jarak 8.2, nilai/lot 820 → 0.0609 → dibulatkan ke bawah 0.06.
	if calc.Lots != 0.06 {
		t.Fatalf("Lots = %v, ingin 0.06", calc.Lots)
	}
	if calc.RiskAmount != 50 {
		t.Fatalf("RiskAmount = %v, ingin 50", calc.RiskAmount)
	}
}

func TestCalculateLotClampsRiskPercent(t *testing.T) {
	calc, err := CalculateLot(LotRequest{
		Equity:      10_000,
		RiskPercent: 5, // melampaui pagar 1%
		EntryPrice:  1.0850,
		StopLoss:    1.0800,
		Spec:        LotSpec{ValuePerLotPerPrice: 100_000, MinLot: 0.01, MaxLot: 100, LotStep: 0.01},
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if calc.RiskPercent != MaxRiskPercent {
		t.Fatalf("RiskPercent = %v, ingin dibatasi %v", calc.RiskPercent, MaxRiskPercent)
	}
}

func TestCalculateLotClampsToMinimum(t *testing.T) {
	calc, err := CalculateLot(LotRequest{
		Equity:      100,
		RiskPercent: 0.1,
		EntryPrice:  1.0850,
		StopLoss:    1.0800,
		Spec:        LotSpec{ValuePerLotPerPrice: 100_000, MinLot: 0.10, MaxLot: 100, LotStep: 0.01},
	})
	if err != nil {
		t.Fatalf("tidak mengharapkan error: %v", err)
	}
	if calc.Lots != 0.10 || !calc.ClampedMin {
		t.Fatalf("Lots = %v clampMin = %v, ingin 0.10 dan true", calc.Lots, calc.ClampedMin)
	}
}

func TestCalculateLotRejectsZeroStop(t *testing.T) {
	_, err := CalculateLot(LotRequest{
		Equity:      10_000,
		RiskPercent: 0.5,
		EntryPrice:  4120,
		StopLoss:    4120,
		Spec:        LotSpec{ValuePerLotPerPrice: 100},
	})
	if !errors.Is(err, ErrNoLot) {
		t.Fatalf("err = %v, ingin ErrNoLot", err)
	}
}

func TestCalculateLotRejectsZeroEquity(t *testing.T) {
	_, err := CalculateLot(LotRequest{
		Equity:      0,
		RiskPercent: 0.5,
		EntryPrice:  4120,
		StopLoss:    4112,
		Spec:        LotSpec{ValuePerLotPerPrice: 100},
	})
	if !errors.Is(err, ErrNoLot) {
		t.Fatalf("err = %v, ingin ErrNoLot", err)
	}
}

func TestEffectiveTier(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	if got := EffectiveTier("free", &future, nil, now); got != "plus" {
		t.Fatalf("trial aktif = %q, ingin plus", got)
	}
	if got := EffectiveTier("basic", &past, nil, now); got != "basic" {
		t.Fatalf("trial habis = %q, ingin basic", got)
	}
	if got := EffectiveTier("plus", nil, &past, now); got != "free" {
		t.Fatalf("langganan habis = %q, ingin free", got)
	}
}

func TestModeAndStrategyEntitlement(t *testing.T) {
	if ModeAllowed("basic", "auto") {
		t.Fatal("basic tidak boleh auto")
	}
	if !ModeAllowed("plus", "auto") {
		t.Fatal("plus harus boleh auto")
	}
	if !ModeAllowed("free", "manual") {
		t.Fatal("manual boleh untuk semua tier")
	}

	if StrategyAllowed("basic", "ila-3.0") {
		t.Fatal("ila-3.0 hanya untuk plus")
	}
	if !StrategyAllowed("plus", "ila-3.0") {
		t.Fatal("plus harus boleh ila-3.0")
	}
	if !StrategyAllowed("free", "ila-2.0") {
		t.Fatal("ila-2.0 tersedia untuk semua tier")
	}
}

func TestMaxAccountsPerTier(t *testing.T) {
	if MaxAccounts("free") != 0 {
		t.Fatal("free tidak boleh punya akun nyata")
	}
	if MaxAccounts("basic") != 1 {
		t.Fatal("basic hanya satu akun")
	}
	if MaxAccounts("plus") != 2 {
		t.Fatal("plus maksimal dua akun")
	}
}

func TestIsTerminal(t *testing.T) {
	for _, s := range []string{SignalConsumed, SignalExpired, SignalCancelled, SignalSuperseded, SignalRejected, SignalSkipped} {
		if !IsTerminal(s) {
			t.Fatalf("%q seharusnya terminal", s)
		}
	}
	for _, s := range []string{SignalReady, SignalAcked} {
		if IsTerminal(s) {
			t.Fatalf("%q seharusnya belum terminal", s)
		}
	}
}

func TestWithinGrace(t *testing.T) {
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)

	if WithinGrace(&future, now) {
		t.Fatal("grace yang belum lewat tidak memblokir")
	}
	if !WithinGrace(&past, now) {
		t.Fatal("grace yang sudah lewat harus memblokir entry")
	}
	if WithinGrace(nil, now) {
		t.Fatal("tanpa grace tidak memblokir")
	}
}

func TestIdempotencyKey(t *testing.T) {
	key := IdempotencyKey("sig-1", "acct-1")
	if key != "exec:sig-1:acct-1" {
		t.Fatalf("key = %q", key)
	}
	if key == IdempotencyKey("sig-1", "acct-2") {
		t.Fatal("akun berbeda harus menghasilkan kunci berbeda")
	}
}
