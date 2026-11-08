package service

import (
	"context"
	"sort"
	"strconv"
	"time"

	"ila/backend/internal/export"
)

// PositionExport adalah berkas CSV riwayat posisi beserta namanya.
type PositionExport struct {
	Filename string
	Content  []byte
	Rows     int
}

// ExportPositions menyusun riwayat posisi user sebagai CSV.
// Rentang tanggal memakai zona hari akuntansi agar "1 Mei" berarti hari yang
// sama dengan yang dipakai Runner, bukan tengah malam UTC.
func (s *tradingService) ExportPositions(ctx context.Context, userID string, from, to time.Time) (*PositionExport, error) {
	rows, err := s.positions.ListByUser(ctx, userID, maxExportPositions)
	if err != nil {
		return nil, err
	}

	builder := export.New(maxExportPositions)
	if err := builder.Header(
		"simbol", "strategi", "arah", "tiket", "lot", "harga_masuk", "stop_loss", "take_profit",
		"harga_keluar", "laba_rugi_usd", "hasil_r", "risiko_persen", "status", "simulasi",
		"dibuka_wib", "dibuka_ny", "ditutup_wib", "ditutup_ny",
	); err != nil {
		return nil, err
	}

	// Baris terbaru lebih dulu; ekspor mengikuti urutan tampilan.
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].OpenedAt.After(rows[j].OpenedAt) })

	for _, row := range rows {
		if !withinRange(row.OpenedAt, from, to) {
			continue
		}
		openedWIB, openedNY := export.TimeColumns(&row.OpenedAt)
		closedWIB, closedNY := export.TimeColumns(row.ClosedAt)
		// PnL kosong berarti belum terbaca dari broker; nol akan menyesatkan.
		pnl := ""
		if pnlKnown(&row) {
			pnl = strconv.FormatFloat(row.ProfitLoss, 'f', 2, 64)
		}
		err := builder.Row(
			row.Symbol, row.StrategyVersion, row.Direction, row.TicketNumber,
			strconv.FormatFloat(row.LotSize, 'f', 2, 64),
			strconv.FormatFloat(row.OpenPrice, 'f', 5, 64),
			strconv.FormatFloat(row.StopLoss, 'f', 5, 64),
			strconv.FormatFloat(row.TakeProfit, 'f', 5, 64),
			strconv.FormatFloat(row.ExitPrice, 'f', 5, 64),
			pnl,
			strconv.FormatFloat(row.OutcomeR, 'f', 2, 64),
			strconv.FormatFloat(row.RiskPercent, 'f', 2, 64),
			row.Status, boolLabel(row.IsDryRun),
			openedWIB, openedNY, closedWIB, closedNY,
		)
		if err != nil {
			return nil, err
		}
	}

	return &PositionExport{
		Filename: "riwayat-posisi-" + time.Now().UTC().Format("20060102-150405") + ".csv",
		Content:  builder.Finished(),
		Rows:     builder.Rows(),
	}, nil
}

// maxExportPositions adalah batas baris ekspor yang sama dengan batas umum.
const maxExportPositions = 10000

// withinRange mencocokkan waktu terhadap rentang tanggal hari akuntansi.
// Rentang nol berarti tanpa batas pada sisi tersebut.
func withinRange(at, from, to time.Time) bool {
	if !from.IsZero() && at.Before(from) {
		return false
	}
	if !to.IsZero() && !at.Before(to.AddDate(0, 0, 1)) {
		return false
	}
	return true
}

// boolLabel menuliskan nilai boolean dalam bahasa Indonesia.
func boolLabel(value bool) string {
	if value {
		return "ya"
	}
	return "tidak"
}
