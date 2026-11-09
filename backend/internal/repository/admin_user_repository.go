package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/internal/pagination"
	"ila/backend/pkg/apperror"
)

// UserFilter adalah syarat daftar pelanggan di panel admin.
type UserFilter struct {
	Statuses    []string
	Tiers       []string
	HasAccounts *bool
	HasTrial    bool
	OnlyAdmin   bool
	// RoleID menyaring menurut role RBAC (kolom u.role_id).
	RoleID string
	From   *time.Time
	To     *time.Time
}

// UserRow adalah baris daftar pelanggan beserta hitungan akun broker.
type UserRow struct {
	model.User
	RoleCode     string `json:"role_code"`
	AccountCount int64  `json:"account_count"`
}

// AdminUserRepository adalah data pelanggan untuk panel admin.
// Tidak ada metode yang mengembalikan kredensial (plans/07 §2 tingkat 4).
type AdminUserRepository interface {
	List(ctx context.Context, filter UserFilter, p pagination.Params) ([]UserRow, error)
	Count(ctx context.Context, filter UserFilter) (int64, error)
	FindRow(ctx context.Context, id string) (*UserRow, error)
	// UpdateStatus menyimpan status beserta jejak ban/suspend.
	UpdateStatus(ctx context.Context, id, status string, fields map[string]any) error
	UpdateTier(ctx context.Context, id, tier string, startedAt, expiresAt *time.Time) error
	Anonymize(ctx context.Context, id string) error
	CountActiveAdmins(ctx context.Context) (int64, error)
	ListAdminIDs(ctx context.Context) ([]string, error)
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByUsername(ctx context.Context, username string) (*model.User, error)
}

type adminUserRepository struct {
	db *gorm.DB
}

// NewAdminUserRepository membuat repository pelanggan untuk panel admin.
func NewAdminUserRepository(db *gorm.DB) AdminUserRepository {
	return &adminUserRepository{db: db}
}

func (r *adminUserRepository) base(ctx context.Context, filter UserFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Table(`"user" AS u`).
		Select(`u.*, COALESCE(ro.code, '') AS role_code,
			(SELECT COUNT(*) FROM broker_account ba WHERE ba.user_id = u.id AND ba.deleted_at IS NULL) AS account_count`).
		Joins("LEFT JOIN role ro ON ro.id = u.role_id").
		Where("u.deleted_at IS NULL")

	if len(filter.Statuses) > 0 {
		q = q.Where("u.status IN ?", filter.Statuses)
	}
	if len(filter.Tiers) > 0 {
		q = q.Where("u.tier IN ?", filter.Tiers)
	}
	if filter.OnlyAdmin {
		q = q.Where("u.role = ?", "admin")
	}
	if filter.RoleID != "" {
		q = q.Where("u.role_id = ?", filter.RoleID)
	}
	if filter.HasTrial {
		q = q.Where("u.trial_used = ?", true)
	}
	if filter.HasAccounts != nil {
		if *filter.HasAccounts {
			q = q.Where("EXISTS (SELECT 1 FROM broker_account ba WHERE ba.user_id = u.id AND ba.deleted_at IS NULL)")
		} else {
			q = q.Where("NOT EXISTS (SELECT 1 FROM broker_account ba WHERE ba.user_id = u.id AND ba.deleted_at IS NULL)")
		}
	}
	if filter.From != nil {
		q = q.Where("u.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("u.created_at < ?", *filter.To)
	}
	return q
}

func (r *adminUserRepository) List(ctx context.Context, filter UserFilter, p pagination.Params) ([]UserRow, error) {
	q := r.base(ctx, filter)

	if p.Query != "" {
		// Pencarian email dan nama memakai awalan: masih bisa memakai indeks
		// dan tidak memindai seluruh tabel (plans/24 §3.1).
		like := pagination.EscapeLike(p.Query)
		q = q.Where(`u.email ILIKE ? OR u.name ILIKE ? OR u.id = ?`, like+"%", like+"%", p.Query)
	}

	direction := "DESC"
	if p.Order == "asc" {
		direction = "ASC"
	}
	q = q.Order("u." + p.Sort + " " + direction + ", u.id " + direction)
	q = q.Limit(p.PerPage).Offset(p.Offset())

	var rows []UserRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca daftar pelanggan: %w", err)
	}
	return rows, nil
}

