package model

import "time"

// Role mengelompokkan permission. Akun admin memakai role untuk menentukan
// tindakan yang boleh dilakukan (plans/04 §3).
type Role struct {
	ID          string `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Code        string `gorm:"type:varchar(40);uniqueIndex;not null" json:"code"`
	Name        string `gorm:"type:varchar(80);not null" json:"name"`
	Description string `gorm:"type:varchar(255)" json:"description"`
	// IsSystem menandai role bawaan: tidak bisa dihapus atau diubah permissionnya.
	IsSystem  bool      `gorm:"not null;default:false" json:"is_system"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel.
func (Role) TableName() string {
	return "role"
}

// Permission adalah satu kewenangan dengan format grup.aksi.
type Permission struct {
	Key   string `gorm:"primaryKey;type:varchar(60)" json:"key"`
	Group string `gorm:"type:varchar(30);index;not null" json:"group"`
	Label string `gorm:"type:varchar(120);not null" json:"label"`
	// RiskLabel menjelaskan dampak penyalahgunaan: rendah, sedang, tinggi.
	RiskLabel string `gorm:"type:varchar(20)" json:"risk_label"`
}

// TableName menetapkan nama tabel.
func (Permission) TableName() string {
	return "permission"
}

// RolePermission menghubungkan role dengan permission.
type RolePermission struct {
	RoleID       string `gorm:"type:varchar(36);primaryKey" json:"role_id"`
	PermissionID string `gorm:"type:varchar(60);primaryKey" json:"permission_key"`
}

// TableName menetapkan nama tabel.
func (RolePermission) TableName() string {
	return "role_permission"
}
