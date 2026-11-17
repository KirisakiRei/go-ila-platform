package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
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
	Pantau(ctx context.Context, user *model.User, link string, halaman int) (telegram.Message, error)
	Riwayat(ctx context.Context, user *model.User, offset int) (telegram.Message, error)
	HariIni(ctx context.Context, user *model.User, link string) (telegram.Message, error)
}

type telegramMenuService struct {
	journal   JournalRepositoryReader
	akun      TelegramPositionReader
	timezone  func() time.Time
	maksTrade int
	// radar membaca slot yang sedang berjalan untuk daftar setup menunggu.
	// Boleh nil: tanpa pembaca ini halaman pantauan tetap tampil tanpa daftar itu.
	radar RadarReader
}

// RadarReader membaca slot radar yang sedang berjalan untuk satu user.
type RadarReader interface {
	ListRadarForUser(ctx context.Context, userID string, batas int) ([]model.Signal, error)
}

// TelegramPositionReader membaca posisi berjalan dan ringkasan harian.
type TelegramPositionReader interface {
	ListOpenPositionsByUser(ctx context.Context, userID string) ([]model.Position, error)
	FindDailySummary(ctx context.Context, userID, day string) (*model.DailySummary, error)
}

// NewTelegramMenuService merangkai penyusun halaman menu Telegram.
//
// maksTrade adalah batas eksekusi otomatis per hari yang berlaku di Runner,
// dipakai sebagai penyebut kuota pada halaman ringkasan hari. radar boleh nil:
// tanpa itu halaman pantauan tetap tampil, hanya tanpa daftar setup menunggu.
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

// NewTelegramMenuServiceWithRadar merangkai menu Telegram sekaligus pembaca
// slot radar, sehingga halaman pantauan bisa memuat setup yang sedang menunggu
// beserta tahapnya. Konstruktor publik lama tetap ada agar pemanggil lain tidak
// ikut berubah.
func NewTelegramMenuServiceWithRadar(journal JournalRepositoryReader, akun TelegramPositionReader, radar RadarReader, maksTrade int) TelegramMenuService {
	svc := NewTelegramMenuService(journal, akun, maksTrade).(*telegramMenuService)
	svc.radar = radar
	return svc
}

// Pantau menyusun posisi berjalan, setup yang menunggu, dan yang selesai hari ini.
//
// Harga terkini sengaja tidak diambil di sini: menghubungi broker untuk setiap
// tekanan tombol akan memperlambat balasan. Kolom "Kini" dikosongkan dan
// ditandai belum tersedia, bukan diisi angka lama.
//
// Halaman mulai dari 1 dan hanya membatasi daftar setup menunggu; posisi
// berjalan serta ringkasan hari tetap utuh di setiap halaman.
func (s *telegramMenuService) Pantau(ctx context.Context, user *model.User, link string, halaman int) (telegram.Message, error) {
	sekarang := s.timezone()
	posisi, err := s.akun.ListOpenPositionsByUser(ctx, user.ID)
	if err != nil {
		return telegram.Message{}, fmt.Errorf("membaca posisi berjalan: %w", err)
	}
	barisPosisi := make([]telegram.BarisPantau, 0, len(posisi))
	for _, pos := range posisi {
		barisPosisi = append(barisPosisi, telegram.BarisPantau{
			DetailURL: "view:position:" + pos.ID,
			StopLoss:  pos.StopLoss, TakeProfit: pos.TakeProfit, OpenedAt: pos.OpenedAt,
			Ticket:     pos.TicketNumber,
			Symbol:     pos.Symbol,
			Direction:  pos.Direction,
			Lots:       pos.LotSize,
			EntryPrice: pos.OpenPrice,
		})
	}
	menunggu, totalHalaman, err := s.setupMenunggu(ctx, user.ID, posisi, halaman)
	if err != nil {
		return telegram.Message{}, err
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
		Menunggu:      menunggu,
		Selesai:       barisSelesai,
		HargaTersedia: false,
		DashboardLink: link,
		Halaman:       min(max(1, halaman), max(1, totalHalaman)),
		TotalHalaman:  totalHalaman,
	}), nil
}

