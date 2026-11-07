package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/repository"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/runner/notify"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/telegram"
)

// runnerPublisher menerjemahkan perubahan status Runner menjadi notifikasi.
//
// Dipisah dari inti Runner agar aturan trading bisa diuji tanpa kanal
// pesan, dan agar kegagalan kanal tidak pernah mengubah keputusan trading.
type runnerPublisher struct {
	store    repository.RunnerRepository
	notifier *notify.Notifier
	baseURL  string
	logger   *slog.Logger
}

// PublishSignal mengirim notifikasi untuk satu perubahan status sinyal user.
func (p *runnerPublisher) PublishSignal(ctx context.Context, userID, signalID, status, reason string) {
	signal, err := p.store.FindSignal(ctx, signalID)
	if err != nil {
		p.logger.Warn("notifikasi dilewati: sinyal tidak ditemukan", "signal_id", signalID)
		return
	}

	// Notifikasi hanya untuk status yang punya arti bagi user.
	switch status {
	case model.SignalStatusReadyToEntry, model.SignalStatusRejected,
		model.SignalStatusCancelled, model.SignalStatusExpired,
		model.SignalStatusSuperseded:
	default:
		return
	}

	messageKey := "sig:" + signalID + ":" + userID + ":" + status

	p.notifier.Enqueue(userID, messageKey, func(chatID string) telegram.Message {
		_ = chatID

		switch status {
		case model.SignalStatusReadyToEntry:
			link := p.approveLink(ctx, userID, signalID)
			return telegram.FormatSetupMessage(
				signal.Symbol, signal.Direction, signal.StrategyVersion,
				signal.EntryPrice, signal.StopLoss, signal.TakeProfit,
				signal.RiskReward, defaultRiskPercent, 0, link,
			)

		case model.SignalStatusRejected:
			return telegram.FormatHeldMessage(
				signal.Symbol, signal.Direction, signal.StrategyVersion,
				signal.RiskReward, reason,
			)

		default:
			text := reason
			if text == "" {
				text = "Setup tidak lagi berlaku."
			}
			return telegram.FormatFinishedMessage(signal.Symbol, text)
		}
	})
}

// PublishTelemetry hanya mencatat perubahan; backend-api membaca langsung
// dari tabel status akun untuk dashboard.
func (p *runnerPublisher) PublishTelemetry(_ context.Context, accountID string) {
	p.logger.Debug("telemetry akun diperbarui", "account_id", accountID)
}

// approveLink membuat token sekali pakai lalu menyusun deep-link.
// Token disimpan sebagai hash; plaintext hanya dikirim ke user.
func (p *runnerPublisher) approveLink(ctx context.Context, userID, signalID string) string {
	token, hash, err := newApprovalToken()
	if err != nil {
		p.logger.Error("gagal membuat token konfirmasi", "error", err.Error())
		return ""
	}

	record := &model.ApprovalToken{
		TokenHash: hash,
		UserID:    userID,
		SignalID:  signalID,
		Action:    "ENTRY",
		ExpiresAt: time.Now().UTC().Add(5 * time.Minute),
	}

	if err := p.store.CreateApprovalToken(ctx, record); err != nil {
		p.logger.Error("gagal menyimpan token konfirmasi", "error", err.Error())
		return ""
	}

	return telegram.DeepLink(p.baseURL, token)
}

// defaultRiskPercent adalah risiko acuan yang ditampilkan di pesan.
// Nilai final tetap dihitung Runner dari pengaturan akun.
const defaultRiskPercent = 0.5
