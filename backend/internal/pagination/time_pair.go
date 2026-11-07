package pagination

import (
	"time"

	"ila/backend/internal/timezone"
)

// TimePair menyajikan satu waktu dalam dua zona: jam dinding pengguna (WIB)
// dan jam pasar (NY). Frontend tidak menghitung zona sendiri (plans/22 §6.3).
type TimePair struct {
	UTC time.Time `json:"utc"`
	WIB string    `json:"wib"`
	NY  string    `json:"ny"`
}

const pairLayout = "15:04 · 02 Jan"

// NewTimePair memetakan satu waktu ke dua zona.
func NewTimePair(t time.Time) TimePair {
	return TimePair{
		UTC: t.UTC(),
		WIB: t.In(timezone.WIB).Format(pairLayout),
		NY:  t.In(timezone.NY).Format(pairLayout),
	}
}

// NewTimePairPtr mengembalikan nil bila waktunya kosong.
func NewTimePairPtr(t *time.Time) *TimePair {
	if t == nil {
		return nil
	}
	pair := NewTimePair(*t)
	return &pair
}
