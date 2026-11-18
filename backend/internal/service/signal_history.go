package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

// MaxSignalLimit adalah batas atas aman jumlah setup unik pada satu respons API.
const MaxSignalLimit = 100

func isRadarSignalRow(signal *model.Signal) bool {
	if signal == nil {
		return false
	}
	return strings.HasPrefix(signal.SignalID, "status-") || model.IsWaitingSignalStatus(signal.Status)
}

func isRadarView(view *dto.SignalResponse) bool {
	if view == nil {
		return false
	}
	return strings.HasPrefix(view.SignalID, "status-") || model.IsWaitingSignalStatus(view.Status)
}

func roundCohortPrice(v float64) int64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return int64(math.Round(v * 1e6))
}

func setupCohortBaseKey(view *dto.SignalResponse) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s|%s|%d|%d|%d",
		strings.ToUpper(strings.TrimSpace(view.Symbol)),
		strings.TrimSpace(view.StrategyVersion),
		repository.CanonEvalTimeframe(view.Htf),
		repository.CanonEvalTimeframe(view.Ltf),
		strings.ToUpper(strings.TrimSpace(view.Direction)),
		roundCohortPrice(view.EntryPrice),
		roundCohortPrice(view.StopLoss),
		roundCohortPrice(view.TakeProfit),
	)
}

func feedsCompatible(a, b string) bool {
	fa := strings.ToUpper(strings.TrimSpace(a))
	fb := strings.ToUpper(strings.TrimSpace(b))
	return fa == "" || fb == "" || fa == fb
}

func evalPriority(status string) int {
	switch status {
	case model.EvalTP, model.EvalSL:
		return 4
	case model.EvalRunning:
		return 3
	case model.EvalNoEntry, model.EvalUnknown:
		return 2
	case model.EvalPending:
		return 1
	default:
		return 0
	}
}

func applySignalEvaluation(view *dto.SignalResponse, signal *model.Signal, us *model.UserSignal) {
	if signal != nil && signal.SetupEvalStatus != "" {
		view.EvalStatus = signal.SetupEvalStatus
		view.EvalEntryAt = signal.SetupEntryAt
		view.EvalAt = signal.SetupEvalAt
		view.EvalPrice = signal.SetupEvalPrice
		view.EvalRLower = signal.SetupEvalRLower
		view.EvalNote = signal.SetupEvalNote
		view.EvalEvidence = signal.SetupEvidence
		return
	}
	if us != nil && us.EvalStatus != "" {
		view.EvalStatus = us.EvalStatus
		view.EvalEntryAt = us.EvalEntryAt
		view.EvalAt = us.EvalAt
		view.EvalPrice = us.EvalPrice
		view.EvalRLower = us.EvalRLower
		view.EvalNote = us.EvalNote
		view.EvalEvidence = us.EvalEvidence
	}
}

func applySignalPosition(view *dto.SignalResponse, pos *model.Position) {
	if pos == nil {
		return
	}
	view.IsExecuted = true
	view.PositionID = pos.ID
	view.PositionStatus = pos.Status
	view.OutcomeR = pos.OutcomeR
	view.ProfitLoss = pos.ProfitLoss
	view.PnLKnown = pnlKnown(pos)
	view.ClosedAt = pos.ClosedAt
	if view.Status != model.SignalStatusReadyToEntry && view.Status != model.SignalStatusOpportunity {
		view.Status = model.SignalStatusConsumed
	}
}

func isLiveActionable(view *dto.SignalResponse, now time.Time) bool {
	if view == nil {
		return false
	}
	if view.CanConfirm {
		return true
	}
	return (view.Status == model.SignalStatusReadyToEntry || view.Status == model.SignalStatusOpportunity) &&
		now.Before(view.ExpiresAt)
}

func cohortAlreadyClosedBefore(root *dto.SignalResponse, nextPublished time.Time) bool {
	if root == nil {
		return true
	}
	if (root.EvalStatus == model.EvalTP || root.EvalStatus == model.EvalSL) &&
		root.EvalAt != nil && !root.EvalAt.After(nextPublished) {
		return true
	}
	if root.IsExecuted && root.ClosedAt != nil && !root.ClosedAt.After(nextPublished) {
		return true
	}
	return false
}

