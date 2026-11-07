package model

import "time"

// Status akun pengguna. Nilainya dipakai middleware sesi dan Runner.
const (
	UserStatusActive    = "active"
	UserStatusSuspended = "suspended"
	UserStatusBanned    = "banned"
	UserStatusDeleted   = "deleted"
)

// Tier langganan. Premium sengaja tidak ada: paket yang dijual hanya Basic
// dan Plus, sedangkan Free adalah bawaan tanpa langganan.
const (
	TierFree  = "free"
	TierBasic = "basic"
	TierPlus  = "plus"
)

// UserAdminNote adalah catatan internal yang hanya terlihat staf.
// Dipisah dari user agar setiap catatan punya penulis dan waktu.
type UserAdminNote struct {
	Base
	UserID   string `gorm:"type:varchar(36);index;not null" json:"user_id"`
	AuthorID string `gorm:"type:varchar(36);index;not null" json:"author_id"`
	Body     string `gorm:"type:varchar(2000);not null" json:"body"`
}

// TableName menetapkan nama tabel.
func (UserAdminNote) TableName() string {
	return "user_admin_note"
}

// Kategori alasan aksi admin. Kategori dipakai untuk laporan penyalahgunaan
// tanpa harus membaca teks bebas (plans/07 §4.3).
const (
	ReasonToS         = "tos"
	ReasonTrialAbuse  = "trial_abuse"
	ReasonUnpaid      = "unpaid"
	ReasonUserRequest = "user_request"
	ReasonFraud       = "fraud"
	ReasonOther       = "other"
)

// ReasonCategories adalah pilihan kategori yang dikenali sistem.
var ReasonCategories = []string{
	ReasonToS, ReasonTrialAbuse, ReasonUnpaid, ReasonUserRequest, ReasonFraud, ReasonOther,
}

// KnownReasonCategory menandai kategori alasan yang sah.
func KnownReasonCategory(value string) bool {
	for _, item := range ReasonCategories {
		if item == value {
			return true
		}
	}
	return false
}

// AccountFreeze menambahkan jejak waktu dan alasan pada pembekuan akun broker.
type AccountFreezeFields struct {
	FrozenAt     *time.Time `json:"frozen_at"`
	FrozenReason string     `gorm:"type:varchar(255)" json:"frozen_reason"`
}
