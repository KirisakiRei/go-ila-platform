package model

import "time"

// Mode risiko per akun.
const (
	// RiskModePercent memakai persentase ekuitas sebagai batas risiko.
	RiskModePercent = "percent"
	// RiskModeLot memakai ukuran lot tetap yang dipilih user.
	RiskModeLot = "lot"
)

// BrokerAccount adalah akun broker milik user yang dihubungkan ke platform.
// Kredensial TIDAK disimpan di sini; lihat Credential dengan enkripsi envelope.
type BrokerAccount struct {
	Base
	UserID            string  `gorm:"type:varchar(36);index;not null" json:"user_id"`
	BrokerType        string  `gorm:"type:varchar(20);not null" json:"broker_type"`
	Label             string  `gorm:"type:varchar(60)" json:"label"`
	ExternalAccountID string  `gorm:"type:varchar(64)" json:"external_account_id"`
	Mode              string  `gorm:"type:varchar(10);not null;default:manual" json:"mode"`
	RiskMode          string  `gorm:"type:varchar(20);not null;default=percent" json:"risk_mode"`
	RiskPercent       float64 `gorm:"not null;default:0.5" json:"risk_percent"`
	// LotSize dipakai saat RiskMode=lot. Nilainya tetap disimpan walau mode
	// sedang persen, supaya user bisa berpindah mode tanpa kehilangan angka.
	LotSize        float64    `gorm:"type:numeric(10,2);not null;default:0.01" json:"lot_size"`
	IsActive       bool       `gorm:"not null;default:true" json:"is_active"`
	IsFrozen       bool       `gorm:"not null;default:false" json:"is_frozen"`
	IsDryRun       bool       `gorm:"not null;default:true" json:"is_dry_run"`
	LastVerifiedAt *time.Time `json:"last_verified_at"`
	LastError      string     `gorm:"type:varchar(255)" json:"last_error"`
}

// TableName menetapkan nama tabel.
func (BrokerAccount) TableName() string {
	return "broker_account"
}

// Credential menyimpan blob kredensial yang sudah terenkripsi.
// Plaintext hanya hidup sesaat di memori Runner.
type Credential struct {
	Base
	BrokerAccountID string `gorm:"type:varchar(36);uniqueIndex;not null" json:"broker_account_id"`
	UserID          string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	UserKeyWrapped  []byte `gorm:"type:bytea;not null" json:"-"`
	UserKeyNonce    []byte `gorm:"type:bytea;not null" json:"-"`
	Ciphertext      []byte `gorm:"type:bytea;not null" json:"-"`
	Nonce           []byte `gorm:"type:bytea;not null" json:"-"`
	KeyVersion      int    `gorm:"not null;default:1" json:"key_version"`
	// TokenFingerprint adalah SHA-256 bertanda kunci dari token broker.
	// Dipakai mendeteksi dua akun yang memakai token sama, bukan untuk login.
	TokenFingerprint string    `gorm:"type:varchar(64);index" json:"-"`
	RotatedAt        time.Time `json:"rotated_at"`
}

// TableName menetapkan nama tabel.
func (Credential) TableName() string {
	return "credential"
}

// Subscription memilih simbol yang dipantau user.
//
// Dulu tabel ini juga menyimpan kolom `strategy`. Sejak metode analisis dipilih
// per akun (tabel account_pair_strategy), kolom itu sudah tidak dibaca dan telah
// dihapus lewat migrasi. Satu pair dipantau seluruh metode; Runner menentukan
// metode mana yang dieksekusi dari saklar tiap akun.
type Subscription struct {
	Base
	UserID string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	Symbol string `gorm:"type:varchar(20);not null" json:"symbol"`
	// Tanpa default di tag: GORM akan mengisi true saat nilainya false,
	// sehingga langganan yang dicabut bisa hidup lagi. Pencabutan ditandai
	// is_active=false dan harus bisa tersimpan apa adanya.
	IsActive        bool    `gorm:"not null" json:"is_active"`
	BrokerAccountID *string `gorm:"type:varchar(36);index" json:"broker_account_id"`
	// PairID menautkan langganan ke master pair. Kosong untuk data lama yang
	// dibuat sebelum katalog pair ada; Symbol tetap menjadi kunci eksekusi.
	PairID *string `gorm:"type:varchar(36);index" json:"pair_id"`
}

// TableName menetapkan nama tabel.
func (Subscription) TableName() string {
	return "subscription"
}
