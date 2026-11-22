package model

import "time"

// Status permohonan undangan akses dari beranda publik.
const (
	InviteRequestPending  = "pending"
	InviteRequestApproved = "approved"
	InviteRequestRejected = "rejected"
)

// InviteRequest menyimpan permohonan akses dari calon pengguna selama platform
// masih invite-only. Tidak pernah memuat kata sandi atau kredensial broker.
type InviteRequest struct {
	Base
	Name       string     `gorm:"type:varchar(100);not null" json:"name"`
	Email      string     `gorm:"type:varchar(150);index;not null" json:"email"`
	Contact    string     `gorm:"type:varchar(100);not null;default:''" json:"contact"`
	Note       string     `gorm:"type:varchar(500);not null;default:''" json:"note"`
	Status     string     `gorm:"type:varchar(20);index;not null;default:'pending'" json:"status"`
	InviteID   string     `gorm:"type:varchar(36);not null;default:''" json:"invite_id"`
	ReviewedBy string     `gorm:"type:varchar(36);not null;default:''" json:"reviewed_by"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote string     `gorm:"type:varchar(255);not null;default:''" json:"review_note"`
}

// TableName menetapkan nama tabel.
func (InviteRequest) TableName() string {
	return "invite_request"
}
