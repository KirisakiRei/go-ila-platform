package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base dipakai bersama seluruh entitas: ID string UUID, timestamp, dan soft delete.
type Base struct {
	ID        string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate mengisi ID bila belum ada.
// Ini satu-satunya urusan pemetaan data; logika bisnis tetap di service layer.
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}
