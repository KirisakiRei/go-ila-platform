//go:build !linux

package main

// diskUsage tidak tersedia di Windows atau macOS saat pengembangan;
// Watchdog melewati pemeriksaan disk bila pembacaan tidak ada.
func diskUsage() (float64, bool) {
	return 0, false
}
