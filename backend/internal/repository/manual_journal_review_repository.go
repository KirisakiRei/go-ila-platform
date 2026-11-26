package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"ila/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ManualJournalReviewRepository struct{ db *gorm.DB }

func NewManualJournalReviewRepository(db *gorm.DB) *ManualJournalReviewRepository {
	return &ManualJournalReviewRepository{db: db}
}

// ReviewAccounts uses keyset pages so every active account is polled without an unbounded list.
func (r *ManualJournalReviewRepository) ReviewAccounts(ctx context.Context, after string) ([]model.ManualJournalAccount, error) {
	var rows []model.ManualJournalAccount
	err := r.db.WithContext(ctx).Select("manual_journal_accounts.id, manual_journal_accounts.user_id").
		Joins("JOIN \"user\" AS review_user ON review_user.id = manual_journal_accounts.user_id AND review_user.deleted_at IS NULL AND review_user.status = ? AND review_user.suspended_at IS NULL AND review_user.banned_at IS NULL AND review_user.deleted_at_user IS NULL", "active").
		Where("manual_journal_accounts.id > ? AND archived = ?", after, false).Order("manual_journal_accounts.id").Limit(100).Find(&rows).Error
	return rows, err
}

// Fingerprint includes tombstones, so edits, deletion and same-count replacements invalidate the cache.
func (r *ManualJournalReviewRepository) Fingerprint(ctx context.Context, user, account string) (string, error) {
	h := sha256.New()
	for _, entity := range []any{&model.ManualJournalAccount{}, &model.ManualJournalEntry{}, &model.ManualJournalCapital{}} {
		after := ""
		for {
			var rows []struct {
				ID        string
				UpdatedAt time.Time
				DeletedAt gorm.DeletedAt
			}
			q := r.db.WithContext(ctx).Unscoped().Model(entity).Select("id, updated_at, deleted_at").Where("user_id = ? AND id > ?", user, after)
			if _, ok := entity.(*model.ManualJournalAccount); ok {
				q = q.Where("id = ?", account)
			} else {
				q = q.Where("account_id = ?", account)
			}
			if err := q.Order("id").Limit(100).Find(&rows).Error; err != nil {
				return "", err
			}
			if err := json.NewEncoder(h).Encode(rows); err != nil {
				return "", err
			}
			if len(rows) < 100 {
				break
			}
			after = rows[len(rows)-1].ID
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func reviewKey(db *gorm.DB, user, account, day string) *gorm.DB {
	return db.Where("user_id = ? AND account_id = ? AND date = ?", user, account, day)
}
func (r *ManualJournalReviewRepository) Prepare(ctx context.Context, row model.ManualJournalReview) (*model.ManualJournalReview, error) {
	db := r.db.WithContext(ctx)
	placeholder := model.ManualJournalReview{UserID: row.UserID, AccountID: row.AccountID, Date: row.Date, Status: "pending"}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&placeholder).Error; err != nil {
		return nil, err
	}
	if err := reviewGeneration(reviewKey(db.Model(&model.ManualJournalReview{}), row.UserID, row.AccountID, row.Date), row).Where("revision <> ?", row.Revision).Updates(map[string]any{"settings_generation": row.SettingsGeneration, "revision": row.Revision, "status": "pending", "text": "", "source": "", "claim": "", "lease_until": time.Time{}, "updated_at": row.UpdatedAt}).Error; err != nil {
		return nil, err
	}
	var out model.ManualJournalReview
	err := reviewKey(db, row.UserID, row.AccountID, row.Date).First(&out).Error
	return &out, err
}
func (r *ManualJournalReviewRepository) ClaimReview(ctx context.Context, row model.ManualJournalReview, token string, now time.Time) (bool, error) {
	q := reviewGeneration(reviewKey(r.db.WithContext(ctx).Model(&model.ManualJournalReview{}), row.UserID, row.AccountID, row.Date), row).Where("settings_generation = ?", row.SettingsGeneration).Where("revision = ? AND (status = ? OR (status = ? AND lease_until < ?))", row.Revision, "pending", "running", now).Updates(map[string]any{"status": "running", "claim": token, "lease_until": now.Add(time.Minute), "updated_at": now})
	return q.RowsAffected == 1, q.Error
}
func (r *ManualJournalReviewRepository) FinishReview(ctx context.Context, row model.ManualJournalReview, token, text, source string, now time.Time) error {
	return reviewGeneration(reviewKey(r.db.WithContext(ctx).Model(&model.ManualJournalReview{}), row.UserID, row.AccountID, row.Date), row).Where("settings_generation = ?", row.SettingsGeneration).Where("revision = ? AND claim = ? AND status = ? AND lease_until > ?", row.Revision, token, "running", now).Updates(map[string]any{"status": "ready", "text": text, "source": source, "claim": "", "updated_at": now}).Error
}
func (r *ManualJournalReviewRepository) ReviewSettings(ctx context.Context, user, account string) ([]model.ManualJournalReviewSetting, error) {
	var rows []model.ManualJournalReviewSetting
	err := r.db.WithContext(ctx).Where("user_id = ? AND account_id IN ?", user, []string{"", account}).Limit(2).Find(&rows).Error
	return rows, err
}
func (r *ManualJournalReviewRepository) SaveReviewSetting(ctx context.Context, row model.ManualJournalReviewSetting) error {
	row.Generation = 1
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "account_id"}}, DoUpdates: clause.Assignments(map[string]any{"ai_enabled": row.AIEnabled, "updated_at": row.UpdatedAt, "generation": gorm.Expr("manual_journal_review_settings.generation + 1")})}).Create(&row).Error; err != nil {
			return err
		}
		q := tx.Model(&model.ManualJournalReview{}).Where("user_id = ?", row.UserID)
		if row.AccountID != "" {
			q = q.Where("account_id = ?", row.AccountID)
		}
		// Revoke in-flight claims immediately when privacy preferences change.
		return q.Updates(map[string]any{"revision": "", "status": "pending", "text": "", "source": "", "claim": "", "lease_until": time.Time{}}).Error
	})
}

