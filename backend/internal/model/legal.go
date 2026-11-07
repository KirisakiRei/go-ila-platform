package model

import "time"

// LegalDocument adalah satu versi dokumen legal. Dokumen tidak pernah ditimpa:
// perubahan berarti versi baru (plans/09 §2.1).
type LegalDocument struct {
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	// Slug menandai jenis dokumen: tos, privacy, refund, risk, fair_use.
	Slug    string `gorm:"type:varchar(40);index;not null" json:"slug"`
	Version int    `gorm:"not null" json:"version"`
	Title   string `gorm:"type:varchar(150);not null" json:"title"`
	BodyMD  string `gorm:"type:text;not null" json:"body_md"`
	// RequireReaccept menandai perubahan yang wajib disetujui ulang.
	RequireReaccept bool `gorm:"not null;default:false" json:"require_reaccept"`
	// PublishedAt kosong berarti masih draf.
	PublishedAt *time.Time `json:"published_at"`
	// EffectiveAt boleh di masa depan untuk menjadwalkan berlaku.
	EffectiveAt time.Time `gorm:"not null" json:"effective_at"`
	CreatedBy   string    `gorm:"type:varchar(36)" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (LegalDocument) TableName() string {
	return "legal_document"
}

// LegalAcceptance adalah bukti seorang akun menyetujui versi tertentu.
// Tidak pernah dihapus; dianonimkan bila akun dihapus (plans/09 §11).
type LegalAcceptance struct {
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	// UserID kosong setelah akun dihapus, tetapi bukti persetujuan tetap ada.
	UserID       string    `gorm:"type:varchar(36);index" json:"user_id"`
	DocumentSlug string    `gorm:"type:varchar(40);index;not null" json:"document_slug"`
	Version      int       `gorm:"not null" json:"version"`
	AcceptedAt   time.Time `gorm:"index;not null" json:"accepted_at"`
	IPHash       string    `gorm:"type:varchar(64)" json:"-"`
	UserAgent    string    `gorm:"type:varchar(255)" json:"-"`
}

// TableName menetapkan nama tabel.
func (LegalAcceptance) TableName() string {
	return "legal_acceptance"
}
