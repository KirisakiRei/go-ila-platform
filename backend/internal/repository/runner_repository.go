package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	// ExpireStaleSignals menutup sinyal yang masa berlakunya sudah lewat.
	// Engine hanya mengirim status terminal lewat SignalUpdate, sehingga tanpa
	// penyapu ini baris READY_TO_ENTRY lama tertinggal selamanya di database
	// dan ikut tampil di radar.
	ExpireStaleSignals(ctx context.Context, now time.Time) (int64, error)

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
	// ClaimAccountLock mengunci satu akun broker selama eksekusi berlangsung.
	// Nilai balik false berarti akun sedang mengeksekusi sinyal lain.
	ClaimAccountLock(ctx context.Context, accountID, signalID string, now time.Time) (bool, error)
	// ReleaseAccountLock melepas kunci akun setelah eksekusi selesai.
	ReleaseAccountLock(ctx context.Context, accountID string) error
	// ReleaseStaleAccountLock melepas kunci yang tertinggal karena proses
	// berhenti di tengah eksekusi, agar akun tidak terkunci selamanya.
	ReleaseStaleAccountLock(ctx context.Context, staleBefore time.Time) error
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
	// MarkAccountVerified mencatat bahwa broker menjawab pada waktu tertentu.
	// Sengaja hanya menyentuh dua kolom: menulis seluruh baris (Save) bisa
	// menimpa perubahan user yang terjadi bersamaan, mis. perubahan lot atau
	// status freeze.
	MarkAccountVerified(ctx context.Context, accountID string, at time.Time) error
	UpsertAccountState(ctx context.Context, state *model.BrokerAccountState) error
	FindCredential(ctx context.Context, accountID string) (*model.Credential, error)

	// Subscription: kebutuhan engine per (symbol, strategy).
	ListActiveSubscriptions(ctx context.Context) ([]model.Subscription, error)
	ListUsersBySymbol(ctx context.Context, symbol string) ([]model.User, error)
	// ListAccountsBySymbol mengembalikan akun aktif yang berhak menerima sinyal
	// satu simbol. Satu user dapat menyumbang lebih dari satu akun, karena
	// pilihan pair kini berada di level akun.
	ListAccountsBySymbol(ctx context.Context, symbol string) ([]model.SubscriptionAccount, error)
	// EnabledAccountStrategies mengembalikan saklar metode untuk satu simbol,
	// dipetakan dari akun ke himpunan nama metode yang MENYALA, beserta daftar
	// akun yang sudah pernah diatur.
	//
	// Akun yang sudah diatur tetapi seluruh metodenya dimatikan tidak muncul di
	// peta saklar; itulah sebabnya ConfiguredAccounts ikut dikembalikan agar
	// pemanggil tidak salah menganggapnya "belum pernah diatur".
	EnabledAccountStrategies(ctx context.Context, symbol string) (map[string]map[string]bool, error)
	// ConfiguredAccounts melaporkan akun yang sudah punya baris saklar untuk
	// simbol tertentu. Akun di luar daftar ini memakai saklar bawaan.
	ConfiguredAccounts(ctx context.Context, symbol string) (map[string]bool, error)
	// TimeframeSwitches membaca saklar pasangan timeframe untuk satu simbol,
	// dipetakan akun -> metode -> pasangan -> saklar.
	TimeframeSwitches(ctx context.Context, symbol string) (map[string]map[string]map[TimeframeKey]TimeframeState, error)
	// EnabledPairSymbols mengembalikan simbol kanonik yang masih diaktifkan
	// admin. Symbol lain tidak boleh diminta ke engine.
	EnabledPairSymbols(ctx context.Context) ([]string, error)
	// CorrelationMap mengembalikan peta simbol ke simbol pembanding SMT-nya.
	// Hanya pair aktif yang punya korelasi yang ikut, sehingga engine tidak
	// pernah diminta menarik feed instrumen yang sudah dinonaktifkan admin.
	CorrelationMap(ctx context.Context) (map[string][]string, error)
	// PairFeeds mengembalikan peta simbol ke feed candle pilihan admin.
	// Simbol tanpa nilai memakai pemetaan bawaan engine.
	PairFeeds(ctx context.Context) (map[string]string, error)

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
				"summary", "next_action", "expires_at", "feed_source", "updated_at",
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

