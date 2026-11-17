package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
)

// ErrKuotaHabis dikembalikan bila jatah backtest harian pengguna sudah penuh.
var ErrKuotaHabis = errors.New("kuota backtest harian habis")

// BacktestRepository menyimpan dan membaca data backtest.
type BacktestRepository struct {
	db *gorm.DB
}

// NewBacktestRepository menyiapkan repository.
func NewBacktestRepository(db *gorm.DB) *BacktestRepository {
	return &BacktestRepository{db: db}
}

// Buat menyimpan satu run baru.
func (r *BacktestRepository) Buat(ctx context.Context, run *model.BacktestRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

// Ambil mengambil satu run berdasarkan ID tanpa memandang pemiliknya.
//
// Pemeriksaan kepemilikan dilakukan pemanggil, bukan di sini: repository tidak
// tahu siapa yang sedang meminta. Endpoint wajib membandingkan run.UserID
// dengan sesi sebelum mengirim apa pun ke klien.
func (r *BacktestRepository) Ambil(ctx context.Context, id string) (*model.BacktestRun, error) {
	var run model.BacktestRun
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}

// AmbilMilik mengambil satu run bila memang milik pengguna tersebut.
func (r *BacktestRepository) AmbilMilik(ctx context.Context, id, userID string) (*model.BacktestRun, error) {
	var run model.BacktestRun
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}

// Daftar mengambil riwayat run milik satu pengguna, terbaru lebih dulu.
func (r *BacktestRepository) Daftar(ctx context.Context, userID string, limit, offset int) ([]model.BacktestRun, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.BacktestRun{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var daftar []model.BacktestRun
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&daftar).Error
	return daftar, total, err
}

// PerbaruiStatus mengubah status run beserta pesan galatnya.
func (r *BacktestRepository) PerbaruiStatus(ctx context.Context, id, status, pesanGalat string) error {
	perubahan := map[string]any{
		"status":    status,
		"error_msg": pesanGalat,
	}
	if status == model.BacktestStatusRunning {
		perubahan["started_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&model.BacktestRun{}).Where("id = ?", id).Updates(perubahan).Error
}

// SimpanHasil menyimpan statistik lengkap dan seluruh trade dalam satu transaksi.
//
// Transaksi dipakai agar run tidak pernah terlihat "selesai" dengan daftar trade
// yang setengah tersimpan. Bila penyimpanan trade gagal, status run ikut batal.
func (r *BacktestRepository) SimpanHasil(ctx context.Context, run *model.BacktestRun, trades []model.BacktestTrade) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.BacktestRun{}).Where("id = ?", run.ID).Updates(map[string]any{
			"status":            run.Status,
			"range_from":        run.RangeFrom,
			"range_to":          run.RangeTo,
			"total_trades":      run.TotalTrades,
			"wins":              run.Wins,
			"losses":            run.Losses,
			"expired":           run.Expired,
			"ambiguous_exits":   run.AmbiguousExits,
			"signals_rejected":  run.SignalsRejected,
			"win_rate":          run.WinRate,
			"total_r":           run.TotalR,
			"max_drawdown":      run.MaxDrawdown,
			"profit_factor":     run.ProfitFactor,
			"equity_curve_json": run.EquityCurveJSON,
			"finished_at":       run.FinishedAt,
			"error_msg":         run.ErrorMsg,
		}).Error; err != nil {
			return err
		}

		if len(trades) == 0 {
			return nil
		}
		return tx.CreateInBatches(trades, 100).Error
	})
}

// TradeDaftar mengambil daftar trade sebuah run, urut menurut Seq.
func (r *BacktestRepository) TradeDaftar(ctx context.Context, runID string, limit, offset int) ([]model.BacktestTrade, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.BacktestTrade{}).Where("run_id = ?", runID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var daftar []model.BacktestTrade
	err := query.Order("seq ASC").Limit(limit).Offset(offset).Find(&daftar).Error
	return daftar, total, err
}

// TradeAmbil mengambil satu trade berdasarkan nomor urutnya.
func (r *BacktestRepository) TradeAmbil(ctx context.Context, runID string, seq int) (*model.BacktestTrade, error) {
	var trade model.BacktestTrade
	err := r.db.WithContext(ctx).Where("run_id = ? AND seq = ?", runID, seq).First(&trade).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &trade, nil
}

// Hapus menghapus run beserta seluruh trade-nya.
func (r *BacktestRepository) Hapus(ctx context.Context, id, userID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("run_id = ?", id).Delete(&model.BacktestTrade{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.BacktestRun{}).Error
	})
}

// KlaimKuota memakai satu jatah harian secara atomik.
//
// Memakai UPDATE bersyarat, bukan baca-lalu-tulis. Dengan cara lama, dua
// permintaan yang datang bersamaan sama-sama membaca sisa satu lalu keduanya
// berhasil, sehingga batasnya bisa ditembus. Di sini database yang memutuskan.
func (r *BacktestRepository) KlaimKuota(ctx context.Context, userID, tanggal string, batas int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Pastikan baris ada lebih dulu. ON CONFLICT DO NOTHING membuat dua
		// permintaan bersamaan tidak saling menjatuhkan.
		awal := &model.BacktestQuota{UserID: userID, Tanggal: tanggal}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(awal).Error; err != nil {
			return err
		}

		hasil := tx.Model(&model.BacktestQuota{}).
			Where("user_id = ? AND tanggal = ? AND terpakai < ?", userID, tanggal, batas).
			Update("terpakai", gorm.Expr("terpakai + 1"))
		if hasil.Error != nil {
			return hasil.Error
		}
		if hasil.RowsAffected == 0 {
			return ErrKuotaHabis
		}
		return nil
	})
}

