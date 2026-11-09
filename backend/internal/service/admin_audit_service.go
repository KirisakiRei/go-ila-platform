package service

import (
	"context"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/export"
	"ila/backend/internal/pagination"
	"ila/backend/internal/repository"
)

// AdminAuditService membaca jejak audit untuk panel admin.
// Tidak ada metode menulis atau menghapus: tabel ini append-only.
type AdminAuditService interface {
	List(ctx context.Context, filter dto.AdminAuditFilter) ([]dto.AuditEntry, pagination.Meta, error)
	Detail(ctx context.Context, id string) (*dto.AuditEntry, error)
	Catalog() []dto.AuditCatalogEntry
	// Export mengunduh jejak audit sebagai CSV dan mencatat ekspornya sendiri.
	Export(ctx context.Context, actor AdminActor, filter dto.AdminAuditFilter) (*PositionExport, error)
}

type auditReaderStore interface {
	Search(ctx context.Context, filter repository.AuditFilter, p pagination.Params) ([]repository.AuditRow, error)
	Find(ctx context.Context, id string) (*repository.AuditRow, error)
	ApproxCount(ctx context.Context) (int64, error)
}

type adminAuditService struct {
	store    auditReaderStore
	recorder AdminAudit
}

// NewAdminAuditService merangkai pembaca audit.
func NewAdminAuditService(store auditReaderStore, recorder AdminAudit) AdminAuditService {
	return &adminAuditService{store: store, recorder: recorder}
}

// auditSortColumns adalah daftar putih kolom urut audit.
var auditSortColumns = map[string]string{
	"created_at": "created_at",
	"action":     "action",
}

func (s *adminAuditService) List(ctx context.Context, filter dto.AdminAuditFilter) ([]dto.AuditEntry, pagination.Meta, error) {
	params, err := s.params(filter)
	if err != nil {
		return nil, pagination.Meta{}, err
	}

	rows, err := s.store.Search(ctx, repository.AuditFilter{
		ActorID:    filter.ActorID,
		Action:     filter.Action,
		TargetType: filter.TargetType,
		TargetID:   filter.TargetID,
		From:       params.From,
		To:         params.To,
	}, params)
	if err != nil {
		return nil, pagination.Meta{}, err
	}

	items := make([]dto.AuditEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAuditEntry(row))
	}

	approx, err := s.store.ApproxCount(ctx)
	if err != nil {
		approx = 0
	}
	next := pagination.Cursor{}
	if len(rows) == params.PerPage && len(rows) > 0 {
		last := rows[len(rows)-1]
		next = pagination.CursorOfCreatedAt(last.CreatedAt, last.ID, params.Sort, params.Order)
	}
	return items, pagination.NewCursorMeta(params, approx, next), nil
}

// params memvalidasi filter menjadi parameter kueri.
// Dipakai bersama oleh daftar dan ekspor agar keduanya menyaring sama persis.
func (s *adminAuditService) params(filter dto.AdminAuditFilter) (pagination.Params, error) {
	params, err := pagination.Build(pagination.Request{
		Page:        filter.Page,
		PerPage:     filter.PerPage,
		Query:       filter.Query,
		Cursor:      filter.Cursor,
		Sort:        filter.Sort,
		Order:       filter.Order,
		From:        filter.From,
		To:          filter.To,
		AllowedSort: auditSortColumns,
		DefaultSort: "created_at",
		CursorMode:  true,
		// Rentang wajib: tanpa itu halaman audit memindai seluruh tabel.
		RequireRange:     true,
		DefaultRangeDays: 7,
	})
	if err != nil {
		return pagination.Params{}, err
	}
	if filter.Action != "" && !audit.Known(audit.Action(filter.Action)) {
		return pagination.Params{}, validation("aksi audit tidak dikenal")
	}
	return params, nil
}

// Export menyusun jejak audit sebagai CSV.
//
// Ekspor sendiri dicatat: tanpa itu, membawa keluar jejak audit tidak
// meninggalkan bukti siapa yang melakukannya.
func (s *adminAuditService) Export(ctx context.Context, actor AdminActor, filter dto.AdminAuditFilter) (*PositionExport, error) {
	params, err := s.params(filter)
	if err != nil {
		return nil, err
	}
	// Ekspor memakai batas baris keras, bukan per halaman.
	params.PerPage = pagination.ExportMaxRows
	params.Cursor = pagination.Cursor{}

	rows, err := s.store.Search(ctx, repository.AuditFilter{
		ActorID:    filter.ActorID,
		Action:     filter.Action,
		TargetType: filter.TargetType,
		TargetID:   filter.TargetID,
		From:       params.From,
		To:         params.To,
	}, params)
	if err != nil {
		return nil, err
	}

	builder := export.New(pagination.ExportMaxRows)
	if err := builder.Header(
		"waktu_wib", "waktu_ny", "aksi", "aktor", "email_aktor",
		"tipe_target", "target", "alasan", "request_id", "metadata",
	); err != nil {
		return nil, err
	}
	for _, row := range rows {
		entry := toAuditEntry(row)
		wib, ny := export.TimeColumns(&row.CreatedAt)
		if err := builder.Row(
			wib, ny, entry.Action, entry.ActorID, entry.ActorEmail,
			entry.TargetType, entry.TargetID, entry.Reason, entry.RequestID, row.Metadata,
		); err != nil {
			return nil, err
		}
	}

	if s.recorder != nil {
		if err := s.recorder.Record(ctx, AuditEntry{
			ActorID: actor.ID, Action: audit.AuditLogExport,
			TargetType: "audit_log", RequestID: actor.RequestID,
			Reason: "mengekspor jejak audit",
		}); err != nil {
			return nil, err
		}
	}

	return &PositionExport{
		Filename: "audit-log-" + time.Now().UTC().Format("20060102-150405") + ".csv",
		Content:  builder.Finished(),
		Rows:     builder.Rows(),
	}, nil
}

func (s *adminAuditService) Detail(ctx context.Context, id string) (*dto.AuditEntry, error) {
	row, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	entry := toAuditEntry(*row)
	return &entry, nil
}

// Catalog mengembalikan pilihan filter aksi untuk antarmuka admin.
func (s *adminAuditService) Catalog() []dto.AuditCatalogEntry {
	out := make([]dto.AuditCatalogEntry, 0, len(audit.Catalog))
	for _, entry := range audit.Catalog {
		out = append(out, dto.AuditCatalogEntry{
			Action:    string(entry.Action),
			Group:     entry.Group,
			Label:     entry.Label,
			Sensitive: entry.Sensitive,
		})
	}
	return out
}

func toAuditEntry(row repository.AuditRow) dto.AuditEntry {
	actorID := row.ActorID
	if actorID == "" {
		actorID = row.UserID
	}
	action := row.Action
	if action == "" {
		action = row.Event
	}
	return dto.AuditEntry{
		ID:          row.ID,
		ActorID:     actorID,
		ActorType:   row.ActorType,
		ActorEmail:  row.ActorEmail,
		Action:      action,
		TargetType:  firstNonEmpty(row.TargetType, row.Entity),
		TargetID:    firstNonEmpty(row.TargetID, row.EntityID),
		Reason:      row.Reason,
		BeforeState: row.BeforeState,
		AfterState:  row.AfterState,
		RequestID:   row.RequestID,
		CreatedAt:   dto.NewTimePair(row.CreatedAt),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