// setupMenunggu menyusun daftar setup yang belum menjadi posisi.
//
// Sumbernya baris radar yang masih berjalan. Setup yang sudah membuka posisi
// dilewati: posisinya sudah tampil di bagian atas halaman, jadi menampilkannya
// di sini akan membuat trader mengira ada peluang kedua.
//
// Nilai balik kedua adalah jumlah halaman daftar tersebut.
func (s *telegramMenuService) setupMenunggu(ctx context.Context, userID string, posisi []model.Position, halaman int) ([]telegram.BarisMenunggu, int, error) {
	if s.radar == nil {
		return nil, 0, nil
	}
	radar, err := s.radar.ListRadarForUser(ctx, userID, 100)
	if err != nil {
		return nil, 0, fmt.Errorf("membaca setup menunggu: %w", err)
	}
	sedangJalan := make(map[string]struct{}, len(posisi))
	for _, pos := range posisi {
		if pos.SourceSignalID != "" {
			sedangJalan[pos.SourceSignalID] = struct{}{}
		}
	}

	baris := make([]telegram.BarisMenunggu, 0, len(radar))
	for i := range radar {
		sig := &radar[i]
		if _, sudahJalan := sedangJalan[sig.SignalID]; sudahJalan {
			continue
		}
		tahap, teks := tahapRadar(sig.Status, sig.Htf, sig.Ltf)
		if tahap == "" {
			continue
		}
		// Acuan "sejak" adalah publikasi slot pertama, bukan pembaruan terakhir:
		// setup duplikat yang ditahan tetap memakai slot yang sama, jadi jamnya
		// tidak meloncat setiap tahap maju.
		mulai := sig.PublishedAt
		if mulai.IsZero() {
			mulai = sig.UpdatedAt
		}
		baris = append(baris, telegram.BarisMenunggu{
			DetailURL: "view:radar:" + radarReference(sig.SignalID), StrategyVersion: sig.StrategyVersion,
			Symbol:       sig.Symbol,
			Htf:          sig.Htf,
			Ltf:          sig.Ltf,
			Direction:    sig.Direction,
			Tahap:        tahap,
			TahapTeks:    teks,
			Target:       sig.TakeProfit,
			StopLoss:     sig.StopLoss,
			TriggerHarga: sig.EntryPrice,
			Sejak:        mulai,
			Lama:         lamaRingkas(s.timezone().Sub(mulai)),
		})
	}

	totalHalaman := (len(baris) + telegram.PantauPerHalaman - 1) / telegram.PantauPerHalaman
	if halaman < 1 {
		halaman = 1
	}
	halaman = min(halaman, max(1, totalHalaman))
	awal := (halaman - 1) * telegram.PantauPerHalaman
	if awal > len(baris) {
		awal = len(baris)
	}
	akhir := awal + telegram.PantauPerHalaman
	if akhir > len(baris) {
		akhir = len(baris)
	}
	return baris[awal:akhir], totalHalaman, nil
}

// tahapRadar menerjemahkan status radar menjadi nomor tahap beserta kalimat
// singkatnya. Nomornya sengaja sama dengan yang dipakai notifikasi sehingga
// trader mengenali posisinya tanpa harus belajar istilah baru.
//
// String kosong berarti status itu bukan tahap menunggu entry.
func tahapRadar(status, htf, ltf string) (string, string) {
	tahap := telegram.TahapDariStatusRadar(status)
	if tahap == 0 {
		return "", ""
	}
	return tahap.String(), telegram.KalimatTahapRadar(tahap, htf, ltf)
}

// lamaRingkas mengubah selang waktu menjadi teks pendek yang mudah dibaca.
func lamaRingkas(d time.Duration) string {
	if d < 0 {
		return ""
	}
	menit := int(d.Minutes())
	switch {
	case menit < 60:
		return fmt.Sprintf("%d menit", menit)
	case menit < 24*60:
		return fmt.Sprintf("%d jam %d menit", menit/60, menit%60)
	default:
		return fmt.Sprintf("%d hari %d jam", menit/(24*60), (menit/60)%24)
	}
}

