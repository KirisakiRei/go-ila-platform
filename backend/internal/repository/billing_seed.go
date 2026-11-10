package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
)

// SeedPlans mengisi paket bawaan bila belum ada. Nilainya sengaja dapat
// diubah admin lewat panel, jadi seed ini hanya titik awal.
func SeedPlans(ctx context.Context, db *gorm.DB) error {
	now := time.Now().UTC()

	plans := []model.Plan{
		{
			Code: model.TierFree, Name: "Free", PriceIDR: 0, Interval: "monthly",
			IsActive: true, SortOrder: 0,
			Description: "Melihat sinyal dengan penundaan, tanpa eksekusi otomatis.",
			UpdatedAt:   now,
		},
		{
			Code: model.TierBasic, Name: "Basic", PriceIDR: 666000, Interval: "monthly",
			IsActive: true, SortOrder: 1,
			Description: "Seluruh sinyal tanpa penundaan, eksekusi manual.",
			UpdatedAt:   now,
		},
		{
			Code: model.TierPlus, Name: "Plus", PriceIDR: 999000, Interval: "monthly",
			IsActive: true, SortOrder: 2,
			Description: "Eksekusi otomatis, risiko dapat diatur, hingga 2 akun broker.",
			UpdatedAt:   now,
		},
	}

	features := map[string]map[string]string{
		model.TierFree: {
			model.FeatureAccountLimit: "1",
			model.FeatureMaxSymbols:   "1",
			model.FeatureAutoExecute:  "false",
			// Free menerima sinyal telat 15 menit.
			model.FeatureSignalDelay: "15",
			model.FeatureStrategies:  "ila-2.0",
			model.FeatureILA3Status:  "coming_soon",
			model.FeatureReasonAI:    "false",
			// Free tidak auto-execute, jadi tidak ada total risiko untuk dibatasi.
			model.FeatureMaxTotalRisk: "0",
		},
		model.TierBasic: {
			model.FeatureAccountLimit: "1",
			model.FeatureMaxSymbols:   "8",
			model.FeatureAutoExecute:  "false",
			model.FeatureSignalDelay:  "0",
			model.FeatureStrategies:   "ila-2.0",
			model.FeatureILA3Status:   "coming_soon",
			model.FeatureReasonAI:     "false",
			// Pagar risiko kini 5% per trade; batas total ikut dinaikkan agar
			// nilai di atas 1% benar-benar dapat dipakai.
			model.FeatureMaxTotalRisk: "5",
		},
		model.TierPlus: {
			model.FeatureAccountLimit: "2",
			model.FeatureMaxSymbols:   "8",
			model.FeatureAutoExecute:  "true",
			model.FeatureSignalDelay:  "0",
			// ILA 3.0 sudah dirilis untuk tier Plus. Paket lain tetap
			// "coming_soon" sehingga tidak menahan tahap rilis paket ini.
			model.FeatureStrategies: "ila-2.0,ila-3.0",
			model.FeatureILA3Status: "general",
			model.FeatureReasonAI:   "true",
			// Batas total risiko seluruh akun aktif milik satu pengguna (persen).
			model.FeatureMaxTotalRisk: "5",
		},
	}

	guards := map[string]int64{
		model.TierFree:  0,
		model.TierBasic: 333000,
		model.TierPlus:  499000,
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, plan := range plans {
			// DoNothing: harga yang sudah disesuaikan admin tidak ditimpa seed.
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&plan).Error; err != nil {
				return fmt.Errorf("mengisi paket %s: %w", plan.Code, err)
			}
		}

		for code, values := range features {
			for key, value := range values {
				row := &model.PlanFeature{PlanCode: code, FeatureKey: key, Value: value}
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
					return fmt.Errorf("mengisi fitur %s.%s: %w", code, key, err)
				}
			}
		}

		for code, min := range guards {
			// Pagar harga wajib ada: tanpa baris ini paket berbayar ditolak
			// saat checkout. Paket yang ditambah admin juga harus punya pagar,
			// jadi baris yang hilang dibuatkan dengan batas yang sama.
			guard := &model.PricingGuard{PlanCode: code, MinPriceIDR: min, UpdatedBy: "seed", UpdatedAt: now}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(guard).Error; err != nil {
				return fmt.Errorf("mengisi batas bawah %s: %w", code, err)
			}
		}
		return nil
	})
}

// TierFeature membaca satu fitur paket dari database dengan nilai cadangan.
// Dipakai Runner dan backend agar batas tier tidak lagi tertulis di kode.
func TierFeature(ctx context.Context, db *gorm.DB, tier, key, fallback string) string {
	var row model.PlanFeature
	err := db.WithContext(ctx).Where("plan_code = ? AND feature_key = ?", tier, key).Limit(1).Find(&row).Error
	if err != nil || row.Value == "" {
		return fallback
	}
	return row.Value
}

// tierFeatureReader menyediakan TierFeature dalam bentuk antarmuka service.
type tierFeatureReader struct{ db *gorm.DB }

// NewTierFeatureReader membuat pembaca fitur paket dari database.
func NewTierFeatureReader(db *gorm.DB) *tierFeatureReader {
	return &tierFeatureReader{db: db}
}

// Feature membaca satu fitur paket dengan nilai cadangan.
func (r *tierFeatureReader) Feature(ctx context.Context, tier, key, fallback string) string {
	return TierFeature(ctx, r.db, tier, key, fallback)
}
