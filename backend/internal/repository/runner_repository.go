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

// RunnerRepository adalah seluruh akses data yang dibutuhkan proses Runner.
// Interface dideklarasikan di sisi pemakai (runner), implementasinya di sini.
type RunnerRepository interface {
	// Sinyal global.
	UpsertSignal(ctx context.Context, signal *model.Signal) error
	FindSignal(ctx context.Context, signalID string) (*model.Signal, error)
	UpdateSignalStatus(ctx context.Context, signalID, status, reason string) error
	MarkSuperseded(ctx context.Context, signalID, newSignalID, reason string) error
	TouchSignal(ctx context.Context, signalID string, seenAt time.Time) error

	// Status sinyal per user.
	UpsertUserSignal(ctx context.Context, us *model.UserSignal) error
	FindUserSignal(ctx context.Context, userID, signalID string) (*model.UserSignal, error)
	FindUserSignalByPosition(ctx context.Context, positionID string) (*model.UserSignal, error)
	RecordSignalEvent(ctx context.Context, event *model.SignalEvent) error

	// Posisi.
	CreatePosition(ctx context.Context, pos *model.Position) error
	FindPosition(ctx context.Context, id string) (*model.Position, error)
	FindOpenPositionBySignal(ctx context.Context, userID, signalID string) (*model.Position, error)
	ListOpenPositions(ctx context.Context) ([]model.Position, error)
	ListOpenPositionsByUser(ctx context.Context, userID string) ([]model.Position, error)
	ClosePosition(ctx context.Context, pos *model.Position) error

	// Idempotensi eksekusi.
	ClaimExecution(ctx context.Context, record *model.ExecutionRecord) (bool, error)
	CompleteExecution(ctx context.Context, key string, positionID string) error
	FailExecution(ctx context.Context, key, reason string) error
	FindExecution(ctx context.Context, key string) (*model.ExecutionRecord, error)
	// ReclaimStaleExecution mengambil alih klaim yang tertinggal karena proses
	// berhenti sebelum order terkirim. Nilai false berarti klaim sudah selesai.
	ReclaimStaleExecution(ctx context.Context, key string, staleBefore time.Time) (bool, error)
	ListStaleExecutions(ctx context.Context, staleBefore time.Time) ([]model.ExecutionRecord, error)

	// Kuota harian.
	GetDailySummary(ctx context.Context, userID, day string) (*model.DailySummary, error)
	IncrementDailyTrade(ctx context.Context, userID, day string) error
	RecordDailyOutcome(ctx context.Context, userID, day string, won bool) error
	LockDaily(ctx context.Context, userID, day, reason string) error

	// Akun & telemetry.
	ListBrokerAccountsByUser(ctx context.Context, userID string) ([]model.BrokerAccount, error)
	FindBrokerAccount(ctx context.Context, accountID, userID string) (*model.BrokerAccount, error)
	FindUserByID(ctx context.Context, userID string) (*model.User, error)
	ListActiveBrokerAccounts(ctx context.Context) ([]model.BrokerAccount, error)
	UpdateBrokerAccount(ctx context.Context, account *model.BrokerAccount) error
	UpsertAccountState(ctx context.Context, state *model.BrokerAccountState) error
	FindCredential(ctx context.Context, accountID string) (*model.Credential, error)

	// Subscription: kebutuhan engine per (symbol, strategy).
	ListActiveSubscriptions(ctx context.Context) ([]model.Subscription, error)
	ListUsersBySymbol(ctx context.Context, symbol string) ([]model.User, error)

	// Status engine.
	UpsertEngineState(ctx context.Context, state *model.EngineState) error
	ListEngines(ctx context.Context) ([]model.EngineState, error)

	// Kuota harian untuk dashboard.
	FindDailySummary(ctx context.Context, userID, day string) (*model.DailySummary, error)

	// Token konfirmasi entry (deep-link).
	CreateApprovalToken(ctx context.Context, token *model.ApprovalToken) error
}

type runnerRepository struct {
	db *gorm.DB
}

// NewRunnerRepository membuat repository untuk proses Runner.
func NewRunnerRepository(db *gorm.DB) RunnerRepository {
	return &runnerRepository{db: db}
}

func (r *runnerRepository) UpsertSignal(ctx context.Context, signal *model.Signal) error {
	// Sinyal yang sudah ada tidak ditimpa: status lokal (ACKED/CONSUMED)
	// lebih baru daripada payload PUBLISH yang mungkin terkirim ulang.
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "signal_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"summary", "next_action", "expires_at", "updated_at",
			}),
		}).
		Create(signal).Error
	if err != nil {
		return fmt.Errorf("menyimpan sinyal: %w", err)
	}
	return nil
}

