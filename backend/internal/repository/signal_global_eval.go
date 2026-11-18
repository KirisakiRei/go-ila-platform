package repository

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ila/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GlobalSignalWindow adalah rentang maksimal penggabungan penerbitan ulang
// setup identik pada tabel signal. Selama setup pertama belum selesai (TP/SL)
// dalam jendela ini, terbitan ulang dengan harga Entry/SL/TP yang sama persis
// disatukan ke satu akar kanonik.
const GlobalSignalWindow = 12 * time.Hour

// GlobalSignalEvalCandidate membawa sinyal kanonik dan lease evaluasi.
type GlobalSignalEvalCandidate struct {
	Signal model.Signal
	Lease  time.Time
}

// CanonEvalTimeframe menyamakan penulisan timeframe (mis. 4H vs H4, 5m vs M5)
// agar kunci kohort setup tidak terpecah karena perbedaan gaya penulisan.
func CanonEvalTimeframe(tf string) string {
	n := strings.ToUpper(strings.TrimSpace(tf))
	n = strings.NewReplacer(" ", "", "-", "", "_", "").Replace(n)
	switch n {
	case "D", "D1", "1D":
		return "D"
	case "W", "W1", "1W":
		return "W"
	case "4H", "H4":
		return "H4"
	case "1H", "H1":
		return "H1"
	case "30M", "M30":
		return "M30"
	case "15M", "M15":
		return "M15"
	case "5M", "M5":
		return "M5"
	case "3M", "M3":
		return "M3"
	case "1M", "M1":
		return "M1"
	default:
		return n
	}
}

// SignalCohortIdentityKey menghitung sidik jari setup (simbol, metode, feed,
// pasangan timeframe, arah, entry, SL, TP) selaras dengan logika evaluasi jurnal.
func SignalCohortIdentityKey(signal *model.Signal) (string, error) {
	if signal == nil {
		return "", fmt.Errorf("nil signal")
	}
	identity, err := json.Marshal([]any{
		strings.ToUpper(strings.TrimSpace(signal.Symbol)),
		strings.TrimSpace(signal.StrategyVersion),
		strings.ToUpper(strings.TrimSpace(signal.FeedSource)),
		CanonEvalTimeframe(signal.Htf),
		CanonEvalTimeframe(signal.Ltf),
		strings.ToUpper(strings.TrimSpace(signal.Direction)),
		signal.EntryPrice,
		signal.StopLoss,
		signal.TakeProfit,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(identity)), nil
}

func openGlobalSignalEval(q *gorm.DB) *gorm.DB {
	return q.Where("signal.signal_id NOT LIKE ?", "status-%").
		Where("signal.entry_price > 0 AND signal.stop_loss > 0 AND signal.take_profit > 0").
		Where("signal.status NOT IN ?", []string{model.SignalStatusCancelled, model.SignalStatusInvalidated}).
		Where("COALESCE(signal.setup_eval_status, '') IN ?", []string{"", model.EvalPending, model.EvalRunning, model.EvalUnknown}).
		Where("NOT (signal.setup_eval_status = ? AND COALESCE(signal.setup_evidence, '') <> '')", model.EvalUnknown)
}

// ClaimSignals mengklaim akar setup global yang belum memiliki hasil akhir TP/SL.
func (r *SignalEvalRepository) ClaimSignals(ctx context.Context, now time.Time, limit int) ([]GlobalSignalEvalCandidate, error) {
	if limit <= 0 || limit > SignalEvalBatchLimit {
		limit = SignalEvalBatchLimit
	}
	var result []GlobalSignalEvalCandidate
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := classifyGlobalSignalCohorts(ctx, tx); err != nil {
			return err
		}
		var rows []model.Signal
		q := openGlobalSignalEval(tx.Model(&model.Signal{})).
			Where("signal.setup_canonical_id = signal.signal_id").
			Where("signal.setup_next_at IS NULL OR signal.setup_next_at <= ?", now).
			Order("COALESCE(signal.setup_next_at, signal.published_at), signal.id").
			Limit(limit).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})
		if err := q.Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		lease := now.Add(2 * time.Minute)
		if err := tx.Model(&model.Signal{}).Where("id IN ?", ids).UpdateColumn("setup_next_at", lease).Error; err != nil {
			return err
		}
		for _, row := range rows {
			result = append(result, GlobalSignalEvalCandidate{Signal: row, Lease: lease})
		}
		return nil
	})
	return result, err
}