// Both preference scopes advance monotonically, including opt-out followed by opt-in.
func reviewGeneration(db *gorm.DB, row model.ManualJournalReview) *gorm.DB {
	return db.Where("(SELECT COALESCE(SUM(generation), 0) FROM manual_journal_review_settings WHERE user_id = ? AND account_id IN ?) = ?", row.UserID, []string{"", row.AccountID}, row.SettingsGeneration)
}

// AuthorizeReview re-reads permission and the live claim together immediately before each send.
func (r *ManualJournalReviewRepository) AuthorizeReview(ctx context.Context, row model.ManualJournalReview, token string, now time.Time) (bool, error) {
	var count int64
	q := reviewGeneration(reviewKey(r.db.WithContext(ctx).Model(&model.ManualJournalReview{}), row.UserID, row.AccountID, row.Date), row).
		Where("settings_generation = ? AND revision = ? AND claim = ? AND status = ? AND lease_until > ?", row.SettingsGeneration, row.Revision, token, "running", now).
		Where("EXISTS (SELECT 1 FROM \"user\" AS review_user WHERE review_user.id = ? AND review_user.deleted_at IS NULL AND review_user.status = ? AND review_user.suspended_at IS NULL AND review_user.banned_at IS NULL AND review_user.deleted_at_user IS NULL)", row.UserID, "active").
		Where("EXISTS (SELECT 1 FROM manual_journal_review_settings WHERE user_id = ? AND account_id = ? AND ai_enabled = ?)", row.UserID, "", true).
		Where("NOT EXISTS (SELECT 1 FROM manual_journal_review_settings WHERE user_id = ? AND account_id IN ? AND ai_enabled = ?)", row.UserID, []string{"", row.AccountID}, false)
	err := q.Count(&count).Error
	return count == 1, err
}
