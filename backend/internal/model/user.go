package model

import "time"

// User adalah akun pengguna platform.
// Tier dan status trial menentukan apa yang boleh dilakukan Runner atas namanya.
type User struct {
	Base
	Name         string `gorm:"type:varchar(100);not null" json:"name"`
	Email        string `gorm:"type:varchar(150);uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"`
	Role         string `gorm:"type:varchar(20);not null;default:user" json:"role"`
	// RoleID menunjuk role RBAC. Kosong berarti akun tanpa kewenangan admin.
	RoleID *string `gorm:"type:varchar(36);index" json:"role_id,omitempty"`
	Tier   string  `gorm:"type:varchar(20);not null;default:free" json:"tier"`

	TierStartedAt *time.Time `json:"tier_started_at"`
	TierExpiresAt *time.Time `json:"tier_expires_at"`
	GraceUntil    *time.Time `json:"grace_until"`
	SuspendedAt   *time.Time `json:"suspended_at"`

	TrialUsed      bool       `gorm:"not null;default:false" json:"trial_used"`
	TrialStartedAt *time.Time `json:"trial_started_at"`
	TrialExpiresAt *time.Time `json:"trial_expires_at"`

	// Anti-abuse trial: disimpan sebagai hash, bukan plaintext (plans/14 §5.5.5).
	DeviceFingerprintHash string `gorm:"type:varchar(64);index" json:"-"`
	SignupIPHash          string `gorm:"type:varchar(64)" json:"-"`

	// Flags menyimpan fitur tambahan per user, mis. akses uji strategi.
	Flags string `gorm:"type:varchar(255)" json:"-"`

	TelegramChatID   *string    `gorm:"type:varchar(64);index" json:"telegram_chat_id"`
	TelegramLinkedAt *time.Time `json:"telegram_linked_at"`

	NotifyEnabled bool       `gorm:"not null;default:true" json:"notify_enabled"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	Status        string     `gorm:"type:varchar(20);not null;default:active" json:"status"`

	// SessionVersion dinaikkan saat sesi harus dicabut seketika: ganti password,
	// ban, ubah role, atau 2FA berubah. Token yang membawa nilai lama ditolak.
	SessionVersion int `gorm:"not null;default:1" json:"-"`

	// IdleWarning menentukan apakah modal peringatan "masih di sini?" muncul.
	// Batas diam di server tidak ikut berubah (plans/23 §5).
	IdleWarning bool `gorm:"not null;default:true" json:"idle_warning"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (User) TableName() string {
	return "user"
}
