package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ila/backend/internal/model"
)

// ReportRepository membaca angka ringkas untuk laporan bisnis.
// Tidak ada laporan PnL lintas pengguna: angka trading bukan urusan bisnis.
type ReportRepository interface {
	Revenue(ctx context.Context, from, to time.Time) ([]RevenueRow, error)
	CustomerSummary(ctx context.Context, from, to time.Time) (CustomerRow, error)
	CouponUsage(ctx context.Context, from, to time.Time) ([]CouponUsageRow, error)
	ExecutionSummary(ctx context.Context, from, to time.Time) (ExecutionRow, error)
	NotificationSummary(ctx context.Context, from, to time.Time) (NotificationRow, error)
}

// RevenueRow adalah pendapatan satu hari.
type RevenueRow struct {
	Day       time.Time
	PaidIDR   int64
	RefundIDR int64
	Count     int64
}

// CustomerRow adalah ringkasan pelanggan pada satu rentang.
type CustomerRow struct {
	NewUsers     int64
	PaidUsers    int64
	Canceled     int64
	TrialStarted int64
}

// CouponUsageRow adalah pemakaian satu kupon.
type CouponUsageRow struct {
	Code        string
	Uses        int64
	DiscountIDR int64
}

// ExecutionRow adalah ringkasan eksekusi.
type ExecutionRow struct {
	Total  int64
	Closed int64
	DryRun int64
}

// NotificationRow adalah ringkasan pengiriman notifikasi.
type NotificationRow struct {
	Sent   int64
	Failed int64
}

type reportRepository struct {
	db *gorm.DB
}

// NewReportRepository membuat pembaca laporan.
func NewReportRepository(db *gorm.DB) ReportRepository {
	return &reportRepository{db: db}
}

// Revenue menghitung hanya invoice lunas. Refund dikurangkan terpisah agar
// tidak terhitung dua kali (plans/15).
func (r *reportRepository) Revenue(ctx context.Context, from, to time.Time) ([]RevenueRow, error) {
	const query = `SELECT date_trunc('day', paid_at) AS day,
			COALESCE(SUM(total_idr), 0) AS paid_idr,
			0 AS refund_idr,
			COUNT(*) AS count
		FROM invoice
		WHERE status = 'paid' AND paid_at >= ? AND paid_at < ?
		GROUP BY 1 ORDER BY 1`
	var rows []RevenueRow
	if err := r.db.WithContext(ctx).Raw(query, from, to).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca pendapatan: %w", err)
	}

	var refunds []struct {
		Day      time.Time
		TotalIDR int64
	}
	refundQuery := `SELECT date_trunc('day', created_at) AS day, COALESCE(SUM(amount_idr), 0) AS total_idr
		FROM refund WHERE created_at >= ? AND created_at < ? GROUP BY 1`
	if err := r.db.WithContext(ctx).Raw(refundQuery, from, to).Scan(&refunds).Error; err != nil {
		return nil, fmt.Errorf("membaca refund: %w", err)
	}

	// Refund ditempel ke hari yang sama; hari tanpa pendapatan tetap muncul
	// bila ada refund agar tidak ada angka yang hilang.
	index := make(map[string]int, len(rows))
	for i, row := range rows {
		index[row.Day.Format("2006-01-02")] = i
	}
	for _, refund := range refunds {
		key := refund.Day.Format("2006-01-02")
		pos, ok := index[key]
		if !ok {
			rows = append(rows, RevenueRow{Day: refund.Day})
			pos = len(rows) - 1
			index[key] = pos
		}
		rows[pos].RefundIDR = refund.TotalIDR
	}
	return rows, nil
}

func (r *reportRepository) CustomerSummary(ctx context.Context, from, to time.Time) (CustomerRow, error) {
	var row CustomerRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM "user" WHERE created_at >= ? AND created_at < ? AND deleted_at IS NULL) AS new_users,
			(SELECT COUNT(DISTINCT user_id) FROM invoice WHERE status = 'paid' AND paid_at >= ? AND paid_at < ?) AS paid_users,
			(SELECT COUNT(*) FROM subscription_record WHERE status = 'canceled' AND updated_at >= ? AND updated_at < ?) AS canceled,
			(SELECT COUNT(*) FROM "user" WHERE trial_started_at >= ? AND trial_started_at < ?) AS trial_started`,
		from, to, from, to, from, to, from, to).Scan(&row).Error
	if err != nil {
		return CustomerRow{}, fmt.Errorf("membaca ringkasan pelanggan: %w", err)
	}
	return row, nil
}

func (r *reportRepository) CouponUsage(ctx context.Context, from, to time.Time) ([]CouponUsageRow, error) {
	const query = `SELECT c.code, COUNT(r.id) AS uses, COALESCE(SUM(r.discount_idr), 0) AS discount_idr
		FROM coupon c
		LEFT JOIN coupon_redemption r ON r.coupon_id = c.id AND r.redeemed_at >= ? AND r.redeemed_at < ?
		GROUP BY c.code ORDER BY uses DESC`
	var rows []CouponUsageRow
	if err := r.db.WithContext(ctx).Raw(query, from, to).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca pemakaian kupon: %w", err)
	}
	return rows, nil
}

func (r *reportRepository) ExecutionSummary(ctx context.Context, from, to time.Time) (ExecutionRow, error) {
	var row ExecutionRow
	err := r.db.WithContext(ctx).Model(&model.Position{}).
		Select(`COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status LIKE 'CLOSED%') AS closed,
			COUNT(*) FILTER (WHERE is_dry_run) AS dry_run`).
		Where("opened_at >= ? AND opened_at < ?", from, to).Scan(&row).Error
	if err != nil {
		return ExecutionRow{}, fmt.Errorf("membaca ringkasan eksekusi: %w", err)
	}
	return row, nil
}

func (r *reportRepository) NotificationSummary(ctx context.Context, from, to time.Time) (NotificationRow, error) {
	var row NotificationRow
	err := r.db.WithContext(ctx).Model(&model.NotificationLog{}).
		Select(`COUNT(*) FILTER (WHERE status = 'sent') AS sent,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed`).
		Where("created_at >= ? AND created_at < ?", from, to).Scan(&row).Error
	if err != nil {
		return NotificationRow{}, fmt.Errorf("membaca ringkasan notifikasi: %w", err)
	}
	return row, nil
}
