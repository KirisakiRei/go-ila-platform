package repository

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"ila/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const SignalEvalBatchLimit = 100

// SignalEvalCandidate carries the immutable setup and the lease used for CAS writes.
type SignalEvalCandidate struct {
	UserSignal model.UserSignal
	Signal     model.Signal
	Lease      time.Time
}

type SignalEvalRepository struct{ db *gorm.DB }

func NewSignalEvalRepository(db *gorm.DB) *SignalEvalRepository { return &SignalEvalRepository{db: db} }

// Exclude relations even if user_signal.position_id was not populated after a crash.
func nonexecutedEval(q *gorm.DB) *gorm.DB {
	return q.Where("COALESCE(user_signal.position_id, '') = '' AND user_signal.status <> ?", model.SignalStatusConsumed).
		Where("NOT EXISTS (SELECT 1 FROM position p WHERE p.user_id = user_signal.user_id AND p.source_signal_id = user_signal.signal_id)").
		Where("NOT EXISTS (SELECT 1 FROM execution_record e WHERE e.user_id = user_signal.user_id AND e.signal_id = user_signal.signal_id AND (COALESCE(e.position_id, '') <> '' OR e.status IN ?))", []string{"CLAIMED", "COMPLETED"})
}

// Persisted roots are never re-deduplicated against mutable signal metadata.
// Unclassified rows remain visible until the bounded classifier reaches them.
const canonicalEvalSQL = "(COALESCE(user_signal.eval_canonical_id, '') = '' OR user_signal.eval_canonical_id = user_signal.id)"

func openEval(q *gorm.DB) *gorm.DB {
	return nonexecutedEval(q).Where("COALESCE(user_signal.eval_status, '') IN ?", []string{"", model.EvalPending, model.EvalRunning, model.EvalUnknown}).
		Where("NOT (user_signal.eval_status = ? AND COALESCE(user_signal.eval_evidence, '') <> '')", model.EvalUnknown)
}

// Claim durably rotates attempted work, including feed failures. A crashed worker's
// lease becomes eligible again without needing a browser request or an expiry event.
func (r *SignalEvalRepository) Claim(ctx context.Context, now time.Time, limit int) ([]SignalEvalCandidate, error) {
	if limit <= 0 || limit > SignalEvalBatchLimit {
		limit = SignalEvalBatchLimit
	}
	var result []SignalEvalCandidate
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := classifyEvalCohorts(ctx, tx); err != nil {
			return err
		}
		var rows []model.UserSignal
		q := openEval(tx.Model(&model.UserSignal{})).Where("eval_canonical_id = user_signal.id").
			Where("eval_next_at IS NULL OR eval_next_at <= ?", now).
			Where("COALESCE(eval_snapshot, '') <> ''").
			Order("COALESCE(eval_next_at, user_signal.created_at), user_signal.id").
			Limit(limit).Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		if err := q.Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]string, 0, len(rows))
		signalIDs := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
			signalIDs = append(signalIDs, row.SignalID)
		}
		var signals []model.Signal
		if err := tx.Where("signal_id IN ?", signalIDs).Find(&signals).Error; err != nil {
			return err
		}
		byID := make(map[string]model.Signal, len(signals))
		for _, signal := range signals {
			byID[signal.SignalID] = signal
		}
		lease := now.Add(2 * time.Minute)
		if err := tx.Model(&model.UserSignal{}).Where("id IN ?", ids).UpdateColumn("eval_next_at", lease).Error; err != nil {
			return err
		}
		for _, row := range rows {
			signal := byID[row.SignalID]
			if row.EvalSnapshot != "" {
				signal = model.Signal{}
				if err := json.Unmarshal([]byte(row.EvalSnapshot), &signal); err != nil {
					return fmt.Errorf("decode eval snapshot %s: %w", row.ID, err)
				}
			} else {
				data, err := json.Marshal(signal)
				if err != nil {
					return err
				}
				row.EvalSnapshot = string(data)
				if err := tx.Model(&model.UserSignal{}).Where("id = ?", row.ID).UpdateColumn("eval_snapshot", row.EvalSnapshot).Error; err != nil {
					return err
				}
			}
			result = append(result, SignalEvalCandidate{UserSignal: row, Signal: signal, Lease: lease})
		}
		return nil
	})
	return result, err
}

