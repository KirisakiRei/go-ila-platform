package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// JournalFilter membatasi baris jurnal yang dibaca.
//
// Semua kolom opsional: nilai nol berarti tidak menyaring. Filter dipakai
// bersama oleh daftar dan ringkasan supaya keduanya selalu berbicara tentang
// kumpulan data yang sama.
type JournalFilter struct {
	// From dan To membatasi waktu buka posisi. To eksklusif agar rentang
	// hari tidak tumpang tindih di batas tengah malam.
	From time.Time
	To   time.Time
	// StrategyVersion membatasi metode analisis, mis. "ila-2.0".
	StrategyVersion string
	// Symbol membatasi satu simbol.
	Symbol string
	// Status membatasi status posisi, mis. hanya CLOSED_TP.
	Status string
}

// JournalStats adalah ringkasan angka jurnal eksekusi.
//
// Angka dihitung dari posisi yang sudah ditutup saja untuk winrate dan total R:
// memasukkan posisi yang masih berjalan akan membuat hasilnya berubah setiap
// kali halaman dimuat, sehingga tidak bisa dipakai menilai strategi.
type JournalStats struct {
	// TotalPositions menghitung seluruh posisi pada rentang, termasuk yang
	// masih berjalan.
	TotalPositions int
	// OpenPositions adalah posisi yang masih hidup di broker.
	OpenPositions int
	// ClosedTP, ClosedSL, ClosedManual, Untracked memecah hasil penutupan.
	ClosedTP     int
	ClosedSL     int
	ClosedManual int
	Untracked    int
	// Wins dan Losses hanya menghitung penutupan yang hasilnya diketahui.
	Wins   int
	Losses int
	// TotalR adalah jumlah outcome_r dari posisi tutup yang hasilnya diketahui.
	TotalR float64
	// TotalProfit adalah jumlah profit_loss dari posisi tutup.
	TotalProfit float64
	// DryRunPositions menghitung posisi simulasi pada rentang ini.
	DryRunPositions int
}

// JournalRepository membaca jurnal eksekusi user.
type JournalRepository interface {
	// ListPositions membaca posisi sesuai filter, terbaru lebih dulu.
	ListPositions(ctx context.Context, userID string, filter JournalFilter, limit int) ([]model.Position, error)
	// Stats menghitung ringkasan jurnal untuk filter yang sama.
	Stats(ctx context.Context, userID string, filter JournalFilter) (JournalStats, error)
	// ListSkipped membaca satu halaman sinyal yang tidak berakhir sebagai
	// posisi: ditolak, dilewati, kedaluwarsa, atau masih menunggu keputusan user.
	// offset dan limit dipakai agar daftar bisa ditelusuri halaman demi halaman.
	ListSkipped(ctx context.Context, userID string, filter JournalFilter, offset, limit int) ([]model.UserSignal, error)
	// CountSkipped menghitung seluruh baris yang cocok, tanpa batas halaman.
	// Diperlukan agar antarmuka tahu berapa halaman yang ada.
	CountSkipped(ctx context.Context, userID string, filter JournalFilter) (int, error)
}

type journalRepository struct {
	db *gorm.DB
}

// NewJournalRepository membuat pembaca jurnal eksekusi.
func NewJournalRepository(db *gorm.DB) JournalRepository {
	return &journalRepository{db: db}
}

