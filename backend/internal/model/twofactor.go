package model

import "time"

// TwoFactor menyimpan pengaturan TOTP milik satu akun (plans/05 §9).
// Dipakai bersama oleh admin dan user; secret tidak pernah keluar dari server.
type TwoFactor struct {
	ID     string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID string `gorm:"type:varchar(36);uniqueIndex;not null" json:"user_id"`
	// SecretEnc adalah secret TOTP terenkripsi dengan master key.
	SecretEnc string `gorm:"type:text;not null" json:"-"`
	// SecretVersion menunjuk versi master key yang membungkus secret.
	SecretVersion int `gorm:"not null;default:1" json:"-"`
	// ActivatedAt kosong berarti aktivasi belum selesai (masih scan QR).
	ActivatedAt *time.Time `json:"activated_at"`
	// LastUsedStep mencegah kode yang sama dipakai dua kali pada jendela TOTP.
	LastUsedStep int64      `gorm:"not null;default:0" json:"-"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (TwoFactor) TableName() string {
	return "user_2fa"
}

// RecoveryCode adalah kode pemulihan sekali pakai saat perangkat hilang.
// Yang disimpan hanya hash, sama seperti password.
type RecoveryCode struct {
	ID     string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	// CodeHash adalah SHA-256 dari kode yang sudah dinormalkan.
	CodeHash  string     `gorm:"type:varchar(64);index;not null" json:"-"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (RecoveryCode) TableName() string {
	return "user_recovery_code"
}

// TwoFactorChallenge adalah keadaan antara antara password benar dan 2FA
// berhasil (plans/05 §4.1). Klien hanya memegang ID-nya, bukan sesi.
type TwoFactorChallenge struct {
	ID     string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	// Purpose membedakan pintu user dan pintu admin.
	Purpose string `gorm:"type:varchar(20);not null" json:"purpose"`
	// Attempts menghitung kode salah; 5 kali hangus.
	Attempts int `gorm:"not null;default:0" json:"attempts"`
	// RecoveryMode menandai challenge yang lahir dari recovery code.
	RecoveryMode bool       `gorm:"not null;default:false" json:"recovery_mode"`
	ExpiresAt    time.Time  `gorm:"index;not null" json:"expires_at"`
	ConsumedAt   *time.Time `json:"consumed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (TwoFactorChallenge) TableName() string {
	return "two_factor_challenge"
}
