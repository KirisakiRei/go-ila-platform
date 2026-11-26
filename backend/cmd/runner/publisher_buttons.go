package main

import (
	"context"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/telegram"
)

// statusTerminal menandai status yang membuat setup tidak bisa dieksekusi lagi.
//
// Hanya status di sini yang memicu pelepasan tombol: status menengah (radar,
// mencari CISD) masih bisa berubah menjadi entry.
func statusTerminal(status string) bool {
	switch status {
	case model.SignalStatusConsumed, model.SignalStatusCancelled,
		model.SignalStatusExpired, model.SignalStatusSuperseded,
		model.SignalStatusRejected, model.SignalStatusSkipped:
		return true
	}
	return false
}

// lepasTombol menghapus tombol pada semua pesan setup milik satu sinyal.
//
// Kegagalan edit bukan kegagalan trading: pesan yang terlalu tua untuk diedit
// hanya dicatat, lalu jejaknya dibuang agar tidak dicoba terus-menerus.
func (p *runnerPublisher) lepasTombol(ctx context.Context, userID, signalID, status, reason string) {
	if p.jejak == nil || p.tg == nil {
		return
	}
	rows, err := p.jejak.BelumDibersihkan(ctx, signalID)
	if err != nil {
		p.logger.Warn("gagal membaca jejak pesan bertombol", "signal_id", signalID, "error", err.Error())
		return
	}
	symbol, htf, ltf := "", "", ""
	if len(rows) > 0 {
		signal, err := p.store.FindSignal(ctx, signalID)
		if err != nil {
			p.logger.Warn("konteks setup pesan tidak ditemukan", "signal_id", signalID, "error", err.Error())
		} else if signal != nil {
			symbol, htf, ltf = signal.Symbol, signal.Htf, signal.Ltf
		}
	}
	for _, row := range rows {
		if row.UserID != userID {
			continue
		}
		pesan := telegram.FormatSudahTidakBerlaku(symbol, htf, ltf, alasanTidakBerlaku(status, reason))
		if err := p.tg.EditMessageText(ctx, row.ChatID, row.MessageID, pesan.Text, nil); err != nil {
			p.logger.Info("pesan bertombol tidak bisa diperbarui",
				"signal_id", signalID, "message_id", row.MessageID, "error", err.Error())
		}
		if err := p.jejak.TandaiDibersihkan(ctx, row.ID, time.Now().UTC()); err != nil {
			p.logger.Warn("gagal menandai pesan dibersihkan", "signal_id", signalID, "error", err.Error())
		}
	}
}

// alasanTidakBerlaku menerjemahkan status akhir menjadi kalimat singkat.
func alasanTidakBerlaku(status, reason string) string {
	switch status {
	case model.SignalStatusConsumed:
		return "Setup sudah dieksekusi."
	case model.SignalStatusSkipped:
		return "Setup dilewati."
	case model.SignalStatusRejected:
		if reason != "" {
			return reason
		}
		return "Setup tidak lolos pemeriksaan."
	case model.SignalStatusExpired:
		return "Waktu setup sudah habis."
	case model.SignalStatusSuperseded:
		return "Sudah digantikan setup yang lebih baru."
	case model.SignalStatusCancelled:
		return "Struktur pasar berubah, setup dibatalkan."
	}
	return ""
}
