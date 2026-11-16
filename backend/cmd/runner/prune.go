package main

import (
	"context"
	"log/slog"
	"time"

	"ila/backend/internal/repository"
)

// rentangSimpanJejak adalah masa simpan catatan pesan bertombol.
//
// Telegram menolak mengedit pesan yang lebih tua dari 48 jam, jadi catatan
// tiga hari sudah pasti tidak terpakai lagi.
const rentangSimpanJejak = 72 * time.Hour

// jedaBersihJejak membatasi pembersihan ke sekali sehari; tabelnya kecil,
// tidak perlu diperiksa lebih sering.
const jedaBersihJejak = 24 * time.Hour

// pruneLoop membuang jejak pesan lama secara berkala.
func pruneLoop(ctx context.Context, jejak repository.TelegramMessageRepository, logger *slog.Logger) {
	if jejak == nil {
		return
	}
	bersihkan := func() {
		batas := time.Now().UTC().Add(-rentangSimpanJejak)
		jumlah, err := jejak.HapusLama(ctx, batas)
		if err != nil {
			logger.Warn("gagal membersihkan jejak pesan telegram", "error", err.Error())
			return
		}
		if jumlah > 0 {
			logger.Info("jejak pesan telegram dibersihkan", "jumlah", jumlah)
		}
	}
	bersihkan()

	ticker := time.NewTicker(jedaBersihJejak)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bersihkan()
		}
	}
}
