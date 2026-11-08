package main

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// slogLevel menerjemahkan LOG_LEVEL menjadi level slog.
func slogLevel(v string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// envInt membaca variabel lingkungan berisi bilangan bulat positif.
func envInt(key string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
