package service

import (
	"context"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
)

// UserRepository adalah kebutuhan service atas penyimpanan akun.
// Interface didefinisikan di sisi konsumen agar service tidak bergantung pada
// implementasi GORM dan mudah diganti saat pengujian.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByTelegramChatID(ctx context.Context, chatID string) (*model.User, error)
	UpdateLastLogin(ctx context.Context, id string, at time.Time) error
	Update(ctx context.Context, user *model.User) error
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
	Deactivate(ctx context.Context, id, userID string) error
}

// PositionRepository adalah kebutuhan service atas posisi terbuka user.
type PositionRepository interface {
	FindOpenByUser(ctx context.Context, userID string) ([]model.Position, error)
}
