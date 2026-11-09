package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/internal/pagination"
)

// AdminSignalFilter menyaring daftar sinyal dari panel admin.
type AdminSignalFilter struct {
	Symbol   string
	Status   string
	Strategy string
	From     *time.Time
	To       *time.Time
}

// AdminSignalRow adalah satu sinyal beserta ringkasan penerimanya.
type AdminSignalRow struct {
	model.Signal
	// Recipients adalah jumlah user_signal yang lahir dari sinyal ini.
	Recipients int64 `json:"recipients"`
	// Executed adalah jumlah user_signal yang berakhir CONSUMED.
	Executed int64 `json:"executed"`
}

// AdminSignalStore membaca sinyal lintas pengguna.
//
// Engine tetap buta terhadap identitas: pembaca ini hanya dipakai panel admin
// untuk menjawab "sinyal ini diterima siapa saja".
type AdminSignalStore interface {
	List(ctx context.Context, filter AdminSignalFilter, p pagination.Params) ([]AdminSignalRow, error)
	Count(ctx context.Context, filter AdminSignalFilter) (int64, error)
	Find(ctx context.Context, signalID string) (*AdminSignalRow, error)
	Trace(ctx context.Context, signalID string) (*AdminSignalTrace, error)
}

// AdminSignalTrace merangkum perjalanan satu sinyal.
type AdminSignalTrace struct {
	// Signal adalah baris sinyal beserta ringkasan penerimanya.
	Signal AdminSignalRow
	// Recipients adalah nasib sinyal per user.
	Recipients []AdminSignalRecipient
	// Events adalah riwayat perubahan status.
	Events []model.SignalEvent
	// Executions adalah klaim eksekusi per akun broker.
	Executions []model.ExecutionRecord
	// Positions adalah posisi yang lahir dari sinyal ini.
	Positions []model.Position
}

// AdminSignalRecipient adalah nasib sinyal pada satu user.
type AdminSignalRecipient struct {
	UserID          string     `json:"user_id"`
	UserName        string     `json:"user_name"`
	UserEmail       string     `json:"user_email"`
	BrokerAccountID string     `json:"broker_account_id"`
	Status          string     `json:"status"`
	RejectReason    string     `json:"reject_reason"`
	PositionID      string     `json:"position_id"`
	DecidedAt       *time.Time `json:"decided_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type adminSignalRepository struct {
	db *gorm.DB
}

// NewAdminSignalRepository membuat pembaca sinyal untuk panel admin.
func NewAdminSignalRepository(db *gorm.DB) *adminSignalRepository {
	return &adminSignalRepository{db: db}
}

func (r *adminSignalRepository) base(ctx context.Context, filter AdminSignalFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Model(&model.Signal{})
	if filter.Symbol != "" {
		q = q.Where("symbol = ?", strings.ToUpper(strings.TrimSpace(filter.Symbol)))
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Strategy != "" {
		q = q.Where("strategy_version = ?", filter.Strategy)
	}
	if filter.From != nil {
		q = q.Where("published_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("published_at < ?", *filter.To)
	}
	return q
}

func (r *adminSignalRepository) List(ctx context.Context, filter AdminSignalFilter, p pagination.Params) ([]AdminSignalRow, error) {
	var rows []AdminSignalRow
	err := r.base(ctx, filter).
		Select(`signal.*,
			(SELECT COUNT(*) FROM user_signal us WHERE us.signal_id = signal.signal_id AND us.deleted_at IS NULL) AS recipients,
			(SELECT COUNT(*) FROM user_signal us WHERE us.signal_id = signal.signal_id AND us.status = ? AND us.deleted_at IS NULL) AS executed`,
			model.SignalStatusConsumed).
		Order("published_at DESC").
		Limit(p.PerPage).
		Offset(p.Offset()).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca sinyal admin: %w", err)
	}
	return rows, nil
}

func (r *adminSignalRepository) Count(ctx context.Context, filter AdminSignalFilter) (int64, error) {
	var total int64
	if err := r.base(ctx, filter).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("menghitung sinyal admin: %w", err)
	}
	return total, nil
}

func (r *adminSignalRepository) Find(ctx context.Context, signalID string) (*AdminSignalRow, error) {
	var row AdminSignalRow
	err := r.db.WithContext(ctx).Model(&model.Signal{}).
		Select(`signal.*,
			(SELECT COUNT(*) FROM user_signal us WHERE us.signal_id = signal.signal_id AND us.deleted_at IS NULL) AS recipients,
			(SELECT COUNT(*) FROM user_signal us WHERE us.signal_id = signal.signal_id AND us.status = ? AND us.deleted_at IS NULL) AS executed`,
			model.SignalStatusConsumed).
		Where("signal_id = ?", signalID).
		Limit(1).
		Find(&row).Error
	if err != nil {
		return nil, fmt.Errorf("membaca sinyal: %w", err)
	}
	if row.ID == "" {
		return nil, fmt.Errorf("sinyal %s tidak ditemukan", signalID)
	}
	return &row, nil
}

func (r *adminSignalRepository) Trace(ctx context.Context, signalID string) (*AdminSignalTrace, error) {
	found, err := r.Find(ctx, signalID)
	if err != nil {
		return nil, err
	}

	var recipients []AdminSignalRecipient
	err = r.db.WithContext(ctx).
		Table("user_signal AS us").
		Select(`us.user_id, COALESCE(u.name, '') AS user_name, COALESCE(u.email, '') AS user_email,
			COALESCE(us.broker_account_id, '') AS broker_account_id, us.status,
			COALESCE(us.reject_reason, '') AS reject_reason, COALESCE(us.position_id, '') AS position_id,
			us.decided_at, us.created_at`).
		Joins(`LEFT JOIN "user" AS u ON u.id = us.user_id`).
		Where("us.signal_id = ? AND us.deleted_at IS NULL", signalID).
		Order("us.created_at ASC").
		Scan(&recipients).Error
	if err != nil {
		return nil, fmt.Errorf("membaca penerima sinyal: %w", err)
	}

	var events []model.SignalEvent
	if err := r.db.WithContext(ctx).
		Where("signal_id = ?", signalID).
		Order("created_at ASC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("membaca riwayat sinyal: %w", err)
	}

	var executions []model.ExecutionRecord
	if err := r.db.WithContext(ctx).
		Where("signal_id = ?", signalID).
		Order("created_at ASC").
		Find(&executions).Error; err != nil {
		return nil, fmt.Errorf("membaca eksekusi sinyal: %w", err)
	}

	var positions []model.Position
	if err := r.db.WithContext(ctx).
		Where("source_signal_id = ?", signalID).
		Order("opened_at ASC").
		Find(&positions).Error; err != nil {
		return nil, fmt.Errorf("membaca posisi sinyal: %w", err)
	}

	return &AdminSignalTrace{
		Signal:     *found,
		Recipients: recipients,
		Events:     events,
		Executions: executions,
		Positions:  positions,
	}, nil
}
