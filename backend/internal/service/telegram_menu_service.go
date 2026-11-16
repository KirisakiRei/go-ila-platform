package service

import (
	"context"
	"fmt"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/runner"
	"ila/backend/internal/telegram"
	"ila/backend/internal/timezone"
)

// TelegramMenuService menyusun isi halaman menu yang dibuka dari Telegram.
//
// Semua halaman di sini hanya membaca data. Tidak ada perubahan pengaturan
// akun yang bisa dilakukan lewat Telegram: kredensial, risiko, dan pair tetap
// dikelola di website.
type TelegramMenuService interface {
	Pantau(ctx context.Context, user *model.User, link string) (telegram.Message, error)
	Riwayat(ctx context.Context, user *model.User, offset int) (telegram.Message, error)
	HariIni(ctx context.Context, user *model.User, link string) (telegram.Message, error)
}

type telegramMenuService struct {
	journal   JournalRepositoryReader
	akun      TelegramPositionReader
	timezone  func() time.Time
	maksTrade int
}

// TelegramPositionReader membaca posisi berjalan dan ringkasan harian.
type TelegramPositionReader interface {
	ListOpenPositionsByUser(ctx context.Context, userID string) ([]model.Position, error)
	FindDailySummary(ctx context.Context, userID, day string) (*model.DailySummary, error)
}

// NewTelegramMenuService merangkai penyusun halaman menu Telegram.
//
// maksTrade adalah batas eksekusi otomatis per hari yang berlaku di Runner,
// dipakai sebagai penyebut kuota pada halaman ringkasan hari.
func NewTelegramMenuService(journal JournalRepositoryReader, akun TelegramPositionReader, maksTrade int) TelegramMenuService {
	if maksTrade <= 0 {
		maksTrade = defaultMaxDailyTrades
	}
	return &telegramMenuService{
		journal:   journal,
		akun:      akun,
		timezone:  time.Now,
		maksTrade: maksTrade,
	}
}

// Pantau menyusun posisi berjalan, setup yang menunggu, dan yang selesai hari ini.
//
// Harga terkini sengaja tidak diambil di sini: menguhubungi broker untuk setiap
// tekanan tombol akan memperlambat balasan. Kolom "Kini" dikosongkan dan
// ditandai belum tersedia, bukan diisi angka lama.
func (s *telegramMenuService) Pantau(ctx context.Context, user *model.User, link string) (telegram.Message, error) {
	sekarang := s.timezone()

	posisi, err := s.akun.ListOpenPositionsByUser(ctx, user.ID)
	if err != nil {
		return telegram.Message{}, fmt.Errorf("membaca posisi berjalan: %w", err)
	}

	barisPosisi := make([]telegram.BarisPantau, 0, len(posisi))
	for _, pos := range posisi {
		barisPosisi = append(barisPosisi, telegram.BarisPantau{
			Ticket:     pos.TicketNumber,
			Symbol:     pos.Symbol,
			Direction:  pos.Direction,
			Lots:       pos.LotSize,
			EntryPrice: pos.OpenPrice,
		})
	}

	// Posisi yang tutup sejak awal hari ikut ditampilkan sebagai penutup
	// halaman, supaya trader tahu apa yang sudah terjadi tanpa membuka riwayat.
	awalHari := runner.DayStart(sekarang)
	tutup, err := s.journal.ListPositions(ctx, user.ID, repository.JournalFilter{
		From:   awalHari,
		To:     awalHari.Add(24 * time.Hour),
		Status: "",
	}, 5)
	if err != nil {
		return telegram.Message{}, fmt.Errorf("membaca posisi selesai hari ini: %w", err)
	}

	barisSelesai := make([]telegram.BarisSelesai, 0, len(tutup))
	for _, pos := range tutup {
		if pos.ClosedAt == nil {
			continue
		}
		barisSelesai = append(barisSelesai, telegram.BarisSelesai{
			Symbol:    pos.Symbol,
			Direction: pos.Direction,
			OutcomeR:  pos.OutcomeR,
			ClosedAt:  *pos.ClosedAt,
		})
	}

	return telegram.FormatPantauMessage(telegram.PantauParams{
		Diperbarui:    sekarang,
		Posisi:        barisPosisi,
		Selesai:       barisSelesai,
		HargaTersedia: false,
		DashboardLink: link,
	}), nil
}

