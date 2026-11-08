package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// PairSeedCategory mendeskripsikan satu kategori beserta grup dan pair awalnya.
type PairSeedCategory struct {
	Key       string
	Label     string
	SortOrder int
	Groups    []PairSeedGroup
}

// PairSeedGroup adalah sub-kategori bawaan.
type PairSeedGroup struct {
	Key       string
	Label     string
	SortOrder int
}

// PairSeedPair adalah pair bawaan beserta kategori dan grupnya.
type PairSeedPair struct {
	Symbol      string
	DisplayName string
	CategoryKey string
	GroupKey    string
	Aliases     string
	SortOrder   int
	Description string
}

// DefaultPairSeeds mengembalikan katalog awal yang mencakup simbol yang sudah
// didukung adapter broker (XAUUSD, EURUSD, USTEC) plus beberapa pair umum yang
// bisa langsung diaktifkan admin.
func DefaultPairSeeds() (categories []PairSeedCategory, pairs []PairSeedPair) {
	categories = []PairSeedCategory{
		{
			Key: "forex", Label: "Forex", SortOrder: 10,
			Groups: []PairSeedGroup{
				{Key: "usd", Label: "USD Quote", SortOrder: 10},
				{Key: "jpy", Label: "JPY Quote", SortOrder: 20},
				{Key: "eur", Label: "EUR Quote", SortOrder: 30},
				{Key: "gbp", Label: "GBP Quote", SortOrder: 40},
			},
		},
		{
			Key: "indices", Label: "Indices", SortOrder: 20,
			Groups: []PairSeedGroup{
				{Key: "us", Label: "US", SortOrder: 10},
				{Key: "eu", Label: "Europe", SortOrder: 20},
			},
		},
		{
			Key: "commodity", Label: "Commodity", SortOrder: 30,
			Groups: []PairSeedGroup{
				{Key: "metals", Label: "Metals", SortOrder: 10},
				{Key: "energy", Label: "Energy", SortOrder: 20},
			},
		},
		{
			Key: "crypto", Label: "Crypto", SortOrder: 40,
			Groups: []PairSeedGroup{
				{Key: "major", Label: "Major", SortOrder: 10},
			},
		},
	}

	pairs = []PairSeedPair{
		{Symbol: "EURUSD", DisplayName: "Euro vs US Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "EUR/USD", SortOrder: 10},
		{Symbol: "GBPUSD", DisplayName: "British Pound vs US Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "GBP/USD", SortOrder: 20},
		{Symbol: "AUDUSD", DisplayName: "Australian Dollar vs US Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "AUD/USD", SortOrder: 30},
		{Symbol: "NZDUSD", DisplayName: "New Zealand Dollar vs US Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "NZD/USD", SortOrder: 40},
		{Symbol: "USDCAD", DisplayName: "US Dollar vs Canadian Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "USD/CAD", SortOrder: 50},
		{Symbol: "USDCHF", DisplayName: "US Dollar vs Swiss Franc", CategoryKey: "forex", GroupKey: "usd", Aliases: "USD/CHF", SortOrder: 60},
		{Symbol: "USDJPY", DisplayName: "US Dollar vs Japanese Yen", CategoryKey: "forex", GroupKey: "jpy", Aliases: "USD/JPY", SortOrder: 70},
		{Symbol: "EURJPY", DisplayName: "Euro vs Japanese Yen", CategoryKey: "forex", GroupKey: "jpy", Aliases: "EUR/JPY", SortOrder: 80},
		{Symbol: "GBPJPY", DisplayName: "British Pound vs Japanese Yen", CategoryKey: "forex", GroupKey: "jpy", Aliases: "GBP/JPY", SortOrder: 90},
		{Symbol: "EURGBP", DisplayName: "Euro vs British Pound", CategoryKey: "forex", GroupKey: "gbp", Aliases: "EUR/GBP", SortOrder: 100},

		// USTEC adalah simbol kanonik yang dipakai adapter cTrader (ID 10014).
		// NAS100 dan US100 hanya alias, supaya sinyal nama lain tetap dikenali.
		{Symbol: "USTEC", DisplayName: "US Tech 100", CategoryKey: "indices", GroupKey: "us", Aliases: "NAS100,US100,NDX", SortOrder: 10},
		{Symbol: "US30", DisplayName: "Wall Street 30", CategoryKey: "indices", GroupKey: "us", Aliases: "DOW30,DJ30", SortOrder: 20},
		{Symbol: "US500", DisplayName: "US 500", CategoryKey: "indices", GroupKey: "us", Aliases: "SPX500,SP500", SortOrder: 30},
		{Symbol: "GER40", DisplayName: "Germany 40", CategoryKey: "indices", GroupKey: "eu", Aliases: "DAX40,DE40", SortOrder: 40},

		{Symbol: "XAUUSD", DisplayName: "Gold vs US Dollar", CategoryKey: "commodity", GroupKey: "metals", Aliases: "GOLD,XAU/USD", SortOrder: 10},
		{Symbol: "XAGUSD", DisplayName: "Silver vs US Dollar", CategoryKey: "commodity", GroupKey: "metals", Aliases: "SILVER,XAG/USD", SortOrder: 20},
		{Symbol: "USOIL", DisplayName: "WTI Crude Oil", CategoryKey: "commodity", GroupKey: "energy", Aliases: "WTI,CRUDEOIL", SortOrder: 30},

		{Symbol: "BTCUSD", DisplayName: "Bitcoin vs US Dollar", CategoryKey: "crypto", GroupKey: "major", Aliases: "BTC/USD,XBTUSD", SortOrder: 10},
		{Symbol: "ETHUSD", DisplayName: "Ethereum vs US Dollar", CategoryKey: "crypto", GroupKey: "major", Aliases: "ETH/USD", SortOrder: 20},
	}

	return categories, pairs
}

// SeedPairs mengisi katalog pair bawaan bila belum ada.
//
// Seed bersifat aditif dan idempoten: kategori, grup, atau pair yang sudah ada
// tidak pernah ditimpa, sehingga perubahan admin tidak hilang saat migrasi
// dijalankan ulang. Nilai yang dihitung adalah jumlah pair baru.
func SeedPairs(ctx context.Context, db *gorm.DB, at time.Time) (int, error) {
	categories, pairs := DefaultPairSeeds()

	seeded := 0
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		categoryID := make(map[string]string, len(categories))

		for _, cat := range categories {
			var existing model.PairCategory
			err := tx.Where("LOWER(key) = ?", cat.Key).First(&existing).Error
			if err == nil {
				categoryID[cat.Key] = existing.ID
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				row := &model.PairCategory{
					Key:       cat.Key,
					Label:     cat.Label,
					SortOrder: cat.SortOrder,
					IsEnabled: true,
				}
				row.CreatedAt = at
				row.UpdatedAt = at
				if err := tx.Create(row).Error; err != nil {
					return fmt.Errorf("menyiapkan kategori %s: %w", cat.Key, err)
				}
				categoryID[cat.Key] = row.ID
			} else {
				return fmt.Errorf("membaca kategori %s: %w", cat.Key, err)
			}

			for _, grp := range cat.Groups {
				var existingGroup model.PairGroup
				err := tx.Where("category_id = ? AND LOWER(key) = ?", categoryID[cat.Key], grp.Key).First(&existingGroup).Error
				if err == nil {
					continue
				}
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("membaca grup %s: %w", grp.Key, err)
				}
				group := &model.PairGroup{
					CategoryID: categoryID[cat.Key],
					Key:        grp.Key,
					Label:      grp.Label,
					SortOrder:  grp.SortOrder,
					IsEnabled:  true,
				}
				group.CreatedAt = at
				group.UpdatedAt = at
				if err := tx.Create(group).Error; err != nil {
					return fmt.Errorf("menyiapkan grup %s: %w", grp.Key, err)
				}
			}
		}

		groupID := make(map[string]string)
		var groups []model.PairGroup
		if err := tx.Find(&groups).Error; err != nil {
			return fmt.Errorf("membaca grup pair: %w", err)
		}
		for _, grp := range groups {
			groupID[grp.CategoryID+"|"+grp.Key] = grp.ID
		}

		for _, pair := range pairs {
			catID, ok := categoryID[pair.CategoryKey]
			if !ok {
				continue
			}
			var count int64
			if err := tx.Model(&model.TradingPair{}).Where("symbol = ?", pair.Symbol).Count(&count).Error; err != nil {
				return fmt.Errorf("memeriksa pair %s: %w", pair.Symbol, err)
			}
			if count > 0 {
				continue
			}

			row := &model.TradingPair{
				Symbol:      pair.Symbol,
				DisplayName: pair.DisplayName,
				CategoryID:  catID,
				Aliases:     pair.Aliases,
				SortOrder:   pair.SortOrder,
				IsEnabled:   true,
				Description: pair.Description,
			}
			if id, ok := groupID[catID+"|"+pair.GroupKey]; ok {
				gid := id
				row.GroupID = &gid
			}
			row.CreatedAt = at
			row.UpdatedAt = at
			if err := tx.Create(row).Error; err != nil {
				return fmt.Errorf("menyiapkan pair %s: %w", pair.Symbol, err)
			}
			seeded++
		}

		return nil
	})

	return seeded, err
}
