package runner

import (
	"context"
	"math"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/model"
)

// MonitorInterval adalah jeda pemeriksaan posisi yang masih hidup.
// Cukup rapat untuk PnL yang terasa real-time, cukup longgar untuk VPS 1 GB.
const MonitorInterval = 20 * time.Second

// ApplyClose mencatat penutupan posisi beserta notifikasi ke user.
// Diekspor agar perilaku ini bisa diuji tanpa broker sungguhan.
func (s *Service) ApplyClose(ctx context.Context, pos *model.Position, closeStatus string, exitPrice, profit float64) {
	s.applyClose(ctx, pos, broker.ClosedPosition{
		Ticket:    pos.TicketNumber,
		Reason:    closeReasonFromStatus(closeStatus),
		ExitPrice: exitPrice,
		Profit:    profit,
		ClosedAt:  s.opts.Clock().Unix(),
	})
}

// SetClock mengganti sumber waktu, dipakai saat pengujian.
func (s *Service) SetClock(clock func() time.Time) {
	if clock != nil {
		s.opts.Clock = clock
	}
}

// closeReasonFromStatus menerjemahkan status akhir posisi ke alasan broker.
func closeReasonFromStatus(status string) broker.CloseReason {
	switch status {
	case model.PositionStatusClosedTP:
		return broker.CloseReasonTP
	case model.PositionStatusClosedSL:
		return broker.CloseReasonSL
	default:
		return broker.CloseReasonManual
	}
}

// MonitorPositions memeriksa posisi yang masih OPEN lalu menutup yang sudah
// tidak ada di broker.
//
// Runner tidak menutup posisi berdasarkan feed lokal: status tutup selalu
// berasal dari laporan broker (plans/05 §10).
func (s *Service) MonitorPositions(ctx context.Context) {
	positions, err := s.store.ListOpenPositions(ctx)
	if err != nil {
		s.logger.Error("gagal membaca posisi terbuka", "error", err.Error())
		return
	}

	for i := range positions {
		if ctx.Err() != nil {
			return
		}
		s.syncPosition(ctx, &positions[i])
	}
}

// syncPosition menyelaraskan satu posisi lokal dengan keadaan broker.
func (s *Service) syncPosition(ctx context.Context, pos *model.Position) {
	// Simulasi tidak punya broker; penutupan simulasi dihitung dari harga.
	if pos.IsDryRun {
		return
	}

	account, err := s.store.FindBrokerAccount(ctx, pos.BrokerAccountID, pos.UserID)
	if err != nil {
		s.logger.Warn("akun posisi tidak ditemukan",
			"position_id", pos.ID, "account_id", pos.BrokerAccountID)
		return
	}

	client, err := s.client(ctx, account)
	if err != nil {
		s.markBrokerDown(ctx, account, err)
		return
	}

	live, err := client.ListPositions(ctx)
	if err != nil {
		s.markBrokerDown(ctx, account, err)
		return
	}

	for _, item := range live {
		if item.Ticket == pos.TicketNumber {
			// Masih hidup: tidak ada perubahan status. Runner tidak pernah
			// mengubah SL/TP yang sudah terpasang di broker.
			return
		}
	}

	// Posisi hilang dari broker: ambil alasan tutupnya dari riwayat.
	closed, err := client.ClosedSince(ctx, pos.OpenedAt.Unix()-60)
	if err != nil {
		s.logger.Warn("riwayat penutupan tidak terbaca",
			"position_id", pos.ID, "error", err.Error())
		return
	}

	for _, item := range closed {
		if item.Ticket != pos.TicketNumber {
			continue
		}
		s.applyClose(ctx, pos, item)
		return
	}

	// Tidak ada di posisi hidup maupun riwayat: catat sebagai ditutup manual
	// tanpa angka. Hasilnya tidak diketahui, jadi tidak boleh dihitung sebagai
	// menang maupun kalah (plans/19 C9).
	s.applyClose(ctx, pos, broker.ClosedPosition{
		Ticket: pos.TicketNumber,
		Reason: broker.CloseReasonOther,
	})
}

