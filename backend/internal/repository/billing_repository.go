package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

type billingRepository struct {
	db *gorm.DB
}

// NewBillingRepository membuat data langganan, invoice, dan pembayaran.
func NewBillingRepository(db *gorm.DB) *billingRepository {
	return &billingRepository{db: db}
}

// NextInvoiceNumber menyusun nomor ILA-<tahun>-<urut 6 digit>.
// Urutan diambil dari jumlah invoice tahun berjalan dan diacak ulang saat bentrok.
func (r *billingRepository) NextInvoiceNumber(ctx context.Context, at time.Time) (string, error) {
	var total int64
	year := at.Format("2006")
	prefix := "ILA-" + year + "-"
	err := r.db.WithContext(ctx).Model(&model.Invoice{}).
		Where("number LIKE ?", prefix+"%").Count(&total).Error
	if err != nil {
		return "", fmt.Errorf("menghitung nomor invoice: %w", err)
	}
	return prefix + fmt.Sprintf("%06d", total+1), nil
}

func (r *billingRepository) CreateInvoice(ctx context.Context, invoice *model.Invoice) error {
	if invoice.ID == "" {
		invoice.ID = uuid.NewString()
	}
	if err := r.db.WithContext(ctx).Create(invoice).Error; err != nil {
		return fmt.Errorf("membuat invoice: %w", err)
	}
	return nil
}

func (r *billingRepository) FindInvoiceByNumber(ctx context.Context, number string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.WithContext(ctx).Where("number = ?", number).First(&invoice).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca invoice: %w", err)
	}
	return &invoice, nil
}

func (r *billingRepository) FindInvoiceByID(ctx context.Context, id string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&invoice).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca invoice: %w", err)
	}
	return &invoice, nil
}

func (r *billingRepository) UpdateInvoice(ctx context.Context, invoice *model.Invoice) error {
	invoice.UpdatedAt = time.Now().UTC()
	if err := r.db.WithContext(ctx).Save(invoice).Error; err != nil {
		return fmt.Errorf("memperbarui invoice: %w", err)
	}
	return nil
}

func (r *billingRepository) ListInvoices(ctx context.Context, userID string, limit int) ([]model.Invoice, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []model.Invoice
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca riwayat invoice: %w", err)
	}
	return rows, nil
}

// SavePayment menyimpan pembayaran. Nilai balik false berarti webhook ini
// sudah pernah diproses (duplikat), sehingga periode tidak ditambah lagi.
func (r *billingRepository) SavePayment(ctx context.Context, row *model.Payment) (bool, error) {
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	if row.ReceivedAt.IsZero() {
		row.ReceivedAt = time.Now().UTC()
	}
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "gateway"}, {Name: "gateway_txn_id"}},
		DoNothing: true,
	}).Create(row)
	if res.Error != nil {
		return false, fmt.Errorf("menyimpan pembayaran: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *billingRepository) SaveRedemption(ctx context.Context, redemption *model.CouponRedemption) error {
	if redemption.ID == "" {
		redemption.ID = uuid.NewString()
	}
	if err := r.db.WithContext(ctx).Create(redemption).Error; err != nil {
		return fmt.Errorf("mencatat pemakaian kupon: %w", err)
	}
	return nil
}

func (r *billingRepository) ActiveSubscription(ctx context.Context, userID string) (*model.SubscriptionRecord, error) {
	var row model.SubscriptionRecord
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []string{model.SubStatusActive, model.SubStatusGrace}).
		Order("current_period_end DESC").Limit(1).Find(&row).Error
	if err != nil {
		return nil, fmt.Errorf("membaca langganan: %w", err)
	}
	if row.ID == "" {
		return nil, nil
	}
	return &row, nil
}

func (r *billingRepository) UpsertSubscription(ctx context.Context, sub *model.SubscriptionRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.SubscriptionRecord
		err := tx.Where("user_id = ? AND status IN ?", sub.UserID,
			[]string{model.SubStatusActive, model.SubStatusGrace, model.SubStatusPastDue}).
			Order("current_period_end DESC").Limit(1).Find(&existing).Error
		if err != nil {
			return fmt.Errorf("membaca langganan berjalan: %w", err)
		}

		if existing.ID == "" {
			sub.ID = uuid.NewString()
			if sub.CreatedAt.IsZero() {
				sub.CreatedAt = time.Now().UTC()
			}
			sub.UpdatedAt = time.Now().UTC()
			if err := tx.Create(sub).Error; err != nil {
				return fmt.Errorf("membuat langganan: %w", err)
			}
			return nil
		}

		existing.PlanCode = sub.PlanCode
		existing.Status = sub.Status
		existing.CurrentPeriodStart = sub.CurrentPeriodStart
		existing.CurrentPeriodEnd = sub.CurrentPeriodEnd
		existing.CancelAtPeriodEnd = false
		existing.CanceledAt = nil
		existing.UpdatedAt = time.Now().UTC()
		if err := tx.Save(&existing).Error; err != nil {
			return fmt.Errorf("memperbarui langganan: %w", err)
		}
		return nil
	})
}