// dedupeSignalViews menggabungkan penerbitan ulang setup identik menjadi satu
// kartu kanonik pada Histori Setup Sinyal, sambil menjaga baris radar di depan.
func dedupeSignalViews(items []dto.SignalResponse, now time.Time, limit int) []dto.SignalResponse {
	if len(items) == 0 {
		return items
	}

	radar := make([]dto.SignalResponse, 0)
	history := make([]dto.SignalResponse, 0, len(items))
	for i := range items {
		if isRadarView(&items[i]) {
			radar = append(radar, items[i])
		} else {
			history = append(history, items[i])
		}
	}

	// Urutkan kronologis (terlama -> terbaru) agar akar setup mengikuti
	// penerbitan pertama seperti di antrean evaluasi jurnal.
	sort.SliceStable(history, func(i, j int) bool {
		return history[i].PublishedAt.Before(history[j].PublishedAt)
	})

	deduped := make([]dto.SignalResponse, 0, len(history))
	rootsByBase := make(map[string][]int)

	for i := range history {
		curr := history[i]
		if curr.EntryPrice <= 0 || curr.StopLoss <= 0 || curr.TakeProfit <= 0 {
			deduped = append(deduped, curr)
			continue
		}

		baseKey := setupCohortBaseKey(&curr)
		indices := rootsByBase[baseKey]
		matchedIdx := -1
		for k := len(indices) - 1; k >= 0; k-- {
			candIdx := indices[k]
			cand := &deduped[candIdx]
			if !feedsCompatible(cand.FeedSource, curr.FeedSource) {
				continue
			}
			if curr.PublishedAt.Sub(cand.PublishedAt) > repository.GlobalSignalWindow {
				continue
			}
			if cohortAlreadyClosedBefore(cand, curr.PublishedAt) {
				continue
			}
			matchedIdx = candIdx
			break
		}

		if matchedIdx < 0 {
			newIdx := len(deduped)
			deduped = append(deduped, curr)
			rootsByBase[baseKey] = append(rootsByBase[baseKey], newIdx)
			continue
		}

		mergeSetupCohort(&deduped[matchedIdx], &curr, now)
	}

	// Kembalikan urutan riwayat terbaru lebih dulu untuk antarmuka.
	sort.SliceStable(deduped, func(i, j int) bool {
		return deduped[i].PublishedAt.After(deduped[j].PublishedAt)
	})

	if limit > 0 && len(deduped) > limit {
		deduped = deduped[:limit]
	}

	out := make([]dto.SignalResponse, 0, len(radar)+len(deduped))
	out = append(out, radar...)
	out = append(out, deduped...)
	return out
}

func mergeSetupCohort(root, newer *dto.SignalResponse, now time.Time) {
	if root.FeedSource == "" && newer.FeedSource != "" {
		root.FeedSource = newer.FeedSource
	}
	if newer.UpdatedAt.After(root.UpdatedAt) {
		root.UpdatedAt = newer.UpdatedAt
	}
	if newer.ExpiresAt.After(root.ExpiresAt) {
		root.ExpiresAt = newer.ExpiresAt
	}
	if root.HtfSweep == nil && newer.HtfSweep != nil {
		root.HtfSweep = newer.HtfSweep
	}
	if root.Cisd == nil && newer.Cisd != nil {
		root.Cisd = newer.Cisd
	}

	// Bila terbitan baru masih aktif siap dikonfirmasi/dieksekusi dan belum ada
	// posisi yang berjalan dari kohort ini, pertahankan identitas sinyal aktifnya
	// agar tombol konfirmasi tetap bekerja.
	if isLiveActionable(newer, now) && !root.IsExecuted {
		root.SignalID = newer.SignalID
		root.Status = newer.Status
		root.CanConfirm = newer.CanConfirm
		root.RejectReason = newer.RejectReason
		root.Summary = newer.Summary
		root.NextAction = newer.NextAction
	}

	// Bila terbitan berikutnya yang justru dieksekusi user menjadi posisi broker,
	// wariskan jejak eksekusi posisi tersebut ke kartu setup kanonik.
	if newer.IsExecuted && !root.IsExecuted {
		root.SignalID = newer.SignalID
		root.Status = newer.Status
		root.IsExecuted = true
		root.PositionID = newer.PositionID
		root.PositionStatus = newer.PositionStatus
		root.OutcomeR = newer.OutcomeR
		root.ProfitLoss = newer.ProfitLoss
		root.PnLKnown = newer.PnLKnown
		root.ClosedAt = newer.ClosedAt
	}

	if evalPriority(newer.EvalStatus) > evalPriority(root.EvalStatus) {
		root.EvalStatus = newer.EvalStatus
		root.EvalEntryAt = newer.EvalEntryAt
		root.EvalAt = newer.EvalAt
		root.EvalPrice = newer.EvalPrice
		root.EvalRLower = newer.EvalRLower
		root.EvalNote = newer.EvalNote
		root.EvalEvidence = newer.EvalEvidence
	}
}
