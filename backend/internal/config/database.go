package config

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// NewDatabase membuka koneksi GORM beserta pengaturan connection pool.
// Pool dibatasi karena VPS hanya 1 GB; koneksi tak terbatas akan menghabiskan memori.
func NewDatabase(ctx context.Context, cfg *Config) (*gorm.DB, error) {
	logMode := gormlogger.Warn
	if !cfg.IsProduction() {
		logMode = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DBDSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logMode),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("membuka koneksi database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("mengambil sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLife)

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// AutoMigrate membuat atau memperbarui tabel dari struct model.
// Hanya dijalankan bila DB_AUTO_MIGRATE aktif; produksi memakai migrasi eksplisit.
func AutoMigrate(ctx context.Context, db *gorm.DB, models ...any) error {
	if err := db.WithContext(ctx).AutoMigrate(models...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