func classifyGlobalSignalCohorts(ctx context.Context, tx *gorm.DB) error {
	if tx.Dialector.Name() == "postgres" {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(712346902)").Error; err != nil {
			return err
		}
	}
	var rows []model.Signal
	if err := tx.WithContext(ctx).Model(&model.Signal{}).
		Where("COALESCE(setup_canonical_id, '') = ''").
		Where("signal_id NOT LIKE ?", "status-%").
		Where("entry_price > 0 AND stop_loss > 0 AND take_profit > 0").
		Order("published_at ASC, id ASC").
		Limit(SignalEvalBatchLimit).
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	type pendingItem struct {
		row model.Signal
		key string
	}
	pending := make([]pendingItem, 0, len(rows))
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		key, err := SignalCohortIdentityKey(&row)
		if err != nil {
			return err
		}
		keys = append(keys, key)
		pending = append(pending, pendingItem{row: row, key: key})
	}

	var roots []model.Signal
	if err := tx.Where("setup_canonical_id = signal_id AND setup_cohort_key IN ?", keys).
		Order("published_at ASC, id ASC").
		Limit(10001).
		Find(&roots).Error; err != nil {
		return err
	}
	if len(roots) > 10000 {
		return fmt.Errorf("global signal cohort root limit exceeded")
	}

	for _, item := range pending {
		row := item.row
		canonical := row.SignalID
		var matchedRoot *model.Signal
		for i := len(roots) - 1; i >= 0; i-- {
			root := &roots[i]
			if root.SetupCohortKey != item.key {
				continue
			}
			if root.PublishedAt.After(row.PublishedAt) {
				continue
			}
			if row.PublishedAt.Sub(root.PublishedAt) > GlobalSignalWindow {
				continue
			}
			// Bila akar sebelumnya sudah menyentuh TP/SL sebelum sinyal baru ini terbit,
			// maka sinyal baru ini adalah siklus setup baru, bukan duplikat.
			if (root.SetupEvalStatus == model.EvalTP || root.SetupEvalStatus == model.EvalSL) &&
				root.SetupEvalAt != nil && !root.SetupEvalAt.After(row.PublishedAt) {
				continue
			}
			canonical = root.SignalID
			matchedRoot = root
			break
		}

		updates := map[string]any{
			"setup_canonical_id": canonical,
			"setup_cohort_key":   item.key,
		}
		if matchedRoot != nil && matchedRoot.SetupEvalStatus != "" {
			updates["setup_eval_status"] = matchedRoot.SetupEvalStatus
			updates["setup_entry_at"] = matchedRoot.SetupEntryAt
			updates["setup_eval_at"] = matchedRoot.SetupEvalAt
			updates["setup_eval_price"] = matchedRoot.SetupEvalPrice
			updates["setup_eval_r_lower"] = matchedRoot.SetupEvalRLower
			updates["setup_eval_note"] = matchedRoot.SetupEvalNote
			updates["setup_checked_at"] = matchedRoot.SetupCheckedAt
		}
		if err := tx.Model(&model.Signal{}).
			Where("id = ? AND COALESCE(setup_canonical_id, '') = ''", row.ID).
			Updates(updates).Error; err != nil {
			return err
		}
		if canonical == row.SignalID {
			row.SetupCanonicalID = canonical
			row.SetupCohortKey = item.key
			roots = append(roots, row)
		}
	}
	return nil
}

// SaveSignal menyimpan hasil evaluasi setup pada akar signal serta menyalinnya
// ke seluruh baris kembaran dalam kohort yang sama.
func (r *SignalEvalRepository) SaveSignal(ctx context.Context, job GlobalSignalEvalCandidate, out *model.UserSignalEval) error {
	if out == nil {
		return fmt.Errorf("nil signal evaluation")
	}
	switch out.Status {
	case model.EvalPending, model.EvalRunning, model.EvalTP, model.EvalSL, model.EvalNoEntry, model.EvalUnknown:
	default:
		return fmt.Errorf("invalid evaluation status %q", out.Status)
	}

	snapshotBytes, err := json.Marshal(job.Signal)
	if err != nil {
		return fmt.Errorf("encode signal setup snapshot: %w", err)
	}
	evidence, err := json.Marshal(struct {
		Setup      json.RawMessage `json:"setup"`
		EntryAt    *time.Time      `json:"entry_at,omitempty"`
		EntryPrice float64         `json:"entry_price"`
		EntryBasis string          `json:"entry_basis"`
		Evaluation string          `json:"evaluation"`
	}{
		Setup:      snapshotBytes,
		EntryAt:    out.EntryAt,
		EntryPrice: job.Signal.EntryPrice,
		EntryBasis: "signal_level_at_publication",
		Evaluation: out.Evidence,
	})
	if err != nil {
		return fmt.Errorf("encode global signal evaluation evidence: %w", err)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := openGlobalSignalEval(tx.Model(&model.Signal{})).
			Where("id = ? AND setup_next_at = ?", job.Signal.ID, job.Lease)
		if out.CheckedAt != nil {
			q = q.Where("setup_checked_at IS NULL OR setup_checked_at <= ?", out.CheckedAt)
		} else {
			q = q.Where("setup_checked_at IS NULL")
		}
		if out.EntryAt == nil {
			q = q.Where("setup_entry_at IS NULL")
		} else {
			q = q.Where("setup_entry_at IS NULL OR setup_entry_at = ?", out.EntryAt)
		}
		if out.Status == model.EvalPending || out.Status == model.EvalNoEntry {
			q = q.Where("COALESCE(setup_eval_status, '') <> ? AND setup_entry_at IS NULL", model.EvalRunning)
		}

		sharedCols := map[string]any{
			"setup_eval_status":  out.Status,
			"setup_entry_at":     out.EntryAt,
			"setup_eval_at":      out.At,
			"setup_eval_price":   out.Price,
			"setup_eval_r_lower": out.RLower,
			"setup_eval_note":    out.Note,
			"setup_checked_at":   out.CheckedAt,
		}
		rootCols := make(map[string]any, len(sharedCols)+1)
		for k, v := range sharedCols {
			rootCols[k] = v
		}
		rootCols["setup_evidence"] = string(evidence)

		res := q.Updates(rootCols)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}

		return tx.Model(&model.Signal{}).
			Where("setup_canonical_id = ? AND signal_id <> ?", job.Signal.SignalID, job.Signal.SignalID).
			Updates(sharedCols).Error
	})
}
