package dto_test

import (
	"strings"
	"testing"
	"time"

	"ila/backend/internal/dto"
)

// Satu waktu harus tersaji dalam WIB dan NY dengan benar. Frontend tidak
// menghitung zona sendiri, jadi kebenarannya harus diuji di sini (plans/22).
func TestTimePairDuaZona(t *testing.T) {
	// 13:14 UTC = 20:14 WIB = 08:14 NY (saat DST New York aktif).
	at := time.Date(2026, 11, 5, 13, 14, 0, 0, time.UTC)

	pair := dto.NewTimePair(at)

	if !strings.HasPrefix(pair.WIB, "20:14") {
		t.Fatalf("WIB = %q, ingin 20:14", pair.WIB)
	}
	if !strings.HasPrefix(pair.NY, "08:14") {
		t.Fatalf("NY = %q, ingin 08:14", pair.NY)
	}
	if !pair.UTC.Equal(at) {
		t.Fatalf("UTC berubah menjadi %v", pair.UTC)
	}
}

// Saat DST tidak aktif, NY menjadi UTC-5 sehingga selisih WIB-NY jadi 12 jam.
func TestTimePairMenghormatiDST(t *testing.T) {
	winter := time.Date(2026, 1, 15, 13, 0, 0, 0, time.UTC)
	pair := dto.NewTimePair(winter)

	if !strings.HasPrefix(pair.NY, "08:00") {
		t.Fatalf("NY musim dingin = %q, ingin 08:00", pair.NY)
	}
	if !strings.HasPrefix(pair.WIB, "20:00") {
		t.Fatalf("WIB musim dingin = %q, ingin 20:00", pair.WIB)
	}
}

// Waktu kosong tidak boleh dipaksakan menjadi nilai palsu.
func TestTimePairPtrNihil(t *testing.T) {
	if dto.NewTimePairPtr(nil) != nil {
		t.Fatal("waktu kosong harus menjadi nil")
	}
}

// Batas hari harus jatuh pada 00:00 WIB, bukan 00:00 NY.
func TestBatasHariWIBBerbedaDariNY(t *testing.T) {
	// 17:30 UTC = 00:30 WIB tanggal 6, tetapi masih tanggal 5 di NY.
	at := time.Date(2026, 11, 5, 17, 30, 0, 0, time.UTC)
	pair := dto.NewTimePair(at)

	if !strings.HasPrefix(pair.WIB, "00:30") {
		t.Fatalf("WIB = %q, ingin 00:30", pair.WIB)
	}
	if !strings.HasPrefix(pair.NY, "12:30") {
		t.Fatalf("NY = %q, ingin 12:30", pair.NY)
	}
	if !strings.Contains(pair.WIB, "06 Nov") {
		t.Fatalf("WIB harus tanggal 6, dapat %q", pair.WIB)
	}
	if !strings.Contains(pair.NY, "05 Nov") {
		t.Fatalf("NY harus tanggal 5, dapat %q", pair.NY)
	}
}
