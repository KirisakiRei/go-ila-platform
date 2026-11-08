package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Status antrean surel.
const (
	EmailPending = "pending"
	EmailSent    = "sent"
	EmailFailed  = "failed"
)

// EmailOutbox adalah antrean surel keluar.
//
// Payload hanya memuat pengenal (misalnya token_id), tidak pernah token mentah:
// tabel ini dibaca operator saat menyelidiki masalah (plans/17 §2).
type EmailOutbox struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	ToEmail     string     `gorm:"type:varchar(150);index;not null" json:"to_email"`
	Template    string     `gorm:"type:varchar(40);not null" json:"template"`
	Payload     string     `gorm:"type:text;not null" json:"payload"`
	Status      string     `gorm:"type:varchar(20);index;not null;default:'pending'" json:"status"`
	Attempts    int        `gorm:"not null;default:0" json:"attempts"`
	LastError   string     `gorm:"type:varchar(255);not null;default:''" json:"last_error"`
	ScheduledAt time.Time  `gorm:"index;not null" json:"scheduled_at"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (EmailOutbox) TableName() string {
	return "email_outbox"
}

// BeforeCreate mengisi ID bila kosong.
func (e *EmailOutbox) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}

// Tujuan token sekali pakai.
const (
	TokenPurposeVerify = "verify"
	TokenPurposeReset  = "reset"
)

// EmailToken adalah token sekali pakai untuk verifikasi email dan reset sandi.
// Hanya hash yang disimpan, sehingga bocornya tabel tidak memberi akses.
type EmailToken struct {
	ID        string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID    string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Purpose   string     `gorm:"type:varchar(20);index;not null" json:"purpose"`
	TokenHash string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"index;not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (EmailToken) TableName() string {
	return "email_token"
}

// BeforeCreate mengisi ID bila kosong.
func (e *EmailToken) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
