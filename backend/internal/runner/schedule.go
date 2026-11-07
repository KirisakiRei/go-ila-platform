package runner

import (
	"time"
)

// NY adalah zona sesi pasar. Jam trading selalu mengikuti New York, sedangkan
// hari akuntansi mengikuti WIB (plans/22 §1).
var NY = mustLoadLocation("America/New_York")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Fallback konservatif; hanya terjadi bila basis data zona tidak ada.
		return time.UTC
	}
	return loc
}

// MarketScheduleAnomaly menandai status pasar yang janggal pada waktu tertentu.
type MarketScheduleAnomaly string

const (
	// AnomalyNone berarti status pasar sesuai perkiraan jadwal.
	AnomalyNone MarketScheduleAnomaly = ""
	// AnomalyWeekendOpen berarti engine melaporkan pasar buka di akhir pekan.
	AnomalyWeekendOpen MarketScheduleAnomaly = "Pasar dilaporkan buka pada akhir pekan"
	// AnomalyUnexpectedClosed berarti pasar tutup padahal seharusnya buka.
	AnomalyUnexpectedClosed MarketScheduleAnomaly = "Pasar tutup di luar jadwal normal"
	// AnomalyUnexpectedOpen berarti pasar buka padahal seharusnya tutup.
	AnomalyUnexpectedOpen MarketScheduleAnomaly = "Pasar buka di luar jadwal normal"
)

// expectedOpen memperkirakan apakah pasar seharusnya buka pada waktu t.
// Aturan: tutup penuh Sabtu dan Minggu; Jumat tutup sejak 17:00 NY dan
// Minggu baru buka 17:00 NY (plans/22 §3).
func expectedOpen(t time.Time) bool {
	ny := t.In(NY)
	switch ny.Weekday() {
	case time.Saturday:
		return false
	case time.Sunday:
		return ny.Hour() >= 17
	case time.Friday:
		return ny.Hour() < 17
	default:
		return true
	}
}

// CheckMarketSchedule membandingkan status dari engine dengan perkiraan jadwal.
// Dipakai untuk mendeteksi engine yang salah zona waktu.
func CheckMarketSchedule(reportedOpen bool, at time.Time) MarketScheduleAnomaly {
	expected := expectedOpen(at)
	if reportedOpen == expected {
		return AnomalyNone
	}
	if reportedOpen && !expected {
		if at.In(NY).Weekday() == time.Saturday || at.In(NY).Weekday() == time.Sunday {
			return AnomalyWeekendOpen
		}
		return AnomalyUnexpectedOpen
	}
	return AnomalyUnexpectedClosed
}

// ScheduleClock membatasi pemeriksaan agar tidak berjalan setiap sinyal.
// SignalAlertterlaluSering menyaring peringatan berulang ke interval tetap.
type ScheduleWatch struct {
	lastReported time.Time
	lastState    engineMarketState
	interval     time.Duration
	staleAfter   time.Duration
}

// engineMarketState adalah bentuk ringkas agar pemantau tidak bergantung pada
// protobuf di bagian aturan.
type engineMarketState struct {
	known  bool
	isOpen bool
	seenAt time.Time
}

// NewScheduleWatch membuat pemantau jadwal dengan interval pemeriksaan.
func NewScheduleWatch(interval, staleAfter time.Duration) *ScheduleWatch {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	if staleAfter <= 0 {
		staleAfter = 10 * time.Minute
	}
	return &ScheduleWatch{interval: interval, staleAfter: staleAfter}
}

// Observe mencatat status pasar dari engine dan mengembalikan masalah yang
// perlu dilaporkan (kosong berarti tidak ada).
func (w *ScheduleWatch) Observe(open bool, at time.Time) []MarketScheduleAnomaly {
	w.lastState = engineMarketState{known: true, isOpen: open, seenAt: at}
	if at.Sub(w.lastReported) < w.interval {
		return nil
	}

	anomaly := CheckMarketSchedule(open, at)
	if anomaly == AnomalyNone {
		return nil
	}
	w.lastReported = at
	return []MarketScheduleAnomaly{anomaly}
}

// Stale menandai status pasar yang tidak diperbarui melewati batas.
func (w *ScheduleWatch) Stale(at time.Time) bool {
	if !w.lastState.known {
		return true
	}
	return at.Sub(w.lastState.seenAt) > w.staleAfter
}