func (r *runnerRepository) FindSignal(ctx context.Context, signalID string) (*model.Signal, error) {
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

func (r *runnerRepository) UpdateSignalStatus(ctx context.Context, signalID, status, reason string) error {
	res := r.db.WithContext(ctx).Model(&model.Signal{}).
		Where("signal_id = ?", signalID).
		Updates(map[string]any{
			"status":        status,
			"cancel_reason": reason,
		})
	if res.Error != nil {
		return fmt.Errorf("memperbarui status sinyal: %w", res.Error)
	}
	return nil
}

func (r *runnerRepository) MarkSuperseded(ctx context.Context, signalID, newSignalID, reason string) error {
	res := r.db.WithContext(ctx).Model(&model.Signal{}).
		Where("signal_id = ?", signalID).
		Updates(map[string]any{
			"status":        "SUPERSEDED",
			"superseded_by": newSignalID,
			"cancel_reason": reason,
		})
	if res.Error != nil {
		return fmt.Errorf("menandai sinyal digantikan: %w", res.Error)
	}
	return nil
}

// TouchSignal mencatat kapan terakhir sinyal terlihat oleh runner.
// Dipakai pembersih data untuk sinyal yang batal/kedaluwarsa.
func (r *runnerRepository) TouchSignal(ctx context.Context, signalID string, seenAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&model.Signal{}).
		Where("signal_id = ?", signalID).
		Update("updated_at", seenAt)
	if res.Error != nil {
		return fmt.Errorf("memperbarui waktu sinyal: %w", res.Error)
	}
	return nil
}

func (r *runnerRepository) UpsertUserSignal(ctx context.Context, us *model.UserSignal) error {
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "signal_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"status", "reject_reason", "not_before",
				"decided_at", "position_id", "updated_at",
			}),
		}).
		Create(us).Error
	if err != nil {
		return fmt.Errorf("menyimpan status sinyal user: %w", err)
	}
	return nil
}

func (r *runnerRepository) FindUserSignal(ctx context.Context, userID, signalID string) (*model.UserSignal, error) {
	var us model.UserSignal
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND signal_id = ?", userID, signalID).
		First(&us).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca status sinyal user: %w", err)
	}
	return &us, nil
}

func (r *runnerRepository) FindUserSignalByPosition(ctx context.Context, positionID string) (*model.UserSignal, error) {
	var us model.UserSignal
	err := r.db.WithContext(ctx).Where("position_id = ?", positionID).First(&us).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca status sinyal dari posisi: %w", err)
	}
	return &us, nil
}

func (r *runnerRepository) RecordSignalEvent(ctx context.Context, event *model.SignalEvent) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("mencatat peristiwa sinyal: %w", err)
	}
	return nil
}

func (r *runnerRepository) CreatePosition(ctx context.Context, pos *model.Position) error {
	if err := r.db.WithContext(ctx).Create(pos).Error; err != nil {
		return fmt.Errorf("mencatat posisi: %w", err)
	}
	return nil
}

func (r *runnerRepository) FindPosition(ctx context.Context, id string) (*model.Position, error) {
	var pos model.Position
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&pos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca posisi: %w", err)
	}
	return &pos, nil
}

func (r *runnerRepository) FindOpenPositionBySignal(ctx context.Context, userID, signalID string) (*model.Position, error) {
	var pos model.Position
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND source_signal_id = ? AND status = ?", userID, signalID, model.PositionStatusOpen).
		First(&pos).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca posisi dari sinyal: %w", err)
	}
	return &pos, nil
}

func (r *runnerRepository) ListOpenPositions(ctx context.Context) ([]model.Position, error) {
	var positions []model.Position
	err := r.db.WithContext(ctx).
		Where("status = ?", model.PositionStatusOpen).
		Order("opened_at ASC").
		Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("membaca posisi terbuka: %w", err)
	}
	return positions, nil
}

func (r *runnerRepository) ListOpenPositionsByUser(ctx context.Context, userID string) ([]model.Position, error) {
	var positions []model.Position
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.PositionStatusOpen).
		Order("opened_at ASC").
		Find(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("membaca posisi terbuka user: %w", err)
	}
	return positions, nil
}

func (r *runnerRepository) ClosePosition(ctx context.Context, pos *model.Position) error {
	if err := r.db.WithContext(ctx).Save(pos).Error; err != nil {
		return fmt.Errorf("menutup posisi: %w", err)
	}
	return nil
}

