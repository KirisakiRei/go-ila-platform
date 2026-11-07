package model

import "time"

// AuditLog mencatat peristiwa penting secara append-only.
// Tidak pernah memuat rahasia: hanya identitas samaran dan metadata kejadian.
type AuditLog struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID    string    `gorm:"type:varchar(36);index" json:"user_id"`
	Event     string    `gorm:"type:varchar(60);index;not null" json:"event"`
	Entity    string    `gorm:"type:varchar(40)" json:"entity"`
	EntityID  string    `gorm:"type:varchar(64)" json:"entity_id"`
	Metadata  string    `gorm:"type:text" json:"metadata"`
	IPHash    string    `gorm:"type:varchar(64)" json:"ip_hash"`
	UserAgent string    `gorm:"type:varchar(255)" json:"user_agent"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// TableName menetapkan nama tabel.
func (AuditLog) TableName() string {
	return "audit_log"
}

// ApprovalToken adalah token deep-link sekali pakai untuk konfirmasi entry.
// Yang disimpan hanya hash; token asli hanya dikirim ke user lewat Telegram.
type ApprovalToken struct {
	ID        string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TokenHash string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	UserID    string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	SignalID  string     `gorm:"type:varchar(64);index;not null" json:"signal_id"`
	Action    string     `gorm:"type:varchar(20);not null" json:"action"`
	UsedAt    *time.Time `json:"used_at"`
	ExpiresAt time.Time  `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (ApprovalToken) TableName() string {
	return "approval_token"
}

// OnboardingToken menghubungkan chat Telegram dengan akun lewat /start <token>.
// Token asli tidak pernah disimpan; hanya hash-nya.
type OnboardingToken struct {
	Base
	TokenHash string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	UserID    string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	UsedAt    *time.Time `json:"used_at"`
	ExpiresAt time.Time  `gorm:"index;not null" json:"expires_at"`
}

// TableName menetapkan nama tabel.
func (OnboardingToken) TableName() string {
	return "onboarding_token"
}

// NotificationLog mencegah pesan yang sama dikirim berulang ke user yang sama.
type NotificationLog struct {
	ID         string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID     string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Channel    string     `gorm:"type:varchar(20);not null" json:"channel"`
	MessageKey string     `gorm:"type:varchar(120);uniqueIndex;not null" json:"message_key"`
	Status     string     `gorm:"type:varchar(20);not null" json:"status"`
	Attempts   int        `gorm:"not null;default:0" json:"attempts"`
	LastError  string     `gorm:"type:varchar(255)" json:"last_error"`
	SentAt     *time.Time `json:"sent_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (NotificationLog) TableName() string {
	return "notification_log"
}