// Riwayat menyusun satu halaman riwayat posisi.
func (s *telegramMenuService) Riwayat(ctx context.Context, user *model.User, offset int) (telegram.Message, error) {
	if offset < 0 {
		offset = 0
	}

	// Satu baris tambahan dipakai untuk mengetahui apakah masih ada halaman
	// berikutnya, sehingga jumlah halaman tidak perlu dihitung dua kali.
	rows, err := s.journal.ListPositions(ctx, user.ID, repository.JournalFilter{},
		offset+telegram.RiwayatPerHalaman+1)
	if err != nil {
		return telegram.Message{}, fmt.Errorf("membaca riwayat posisi: %w", err)
	}

	stats, err := s.journal.Stats(ctx, user.ID, repository.JournalFilter{})
	if err != nil {
		return telegram.Message{}, fmt.Errorf("menghitung ringkasan posisi: %w", err)
	}

	total := stats.TotalPositions
	totalHalaman := (total + telegram.RiwayatPerHalaman - 1) / telegram.RiwayatPerHalaman
	if totalHalaman < 1 {
		totalHalaman = 1
	}

	akhir := offset + telegram.RiwayatPerHalaman
	if akhir > len(rows) {
		akhir = len(rows)
	}
	if offset > len(rows) {
		offset = len(rows)
	}

	baris := make([]telegram.BarisRiwayat, 0, telegram.RiwayatPerHalaman)
	for _, pos := range rows[offset:akhir] {
		tutup := time.Time{}
		if pos.ClosedAt != nil {
			tutup = *pos.ClosedAt
		}
		baris = append(baris, telegram.BarisRiwayat{
			PositionID: pos.ID,
			Symbol:     pos.Symbol,
			Direction:  pos.Direction,
			OutcomeR:   pos.OutcomeR,
			ProfitLoss: pos.ProfitLoss,
			ClosedAt:   tutup,
		})
	}

	halaman := offset/telegram.RiwayatPerHalaman + 1

	return telegram.FormatRiwayatMessage(telegram.RiwayatParams{
		Baris:        baris,
		Halaman:      halaman,
		TotalHalaman: totalHalaman,
		TotalPosisi:  total,
	}), nil
}

// HariIni menyusun ringkasan hari ini beserta angka bulan berjalan.
func (s *telegramMenuService) HariIni(ctx context.Context, user *model.User, link string) (telegram.Message, error) {
	sekarang := s.timezone().In(timezone.WIB)
	hari := sekarang.Format("02 Jan 2006")

	ringkasan, err := s.akun.FindDailySummary(ctx, user.ID, runner.DayKey(sekarang))
	if err != nil {
		return telegram.Message{}, fmt.Errorf("membaca ringkasan harian: %w", err)
	}

	awalBulan := time.Date(sekarang.Year(), sekarang.Month(), 1, 0, 0, 0, 0, timezone.WIB)
	stats, err := s.journal.Stats(ctx, user.ID, repository.JournalFilter{From: awalBulan})
	if err != nil {
		return telegram.Message{}, fmt.Errorf("menghitung ringkasan bulan: %w", err)
	}

	winRate := 0.0
	if dinilai := stats.Wins + stats.Losses; dinilai > 0 {
		winRate = float64(stats.Wins) / float64(dinilai) * 100
	}

	return telegram.FormatHariMessage(telegram.HariParams{
		Hari:          hari,
		TradeTerpakai: ringkasan.TotalTrades,
		TradeMaks:     s.maksTrade,
		Win:           ringkasan.WinCount,
		Loss:          ringkasan.LossCount,
		// Tabel ringkasan harian hanya menyimpan jumlah menang dan kalah,
		// tanpa nominal. Nilainya tidak dikarang dari perkiraan.
		PosisiJalan:   stats.OpenPositions,
		BulanTotal:    stats.TotalPositions,
		BulanWinRate:  winRate,
		BulanNet:      stats.TotalProfit,
		DashboardLink: link,
	}), nil
}

// hariKunci menyusun kunci hari untuk tabel ringkasan harian.
func hariKunci(t time.Time) string {
	return t.In(timezone.WIB).Format("2006-01-02")
}
