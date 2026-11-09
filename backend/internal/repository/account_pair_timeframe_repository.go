package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
)

// TimeframeSwitch adalah satu saklar yang dikirim user untuk sebuah pasangan.
type TimeframeSwitch struct {
	Strategy string
	Htf      string
	Ltf      string
	Watch    bool
	Auto     bool
}

// AccountPairTimeframeRepository menyimpan pengaturan pantau dan auto-eksekusi
// per pasangan timeframe.
type AccountPairTimeframeRepository interface {
	// ListByUser mengambil seluruh saklar milik satu pengguna.
	ListByUser(ctx context.Context, userID string) ([]model.AccountPairTimeframe, error)
	// ListByAccount mengambil saklar satu akun.
	ListByAccount(ctx context.Context, accountID string) ([]model.AccountPairTimeframe, error)
	// Replace menimpa seluruh saklar satu akun untuk satu pair dan metode.
	Replace(ctx context.Context, accountID, userID, symbol, strategy string, rows []TimeframeSwitch) error
	// Ensure membuat saklar bawaan tanpa mengubah yang sudah ada.
	Ensure(ctx context.Context, rows []model.AccountPairTimeframe) error
	// WatchedPairs mengembalikan pasangan timeframe yang dipantau untuk satu
	// simbol, dipakai Runner untuk menyusun kebutuhan engine.
	WatchedPairs(ctx context.Context, symbol string) (map[string]map[PairKey]bool, error)
}

// PairKey adalah pasangan timeframe sebagai kunci peta.
type PairKey struct {
	Htf string
	Ltf string
}

type accountPairTimeframeRepository struct {
	db *gorm.DB
}

// NewAccountPairTimeframeRepository merangkai penyimpanan saklar timeframe.
func NewAccountPairTimeframeRepository(db *gorm.DB) AccountPairTimeframeRepository {
	return &accountPairTimeframeRepository{db: db}
}

func (r *accountPairTimeframeRepository) ListByUser(ctx context.Context, userID string) ([]model.AccountPairTimeframe, error) {
	var rows []model.AccountPairTimeframe
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca saklar timeframe: %w", err)
	}
	return rows, nil
}

func (r *accountPairTimeframeRepository) ListByAccount(ctx context.Context, accountID string) ([]model.AccountPairTimeframe, error) {
	var rows []model.AccountPairTimeframe
	if err := r.db.WithContext(ctx).
		Where("broker_account_id = ?", accountID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca saklar timeframe akun: %w", err)
	}
	return rows, nil
}

// Replace menulis ulang saklar satu akun untuk satu pair dan metode.
//
// Baris yang tidak dikirim tidak dihapus: pasangan timeframe yang tidak lagi
// dikirim tetap tersimpan sebagai nonaktif, sehingga riwayat pilihan user tidak
// hilang dan pengaturan bisa dipulihkan.
func (r *accountPairTimeframeRepository) Replace(ctx context.Context, accountID, userID, symbol, strategy string, rows []TimeframeSwitch) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			record := &model.AccountPairTimeframe{
				BrokerAccountID: accountID,
				UserID:          userID,
				Symbol:          symbol,
				Strategy:        strategy,
				Htf:             row.Htf,
				Ltf:             row.Ltf,
				IsWatch:         row.Watch,
				IsAuto:          row.Auto,
			}
			err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "broker_account_id"}, {Name: "symbol"}, {Name: "strategy"},
					{Name: "htf"}, {Name: "ltf"},
				},
				DoUpdates: clause.AssignmentColumns([]string{"is_watch", "is_auto", "updated_at"}),
			}).Create(record).Error
			if err != nil {
				return fmt.Errorf("menyimpan saklar timeframe %s/%s: %w", row.Htf, row.Ltf, err)
			}
		}
		return nil
	})
}

// Ensure membuat baris yang belum ada tanpa mengubah yang sudah ada.
func (r *accountPairTimeframeRepository) Ensure(ctx context.Context, rows []model.AccountPairTimeframe) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

// WatchedPairs mengembalikan pasangan timeframe yang dipantau untuk satu simbol.
//
// Hasilnya akun -> metode -> pasangan. Hanya baris yang dipantau yang masuk,
// karena hanya pasangan itu yang perlu dihitung engine.
func (r *accountPairTimeframeRepository) WatchedPairs(ctx context.Context, symbol string) (map[string]map[PairKey]bool, error) {
	var rows []model.AccountPairTimeframe
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND is_watch = ? AND deleted_at IS NULL", symbol, true).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca pasangan timeframe dipantau: %w", err)
	}

	out := make(map[string]map[PairKey]bool, len(rows))
	for _, row := range rows {
		if out[row.BrokerAccountID] == nil {
			out[row.BrokerAccountID] = make(map[PairKey]bool, 2)
		}
		out[row.BrokerAccountID][PairKey{Htf: row.Htf, Ltf: row.Ltf}] = true
	}
	return out, nil
}