// ExpireStaleSignals menandai EXPIRED sinyal yang belum dieksekusi tetapi
// sudah melewati expires_at.
//
// Sengaja HANYA menyasar READY_TO_ENTRY dan OPPORTUNITY. Baris status menunggu
// (WAIT_*) memakai identitas tetap per slot dan terus dikirim ulang engine;
// menandainya EXPIRED akan membuat Runner menganggapnya final selamanya
// (HandleSignal melewati status terminal), sehingga radar satu slot mati
// permanen. Status ACKED/CONSUMED juga tidak disentuh agar jejak eksekusi
// tidak tersamarkan. Query ini idempotent dan tidak menyentuh tabel posisi.
func (r *runnerRepository) ExpireStaleSignals(ctx context.Context, now time.Time) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.Signal{}).
		Where("expires_at < ?", now).
		Where("status IN ?", []string{
			model.SignalStatusReadyToEntry,
			model.SignalStatusOpportunity,
		}).
		Updates(map[string]any{
			"status":        model.SignalStatusExpired,
			"cancel_reason": "masa berlaku habis",
		})
	if res.Error != nil {
		return 0, fmt.Errorf("menyapu sinyal kedaluwarsa: %w", res.Error)
	}
	return res.RowsAffected, nil
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

// ClaimAccountLock mengunci satu akun broker selama eksekusi berlangsung.
//
// Kunci berbentuk baris dengan primary key akun, sehingga percobaan kedua pada
// akun yang sama ditolak oleh basis data — bukan oleh pemeriksaan di memori
// yang bisa dilewati dua goroutine sekaligus.
func (r *runnerRepository) ClaimAccountLock(ctx context.Context, accountID, signalID string, now time.Time) (bool, error) {
	row := &model.AccountExecutionLock{
		BrokerAccountID: accountID,
		SignalID:        signalID,
		ClaimedAt:       now,
	}
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(row)
	if res.Error != nil {
		return false, fmt.Errorf("mengunci akun untuk eksekusi: %w", res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ReleaseAccountLock melepas kunci akun. Selalu dipanggil setelah eksekusi,
// termasuk saat gagal, agar akun tidak terkunci karena satu order bermasalah.
func (r *runnerRepository) ReleaseAccountLock(ctx context.Context, accountID string) error {
	res := r.db.WithContext(ctx).
		Where("broker_account_id = ?", accountID).
		Delete(&model.AccountExecutionLock{})
	if res.Error != nil {
		return fmt.Errorf("melepas kunci akun: %w", res.Error)
	}
	return nil
}

// ReleaseStaleAccountLock melepas kunci yang tertinggal karena proses berhenti
// di tengah eksekusi. Tanpa ini, akun bisa terkunci selamanya setelah restart.
func (r *runnerRepository) ReleaseStaleAccountLock(ctx context.Context, staleBefore time.Time) error {
	res := r.db.WithContext(ctx).
		Where("claimed_at < ?", staleBefore).
		Delete(&model.AccountExecutionLock{})
	if res.Error != nil {
		return fmt.Errorf("melepas kunci akun tertinggal: %w", res.Error)
	}
	return nil
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

// MarkAccountVerified mencatat waktu terakhir broker terbukti menjawab.
//
// last_error ikut dikosongkan karena keberhasilan membaca akun berarti galat
// sebelumnya sudah tidak berlaku.
func (r *runnerRepository) MarkAccountVerified(ctx context.Context, accountID string, at time.Time) error {
	err := r.db.WithContext(ctx).Model(&model.BrokerAccount{}).
		Where("id = ?", accountID).
		Updates(map[string]any{"last_verified_at": at, "last_error": ""}).Error
	if err != nil {
		return fmt.Errorf("menandai akun terverifikasi: %w", err)
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
		// "user" adalah kata kunci SQL, jadi wajib dikutip. Tanpa kutip,
		// PostgreSQL menolak query dan tidak ada user yang pernah menerima
		// sinyal walau semuanya tampak benar.
		Joins(`JOIN subscription s ON s.user_id = "user".id`).
		Where("s.is_active = ? AND s.symbol = ? AND s.deleted_at IS NULL", true, symbol).
		Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("membaca user pemantau simbol: %w", err)
	}
	return users, nil
}

// ListAccountsBySymbol mengembalikan pasangan user+akun yang berhak menerima
// sinyal satu simbol.
//
// Logika pemilihan akun:
//   - Langganan tanpa broker_account_id berlaku untuk semua akun aktif user
//     (kompatibel dengan data sebelum katalog pair).
//   - Langganan dengan broker_account_id hanya untuk akun itu.
//   - User nonaktif atau terhapus tidak pernah diikutkan.
//   - Akunnya sendiri harus aktif.
func (r *runnerRepository) ListAccountsBySymbol(ctx context.Context, symbol string) ([]model.SubscriptionAccount, error) {
	type row struct {
		AccountID     string
		AccountUserID string
	}

	var rows []row
	err := r.db.WithContext(ctx).
		Table("subscription AS s").
		Select("ba.id AS account_id, ba.user_id AS account_user_id").
		Joins(`JOIN "user" u ON u.id = s.user_id`).
		Joins("JOIN broker_account ba ON ba.user_id = s.user_id AND ba.is_active = ?", true).
		Where("s.is_active = ? AND s.symbol = ? AND s.deleted_at IS NULL", true, symbol).
		Where("u.deleted_at IS NULL AND (u.status = '' OR u.status = ?)", "active").
		Where("s.broker_account_id IS NULL OR s.broker_account_id = ba.id").
		Group("ba.id, ba.user_id").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca akun pemantau simbol: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}

	accountIDs := make([]string, 0, len(rows))
	userIDs := make([]string, 0, len(rows))
	seenUser := make(map[string]bool, len(rows))
	for _, row := range rows {
		accountIDs = append(accountIDs, row.AccountID)
		if !seenUser[row.AccountUserID] {
			seenUser[row.AccountUserID] = true
			userIDs = append(userIDs, row.AccountUserID)
		}
	}

	var accounts []model.BrokerAccount
	if err := r.db.WithContext(ctx).Where("id IN ?", accountIDs).Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("membaca akun broker: %w", err)
	}
	var users []model.User
	if err := r.db.WithContext(ctx).Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("membaca pemilik akun: %w", err)
	}

	userByID := make(map[string]model.User, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}

	out := make([]model.SubscriptionAccount, 0, len(accounts))
	for _, acc := range accounts {
		owner, ok := userByID[acc.UserID]
		if !ok {
			continue
		}
		out = append(out, model.SubscriptionAccount{User: owner, Account: acc})
	}
	return out, nil
}

// EnabledAccountStrategies membaca saklar metode yang menyala untuk satu simbol.
//
// Hasilnya akun -> himpunan metode yang aktif. Hanya baris yang benar-benar
// menyala yang dilaporkan. Untuk membedakan "belum pernah diatur" dari
// "sudah diatur tetapi semuanya dimatikan", pasangkan dengan ConfiguredAccounts.
func (r *runnerRepository) EnabledAccountStrategies(ctx context.Context, symbol string) (map[string]map[string]bool, error) {
	var rows []model.AccountPairStrategy
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND is_enabled = ? AND deleted_at IS NULL", symbol, true).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca saklar metode akun: %w", err)
	}

	out := make(map[string]map[string]bool, len(rows))
	for _, row := range rows {
		if out[row.BrokerAccountID] == nil {
			out[row.BrokerAccountID] = make(map[string]bool, 2)
		}
		out[row.BrokerAccountID][row.Strategy] = true
	}
	return out, nil
}

