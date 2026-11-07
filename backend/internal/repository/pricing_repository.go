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

type pricingRepository struct {
	db *gorm.DB
}

// NewPricingRepository membuat data harga, kupon, dan kredit.
func NewPricingRepository(db *gorm.DB) *pricingRepository {
	return &pricingRepository{db: db}
}

func (r *pricingRepository) ListPlans(ctx context.Context) ([]model.Plan, error) {
	var plans []model.Plan
	if err := r.db.WithContext(ctx).Order("sort_order ASC, code ASC").Find(&plans).Error; err != nil {
		return nil, fmt.Errorf("membaca paket: %w", err)
	}
	return plans, nil
}

func (r *pricingRepository) FindPlan(ctx context.Context, code string) (*model.Plan, error) {
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

func (r *pricingRepository) SavePlan(ctx context.Context, plan *model.Plan, features map[string]string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		plan.UpdatedAt = time.Now().UTC()
		if err := tx.Save(plan).Error; err != nil {
			return fmt.Errorf("menyimpan paket: %w", err)
		}
		for key, value := range features {
			row := &model.PlanFeature{PlanCode: plan.Code, FeatureKey: key, Value: value}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "plan_code"}, {Name: "feature_key"}},
				DoUpdates: clause.AssignmentColumns([]string{"value"}),
			}).Create(row).Error; err != nil {
				return fmt.Errorf("menyimpan fitur paket: %w", err)
			}
		}
		return nil
	})
}

func (r *pricingRepository) FeaturesOfPlan(ctx context.Context, code string) (map[string]string, error) {
	var rows []model.PlanFeature
	if err := r.db.WithContext(ctx).Where("plan_code = ?", code).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca fitur paket: %w", err)
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.FeatureKey] = row.Value
	}
	return out, nil
}

func (r *pricingRepository) ActiveOverride(ctx context.Context, planCode string, at time.Time) (*model.PriceOverride, error) {
	var row model.PriceOverride
	err := r.db.WithContext(ctx).
		Where("plan_code = ? AND is_active = ? AND starts_at <= ? AND ends_at > ?", planCode, true, at, at).
		Order("starts_at DESC").Limit(1).Find(&row).Error
	if err != nil {
		return nil, fmt.Errorf("membaca harga kampanye: %w", err)
	}
	if row.ID == "" {
		return nil, nil
	}
	return &row, nil
}

func (r *pricingRepository) FindGuard(ctx context.Context, planCode string) (*model.PricingGuard, error) {
	var row model.PricingGuard
	err := r.db.WithContext(ctx).Where("plan_code = ?", planCode).Limit(1).Find(&row).Error
	if err != nil {
		return nil, fmt.Errorf("membaca batas bawah harga: %w", err)
	}
	if row.PlanCode == "" {
		return nil, nil
	}
	return &row, nil
}

func (r *pricingRepository) SaveGuard(ctx context.Context, guard *model.PricingGuard) error {
	guard.UpdatedAt = time.Now().UTC()
	if err := r.db.WithContext(ctx).Save(guard).Error; err != nil {
		return fmt.Errorf("menyimpan batas bawah harga: %w", err)
	}
	return nil
}

func (r *pricingRepository) FindCouponByCode(ctx context.Context, code string) (*model.Coupon, error) {
	var coupon model.Coupon
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&coupon).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca kupon: %w", err)
	}
	return &coupon, nil
}

func (r *pricingRepository) CountRedemptions(ctx context.Context, couponID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.CouponRedemption{}).
		Where("coupon_id = ?", couponID).Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung pemakaian kupon: %w", err)
	}
	return total, nil
}

func (r *pricingRepository) CountUserRedemptions(ctx context.Context, couponID, userID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.CouponRedemption{}).
		Where("coupon_id = ? AND user_id = ?", couponID, userID).Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung pemakaian kupon pengguna: %w", err)
	}
	return total, nil
}

func (r *pricingRepository) CountUserInvoices(ctx context.Context, userID string) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Table("invoice").
		Where("user_id = ? AND status = ?", userID, "paid").Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung invoice lunas pengguna: %w", err)
	}
	return total, nil
}

