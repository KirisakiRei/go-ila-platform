package service

import (
	"context"
	"math"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/timezone"
)

// JournalQuery adalah permintaan jurnal eksekusi dari antarmuka.
//
// Semua kolom opsional. PeriodeHari dipakai bila From/To kosong agar
// antarmuka cukup mengirim "30" untuk jurnal 30 hari terakhir.
type JournalQuery struct {
	From     time.Time
	To       time.Time
	Hari     int
	Strategy string
	Symbol   string
	// LimitPositions dan LimitSkipped membatasi jumlah baris per bagian.
	LimitPositions int
	LimitSkipped   int
	// HalamanSkipped adalah halaman daftar peluang terlewat, dimulai dari 1.
	HalamanSkipped int
}

// DefaultJournalDays adalah rentang bawaan jurnal bila pengguna tidak memilih.
//
// Tiga puluh hari dipilih karena cukup memuat banyak trade untuk menilai
// strategi, tanpa memuat riwayat begitu lama sampai halaman berat.
const DefaultJournalDays = 30

// JournalRepositoryReader membaca data jurnal eksekusi user.
type JournalRepositoryReader interface {
	ListPositions(ctx context.Context, userID string, filter repository.JournalFilter, limit int) ([]model.Position, error)
	Stats(ctx context.Context, userID string, filter repository.JournalFilter) (repository.JournalStats, error)
	ListSkipped(ctx context.Context, userID string, filter repository.JournalFilter, offset, limit int) ([]model.UserSignal, error)
	CountSkipped(ctx context.Context, userID string, filter repository.JournalFilter) (int, error)
}

// SignalDetailReader membaca rincian sinyal untuk melengkapi daftar peluang
// yang terlewat. Tanpa ini, baris "tidak tereksekusi" hanya berisi id sinyal.
type SignalDetailReader interface {
	FindSignal(ctx context.Context, signalID string) (*model.Signal, error)
}

