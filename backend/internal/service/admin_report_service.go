package service

import (
	"context"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/pagination"
	"ila/backend/internal/repository"
)

// AdminReportService menyajikan laporan bisnis dan operasional.
//
// Prinsip: pendapatan hanya menghitung invoice lunas, refund dikurangkan
// terpisah, dan periode tanpa data tetap tampil sebagai 0, bukan hilang.
type AdminReportService interface {
	Revenue(ctx context.Context, from, to string) (*dto.RevenueReport, error)
	Customers(ctx context.Context, from, to string) (*dto.CustomerReport, error)
	Coupons(ctx context.Context, from, to string) (*dto.CouponReport, error)
	Operations(ctx context.Context, from, to string) (*dto.OperationReport, error)
}

type adminReportService struct {
	store reportStore
	now   func() time.Time
}

type reportStore interface {
	Revenue(ctx context.Context, from, to time.Time) ([]repository.RevenueRow, error)
	CustomerSummary(ctx context.Context, from, to time.Time) (repository.CustomerRow, error)
	CouponUsage(ctx context.Context, from, to time.Time) ([]repository.CouponUsageRow, error)
	ExecutionSummary(ctx context.Context, from, to time.Time) (repository.ExecutionRow, error)
	NotificationSummary(ctx context.Context, from, to time.Time) (repository.NotificationRow, error)
}

// NewAdminReportService merangkai laporan.
func NewAdminReportService(store reportStore) AdminReportService {
	return &adminReportService{store: store, now: func() time.Time { return time.Now().UTC() }}
}

// reportRange membaca rentang tanggal WIB; bawaan 30 hari terakhir.
func (s *adminReportService) reportRange(from, to string) (time.Time, time.Time, error) {
	params, err := pagination.Build(pagination.Request{
		From: from, To: to,
		AllowedSort:      map[string]string{"created_at": "created_at"},
		DefaultSort:      "created_at",
		RequireRange:     true,
		DefaultRangeDays: 30,
		Now:              s.now(),
	})
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	start := *params.From
	end := s.now()
	if params.To != nil {
		end = *params.To
	}
	return start, end, nil
}

func (s *adminReportService) Revenue(ctx context.Context, from, to string) (*dto.RevenueReport, error) {
	start, end, err := s.reportRange(from, to)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.Revenue(ctx, start, end)
	if err != nil {
		return nil, err
	}

	daily := make([]dto.RevenueDay, 0, len(rows))
	var gross, refunds int64
	for _, row := range rows {
		gross += row.PaidIDR
		refunds += row.RefundIDR
		daily = append(daily, dto.RevenueDay{
			Date:      row.Day.In(pagination.WIBLocation()).Format("2006-01-02"),
			PaidIDR:   row.PaidIDR,
			RefundIDR: row.RefundIDR,
			Count:     row.Count,
		})
	}
	return &dto.RevenueReport{
		From:      dto.NewTimePair(start),
		To:        dto.NewTimePair(end),
		GrossIDR:  gross,
		RefundIDR: refunds,
		NetIDR:    gross - refunds,
		Daily:     daily,
	}, nil
}

func (s *adminReportService) Customers(ctx context.Context, from, to string) (*dto.CustomerReport, error) {
	start, end, err := s.reportRange(from, to)
	if err != nil {
		return nil, err
	}
	row, err := s.store.CustomerSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return &dto.CustomerReport{
		From:         dto.NewTimePair(start),
		To:           dto.NewTimePair(end),
		NewUsers:     row.NewUsers,
		PaidUsers:    row.PaidUsers,
		Canceled:     row.Canceled,
		TrialStarted: row.TrialStarted,
	}, nil
}

func (s *adminReportService) Coupons(ctx context.Context, from, to string) (*dto.CouponReport, error) {
	start, end, err := s.reportRange(from, to)
	if err != nil {
		return nil, err
	}
	rows, err := s.store.CouponUsage(ctx, start, end)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CouponUsageItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.CouponUsageItem{
			Code: row.Code, Uses: row.Uses, DiscountIDR: row.DiscountIDR,
		})
	}
	return &dto.CouponReport{From: dto.NewTimePair(start), To: dto.NewTimePair(end), Items: items}, nil
}

func (s *adminReportService) Operations(ctx context.Context, from, to string) (*dto.OperationReport, error) {
	start, end, err := s.reportRange(from, to)
	if err != nil {
		return nil, err
	}
	exec, err := s.store.ExecutionSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}
	notif, err := s.store.NotificationSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return &dto.OperationReport{
		From:                dto.NewTimePair(start),
		To:                  dto.NewTimePair(end),
		PositionsOpened:     exec.Total,
		PositionsClosed:     exec.Closed,
		PositionsDryRun:     exec.DryRun,
		NotificationsSent:   notif.Sent,
		NotificationsFailed: notif.Failed,
	}, nil
}
