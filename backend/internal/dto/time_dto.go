package dto

import (
	"time"

	"ila/backend/internal/runner"
)

// TimePair menyajikan satu waktu dalam bentuk yang bisa langsung ditampilkan.
// Frontend tidak menghitung zona waktu sendiri karena browser pengguna bisa
// berzona apa pun (plans/22 §6.3).
type TimePair struct {
	// UTC adalah nilai mentah untuk keperluan urutan dan integrasi.
	UTC time.Time `json:"utc"`
	// WIB adalah jam dinding pengguna: "20:14 · 05 Nov".
	WIB string `json:"wib"`
	// NY adalah jam pasar: "08:14 · 05 Nov".
	NY string `json:"ny"`
}

const pairLayout = "15:04 · 02 Jan"

// NewTimePair memetakan satu waktu ke dua zona.
func NewTimePair(t time.Time) TimePair {
	return TimePair{
		UTC: t.UTC(),
		WIB: t.In(runner.WIB).Format(pairLayout),
		NY:  t.In(runner.NY).Format(pairLayout),
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

// SessionInfo menjelaskan jam pasar dalam dua bahasa.
type SessionInfo struct {
	Open      TimePair `json:"open"`
	Close     TimePair `json:"close"`
	NextEvent string   `json:"next_event,omitempty"`
	// IsOpen berasal dari Engine, bukan tebakan backend.
	IsOpen bool `json:"is_open"`
}