// ClaimExecution mendaftarkan kunci idempotensi. Nilai balik false berarti
// kunci sudah pernah dipakai, sehingga eksekusi harus dibatalkan.
func (r *runnerRepository) ClaimExecution(ctx context.Context, record *model.ExecutionRecord) (bool, error) {
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(record)
	if res.Error != nil {
		return false, fmt.Errorf("mengklaim eksekusi: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

func (r *runnerRepository) CompleteExecution(ctx context.Context, key string, positionID string) error {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&model.ExecutionRecord{}).
		Where("idempotency_key = ?", key).
		Updates(map[string]any{
			"status":       "COMPLETED",
			"position_id":  positionID,
			"completed_at": now,
		})
	if res.Error != nil {
		return fmt.Errorf("menandai eksekusi selesai: %w", res.Error)
	}
	return nil
}

func (r *runnerRepository) FailExecution(ctx context.Context, key, reason string) error {
	res := r.db.WithContext(ctx).Model(&model.ExecutionRecord{}).
		Where("idempotency_key = ?", key).
		Updates(map[string]any{
			"status":         "FAILED",
			"failure_reason": reason,
		})
	if res.Error != nil {
		return fmt.Errorf("menandai eksekusi gagal: %w", res.Error)
	}
	return nil
}

// ReclaimStaleExecution memindahkan klaim lama menjadi FAILED agar sinyal yang
// sama bisa diproses ulang. Hanya baris yang masih CLAIMED dan sudah basi yang
// diambil, sehingga eksekusi yang sedang berjalan tidak tersentuh.
func (r *runnerRepository) ReclaimStaleExecution(ctx context.Context, key string, staleBefore time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.ExecutionRecord{}).
		Where("idempotency_key = ? AND status = ? AND created_at < ?", key, "CLAIMED", staleBefore).
		Updates(map[string]any{
			"status":         "FAILED",
			"failure_reason": "klaim kedaluwarsa: proses berhenti sebelum order terkirim",
		})
	if res.Error != nil {
		return false, fmt.Errorf("mengambil alih klaim eksekusi: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ListStaleExecutions membaca seluruh klaim yang tertinggal, untuk dilaporkan
// ke admin saat rekonsiliasi.
func (r *runnerRepository) ListStaleExecutions(ctx context.Context, staleBefore time.Time) ([]model.ExecutionRecord, error) {
	var rows []model.ExecutionRecord
	err := r.db.WithContext(ctx).
		Where("status = ? AND created_at < ?", "CLAIMED", staleBefore).
		Order("created_at ASC").
		Limit(200).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca klaim tertinggal: %w", err)
	}
	return rows, nil
}

func (r *runnerRepository) FindExecution(ctx context.Context, key string) (*model.ExecutionRecord, error) {
	var record model.ExecutionRecord
	err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca catatan eksekusi: %w", err)
	}
	return &record, nil
}

func (r *runnerRepository) GetDailySummary(ctx context.Context, userID, day string) (*model.DailySummary, error) {
	var summary model.DailySummary
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND day = ?", userID, day).
		First(&summary).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Hari belum memiliki aktivitas: kembalikan nol, bukan error.
			return &model.DailySummary{UserID: userID, Day: day}, nil
		}
		return nil, fmt.Errorf("membaca ringkasan harian: %w", err)
	}
	return &summary, nil
}

func (r *runnerRepository) IncrementDailyTrade(ctx context.Context, userID, day string) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "day"}},
			DoUpdates: clause.Assignments(map[string]any{
				"total_trades": gorm.Expr("daily_summary.total_trades + 1"),
				"updated_at":   now,
			}),
		}).
		Create(&model.DailySummary{
			UserID:      userID,
			Day:         day,
			TotalTrades: 1,
			UpdatedAt:   now,
		}).Error
	if err != nil {
		return fmt.Errorf("menambah hitungan trade harian: %w", err)
	}
	return nil
}

func (r *runnerRepository) RecordDailyOutcome(ctx context.Context, userID, day string, won bool) error {
	now := time.Now().UTC()
	column := "loss_count"
	if won {
		column = "win_count"
	}

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "day"}},
			DoUpdates: clause.Assignments(map[string]any{
				column:       gorm.Expr(fmt.Sprintf("daily_summary.%s + 1", column)),
				"updated_at": now,
			}),
		}).
		Create(&model.DailySummary{
			UserID:    userID,
			Day:       day,
			WinCount:  boolToInt(won),
			LossCount: boolToInt(!won),
			UpdatedAt: now,
		}).Error
	if err != nil {
		return fmt.Errorf("mencatat hasil harian: %w", err)
	}
	return nil
}

