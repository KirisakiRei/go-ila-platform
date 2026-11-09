package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"ila/backend/internal/model"
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
	if err := db.WithContext(ctx).Model(&model.User{}).
		Where("is_protected = ?", true).
		Count(&sudahAda).Error; err != nil {
		return false, fmt.Errorf("memeriksa akun terlindungi: %w", err)
	}
	if sudahAda > 0 {
		return false, nil
	}

	// Kueri dibentuk lewat Model(&model.User{}) supaya GORM sendiri yang
	// mengutip nama tabel: `user` adalah kata kunci SQL di Postgres, sedangkan
	// SQLite menolak kutip ganda bertingkat. Menulis nama tabel di dalam
	// Table() membuat GORM mengutipnya dua kali dan kueri gagal di SQLite.
	//
	// Role dicocokkan lewat kode agar tidak bergantung pada id role yang
	// berbeda antar lingkungan.
	var kandidat []string
	err := db.WithContext(ctx).Model(&model.User{}).
		Joins(`JOIN role r ON r.id = "user".role_id`).
		Where("r.code = ?", "superadmin").
		Where(`"user".status = ?`, "active").
		Where(`"user".deleted_at IS NULL`).
		Order(`"user".created_at ASC`).
		Limit(1).
		Pluck(`"user".id`, &kandidat).Error
	if err != nil {
		return false, fmt.Errorf("mencari superadmin utama: %w", err)
	}
	if len(kandidat) == 0 || kandidat[0] == "" {
		return false, nil
	}

	res := db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", kandidat[0]).
		Update("is_protected", true)
	if res.Error != nil {
		return false, fmt.Errorf("menandai superadmin utama: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}
