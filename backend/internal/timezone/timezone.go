// Package timezone adalah rumah tunggal zona waktu platform.
//
// Dipisah dari paket lain agar bisa dipakai lapisan mana pun tanpa import
// cycle. Aturannya tetap: sesi pasar memakai NY, hari akuntansi memakai WIB.
package timezone

import (
	"strings"
	"time"
)

// WIB adalah zona hari akuntansi. Nilainya dapat diganti lewat SetDaily.
var WIB = time.FixedZone("WIB", 7*60*60)

// NY adalah zona sesi pasar.
var NY = load("America/New_York")

func load(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Tanpa database zona waktu, pakai selisih tetap EST sebagai cadangan.
		return time.FixedZone("EST", -5*60*60)
	}
	return loc
}

// SetDaily mengganti zona hari akuntansi dari konfigurasi.
// Nama tak dikenal diabaikan agar platform tetap memakai WIB.
func SetDaily(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return
	}
	WIB = loc
}

// DayKey mengubah waktu menjadi kunci hari YYYY-MM-DD pada zona akuntansi.
func DayKey(t time.Time) string {
	return t.In(WIB).Format("2006-01-02")
}

// DayStart mengembalikan 00:00 hari akuntansi yang memuat t.
func DayStart(t time.Time) time.Time {
	local := t.In(WIB)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, WIB)
}

// DayEnd mengembalikan 00:00 hari akuntansi berikutnya.
func DayEnd(t time.Time) time.Time {
	return DayStart(t).AddDate(0, 0, 1)
}
