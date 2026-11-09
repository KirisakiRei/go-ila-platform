package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/export"
	"ila/backend/internal/pagination"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
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
	// Export mengunduh laporan bisnis sebagai CSV. kind: revenue, customers, coupons.
	Export(ctx context.Context, actor AdminActor, kind, from, to string) (*PositionExport, error)
}

type adminReportService struct {
	store reportStore
	now   func() time.Time
	audit AdminAudit
}

type reportStore interface {
	Revenue(ctx context.Context, from, to time.Time) ([]repository.RevenueRow, error)
	CustomerSummary(ctx context.Context, from, to time.Time) (repository.CustomerRow, error)
	CouponUsage(ctx context.Context, from, to time.Time) ([]repository.CouponUsageRow, error)
	ExecutionSummary(ctx context.Context, from, to time.Time) (repository.ExecutionRow, error)
	NotificationSummary(ctx context.Context, from, to time.Time) (repository.NotificationRow, error)
}

// NewAdminReportService merangkai laporan.
func NewAdminReportService(store reportStore, recorder AdminAudit) AdminReportService {
	return &adminReportService{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
		audit: recorder,
	}
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

// Export mengunduh laporan bisnis sebagai CSV.
//
// Ekspor data uang selalu dicatat agar jejak "siapa membawa keluar angka
// pendapatan" bisa dipertanggungjawabkan.
func (s *adminReportService) Export(ctx context.Context, actor AdminActor, kind, from, to string) (*PositionExport, error) {
	builder := export.New(pagination.ExportMaxRows)
	var rows int

	switch kind {
	case "revenue":
		report, err := s.Revenue(ctx, from, to)
		if err != nil {
			return nil, err
		}
		if err := builder.Header("tanggal", "dibayar_idr", "refund_idr", "jumlah_invoice"); err != nil {
			return nil, err
		}
		for _, day := range report.Daily {
			if err := builder.Row(day.Date,
				strconv.FormatInt(day.PaidIDR, 10),
				strconv.FormatInt(day.RefundIDR, 10),
				strconv.FormatInt(day.Count, 10),
			); err != nil {
				return nil, err
			}
		}
		// Baris penutup berisi angka yang dipakai laporan: tanpa ini pembaca
		// harus menjumlahkan sendiri dan mudah salah.
		if err := builder.Row("TOTAL",
			strconv.FormatInt(report.GrossIDR, 10),
			strconv.FormatInt(report.RefundIDR, 10),
			"",
		); err != nil {
			return nil, err
		}

	case "customers":
		report, err := s.Customers(ctx, from, to)
		if err != nil {
			return nil, err
		}
		if err := builder.Header("metrik", "nilai"); err != nil {
			return nil, err
		}
		metrics := [][2]string{
			{"pengguna_baru", strconv.FormatInt(report.NewUsers, 10)},
			{"pengguna_berbayar", strconv.FormatInt(report.PaidUsers, 10)},
			{"berhenti", strconv.FormatInt(report.Canceled, 10)},
			{"trial_dimulai", strconv.FormatInt(report.TrialStarted, 10)},
		}
		for _, metric := range metrics {
			if err := builder.Row(metric[0], metric[1]); err != nil {
				return nil, err
			}
		}

	case "coupons":
		report, err := s.Coupons(ctx, from, to)
		if err != nil {
			return nil, err
		}
		if err := builder.Header("kode", "pemakaian", "diskon_idr"); err != nil {
			return nil, err
		}
		for _, item := range report.Items {
			if err := builder.Row(item.Code,
				strconv.FormatInt(item.Uses, 10),
				strconv.FormatInt(item.DiscountIDR, 10),
			); err != nil {
				return nil, err
			}
		}

	default:
		return nil, fmt.Errorf("%w: jenis laporan harus revenue, customers, atau coupons", apperror.ErrBadRequest)
	}
	rows = builder.Rows()

	if s.audit != nil {
		if err := s.audit.Record(ctx, AuditEntry{
			ActorID: actor.ID, Action: audit.BusinessReportExport,
			TargetType: "report", TargetID: kind, RequestID: actor.RequestID,
			Reason: "mengekspor laporan bisnis " + kind,
		}); err != nil {
			return nil, err
		}
	}

	return &PositionExport{
		Filename: kind + "-" + time.Now().UTC().Format("20060102-150405") + ".csv",
		Content:  builder.Finished(),
		Rows:     rows,
	}, nil
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
