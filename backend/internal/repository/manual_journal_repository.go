package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ManualJournalRepository struct{ db *gorm.DB }

func NewManualJournalRepository(db *gorm.DB) *ManualJournalRepository {
	return &ManualJournalRepository{db: db}
}
func manualLike(s string) string {
	return "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(s) + "%"
}
func (r *ManualJournalRepository) Account(ctx context.Context, user, id string) (*model.ManualJournalAccount, error) {
	var v model.ManualJournalAccount
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", user, id).First(&v).Error
	return &v, manualNotFound(err)
}
func (r *ManualJournalRepository) Capital(ctx context.Context, user, id string) (*model.ManualJournalCapital, error) {
	var v model.ManualJournalCapital
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", user, id).First(&v).Error
	return &v, manualNotFound(err)
}
func (r *ManualJournalRepository) Entry(ctx context.Context, user, id string) (*model.ManualJournalEntry, error) {
	var v model.ManualJournalEntry
	err := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", user, id).First(&v).Error
	return &v, manualNotFound(err)
}
func (r *ManualJournalRepository) Currencies(ctx context.Context, user string, ids []string) ([]string, error) {
	rows := []string{}
	if len(ids) == 0 {
		return rows, nil
	}
	err := r.db.WithContext(ctx).Model(&model.ManualJournalAccount{}).Where("user_id = ? AND id IN ?", user, ids).Distinct("currency").Limit(2).Pluck("currency", &rows).Error
	return rows, err
}

// Stream only balance inputs: exact decimal aggregation belongs in the service, not SQL float casts.
func (r *ManualJournalRepository) BalanceValues(ctx context.Context, user string, ids []string, visit func(string, string, *string) error) (err error) {
	if len(ids) == 0 {
		return nil
	}
	capital := r.db.WithContext(ctx).Model(&model.ManualJournalCapital{}).Select("account_id, kind, amount").Where("user_id = ? AND account_id IN ?", user, ids)
	pnl := r.db.WithContext(ctx).Model(&model.ManualJournalEntry{}).Select("account_id, 'pnl' AS kind, net_pnl AS amount").Where("user_id = ? AND account_id IN ? AND status = ?", user, ids, "closed")
	rows, err := r.db.WithContext(ctx).Raw("? UNION ALL ?", capital, pnl).Rows()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()
	for rows.Next() {
		var account, kind string
		var amount *string
		if err := rows.Scan(&account, &kind, &amount); err != nil {
			return err
		}
		if err := visit(account, kind, amount); err != nil {
			return err
		}
	}
	return rows.Err()
}

func manualNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.ErrNotFound
	}
	return err
}
func (r *ManualJournalRepository) Create(ctx context.Context, v any) error {
	if row, ok := v.(*model.ManualJournalScreenshot); ok {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var entry model.ManualJournalEntry
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND id = ?", row.UserID, row.EntryID).First(&entry).Error; err != nil {
				return manualNotFound(err)
			}
			return tx.Create(row).Error
		})
	}
	return r.db.WithContext(ctx).Create(v).Error
}