func (r *pricingRepository) CreditBalance(ctx context.Context, userID string) (int64, error) {
	var balance model.CreditBalance
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Limit(1).Find(&balance).Error
	if err != nil {
		return 0, fmt.Errorf("membaca saldo kredit: %w", err)
	}
	return balance.BalanceIDR, nil
}

// ApplyCredit mengubah saldo dan menulis buku besar dalam satu transaksi.
// Saldo negatif ditolak di sini dan oleh CHECK constraint di tabel.
func (r *pricingRepository) ApplyCredit(ctx context.Context, tx *ApplyCreditInput) (int64, error) {
	if tx.DeltaIDR == 0 {
		return 0, errors.Join(apperror.ErrBadRequest, errors.New("perubahan kredit tidak boleh nol"))
	}

	var after int64
	err := r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		// Kunci baris agar dua permintaan bersamaan tidak saling menimpa.
		var balance model.CreditBalance
		err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", tx.UserID).Limit(1).Find(&balance).Error
		if err != nil {
			return err
		}
		current := balance.BalanceIDR
		next := current + tx.DeltaIDR
		if next < 0 {
			return errors.Join(apperror.ErrBadRequest, errors.New("saldo kredit tidak boleh negatif"))
		}

		now := time.Now().UTC()
		row := &model.CreditBalance{UserID: tx.UserID, BalanceIDR: next, UpdatedAt: now}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"balance_idr", "updated_at"}),
		}).Create(row).Error; err != nil {
			return fmt.Errorf("menyimpan saldo kredit: %w", err)
		}

		entry := &model.CreditTransaction{
			UserID:       tx.UserID,
			DeltaIDR:     tx.DeltaIDR,
			BalanceAfter: next,
			Kind:         tx.Kind,
			InvoiceID:    tx.InvoiceID,
			Reason:       tx.Reason,
			ActorID:      tx.ActorID,
			CreatedAt:    now,
		}
		if entry.ID == "" {
			entry.ID = uuid.NewString()
		}
		if err := db.Create(entry).Error; err != nil {
			return fmt.Errorf("menulis buku besar kredit: %w", err)
		}
		after = next
		return nil
	})
	if err != nil {
		return 0, err
	}
	return after, nil
}

// ApplyCreditInput adalah masukan satu perubahan saldo kredit.
type ApplyCreditInput struct {
	UserID    string
	DeltaIDR  int64
	Kind      string
	InvoiceID *string
	Reason    string
	ActorID   *string
}

func (r *pricingRepository) CreditHistory(ctx context.Context, userID string, limit int) ([]model.CreditTransaction, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows []model.CreditTransaction
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("membaca buku besar kredit: %w", err)
	}
	return rows, nil
}

func (r *pricingRepository) ListCoupons(ctx context.Context, status string, q string) ([]model.Coupon, error) {
	query := r.db.WithContext(ctx).Model(&model.Coupon{})
	if q != "" {
		query = query.Where("code ILIKE ?", q+"%")
	}
	var rows []model.Coupon
	if err := query.Order("created_at DESC").Limit(500).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("membaca kupon: %w", err)
	}
	return rows, nil
}

func (r *pricingRepository) SaveCoupon(ctx context.Context, coupon *model.Coupon) error {
	if err := r.db.WithContext(ctx).Save(coupon).Error; err != nil {
		return fmt.Errorf("menyimpan kupon: %w", err)
	}
	return nil
}

func (r *pricingRepository) FindCouponByID(ctx context.Context, id string) (*model.Coupon, error) {
	var coupon model.Coupon
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&coupon).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca kupon: %w", err)
	}
	return &coupon, nil
}

// SaveRedemption mencatat pemakaian kupon satu invoice.
func (r *pricingRepository) SaveRedemption(ctx context.Context, redemption *model.CouponRedemption) error {
	if redemption.ID == "" {
		redemption.ID = uuid.NewString()
	}
	if redemption.RedeemedAt.IsZero() {
		redemption.RedeemedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Create(redemption).Error; err != nil {
		return fmt.Errorf("mencatat pemakaian kupon: %w", err)
	}
	return nil
}
