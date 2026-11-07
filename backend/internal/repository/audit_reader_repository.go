package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/internal/pagination"
	"ila/backend/pkg/apperror"
)

// AuditFilter adalah syarat pembacaan audit di panel admin.
type AuditFilter struct {
	ActorID    string
	Action     string
	TargetType string
	TargetID   string
	From       *time.Time
	To         *time.Time
	Query      string
}

// AuditRow menambahkan email pelaku untuk tampilan admin.
type AuditRow struct {
	model.AuditLog
	ActorEmail string `json:"actor_email"`
}

type auditReader interface {
	Search(ctx context.Context, filter AuditFilter, p pagination.Params) ([]AuditRow, error)
	Find(ctx context.Context, id string) (*AuditRow, error)
	ApproxCount(ctx context.Context) (int64, error)
}

type auditReaderRepo struct {
	db *gorm.DB
}

// NewAuditReader membuat pembaca audit log untuk panel admin.
func NewAuditReader(db *gorm.DB) auditReader {
	return &auditReaderRepo{db: db}
}

func (r *auditReaderRepo) base(ctx context.Context, filter AuditFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Table("audit_log AS a").
		Select("a.*, COALESCE(u.email, '') AS actor_email").
		Joins(`LEFT JOIN "user" AS u ON u.id = COALESCE(NULLIF(a.actor_id, ''), a.user_id)`)

	if filter.ActorID != "" {
		q = q.Where(`COALESCE(NULLIF(a.actor_id, ''), a.user_id) = ?`, filter.ActorID)
	}
	if filter.Action != "" {
		q = q.Where("a.action = ?", filter.Action)
	}
	if filter.TargetType != "" {
		q = q.Where("a.target_type = ?", filter.TargetType)
	}
	if filter.TargetID != "" {
		q = q.Where("a.target_id = ?", filter.TargetID)
	}
	if filter.From != nil {
		q = q.Where("a.created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		q = q.Where("a.created_at < ?", *filter.To)
	}
	// Pencarian bebas tidak dipakai di audit: tabel tumbuh paling cepat, dan
	// LIKE '%x%' di sini berarti pemindaian penuh (plans/24 §3.1).
	return q
}

func (r *auditReaderRepo) Search(ctx context.Context, filter AuditFilter, p pagination.Params) ([]AuditRow, error) {
	q := r.base(ctx, filter)

	direction := "DESC"
	if p.Order == "asc" {
		direction = "ASC"
	}
	if p.Cursor.Present() {
		op := "<"
		if direction == "ASC" {
			op = ">"
		}
		q = q.Where("(a.created_at, a.id) "+op+" (?, ?)", p.Cursor.Value, p.Cursor.ID)
	}
	q = q.Order("a.created_at " + direction + ", a.id " + direction).Limit(p.PerPage)
	if !p.CursorMode {
		q = q.Offset(p.Offset())
	}

	var rows []AuditRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca audit log: %w", err)
	}
	return rows, nil
}

func (r *auditReaderRepo) Find(ctx context.Context, id string) (*AuditRow, error) {
	var row AuditRow
	err := r.base(ctx, AuditFilter{}).Where("a.id = ?", id).Limit(1).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("membaca audit log: %w", err)
	}
	if strings.TrimSpace(row.ID) == "" {
		return nil, apperror.ErrNotFound
	}
	return &row, nil
}

func (r *auditReaderRepo) ApproxCount(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Raw(`SELECT COALESCE(reltuples::bigint, 0) FROM pg_class WHERE relname = 'audit_log'`).
		Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung perkiraan audit: %w", err)
	}
	if total < 0 {
		total = 0
	}
	return total, nil
}
