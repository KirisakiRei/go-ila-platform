package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/runner/notify"
	"ila/backend/internal/telegram"
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
		model.SignalStatusOpportunity,
		model.SignalStatusWaitHTFSweep, model.SignalStatusWaitHTFCandleClose, "WAIT_ILA3_HTF_KEY_LEVEL",
		model.SignalStatusWaitLTFCISD, "WAIT_ILA3_MTF_REVERSAL", "WAIT_ILA3_LTF_CISD_1", "WAIT_ILA3_MTF_FVG_RETR", "WAIT_ILA3_LTF_CISD_2",
		model.SignalStatusConsumed,
		model.SignalStatusCancelled, model.SignalStatusExpired,
		model.SignalStatusSuperseded:
	default:
		return
	}

	messageKey := "sig:" + signalID + ":" + userID + ":" + status

	p.notifier.Enqueue(userID, messageKey, func(chatID string) telegram.Message {
		_ = chatID

		switch status {
		case model.SignalStatusWaitHTFSweep, model.SignalStatusWaitHTFCandleClose, "WAIT_ILA3_HTF_KEY_LEVEL":
			dashboardLink := telegram.AppURL(p.baseURL, "/sinyal?pair="+signal.Symbol)
			var htfSweep dto.SignalHtfSweep
			if signal.HtfSweepJSON != "" {
				_ = json.Unmarshal([]byte(signal.HtfSweepJSON), &htfSweep)
			}
			targetDOL := htfSweep.TargetDOL
			stopLoss := htfSweep.StopLoss
			if stopLoss == 0 {
				stopLoss = signal.StopLoss
			}
			details := htfSweep.Details
			if details == "" {
				details = signal.Summary
			}
			return telegram.FormatRadarHTFMessage(telegram.RadarHTFParams{
				Symbol:          signal.Symbol,
				Direction:       signal.Direction,
				StrategyVersion: signal.StrategyVersion,
				Htf:             signal.Htf,
				Ltf:             signal.Ltf,
				TargetDOL:       targetDOL,
				StopLoss:        stopLoss,
				Details:         details,
				DashboardLink:   dashboardLink,
			})

		case model.SignalStatusWaitLTFCISD, "WAIT_ILA3_MTF_REVERSAL", "WAIT_ILA3_LTF_CISD_1", "WAIT_ILA3_MTF_FVG_RETR", "WAIT_ILA3_LTF_CISD_2":
			dashboardLink := telegram.AppURL(p.baseURL, "/sinyal?pair="+signal.Symbol)
			var cisd dto.SignalCISD
			if signal.CisdJSON != "" {
				_ = json.Unmarshal([]byte(signal.CisdJSON), &cisd)
			}
			details := cisd.Details
			if details == "" {
				details = signal.Summary
			}
			return telegram.FormatRadarLTFMessage(telegram.RadarLTFParams{
				Symbol:          signal.Symbol,
				Direction:       signal.Direction,
				StrategyVersion: signal.StrategyVersion,
				Htf:             signal.Htf,
				Ltf:             signal.Ltf,
				EntryPrice:      signal.EntryPrice,
				StopLoss:        signal.StopLoss,
				TakeProfit:      signal.TakeProfit,
				RiskReward:      signal.RiskReward,
				CISDLevel:       cisd.CISDLevel,
				Details:         details,
				DashboardLink:   dashboardLink,
			})

		case model.SignalStatusReadyToEntry:
			link := p.approveLink(ctx, userID, signalID)
			return telegram.FormatSetupDetailedMessage(telegram.SetupMessageParams{
				Symbol:          signal.Symbol,
				Direction:       signal.Direction,
				StrategyVersion: signal.StrategyVersion,
				Htf:             signal.Htf,
				Ltf:             signal.Ltf,
				EntryPrice:      signal.EntryPrice,
				StopLoss:        signal.StopLoss,
				TakeProfit:      signal.TakeProfit,
				RiskReward:      signal.RiskReward,
				RiskPercent:     defaultRiskPercent,
				RiskAmount:      0,
				ApprovalLink:    link,
			})

		case model.SignalStatusConsumed:
			posisiLink := telegram.AppURL(p.baseURL, "/posisi")
			pos, _ := p.store.FindOpenPositionBySignal(ctx, userID, signalID)
			if pos != nil {
				return telegram.FormatPositionOpenedMessage(telegram.PositionOpenedParams{
					Symbol:       pos.Symbol,
					Direction:    pos.Direction,
					TicketNumber: pos.TicketNumber,
					BrokerType:   "",
					LotSize:      pos.LotSize,
					OpenPrice:    pos.OpenPrice,
					StopLoss:     pos.StopLoss,
					TakeProfit:   pos.TakeProfit,
					OutcomeR:     pos.OutcomeR,
					OpenedAt:     pos.OpenedAt,
					PosisiLink:   posisiLink,
				})
			}
			return telegram.FormatPositionOpenedMessage(telegram.PositionOpenedParams{
				Symbol:       signal.Symbol,
				Direction:    signal.Direction,
				TicketNumber: "",
				BrokerType:   "",
				LotSize:      0,
				OpenPrice:    signal.EntryPrice,
				StopLoss:     signal.StopLoss,
				TakeProfit:   signal.TakeProfit,
				OutcomeR:     signal.RiskReward,
				OpenedAt:     time.Now(),
				PosisiLink:   posisiLink,
			})

		case model.SignalStatusRejected:
			return telegram.FormatHeldMessage(
				signal.Symbol, signal.Direction, signal.StrategyVersion,
				signal.RiskReward, reason,
			)

		case model.SignalStatusOpportunity:
			// Peluang bukan penolakan: pesannya menawarkan langkah berikutnya.
			dashboardLink := p.approveLink(ctx, userID, signalID)
			if dashboardLink == "" {
				dashboardLink = telegram.AppURL(p.baseURL, "/sinyal?pair="+signal.Symbol)
			}
			return telegram.FormatOpportunityMessage(
				signal.Symbol, signal.Direction, signal.StrategyVersion,
				signal.Htf, signal.Ltf,
				signal.EntryPrice, signal.StopLoss, signal.TakeProfit,
				signal.RiskReward, dashboardLink,
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

// PublishPositionClosed memberi tahu user bahwa posisinya sudah ditutup,
// beserta hasil dan sisa kuota hari ini.
func (p *runnerPublisher) PublishPositionClosed(ctx context.Context, pos *model.Position, daily *runner.DailyOutcome) {
	if pos == nil {
		return
	}

	messageKey := "pos:closed:" + pos.ID

	dailyNote := p.dailyNote(daily)

	p.notifier.Enqueue(pos.UserID, messageKey, func(chatID string) telegram.Message {
		_ = chatID
		posisiLink := telegram.AppURL(p.baseURL, "/posisi")
		return telegram.FormatClosedMessage(
			pos.Symbol,
			positionResultLabel(pos.Status),
			pos.OpenPrice,
			pos.ExitPrice,
			pos.ProfitLoss,
			pos.OutcomeR,
			dailyNote,
			posisiLink,
		)
	})
}

// positionResultLabel menerjemahkan status teknis menjadi bahasa user.
func positionResultLabel(status string) string {
	switch status {
	case model.PositionStatusClosedTP:
		return "Target tercapai"
	case model.PositionStatusClosedSL:
		return "Stop loss"
	case model.PositionStatusClosedManual:
		return "Ditutup manual"
	default:
		return "Selesai"
	}
}

// dailyNote menyusun catatan kuota agar user tahu apakah masih bisa trade.
func (p *runnerPublisher) dailyNote(daily *runner.DailyOutcome) string {
	if daily == nil {
		return ""
	}
	if daily.Locked {
		return "Trading dihentikan untuk hari ini. Lanjut besok."
	}
	remaining := daily.MaxTrades - daily.TotalTrades
	if remaining < 0 {
		remaining = 0
	}
	if remaining == 0 {
		return "Kuota trading hari ini sudah terpakai."
	}
	return fmt.Sprintf("Sisa kuota hari ini: %d trade.", remaining)
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
