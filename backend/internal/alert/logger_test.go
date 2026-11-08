package alert

import (
	"io"
	"log/slog"
)

// testLogger membuang keluaran agar hasil tes tetap ringkas.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}