// Riwayat menyusun satu halaman riwayat posisi.
func (s *telegramMenuService) Riwayat(ctx context.Context, user *model.User, offset int) (telegram.Message, error) {
	if offset < 0 {
		offset = 0
	}

	stats, err := s.journal.Stats(ctx, user.ID, repository.JournalFilter{ClosedOnly: true})
	if err != nil {
		return telegram.Message{}, err
	}
	total := stats.ClosedTP + stats.ClosedSL + stats.ClosedManual + stats.Untracked
	totalHalaman := max(1, (total+telegram.RiwayatPerHalaman-1)/telegram.RiwayatPerHalaman)
	offset = min(max(0, offset), (totalHalaman-1)*telegram.RiwayatPerHalaman)
	rows, err := s.journal.ListPositions(ctx, user.ID, repository.JournalFilter{ClosedOnly: true, Offset: offset}, telegram.RiwayatPerHalaman)
	if err != nil {
		return telegram.Message{}, err
	}
	baris := make([]telegram.BarisRiwayat, 0, telegram.RiwayatPerHalaman)
	for _, pos := range rows {
		tutup := time.Time{}
		if pos.ClosedAt != nil {
			tutup = *pos.ClosedAt
		}
		baris = append(baris, telegram.BarisRiwayat{
			PositionID: pos.ID, Status: pos.Status,
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
		Hari: hari, UpdatedAt: sekarang,
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

// Simulasi reads the same persisted outcomes as the website; it never fetches prices.
func (s *telegramMenuService) Simulasi(ctx context.Context, user *model.User, link string, page int) (telegram.Message, error) {
	if page < 1 {
		page = 1
	}
	if page > 10000 {
		page = 10000
	}
	total, err := s.journal.CountSkipped(ctx, user.ID, repository.JournalFilter{})
	if err != nil {
		return telegram.Message{}, err
	}
	pages := max(1, (total+4)/5)
	page = min(page, pages)
	rows, err := s.journal.ListSkipped(ctx, user.ID, repository.JournalFilter{}, (page-1)*5, 5)
	if err != nil {
		return telegram.Message{}, err
	}
	var b strings.Builder
	var details [][]telegram.InlineButton
	fmt.Fprintf(&b, "📋 Tidak tereksekusi\nHasil simulasi, bukan PnL akun.\nHalaman %d dari %d\n", page, pages)
	if len(rows) == 0 {
		b.WriteString("\nBelum ada sinyal untuk ditampilkan.\n")
	}
	for i, row := range rows {
		details = append(details, []telegram.InlineButton{{Text: fmt.Sprintf("%d · Detail simulasi", i+1), CallbackData: "view:simulation:" + row.ID}})
		var sig model.Signal
		if row.EvalSnapshot != "" {
			if err := json.Unmarshal([]byte(row.EvalSnapshot), &sig); err != nil {
				return telegram.Message{}, fmt.Errorf("membaca snapshot: %w", err)
			}
		}
		if row.EvalSnapshot == "" {
			sig.Symbol = "Rincian sinyal belum tersedia"
		}
		label := "Menunggu penilaian"
		switch row.EvalStatus {
		case model.EvalRunning:
			label = "Sedang berjalan"
		case model.EvalTP:
			label = "Take profit"
		case model.EvalSL:
			label = "Stop loss"
		case model.EvalNoEntry:
			label = "Tidak masuk"
		case model.EvalUnknown:
			label = "Tidak dapat dipastikan"
		}
		fmt.Fprintf(&b, "\n%d. %s · %s\n%s → %s · %s\nSimulasi: %s\n", (page-1)*5+i+1, telegram.Escape(sig.Symbol), telegram.Escape(sig.Direction), telegram.Escape(sig.Htf), telegram.Escape(sig.Ltf), telegram.StrategyLabel(sig.StrategyVersion), label)
		if row.EvalEntryAt != nil {
			fmt.Fprintf(&b, "Entry: %s\n", row.EvalEntryAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB"))
		}
		if row.EvalAt != nil {
			fmt.Fprintf(&b, "Penilaian akhir: %s\n", row.EvalAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB"))
		}
		if row.EvalStatus == model.EvalTP || row.EvalStatus == model.EvalSL {
			fmt.Fprintf(&b, "Hasil: %+.2fR\n", row.EvalRLower)
		}
	}
	prev, next := telegram.CallbackNoop, telegram.CallbackNoop
	if page > 1 {
		prev = fmt.Sprintf("%s%d", telegram.CallbackSimulasi, page-1)
	}
	if page < pages {
		next = fmt.Sprintf("%s%d", telegram.CallbackSimulasi, page+1)
	}
	return telegram.Message{Text: b.String(), Buttons: append([][]telegram.InlineButton{
		{{Text: "‹ Sebelumnya", CallbackData: prev}, {Text: "Kembali", CallbackData: telegram.CallbackRiwayatMenu}, {Text: "Berikutnya ›", CallbackData: next}},
		{{Text: "Detail & bukti harga", URL: telegram.AppURL(link, "/posisi"), Style: "primary"}},
	}, details...)}, nil
}

func radarReference(id string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(id)))[:16] }

// Detail resolves identifiers only within the authenticated user's data.
func (s *telegramMenuService) Detail(ctx context.Context, user *model.User, link, kind, id string) (telegram.Message, error) {
	if kind == "radar" && s.radar != nil {
		rows, err := s.radar.ListRadarForUser(ctx, user.ID, 100)
		if err != nil {
			return telegram.Message{}, err
		}
		for _, sig := range rows {
			if radarReference(sig.SignalID) != id {
				continue
			}
			stage, text := tahapRadar(sig.Status, sig.Htf, sig.Ltf)
			body := fmt.Sprintf("<b>%s · %s → %s</b>\n%s\n\nTahap %s · %s\nArah: %s\n\nSinyal pertama: %s\nPembaruan terakhir: %s", telegram.Escape(sig.Symbol), telegram.Escape(sig.Htf), telegram.Escape(sig.Ltf), telegram.StrategyLabel(sig.StrategyVersion), stage, telegram.Escape(text), telegram.Escape(sig.Direction), sig.PublishedAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB"), sig.UpdatedAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB"))
			if sig.EntryPrice > 0 {
				body += fmt.Sprintf("\n\nRencana entry: %g\nStop loss: %g\nTake profit: %g", sig.EntryPrice, sig.StopLoss, sig.TakeProfit)
			}
			return telegram.Message{Text: body, Buttons: [][]telegram.InlineButton{{{Text: "Lihat di dashboard", URL: telegram.AppURL(link, "/sinyal"), Style: "primary"}}, {{Text: "Segarkan", CallbackData: "view:radar:" + id}, {Text: "Kembali ke Pantau", CallbackData: telegram.CallbackPantau}}}}, nil
		}
	}
	if kind == "simulation" {
		rows, err := s.journal.ListSkipped(ctx, user.ID, repository.JournalFilter{SkippedID: id}, 0, 1)
		if err != nil {
			return telegram.Message{}, err
		}
		if len(rows) == 1 {
			row := rows[0]
			var sig model.Signal
			if err := json.Unmarshal([]byte(row.EvalSnapshot), &sig); err != nil {
				return telegram.Message{}, fmt.Errorf("snapshot simulasi tidak tersedia: %w", err)
			}
			body := fmt.Sprintf("<b>%s · %s</b>\n%s\nSimulasi sinyal tidak tereksekusi\n\nStatus: %s\nRencana entry: %g\nStop loss: %g\nTake profit: %g", telegram.Escape(sig.Symbol), telegram.Escape(sig.Direction), telegram.StrategyLabel(sig.StrategyVersion), simulationStatusLabel(row.EvalStatus), sig.EntryPrice, sig.StopLoss, sig.TakeProfit)
			if row.EvalStatus == model.EvalTP || row.EvalStatus == model.EvalSL {
				body += fmt.Sprintf("%sHasil: %+.2fR", string(rune(10)), row.EvalRLower)
			}
			if row.EvalEntryAt != nil {
				body += "\nEntry simulasi: " + row.EvalEntryAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB")
			}
			if row.EvalAt != nil {
				body += "\nPenilaian akhir: " + row.EvalAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB")
			}
			if row.EvalEntryAt != nil && row.EvalAt != nil {
				body += "\nDurasi: " + lamaRingkas(row.EvalAt.Sub(*row.EvalEntryAt))
			}
			if row.RejectReason != "" {
				body += "\n\nTidak masuk akun: " + telegram.Escape(row.RejectReason)
			}
			if row.EvalNote != "" {
				body += "\nCatatan simulasi: " + telegram.Escape(row.EvalNote)
			}
			body += "\n\nHasil simulasi bukan PnL akun. Entry teoritis menggunakan level sinyal, bukan fill broker. Data lama dapat memakai metode penilaian sebelumnya."
			return telegram.Message{Text: body, Buttons: [][]telegram.InlineButton{{{Text: "Lihat bukti harga di web", URL: telegram.AppURL(link, "/posisi"), Style: "primary"}}, {{Text: "Kembali ke simulasi", CallbackData: telegram.CallbackSimulasi + "1"}, {Text: telegram.LabelMenu, CallbackData: telegram.CallbackMenu}}}}, nil
		}
	}
	if kind == "history" {
		rows, err := s.journal.ListPositions(ctx, user.ID, repository.JournalFilter{ClosedOnly: true, PositionID: id}, 1)
		if err != nil {
			return telegram.Message{}, err
		}
		if len(rows) == 1 {
			pos := rows[0]
			closed := "Belum tersedia"
			if pos.ClosedAt != nil {
				closed = pos.ClosedAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB")
			}
			body := fmt.Sprintf("<b>%s · %s</b>\nTransaksi akun\n\nStatus: %s\nEntry: %g\nExit: %g\nStop loss: %g\nTake profit: %g\nHasil: %+.2fR\n\nDibuka: %s\nDitutup: %s", telegram.Escape(pos.Symbol), telegram.Escape(pos.Direction), positionStatusLabel(pos.Status), pos.OpenPrice, pos.ExitPrice, pos.StopLoss, pos.TakeProfit, pos.OutcomeR, pos.OpenedAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB"), closed)
			if pos.ClosedAt != nil {
				body += "\nDurasi: " + lamaRingkas(pos.ClosedAt.Sub(pos.OpenedAt))
			}
			return telegram.Message{Text: body, Buttons: [][]telegram.InlineButton{{{Text: "Kembali ke transaksi", CallbackData: telegram.CallbackRiwayat + "0"}, {Text: "Menu", CallbackData: telegram.CallbackMenu}}}}, nil
		}
	}
	if kind == "position" {
		rows, err := s.akun.ListOpenPositionsByUser(ctx, user.ID)
		if err != nil {
			return telegram.Message{}, err
		}
		for _, pos := range rows {
			if pos.ID != id {
				continue
			}
			body := fmt.Sprintf("<b>%s · %s</b>\nPosisi akun berjalan\n\nOrder: %s\nLot: %g\nEntry: %g\nStop loss: %g\nTake profit: %g\n\nDibuka: %s\nDurasi: %s\n\nPnL berjalan belum tersedia di halaman ini.", telegram.Escape(pos.Symbol), telegram.Escape(pos.Direction), telegram.Escape(pos.TicketNumber), pos.LotSize, pos.OpenPrice, pos.StopLoss, pos.TakeProfit, pos.OpenedAt.In(timezone.WIB).Format("02 Jan 2006, 15:04 WIB"), lamaRingkas(s.timezone().Sub(pos.OpenedAt)))
			return telegram.Message{Text: body, Buttons: [][]telegram.InlineButton{{{Text: "Lihat posisi di web", URL: telegram.AppURL(link, "/posisi"), Style: "primary"}}, {{Text: "Segarkan", CallbackData: "view:position:" + id}, {Text: "Kembali ke Pantau", CallbackData: telegram.CallbackPantau}}}}, nil
		}
	}
	return telegram.Message{Text: "Item ini sudah tidak aktif atau tidak tersedia untuk akun Anda.", Buttons: [][]telegram.InlineButton{{{Text: "Kembali ke Pantau", CallbackData: telegram.CallbackPantau}}}}, nil
}

func simulationStatusLabel(status string) string {
	switch status {
	case model.EvalRunning:
		return "Sedang berjalan"
	case model.EvalTP:
		return "Take profit"
	case model.EvalSL:
		return "Stop loss"
	case model.EvalNoEntry:
		return "Tidak masuk"
	case model.EvalUnknown:
		return "Tidak dapat dipastikan"
	default:
		return "Menunggu penilaian"
	}
}
func positionStatusLabel(status string) string {
	switch status {
	case "CLOSED_TP":
		return "Take profit"
	case "CLOSED_SL":
		return "Stop loss"
	case "CLOSED_MANUAL":
		return "Ditutup manual"
	case "UNTRACKED":
		return "Hasil belum terverifikasi"
	default:
		return "Posisi selesai"
	}
}
