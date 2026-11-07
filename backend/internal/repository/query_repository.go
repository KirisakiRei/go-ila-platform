package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// Pembacaan daftar untuk dashboard. Semua kueri dibatasi userID agar tidak ada
// kebocoran antar akun (plans/12).

// UserSignalLister membaca sinyal pada simbol yang dipantau user.
type UserSignalLister interface {
	ListForUser(ctx context.Context, userID string, since time.Time, limit int) ([]model.Signal, error)
}

// UserPositionLister membaca posisi user.
type UserPositionLister interface {
	ListByUser(ctx context.Context, userID string, limit int) ([]model.Position, error)
}

// UserAccountStateLister membaca telemetry akun broker user.
type UserAccountStateLister interface {
	ListByUser(ctx context.Context, userID string) ([]model.BrokerAccountState, error)
}

// ListForUser mengembalikan sinyal pada simbol yang dipantau user.
func (r *signalRepository) ListForUser(ctx context.Context, userID string, since time.Time, limit int) ([]model.Signal, error) {
	var signals []model.Signal

	err := r.db.WithContext(ctx).
		Model(&model.Signal{}).
		Joins("JOIN subscription s ON s.symbol = signal.symbol AND s.user_id = ? AND s.is_active = ?", userID, true).
		Where("signal.published_at >= ?", since).
		Distinct("signal.*").
		Order("signal.published_at DESC").
		Limit(limit).
		Find(&signals).Error
	if err != nil {
		return nil, fmt.Errorf("membaca sinyal user: %w", err)
	}
	return signals, nil
}

// ListByUser mengembalikan posisi user, terbaru lebih dulu.
func (r *positionRepository) ListByUser(ctx context.Context, userID string, limit int) ([]model.Position, error) {
	var positions []model.Position

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("opened_at DESC").
		Limit(limit).
		Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("membaca posisi user: %w", err)
	}
	return positions, nil
}

type brokerAccountStateRepository struct {
	db *gorm.DB
}

// NewBrokerAccountStateRepository membuat pembaca telemetry akun.
func NewBrokerAccountStateRepository(db *gorm.DB) BrokerAccountStateRepository {
	return &brokerAccountStateRepository{db: db}
}

// ListByUser mengembalikan telemetry terakhir seluruh akun broker user.
func (r *brokerAccountStateRepository) ListByUser(ctx context.Context, userID string) ([]model.BrokerAccountState, error) {
	var states []model.BrokerAccountState

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&states).Error
	if err != nil {
		return nil, fmt.Errorf("membaca status akun: %w", err)
	}
	return states, nil
}
