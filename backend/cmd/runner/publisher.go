package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/runner/notify"
	"ila/backend/internal/runner/pushnotify"
	"ila/backend/internal/service"
	"ila/backend/internal/settings"
	"ila/backend/internal/telegram"
	"ila/backend/internal/timezone"
)

// runnerPublisher menerjemahkan perubahan status Runner menjadi notifikasi.
//
// Dipisah dari inti Runner agar aturan trading bisa diuji tanpa kanal
// pesan, dan agar kegagalan kanal tidak pernah mengubah keputusan trading.
type runnerPublisher struct {
	store    repository.RunnerRepository
	notifier *notify.Notifier
	push     *pushnotify.Notifier
	baseURL  string
	logger   *slog.Logger
	// flags memberi angka kuota harian yang berlaku saat ini. Tanpa ini
	// notifikasi memakai nilai bawaan yang bisa berbeda dari pengaturan admin.
	flags runner.Flags
	// jejak menyimpan pesan bot yang masih bertombol, agar tombol lama bisa
	// dilepas saat setup tidak berlaku lagi.
	jejak repository.TelegramMessageRepository
	tg    *telegram.Client
}

// PublishSignal mengirim notifikasi untuk satu perubahan status sinyal user.
func (p *runnerPublisher) PublishSignal(ctx context.Context, userID, signalID, status, reason string) {
	if statusTerminal(status) {
		p.lepasTombol(ctx, userID, signalID, status, reason)
	}
	signal, err := p.store.FindSignal(ctx, signalID)
	if err != nil {
		p.logger.Warn("notifikasi dilewati: sinyal tidak ditemukan", "signal_id", signalID)
		return
	}
	if p.sudahPunyaPosisiSejenis(ctx, userID, signal) {
		// Posisi pada pair dan arah yang sama sudah terbuka: kemajuan radar untuk
		// slot itu hanya mengulang setup yang sudah dijalankan.
		return
	}

	// Notifikasi hanya untuk status yang punya arti bagi user.
	//
	// WAIT_HTF_SWEEP ikut dikirim: itu tahap 1, titik masuk siklus. Tanpa
	// kabar tahap 1, user hanya melihat lompatan ke tahap 2 atau 3 tanpa tahu
	// setup baru dimulai (kasus USTEC 17 Nov 2026). Status idle lain
	// (WAIT_ILA3_HTF_KEY_LEVEL) tetap diam karena tidak ada reaksi harga.
	switch status {
	case model.SignalStatusWaitHTFSweep:
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

	// Penanda kejadian membuat tahap yang sama bisa dikirim lagi setelah setup
	// mundur ke tahap itu: kunci `WAIT_HTF_SWEEP` pertama dan `WAIT_HTF_SWEEP`
	// setelah gagal di tahap 3 adalah dua kejadian berbeda. Tanpa penanda ini,
	// kabar bahwa setup kembali menunggu struktur tidak pernah sampai.
	// Kunci notifikasi memakai SLOT, bukan signal_id.
	//
	// Engine memakai dua identitas untuk satu setup: baris status menunggu
	// (`status-USTEC-H4-M15`) untuk tahap 1-3, dan sinyal baru (`sig-...-3`)
	// untuk tahap 4-6. Dengan signal_id sebagai kunci, tahap 5 tidak sadar
	// bahwa tahap 3 sudah dikirim untuk slot yang sama, dan mundurnya setup ke
	// tahap 1 tidak pernah bisa dikenali (kasus USTEC 17 Nov 2026).
	messageKey := fmt.Sprintf("slot:%s:%s:%s:%s:%s:%d",
		signal.Symbol, signal.Htf, signal.Ltf, signal.Direction, userID,
		p.kejadianSlot(ctx, userID, signal))

	if p.notifier != nil {
		p.notifier.EnqueueSignal(userID, messageKey, signalID, func(chatID string) telegram.Message {
		_ = chatID

		switch status {
		case model.SignalStatusWaitHTFSweep:
			// Tahap 1: arah belum dikonfirmasi, jangan pakai kalimat "area kena".
			return telegram.FormatRadarHTFMessage(p.radarHTFParams(ctx, signal))
		case model.SignalStatusWaitHTFCandleClose, "WAIT_ILA3_MTF_REVERSAL":
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
			token, link := p.approveLink(ctx, userID, signalID)
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
				ApproveToken:    token,
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
			_, dashboardLink := p.approveLink(ctx, userID, signalID)
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

	// Web Push HANYA dikirim saat sinyal siap konfirmasi entry manual
	// (READY_TO_ENTRY / OPPORTUNITY), bukan untuk tahap radar (WAIT_*),
	// penahanan (REJECTED), pembatalan, atau user tanpa hak eksekusi.
	if p.push != nil &&
		(status == model.SignalStatusReadyToEntry || status == model.SignalStatusOpportunity) &&
		reason != string(runner.RejectEntitlement) {
		p.enqueueReadyWebPush(userID, signal, messageKey)
	}
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
//
// Zona memakai America/New_York, bukan offset tetap: November masih EDT (UTC-4)
// sementara Desember EST (UTC-5), dan offset tetap membuat jam meleset satu jam
// selama separuh tahun.
func siklusCandle(htf string, terbit time.Time) string {
	if htf == "" || terbit.IsZero() {
		return ""
	}
	ny := terbit.In(timezone.NY)
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
	if _, link := p.approveLink(ctx, userID, signalID); link != "" {
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

	if p.notifier != nil {
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

	// Web Push penutupan posisi dikirim untuk TP, SL, dan tutup manual.
	if p.push != nil &&
		(pos.Status == model.PositionStatusClosedTP ||
			pos.Status == model.PositionStatusClosedSL ||
			pos.Status == model.PositionStatusClosedManual) {
		p.enqueueClosedWebPush(pos)
	}
}

// PublishPositionOpened mengirim notifikasi Web Push ketika posisi entry baru
// berhasil dibuka di broker (baik entry otomatis maupun konfirmasi manual).
func (p *runnerPublisher) PublishPositionOpened(_ context.Context, pos *model.Position, _ *model.Signal, isAuto bool) {
	if p.push == nil || pos == nil {
		return
	}

	posCopy := *pos
	p.push.Enqueue(pushnotify.Job{
		UserID:     posCopy.UserID,
		MessageKey: "wp:open:" + posCopy.ID,
		Kind:       pushnotify.KindEntry,
		TTLSeconds: 86400,
		Topic:      "pos-open-" + posCopy.Symbol,
		Build: func(_ context.Context) service.WebPushPayload {
			title := fmt.Sprintf("✅ %s · %s %.2f lot Tereksekusi", posCopy.Symbol, posCopy.Direction, posCopy.LotSize)
			if isAuto {
				title = fmt.Sprintf("⚡ %s · %s %.2f lot Dibuka (Otomatis)", posCopy.Symbol, posCopy.Direction, posCopy.LotSize)
			}
			body := fmt.Sprintf(
				"Open %s · SL %s · TP %s",
				formatPushPrice(posCopy.Symbol, posCopy.OpenPrice),
				formatPushPrice(posCopy.Symbol, posCopy.StopLoss),
				formatPushPrice(posCopy.Symbol, posCopy.TakeProfit),
			)
			return service.WebPushPayload{
				Title: title,
				Body:  body,
				URL:   "/app/posisi",
				Tag:   "ila-pos-open-" + posCopy.ID,
				Kind:  "entry",
			}
		},
	})
}

func (p *runnerPublisher) enqueueReadyWebPush(userID string, signal *model.Signal, slotKey string) {
	if signal == nil {
		return
	}
	sigCopy := *signal
	ttl := 300
	if !sigCopy.ExpiresAt.IsZero() {
		rem := int(time.Until(sigCopy.ExpiresAt).Seconds())
		if rem < 60 {
			ttl = 60
		} else if rem > 900 {
			ttl = 900
		} else {
			ttl = rem
		}
	}

	p.push.Enqueue(pushnotify.Job{
		UserID:     userID,
		MessageKey: "wp:ready:" + slotKey,
		Kind:       pushnotify.KindEntry,
		TTLSeconds: ttl,
		Topic:      "ready-" + sigCopy.Symbol,
		Build: func(bctx context.Context) service.WebPushPayload {
			rawToken, _ := p.approveLink(bctx, userID, sigCopy.SignalID)
			targetURL := "/app/sinyal?pair=" + sigCopy.Symbol
			if rawToken != "" {
				targetURL = "/app/approve/" + rawToken
			}

			ikon := "🟢"
			if sigCopy.Direction == "SELL" {
				ikon = "🔴"
			}
			tfLabel := strings.TrimSpace(sigCopy.Htf)
			if sigCopy.Ltf != "" {
				if tfLabel != "" {
					tfLabel += " ➔ " + sigCopy.Ltf
				} else {
					tfLabel = sigCopy.Ltf
				}
			}

			title := fmt.Sprintf("%s %s · %s — Siap Entry (%s)", ikon, sigCopy.Symbol, tfLabel, sigCopy.Direction)
			if tfLabel == "" {
				title = fmt.Sprintf("%s %s — Siap Entry (%s)", ikon, sigCopy.Symbol, sigCopy.Direction)
			}

			body := fmt.Sprintf(
				"Entry %s · SL %s · TP %s (RR 1:%.2f). Ketuk untuk konfirmasi.",
				formatPushPrice(sigCopy.Symbol, sigCopy.EntryPrice),
				formatPushPrice(sigCopy.Symbol, sigCopy.StopLoss),
				formatPushPrice(sigCopy.Symbol, sigCopy.TakeProfit),
				sigCopy.RiskReward,
			)

			return service.WebPushPayload{
				Title:              title,
				Body:               body,
				URL:                targetURL,
				Tag:                "ila-ready-" + sigCopy.SignalID,
				Kind:               "entry",
				RequireInteraction: true,
			}
		},
	})
}

func (p *runnerPublisher) enqueueClosedWebPush(pos *model.Position) {
	posCopy := *pos
	p.push.Enqueue(pushnotify.Job{
		UserID:     posCopy.UserID,
		MessageKey: "wp:close:" + posCopy.ID,
		Kind:       pushnotify.KindExit,
		TTLSeconds: 86400,
		Topic:      "pos-close-" + posCopy.Symbol,
		Build: func(_ context.Context) service.WebPushPayload {
			var title string
			switch posCopy.Status {
			case model.PositionStatusClosedTP:
				title = fmt.Sprintf("🎯 %s · Target Tercapai (%+.2fR)", posCopy.Symbol, posCopy.OutcomeR)
			case model.PositionStatusClosedSL:
				title = fmt.Sprintf("🛑 %s · Stop Loss (%+.2fR)", posCopy.Symbol, posCopy.OutcomeR)
			default:
				title = fmt.Sprintf("⏹️ %s · Posisi Ditutup Manual", posCopy.Symbol)
			}

			var body string
			if posCopy.ProfitLoss != 0 || posCopy.OutcomeR != 0 {
				sign := ""
				if posCopy.ProfitLoss > 0 {
					sign = "+"
				}
				body = fmt.Sprintf(
					"Hasil %s$%.2f · Exit %s",
					sign,
					posCopy.ProfitLoss,
					formatPushPrice(posCopy.Symbol, posCopy.ExitPrice),
				)
			} else {
				body = fmt.Sprintf(
					"Exit %s · Ketuk untuk melihat jurnal eksekusi.",
					formatPushPrice(posCopy.Symbol, posCopy.ExitPrice),
				)
			}

			return service.WebPushPayload{
				Title: title,
				Body:  body,
				URL:   "/app/posisi",
				Tag:   "ila-pos-close-" + posCopy.ID,
				Kind:  "exit",
			}
		},
	})
}

func formatPushPrice(symbol string, price float64) string {
	if price <= 0 {
		return "—"
	}
	upper := strings.ToUpper(symbol)
	switch {
	case strings.Contains(upper, "XAU"), strings.Contains(upper, "XAG"),
		strings.Contains(upper, "JPY"), strings.Contains(upper, "US30"),
		strings.Contains(upper, "USTEC"), strings.Contains(upper, "NAS"),
		strings.Contains(upper, "SPX"), strings.Contains(upper, "BTC"),
		strings.Contains(upper, "ETH"):
		return fmt.Sprintf("%.2f", price)
	default:
		return fmt.Sprintf("%.5f", price)
	}
}

// kejadianSlot mengembalikan nomor peralihan status terakhir untuk satu slot
// (pair, pasangan timeframe, dan arah) milik satu user.
//
// Nomor ini yang membuat tahap yang sama bisa dikirim lagi setelah setup
// mundur: `WAIT_HTF_SWEEP` pertama dan `WAIT_HTF_SWEEP` setelah gagal di tahap
// 3 adalah dua kejadian berbeda, sehingga kuncinya berbeda dan kabar mundur
// tetap sampai.
//
// Kegagalan membaca dianggap 0: notifikasi lebih baik terkirim sekali lebih
// sering daripada hilang sama sekali.
func (p *runnerPublisher) kejadianSlot(ctx context.Context, userID string, signal *model.Signal) int {
	jumlah, err := p.store.HitungKejadianSlot(ctx, userID, signal.Symbol, signal.Htf, signal.Ltf, signal.Direction)
	if err != nil {
		p.logger.Warn("gagal menghitung peristiwa slot",
			"symbol", signal.Symbol, "htf", signal.Htf, "ltf", signal.Ltf,
			"error", err.Error())
		return 0
	}
	return jumlah
}

// kejadianSinyal mengembalikan nomor peralihan status terakhir untuk satu
// sinyal dan user. Nilainya stabil selama status tidak berpindah, sehingga
// kiriman berulang tetap memakai kunci yang sama dan tertahan oleh pengklaim
// pesan.
//
// Kegagalan membaca dianggap 0: notifikasi lebih baik terkirim sekali lebih
// sering daripada hilang sama sekali.
func (p *runnerPublisher) kejadianSinyal(ctx context.Context, userID, signalID string) int {
	jumlah, err := p.store.HitungKejadianSinyal(ctx, userID, signalID)
	if err != nil {
		p.logger.Warn("gagal menghitung peristiwa sinyal",
			"signal_id", signalID, "error", err.Error())
		return 0
	}
	return jumlah
}

// sudahPunyaPosisiSejenis melaporkan apakah user sudah memegang posisi terbuka
// pada pair, pasangan timeframe, dan arah yang sama dengan sinyal ini.
//
// Kemajuan radar untuk slot yang sama datang berkali-kali; setelah posisinya
// terbuka, kiriman berikutnya hanya mengulang setup yang sudah dijalankan.
func (p *runnerPublisher) sudahPunyaPosisiSejenis(ctx context.Context, userID string, signal *model.Signal) bool {
	posisi, err := p.store.ListOpenPositionsByUser(ctx, userID)
	if err != nil {
		// Gagal membaca posisi tidak boleh menelan notifikasi.
		return false
	}
	for i := range posisi {
		pos := &posisi[i]
		if pos.SourceSignalID == signal.SignalID {
			// Status untuk sinyal yang membuka posisi itu sendiri tetap harus
			// dikirim: itulah kabar eksekusinya.
			return false
		}
		if !strings.EqualFold(pos.Symbol, signal.Symbol) || pos.Direction != signal.Direction {
			continue
		}
		htf, ltf := p.timeframePosisi(ctx, pos)
		if htf == signal.Htf && ltf == signal.Ltf {
			return true
		}
	}
	return false
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
//
// Token mentah ikut dikembalikan karena tombol inline di Telegram mengirim
// callback berisi token, bukan tautan.
func (p *runnerPublisher) approveLink(ctx context.Context, userID, signalID string) (token string, link string) {
	raw, hash, err := newApprovalToken()
	if err != nil {
		p.logger.Error("gagal membuat token konfirmasi", "error", err.Error())
		return "", ""
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
		return "", ""
	}
	return raw, telegram.DeepLink(p.baseURL, raw)
}

// defaultRiskPercent adalah risiko acuan yang ditampilkan di pesan.
// Nilai final tetap dihitung Runner dari pengaturan akun.
const defaultRiskPercent = 0.5
