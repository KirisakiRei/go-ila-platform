package service

import (
	"context"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/pagination"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// AdminSignalService membaca sinyal lintas pengguna dari panel admin.
//
// Batasnya jelas: pembaca ini tidak pernah mengubah sinyal, tidak memicu
// eksekusi, dan tidak menyentuh kredensial. Fungsinya menjawab pertanyaan
// operasional "sinyal ini terbit kapan, diterima siapa, dan berakhir apa".
type AdminSignalService interface {
	List(ctx context.Context, actor AdminActor, filter dto.AdminSignalFilter) ([]dto.AdminSignalItem, pagination.Meta, error)
	Trace(ctx context.Context, actor AdminActor, signalID string) (*dto.AdminSignalTrace, error)
}

type adminSignalService struct {
	store AdminSignalStoreReader
	audit AdminAudit
	now   func() time.Time
}

// AdminSignalStoreReader adalah kebutuhan penyimpanan pembaca sinyal admin.
type AdminSignalStoreReader interface {
	List(ctx context.Context, filter repository.AdminSignalFilter, p pagination.Params) ([]repository.AdminSignalRow, error)
	Count(ctx context.Context, filter repository.AdminSignalFilter) (int64, error)
	Trace(ctx context.Context, signalID string) (*repository.AdminSignalTrace, error)
}

// NewAdminSignalService merangkai pembaca sinyal admin.
func NewAdminSignalService(store AdminSignalStoreReader, recorder AdminAudit) AdminSignalService {
	return &adminSignalService{
		store: store,
		audit: recorder,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

var signalSortColumns = map[string]string{
	"published_at": "published_at",
	"symbol":       "symbol",
	"status":       "status",
}

func (s *adminSignalService) List(ctx context.Context, actor AdminActor, filter dto.AdminSignalFilter) ([]dto.AdminSignalItem, pagination.Meta, error) {
	params, err := pagination.Build(pagination.Request{
		Page:        filter.Page,
		PerPage:     filter.PerPage,
		Query:       filter.Query,
		Sort:        filter.Sort,
		Order:       filter.Order,
		From:        filter.From,
		To:          filter.To,
		AllowedSort: signalSortColumns,
		DefaultSort: "published_at",
		// Rentang wajib: tabel sinyal tumbuh terus dan tidak boleh dipindai
		// seluruhnya pada setiap permintaan.
		RequireRange:     true,
		DefaultRangeDays: 7,
		Now:              s.now(),
	})
	if err != nil {
		return nil, pagination.Meta{}, err
	}

	storeFilter := repository.AdminSignalFilter{
		Symbol:   filter.Symbol,
		Status:   filter.Status,
		Strategy: filter.Strategy,
		From:     params.From,
		To:       params.To,
	}
	rows, err := s.store.List(ctx, storeFilter, params)
	if err != nil {
		return nil, pagination.Meta{}, err
	}
	total, err := s.store.Count(ctx, storeFilter)
	if err != nil {
		return nil, pagination.Meta{}, err
	}

	items := make([]dto.AdminSignalItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAdminSignalItem(row))
	}
	return items, pagination.NewMeta(params, total), nil
}

// Trace mengembalikan perjalanan satu sinyal lintas pengguna dan akun.
func (s *adminSignalService) Trace(ctx context.Context, actor AdminActor, signalID string) (*dto.AdminSignalTrace, error) {
	if signalID == "" {
		return nil, apperror.ErrBadRequest
	}
	trace, err := s.store.Trace(ctx, signalID)
	if err != nil {
		return nil, err
	}

	// Menelusuri sinyal berarti membuka perilaku trading banyak pelanggan
	// sekaligus, jadi aksesnya wajib tercatat (plans/07 §9).
	if err := s.audit.Record(ctx, AuditEntry{
		ActorID: actor.ID, Action: audit.UserViewSensitive,
		TargetType: "signal", TargetID: signalID, RequestID: actor.RequestID,
		Reason: "menelusuri perjalanan sinyal",
	}); err != nil {
		return nil, err
	}

	recipients := make([]dto.AdminSignalRecipient, 0, len(trace.Recipients))
	for _, row := range trace.Recipients {
		recipients = append(recipients, dto.AdminSignalRecipient{
			UserID:          row.UserID,
			UserName:        row.UserName,
			UserEmail:       row.UserEmail,
			BrokerAccountID: row.BrokerAccountID,
			Status:          row.Status,
			RejectReason:    row.RejectReason,
			PositionID:      row.PositionID,
			DecidedAt:       dto.NewTimePairPtr(row.DecidedAt),
			CreatedAt:       dto.NewTimePair(row.CreatedAt),
		})
	}

	events := make([]dto.AdminSignalEvent, 0, len(trace.Events))
	for _, row := range trace.Events {
		events = append(events, dto.AdminSignalEvent{
			UserID:     row.UserID,
			FromStatus: row.FromStatus,
			ToStatus:   row.ToStatus,
			Reason:     row.Reason,
			CreatedAt:  dto.NewTimePair(row.CreatedAt),
		})
	}

	executions := make([]dto.AdminSignalExecution, 0, len(trace.Executions))
	for _, row := range trace.Executions {
		executions = append(executions, dto.AdminSignalExecution{
			UserID:          row.UserID,
			BrokerAccountID: row.BrokerAccountID,
			Status:          row.Status,
			CreatedAt:       dto.NewTimePair(row.CreatedAt),
		})
	}

	positions := make([]dto.AdminPositionItem, 0, len(trace.Positions))
	for _, row := range trace.Positions {
		positions = append(positions, toAdminPositionItem(row))
	}

	return &dto.AdminSignalTrace{
		Signal:     toAdminSignalItem(trace.Signal),
		Recipients: recipients,
		Events:     events,
		Executions: executions,
		Positions:  positions,
	}, nil
}

func toAdminSignalItem(row repository.AdminSignalRow) dto.AdminSignalItem {
	return dto.AdminSignalItem{
		SignalID:        row.SignalID,
		Symbol:          row.Symbol,
		StrategyVersion: row.StrategyVersion,
		EngineVersion:   row.EngineVersion,
		Direction:       row.Direction,
		EntryPrice:      row.EntryPrice,
		StopLoss:        row.StopLoss,
		TakeProfit:      row.TakeProfit,
		RiskReward:      row.RiskReward,
		Status:          row.Status,
		Summary:         row.Summary,
		NextAction:      row.NextAction,
		PublishedAt:     dto.NewTimePair(row.PublishedAt),
		ExpiresAt:       dto.NewTimePair(row.ExpiresAt),
		CancelReason:    row.CancelReason,
		Recipients:      row.Recipients,
		Executed:        row.Executed,
	}
}
