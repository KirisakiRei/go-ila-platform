package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// DropSubscriptionStrategy menghapus kolom `subscription.strategy` setelah
// saklar per akun menggantikannya.
//
// Ini tahap kedua penggantian metode. Fungsi menolak berjalan bila masih ada
// langganan aktif yang belum punya saklar sama sekali — kolom lama adalah
// satu-satunya sumber pilihan metode pada data seperti itu, jadi menghapusnya
// lebih dulu akan menghilangkan preferensi user tanpa bisa dipulihkan.
//
// Idempoten: kolom yang sudah tidak ada dilaporkan sebagai sudah selesai.
func DropSubscriptionStrategy(ctx context.Context, db *gorm.DB) (bool, error) {
	ada, err := columnExists(ctx, db, "subscription", "strategy")
	if err != nil {
		return false, err
	}
	if !ada {
		return false, nil
	}

	// Pastikan setiap langganan aktif yang punya akun aktif sudah punya saklar
	// untuk pair-nya. Langganan tanpa akun aktif tidak butuh saklar: saklar
	// hanya bermakna per akun, dan pair itu akan mendapat saklar bawaan begitu
	// akun dihubungkan.
	var belum bool
	err = db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM subscription s
			WHERE s.is_active = true
			  AND s.deleted_at IS NULL
			  AND EXISTS (
				SELECT 1 FROM broker_account ba
				WHERE ba.user_id = s.user_id
				  AND ba.is_active = true
				  AND ba.deleted_at IS NULL
				  AND (s.broker_account_id IS NULL OR s.broker_account_id = ba.id)
			  )
			  AND NOT EXISTS (
				SELECT 1 FROM account_pair_strategy a
				WHERE a.user_id = s.user_id
				  AND a.symbol = UPPER(s.symbol)
				  AND a.deleted_at IS NULL
			  )
		)`).Scan(&belum).Error
	if err != nil {
		return false, fmt.Errorf("memeriksa saklar sebelum hapus kolom: %w", err)
	}
	if belum {
		return false, fmt.Errorf(
			"kolom subscription.strategy belum bisa dihapus: ada langganan aktif berakun yang belum punya saklar metode; jalankan BackfillAccountPairStrategy lebih dulu")
	}

	if err := db.WithContext(ctx).Exec("ALTER TABLE subscription DROP COLUMN strategy").Error; err != nil {
		return false, fmt.Errorf("menghapus kolom subscription.strategy: %w", err)
	}
	return true, nil
}

// columnExists memeriksa keberadaan kolom di skema aktif.
func columnExists(ctx context.Context, db *gorm.DB, table, column string) (bool, error) {
	var ada bool
	err := db.WithContext(ctx).Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = ?
			  AND column_name = ?
		)`, table, column).Scan(&ada).Error
	if err != nil {
		return false, fmt.Errorf("memeriksa kolom %s.%s: %w", table, column, err)
	}
	return ada, nil
}
