package model

import "time"

// ManualJournalReview is a durable, replaceable cache; it contains no journal prose.
type ManualJournalReview struct {
	UserID             string    `gorm:"primaryKey;type:varchar(36)" json:"-"`
	AccountID          string    `gorm:"primaryKey;type:varchar(36)" json:"account_id"`
	Date               string    `gorm:"primaryKey;type:varchar(10)" json:"date"`
	SettingsGeneration int64     `gorm:"not null;default:0" json:"-"`
	Revision           string    `gorm:"type:varchar(64);not null" json:"revision"`
	Status             string    `gorm:"type:varchar(16);not null" json:"status"`
	Text               string    `gorm:"type:text" json:"text"`
	Source             string    `gorm:"type:varchar(16)" json:"source"`
	Claim              string    `gorm:"type:varchar(36)" json:"-"`
	LeaseUntil         time.Time `json:"-"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (ManualJournalReview) TableName() string { return "manual_journal_reviews" }

// An empty AccountID denotes the user-wide preference; either opt-out disables AI.
type ManualJournalReviewSetting struct {
	UserID     string    `gorm:"primaryKey;type:varchar(36)" json:"-"`
	AccountID  string    `gorm:"primaryKey;type:varchar(36)" json:"account_id"`
	Generation int64     `gorm:"not null;default:0" json:"-"`
	AIEnabled  bool      `gorm:"not null" json:"ai_enabled"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (ManualJournalReviewSetting) TableName() string { return "manual_journal_review_settings" }
