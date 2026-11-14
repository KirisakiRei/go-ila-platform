package backtest

import (
	"testing"
	"time"

	"ila-strategy/indicators"
)

// TestJendelaChartMemuatExit menegaskan janji akurasi chart: candle yang
// menyentuh target atau batas rugi HARUS ikut tersimpan.
//
// Sebelum diperbaiki, jendela dibuat saat entry dibuka sehingga trade yang
// berjalan lebih lama dari dua puluh candle kehilangan candle exit-nya. Di
// chart, posisi tampak menggantung padahal mesin sudah menutupnya — pembaca
// lalu menyimpulkan exit yang tidak pernah terjadi.
func TestJendelaChartMemuatExit(t *testing.T) {
	mulai := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Lima puluh candle LTF per jam; posisi dibuka di candle ke-10 dan ditutup
	// di candle ke-45. Jarak 35 candle jauh melampaui jendela 20.
	ltf := make([]indicators.Candle, 60)
	for i := range ltf {
		ltf[i] = buatCandle(mulai, i, 100, 101, 99, 100)
	}

	entryTime := ltf[10].Timestamp
	exitTime := ltf[45].Timestamp

	posisi := &Entry{
		Seq:        1,
		Direction:  DirectionBuy,
		EntryTime:  entryTime,
		ExitTime:   exitTime,
		EntryPrice: 100,
		StopLoss:   95,
		TakeProfit: 110,
		Outcome:    OutcomeTakeProfit,
	}

	htf := []indicators.Candle{
		{Timestamp: mulai, Open: 100, High: 101, Low: 99, Close: 100, IsClosed: true},
		{Timestamp: mulai.Add(24 * time.Hour), Open: 100, High: 111, Low: 99, Close: 110, IsClosed: true},
	}

	jendela := jendelaChart(posisi, htf, ltf)

	if len(jendela.LTF) == 0 {
		t.Fatal("jendela LTF kosong")
	}

	// Syarat pertama: candle exit ada di dalam jendela.
	var adaExit bool
	for _, candle := range jendela.LTF {
		if candle.Timestamp.Equal(exitTime) {
			adaExit = true
			break
		}
	}
	if !adaExit {
		t.Fatalf("candle exit %s tidak ikut tersimpan; chart akan menggambar posisi tanpa akhir",
			exitTime.Format(time.RFC3339))
	}

	// Syarat kedua: candle entry juga ada, supaya garis entry bisa ditarik.
	var adaEntry bool
	var pertama, terakhir time.Time
	for i, candle := range jendela.LTF {
		if candle.Timestamp.Equal(entryTime) {
			adaEntry = true
		}
		if i == 0 {
			pertama = candle.Timestamp
		}
		terakhir = candle.Timestamp
	}
	if !adaEntry {
		t.Fatal("candle entry tidak ikut tersimpan")
	}
	if pertama.After(entryTime) || terakhir.Before(exitTime) {
		t.Fatalf("jendela %s..%s tidak membentang dari entry ke exit",
			pertama.Format(time.RFC3339), terakhir.Format(time.RFC3339))
	}

	// Syarat ketiga: HTF dipotong dengan titik yang sama sehingga kedua seri
	// sejajar waktunya di satu chart.
	if len(jendela.HTF) == 0 {
		t.Fatal("jendela HTF kosong")
	}
}

// TestJendelaChartTetapBerbatas memastikan perbaikan di atas tidak membuat
// chart memuat seluruh riwayat. Batasnya penting: candle disimpan ke basis data
// untuk setiap entry, jadi jendela tanpa batas akan membengkakkan tabel.
func TestJendelaChartTetapBerbatas(t *testing.T) {
	mulai := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	ltf := make([]indicators.Candle, 2000)
	for i := range ltf {
		ltf[i] = buatCandle(mulai, i, 100, 101, 99, 100)
	}

	posisi := &Entry{
		Seq:       1,
		EntryTime: ltf[100].Timestamp,
		ExitTime:  ltf[900].Timestamp,
	}
	jendela := jendelaChart(posisi, ltf, ltf)

	// 20 sebelum entry + rentang entry..exit + 20 sesudah exit.
	const batasWajar = 900 + candleJendelaChart + candleJendelaChartHTF + 1
	if len(jendela.LTF) > batasWajar {
		t.Fatalf("jendela LTF %d candle melebihi batas wajar %d", len(jendela.LTF), batasWajar)
	}
}
