package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// AdminAccountRow adalah akun broker tanpa kredensial.
// Kolom credential tidak pernah ikut dibaca (plans/07 §2 tingkat 4).
type AdminAccountRow struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	Label             string     `json:"label"`
	BrokerType        string     `json:"broker_type"`
	ExternalAccountID string     `json:"external_account_id"`
	Mode              string     `json:"mode"`
	RiskMode          string     `json:"risk_mode"`
	RiskPercent       float64    `json:"risk_percent"`
	IsActive          bool       `json:"is_active"`
	IsFrozen          bool       `json:"is_frozen"`
	IsDryRun          bool       `json:"is_dry_run"`
	FrozenAt          *time.Time `json:"frozen_at"`
	FrozenReason      string     `json:"frozen_reason"`
	LastVerifiedAt    *time.Time `json:"last_verified_at"`
	LastError         string     `json:"last_error"`
}

type adminAccountRepository struct {
	db *gorm.DB
}

// NewAdminAccountRepository membuat pembaca akun broker untuk admin.
func NewAdminAccountRepository(db *gorm.DB) *adminAccountRepository {
	return &adminAccountRepository{db: db}
}

const adminAccountColumns = `id, user_id, label, broker_type, external_account_id, mode,
	risk_mode, risk_percent, is_active, is_frozen, is_dry_run,
	frozen_at, frozen_reason, last_verified_at, last_error`

func (r *adminAccountRepository) ListForAdmin(ctx context.Context, userID string) ([]AdminAccountRow, error) {
	var rows []AdminAccountRow
	err := r.db.WithContext(ctx).Model(&model.BrokerAccount{}).
		Select(adminAccountColumns).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("created_at DESC").Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca akun broker pelanggan: %w", err)
	}
	return rows, nil
}

func (r *adminAccountRepository) FindForAdmin(ctx context.Context, accountID string) (*AdminAccountRow, error) {
	var row AdminAccountRow
	err := r.db.WithContext(ctx).Model(&model.BrokerAccount{}).
		Select(adminAccountColumns).
		Where("id = ? AND deleted_at IS NULL", accountID).
		Limit(1).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("membaca akun broker: %w", err)
	}
	if row.ID == "" {
		return nil, apperror.ErrNotFound
	}
	return &row, nil
}

func (r *adminAccountRepository) SetFrozen(ctx context.Context, accountID string, frozen bool, at *time.Time, reason string) error {
	reason = truncate(reason, 255)
	res := r.db.WithContext(ctx).Model(&model.BrokerAccount{}).
		Where("id = ?", accountID).
		Updates(map[string]any{
			"is_frozen":     frozen,
			"frozen_at":     at,
			"frozen_reason": reason,
		})
	if res.Error != nil {
		return fmt.Errorf("membekukan akun broker: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// DeleteAllForUser menghapus akun broker beserta kredensialnya.
// Dipakai saat penghapusan akun: kredensial tidak disimpan tanpa pemilik.
func (r *adminAccountRepository) DeleteAllForUser(ctx context.Context, userID string) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.Credential{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.Subscription{}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Delete(&model.BrokerAccount{}).Error
	})
	if err != nil {
		return fmt.Errorf("menghapus akun broker pelanggan: %w", err)
	}
	return nil
}

// adminPositionRepository membaca riwayat posisi untuk admin.
type adminPositionRepository struct {
	db *gorm.DB
}

// NewAdminPositionRepository membuat pembaca posisi untuk admin.
func NewAdminPositionRepository(db *gorm.DB) *adminPositionRepository {
	return &adminPositionRepository{db: db}
}

func (r *adminPositionRepository) ListForAdmin(ctx context.Context, userID string, limit int) ([]model.Position, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []model.Position
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("opened_at DESC, id DESC").
		Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca riwayat posisi pelanggan: %w", err)
	}
	return rows, nil
}

// adminLegalRepository membaca bukti persetujuan dokumen.
type adminLegalRepository struct {
	db *gorm.DB
}

// NewAdminLegalRepository membuat pembaca persetujuan legal.
func NewAdminLegalRepository(db *gorm.DB) *adminLegalRepository {
	return &adminLegalRepository{db: db}
}

func (r *adminLegalRepository) ListAcceptance(ctx context.Context, userID string) ([]model.AdminLegalAcceptance, error) {
	var rows []model.AdminLegalAcceptance
	const query = `SELECT d.slug, d.title, d.version, a.accepted_at, a.ip_hash
		FROM legal_acceptance a
		JOIN legal_document d ON d.id = a.document_id
		WHERE a.user_id = ?
		ORDER BY a.accepted_at DESC`
	var raw []struct {
		Slug       string
		Title      string
		Version    string
		AcceptedAt time.Time
		IPHash     string
	}
	if err := r.db.WithContext(ctx).Raw(query, userID).Scan(&raw).Error; err != nil {
		return nil, fmt.Errorf("membaca persetujuan legal: %w", err)
	}
	rows = make([]model.AdminLegalAcceptance, 0, len(raw))
	for _, item := range raw {
		rows = append(rows, model.AdminLegalAcceptance{
			Slug:       item.Slug,
			Title:      item.Title,
			Version:    item.Version,
			AcceptedAt: item.AcceptedAt,
			IPHash:     item.IPHash,
		})
	}
	return rows, nil
}

// adminNoteRepository menyimpan catatan internal staf.
type adminNoteRepository struct {
	db *gorm.DB
}

// NewAdminNoteRepository membuat repository catatan internal.
func NewAdminNoteRepository(db *gorm.DB) *adminNoteRepository {
	return &adminNoteRepository{db: db}
}

func (r *adminNoteRepository) List(ctx context.Context, userID string) ([]model.AdminNoteRow, error) {
	var raw []struct {
		ID         string
		AuthorID   string
		AuthorName string
		Body       string
		CreatedAt  time.Time
	}
	err := r.db.WithContext(ctx).Table("user_admin_note AS n").
		Select("n.id, n.author_id, COALESCE(u.name, '') AS author_name, n.body, n.created_at").
		Joins(`LEFT JOIN "user" AS u ON u.id = n.author_id`).
		Where("n.user_id = ? AND n.deleted_at IS NULL", userID).
		Order("n.created_at DESC").Limit(200).Scan(&raw).Error
	if err != nil {
		return nil, fmt.Errorf("membaca catatan internal: %w", err)
	}
	items := make([]model.AdminNoteRow, 0, len(raw))
	for _, row := range raw {
		items = append(items, model.AdminNoteRow{
			ID:         row.ID,
			AuthorID:   row.AuthorID,
			AuthorName: row.AuthorName,
			Body:       row.Body,
			CreatedAt:  row.CreatedAt,
		})
	}
	return items, nil
}

func (r *adminNoteRepository) Add(ctx context.Context, userID, authorID, body string) error {
	note := &model.UserAdminNote{UserID: userID, AuthorID: authorID, Body: body}
	if err := r.db.WithContext(ctx).Create(note).Error; err != nil {
		return fmt.Errorf("menyimpan catatan internal: %w", err)
	}
	return nil
}

// truncate memotong teks agar muat pada kolom berukuran tetap.
func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}
