package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// idHook mengisi ID untuk model yang tidak memakai Base.
// Tanpa ini, insert gagal karena kolom id NOT NULL (ditemukan saat uji integrasi).

// BeforeCreate mengisi ID AuditLog bila kosong.
func (a *AuditLog) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	return nil
}

// BeforeCreate mengisi ID NotificationLog bila kosong.
func (n *NotificationLog) BeforeCreate(_ *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	return nil
}

// BeforeCreate mengisi ID SignalEvent bila kosong.
func (e *SignalEvent) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}

// BeforeCreate mengisi ID ExecutionRecord bila kosong.
func (e *ExecutionRecord) BeforeCreate(_ *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	return nil
}
