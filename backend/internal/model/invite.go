package model

import "time"

// InviteCode adalah tautan undangan untuk membuat akun saat platform tertutup.
//
// Kode asli TIDAK pernah disimpan, hanya SHA-256-nya. Alasannya: tautan undangan
// ikut tersebar di luar sistem (grup chat, pesan pribadi). Bila tabel ini terbaca
// pihak lain, kode di dalamnya tidak bisa dipakai ulang. Kode mentah hanya
// ditampilkan sekali saat admin membuatnya.
type InviteCode struct {
	Base
	// CodeHash adalah SHA-256 heksadesimal dari kode mentah.
	CodeHash string `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	// Prefix adalah empat karakter awal kode, hanya agar admin bisa mengenali
	// baris yang mana di daftar. Terlalu pendek untuk menebak kode utuh.
	Prefix string `gorm:"type:varchar(12);not null;default:''" json:"prefix"`
	// EmailTujuan kosong berarti tautan berlaku untuk siapa pun pemegangnya.
	// Bila diisi, hanya alamat itu yang boleh memakainya.
	EmailTujuan string `gorm:"type:varchar(150);not null;default:''" json:"email"`
	// Tier menentukan tingkat akun yang lahir dari undangan ini.
	Tier string `gorm:"type:varchar(20);not null;default:'plus'" json:"tier"`
	// MaxUses selalu >= 1. Tidak ada keadaan "0 berarti tanpa batas": kuota adalah
	// satu-satunya batas nyata selain masa berlaku, jadi ia wajib ada.
	MaxUses int `gorm:"not null;default:1" json:"max_uses"`
	// UsedCount hanya bertambah lewat UPDATE bersyarat (lihat InviteRepository).
	UsedCount int       `gorm:"not null;default:0" json:"used_count"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	Note      string    `gorm:"type:varchar(255);not null;default:''" json:"note"`
	CreatedBy string    `gorm:"type:varchar(36);not null" json:"created_by"`
}

// TableName menetapkan nama tabel.
func (InviteCode) TableName() string {
	return "invite_code"
}

// SisaKuota mengembalikan sisa pemakaian; tidak pernah negatif.
func (i InviteCode) SisaKuota() int {
	sisa := i.MaxUses - i.UsedCount
	if sisa < 0 {
		return 0
	}
	return sisa
}

// DapatDipakai melaporkan apakah undangan masih bisa dipakai pada waktu tertentu.
// Dipakai untuk tampilan status; keputusan sebenarnya tetap di UPDATE bersyarat.
func (i InviteCode) DapatDipakai(now time.Time) bool {
	return i.IsActive && now.Before(i.ExpiresAt) && i.SisaKuota() > 0
}

// Status menurunkan label tampilan dari keadaan baris.
func (i InviteCode) Status(now time.Time) string {
	switch {
	case !i.IsActive:
		return "dicabut"
	case i.SisaKuota() == 0:
		return "habis"
	case !now.Before(i.ExpiresAt):
		return "kedaluwarsa"
	default:
		return "aktif"
	}
}

// InviteRedemption mencatat akun mana lahir dari undangan mana.
//
// Baris ini tidak pernah dihapus walau undangannya dicabut: ia bukti asal akun,
// dan dipakai admin menjawab "akun ini dari undangan yang mana".
type InviteRedemption struct {
	Base
	InviteID string `gorm:"type:varchar(36);index;not null" json:"invite_id"`
	// UserID unik: satu akun tidak bisa diklaim oleh dua undangan.
	UserID string    `gorm:"type:varchar(36);uniqueIndex;not null" json:"user_id"`
	Email  string    `gorm:"type:varchar(150);not null" json:"email"`
	UsedAt time.Time `gorm:"not null" json:"used_at"`
}

// TableName menetapkan nama tabel.
func (InviteRedemption) TableName() string {
	return "invite_redemption"
}
