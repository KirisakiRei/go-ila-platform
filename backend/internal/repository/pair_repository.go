package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
)

// PairFilter menyaring daftar pair saat dibaca.
// Zero value berarti tanpa saringan apa pun.
type PairFilter struct {
	CategoryID string
	GroupID    string
	// OnlyEnabled menyembunyikan kategori, grup, atau pair yang dinonaktifkan.
	OnlyEnabled bool
}

// pairRepository menyimpan katalog kategori, grup, dan pair.
type pairRepository struct {
	db *gorm.DB
}

// NewPairRepository membuat repository katalog pair.
func NewPairRepository(db *gorm.DB) PairRepository {
	return &pairRepository{db: db}
}

// --- Kategori ---------------------------------------------------------------

func (r *pairRepository) CreateCategory(ctx context.Context, category *model.PairCategory) error {
	if err := r.db.WithContext(ctx).Create(category).Error; err != nil {
		return fmt.Errorf("membuat kategori pair: %w", err)
	}
	return nil
}

func (r *pairRepository) FindCategoryByID(ctx context.Context, id string) (*model.PairCategory, error) {
	var category model.PairCategory
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca kategori pair: %w", err)
	}
	return &category, nil
}

func (r *pairRepository) FindCategoryByKey(ctx context.Context, key string) (*model.PairCategory, error) {
	var category model.PairCategory
	err := r.db.WithContext(ctx).Where("LOWER(key) = ?", strings.ToLower(strings.TrimSpace(key))).First(&category).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca kategori pair: %w", err)
	}
	return &category, nil
}

func (r *pairRepository) ListCategories(ctx context.Context, onlyEnabled bool) ([]model.PairCategory, error) {
	var list []model.PairCategory
	q := r.db.WithContext(ctx)
	if onlyEnabled {
		q = q.Where("is_enabled = ?", true)
	}
	if err := q.Order("sort_order ASC, label ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("membaca kategori pair: %w", err)
	}
	return list, nil
}

func (r *pairRepository) UpdateCategory(ctx context.Context, category *model.PairCategory) error {
	if err := r.db.WithContext(ctx).Save(category).Error; err != nil {
		return fmt.Errorf("memperbarui kategori pair: %w", err)
	}
	return nil
}

func (r *pairRepository) DeleteCategory(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Unscoped().Delete(&model.PairCategory{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("menghapus kategori pair: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// --- Grup -------------------------------------------------------------------

func (r *pairRepository) CreateGroup(ctx context.Context, group *model.PairGroup) error {
	if err := r.db.WithContext(ctx).Create(group).Error; err != nil {
		return fmt.Errorf("membuat grup pair: %w", err)
	}
	return nil
}

func (r *pairRepository) FindGroupByID(ctx context.Context, id string) (*model.PairGroup, error) {
	var group model.PairGroup
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca grup pair: %w", err)
	}
	return &group, nil
}

func (r *pairRepository) ListGroups(ctx context.Context, categoryID string, onlyEnabled bool) ([]model.PairGroup, error) {
	var list []model.PairGroup
	q := r.db.WithContext(ctx)
	if categoryID != "" {
		q = q.Where("category_id = ?", categoryID)
	}
	if onlyEnabled {
		q = q.Where("is_enabled = ?", true)
	}
	if err := q.Order("sort_order ASC, label ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("membaca grup pair: %w", err)
	}
	return list, nil
}

func (r *pairRepository) UpdateGroup(ctx context.Context, group *model.PairGroup) error {
	if err := r.db.WithContext(ctx).Save(group).Error; err != nil {
		return fmt.Errorf("memperbarui grup pair: %w", err)
	}
	return nil
}

func (r *pairRepository) DeleteGroup(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Unscoped().Delete(&model.PairGroup{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("menghapus grup pair: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// --- Pair -------------------------------------------------------------------

func (r *pairRepository) CreatePair(ctx context.Context, pair *model.TradingPair) error {
	if err := r.db.WithContext(ctx).Create(pair).Error; err != nil {
		return fmt.Errorf("membuat pair: %w", err)
	}
	return nil
}

func (r *pairRepository) FindPairByID(ctx context.Context, id string) (*model.TradingPair, error) {
	var pair model.TradingPair
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&pair).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.ErrNotFound
		}
		return nil, fmt.Errorf("membaca pair: %w", err)
	}
	return &pair, nil
}

// FindPairBySymbol mencari pair berdasarkan simbol kanonik ATAU alias.
// Pencocokan case-insensitive agar "gold" tetap menemukan "XAUUSD".
func (r *pairRepository) FindPairBySymbol(ctx context.Context, symbol string) (*model.TradingPair, error) {
	needle := strings.ToUpper(strings.TrimSpace(symbol))
	if needle == "" {
		return nil, apperror.ErrNotFound
	}

	var pairs []model.TradingPair
	if err := r.db.WithContext(ctx).Find(&pairs).Error; err != nil {
		return nil, fmt.Errorf("membaca pair: %w", err)
	}
	for i := range pairs {
		if pairMatchesSymbol(pairs[i], needle) {
			return &pairs[i], nil
		}
	}
	return nil, apperror.ErrNotFound
}

// pairMatchesSymbol mencocokkan simbol atau salah satu aliasnya.
func pairMatchesSymbol(pair model.TradingPair, needleUpper string) bool {
	if strings.EqualFold(pair.Symbol, needleUpper) {
		return true
	}
	for _, alias := range strings.Split(pair.Aliases, ",") {
		if strings.EqualFold(strings.TrimSpace(alias), needleUpper) {
			return true
		}
	}
	return false
}

func (r *pairRepository) ListPairs(ctx context.Context, filter PairFilter) ([]model.TradingPair, error) {
	var list []model.TradingPair
	q := r.db.WithContext(ctx)
	if filter.CategoryID != "" {
		q = q.Where("category_id = ?", filter.CategoryID)
	}
	if filter.GroupID != "" {
		q = q.Where("group_id = ?", filter.GroupID)
	}
	if filter.OnlyEnabled {
		q = q.Where("is_enabled = ?", true)
	}
	if err := q.Order("sort_order ASC, symbol ASC").Find(&list).Error; err != nil {
		return nil, fmt.Errorf("membaca pair: %w", err)
	}
	return list, nil
}

func (r *pairRepository) UpdatePair(ctx context.Context, pair *model.TradingPair) error {
	if err := r.db.WithContext(ctx).Save(pair).Error; err != nil {
		return fmt.Errorf("memperbarui pair: %w", err)
	}
	return nil
}

func (r *pairRepository) DeletePair(ctx context.Context, id string) error {
	res := r.db.WithContext(ctx).Unscoped().Delete(&model.TradingPair{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("menghapus pair: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// CountActiveSubscriptionsBySymbol menghitung langganan aktif yang memakai
// simbol tertentu. Dipakai admin service untuk menolak penghapusan atau
// penonaktifan pair yang masih dipakai user.
func (r *pairRepository) CountActiveSubscriptionsBySymbol(ctx context.Context, symbol string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Subscription{}).
		Where("symbol = ? AND is_active = ?", symbol, true).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("menghitung langganan pair: %w", err)
	}
	return count, nil
}
