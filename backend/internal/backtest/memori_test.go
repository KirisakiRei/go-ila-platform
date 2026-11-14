package backtest

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"ila-strategy/indicators"
)

// TestJejakMemoriDataTerburuk mengukur memori yang ditahan oleh DATA satu run,
// bukan oleh jalur sinyalnya.
//
// Alasannya: backend-api berjalan dengan mem_limit 160 MB dan worker backtest
// hidup di dalam proses itu. Yang benar-benar menguasai memori adalah deret
// candle dua timeframe (sampai lima ribu baris masing-masing) dan potongan
// candle per entry yang disimpan untuk chart. Jumlah entry tidak bisa dipaksa
// dari data sintetis tanpa membangun struktur pasar yang lengkap, tetapi
// ukuran kedua hal di atas bisa diukur tepat.
//
// Uji ini menjaga agar batasnya tidak diam-diam terlampaui saat batas candle
// dinaikkan atau jendela chart diperlebar.
func TestJejakMemoriDataTerburuk(t *testing.T) {
	if testing.Short() {
		t.Skip("pengukuran memori dilewati pada mode -short")
	}

	const (
		jumlahCandlePerTimeframe = 5000
		jumlahEntryMaksimal      = 60
		candlePerJendela         = 80
	)

	var sebelum, sesudah runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&sebelum)

	// 1. Dua deret candle masukan, ukuran maksimum yang diizinkan.
	awal := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	htf := deretCandle(awal, jumlahCandlePerTimeframe, time.Duration(24)*time.Hour)
	ltf := deretCandle(awal, jumlahCandlePerTimeframe, time.Hour)

	// 2. Potongan candle seluruh entry, dalam bentuk tersimpan (JSON), karena
	//    itulah yang benar-benar ditulis ke basis data dan ditahan saat
	//    penyimpanan berlangsung.
	potongan := make([]string, 0, jumlahEntryMaksimal)
	for i := 0; i < jumlahEntryMaksimal; i++ {
		jendela := CandleWindow{
			HTF: htf[i*2 : i*2+candlePerJendela],
			LTF: ltf[i*2 : i*2+candlePerJendela],
		}
		bahan, err := json.Marshal(jendela)
		if err != nil {
			t.Fatalf("menyusun potongan candle: %v", err)
		}
		potongan = append(potongan, string(bahan))
	}

	runtime.ReadMemStats(&sesudah)

	terpakaiMB := float64(sesudah.HeapAlloc-sebelum.HeapAlloc) / 1024 / 1024
	var ukuranPotonganKB float64
	for _, satu := range potongan {
		ukuranPotonganKB += float64(len(satu)) / 1024
	}

	t.Logf("candle %d x 2 timeframe + %d potongan chart → heap bertambah %.1f MB, potongan total %.0f KB",
		jumlahCandlePerTimeframe, len(potongan), terpakaiMB, ukuranPotonganKB)

	// Batas 60 MB: menyisakan lebih dari separuh mem_limit 160 MB untuk server
	// HTTP, koneksi basis data, dan cache candle dashboard.
	const batasMB = 60.0
	if terpakaiMB > batasMB {
		t.Fatalf("jejak memori %.1f MB melebihi batas %.0f MB; turunkan batas candle atau jendela chart",
			terpakaiMB, batasMB)
	}
}

func deretCandle(mulai time.Time, jumlah int, jeda time.Duration) []indicators.Candle {
	hasil := make([]indicators.Candle, jumlah)
	for i := range hasil {
		harga := 2300 + float64(i%97)
		hasil[i] = indicators.Candle{
			Timestamp: mulai.Add(time.Duration(i) * jeda),
			Open:      harga,
			High:      harga + 7,
			Low:       harga - 7,
			Close:     harga + 2,
			Volume:    1000,
			IsClosed:  true,
		}
	}
	return hasil
}