// ConfiguredAccounts melaporkan akun yang sudah punya baris saklar untuk satu
// simbol, termasuk yang saklarnya mati.
//
// Tanpa daftar ini, akun yang seluruh metodenya dimatikan akan terlihat sama
// dengan akun yang belum pernah diatur, sehingga saklar mati bisa diabaikan.
func (r *runnerRepository) ConfiguredAccounts(ctx context.Context, symbol string) (map[string]bool, error) {
	var accountIDs []string
	err := r.db.WithContext(ctx).
		Model(&model.AccountPairStrategy{}).
		Where("symbol = ? AND deleted_at IS NULL", symbol).
		Distinct().
		Pluck("broker_account_id", &accountIDs).Error
	if err != nil {
		return nil, fmt.Errorf("membaca akun dengan saklar metode: %w", err)
	}

	out := make(map[string]bool, len(accountIDs))
	for _, id := range accountIDs {
		out[id] = true
	}
	return out, nil
}

// TimeframeKey adalah pasangan timeframe sebagai kunci peta.
// TimeframeKey memegang pasangan timeframe untuk pemetaan saklar akun.
type TimeframeKey struct {
	Htf string
	Ltf string
}

// CanonTimeframe membakukan penulisan timeframe agar bentuk alias saling cocok
// ("4H" vs "H4", "1H" vs "H1", "M15" vs "15M", "M5" vs "5M", "D1" vs "D", "W1" vs "W").
func CanonTimeframe(s string) string {
	clean := strings.ToUpper(strings.TrimSpace(s))
	switch clean {
	case "H4", "4H":
		return "4H"
	case "H1", "1H":
		return "1H"
	case "M15", "15M":
		return "15M"
	case "M5", "5M":
		return "5M"
	case "D1", "1D", "D":
		return "D"
	case "W1", "1W", "W":
		return "W"
	default:
		return clean
	}
}

