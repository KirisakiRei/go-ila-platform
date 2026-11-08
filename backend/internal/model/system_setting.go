package model

import "time"

// SystemSetting adalah penimpa nilai flag dari database.
//
// Tabel ini murni override: bila sebuah kunci tidak ada di sini, nilai dari
// environment yang berlaku. Menghapus satu baris berarti kembali ke bawaan
// (plans/11 Fase I §11.2).
type SystemSetting struct {
	Key       string    `gorm:"primaryKey;type:varchar(60)" json:"key"`
	Value     string    `gorm:"type:varchar(120);not null" json:"value"`
	Reason    string    `gorm:"type:varchar(500);not null" json:"reason"`
	UpdatedBy string    `gorm:"type:varchar(36);not null" json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (SystemSetting) TableName() string {
	return "system_setting"
}

// Tingkat mode pemeliharaan. Posisi berjalan tidak pernah disentuh oleh
// ketiganya; yang berubah hanya penerimaan entry baru (plans/18 §4).
const (
	ModeNormal      = "normal"
	ModeReadOnly    = "read-only"
	ModeMaintenance = "maintenance"
)

// SystemState adalah satu-satunya baris yang menyimpan mode platform.
// Sengaja dipaksa satu baris (id = 1) agar tidak ada dua mode yang berlaku.
type SystemState struct {
	ID          int        `gorm:"primaryKey" json:"id"`
	Mode        string     `gorm:"type:varchar(20);not null;default:'normal'" json:"mode"`
	Reason      string     `gorm:"type:varchar(255);not null;default:''" json:"reason"`
	ExpectedEnd *time.Time `json:"expected_end,omitempty"`
	ChangedBy   string     `gorm:"type:varchar(36)" json:"changed_by"`
	ChangedAt   time.Time  `json:"changed_at"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (SystemState) TableName() string {
	return "system_state"
}

// SupportNote adalah jurnal dukungan per pelanggan, bukan sistem tiket.
// Beda dengan UserAdminNote: catatan di sini juga menampung pesan masuk dari
// Telegram agar riwayat percakapan terlihat utuh (plans/18 §3).
type SupportNote struct {
	Base
	UserID   string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	ActorID  string `gorm:"type:varchar(36);index;not null" json:"actor_id"`
	Channel  string `gorm:"type:varchar(20);not null;default:'telegram'" json:"channel"`
	Body     string `gorm:"type:text;not null" json:"body"`
	IsPinned bool   `gorm:"not null;default:false" json:"is_pinned"`
}

// TableName menetapkan nama tabel tanpa bentuk jamak.
func (SupportNote) TableName() string {
	return "support_note"
}
