package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	// CorrelatedSymbols adalah simbol pembanding SMT bawaan, dipisah koma.
	// Hanya data awal; admin bebas mengubahnya lewat panel.
	CorrelatedSymbols string
	SortOrder         int
	Description       string
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
		// Korelasi awal mengikuti pasangan yang lazim dipakai evaluasi SMT.
		// Arah korelasi tidak harus simetris; admin bisa mengubahnya kapan saja.
		{Symbol: "EURUSD", DisplayName: "Euro vs US Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "EUR/USD", CorrelatedSymbols: "GBPUSD", SortOrder: 10},
		{Symbol: "GBPUSD", DisplayName: "British Pound vs US Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "GBP/USD", CorrelatedSymbols: "EURUSD", SortOrder: 20},
		{Symbol: "AUDUSD", DisplayName: "Australian Dollar vs US Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "AUD/USD", SortOrder: 30},
		{Symbol: "NZDUSD", DisplayName: "New Zealand Dollar vs US Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "NZD/USD", SortOrder: 40},
		{Symbol: "USDCAD", DisplayName: "US Dollar vs Canadian Dollar", CategoryKey: "forex", GroupKey: "usd", Aliases: "USD/CAD", SortOrder: 50},
		{Symbol: "USDCHF", DisplayName: "US Dollar vs Swiss Franc", CategoryKey: "forex", GroupKey: "usd", Aliases: "USD/CHF", SortOrder: 60},
		{Symbol: "USDJPY", DisplayName: "US Dollar vs Japanese Yen", CategoryKey: "forex", GroupKey: "jpy", Aliases: "USD/JPY", CorrelatedSymbols: "EURJPY,GBPJPY", SortOrder: 70},
		{Symbol: "EURJPY", DisplayName: "Euro vs Japanese Yen", CategoryKey: "forex", GroupKey: "jpy", Aliases: "EUR/JPY", CorrelatedSymbols: "GBPJPY,USDJPY", SortOrder: 80},
		{Symbol: "GBPJPY", DisplayName: "British Pound vs Japanese Yen", CategoryKey: "forex", GroupKey: "jpy", Aliases: "GBP/JPY", CorrelatedSymbols: "EURJPY,USDJPY", SortOrder: 90},
		{Symbol: "EURGBP", DisplayName: "Euro vs British Pound", CategoryKey: "forex", GroupKey: "gbp", Aliases: "EUR/GBP", CorrelatedSymbols: "EURUSD,GBPUSD", SortOrder: 100},

		// USTEC adalah simbol kanonik yang dipakai adapter cTrader (ID 10014).
		// NAS100 dan US100 hanya alias, supaya sinyal nama lain tetap dikenali.
		{Symbol: "USTEC", DisplayName: "US Tech 100", CategoryKey: "indices", GroupKey: "us", Aliases: "NAS100,US100,NDX", CorrelatedSymbols: "US30,US500", SortOrder: 10},
		{Symbol: "US30", DisplayName: "Wall Street 30", CategoryKey: "indices", GroupKey: "us", Aliases: "DOW30,DJ30", CorrelatedSymbols: "US500,USTEC", SortOrder: 20},
		{Symbol: "US500", DisplayName: "US 500", CategoryKey: "indices", GroupKey: "us", Aliases: "SPX500,SP500", CorrelatedSymbols: "US30,USTEC", SortOrder: 30},
		{Symbol: "GER40", DisplayName: "Germany 40", CategoryKey: "indices", GroupKey: "eu", Aliases: "DAX40,DE40", SortOrder: 40},

		{Symbol: "XAUUSD", DisplayName: "Gold vs US Dollar", CategoryKey: "commodity", GroupKey: "metals", Aliases: "GOLD,XAU/USD", CorrelatedSymbols: "XAGUSD,XAUEUR", SortOrder: 10},
		{Symbol: "XAGUSD", DisplayName: "Silver vs US Dollar", CategoryKey: "commodity", GroupKey: "metals", Aliases: "SILVER,XAG/USD", CorrelatedSymbols: "XAUUSD", SortOrder: 20},
		{Symbol: "XAUEUR", DisplayName: "Gold vs Euro", CategoryKey: "commodity", GroupKey: "metals", Aliases: "XAU/EUR", CorrelatedSymbols: "XAUUSD", SortOrder: 25},
		{Symbol: "USOIL", DisplayName: "WTI Crude Oil", CategoryKey: "commodity", GroupKey: "energy", Aliases: "WTI,CRUDEOIL", SortOrder: 30},

		{Symbol: "BTCUSD", DisplayName: "Bitcoin vs US Dollar", CategoryKey: "crypto", GroupKey: "major", Aliases: "BTC/USD,XBTUSD", CorrelatedSymbols: "ETHUSD", SortOrder: 10},
		{Symbol: "ETHUSD", DisplayName: "Ethereum vs US Dollar", CategoryKey: "crypto", GroupKey: "major", Aliases: "ETH/USD", CorrelatedSymbols: "BTCUSD", SortOrder: 20},
	}

	return categories, pairs
}

// SeedPairs mengisi katalog pair bawaan bila belum ada.
//
// Seed bersifat aditif dan idempoten: kategori, grup, atau pair yang sudah ada
// tidak pernah ditimpa, sehingga perubahan admin tidak hilang saat migrasi
// dijalankan ulang. Nilai yang dihitung adalah jumlah pair baru.
// PairContract adalah metadata kontrak bawaan sebuah simbol.
//
// Dipakai kalkulator risiko untuk mengubah jumlah lot menjadi nominal uang:
//
//	risiko_nominal = |entry - sl| x value_per_lot x lots
//
// Angka-angkanya adalah nilai umum retail broker. Admin dapat mengubahnya per
// pair lewat panel katalog, dan spesifikasi broker tetap menimpa saat eksekusi.
type PairContract struct {
	ValuePerLot  float64
	ContractSize float64
	MinLot       float64
	MaxLot       float64
	LotStep      float64
}

// DefaultContractForCategory mengembalikan metadata kontrak menurut kategori
// dan simbol. Nilai ini hanya titik awal; tidak pernah menimpa isian admin.
func DefaultContractForCategory(categoryKey, symbol string) PairContract {
	base := PairContract{MinLot: 0.01, MaxLot: 10, LotStep: 0.01}

	switch categoryKey {
	case "forex":
		base.ContractSize = 100_000
		base.ValuePerLot = 100_000
		// Pair ber-quote JPY: nilai 1.0 pergerakan harga sekitar 1/100 dari
		// pair ber-quote USD, karena harga dinyatakan per 0.01 yen.
		if strings.HasSuffix(symbol, "JPY") {
			base.ValuePerLot = 1_000
		}
	case "indices":
		base.ContractSize = 1
		base.ValuePerLot = 1
	case "crypto":
		base.ContractSize = 1
		base.ValuePerLot = 1
	case "commodity":
		switch {
		case symbol == "XAUUSD", symbol == "XAUEUR":
			base.ContractSize = 100 // 1 lot = 100 troy ounce
			base.ValuePerLot = 100
		case symbol == "XAGUSD":
			base.ContractSize = 5_000 // 1 lot = 5000 troy ounce
			base.ValuePerLot = 5_000
		default:
			// Energi (USOIL): 1 lot = 1000 barel.
			base.ContractSize = 1_000
			base.ValuePerLot = 1_000
		}
	}

	return base
}

func SeedPairs(ctx context.Context, db *gorm.DB, at time.Time) (int, error) {
	categories, pairs := DefaultPairSeeds()

	seeded := 0
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		categoryID := make(map[string]string, len(categories))

		for _, cat := range categories {
			var existing model.PairCategory
			// Unscoped: baris yang pernah dihapus (soft delete) masih memegang
			// kunci unik, sehingga pembuatan ulang akan gagal. Baris terhapus
			// sengaja dilewati agar katalog tidak menghidupkan kembali data
			// yang sudah dibuang admin.
			err := tx.Unscoped().Where("LOWER(key) = ?", cat.Key).First(&existing).Error
			if err == nil {
				if existing.DeletedAt.Valid {
					continue
				}
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
				err := tx.Unscoped().Where("category_id = ? AND LOWER(key) = ?", categoryID[cat.Key], grp.Key).First(&existingGroup).Error
				if err == nil {
					// Grup terhapus tidak dibuatkan ulang, juga tidak dipakai
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
			var existing model.TradingPair
			err := tx.Unscoped().Where("symbol = ?", pair.Symbol).First(&existing).Error
			if err == nil {
				if existing.DeletedAt.Valid {
					// Pair yang sudah dihapus admin tidak dihidupkan kembali.
					continue
				}
				// Korelasi dan metadata kontrak hanya diisi bila masih kosong;
				// nilai yang sudah diubah admin tidak pernah ditimpa migrasi.
				patch := map[string]any{}
				if existing.CorrelatedSymbols == "" && pair.CorrelatedSymbols != "" {
					patch["correlated_symbols"] = pair.CorrelatedSymbols
				}
				contract := DefaultContractForCategory(pair.CategoryKey, pair.Symbol)
				if existing.ValuePerLot <= 0 && contract.ValuePerLot > 0 {
					patch["value_per_lot"] = contract.ValuePerLot
				}
				if existing.ContractSize <= 0 && contract.ContractSize > 0 {
					patch["contract_size"] = contract.ContractSize
				}
				if len(patch) > 0 {
					if err := tx.Model(&model.TradingPair{}).
						Where("id = ?", existing.ID).
						Updates(patch).Error; err != nil {
						return fmt.Errorf("mengisi metadata %s: %w", pair.Symbol, err)
					}
				}
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("memeriksa pair %s: %w", pair.Symbol, err)
			}

			contract := DefaultContractForCategory(pair.CategoryKey, pair.Symbol)
			row := &model.TradingPair{
				Symbol:            pair.Symbol,
				DisplayName:       pair.DisplayName,
				CategoryID:        catID,
				Aliases:           pair.Aliases,
				CorrelatedSymbols: pair.CorrelatedSymbols,
				SortOrder:         pair.SortOrder,
				IsEnabled:         true,
				Description:       pair.Description,
				ValuePerLot:       contract.ValuePerLot,
				ContractSize:      contract.ContractSize,
				MinLot:            contract.MinLot,
				MaxLot:            contract.MaxLot,
				LotStep:           contract.LotStep,
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
