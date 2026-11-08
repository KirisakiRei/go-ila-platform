package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
)

// SystemRepository menyimpan mode platform dan menghitung ringkasan operasional.
type SystemRepository struct {
	db  *gorm.DB
	now func() time.Time
}

// NewSystemRepository merangkai repository sistem.
func NewSystemRepository(db *gorm.DB) *SystemRepository {
	return &SystemRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

// State membaca mode platform. Baris yang belum pernah dibuat dianggap normal,
// sehingga sistem baru berjalan tanpa penyiapan tambahan.
func (r *SystemRepository) State(ctx context.Context) (*model.SystemState, error) {
	var row model.SystemState
	err := r.db.WithContext(ctx).Where("id = ?", 1).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.SystemState{ID: 1, Mode: model.ModeNormal}, nil
		}
		return nil, fmt.Errorf("membaca mode sistem: %w", err)
	}
	return &row, nil
}

// SaveState menulis satu-satunya baris mode platform.
func (r *SystemRepository) SaveState(ctx context.Context, state *model.SystemState) error {
	state.ID = 1
	if state.ChangedAt.IsZero() {
		state.ChangedAt = r.now()
	}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"mode", "reason", "expected_end", "changed_by", "changed_at",
		}),
	}).Create(state).Error
	if err != nil {
		return fmt.Errorf("menyimpan mode sistem: %w", err)
	}
	return nil
}

// SupportNotes membaca jurnal dukungan satu pelanggan, terbaru lebih dulu.
// Dibatasi agar riwayat panjang tidak menarik seluruh tabel ke memori.
func (r *SystemRepository) SupportNotes(ctx context.Context, userID string, limit int) ([]model.SupportNote, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []model.SupportNote
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_pinned DESC, created_at DESC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca catatan dukungan: %w", err)
	}
	return rows, nil
}

// AddSupportNote menyimpan satu catatan dukungan.
func (r *SystemRepository) AddSupportNote(ctx context.Context, note *model.SupportNote) error {
	if err := r.db.WithContext(ctx).Create(note).Error; err != nil {
		return fmt.Errorf("menyimpan catatan dukungan: %w", err)
	}
	return nil
}

// BrokerHealth menghitung akun broker yang sehat dan bermasalah.
// Query dijalankan di database agar tetap ringan pada VPS kecil.
func (r *SystemRepository) BrokerHealth(ctx context.Context) (total, connected, errored, frozen int, err error) {
	type row struct {
		Total     int
		Connected int
		Errored   int
		Frozen    int
	}
	var out row
	query := `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE s.connected) AS connected,
			COUNT(*) FILTER (WHERE NOT s.connected AND s.last_error <> '') AS errored,
			COUNT(*) FILTER (WHERE a.is_frozen) AS frozen
		FROM broker_account a
		LEFT JOIN broker_account_state s ON s.broker_account_id = a.id
		WHERE a.deleted_at IS NULL AND a.is_active`
	if err := r.db.WithContext(ctx).Raw(query).Scan(&out).Error; err != nil {
		return 0, 0, 0, 0, fmt.Errorf("membaca kesehatan broker: %w", err)
	}
	return out.Total, out.Connected, out.Errored, out.Frozen, nil
}

// CountActiveAnnouncements menghitung pengumuman yang sedang tayang.
func (r *SystemRepository) CountActiveAnnouncements(ctx context.Context) (int, error) {
	var count int64
	now := r.now()
	err := r.db.WithContext(ctx).Model(&model.Announcement{}).
		Where("is_active AND starts_at <= ? AND (ends_at IS NULL OR ends_at > ?)", now, now).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung pengumuman: %w", err)
	}
	return int(count), nil
}
