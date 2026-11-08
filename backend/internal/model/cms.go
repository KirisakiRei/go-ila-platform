package model

import "time"

// Faq adalah tanya-jawab yang tampil di pusat bantuan.
// Disimpan di database agar bisa diubah tanpa deploy (plans/11 J1).
type Faq struct {
	Base
	Category  string `gorm:"type:varchar(60);not null;index" json:"category"`
	Question  string `gorm:"type:varchar(255);not null" json:"question"`
	AnswerMD  string `gorm:"type:text;not null" json:"answer_md"`
	SortOrder int    `gorm:"not null;default:0" json:"sort_order"`
	IsActive  bool   `gorm:"not null;default:true;index" json:"is_active"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (Faq) TableName() string {
	return "faq"
}

// Tingkat penekanan pengumuman.
const (
	AnnounceInfo     = "info"
	AnnounceWarning  = "warning"
	AnnounceCritical = "critical"
)

// Announcement adalah pesan yang muncul di dashboard pelanggan.
// Jendela waktunya membatasi tampil, sehingga pengumuman kedaluwarsa tidak
// perlu dihapus manual.
type Announcement struct {
	Base
	Title     string     `gorm:"type:varchar(160);not null" json:"title"`
	Body      string     `gorm:"type:text;not null" json:"body"`
	Level     string     `gorm:"type:varchar(20);not null;default:'info'" json:"level"`
	StartsAt  time.Time  `gorm:"index" json:"starts_at"`
	EndsAt    *time.Time `json:"ends_at,omitempty"`
	IsActive  bool       `gorm:"not null;default:true;index" json:"is_active"`
	CreatedBy string     `gorm:"type:varchar(36);not null" json:"created_by"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (Announcement) TableName() string {
	return "announcement"
}

// KnownAnnouncementLevel menandai tingkat pengumuman yang sah.
func KnownAnnouncementLevel(level string) bool {
	switch level {
	case AnnounceInfo, AnnounceWarning, AnnounceCritical:
		return true
	default:
		return false
	}
}
