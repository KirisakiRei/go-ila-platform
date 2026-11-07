package model

import "time"

// BrokerAccount adalah akun broker milik user yang dihubungkan ke platform.
// Kredensial TIDAK disimpan di sini; lihat Credential dengan enkripsi envelope.
type BrokerAccount struct {
	Base
	UserID            string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	BrokerType        string     `gorm:"type:varchar(20);not null" json:"broker_type"`
	Label             string     `gorm:"type:varchar(60)" json:"label"`
	ExternalAccountID string     `gorm:"type:varchar(64)" json:"external_account_id"`
	Mode              string     `gorm:"type:varchar(10);not null;default:manual" json:"mode"`
	RiskMode          string     `gorm:"type:varchar(20);not null;default=percent" json:"risk_mode"`
	RiskPercent       float64    `gorm:"not null;default:0.5" json:"risk_percent"`
	IsActive          bool       `gorm:"not null;default:true" json:"is_active"`
	IsFrozen          bool       `gorm:"not null;default:false" json:"is_frozen"`
	IsDryRun          bool       `gorm:"not null;default:true" json:"is_dry_run"`
	LastVerifiedAt    *time.Time `json:"last_verified_at"`
	LastError         string     `gorm:"type:varchar(255)" json:"last_error"`
}

// TableName menetapkan nama tabel.
func (BrokerAccount) TableName() string {
	return "broker_account"
}

// Credential menyimpan blob kredensial yang sudah terenkripsi.
// Plaintext hanya hidup sesaat di memori Runner.
type Credential struct {
	Base
	BrokerAccountID string    `gorm:"type:varchar(36);uniqueIndex;not null" json:"broker_account_id"`
	UserID          string    `gorm:"type:varchar(36);index;not null" json:"user_id"`
	UserKeyWrapped  []byte    `gorm:"type:bytea;not null" json:"-"`
	UserKeyNonce    []byte    `gorm:"type:bytea;not null" json:"-"`
	Ciphertext      []byte    `gorm:"type:bytea;not null" json:"-"`
	Nonce           []byte    `gorm:"type:bytea;not null" json:"-"`
	KeyVersion      int       `gorm:"not null;default:1" json:"key_version"`
	RotatedAt       time.Time `json:"rotated_at"`
}

// TableName menetapkan nama tabel.
func (Credential) TableName() string {
	return "credential"
}

// Subscription memilih simbol dan strategi yang diaktifkan user.
// Engine tidak pernah tahu isi tabel ini; Runner menyaringnya menjadi daftar
// pasangan (symbol, strategy) saat berlangganan.
type Subscription struct {
	Base
	UserID          string  `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Symbol          string  `gorm:"type:varchar(20);not null" json:"symbol"`
	Strategy        string  `gorm:"type:varchar(20);not null;default:ila-2.0" json:"strategy"`
	IsActive        bool    `gorm:"not null;default:true" json:"is_active"`
	BrokerAccountID *string `gorm:"type:varchar(36)" json:"broker_account_id"`
}

// TableName menetapkan nama tabel.
func (Subscription) TableName() string {
	return "subscription"
}
