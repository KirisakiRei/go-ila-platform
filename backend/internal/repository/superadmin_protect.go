package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// ProtectPrimarySuperadmin menandai SATU akun superadmin sebagai akun utama
// yang tidak boleh dihapus.
//
// Aturan pemilihan: akun superadmin aktif yang paling awal dibuat. Akun itulah
// yang diasumsikan sebagai pemilik workspace, karena dibuat saat penyiapan.
//
// Sifat idempoten dan tidak pernah berpindah:
//   - Bila sudah ada akun bertanda protected, fungsi tidak melakukan apa pun.
//     Tanda sengaja tidak dipindahkan agar pemilik tidak bergeser diam-diam
//     hanya karena urutan akun berubah.
//   - Bila belum ada akun protected, akun superadmin tertua ditandai. Bila
//     ternyata tidak ada akun superadmin sama sekali, tidak ada yang ditandai
//     dan migrasi tetap berjalan (akun akan dibuat lewat seed-superadmin).
//
// Mengembalikan true bila migrasi ini benar-benar menandai sebuah akun.
func ProtectPrimarySuperadmin(ctx context.Context, db *gorm.DB) (bool, error) {
	var sudahAda int64
	if err := db.WithContext(ctx).Model(&userProtectRow{}).
		Where("is_protected = ?", true).
		Count(&sudahAda).Error; err != nil {
		return false, fmt.Errorf("memeriksa akun terlindungi: %w", err)
	}
	if sudahAda > 0 {
		return false, nil
	}

	// Cari superadmin aktif tertua. Role dicocokkan lewat kode agar tidak
	// bergantung pada id role yang berbeda antar lingkungan.
	var kandidat struct {
		ID string `gorm:"column:id"`
	}
	// Nama tabel dibiarkan tanpa tanda kutip: SQLite (dipakai pengujian) tidak
	// menerima `"user"` sebagai nama tabel, sedangkan Postgres menerima
	// keduanya. Kolom tetap dibatasi alias agar tidak bergantung dialek.
	const tabelPengguna = "user AS u"
	err := db.WithContext(ctx).
		Table(tabelPengguna).
		Select("u.id").
		Joins("JOIN role r ON r.id = u.role_id").
		Where("r.code = ?", "superadmin").
		Where("u.status = ?", "active").
		Where("u.deleted_at IS NULL").
		Order("u.created_at ASC").
		Limit(1).
		Scan(&kandidat).Error
	if err != nil {
		return false, fmt.Errorf("mencari superadmin utama: %w", err)
	}
	if kandidat.ID == "" {
		return false, nil
	}

	res := db.WithContext(ctx).
		Table("user").
		Where("id = ?", kandidat.ID).
		Update("is_protected", true)
	if res.Error != nil {
		return false, fmt.Errorf("menandai superadmin utama: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

// userProtectRow adalah bentuk minimal untuk menghitung akun terlindungi tanpa
// menyeret seluruh kolom model.
type userProtectRow struct {
	ID          string `gorm:"column:id;primaryKey"`
	IsProtected bool   `gorm:"column:is_protected"`
}

// TableName menetapkan nama tabel.
func (userProtectRow) TableName() string {
	return "user"
}