// applyClose menulis hasil akhir posisi dan memperbarui kuota harian.
func (s *Service) applyClose(ctx context.Context, pos *model.Position, closed broker.ClosedPosition) {
	now := s.opts.Clock()

	pos.ExitPrice = closed.ExitPrice
	pos.ProfitLoss = closed.Profit
	pos.Status = mapCloseReason(closed.Reason)
	pos.ClosedAt = &now

	// Broker yang tidak melaporkan exit price berarti hasil tidak diketahui.
	// Tanpa tanda ini, PnL 0 akan terbaca sebagai "impas" di dashboard.
	pnlKnown := closed.ExitPrice != 0 || closed.Profit != 0

	if risk := pos.LotSize * absDistance(pos.OpenPrice, pos.StopLoss); risk > 0 && pnlKnown {
		pos.OutcomeR = roundTo(pos.ProfitLoss/risk, 2)
	}

	if err := s.store.ClosePosition(ctx, pos); err != nil {
		s.logger.Error("gagal menutup posisi",
			"position_id", pos.ID, "error", err.Error())
		return
	}

	won := pos.ProfitLoss > 0
	day := DayKey(now)

	// Hasil yang tidak diketahui tidak dicatat sebagai kalah: mencatatnya
	// akan menghabiskan kuota user tanpa trade yang benar-benar kalah
	// (plans/01 O15).
	if pnlKnown {
		if err := s.store.RecordDailyOutcome(ctx, pos.UserID, day, won); err != nil {
			s.logger.Error("gagal mencatat hasil harian", "error", err.Error())
		}
	} else {
		s.logger.Warn("hasil posisi tidak diketahui broker; tidak dihitung ke kuota",
			"position_id", pos.ID, "ticket", pos.TicketNumber)
	}

	daily := &DailyOutcome{
		TotalTrades: 0,
		MaxTrades:   s.opts.Policy.MaxTrades,
	}

	// 1 win = stop trading hari ini.
	if won && s.opts.Policy.StopAfterWin {
		if err := s.store.LockDaily(ctx, pos.UserID, day, string(RejectDailyWin)); err != nil {
			s.logger.Error("gagal mengunci hari user", "error", err.Error())
		} else {
			daily.Locked = true
			daily.LockReason = string(RejectDailyWin)
		}
	}

	// Sisa kuota dibaca setelah pencatatan agar pesan yang dikirim akurat.
	if summary, err := s.store.GetDailySummary(ctx, pos.UserID, day); err == nil && summary != nil {
		daily.TotalTrades = summary.TotalTrades
	}

	s.logger.Info("posisi ditutup",
		"position_id", pos.ID,
		"ticket", pos.TicketNumber,
		"status", pos.Status,
		"pnl", pos.ProfitLoss,
		"outcome_r", pos.OutcomeR)

	s.pub.PublishTelemetry(ctx, pos.BrokerAccountID)

	// Notifikasi tutup posisi wajib dikirim: tanpa ini user tidak tahu hasil
	// trading-nya dan harus membuka dashboard sendiri (plans/19 C1).
	s.pub.PublishPositionClosed(ctx, pos, daily)
}

// Reconcile menyelaraskan database dengan broker saat Runner mulai.
//
// Langkah 4 dan 5 sengaja TIDAK memperbaiki apa pun secara otomatis:
// menyentuh posisi nyata tanpa keputusan manusia lebih berbahaya daripada
// mencatat selisih dan memberi tahu (plans/16 §8.1).
func (s *Service) Reconcile(ctx context.Context) {
	positions, err := s.store.ListOpenPositions(ctx)
	if err != nil {
		s.logger.Error("rekonsiliasi: gagal membaca posisi", "error", err.Error())
		return
	}

	if len(positions) == 0 {
		return
	}

	byAccount := make(map[string][]model.Position)
	for _, pos := range positions {
		if pos.IsDryRun {
			continue
		}
		byAccount[pos.BrokerAccountID] = append(byAccount[pos.BrokerAccountID], pos)
	}

	for accountID, items := range byAccount {
		account, err := s.store.FindBrokerAccount(ctx, accountID, items[0].UserID)
		if err != nil {
			continue
		}

		client, err := s.client(ctx, account)
		if err != nil {
			s.logger.Warn("rekonsiliasi: broker tidak terhubung",
				"account_id", accountID, "error", err.Error())
			continue
		}

		live, err := client.ListPositions(ctx)
		if err != nil {
			s.logger.Warn("rekonsiliasi: posisi broker tidak terbaca",
				"account_id", accountID, "error", err.Error())
			continue
		}

		liveTickets := make(map[string]broker.PositionState, len(live))
		for _, item := range live {
			liveTickets[item.Ticket] = item
		}

		localTickets := make(map[string]bool, len(items))
		for i := range items {
			pos := &items[i]
			localTickets[pos.TicketNumber] = true

			state, alive := liveTickets[pos.TicketNumber]
			if !alive {
				// Ada di DB, tidak ada di broker: catat penutupan.
				s.syncPosition(ctx, pos)
				continue
			}

			// Cocokkan SL/TP: hanya dicatat bila berbeda, tidak diubah.
			if !almostEqual(state.StopLoss, pos.StopLoss) || !almostEqual(state.TakeProfit, pos.TakeProfit) {
				s.logger.Warn("rekonsiliasi: SL/TP berbeda dengan catatan",
					"position_id", pos.ID,
					"ticket", pos.TicketNumber,
					"db_sl", pos.StopLoss, "broker_sl", state.StopLoss,
					"db_tp", pos.TakeProfit, "broker_tp", state.TakeProfit)
			}
		}

		// Ada di broker, tidak ada di DB: jangan ditutup, cukup dilaporkan.
		for ticket := range liveTickets {
			if !localTickets[ticket] {
				s.logger.Error("rekonsiliasi: posisi tak terlacak di broker",
					"account_id", accountID, "ticket", ticket)
			}
		}
	}
}

func mapCloseReason(reason broker.CloseReason) string {
	switch reason {
	case broker.CloseReasonTP:
		return model.PositionStatusClosedTP
	case broker.CloseReasonSL:
		return model.PositionStatusClosedSL
	default:
		return model.PositionStatusClosedManual
	}
}

func absDistance(a, b float64) float64 {
	d := a - b
	if d < 0 {
		return -d
	}
	return d
}

func almostEqual(a, b float64) bool {
	const epsilon = 0.0000001
	return absDistance(a, b) < epsilon
}

func roundTo(value float64, digits int) float64 {
	factor := math.Pow(10, float64(digits))
	return math.Round(value*factor) / factor
}