func (r *runnerRepository) LockDaily(ctx context.Context, userID, day, reason string) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "day"}},
			DoUpdates: clause.Assignments(map[string]any{
				"locked_at":   now,
				"lock_reason": reason,
				"updated_at":  now,
			}),
		}).
		Create(&model.DailySummary{
			UserID:     userID,
			Day:        day,
			LockReason: reason,
			LockedAt:   &now,
			UpdatedAt:  now,
		}).Error
	if err != nil {
		return fmt.Errorf("mengunci hari user: %w", err)
	}
	return nil
}

func (r *runnerRepository) ListBrokerAccountsByUser(ctx context.Context, userID string) ([]model.BrokerAccount, error) {
	var accounts []model.BrokerAccount
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&accounts).Error
	if err != nil {
		return nil, fmt.Errorf("membaca akun broker user: %w", err)
	}
	return accounts, nil
}

func (r *runnerRepository) FindBrokerAccount(ctx context.Context, accountID, userID string) (*model.BrokerAccount, error) {
	var account model.BrokerAccount
	// user_id wajib ikut di query: akses lintas user adalah pelanggaran.
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", accountID, userID).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca akun broker: %w", err)
	}
	return &account, nil
}

func (r *runnerRepository) FindUserByID(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca user: %w", err)
	}
	return &user, nil
}

func (r *runnerRepository) ListActiveBrokerAccounts(ctx context.Context) ([]model.BrokerAccount, error) {
	var accounts []model.BrokerAccount
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("created_at ASC").
		Find(&accounts).Error
	if err != nil {
		return nil, fmt.Errorf("membaca akun broker aktif: %w", err)
	}
	return accounts, nil
}

func (r *runnerRepository) UpdateBrokerAccount(ctx context.Context, account *model.BrokerAccount) error {
	if err := r.db.WithContext(ctx).Save(account).Error; err != nil {
		return fmt.Errorf("memperbarui akun broker: %w", err)
	}
	return nil
}

func (r *runnerRepository) UpsertAccountState(ctx context.Context, state *model.BrokerAccountState) error {
	state.UpdatedAt = time.Now().UTC()
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "broker_account_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"connected", "balance", "equity", "free_margin",
				"currency", "last_error", "last_synced_at", "updated_at",
			}),
		}).
		Create(state).Error
	if err != nil {
		return fmt.Errorf("menyimpan status akun: %w", err)
	}
	return nil
}

func (r *runnerRepository) FindCredential(ctx context.Context, accountID string) (*model.Credential, error) {
	var credential model.Credential
	err := r.db.WithContext(ctx).
		Where("broker_account_id = ?", accountID).
		First(&credential).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca kredensial: %w", err)
	}
	return &credential, nil
}

func (r *runnerRepository) ListActiveSubscriptions(ctx context.Context) ([]model.Subscription, error) {
	var subs []model.Subscription
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&subs).Error
	if err != nil {
		return nil, fmt.Errorf("membaca langganan aktif: %w", err)
	}
	return subs, nil
}

func (r *runnerRepository) ListUsersBySymbol(ctx context.Context, symbol string) ([]model.User, error) {
	var users []model.User
	err := r.db.WithContext(ctx).
		Joins("JOIN subscription s ON s.user_id = user.id").
		Where("s.is_active = ? AND s.symbol = ?", true, symbol).
		Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("membaca user pemantau simbol: %w", err)
	}
	return users, nil
}

func (r *runnerRepository) UpsertEngineState(ctx context.Context, state *model.EngineState) error {
	state.UpdatedAt = time.Now().UTC()
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"connected", "engine_version", "strategy",
				"last_error", "last_seen_at", "updated_at",
			}),
		}).
		Create(state).Error
	if err != nil {
		return fmt.Errorf("menyimpan status engine: %w", err)
	}
	return nil
}

func (r *runnerRepository) CreateApprovalToken(ctx context.Context, token *model.ApprovalToken) error {
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(token).Error; err != nil {
		return fmt.Errorf("membuat token konfirmasi: %w", err)
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ListEngines mengembalikan status seluruh engine yang pernah terhubung.
func (r *runnerRepository) ListEngines(ctx context.Context) ([]model.EngineState, error) {
	var rows []model.EngineState
	if err := r.db.WithContext(ctx).Order("strategy ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca status engine: %w", err)
	}
	return rows, nil
}

// FindDailySummary mengembalikan kuota hari ini, atau ringkasan kosong bila
// belum ada perdagangan. Tidak ditemukannya baris bukan kegagalan.
func (r *runnerRepository) FindDailySummary(ctx context.Context, userID, day string) (*model.DailySummary, error) {
	var row model.DailySummary
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND day = ?", userID, day).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.DailySummary{UserID: userID, Day: day}, nil
		}
		return nil, fmt.Errorf("membaca kuota harian: %w", err)
	}
	return &row, nil
}
