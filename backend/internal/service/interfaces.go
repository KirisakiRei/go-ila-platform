package service

import (
	"context"
	"time"

	"ila/backend/internal/model"
)

// UserRepository adalah kebutuhan service atas penyimpanan akun.
// Interface didefinisikan di sisi konsumen agar service tidak bergantung pada
// implementasi GORM dan mudah diganti saat pengujian.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	// FindByIdentifier mencocokkan email atau username untuk satu kolom masuk.
	FindByIdentifier(ctx context.Context, identifier string) (*model.User, error)
	FindByTelegramChatID(ctx context.Context, chatID string) (*model.User, error)
	UpdateLastLogin(ctx context.Context, id string, at time.Time) error
	Update(ctx context.Context, user *model.User) error
	// RevokeSessions mencabut seluruh sesi aktif akun ini.
	RevokeSessions(ctx context.Context, id string) error
	// UpdatePassword mengganti hash dan mencabut sesi dalam satu operasi.
	UpdatePassword(ctx context.Context, id, passwordHash string) error
}

// AuditRepository adalah kebutuhan service atas pencatatan audit.
type AuditRepository interface {
	Record(ctx context.Context, entry *model.AuditLog) error
}

// BrokerAccountRepository adalah kebutuhan service atas akun broker.
type BrokerAccountRepository interface {
	Create(ctx context.Context, account *model.BrokerAccount) error
	FindByID(ctx context.Context, id string) (*model.BrokerAccount, error)
	ListByUser(ctx context.Context, userID string) ([]model.BrokerAccount, error)
	CountActiveByUser(ctx context.Context, userID string) (int64, error)
	Update(ctx context.Context, account *model.BrokerAccount) error
}

// SubscriptionRepository adalah kebutuhan service atas pilihan simbol dan strategi.
type SubscriptionRepository interface {
	Create(ctx context.Context, sub *model.Subscription) error
	ListActiveByUser(ctx context.Context, userID string) ([]model.Subscription, error)
	ListByUser(ctx context.Context, userID string) ([]model.Subscription, error)
	Deactivate(ctx context.Context, id, userID string) error
}

// CredentialRepository menyimpan blob kredensial terenkripsi.
type CredentialRepository interface {
	Upsert(ctx context.Context, credential *model.Credential) error
	FindByAccount(ctx context.Context, accountID string) (*model.Credential, error)
	DeleteByAccount(ctx context.Context, accountID string) error
}

// BrokerAccountStateLister membaca telemetry akun yang ditulis Runner.
type BrokerAccountStateLister interface {
	ListByUser(ctx context.Context, userID string) ([]model.BrokerAccountState, error)
}

// PositionRepository adalah kebutuhan service atas posisi terbuka user.
type PositionRepository interface {
	FindOpenByUser(ctx context.Context, userID string) ([]model.Position, error)
}