// Updates never upsert: a concurrent deletion must not resurrect an entry.
func (r *ManualJournalRepository) Update(ctx context.Context, user, id string, v any) error {
	q := r.db.WithContext(ctx).Model(v).Where("user_id = ? AND id = ?", user, id).Select("*").Omit("id", "user_id", "created_at", "deleted_at").Updates(v)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
func (r *ManualJournalRepository) Delete(ctx context.Context, user, id string, v any) error {
	if _, ok := v.(*model.ManualJournalAccount); ok {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			q := tx.Where("user_id = ? AND id = ?", user, id).Delete(v)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return apperror.ErrNotFound
			}
			var entryIDs []string
			if err := tx.Model(&model.ManualJournalEntry{}).Where("user_id = ? AND account_id = ?", user, id).Pluck("id", &entryIDs).Error; err != nil {
				return err
			}
			if len(entryIDs) > 0 {
				if err := tx.Where("user_id = ? AND entry_id IN ?", user, entryIDs).Delete(&model.ManualJournalScreenshot{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("user_id = ? AND account_id = ?", user, id).Delete(&model.ManualJournalEntry{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id = ? AND account_id = ?", user, id).Delete(&model.ManualJournalCapital{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id = ? AND account_id = ?", user, id).Delete(&model.ManualJournalReview{}).Error; err != nil {
				return err
			}
			return tx.Where("user_id = ? AND account_id = ?", user, id).Delete(&model.ManualJournalReviewSetting{}).Error
		})
	}
	if _, ok := v.(*model.ManualJournalEntry); ok {
		return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			q := tx.Where("user_id = ? AND id = ?", user, id).Delete(v)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 0 {
				return apperror.ErrNotFound
			}
			return tx.Where("user_id = ? AND entry_id = ?", user, id).Delete(&model.ManualJournalScreenshot{}).Error
		})
	}
	q := r.db.WithContext(ctx).Where("user_id = ? AND id = ?", user, id).Delete(v)
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// DeletedAt is the durable cleanup tombstone; retain it until unlink succeeds.
func (r *ManualJournalRepository) ScreenshotTombstones(ctx context.Context, user, entry string) ([]model.ManualJournalScreenshot, error) {
	rows := []model.ManualJournalScreenshot{}
	q := r.db.WithContext(ctx).Unscoped().Where("deleted_at IS NOT NULL")
	if user != "" {
		q = q.Where("user_id = ?", user)
	}
	if entry != "" {
		q = q.Where("entry_id = ?", entry)
	}
	err := q.Order("deleted_at ASC, id ASC").Limit(100).Find(&rows).Error
	return rows, err
}

func (r *ManualJournalRepository) PurgeScreenshot(ctx context.Context, user, id string) error {
	return r.db.WithContext(ctx).Unscoped().Where("user_id = ? AND id = ? AND deleted_at IS NOT NULL", user, id).Delete(&model.ManualJournalScreenshot{}).Error
}

func manualPage(q *gorm.DB, f model.ManualJournalFilter, total *int64) (*gorm.DB, error) {
	if err := q.Count(total).Error; err != nil {
		return nil, err
	}
	return q.Order("created_at DESC, id DESC").Offset((f.Page - 1) * f.PerPage).Limit(f.PerPage), nil
}
func (r *ManualJournalRepository) Accounts(ctx context.Context, user string, f model.ManualJournalFilter) ([]model.ManualJournalAccount, int64, error) {
	rows := []model.ManualJournalAccount{}
	var total int64
	q := r.db.WithContext(ctx).Model(&model.ManualJournalAccount{}).Where("user_id = ?", user)
	if f.Archived != nil {
		q = q.Where("archived = ?", *f.Archived)
	}
	if f.Search != "" {
		s := manualLike(f.Search)
		q = q.Where("(name LIKE ? ESCAPE '!' OR broker LIKE ? ESCAPE '!' OR notes LIKE ? ESCAPE '!')", s, s, s)
	}
	q, err := manualPage(q, f, &total)
	if err != nil {
		return nil, 0, err
	}
	err = q.Find(&rows).Error
	return rows, total, err
}
func (r *ManualJournalRepository) Capitals(ctx context.Context, user string, f model.ManualJournalFilter) ([]model.ManualJournalCapital, int64, error) {
	rows := []model.ManualJournalCapital{}
	var total int64
	q := r.db.WithContext(ctx).Model(&model.ManualJournalCapital{}).Where("user_id = ?", user)
	if f.AccountID != "" {
		q = q.Where("account_id = ?", f.AccountID)
	}
	if f.From != "" {
		q = q.Where("date >= ?", f.From)
	}
	if f.To != "" {
		q = q.Where("date <= ?", f.To)
	}
	if f.Search != "" {
		q = q.Where("notes LIKE ? ESCAPE '!'", manualLike(f.Search))
	}
	q, err := manualPage(q, f, &total)
	if err != nil {
		return nil, 0, err
	}
	err = q.Find(&rows).Error
	return rows, total, err
}
func (r *ManualJournalRepository) Entries(ctx context.Context, user string, f model.ManualJournalFilter, report bool) ([]model.ManualJournalEntry, int64, error) {
	rows := []model.ManualJournalEntry{}
	var total int64
	q := r.db.WithContext(ctx).Model(&model.ManualJournalEntry{}).Where("user_id = ?", user)
	for _, v := range []struct{ col, val string }{{"account_id", f.AccountID}, {"status", f.Status}, {"asset", f.Asset}, {"side", f.Side}, {"session", f.Session}, {"setup", f.Setup}} {
		if v.val != "" {
			q = q.Where(v.col+" = ?", v.val)
		}
	}
	if f.Search != "" {
		s := manualLike(f.Search)
		q = q.Where("(asset LIKE ? ESCAPE '!' OR setup LIKE ? ESCAPE '!' OR notes LIKE ? ESCAPE '!')", s, s, s)
	}
	for _, v := range []struct{ column, value string }{{"tags", f.Tag}, {"emotions", f.Emotion}, {"mistakes", f.Mistake}} {
		if v.value == "" {
			continue
		}
		b, err := json.Marshal(v.value)
		if err != nil {
			return nil, 0, err
		}
		q = q.Where(v.column+" LIKE ? ESCAPE '!'", manualLike(string(b)))
	}
	wib := time.FixedZone("WIB", 7*3600)
	if f.From != "" {
		d, err := time.ParseInLocation("2006-01-02", f.From, wib)
		if err != nil {
			return nil, 0, err
		}
		q = q.Where("COALESCE(closed_at, opened_at, created_at) >= ?", d.UTC())
	}
	if f.To != "" {
		d, err := time.ParseInLocation("2006-01-02", f.To, wib)
		if err != nil {
			return nil, 0, err
		}
		q = q.Where("COALESCE(closed_at, opened_at, created_at) < ?", d.AddDate(0, 0, 1).UTC())
	}
	if report {
		q = q.Order("COALESCE(closed_at, opened_at, created_at) ASC, id ASC").Limit(10001)
	} else {
		var err error
		q, err = manualPage(q, f, &total)
		if err != nil {
			return nil, 0, err
		}
	}
	err := q.Find(&rows).Error
	return rows, total, err
}
func (r *ManualJournalRepository) Import(ctx context.Context, user, token, hash string, rows []model.ManualJournalEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var preview model.ManualJournalImportToken
		if err := tx.Where("id = ? AND user_id = ?", token, user).First(&preview).Error; err != nil {
			return manualNotFound(err)
		}
		if preview.PayloadHash != hash {
			return fmt.Errorf("%w: CSV differs from preview", apperror.ErrBadRequest)
		}
		for _, row := range rows {
			if row.UserID != user {
				return apperror.ErrForbidden
			}
		}
		now := time.Now().UTC()
		// Compare-and-set takes the row write lock; concurrent confirmations cannot both consume it.
		result := tx.Model(&model.ManualJournalImportToken{}).Where("id = ? AND user_id = ? AND payload_hash = ? AND consumed_at IS NULL AND expires_at > ?", token, user, hash, now).Update("consumed_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("%w: preview expired or already confirmed; preview again", apperror.ErrConflict)
		}
		return tx.CreateInBatches(&rows, 50).Error
	})
}
func (r *ManualJournalRepository) Screenshots(ctx context.Context, user, entry string) ([]model.ManualJournalScreenshot, error) {
	rows := []model.ManualJournalScreenshot{}
	err := r.db.WithContext(ctx).Where("user_id = ? AND entry_id = ?", user, entry).Order("created_at ASC").Limit(100).Find(&rows).Error
	return rows, err
}
func (r *ManualJournalRepository) Screenshot(ctx context.Context, user, entry, id string) (*model.ManualJournalScreenshot, error) {
	var row model.ManualJournalScreenshot
	err := r.db.WithContext(ctx).Where("user_id = ? AND entry_id = ? AND id = ?", user, entry, id).First(&row).Error
	return &row, manualNotFound(err)
}