// SisaKuota mengembalikan sisa jatah hari ini.
func (r *BacktestRepository) SisaKuota(ctx context.Context, userID, tanggal string, batas int) (int, error) {
	var baris model.BacktestQuota
	err := r.db.WithContext(ctx).Where("user_id = ? AND tanggal = ?", userID, tanggal).First(&baris).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return batas, nil
		}
		return 0, err
	}
	sisa := batas - baris.Terpakai
	if sisa < 0 {
		sisa = 0
	}
	return sisa, nil
}

// KembalikanKuota mengembalikan satu jatah yang sudah diklaim.
//
// Dipakai saat pembuatan job gagal setelah kuota terlanjur dipotong, supaya
// pengguna tidak kehilangan jatah karena kesalahan sistem.
func (r *BacktestRepository) KembalikanKuota(ctx context.Context, userID, tanggal string) error {
	return r.db.WithContext(ctx).Model(&model.BacktestQuota{}).
		Where("user_id = ? AND tanggal = ? AND terpakai > 0", userID, tanggal).
		Update("terpakai", gorm.Expr("terpakai - 1")).Error
}

// AmbilAntrean mengambil run yang menunggu dikerjakan, terlama lebih dulu.
func (r *BacktestRepository) AmbilAntrean(ctx context.Context, limit int) ([]model.BacktestRun, error) {
	var daftar []model.BacktestRun
	err := r.db.WithContext(ctx).
		Where("status = ?", model.BacktestStatusQueued).
		Order("created_at ASC").
		Limit(limit).
		Find(&daftar).Error
	return daftar, err
}

// PulihkanRunMenggantung menandai run yang tertinggal berstatus "running" akibat
// restart/OOM proses sebelumnya menjadi "failed", supaya pengguna tidak terkunci
// oleh batas satu job aktif per akun.
func (r *BacktestRepository) PulihkanRunMenggantung(ctx context.Context, pesan string) (int64, error) {
	sekarang := time.Now().UTC()
	hasil := r.db.WithContext(ctx).Model(&model.BacktestRun{}).
		Where("status = ?", model.BacktestStatusRunning).
		Updates(map[string]any{
			"status":      model.BacktestStatusFailed,
			"error_msg":   pesan,
			"finished_at": sekarang,
			"updated_at":  sekarang,
		})
	return hasil.RowsAffected, hasil.Error
}

// JumlahBerjalan menghitung run yang sedang dikerjakan seorang pengguna.
func (r *BacktestRepository) JumlahBerjalan(ctx context.Context, userID string) (int64, error) {
	var jumlah int64
	err := r.db.WithContext(ctx).Model(&model.BacktestRun{}).
		Where("user_id = ? AND status IN ?", userID,
			[]string{model.BacktestStatusQueued, model.BacktestStatusRunning}).
		Count(&jumlah).Error
	return jumlah, err
}

// BersihkanCandleLama mengosongkan potongan candle pada trade yang lebih tua
// dari batas, menggantikannya dengan penanda waktu.
func (r *BacktestRepository) BersihkanCandleLama(ctx context.Context, batas time.Time) (int64, error) {
	hasil := r.db.WithContext(ctx).Model(&model.BacktestTrade{}).
		Where("candles_purged_at IS NULL AND created_at < ?", batas).
		Updates(map[string]any{
			"candles_json":      "",
			"levels_json":       "",
			"candles_purged_at": time.Now(),
		})
	return hasil.RowsAffected, hasil.Error
}

// HapusRunGagal menghapus run berstatus gagal yang lebih tua dari batas.
func (r *BacktestRepository) HapusRunGagal(ctx context.Context, batas time.Time) (int64, error) {
	var jumlah int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var daftar []model.BacktestRun
		if err := tx.Where("status = ? AND created_at < ?", model.BacktestStatusFailed, batas).
			Limit(500).Find(&daftar).Error; err != nil {
			return err
		}
		for _, run := range daftar {
			if err := tx.Where("run_id = ?", run.ID).Delete(&model.BacktestTrade{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", run.ID).Delete(&model.BacktestRun{}).Error; err != nil {
				return err
			}
			jumlah++
		}
		return nil
	})
	return jumlah, err
}

// HapusRunLama menghapus run selesai yang lebih tua dari batas, beserta trade-nya.
func (r *BacktestRepository) HapusRunLama(ctx context.Context, batas time.Time) (int64, error) {
	var jumlah int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var daftar []model.BacktestRun
		if err := tx.Where("status = ? AND created_at < ?", model.BacktestStatusDone, batas).
			Limit(500).Find(&daftar).Error; err != nil {
			return err
		}
		for _, run := range daftar {
			if err := tx.Where("run_id = ?", run.ID).Delete(&model.BacktestTrade{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id = ?", run.ID).Delete(&model.BacktestRun{}).Error; err != nil {
				return err
			}
			jumlah++
		}
		return nil
	})
	return jumlah, err
}

// HapusKuotaLama menghapus baris kuota yang lebih tua dari batas.
func (r *BacktestRepository) HapusKuotaLama(ctx context.Context, batas time.Time) (int64, error) {
	kunci := batas.Format("2006-01-02")
	hasil := r.db.WithContext(ctx).Where("tanggal < ?", kunci).Delete(&model.BacktestQuota{})
	return hasil.RowsAffected, hasil.Error
}
