package runner

import (
	"os"
	"strings"
	"sync/atomic"
)

// Settings adalah pengaturan platform yang dibaca dari environment.
// Nilainya bisa berubah tanpa deploy ulang agar ketersediaan strategi baru
// tidak menuntut rilis per tahap (plans/15 §6.1).
type Settings struct {
	// ILA3Availability: off | internal | beta | general.
	ila3 atomic.Value
}

// Nilai ketersediaan strategi ILA 3.0.
const (
	ILA3Off      = "off"
	ILA3Internal = "internal"
	ILA3Beta     = "beta"
	ILA3General  = "general"
)

// LoadSettings membaca pengaturan dari environment.
func LoadSettings() *Settings {
	s := &Settings{}
	s.ila3.Store(ILA3Off)

	switch strings.ToLower(strings.TrimSpace(os.Getenv("ILA3_AVAILABILITY"))) {
	case ILA3Internal:
		s.ila3.Store(ILA3Internal)
	case ILA3Beta:
		s.ila3.Store(ILA3Beta)
	case ILA3General:
		s.ila3.Store(ILA3General)
	}
	return s
}

// ILA3Avail mengembalikan tahap ketersediaan ILA 3.0.
func (s *Settings) ILA3Avail() string {
	if s == nil {
		return ILA3Off
	}
	v, _ := s.ila3.Load().(string)
	if v == "" {
		return ILA3Off
	}
	return v
}

// ILA3Running menandai apakah engine ILA 3.0 boleh dijalankan sama sekali.
func (s *Settings) ILA3Running() bool {
	return s.ILA3Avail() != ILA3Off
}

// StrategyStatus mengembalikan status strategi yang terlihat oleh user.
func (s *Settings) StrategyStatus(strategy string) string {
	if strategy != "ila-3.0" {
		return "active"
	}
	switch s.ILA3Avail() {
	case ILA3General:
		return "active"
	case ILA3Beta:
		return "beta"
	default:
		return "coming_soon"
	}
}

// StrategySelectable menandai strategi yang boleh dipilih user.
func (s *Settings) StrategySelectable(strategy string) bool {
	if strategy == "ila-3.0" {
		return s.ILA3Avail() == ILA3General || s.ILA3Avail() == ILA3Beta
	}
	return true
}

// NewSettings membuat pengaturan dengan tahap ILA 3.0 tertentu.
// Dipakai saat pengujian dan saat konfigurasi diberikan langsung oleh pemanggil.
func NewSettings(ila3 string) *Settings {
	s := &Settings{}
	s.ila3.Store(ILA3Off)
	switch ila3 {
	case ILA3Internal, ILA3Beta, ILA3General:
		s.ila3.Store(ila3)
	}
	return s
}