// classifyEvalCohorts freezes at most 100 publications in chronological order.
// Serializing this short transaction prevents concurrent classifiers from making
// two roots for the same publication window; evaluation/feed work stays outside.
func classifyEvalCohorts(ctx context.Context, tx *gorm.DB) error {
	if tx.Dialector.Name() == "postgres" {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(712346901)").Error; err != nil {
			return err
		}
	}
	var rows []model.UserSignal
	if err := nonexecutedEval(tx.WithContext(ctx).Model(&model.UserSignal{})).
		Joins("JOIN signal ON signal.signal_id = user_signal.signal_id").
		Where("COALESCE(eval_canonical_id, '') = ''").
		Where("COALESCE(eval_snapshot, '') <> '' OR (signal.entry_price > 0 AND signal.stop_loss > 0 AND signal.take_profit > 0)").
		Select("user_signal.*").Order("signal.published_at, user_signal.id").Limit(SignalEvalBatchLimit).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.SignalID)
	}
	var signals []model.Signal
	if err := tx.Unscoped().Where("signal_id IN ?", ids).Find(&signals).Error; err != nil {
		return err
	}
	byID := make(map[string]model.Signal, len(signals))
	for _, signal := range signals {
		byID[signal.SignalID] = signal
	}
	type frozen struct {
		row    model.UserSignal
		signal model.Signal
		key    string
	}
	pending := make([]frozen, 0, len(rows))
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		signal := byID[row.SignalID]
		if row.EvalSnapshot != "" {
			signal = model.Signal{}
			if err := json.Unmarshal([]byte(row.EvalSnapshot), &signal); err != nil {
				return fmt.Errorf("decode eval snapshot %s: %w", row.ID, err)
			}
		}
		identity, err := json.Marshal([]any{signal.Symbol, signal.StrategyVersion, signal.FeedSource, signal.Htf, signal.Ltf, signal.Direction, signal.EntryPrice, signal.StopLoss, signal.TakeProfit})
		if err != nil {
			return err
		}
		key := fmt.Sprintf("%x", sha256.Sum256(identity))
		keys = append(keys, key)
		pending = append(pending, frozen{row, signal, key})
	}
	// ponytail: bounded root lookup; refuse classification rather than silently
	// misgroup beyond 10,000 matching roots. Partition the lookup if this ceiling is reached.
	var roots []model.UserSignal
	if err := tx.Where("eval_canonical_id = id AND eval_cohort_key IN ?", keys).Order("eval_published_at, id").Limit(10001).Find(&roots).Error; err != nil {
		return err
	}
	if len(roots) > 10000 {
		return fmt.Errorf("evaluation cohort root limit exceeded")
	}
	for _, item := range pending {
		row, signal := item.row, item.signal
		canonical := row.ID
		if row.EvalSnapshot == "" {
			for _, root := range roots {
				if root.UserID == row.UserID && root.EvalCohortKey == item.key && root.EvalPublishedAt != nil && root.EvalExpiresAt != nil && !root.EvalPublishedAt.After(signal.PublishedAt) && !root.EvalExpiresAt.Before(signal.PublishedAt) {
					canonical = root.ID
					break
				}
			}
		}
		snapshot := row.EvalSnapshot
		if snapshot == "" {
			data, err := json.Marshal(signal)
			if err != nil {
				return err
			}
			snapshot = string(data)
		}
		updates := map[string]any{"eval_canonical_id": canonical, "eval_cohort_key": item.key, "eval_published_at": signal.PublishedAt, "eval_expires_at": signal.ExpiresAt, "eval_snapshot": snapshot}
		if err := tx.Model(&model.UserSignal{}).Where("id = ? AND COALESCE(eval_canonical_id, '') = ''", row.ID).Updates(updates).Error; err != nil {
			return err
		}
		if canonical == row.ID {
			row.EvalCanonicalID = canonical
			row.EvalCohortKey = item.key
			row.EvalPublishedAt = &signal.PublishedAt
			row.EvalExpiresAt = &signal.ExpiresAt
			roots = append(roots, row)
		}
	}
	return nil
}

// Save checks lease, broker relations, terminality and watermark in the same UPDATE.
func (r *SignalEvalRepository) Save(ctx context.Context, job SignalEvalCandidate, out *model.UserSignalEval) error {
	if out == nil {
		return fmt.Errorf("nil signal evaluation")
	}
	switch out.Status {
	case model.EvalPending, model.EvalRunning, model.EvalTP, model.EvalSL, model.EvalNoEntry, model.EvalUnknown:
	default:
		return fmt.Errorf("invalid evaluation status %q", out.Status)
	}
	entryBasis := "signal_level_at_publication"
	if job.UserSignal.EvalEntryAt != nil && job.UserSignal.EvalCheckedAt == nil {
		entryBasis = "legacy_unverified"
	}
	evidence, err := json.Marshal(struct {
		Setup      json.RawMessage `json:"setup"`
		EntryAt    *time.Time      `json:"entry_at,omitempty"`
		EntryPrice float64         `json:"entry_price"`
		EntryBasis string          `json:"entry_basis"`
		Evaluation string          `json:"evaluation"`
	}{Setup: json.RawMessage(job.UserSignal.EvalSnapshot), EntryAt: out.EntryAt, EntryPrice: job.Signal.EntryPrice, EntryBasis: entryBasis, Evaluation: out.Evidence})
	if err != nil {
		return fmt.Errorf("encode evaluation evidence: %w", err)
	}
	q := openEval(r.db.WithContext(ctx).Model(&model.UserSignal{})).Where("id = ? AND eval_next_at = ?", job.UserSignal.ID, job.Lease)
	if out.CheckedAt != nil {
		q = q.Where("eval_checked_at IS NULL OR eval_checked_at <= ?", out.CheckedAt)
	} else {
		q = q.Where("eval_checked_at IS NULL")
	}
	if out.EntryAt == nil {
		q = q.Where("eval_entry_at IS NULL")
	} else {
		q = q.Where("eval_entry_at IS NULL OR eval_entry_at = ?", out.EntryAt)
	}
	if out.Status == model.EvalPending || out.Status == model.EvalNoEntry {
		q = q.Where("COALESCE(eval_status, '') <> ? AND eval_entry_at IS NULL", model.EvalRunning)
	}
	return q.Updates(map[string]any{"eval_status": out.Status, "eval_entry_at": out.EntryAt, "eval_at": out.At, "eval_price": out.Price, "eval_r_lower": out.RLower, "eval_note": out.Note, "eval_checked_at": out.CheckedAt, "eval_evidence": string(evidence)}).Error
}
