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
	"ila/backend/internal/settings"
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
	// flags memberi angka kuota harian yang berlaku saat ini. Tanpa ini
	// notifikasi memakai nilai bawaan yang bisa berbeda dari pengaturan admin.
	flags runner.Flags
}

// PublishSignal mengirim notifikasi untuk satu perubahan status sinyal user.
func (p *runnerPublisher) PublishSignal(ctx context.Context, userID, signalID, status, reason string) {
	signal, err := p.store.FindSignal(ctx, signalID)
	if err != nil {
		p.logger.Warn("notifikasi dilewati: sinyal tidak ditemukan", "signal_id", signalID)
		return
	}

	// Notifikasi hanya untuk status yang punya arti bagi user.
	// Status standby/idle (WAIT_HTF_SWEEP dan WAIT_ILA3_HTF_KEY_LEVEL) tidak dikirim
	// ke Telegram agar tidak menyampah saat belum ada reaksi harga aktif.
	switch status {
	case model.SignalStatusReadyToEntry, model.SignalStatusRejected,
		model.SignalStatusOpportunity,
		model.SignalStatusWaitHTFCandleClose, "WAIT_ILA3_MTF_REVERSAL",
		model.SignalStatusWaitLTFCISD, "WAIT_ILA3_LTF_CISD_1", "WAIT_ILA3_MTF_FVG_RETR", "WAIT_ILA3_LTF_CISD_2",
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
		case model.SignalStatusWaitHTFCandleClose, "WAIT_ILA3_MTF_REVERSAL":
			return telegram.FormatRadarHTFMessage(p.radarHTFParams(ctx, signal))

		case "WAIT_HTF_SWEEP":
			return telegram.FormatRadarSweepMessage(p.radarHTFParams(ctx, signal))

		case model.SignalStatusWaitLTFCISD, "WAIT_ILA3_LTF_CISD_1", "WAIT_ILA3_MTF_FVG_RETR", "WAIT_ILA3_LTF_CISD_2":
			dashboardLink := telegram.AppURL(p.baseURL, "/sinyal?pair="+signal.Symbol)
			var cisd dto.SignalCISD
			if signal.CisdJSON != "" {
				_ = json.Unmarshal([]byte(signal.CisdJSON), &cisd)
			}

			// Target dan batas sudah diketahui sejak struktur besar terbentuk,
			// walau konfirmasi di timeframe kecil belum muncul. Mengosongkannya
			// membuat user kehilangan gambaran setup di tahap paling penting.
			var htfSweep dto.SignalHtfSweep
			if signal.HtfSweepJSON != "" {
				_ = json.Unmarshal([]byte(signal.HtfSweepJSON), &htfSweep)
			}
			batas := signal.StopLoss
			if batas == 0 {
				batas = htfSweep.StopLoss
			}
			target := signal.TakeProfit
			if target == 0 {
				target = htfSweep.TargetDOL
			}

			return telegram.FormatRadarLTFMessage(telegram.RadarLTFParams{
				Symbol:          signal.Symbol,
				Direction:       signal.Direction,
				StrategyVersion: signal.StrategyVersion,
				Htf:             signal.Htf,
				Ltf:             signal.Ltf,
				EntryPrice:      signal.EntryPrice,
				StopLoss:        batas,
				TakeProfit:      target,
				RiskReward:      signal.RiskReward,
				CISDLevel:       cisd.CISDLevel,
				Details:         signal.Summary,
				Siklus:          siklusCandle(signal.Htf, signal.PublishedAt),
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
					BrokerType:   p.namaBroker(ctx, pos.BrokerAccountID, pos.UserID),
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
				Htf:          signal.Htf,
				Ltf:          signal.Ltf,
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
			alasan := runner.RejectReason(reason)
			// Penahanan karena tier atau metode hanya tampil di dashboard.
			// Mengirimnya berulang kali mengganggu user yang memang belum berhak.
			if !runner.RejectTerkirimKeTelegram(alasan) {
				return telegram.Message{}
			}
			return telegram.FormatHeldMessage(telegram.HeldParams{
				Symbol:          signal.Symbol,
				Direction:       signal.Direction,
				StrategyVersion: signal.StrategyVersion,
				Htf:             signal.Htf,
				Ltf:             signal.Ltf,
				EntryPrice:      signal.EntryPrice,
				StopLoss:        signal.StopLoss,
				TakeProfit:      signal.TakeProfit,
				RiskReward:      signal.RiskReward,
				Alasan:          alasanPenahanan(alasan),
				Keterangan:      alasan.Teks(),
				KuotaTerpakai:   p.kuotaAngka(ctx, userID, alasan, false),
				KuotaMaks:       p.kuotaAngka(ctx, userID, alasan, true),
				BatasTunggu:     batasTunggu(signal.ExpiresAt),
				Link:            p.tautanTindakan(ctx, userID, signalID, signal.Symbol),
			})

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
			return telegram.FormatCancelledMessage(telegram.HeldParams{
				Symbol:          signal.Symbol,
				Direction:       signal.Direction,
				StrategyVersion: signal.StrategyVersion,
				Htf:             signal.Htf,
				Ltf:             signal.Ltf,
				Keterangan:      text,
				Alasan:          telegram.PenahananBatal,
				Link:            telegram.AppURL(p.baseURL, "/sinyal?pair="+signal.Symbol),
			})
		}
	})
}

