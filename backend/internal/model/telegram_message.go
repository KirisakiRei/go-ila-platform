package model

import "time"

// TelegramMessage mencatat pesan bot yang punya tombol inline.
//
// Telegram tidak bisa mengubah tombol pada pesan yang tidak dikenal nomornya.
// Tanpa catatan ini, tombol "Eksekusi" pada setup yang sudah lewat akan terus
// terlihat bisa ditekan padahal tautannya sudah mati.
type TelegramMessage struct {
	ID        string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID    string     `gorm:"type:varchar(36);index;not null" json:"user_id"`
	ChatID    string     `gorm:"type:varchar(32);not null" json:"chat_id"`
	MessageID int64      `gorm:"not null" json:"message_id"`
	SignalID  string     `gorm:"type:varchar(64);index" json:"signal_id"`
	Kind      string     `gorm:"type:varchar(24);not null" json:"kind"`
	Text      string     `gorm:"type:text" json:"text"`
	ClearedAt *time.Time `json:"cleared_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// TableName menetapkan nama tabel.
func (TelegramMessage) TableName() string {
	return "telegram_message"
}

// Jenis pesan bertombol yang dicatat.
const (
	TelegramPesanSetup = "setup"
)
