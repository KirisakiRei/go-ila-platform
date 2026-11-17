package runner

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ila/backend/internal/evalprice"
	"ila/backend/internal/market"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

// MonitorSignalEvaluations runs on the existing monitor cadence, independently
// of signal expiry and journal reads. Failed fetches leave checkpoints intact.
func (s *Service) MonitorSignalEvaluations(ctx context.Context) {
	if s.opts.Evaluations == nil || s.opts.CandleFetcher == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	now := s.opts.Clock().UTC()
	jobs, err := s.opts.Evaluations.Claim(ctx, now, 20)
	if err != nil {
		s.logger.Warn("claim signal evaluations", "error", err)
		return
	}
	type feedKey struct{ symbol, feed, timeframe string }
	groups := make(map[feedKey][]repository.SignalEvalCandidate)
	var keys []feedKey
	for _, job := range jobs {
		key := feedKey{job.Signal.Symbol, job.Signal.FeedSource, job.Signal.Ltf}
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
		}
		groups[key] = append(groups[key], job)
	}
	// Twenty jobs and five concurrent fetches fit within the 45-second pass
	// even when every distinct feed uses its full ten-second timeout.
	var wg sync.WaitGroup
	slots := make(chan struct{}, 5)
	for _, key := range keys {
		slots <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			seconds := evalCandleSeconds(key.timeframe)
			if seconds == 0 {
				return
			}
			fetchCtx, stop := context.WithTimeout(ctx, 10*time.Second)
			candles, err := s.opts.CandleFetcher(fetchCtx, key.symbol, key.feed, key.timeframe, EvalCandleLimit)
			stop()
			if err != nil || len(candles) == 0 {
				s.logger.Warn("signal evaluation feed unavailable", "symbol", key.symbol, "error", err)
				return
			}
			candles = append([]market.Candle(nil), candles...)
			sort.Slice(candles, func(i, j int) bool { return candles[i].Time < candles[j].Time })
			// A forming candle cannot prove entry order or a terminal outcome.
			closed := candles[:0]
			for _, candle := range candles {
				if candle.Time+seconds <= now.Unix() {
					closed = append(closed, candle)
				}
			}
			if len(closed) == 0 {
				return
			}
			for _, job := range groups[key] {
				in := evalprice.InputDariSinyal(&job.Signal)
				in.ResumeEntryAt = job.UserSignal.EvalEntryAt
				in.ResumeAfter = job.UserSignal.EvalCheckedAt
				in.CandleSeconds = seconds
				out, ok := evalprice.FromCandles(in, closed)
				if !ok {
					continue
				}
				ev := &model.UserSignalEval{Status: out.Status, EntryAt: out.EntryAt, At: out.At, Price: out.Price, RLower: out.RLower, Note: out.Note, CheckedAt: out.CheckedAt, Evidence: out.Evidence}
				if err := s.opts.Evaluations.Save(ctx, job, ev); err != nil {
					s.logger.Warn("save signal evaluation", "signal_id", job.Signal.SignalID, "error", err)
				}
			}
		}()
	}
	wg.Wait()
}

func evalCandleSeconds(tf string) int64 {
	tf = strings.ToUpper(strings.TrimSpace(tf))
	if tf == "D" || tf == "W" {
		tf += "1"
	}
	if len(tf) < 2 {
		return 0
	}
	unit, digits := tf[0], tf[1:]
	if unit >= '0' && unit <= '9' {
		unit, digits = tf[len(tf)-1], tf[:len(tf)-1]
	}
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return 0
		}
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil || n <= 0 || n > 10080 {
		return 0
	}
	switch unit {
	case 'M':
		return n * 60
	case 'H':
		return n * 3600
	case 'D':
		return n * 86400
	case 'W':
		return n * 604800
	}
	return 0
}
