package model

import "time"

// WebPushVAPID menyimpan pasangan kunci VAPID tunggal (id=1) secara terenkripsi
// bila kunci tidak disetel lewat variabel environment, sehingga backend-api dan
// runner memakai identitas VAPID yang persis sama.
type WebPushVAPID struct {
	ID               int       `gorm:"primaryKey;autoIncrement:false" json:"id"`
	PublicKey        string    `gorm:"type:varchar(128);not null" json:"public_key"`
	PrivateKeySealed string    `gorm:"type:text;not null" json:"-"`
	KeyVersion       int       `gorm:"not null;default:1" json:"key_version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (WebPushVAPID) TableName() string {
	return "web_push_vapid"
}

// WebPushSubscription menyimpan langganan Web Push per perangkat browser/PWA.
// Endpoint asli dan kunci enkripsi tidak pernah dikirim kembali ke klien.
type WebPushSubscription struct {
	ID           string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	EndpointHash string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	Endpoint     string     `gorm:"type:text;not null" json:"-"`
	P256dh       string     `gorm:"type:varchar(128);not null" json:"-"`
	Auth         string     `gorm:"type:varchar(64);not null" json:"-"`
	DeviceLabel  string     `gorm:"type:varchar(120)" json:"device_label"`
	NotifyEntry  bool       `gorm:"not null;default:true" json:"notify_entry"`
	NotifyExit   bool       `gorm:"not null;default:true" json:"notify_exit"`
	LastSentAt   *time.Time `json:"last_sent_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (WebPushSubscription) TableName() string {
	return "web_push_subscription"
}