// Journal menyusun jurnal eksekusi: posisi, ringkasan, dan peluang terlewat.
//
// Jurnal ini sengaja membaca ulang posisi lewat repository sendiri, bukan
// memakai Positions(), karena jurnal butuh filter rentang, ringkasan agregat,
// dan daftar sinyal yang tidak menjadi posisi - kebutuhan yang tidak dimiliki
// daftar posisi biasa.
func (s *tradingService) Journal(ctx context.Context, userID string, query JournalQuery) (*dto.JournalResponse, error) {
	filter := jurnalFilter(query)

	limitPositions := query.LimitPositions
	if limitPositions <= 0 || limitPositions > DefaultPositionLimit {
		limitPositions = DefaultPositionLimit
	}
	limitSkipped := query.LimitSkipped
	if limitSkipped <= 0 || limitSkipped > DefaultPositionLimit {
		limitSkipped = DefaultPositionLimit
	}

	// Nomor halaman dijaga minimal 1 agar offset tidak pernah negatif.
	halamanSkipped := query.HalamanSkipped
	if halamanSkipped < 1 {
		halamanSkipped = 1
	}
	offsetSkipped := (halamanSkipped - 1) * limitSkipped

	rows, err := s.journal.ListPositions(ctx, userID, filter, limitPositions)
	if err != nil {
		return nil, err
	}

	stats, err := s.journal.Stats(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	skipped, err := s.journal.ListSkipped(ctx, userID, filter, offsetSkipped, limitSkipped)
	if err != nil {
		return nil, err
	}

	totalSkipped, err := s.journal.CountSkipped(ctx, userID, filter)
	if err != nil {
		return nil, err
	}

	return &dto.JournalResponse{
		Stats:       jurnalStatsResponse(stats),
		Positions:   jurnalPositionsResponse(rows),
		Skipped:     s.jurnalSkippedResponse(ctx, skipped),
		SkippedMeta: jurnalMetaResponse(halamanSkipped, limitSkipped, totalSkipped),
		Filters: dto.JournalFiltersResponse{
			From:            formatOptionalDay(filter.From),
			To:              formatOptionalDay(filter.To),
			StrategyVersion: filter.StrategyVersion,
			Symbol:          filter.Symbol,
		},
	}, nil
}

// jurnalFilter menerjemahkan permintaan antarmuka menjadi filter repository.
//
// Rentang hari dihitung pada zona hari akuntansi (WIB) agar "30 hari" berarti
// 30 hari kalender pengguna, bukan 30x24 jam sejak jam server.
func jurnalFilter(query JournalQuery) repository.JournalFilter {
	filter := repository.JournalFilter{
		From:            query.From,
		To:              query.To,
		StrategyVersion: query.Strategy,
		Symbol:          query.Symbol,
	}

	if filter.From.IsZero() && filter.To.IsZero() {
		hari := query.Hari
		if hari <= 0 {
			hari = DefaultJournalDays
		}
		sekarang := time.Now().In(timezone.WIB)
		awalHari := time.Date(sekarang.Year(), sekarang.Month(), sekarang.Day(), 0, 0, 0, 0, timezone.WIB)
		filter.From = awalHari.AddDate(0, 0, -(hari - 1))
		// To eksklusif: awal hari berikutnya.
		filter.To = awalHari.AddDate(0, 0, 1)
	}

	return filter
}

// jurnalStatsResponse menghitung turunan angka yang dipakai antarmuka.
//
// Winrate hanya memakai penutupan yang hasilnya diketahui: menghitung posisi
// manual tanpa laporan broker sebagai "kalah" akan membuat statistik tampak
// lebih buruk daripada kenyataan.
func jurnalStatsResponse(stats repository.JournalStats) dto.JournalStatsResponse {
	selesai := stats.ClosedTP + stats.ClosedSL + stats.ClosedManual

	out := dto.JournalStatsResponse{
		TotalPositions:  stats.TotalPositions,
		OpenPositions:   stats.OpenPositions,
		ClosedTP:        stats.ClosedTP,
		ClosedSL:        stats.ClosedSL,
		ClosedManual:    stats.ClosedManual,
		Untracked:       stats.Untracked,
		Wins:            stats.Wins,
		Losses:          stats.Losses,
		TotalR:          round2(stats.TotalR),
		TotalProfit:     round2(stats.TotalProfit),
		DryRunPositions: stats.DryRunPositions,
	}

	if dievaluasi := stats.Wins + stats.Losses; dievaluasi > 0 {
		out.WinRate = math.Round(float64(stats.Wins)/float64(dievaluasi)*10000) / 100
		out.AverageR = round2(stats.TotalR / float64(dievaluasi))
	}

	_ = selesai
	return out
}

// jurnalPositionsResponse memetakan posisi ke bentuk tampilan.
func jurnalPositionsResponse(rows []model.Position) []dto.PositionResponse {
	out := make([]dto.PositionResponse, 0, len(rows))
	for i := range rows {
		pos := &rows[i]
		out = append(out, dto.PositionResponse{
			ID:              pos.ID,
			Symbol:          pos.Symbol,
			StrategyVersion: pos.StrategyVersion,
			Direction:       pos.Direction,
			TicketNumber:    pos.TicketNumber,
			LotSize:         pos.LotSize,
			OpenPrice:       pos.OpenPrice,
			StopLoss:        pos.StopLoss,
			TakeProfit:      pos.TakeProfit,
			ExitPrice:       pos.ExitPrice,
			ProfitLoss:      pos.ProfitLoss,
			OutcomeR:        pos.OutcomeR,
			RiskPercent:     pos.RiskPercent,
			Status:          pos.Status,
			IsDryRun:        pos.IsDryRun,
			OpenedAt:        pos.OpenedAt,
			OpenedAtPair:    dto.NewTimePair(pos.OpenedAt),
			ClosedAt:        pos.ClosedAt,
			ClosedAtPair:    dto.NewTimePairPtr(pos.ClosedAt),
			PnLKnown:        pnlKnown(pos),
		})
	}
	return out
}

// jurnalSkippedResponse melengkapi peluang terlewat dengan rincian sinyal.
//
// Kegagalan membaca satu sinyal tidak menggugurkan barisnya: pengguna tetap
// perlu tahu bahwa peluang itu ada, walau rincian harganya tidak tersedia.
func (s *tradingService) jurnalSkippedResponse(ctx context.Context, rows []model.UserSignal) []dto.JournalSkippedResponse {
	out := make([]dto.JournalSkippedResponse, 0, len(rows))

	for i := range rows {
		us := &rows[i]
		item := dto.JournalSkippedResponse{
			SignalID:     us.SignalID,
			Status:       us.Status,
			Reason:       us.RejectReason,
			OccurredAt:   us.CreatedAt,
			OccurredPair: dto.NewTimePair(us.CreatedAt),
		}

		if s.signalDetail != nil {
			if signal, err := s.signalDetail.FindSignal(ctx, us.SignalID); err == nil && signal != nil {
				item.Symbol = signal.Symbol
				item.StrategyVersion = signal.StrategyVersion
				item.Direction = signal.Direction
				item.Htf = signal.Htf
				item.Ltf = signal.Ltf
				item.EntryPrice = signal.EntryPrice
				item.StopLoss = signal.StopLoss
				item.TakeProfit = signal.TakeProfit
				item.RiskReward = signal.RiskReward
			}
		}

		out = append(out, item)
	}

	return out
}

// jurnalMetaResponse menyusun ringkasan halaman untuk daftar peluang terlewat.
//
// Total halaman dibulatkan ke atas supaya sisa baris tetap punya halaman.
// Halaman yang melewati batas (mis. filter berubah saat pengguna sedang di
// halaman 5) tidak menghasilkan foto halaman kosong tanpa penjelasan: has_more
// bernilai false dan antarmuka dapat memindahkan pengguna ke halaman terakhir.
func jurnalMetaResponse(halaman, perHalaman, total int) dto.JournalMetaResponse {
	halamanTotal := 0
	if perHalaman > 0 {
		halamanTotal = (total + perHalaman - 1) / perHalaman
	}

	return dto.JournalMetaResponse{
		Page:       halaman,
		PerPage:    perHalaman,
		Total:      total,
		TotalPages: halamanTotal,
		HasMore:    halaman*perHalaman < total,
	}
}

// formatOptionalDay menulis waktu sebagai tanggal pada zona hari akuntansi.
func formatOptionalDay(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(timezone.WIB).Format("2006-01-02")
}

// round2 membulatkan ke dua desimal untuk tampilan.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
