package model

import "gorm.io/gorm"

// Berkas ini adalah satu-satunya rumah hook pengisian ID.
//
// Entitas berkunci alami (credit_balance memakai user_id, broker_account_state
// memakai broker_account_id, email_outbox/email_token punya hook sendiri)
// sengaja tidak didaftarkan di sini.
//
// Entitas yang memakai Base sudah mendapat ID lewat Base.BeforeCreate. Entitas
// lain yang primary key-nya UUID string wajib punya hook sendiri: PostgreSQL
// menolak id kosong sementara SQLite diam-diam menerimanya, jadi kelalaian ini
// baru terlihat saat pindah ke database sungguhan.

func (m *AuditLog) BeforeCreate(_ *gorm.DB) error           { return ensureID(&m.ID) }
func (m *ApprovalToken) BeforeCreate(_ *gorm.DB) error      { return ensureID(&m.ID) }
func (m *NotificationLog) BeforeCreate(_ *gorm.DB) error    { return ensureID(&m.ID) }
func (m *CouponRedemption) BeforeCreate(_ *gorm.DB) error   { return ensureID(&m.ID) }
func (m *CreditRedemption) BeforeCreate(_ *gorm.DB) error   { return ensureID(&m.ID) }
func (m *CreditTransaction) BeforeCreate(_ *gorm.DB) error  { return ensureID(&m.ID) }
func (m *LegalDocument) BeforeCreate(_ *gorm.DB) error      { return ensureID(&m.ID) }
func (m *LegalAcceptance) BeforeCreate(_ *gorm.DB) error    { return ensureID(&m.ID) }
func (m *Role) BeforeCreate(_ *gorm.DB) error               { return ensureID(&m.ID) }
func (m *SignalEvent) BeforeCreate(_ *gorm.DB) error        { return ensureID(&m.ID) }
func (m *ExecutionRecord) BeforeCreate(_ *gorm.DB) error    { return ensureID(&m.ID) }
func (m *Session) BeforeCreate(_ *gorm.DB) error            { return ensureID(&m.ID) }
func (m *SubscriptionRecord) BeforeCreate(_ *gorm.DB) error { return ensureID(&m.ID) }
func (m *Invoice) BeforeCreate(_ *gorm.DB) error            { return ensureID(&m.ID) }
func (m *Payment) BeforeCreate(_ *gorm.DB) error            { return ensureID(&m.ID) }
func (m *Refund) BeforeCreate(_ *gorm.DB) error             { return ensureID(&m.ID) }
func (m *TwoFactor) BeforeCreate(_ *gorm.DB) error          { return ensureID(&m.ID) }
func (m *RecoveryCode) BeforeCreate(_ *gorm.DB) error       { return ensureID(&m.ID) }
func (m *TwoFactorChallenge) BeforeCreate(_ *gorm.DB) error { return ensureID(&m.ID) }