// TimeframeState adalah saklar satu pasangan timeframe untuk satu akun.
type TimeframeState struct {
	Watch bool
	Auto  bool
}

// TimeframeSwitches membaca saklar pasangan timeframe untuk satu simbol.
//
// Hasilnya akun -> metode -> pasangan -> saklar. Hanya akun yang sudah punya
// baris yang muncul; akun lain berarti belum diatur dan Runner memakai
// perilaku lama (semua diproses, semuanya boleh dieksekusi).
func (r *runnerRepository) TimeframeSwitches(ctx context.Context, symbol string) (map[string]map[string]map[TimeframeKey]TimeframeState, error) {
	var rows []model.AccountPairTimeframe
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND deleted_at IS NULL", symbol).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca saklar timeframe: %w", err)
	}

	out := make(map[string]map[string]map[TimeframeKey]TimeframeState, len(rows))
	for _, row := range rows {
		if out[row.BrokerAccountID] == nil {
			out[row.BrokerAccountID] = make(map[string]map[TimeframeKey]TimeframeState, 2)
		}
		if out[row.BrokerAccountID][row.Strategy] == nil {
			out[row.BrokerAccountID][row.Strategy] = make(map[TimeframeKey]TimeframeState, 2)
		}
		out[row.BrokerAccountID][row.Strategy][TimeframeKey{Htf: row.Htf, Ltf: row.Ltf}] = TimeframeState{
			Watch: row.IsWatch,
			Auto:  row.IsAuto,
		}
	}
	return out, nil
}

