package router

import (
	"context"
	"log/slog"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/service"
)

type manualJournalReviewJournal struct {
	*service.ManualJournalService
}

func (j manualJournalReviewJournal) GetAccount(ctx context.Context, user, account string) (*model.ManualJournalAccount, error) {
	v, err := j.ManualJournalService.GetAccount(ctx, user, account)
	if err != nil {
		return nil, err
	}
	return (*model.ManualJournalAccount)(v), nil
}

func runManualJournalReviews(ctx context.Context, poll func(context.Context) error, logger *slog.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		pollCtx, cancel := context.WithTimeout(ctx, time.Minute)
		err := poll(pollCtx)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Warn("manual journal background task failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
