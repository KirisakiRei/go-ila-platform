package dto

import "time"

// CreateInviteRequest adalah payload pembuatan tautan undangan.
type CreateInviteRequest struct {
	// Tier bawaan plus. Boleh diubah admin per undangan.
	Tier string `json:"tier" binding:"omitempty,oneof=free basic plus"`
	// MaxUses wajib >= 1; tidak ada nilai yang berarti tanpa batas.
	MaxUses int `json:"max_uses" binding:"required,min=1,max=10000"`
	// ExpiresAt kosong berarti 7 hari sejak dibuat.
	ExpiresAt time.Time `json:"expires_at"`
	// Email kosong berarti tautan berlaku untuk siapa pun pemegangnya.
	Email string `json:"email" binding:"omitempty,email,max=150"`
	Note  string `json:"note" binding:"omitempty,max=255"`
}

// InviteItem adalah satu baris daftar undangan untuk admin.
//
// Tidak pernah memuat kode mentah; hanya prefix sebagai pengenal tampilan.
type InviteItem struct {
	ID        string    `json:"id"`
	Prefix    string    `json:"prefix"`
	Email     string    `json:"email"`
	Tier      string    `json:"tier"`
	MaxUses   int       `json:"max_uses"`
	UsedCount int       `json:"used_count"`
	SisaKuota int       `json:"sisa_kuota"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    string    `json:"status"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// CreatedInvite adalah balasan pembuatan undangan.
//
// Code dan URL hanya muncul di sini dan tidak bisa diambil lagi. Admin wajib
// menyalinnya sekarang; bila hilang, buat undangan baru.
type CreatedInvite struct {
	InviteItem
	Code string `json:"code"`
	URL  string `json:"url"`
}

// InvitePublic adalah balasan pemeriksaan undangan untuk halaman daftar.
//
// Sengaja tidak memuat alamat surel tujuan: tautan bisa disebar, dan alamat
// orang lain tidak boleh tampil di halaman publik.
type InvitePublic struct {
	Valid         bool      `json:"valid"`
	Tier          string    `json:"tier"`
	ExpiresAt     time.Time `json:"expires_at"`
	SisaKuota     int       `json:"sisa_kuota"`
	EmailTerkunci bool      `json:"email_terkunci"`
}
