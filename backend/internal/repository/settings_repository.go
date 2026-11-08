package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
)

// SettingsRepository menyimpan penimpa flag sistem.
//
// Baris di tabel ini murni override; tidak ada nilai bawaan yang disalin agar
// menghapus satu baris langsung mengembalikan nilai environment.
type SettingsRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewSettingsRepository merangkai repository flag sistem.
func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// Load membaca seluruh override dalam satu query.
func (r *SettingsRepository) Load(ctx context.Context) (map[string]string, error) {
	var rows []model.SystemSetting
	if err := r.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca flag sistem: %w", err)
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Value
	}
	return out, nil
}

// Rows membaca override beserta alasan dan pelakunya untuk tampilan admin.
func (r *SettingsRepository) Rows(ctx context.Context) ([]model.SystemSetting, error) {
	var rows []model.SystemSetting
	if err := r.db.WithContext(ctx).Order("key ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca flag sistem: %w", err)
	}
	return rows, nil
}

// Save menulis satu override; kunci yang sudah ada ditimpa.
func (r *SettingsRepository) Save(ctx context.Context, key, value, reason, actorID string) error {
	row := &model.SystemSetting{
		Key:       key,
		Value:     value,
		Reason:    reason,
		UpdatedBy: actorID,
		UpdatedAt: r.now(),
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "reason", "updated_by", "updated_at"}),
	}).Create(row).Error
	if err != nil {
		return fmt.Errorf("menyimpan flag sistem: %w", err)
	}
	return nil
}

// Delete menghapus override sehingga nilai environment berlaku kembali.
func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	if err := r.db.WithContext(ctx).Where("key = ?", key).Delete(&model.SystemSetting{}).Error; err != nil {
		return fmt.Errorf("menghapus override flag: %w", err)
	}
	return nil
}
