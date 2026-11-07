package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

type signalRepository struct {
	db *gorm.DB
}

// NewSignalRepository membuat repository sinyal.
func NewSignalRepository(db *gorm.DB) SignalRepository {
	return &signalRepository{db: db}
}

// Upsert menulis sinyal; signal_id unik sehingga pengiriman ulang tidak menggandakan data.
func (r *signalRepository) Upsert(ctx context.Context, signal *model.Signal) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "signal_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "summary", "next_action", "risk_reward"}),
	}).Create(signal).Error
	if err != nil {
		return fmt.Errorf("menyimpan sinyal: %w", err)
	}
	return nil
}

func (r *signalRepository) FindBySignalID(ctx context.Context, signalID string) (*model.Signal, error) {
	var signal model.Signal
	err := r.db.WithContext(ctx).Where("signal_id = ?", signalID).First(&signal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca sinyal: %w", err)
	}
	return &signal, nil
}

func (r *signalRepository) UpdateStatus(ctx context.Context, signalID, status, reason string) error {
	updates := map[string]any{"status": status}
	if reason != "" {
		updates["cancel_reason"] = reason
	}

	res := r.db.WithContext(ctx).Model(&model.Signal{}).
		Where("signal_id = ?", signalID).
		Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("memperbarui status sinyal: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

type positionRepository struct {
	db *gorm.DB
}

// NewPositionRepository membuat repository posisi.
func NewPositionRepository(db *gorm.DB) PositionRepository {
	return &positionRepository{db: db}
}

func (r *positionRepository) Create(ctx context.Context, pos *model.Position) error {
	if err := r.db.WithContext(ctx).Create(pos).Error; err != nil {
		return fmt.Errorf("mencatat posisi: %w", err)
	}
	return nil
}

func (r *positionRepository) FindOpenByUser(ctx context.Context, userID string) ([]model.Position, error) {
	var positions []model.Position
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, "OPEN").
		Order("opened_at ASC").
		Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("membaca posisi terbuka: %w", err)
	}
	return positions, nil
}

func (r *positionRepository) FindByTicket(ctx context.Context, ticket string) (*model.Position, error) {
	var position model.Position
	err := r.db.WithContext(ctx).Where("ticket_number = ?", ticket).First(&position).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca posisi: %w", err)
	}
	return &position, nil
}

// Close hanya menutup posisi yang masih OPEN agar tidak ada penutupan ganda.
func (r *positionRepository) Close(ctx context.Context, id string, exitPrice, pnl, outcomeR float64, status string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.Position{}).
		Where("id = ? AND status = ?", id, "OPEN").
		Updates(map[string]any{
			"exit_price":  exitPrice,
			"profit_loss": pnl,
			"outcome_r":   outcomeR,
			"status":      status,
			"closed_at":   now,
		})
	if res.Error != nil {
		return fmt.Errorf("menutup posisi: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrConflict
	}
	return nil
}
