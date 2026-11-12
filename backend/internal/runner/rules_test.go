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
	// XAUUSD toleransi dasar 0.75; jarak SL 8.20 memberi 15% = 1.23 sehingga
	// persentase SL yang berlaku (lebih longgar daripada tabel pair).
	ok, drift, limit := SlippageCheck("XAUUSD", 4120.50, 4120.90, 4112.30)
	if !ok {
		t.Fatalf("drift %v seharusnya masih dalam batas %v", drift, limit)
	}
	if limit < 1.22 || limit > 1.24 {
		t.Fatalf("limit = %v, ingin sekitar 1.23", limit)
	}

	// 1.5 masih di dalam batas 1.23? Tidak: 1.5 > 1.23, harus ditolak.
	ok, _, _ = SlippageCheck("XAUUSD", 4120.50, 4122.00, 4112.30)
	if ok {
		t.Fatal("pergerakan 1.5 seharusnya ditolak")
	}

	// 1.0 masih lolos: bukti toleransi benar-benar dilonggarkan.
	ok, _, _ = SlippageCheck("XAUUSD", 4120.50, 4121.50, 4112.30)
	if !ok {
		t.Fatal("pergerakan 1.0 pada SL 8.20 seharusnya masih lolos")
	}
}

func TestSlippageCheckForexTolerance(t *testing.T) {
	// EURUSD toleransi dasar 10 pip (0.00100). SL 0.0050 memberi 15% = 0.00075,
	// jadi tabel pair yang berlaku.
	ok, _, limit := SlippageCheck("EURUSD", 1.08500, 1.08530, 1.08000)
	if !ok {
		t.Fatalf("drift kecil seharusnya lolos, limit %v", limit)
	}
	if limit < 0.00099 || limit > 0.00101 {
		t.Fatalf("limit = %v, ingin sekitar 0.00100 (10 pip)", limit)
	}

	// 10 pip persis: harus lolos (batas inklusif).
	//
	// Selisih 1.08600-1.08500 pada float64 menghasilkan 0.0010000000000000009,
	// sedikit di atas 0.00100. Batas dibuat inklusif dengan kelonggaran relatif
	// kecil di SlippageDecision.OK, sehingga kasus tepi ini tetap lolos.
	ok, _, _ = SlippageCheck("EURUSD", 1.08500, 1.08600, 1.08000)
	if !ok {
		t.Fatal("drift tepat 10 pip seharusnya masih lolos")
	}

	// 11 pip: harus ditolak.
	ok, _, _ = SlippageCheck("EURUSD", 1.08500, 1.08610, 1.08000)
	if ok {
		t.Fatal("pergerakan 11 pip seharusnya ditolak")
	}

	// Kasus nyata 12 Nov: drift 6.3 pip kini LOLOS (dulu ditolak).
	ok, _, _ = SlippageCheck("EURUSD", 1.10941, 1.11004, 1.11091)
	if !ok {
		t.Fatal("drift 6.3 pip (kasus produksi 12 Nov) seharusnya lolos")
	}
}

func TestEffectiveSlippageLimitPerPair(t *testing.T) {
	// Setiap pair harus punya toleransi sendiri, bukan jatuh ke nol.
	cases := []struct {
		symbol string
		min    float64
	}{
		{"XAUUSD", 0.75},
		{"XAUEUR", 1.50},
		{"XAGUSD", 0.15},
		{"EURUSD", 0.00100},
		{"GBPUSD", 0.00100},
		{"USDJPY", 0.10},
		{"USTEC", 40},
		{"US30", 30},
		{"US500", 15},
		{"BTCUSD", 50},
	}

	for _, tc := range cases {
		// SL dibuat sangat tipis agar sumber batas murni tabel pair.
		d := EffectiveSlippageLimit(tc.symbol, 100, 99.999, LimitOptions{})
		if d.Limit < tc.min {
			t.Errorf("%s: limit = %v, ingin minimal %v", tc.symbol, d.Limit, tc.min)
		}
	}
}

func TestEffectiveSlippageLimitBasisAndSpread(t *testing.T) {
	// SL tipis: basis harus "pair".
	d := EffectiveSlippageLimit("XAUUSD", 4000, 3999.999, LimitOptions{})
	if d.Basis != "pair" {
		t.Fatalf("basis = %q, ingin %q", d.Basis, "pair")
	}

	// Spread lebar harus mengalahkan tabel pair: 0.40 x 2 = 0.80 > 0.75.
	d = EffectiveSlippageLimit("XAUUSD", 4000, 3999.999, LimitOptions{Spread: 0.40})
	if d.Basis != "spread" {
		t.Fatalf("basis = %q, ingin %q", d.Basis, "spread")
	}
	if d.Limit < 0.79 || d.Limit > 0.81 {
		t.Fatalf("limit = %v, ingin sekitar 0.80", d.Limit)
	}

	// Spread kecil tidak boleh mengecilkan batas tabel pair.
	d = EffectiveSlippageLimit("XAUUSD", 4000, 3999.999, LimitOptions{Spread: 0.05})
	if d.Limit < 0.74 {
		t.Fatalf("limit = %v, spread kecil tidak boleh menurunkan batas", d.Limit)
	}

	// SL lebar harus mengalahkan keduanya: 15% x 20 = 3.0.
	d = EffectiveSlippageLimit("XAUUSD", 4000, 3980, LimitOptions{Spread: 0.40})
	if d.Basis != "sl" {
		t.Fatalf("basis = %q, ingin %q", d.Basis, "sl")
	}
	if d.Limit < 2.99 || d.Limit > 3.01 {
		t.Fatalf("limit = %v, ingin sekitar 3.00", d.Limit)
	}
}

func TestSlippageDecisionOKTanpaBatas(t *testing.T) {
	// Simbol tak dikenal tanpa SL: limit nol, harus menolak (bukan menebak).
	d := EffectiveSlippageLimit("UNKNOWN", 100, 100, LimitOptions{})
	if d.OK() {
		t.Fatal("limit nol seharusnya menolak")
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

	if got := EffectiveTier("free", &future, nil, now); got != "basic" {
		t.Fatalf("trial aktif = %q, ingin basic", got)
	}
	// Trial memberi Basic, bukan Plus: mode auto tetap tidak boleh dipakai.
	if ModeAllowed(EffectiveTier("plus", &future, nil, now), "auto") {
		t.Fatal("trial tidak boleh mendapat mode eksekusi otomatis")
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

func TestSetDailyTimezone(t *testing.T) {
	original := WIB
	t.Cleanup(func() { WIB = original })

	// 18:00 UTC = 01:00 WIB tanggal 6, tetapi 18:00 UTC sendiri di UTC.
	utc := time.Date(2026, 11, 5, 18, 0, 0, 0, time.UTC)

	SetDailyTimezone("UTC")
	if got := DayKey(utc); got != "2026-11-05" {
		t.Fatalf("DayKey(UTC) = %q, ingin 2026-11-05", got)
	}

	SetDailyTimezone("Asia/Jakarta")
	if got := DayKey(utc); got != "2026-11-06" {
		t.Fatalf("DayKey(Asia/Jakarta) = %q, ingin 2026-11-06", got)
	}

	// Nama tidak dikenal tidak boleh mengubah zona yang sedang berlaku.
	SetDailyTimezone("Zona/TidakAda")
	if got := DayKey(utc); got != "2026-11-06" {
		t.Fatalf("DayKey setelah nama tak dikenal = %q, ingin tetap 2026-11-06", got)
	}
}
