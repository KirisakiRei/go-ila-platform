package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// BackfillAccountPairStrategy menyalin pilihan metode lama (kolom
// `subscription.strategy`) ke saklar per akun×pair.
//
// Ini tahap pertama penggantian `subscription.strategy`. Kolom lama sengaja
// dibiarkan agar rollback masih mungkin; setelah saklar terbukti stabil kolom
// itu bisa dihapus pada migrasi berikutnya.
//
// Aturan penyalinan:
//   - Langganan aktif tanpa `broker_account_id` berlaku untuk semua akun aktif
//     milik user, sama seperti pemilihan akun di Runner.
//   - Langganan aktif dengan `broker_account_id` hanya untuk akun itu.
//   - ILA 2.0 selalu disalin menyala; ILA 3.0 hanya bila langganannya memilih
//     metode itu.
//   - Baris yang sudah ada tidak pernah ditimpa, sehingga penyesuaian manual
//     user tidak hilang saat migrasi dijalankan ulang.
//
// Fungsi idempoten dan mengembalikan jumlah baris saklar yang baru dibuat.
func BackfillAccountPairStrategy(ctx context.Context, db *gorm.DB) (int64, error) {
	// Setelah tahap 2 dijalankan, kolom sumbernya sudah tidak ada. Tanpa
	// penjagaan ini, migrasi berikutnya akan gagal sebelum sempat berjalan.
	ada, err := columnExists(ctx, db, "subscription", "strategy")
	if err != nil {
		return 0, err
	}
	if !ada {
		return 0, nil
	}

	type subRow struct {
		UserID          string
		Symbol          string
		Strategy        string
		BrokerAccountID *string
	}

	var subs []subRow
	err = db.WithContext(ctx).
		Table("subscription").
		Select("user_id, symbol, strategy, broker_account_id").
		Where("is_active = ? AND deleted_at IS NULL", true).
		Scan(&subs).Error
	if err != nil {
		return 0, fmt.Errorf("membaca langganan lama: %w", err)
	}
	if len(subs) == 0 {
		return 0, nil
	}

	// Kelompokkan akun aktif per user sekali saja agar tidak menanyakan
	// basis data untuk setiap langganan.
	userIDs := make([]string, 0, len(subs))
	seenUser := make(map[string]bool, len(subs))
	for _, sub := range subs {
		if !seenUser[sub.UserID] {
			seenUser[sub.UserID] = true
			userIDs = append(userIDs, sub.UserID)
		}
	}

	var accounts []model.BrokerAccount
	if err := db.WithContext(ctx).
		Select("id, user_id, is_active").
		Where("user_id IN ? AND is_active = ? AND deleted_at IS NULL", userIDs, true).
		Find(&accounts).Error; err != nil {
		return 0, fmt.Errorf("membaca akun aktif: %w", err)
	}

	accountByUser := make(map[string][]string, len(accounts))
	accountKnown := make(map[string]bool, len(accounts))
	for _, account := range accounts {
		accountByUser[account.UserID] = append(accountByUser[account.UserID], account.ID)
		accountKnown[account.ID] = true
	}

	// Saklar yang sudah ada dipetakan agar tidak ditimpa.
	var existing []model.AccountPairStrategy
	if err := db.WithContext(ctx).
		Select("broker_account_id, symbol, strategy").
		Find(&existing).Error; err != nil {
		return 0, fmt.Errorf("membaca saklar metode lama: %w", err)
	}
	already := make(map[string]bool, len(existing))
	for _, row := range existing {
		already[row.BrokerAccountID+"|"+row.Symbol+"|"+row.Strategy] = true
	}

	buat := func(accountID, userID, symbol, strategy string) bool {
		key := accountID + "|" + symbol + "|" + strategy
		if already[key] {
			return false
		}
		already[key] = true
		return true
	}

	rows := make([]model.AccountPairStrategy, 0, len(subs)*2)
	for _, sub := range subs {
		symbol := normalizeSymbolForBackfill(sub.Symbol)
		if symbol == "" {
			continue
		}

		var targets []string
		if sub.BrokerAccountID != nil && *sub.BrokerAccountID != "" {
			// Langganan per akun: hanya bila akun itu masih aktif.
			if accountKnown[*sub.BrokerAccountID] {
				targets = []string{*sub.BrokerAccountID}
			}
		} else {
			targets = accountByUser[sub.UserID]
		}

		for _, accountID := range targets {
			// ILA 2.0 selalu menyala: metode dasar yang dipantau semua akun.
			if buat(accountID, sub.UserID, symbol, "ila-2.0") {
				rows = append(rows, model.AccountPairStrategy{
					BrokerAccountID: accountID,
					UserID:          sub.UserID,
					Symbol:          symbol,
					Strategy:        "ila-2.0",
					IsEnabled:       true,
				})
			}
			if sub.Strategy == "ila-3.0" && buat(accountID, sub.UserID, symbol, "ila-3.0") {
				rows = append(rows, model.AccountPairStrategy{
					BrokerAccountID: accountID,
					UserID:          sub.UserID,
					Symbol:          symbol,
					Strategy:        "ila-3.0",
					IsEnabled:       true,
				})
			}
		}
	}

	if len(rows) == 0 {
		return 0, nil
	}

	if err := db.WithContext(ctx).CreateInBatches(&rows, 200).Error; err != nil {
		return 0, fmt.Errorf("menulis saklar metode hasil salin: %w", err)
	}
	return int64(len(rows)), nil
}

// normalizeSymbolForBackfill menyamakan penulisan simbol lama agar tidak
// menghasilkan dua baris saklar untuk pair yang sama.
func normalizeSymbolForBackfill(symbol string) string {
	out := make([]rune, 0, len(symbol))
	for _, r := range symbol {
		switch {
		case r >= 'a' && r <= 'z':
			out = append(out, r-('a'-'A'))
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-':
			out = append(out, r)
		}
	}
	return string(out)
}
