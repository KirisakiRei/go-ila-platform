package ila

import (
	"testing"
	"time"

	"ila-strategy/indicators"
)

// TestFVGTermudaGagalTidakMundurKeFVGLebihTua mengunci aturan pemilihan FVG acuan:
//
//  1. FVG aktif TERMUDA dipilih lebih dulu.
//  2. Bila FVG termuda masih dijangkau harga, dialah acuannya.
//  3. Bila FVG termuda TIDAK dijangkau harga, acuan diperiksa pada FVG lebih tua
//     dari yang termuda ke belakang, berhenti pada yang sweep-nya sah.
//  4. Bila tak satu pun menghasilkan sweep sah, tidak ada acuan sama sekali.
//
// Butir 4 adalah yang dikunci di sini. Versi lama menyaring dulu (umur + jarak)
// lalu memilih yang termuda di antara yang lolos, sehingga perilakunya berbeda
// saat semua FVG gagal syarat: ia tetap mengembalikan acuan arah dari FVG
// tertua yang tersisa, dan evaluator bisa menelusuri candle sangat jauh ke
// belakang mencari sweep yang sudah usang.
func TestFVGTermudaGagalTidakMundurKeFVGLebihTua(t *testing.T) {
	eval := NewEvaluator()
	awal := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	jam := func(i int) time.Time { return awal.Add(time.Duration(i) * 4 * time.Hour) }

	// FVG BEARISH 3887.64-3940.00 (Candle 3 = indeks 2), lalu harga TURUN dan
	// bertahan jauh di bawah zona sehingga zona itu tidak pernah disapu.
	// Candle-candle berikutnya membentuk FVG bearish baru yang juga tidak
	// pernah disapu karena harga terus turun.
	candles := []indicators.Candle{
		{Timestamp: jam(0), Open: 3965.88, High: 3972.51, Low: 3940.00, Close: 3954.01, IsClosed: true},
		{Timestamp: jam(1), Open: 3954.01, High: 3955.51, Low: 3887.00, Close: 3890.00, IsClosed: true},
		{Timestamp: jam(2), Open: 3890.00, High: 3887.64, Low: 3852.64, Close: 3882.88, IsClosed: true},
		{Timestamp: jam(3), Open: 3820.00, High: 3830.00, Low: 3790.00, Close: 3800.00, IsClosed: true},
		{Timestamp: jam(4), Open: 3800.00, High: 3815.00, Low: 3770.00, Close: 3780.00, IsClosed: true},
		{Timestamp: jam(5), Open: 3775.00, High: 3810.00, Low: 3750.00, Close: 3800.00, IsClosed: true},
		{Timestamp: jam(6), Open: 3800.00, High: 3805.00, Low: 3740.00, Close: 3745.00, IsClosed: true},
		{Timestamp: jam(7), Open: 3745.00, High: 3760.00, Low: 3700.00, Close: 3705.00, IsClosed: true},
	}

	aktif := []FVG{}
	for _, f := range eval.FindFVGs(candles) {
		if !f.Mitigated {
			aktif = append(aktif, f)
		}
	}
	if len(aktif) == 0 {
		t.Fatalf("prasyarat setup: harus ada FVG aktif")
	}

	// Prasyarat: SEMUA FVG aktif sudah tidak dijangkau harga. Itu keadaan yang
	// membedakan versi lama dan versi baru.
	for _, f := range aktif {
		if fvgMasihDijangkauHarga(f, candles) {
			t.Fatalf("prasyarat setup: FVG c3=%d masih dijangkau harga", f.Candle3Index)
		}
		if len(candles)-1-f.Candle3Index > umurMaksimalFVG {
			t.Fatalf("prasyarat setup: FVG c3=%d sudah lewat batas umur", f.Candle3Index)
		}
	}

	// Inti tes: tidak ada satu pun FVG yang disapu, jadi tidak boleh ada sweep
	// maupun acuan arah yang dilaporkan.
	if sweep := eval.DetectHTFIlaSweep(candles); sweep != nil {
		t.Fatalf("masih ada acuan %s dari FVG %s-%s; seharusnya nil karena tidak ada "+
			"FVG yang dijangkau harga maupun disapu",
			sweep.Direction, formatILAVal(sweep.FVG.Bottom), formatILAVal(sweep.FVG.Top))
	}
}

// TestFVGTermudaJadiAcuanWalauAdaFVGLebihTua memastikan acuan tetap memihak FVG
// termuda saat keduanya sama-sama memenuhi syarat.
func TestFVGTermudaJadiAcuanWalauAdaFVGLebihTua(t *testing.T) {
	eval := NewEvaluator()
	awal := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	jam := func(i int) time.Time { return awal.Add(time.Duration(i) * 4 * time.Hour) }

	// FVG BEARISH tua 3887.64-3940.00 (Candle 3 = indeks 2), lalu FVG BEARISH
	// muda 3860.00-3875.20 (Candle 3 = indeks 5). Setelah itu harga berhenti
	// di bawah zona muda sehingga zona muda TETAP di dalam jangkauan harga
	// (Bottom = 3860.00, harga masih di sekitarnya) tetapi belum disapu.
	candles := []indicators.Candle{
		{Timestamp: jam(0), Open: 3965.88, High: 3972.51, Low: 3940.00, Close: 3954.01, IsClosed: true},
		{Timestamp: jam(1), Open: 3954.01, High: 3955.51, Low: 3887.00, Close: 3890.00, IsClosed: true},
		{Timestamp: jam(2), Open: 3890.00, High: 3887.64, Low: 3852.64, Close: 3882.88, IsClosed: true},
		// Candle 1 FVG muda: Low 3875.20 (Top zona muda).
		{Timestamp: jam(3), Open: 3882.88, High: 3878.00, Low: 3862.00, Close: 3870.00, IsClosed: true},
		// Displacement turun: High 3860.00 (Bottom zona muda).
		{Timestamp: jam(4), Open: 3870.00, High: 3860.00, Low: 3840.00, Close: 3845.00, IsClosed: true},
		{Timestamp: jam(5), Open: 3845.00, High: 3848.00, Low: 3820.00, Close: 3825.00, IsClosed: true},
		// Harga kembali naik menyentuh zona muda tanpa menembus Bottom-nya,
		// jadi zona muda hidup dan dijangkau tetapi belum disapu.
		{Timestamp: jam(6), Open: 3825.00, High: 3858.00, Low: 3822.00, Close: 3850.00, IsClosed: true},
	}

	daftarFVG := eval.FindFVGs(candles)
	termuda := FVG{}
	for _, f := range daftarFVG {
		if f.Candle3Index > termuda.Candle3Index {
			termuda = f
		}
	}
	if termuda.Candle3Index != 5 {
		t.Fatalf("prasyarat setup: FVG termuda harus Candle 3 = indeks 5, dapat %d", termuda.Candle3Index)
	}

	sweep := eval.DetectHTFIlaSweep(candles)
	if sweep == nil {
		t.Fatalf("harus ada acuan arah walau belum ada sweep")
	}
	if sweep.FVG == nil || sweep.FVG.Candle3Index != 5 {
		t.Fatalf("acuan harus FVG termuda (c3=5), dapat %+v", sweep.FVG)
	}
}
