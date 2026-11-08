package dto

import "time"

// FaqItem adalah satu tanya-jawab di pusat bantuan.
type FaqItem struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"`
	Question  string    `json:"question"`
	AnswerMD  string    `json:"answer_md"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	UpdatedAt time.Time `json:"updated_at"`
}

// FaqRequest adalah permintaan menyimpan FAQ.
type FaqRequest struct {
	Category  string `json:"category" binding:"required,min=2,max=60"`
	Question  string `json:"question" binding:"required,min=5,max=255"`
	AnswerMD  string `json:"answer_md" binding:"required,min=5,max=8000"`
	SortOrder int    `json:"sort_order"`
	IsActive  *bool  `json:"is_active"`
}

// AnnouncementItem adalah satu pengumuman di dashboard pelanggan.
type AnnouncementItem struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Level     string     `json:"level"`
	StartsAt  time.Time  `json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
}

// AnnouncementRequest adalah permintaan menyimpan pengumuman.
// Waktu dikirim dalam RFC3339; kosong berarti mulai sekarang / tanpa batas akhir.
type AnnouncementRequest struct {
	Title    string `json:"title" binding:"required,min=3,max=160"`
	Body     string `json:"body" binding:"required,min=5,max=4000"`
	Level    string `json:"level" binding:"required,oneof=info warning critical"`
	StartsAt string `json:"starts_at,omitempty"`
	EndsAt   string `json:"ends_at,omitempty"`
	IsActive *bool  `json:"is_active"`
	Reason   string `json:"reason,omitempty"`
}