// applyFilter memasang penyaring bersama pada kueri posisi.
func (r *journalRepository) applyFilter(query *gorm.DB, userID string, filter JournalFilter) *gorm.DB {
	query = query.Where("user_id = ?", userID)

	if !filter.From.IsZero() {
		query = query.Where("opened_at >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		query = query.Where("opened_at < ?", filter.To)
	}
	if filter.StrategyVersion != "" {
		query = query.Where("strategy_version = ?", filter.StrategyVersion)
	}
	if filter.Symbol != "" {
		query = query.Where("symbol = ?", filter.Symbol)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	return query
}

// ListPositions mengembalikan posisi jurnal sesuai filter.
func (r *journalRepository) ListPositions(ctx context.Context, userID string, filter JournalFilter, limit int) ([]model.Position, error) {
	var rows []model.Position

	query := r.applyFilter(r.db.WithContext(ctx).Model(&model.Position{}), userID, filter)
	if err := query.Order("opened_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca jurnal posisi: %w", err)
	}
	return rows, nil
}

// Stats menghitung ringkasan jurnal.
//
// Penghitungan dilakukan di basis data agar tetap ringan walau riwayat sudah
// panjang; menarik seluruh baris ke aplikasi hanya untuk menghitung akan
// membebani memori seiring bertambahnya riwayat.
func (r *journalRepository) Stats(ctx context.Context, userID string, filter JournalFilter) (JournalStats, error) {
	var stats JournalStats

	query := r.applyFilter(r.db.WithContext(ctx).Model(&model.Position{}), userID, filter)

	// Hitungan per status dalam satu kueri.
	type statusCount struct {
		Status string
		Jumlah int
	}
	var counts []statusCount
	if err := query.Select("status, COUNT(*) AS jumlah").Group("status").Scan(&counts).Error; err != nil {
		return stats, fmt.Errorf("menghitung status jurnal: %w", err)
	}
	for _, row := range counts {
		stats.TotalPositions += row.Jumlah
		switch row.Status {
		case model.PositionStatusOpen:
			stats.OpenPositions = row.Jumlah
		case model.PositionStatusClosedTP:
			stats.ClosedTP = row.Jumlah
		case model.PositionStatusClosedSL:
			stats.ClosedSL = row.Jumlah
		case model.PositionStatusClosedManual:
			stats.ClosedManual = row.Jumlah
		case model.PositionStatusUntracked:
			stats.Untracked = row.Jumlah
		}
	}

	// Agregat hasil. Posisi yang masih terbuka dikecualikan: outcome_r-nya nol
	// dan akan menyeret rata-rata ke bawah seolah selalu impas.
	type outcomeSum struct {
		TotalR      float64
		TotalProfit float64
		Wins        int
		Losses      int
	}
	var outcome outcomeSum
	err := r.applyFilter(r.db.WithContext(ctx).Model(&model.Position{}), userID, filter).
		Where("status <> ?", model.PositionStatusOpen).
		Select(`COALESCE(SUM(outcome_r), 0) AS total_r,
			COALESCE(SUM(profit_loss), 0) AS total_profit,
			COALESCE(SUM(CASE WHEN profit_loss > 0 THEN 1 ELSE 0 END), 0) AS wins,
			COALESCE(SUM(CASE WHEN profit_loss < 0 THEN 1 ELSE 0 END), 0) AS losses`).
		Scan(&outcome).Error
	if err != nil {
		return stats, fmt.Errorf("menghitung hasil jurnal: %w", err)
	}
	stats.TotalR = outcome.TotalR
	stats.TotalProfit = outcome.TotalProfit
	stats.Wins = outcome.Wins
	stats.Losses = outcome.Losses

	var dryRun int64
	err = r.applyFilter(r.db.WithContext(ctx).Model(&model.Position{}), userID, filter).
		Where("is_dry_run = ?", true).
		Count(&dryRun).Error
	if err != nil {
		return stats, fmt.Errorf("menghitung posisi simulasi: %w", err)
	}
	stats.DryRunPositions = int(dryRun)

	return stats, nil
}

// skipStatuses adalah status user_signal yang bukan posisi.
//
// CONSUMED sengaja tidak termasuk: sinyal itu sudah menjadi posisi, sehingga
// menampilkannya lagi di daftar "tidak tereksekusi" akan menghitung dua kali.
var skipStatuses = []string{
	model.SignalStatusRejected,
	model.SignalStatusSkipped,
	model.SignalStatusExpired,
	model.SignalStatusOpportunity,
	model.SignalStatusReadyToEntry,
}

// ListSkipped membaca sinyal yang tidak berakhir sebagai posisi.
//
// Tujuannya memberi gambaran peluang yang terlewat, bukan hanya trade yang
// terjadi. Tanpa ini, jurnal hanya berisi kemenangan dan kekalahan dan
// menyembunyikan berapa banyak setup yang tidak pernah dieksekusi.
func (r *journalRepository) ListSkipped(ctx context.Context, userID string, filter JournalFilter, offset, limit int) ([]model.UserSignal, error) {
	var rows []model.UserSignal

	query := r.skippedQuery(ctx, userID, filter)
	if offset > 0 {
		query = query.Offset(offset)
	}

	// Kolom user_signal dipilih eksplisit agar join tidak menimpa kolom yang
	// bernama sama (id, status, created_at) saat hasil dipindai ke struct.
	if err := query.
		Select("user_signal.*").
		Order("user_signal.created_at DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca sinyal tidak tereksekusi: %w", err)
	}
	return rows, nil
}

// CountSkipped menghitung seluruh peluang terlewat pada filter yang sama.
func (r *journalRepository) CountSkipped(ctx context.Context, userID string, filter JournalFilter) (int, error) {
	var total int64
	// Distinct pada kunci primer: query memakai join ke signal, dan hitungan
	// harus tetap satu baris per user_signal walau join-nya menggandakan.
	if err := r.skippedQuery(ctx, userID, filter).Distinct("user_signal.id").Count(&total).Error; err != nil {
		return 0, fmt.Errorf("menghitung sinyal tidak tereksekusi: %w", err)
	}
	return int(total), nil
}

// skippedQuery menyusun kueri bersama untuk peluang terlewat.
//
// Filter metode dan pair ikut diterapkan di sini. Sebelumnya keduanya diabaikan,
// sehingga memilih "ILA 3.0" atau satu pair tetap menampilkan seluruh peluang
// dari metode dan pair lain - angka tab tidak lagi cocok dengan isinya.
func (r *journalRepository) skippedQuery(ctx context.Context, userID string, filter JournalFilter) *gorm.DB {
	query := r.db.WithContext(ctx).
		Model(&model.UserSignal{}).
		Where("user_signal.user_id = ?", userID).
		Where("user_signal.status IN ?", skipStatuses).
		Where("user_signal.position_id = ?", "")

	if !filter.From.IsZero() {
		query = query.Where("user_signal.created_at >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		query = query.Where("user_signal.created_at < ?", filter.To)
	}

	// StrategyVersion dan Symbol berada di tabel signal, bukan user_signal:
	// user_signal hanya mencatat keputusan per akun. Kolomnya harus dirujuk
	// lewat join, bukan langsung — merujuk kolom yang tidak ada akan gagal
	// sebagai galat basis data, bukan sekadar filter diabaikan.
	if filter.StrategyVersion != "" || filter.Symbol != "" {
		query = query.Joins("JOIN signal ON signal.signal_id = user_signal.signal_id")
	}
	if filter.StrategyVersion != "" {
		query = query.Where("signal.strategy_version = ?", filter.StrategyVersion)
	}
	if filter.Symbol != "" {
		query = query.Where("signal.symbol = ?", filter.Symbol)
	}
	return query
}
