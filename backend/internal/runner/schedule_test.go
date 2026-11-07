package runner

import (
	"testing"
	"time"
)

func TestExpectedMarketOpen(t *testing.T) {
	// Waktu acuan dalam NY: 2026-11-05 adalah Kamis.
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"Kamis siang", time.Date(2026, 11, 5, 12, 0, 0, 0, NY), true},
		{"Jumat sebelum 17:00", time.Date(2026, 11, 6, 16, 59, 0, 0, NY), true},
		{"Jumat 17:00 tutup", time.Date(2026, 11, 6, 17, 0, 0, 0, NY), false},
		{"Sabtu penuh tutup", time.Date(2026, 11, 7, 12, 0, 0, 0, NY), false},
		{"Minggu sebelum 17:00", time.Date(2026, 11, 8, 16, 0, 0, 0, NY), false},
		{"Minggu 17:00 buka", time.Date(2026, 11, 8, 17, 0, 0, 0, NY), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := expectedOpen(tc.at); got != tc.want {
				t.Fatalf("expectedOpen = %v, ingin %v", got, tc.want)
			}
		})
	}
}

func TestCheckMarketSchedule(t *testing.T) {
	saturday := time.Date(2026, 11, 7, 12, 0, 0, 0, NY)
	if got := CheckMarketSchedule(true, saturday); got != AnomalyWeekendOpen {
		t.Fatalf("buka di Sabtu = %q, ingin %q", got, AnomalyWeekendOpen)
	}

	thursday := time.Date(2026, 11, 5, 12, 0, 0, 0, NY)
	if got := CheckMarketSchedule(false, thursday); got != AnomalyUnexpectedClosed {
		t.Fatalf("tutup di Kamis = %q, ingin %q", got, AnomalyUnexpectedClosed)
	}

	if got := CheckMarketSchedule(true, thursday); got != AnomalyNone {
		t.Fatalf("status normal = %q, ingin kosong", got)
	}
}

func TestScheduleWatchThrottle(t *testing.T) {
	watch := NewScheduleWatch(10*time.Minute, 10*time.Minute)
	at := time.Date(2026, 11, 7, 12, 0, 0, 0, NY) // Sabtu, pasar seharusnya tutup

	first := watch.Observe(true, at)
	if len(first) != 1 {
		t.Fatalf("peringatan pertama = %v, ingin satu", first)
	}

	// Masih dalam jendela; tidak boleh membanjiri kanal.
	if again := watch.Observe(true, at.Add(2*time.Minute)); len(again) != 0 {
		t.Fatalf("peringatan berulang = %v, ingin kosong", again)
	}

	// Lewat jendela, peringatan boleh muncul lagi.
	if later := watch.Observe(true, at.Add(11*time.Minute)); len(later) != 1 {
		t.Fatalf("peringatan setelah jeda = %v, ingin satu", later)
	}
}

func TestScheduleWatchStale(t *testing.T) {
	watch := NewScheduleWatch(time.Minute, 10*time.Minute)
	at := time.Date(2026, 11, 5, 12, 0, 0, 0, NY)

	if !watch.Stale(at) {
		t.Fatal("tanpa status sama sekali harus dianggap usang")
	}

	watch.Observe(true, at)
	if watch.Stale(at.Add(5 * time.Minute)) {
		t.Fatal("status baru 5 menit belum usang")
	}
	if !watch.Stale(at.Add(11 * time.Minute)) {
		t.Fatal("status 11 menit harus usang")
	}
}
