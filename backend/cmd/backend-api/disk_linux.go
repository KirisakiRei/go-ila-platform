//go:build linux

package main

import (
	"syscall"
)

// diskUsage membaca persentase pemakaian disk tempat proses berjalan.
// Nilai kedua false berarti pembacaan tidak tersedia di sistem ini.
func diskUsage() (float64, bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, false
	}
	if stat.Blocks == 0 {
		return 0, false
	}
	total := float64(stat.Blocks)
	used := float64(stat.Blocks - stat.Bavail)
	return used / total * 100, true
}
