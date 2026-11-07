package model

import "time"

// Session adalah satu perangkat yang sedang masuk. Disimpan agar user bisa
// melihat daftar sesi aktifnya dan mencabut yang tidak dikenali (plans/03 §8).
//
// Verifikasi tiap permintaan tetap memakai session_version di tabel user.
// Tabel ini untuk pelacakan dan pencabutan per perangkat; sesi yang dicabut
// di sini juga menaikkan session_version lewat jalur service.
type Session struct {
	ID     string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Scope  string `gorm:"type:varchar(10);not null;default:user" json:"scope"`
	// TokenID adalah penanda unik token; bukan token itu sendiri.
	TokenID   string `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	IPHash    string `gorm:"type:varchar(64)" json:"-"`
	UserAgent string `gorm:"type:varchar(255)" json:"user_agent"`
	// DeviceLabel adalah sebutan ramah, mis. "Chrome di Windows".
	DeviceLabel string     `gorm:"type:varchar(80)" json:"device_label"`
	LastSeenAt  time.Time  `gorm:"index;not null" json:"last_seen_at"`
	ExpiresAt   time.Time  `gorm:"index;not null" json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (Session) TableName() string {
	return "session"
}

// Active menandai sesi yang masih boleh dipakai.
func (s *Session) Active(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}
