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

// SubmitInviteRequest adalah payload publik form "Ajukan Bergabung".
// Tidak memuat kredensial apa pun; field Website adalah honeypot anti-bot.
type SubmitInviteRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	Email   string `json:"email" binding:"required,email,max=150"`
	Contact string `json:"contact" binding:"omitempty,max=100"`
	Note    string `json:"note" binding:"omitempty,max=500"`
	Website string `json:"website" binding:"omitempty,max=100"`
}

// InviteRequestItem adalah satu baris permohonan akses di panel admin.
type InviteRequestItem struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Contact    string     `json:"contact"`
	Note       string     `json:"note"`
	Status     string     `json:"status"`
	InviteID   string     `json:"invite_id,omitempty"`
	ReviewedBy string     `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote string     `json:"review_note,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// InviteRequestList adalah daftar berpaginasi permohonan akses untuk admin.
type InviteRequestList struct {
	Items        []InviteRequestItem `json:"items"`
	PendingCount int64               `json:"pending_count"`
}

// ApproveInviteRequestBody adalah opsi saat admin menyetujui permohonan akses.
type ApproveInviteRequestBody struct {
	Tier        string `json:"tier" binding:"omitempty,oneof=free basic plus"`
	ExpiresDays int    `json:"expires_days" binding:"omitempty,min=1,max=365"`
	Note        string `json:"note" binding:"omitempty,max=255"`
}

// RejectInviteRequestBody adalah payload penolakan permohonan akses.
type RejectInviteRequestBody struct {
	Reason string `json:"reason" binding:"omitempty,max=255"`
}

// ApprovedInviteRequestResult mengembalikan permohonan yang disetujui beserta
// tautan undangan sekali tampil dan status pengiriman antrean surel.
type ApprovedInviteRequestResult struct {
	Request     InviteRequestItem `json:"request"`
	Invite      CreatedInvite     `json:"invite"`
	EmailQueued bool              `json:"email_queued"`
}