// EnabledPairSymbols membaca simbol kanonik yang masih diaktifkan admin.
// Kategori atau grup yang dinonaktifkan ikut mematikan pair di dalamnya.
func (r *runnerRepository) EnabledPairSymbols(ctx context.Context) ([]string, error) {
	var symbols []string
	// Table() tidak mengenal model, jadi filter soft delete ditulis eksplisit.
	// Sisa baris yang pernah dihapus tidak boleh dianggap pair aktif.
	err := r.db.WithContext(ctx).
		Table("trading_pair AS p").
		Select("p.symbol").
		Joins("JOIN pair_category c ON c.id = p.category_id AND c.is_enabled = ? AND c.deleted_at IS NULL", true).
		Joins("LEFT JOIN pair_group g ON g.id = p.group_id AND g.deleted_at IS NULL").
		Where("p.is_enabled = ? AND p.deleted_at IS NULL", true).
		Where("g.id IS NULL OR g.is_enabled = ?", true).
		Order("p.symbol ASC").
		Pluck("p.symbol", &symbols).Error
	if err != nil {
		return nil, fmt.Errorf("membaca pair aktif: %w", err)
	}
	return symbols, nil
}

// CorrelationMap membaca pasangan korelasi SMT yang diatur admin.
//
// Hanya pair aktif di kategori/grup aktif yang diikutkan, sehingga instrumen
// yang disembunyikan admin tidak pernah diminta ke engine. Nilai dipisah koma
// dipecah menjadi irisan simbol kanonik tanpa entri kosong.
func (r *runnerRepository) CorrelationMap(ctx context.Context) (map[string][]string, error) {
	type row struct {
		Symbol            string
		CorrelatedSymbols string
	}

	var rows []row
	// Table() tidak mengenal model, jadi filter soft delete ditulis eksplisit.
	err := r.db.WithContext(ctx).
		Table("trading_pair AS p").
		Select("p.symbol, p.correlated_symbols").
		Joins("JOIN pair_category c ON c.id = p.category_id AND c.is_enabled = ? AND c.deleted_at IS NULL", true).
		Joins("LEFT JOIN pair_group g ON g.id = p.group_id AND g.deleted_at IS NULL").
		Where("p.is_enabled = ? AND p.deleted_at IS NULL", true).
		Where("g.id IS NULL OR g.is_enabled = ?", true).
		Where("p.correlated_symbols <> ''").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca korelasi pair: %w", err)
	}

	out := make(map[string][]string, len(rows))
	for _, item := range rows {
		symbol := strings.ToUpper(strings.TrimSpace(item.Symbol))
		if symbol == "" {
			continue
		}
		correlations := make([]string, 0, 4)
		seen := make(map[string]bool, 4)
		for _, part := range strings.Split(item.CorrelatedSymbols, ",") {
			value := strings.ToUpper(strings.TrimSpace(part))
			if value == "" || value == symbol || seen[value] {
				continue
			}
			seen[value] = true
			correlations = append(correlations, value)
		}
		if len(correlations) > 0 {
			out[symbol] = correlations
		}
	}
	return out, nil
}

// PairFeeds membaca feed candle pilihan admin untuk setiap pair aktif.
// Nilai kosong dilewati agar engine tetap memakai pemetaan bawaannya.
func (r *runnerRepository) PairFeeds(ctx context.Context) (map[string]string, error) {
	type row struct {
		Symbol     string
		FeedSource string
	}

	var rows []row
	// Table() tidak mengenal model, jadi filter soft delete ditulis eksplisit.
	err := r.db.WithContext(ctx).
		Table("trading_pair AS p").
		Select("p.symbol, p.feed_source").
		Joins("JOIN pair_category c ON c.id = p.category_id AND c.is_enabled = ? AND c.deleted_at IS NULL", true).
		Joins("LEFT JOIN pair_group g ON g.id = p.group_id AND g.deleted_at IS NULL").
		Where("p.is_enabled = ? AND p.deleted_at IS NULL", true).
		Where("g.id IS NULL OR g.is_enabled = ?", true).
		Where("p.feed_source <> ''").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca feed pair: %w", err)
	}

	out := make(map[string]string, len(rows))
	for _, item := range rows {
		symbol := strings.ToUpper(strings.TrimSpace(item.Symbol))
		feed := strings.ToUpper(strings.TrimSpace(item.FeedSource))
		if symbol == "" || feed == "" {
			continue
		}
		out[symbol] = feed
	}
	return out, nil
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