// radarHTFParams menyusun parameter radar tahap 1 dan 2 dari baris sinyal.
func (p *runnerPublisher) radarHTFParams(ctx context.Context, signal *model.Signal) telegram.RadarHTFParams {
	_ = ctx
	var htfSweep dto.SignalHtfSweep
	if signal.HtfSweepJSON != "" {
		_ = json.Unmarshal([]byte(signal.HtfSweepJSON), &htfSweep)
	}

	batas := htfSweep.StopLoss
	if batas == 0 {
		batas = signal.StopLoss
	}
	detail := htfSweep.Details
	if detail == "" {
		// Summary engine memuat istilah internal, jadi disaring lebih dulu.
		detail = telegram.Sanitize(signal.Summary)
	}

	return telegram.RadarHTFParams{
		Symbol:          signal.Symbol,
		Direction:       signal.Direction,
		StrategyVersion: signal.StrategyVersion,
		Htf:             signal.Htf,
		Ltf:             signal.Ltf,
		TargetDOL:       htfSweep.TargetDOL,
		StopLoss:        batas,
		Details:         detail,
		Siklus:          siklusCandle(signal.Htf, signal.PublishedAt),
		DashboardLink:   telegram.AppURL(p.baseURL, "/sinyal?pair="+signal.Symbol),
	}
}

// siklusCandle menyebut candle yang sedang berjalan dan jam New York-nya.
func siklusCandle(htf string, terbit time.Time) string {
	if htf == "" || terbit.IsZero() {
		return ""
	}
	ny := terbit.UTC().Add(-5 * time.Hour)
	return fmt.Sprintf("Candle %s · %s NY", htf, ny.Format("15:04"))
}

// alasanPenahanan menerjemahkan grup alasan Runner ke kelompok tampilan.
func alasanPenahanan(reason runner.RejectReason) telegram.AlasanPenahanan {
	switch runner.RejectGroupOf(reason) {
	case runner.GroupKuota:
		return telegram.PenahananKuota
	case runner.GroupPosisi:
		return telegram.PenahananPosisi
	case runner.GroupAkun:
		return telegram.PenahananAkun
	case runner.GroupPasar:
		return telegram.PenahananPasar
	case runner.GroupSistem:
		return telegram.PenahananSistem
	case runner.GroupHarga:
		return telegram.PenahananHarga
	default:
		return telegram.PenahananBatal
	}
}

// batasTunggu menampilkan batas waktu dalam WIB; kosong bila tidak ada.
func batasTunggu(expiresAt time.Time) string {
	if expiresAt.IsZero() {
		return ""
	}
	wib := expiresAt.In(time.FixedZone("WIB", 7*3600))
	return wib.Format("02 Jan 15:04") + " WIB"
}

