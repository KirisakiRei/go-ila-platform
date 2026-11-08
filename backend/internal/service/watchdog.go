package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ila/backend/internal/alert"
	"ila/backend/internal/dto"
)

// WatchInterval adalah jeda pemeriksaan kondisi operasional.
// Cukup rapat untuk kabar cepat, cukup longgar untuk VPS 1 GB.
const WatchInterval = time.Minute

// Watchdog memeriksa kondisi platform lalu mengirim peringatan admin.
//
// Yang diperiksa hanya hal yang bisa dibaca tanpa beban tambahan: laporan
// Runner terakhir, kesehatan akun broker, dan sisa disk (plans/18 §2).
type Watchdog struct {
	ops    AdminSystemOps
	notify *alert.Notifier
	logger *slog.Logger
	now    func() time.Time
	// diskFree membaca persentase pemakaian disk; nil berarti tidak diperiksa.
	diskFree func() (usedPercent float64, ok bool)
}

// NewWatchdog merangkai pemeriksa kondisi platform.
func NewWatchdog(ops AdminSystemOps, notifier *alert.Notifier, logger *slog.Logger) *Watchdog {
	return &Watchdog{ops: ops, notify: notifier, logger: logger, now: func() time.Time { return time.Now().UTC() }}
}

// WithDiskProbe memasang pembaca pemakaian disk (dipakai saat uji).
func (w *Watchdog) WithDiskProbe(probe func() (float64, bool)) *Watchdog {
	w.diskFree = probe
	return w
}

// Run memeriksa kondisi tiap interval sampai konteks selesai.
func (w *Watchdog) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = WatchInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Pemeriksaan pertama dilakukan segera agar kondisi awal tidak menunggu.
	w.Check(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Check(ctx)
		}
	}
}

// Check menjalankan satu putaran pemeriksaan.
func (w *Watchdog) Check(ctx context.Context) {
	overview, err := w.ops.Overview(ctx)
	if err != nil {
		w.logger.Warn("pemeriksaan kondisi gagal", "error", err.Error())
		return
	}

	w.checkRunner(ctx, overview)
	w.checkBrokers(ctx, overview)
	w.checkDisk(ctx)
}

// checkRunner melaporkan Runner yang tidak mengirim laporan sama sekali.
func (w *Watchdog) checkRunner(ctx context.Context, overview *dto.SystemOverview) {
	if overview.Runner.Connected {
		w.notify.Recovered(ctx, alert.EngineDisconnected, "Runner kembali mengirim laporan.")
		return
	}
	last := "belum pernah"
	if overview.Runner.LastSeen != nil {
		last = overview.Runner.LastSeen.Format(time.RFC3339)
	}
	w.notify.Fire(ctx, alert.EngineDisconnected,
		fmt.Sprintf("Runner tidak mengirim laporan sejak %s. Posisi berjalan tetap ditangani broker.", last))
}

// checkBrokers melaporkan bila banyak akun broker tidak bisa dihubungi.
func (w *Watchdog) checkBrokers(ctx context.Context, overview *dto.SystemOverview) {
	health := overview.BrokerHealth
	if health.ErrorCount >= 3 {
		w.notify.Fire(ctx, alert.BrokerErrors,
			fmt.Sprintf("%d dari %d akun broker sedang gagal tersambung.", health.ErrorCount, health.Total))
		return
	}
	if health.ErrorCount == 0 {
		w.notify.Recovered(ctx, alert.BrokerErrors, "Seluruh akun broker tersambung kembali.")
	}
}

// checkDisk melaporkan pemakaian disk di atas 85 persen.
func (w *Watchdog) checkDisk(ctx context.Context) {
	if w.diskFree == nil {
		return
	}
	used, ok := w.diskFree()
	if !ok {
		return
	}
	if used >= 85 {
		w.notify.Fire(ctx, alert.DiskAlmostFull, fmt.Sprintf("Ruang disk terpakai %.1f%%.", used))
		return
	}
	w.notify.Recovered(ctx, alert.DiskAlmostFull, fmt.Sprintf("Ruang disk kembali normal (%.1f%%).", used))
}