// ExtendTier menaikkan tier pengguna dan menandai Runner agar memuat ulang.
func (r *billingRepository) ExtendTier(ctx context.Context, userID, planCode string, start, end time.Time) error {
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).
		Updates(map[string]any{
			"tier":            planCode,
			"tier_started_at": start,
			"tier_expires_at": end,
			"grace_until":     nil,
			"runner_dirty_at": time.Now().UTC(),
		}).Error
	if err != nil {
		return fmt.Errorf("memperpanjang tier: %w", err)
	}
	return nil
}

// StartGrace memindahkan langganan aktif yang sudah lewat masa berlaku ke
// masa tenggang 3 hari. Tier belum turun: entry baru yang diblokir Runner.
func (r *billingRepository) StartGrace(ctx context.Context, at time.Time) ([]string, error) {
	var rows []model.SubscriptionRecord
	err := r.db.WithContext(ctx).
		Where("status = ? AND current_period_end <= ?", model.SubStatusActive, at).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca langganan kedaluwarsa: %w", err)
	}

	ids := make([]string, 0, len(rows))
	graceUntil := at.AddDate(0, 0, 3)
	for _, row := range rows {
		if err := r.db.WithContext(ctx).Model(&model.SubscriptionRecord{}).
			Where("id = ?", row.ID).Update("status", model.SubStatusGrace).Error; err != nil {
			return nil, fmt.Errorf("memasuki masa tenggang: %w", err)
		}
		if err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", row.UserID).
			Updates(map[string]any{"grace_until": graceUntil}).Error; err != nil {
			return nil, fmt.Errorf("menandai masa tenggang pengguna: %w", err)
		}
		ids = append(ids, row.UserID)
	}
	return ids, nil
}

// DowngradeExpired menurunkan tier setelah masa tenggang habis.
// Posisi berjalan TIDAK disentuh: SL dan TP tetap di broker (plans/08 §7.1).
func (r *billingRepository) DowngradeExpired(ctx context.Context, at time.Time) ([]string, error) {
	var rows []model.SubscriptionRecord
	err := r.db.WithContext(ctx).
		Where("status = ? AND current_period_end <= ?", model.SubStatusGrace, at.AddDate(0, 0, -3)).
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca masa tenggang habis: %w", err)
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		if err := r.db.WithContext(ctx).Model(&model.SubscriptionRecord{}).
			Where("id = ?", row.ID).Update("status", model.SubStatusPastDue).Error; err != nil {
			return nil, fmt.Errorf("menutup langganan: %w", err)
		}
		err := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", row.UserID).
			Updates(map[string]any{
				"tier":            model.TierFree,
				"tier_expires_at": nil,
				"grace_until":     nil,
				"runner_dirty_at": time.Now().UTC(),
			}).Error
		if err != nil {
			return nil, fmt.Errorf("menurunkan tier: %w", err)
		}
		ids = append(ids, row.UserID)
	}
	return ids, nil
}

// ApplyCredit meneruskan perubahan saldo ke repository harga agar buku besar
// tetap ditulis oleh satu jalur yang sama.
func (r *billingRepository) ApplyCredit(ctx context.Context, input *ApplyCreditInput) (int64, error) {
	return NewPricingRepository(r.db).ApplyCredit(ctx, input)
}

// FindPlan mencari paket; dibutuhkan billing sebelum menyusun invoice.
func (r *billingRepository) FindPlan(ctx context.Context, code string) (*model.Plan, error) {
	var plan model.Plan
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca paket: %w", err)
	}
	return &plan, nil
}

// CreditBalance membaca saldo kredit pengguna.
func (r *billingRepository) CreditBalance(ctx context.Context, userID string) (int64, error) {
	var row model.CreditBalance
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Limit(1).Find(&row).Error; err != nil {
		return 0, fmt.Errorf("membaca saldo kredit: %w", err)
	}
	return row.BalanceIDR, nil
}

// FindCouponByCode mencari kupon untuk dicatat pemakaiannya saat invoice terbit.
func (r *billingRepository) FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error) {
	var row model.Coupon
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca kupon: %w", err)
	}
	return &row, nil
}

func (r *billingRepository) AddRefund(ctx context.Context, row *model.Refund) error {
	if row.ID == "" {
		row.ID = uuid.NewString()
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return fmt.Errorf("mencatat refund: %w", err)
	}
	return nil
}
