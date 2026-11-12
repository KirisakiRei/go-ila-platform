package model_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ila/backend/internal/model"
)

// allModels mendaftar seluruh tabel yang dibuat cmd/migrate.
// Uji ini menangkap tag GORM yang tidak sah (tipe kolom, indeks ganda, kolom
// wajib tanpa default) sebelum menyentuh database sungguhan.
func allModels() []any {
	return []any{
		&model.User{}, &model.BrokerAccount{}, &model.Credential{},
		&model.Subscription{}, &model.Signal{}, &model.Position{},
		&model.AuditLog{}, &model.ApprovalToken{}, &model.OnboardingToken{},
		&model.NotificationLog{}, &model.UserSignal{}, &model.SignalEvent{},
		&model.ExecutionRecord{}, &model.BrokerAccountState{}, &model.DailySummary{},
		&model.EngineState{}, &model.Session{}, &model.Role{}, &model.Permission{},
		&model.RolePermission{}, &model.TwoFactor{}, &model.RecoveryCode{},
		&model.TwoFactorChallenge{}, &model.LegalDocument{}, &model.LegalAcceptance{},
		&model.UserAdminNote{}, &model.Plan{}, &model.PlanFeature{},
		&model.PriceOverride{}, &model.PricingGuard{}, &model.Coupon{},
		&model.CouponRedemption{}, &model.CreditBalance{}, &model.CreditTransaction{},
		&model.CreditCode{}, &model.CreditRedemption{}, &model.SubscriptionRecord{},
		&model.Invoice{}, &model.Payment{}, &model.Refund{},
		&model.SystemSetting{}, &model.SystemState{}, &model.SupportNote{},
		&model.Faq{}, &model.Announcement{}, &model.EmailOutbox{}, &model.EmailToken{},
		&model.Page{}, &model.PageSection{}, &model.PageItem{},
		&model.InviteCode{}, &model.InviteRedemption{},
		&model.BugReport{}, &model.BugReportAttachment{},
	}
}

// TestSkemaDapatDimigrasikan memastikan seluruh model bisa dibuat menjadi tabel.
// SQLite dipakai karena tidak butuh server; tipe kolom yang tidak sah tetap
// terdeteksi di sini, sebelum kegagalan muncul di produksi.
func TestSkemaDapatDimigrasikan(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("membuka sqlite: %v", err)
	}

	models := allModels()
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrasi gagal: %v", err)
	}

	// Setiap model harus benar-benar punya tabel setelah migrasi.
	for _, item := range models {
		if !db.Migrator().HasTable(item) {
			t.Fatalf("tabel untuk %T tidak terbentuk", item)
		}
	}
}

// TestTabelBaruPunyaKolomWajib memeriksa kolom kunci pada tabel Fase I–P,
// karena kolom yang lupa dideklarasikan baru terasa saat fitur dipakai.
func TestTabelBaruPunyaKolomWajib(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("membuka sqlite: %v", err)
	}
	if err := db.AutoMigrate(allModels()...); err != nil {
		t.Fatalf("migrasi gagal: %v", err)
	}

	expectations := []struct {
		value any
		field string
	}{
		{&model.SystemSetting{}, "Value"},
		{&model.SystemSetting{}, "Reason"},
		{&model.SystemState{}, "Mode"},
		{&model.EmailToken{}, "TokenHash"},
		{&model.EmailOutbox{}, "ScheduledAt"},
		{&model.Faq{}, "AnswerMD"},
		{&model.Announcement{}, "Level"},
		{&model.SupportNote{}, "Channel"},
		{&model.User{}, "EmailVerifiedAt"},
		{&model.User{}, "OnboardingStep"},
		{&model.User{}, "RunnerDirtyAt"},
		// Sidik jari token mencegah dua akun berbagi kredensial yang sama.
		{&model.Credential{}, "TokenFingerprint"},
	}

	for _, item := range expectations {
		if !db.Migrator().HasColumn(item.value, item.field) {
			t.Fatalf("%T tidak punya kolom %s", item.value, item.field)
		}
	}
}