func (r *adminUserRepository) Count(ctx context.Context, filter UserFilter) (int64, error) {
	// Hitungan memakai query terpisah tanpa join agar tidak bergantung pada
	// SELECT yang dipakai daftar.
	q := r.db.WithContext(ctx).Model(&model.User{}).Where("deleted_at IS NULL")
	if len(filter.Statuses) > 0 {
		q = q.Where("status IN ?", filter.Statuses)
	}
	if len(filter.Tiers) > 0 {
		q = q.Where("tier IN ?", filter.Tiers)
	}
	if filter.OnlyAdmin {
		q = q.Where("role = ?", "admin")
	}
	if filter.RoleID != "" {
		q = q.Where("role_id = ?", filter.RoleID)
	}
	if filter.HasTrial {
		q = q.Where("trial_used = ?", true)
	}
	if filter.From != nil {
		q = q.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("created_at < ?", *filter.To)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return 0, fmt.Errorf("menghitung pelanggan: %w", err)
	}
	return total, nil
}

func (r *adminUserRepository) FindRow(ctx context.Context, id string) (*UserRow, error) {
	var row UserRow
	err := r.base(ctx, UserFilter{}).Where("u.id = ?", id).Limit(1).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("membaca pelanggan: %w", err)
	}
	if strings.TrimSpace(row.ID) == "" {
		return nil, apperror.ErrNotFound
	}
	return &row, nil
}

func (r *adminUserRepository) UpdateStatus(ctx context.Context, id, status string, fields map[string]any) error {
	updates := map[string]any{"status": status}
	for key, value := range fields {
		updates[key] = value
	}
	// Ban dan suspend mencabut sesi lama: token yang beredar langsung gugur.
	updates["session_version"] = gorm.Expr("session_version + 1")

	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("memperbarui status pelanggan: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *adminUserRepository) UpdateTier(ctx context.Context, id, tier string, startedAt, expiresAt *time.Time) error {
	updates := map[string]any{
		"tier":            tier,
		"tier_started_at": startedAt,
		"tier_expires_at": expiresAt,
		"grace_until":     nil,
		"runner_dirty_at": time.Now().UTC(),
	}
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("memperbarui tier pelanggan: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// Anonymize menghapus data pribadi tetapi mempertahankan baris untuk invoice
// dan audit (plans/07 §5.1).
func (r *adminUserRepository) Anonymize(ctx context.Context, id string) error {
	now := time.Now().UTC()
	var email string
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Select("email").Scan(&email).Error; err != nil {
		return fmt.Errorf("membaca email sebelum anonimisasi: %w", err)
	}

	// Email diganti hash tanpa domain agar tidak bisa dipakai mendaftar ulang
	// dengan alamat yang sama, tanpa menyimpan alamat aslinya.
	hashed := hashEmail(email)
	placeholder := "dihapus+" + hashed + "@invalid.local"

	updates := map[string]any{
		"name":                    "Pengguna dihapus",
		"email":                   placeholder,
		"password_hash":           "",
		"telegram_chat_id":        nil,
		"telegram_linked_at":      nil,
		"device_fingerprint_hash": "",
		"signup_ip_hash":          "",
		"flags":                   "",
		"internal_note":           "",
		"notify_enabled":          false,
		"status":                  model.UserStatusDeleted,
		"deleted_at_user":         now,
		"banned_at":               nil,
		"ban_reason":              "",
		"ban_category":            "",
		"role_id":                 nil,
		"session_version":         gorm.Expr("session_version + 1"),
		"runner_dirty_at":         now,
	}
	if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("menganonimkan pelanggan: %w", err)
	}
	return nil
}

func (r *adminUserRepository) CountActiveAdmins(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("role = ? AND status = ? AND deleted_at IS NULL", "admin", model.UserStatusActive).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung admin aktif: %w", err)
	}
	return total, nil
}

func (r *adminUserRepository) ListAdminIDs(ctx context.Context) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Where("role = ? AND deleted_at IS NULL", "admin").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("membaca daftar id admin: %w", err)
	}
	return ids, nil
}

func (r *adminUserRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("membuat pengguna: %w", err)
	}
	return nil
}

func (r *adminUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("mencari email: %w", err)
	}
	return &user, nil
}

func (r *adminUserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("mencari username: %w", err)
	}
	return &user, nil
}

// hashEmail menyamarkan email tanpa menyimpan bentuk aslinya.
func hashEmail(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])[:24]
}