// tautanTindakan memilih tombol: approve bila masih boleh dieksekusi manual,
// dashboard bila tidak.
func (p *runnerPublisher) tautanTindakan(ctx context.Context, userID, signalID, symbol string) string {
	if link := p.approveLink(ctx, userID, signalID); link != "" {
		return link
	}
	return telegram.AppURL(p.baseURL, "/sinyal?pair="+symbol)
}

// namaBroker membaca label broker akun untuk notifikasi posisi.
// Gagal membaca bukan alasan menahan notifikasi; baris broker hanya dilewati.
func (p *runnerPublisher) namaBroker(ctx context.Context, accountID, userID string) string {
	if accountID == "" || userID == "" {
		return ""
	}
	akun, err := p.store.FindBrokerAccount(ctx, accountID, userID)
	if err != nil || akun == nil {
		return ""
	}
	return akun.BrokerType
}

// kuotaAngka mengisi baris kuota harian untuk penahanan karena aturan harian.
//
// Angka hanya diambil saat memang relevan: membaca ringkasan harian untuk
// alasan lain hanya menambah query tanpa mengubah isi pesan.
func (p *runnerPublisher) kuotaAngka(ctx context.Context, userID string, alasan runner.RejectReason, mauMaks bool) int {
	if runner.RejectGroupOf(alasan) != runner.GroupKuota {
		return 0
	}

	if mauMaks {
		// Batas dibaca dari pengaturan aktif, bukan nilai bawaan, agar pesan
		// menyebut angka yang benar-benar dipakai Runner saat menahan setup.
		maks := runner.DefaultDailyPolicy().MaxTrades
		if p.flags != nil {
			maks = p.flags.Int(ctx, settings.KeyMaxDailyTrades, maks)
		}
		return runner.DailyPolicy{MaxTrades: maks}.Clamp().MaxTrades
	}

	summary, err := p.store.FindDailySummary(ctx, userID, runner.DayKey(time.Now()))
	if err != nil {
		p.logger.Warn("kuota harian tidak terbaca untuk notifikasi", "user_id", userID, "error", err.Error())
		return 0
	}
	return summary.TotalTrades
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
		// Timeframe diambil dari sinyal asal agar user tahu setup mana yang
		// menghasilkan posisi ini. Gagal membaca bukan alasan menahan kabar.
		htf, ltf := p.timeframePosisi(ctx, pos)
		return telegram.FormatClosedMessage(telegram.ClosedParams{
			Symbol:      pos.Symbol,
			Direction:   pos.Direction,
			Htf:         htf,
			Ltf:         ltf,
			Result:      positionResultLabel(pos.Status),
			EntryPrice:  pos.OpenPrice,
			ExitPrice:   pos.ExitPrice,
			ProfitLoss:  pos.ProfitLoss,
			OutcomeR:    pos.OutcomeR,
			Duration:    durasiPosisi(pos.OpenedAt, pos.ClosedAt),
			DailyNote:   dailyNote,
			RiwayatLink: posisiLink,
		})
	})
}

// timeframePosisi membaca pasangan timeframe dari sinyal yang membuka posisi.
// Kosong bila sinyal asalnya tidak ditemukan; kolomnya lalu dilewati saja.
func (p *runnerPublisher) timeframePosisi(ctx context.Context, pos *model.Position) (string, string) {
	if pos.SourceSignalID == "" {
		return "", ""
	}
	signal, err := p.store.FindSignal(ctx, pos.SourceSignalID)
	if err != nil || signal == nil {
		return "", ""
	}
	return signal.Htf, signal.Ltf
}

// durasiPosisi menghitung lama posisi terbuka dalam bahasa yang ringkas.
func durasiPosisi(buka time.Time, tutup *time.Time) string {
	if buka.IsZero() {
		return ""
	}
	akhir := time.Now()
	if tutup != nil && !tutup.IsZero() {
		akhir = *tutup
	}
	lama := akhir.Sub(buka)
	if lama < 0 {
		return ""
	}

	jam := int(lama.Hours())
	menit := int(lama.Minutes()) % 60
	switch {
	case jam >= 24:
		return fmt.Sprintf("%d hari %d jam", jam/24, jam%24)
	case jam > 0:
		return fmt.Sprintf("%d jam %d menit", jam, menit)
	default:
		return fmt.Sprintf("%d menit", menit)
	}
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
