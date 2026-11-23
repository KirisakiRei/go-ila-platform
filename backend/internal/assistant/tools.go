package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/market"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/internal/timezone"
	"ila/backend/pkg/apperror"
)

// ToolDeps mengumpulkan dependensi sempit untuk mengeksekusi tool milik user.
// Didefinisikan sebagai antarmuka di sisi pemakai agar paket assistant bebas
// dari import cycle dengan paket service.
type ToolDeps struct {
	Signals interface {
		Signals(ctx context.Context, userID string, limit int) ([]dto.SignalResponse, error)
	}
	SignalByID interface {
		FindSignal(ctx context.Context, signalID string) (*model.Signal, error)
		FindUserSignal(ctx context.Context, userID, signalID string) (*model.UserSignal, error)
	}
	Journal interface {
		ListPositions(ctx context.Context, userID string, filter repository.JournalFilter, limit int) ([]model.Position, error)
		ListSkipped(ctx context.Context, userID string, filter repository.JournalFilter, offset, limit int) ([]model.UserSignal, error)
		Stats(ctx context.Context, userID string, filter repository.JournalFilter) (repository.JournalStats, error)
	}
	Accounts interface {
		List(ctx context.Context, userID string) ([]dto.AccountResponse, error)
		ListSubscriptions(ctx context.Context, userID string) ([]dto.SubscriptionResponse, error)
		Subscribe(ctx context.Context, userID string, req dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error)
		Unsubscribe(ctx context.Context, userID, subscriptionID string) error
	}
	Pairs interface {
		Catalog(ctx context.Context) (dto.PairCatalogResponse, error)
		ResolveSymbol(ctx context.Context, symbol string) (string, error)
	}
	PairMeta interface {
		FindPairBySymbol(ctx context.Context, symbol string) (*model.TradingPair, error)
	}
	Candles interface {
		Candles(ctx context.Context, symbol, feed, timeframe string, limit int) ([]market.Candle, string, error)
	}
	Quota interface {
		FindDailySummary(ctx context.Context, userID, day string) (*model.DailySummary, error)
	}
	Users interface {
		FindByID(ctx context.Context, id string) (*model.User, error)
	}
	Features interface {
		Feature(ctx context.Context, tier, key, fallback string) string
	}
	MaxDailyTrades int
	Clock          func() time.Time
}

// ToolDefinitions mengembalikan skema 7 tool resmi yang dikenali 9router.
func ToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunctionSchema{
				Name:        "get_signals",
				Description: "Membaca daftar sinyal dan radar aktif milik pengguna saat ini (termasuk detail htf_sweep, cisd, summary, reject_reason, dan status evaluasi).",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"symbol": {"type": "string", "description": "Simbol pair opsional, mis. XAUUSD, EURUSD, USTEC, USDJPY"},
						"htf": {"type": "string", "description": "Filter timeframe besar opsional, mis. H4, D1, H1, M15, W"},
						"signal_id": {"type": "string", "description": "Ambil detail satu sinyal spesifik berdasarkan signal_id"},
						"only_radar": {"type": "boolean", "description": "Jika true, hanya ambil setup yang masih menunggu (WAIT_*)"},
						"limit": {"type": "integer", "description": "Maksimal baris (default 10, maksimal 15)"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunctionSchema{
				Name:        "get_positions_and_journal",
				Description: "Membaca riwayat posisi akun broker (OPEN/CLOSED_TP/CLOSED_SL/CLOSED_MANUAL), statistik ringkas, dan riwayat sinyal tidak tereksekusi (simulasi/forward test).",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"symbol": {"type": "string", "description": "Filter simbol opsional, mis. XAUUSD"},
						"status": {"type": "string", "description": "Filter opsional: OPEN, CLOSED_TP, CLOSED_SL, CLOSED_MANUAL, atau SKIPPED (khusus sinyal tidak tereksekusi)"},
						"limit": {"type": "integer", "description": "Maksimal baris (default 10, maksimal 15)"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunctionSchema{
				Name:        "inspect_trade_candles",
				Description: "Mengambil data candle historis (OHLC + waktu WIB/NY) dari feed resmi pair untuk memverifikasi pergerakan harga sebelum entry atau penyebab menyentuh SL/TP.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"symbol": {"type": "string", "description": "Simbol pair, mis. XAUUSD, EURUSD, USTEC"},
						"timeframe": {"type": "string", "description": "Timeframe candle: M1, M5, M15, H1, H4, D1, W"},
						"from_iso": {"type": "string", "description": "Batas awal waktu ISO-8601/RFC3339 opsional"},
						"to_iso": {"type": "string", "description": "Batas akhir waktu ISO-8601/RFC3339 opsional"},
						"limit": {"type": "integer", "description": "Maksimal jumlah candle dikembalikan (maks 60)"}
					},
					"required": ["symbol", "timeframe"]
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunctionSchema{
				Name:        "get_watchlist_and_quotas",
				Description: "Membaca daftar pair yang sedang dipantau pengguna, jumlah akun broker terhubung, sisa kuota simbol sesuai tier, dan kuota trade harian hari ini.",
				Parameters:  json.RawMessage(`{"type": "object", "properties": {}}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunctionSchema{
				Name:        "get_available_pairs",
				Description: "Membaca daftar pair aktif yang tersedia di katalog platform untuk dipantau.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"search": {"type": "string", "description": "Kata kunci opsional, mis. JPY, XAU, EUR, USD"}
					}
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunctionSchema{
				Name:        "subscribe_pair",
				Description: "Langsung menambahkan pair ke daftar pantau (watchlist) pengguna. Terapkan hanya jika pengguna meminta untuk memantau atau menambahkan pair.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"symbol": {"type": "string", "description": "Simbol pair yang ingin dipantau, mis. USDJPY, XAUUSD"}
					},
					"required": ["symbol"]
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunctionSchema{
				Name:        "unsubscribe_pair",
				Description: "Langsung menghapus pair dari daftar pantau (watchlist) pengguna. Terapkan hanya jika pengguna meminta berhenti memantau atau menghapus pair.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"symbol": {"type": "string", "description": "Simbol pair yang ingin dihapus dari pantauan"}
					},
					"required": ["symbol"]
				}`),
			},
		},
	}
}

// ToolExecResult membawa JSON keluaran tool, ringkasan singkat untuk badge UI,
// serta event perubahan state bila ada aksi tulis (subscribe/unsubscribe).
type ToolExecResult struct {
	OutputJSON string
	OK         bool
	Summary    string
	Action     *ActionAppliedEvent
}

// ToolStartLabel menyusun teks status manusiawi saat tool mulai dijalankan.
func ToolStartLabel(name, rawArgs string) string {
	var args map[string]any
	_ = json.Unmarshal([]byte(rawArgs), &args)
	sym, _ := args["symbol"].(string)
	sym = strings.ToUpper(strings.TrimSpace(sym))

	switch name {
	case "get_signals":
		if sym != "" {
			return fmt.Sprintf("Memeriksa sinyal %s...", sym)
		}
		return "Memeriksa sinyal dan radar..."
	case "get_positions_and_journal":
		if sym != "" {
			return fmt.Sprintf("Membaca riwayat posisi %s...", sym)
		}
		return "Membaca riwayat posisi & jurnal..."
	case "inspect_trade_candles":
		tf, _ := args["timeframe"].(string)
		if sym != "" && tf != "" {
			return fmt.Sprintf("Memeriksa candle %s %s...", sym, strings.ToUpper(tf))
		}
		return "Memeriksa data candle pasar..."
	case "get_watchlist_and_quotas":
		return "Memeriksa watchlist & kuota..."
	case "get_available_pairs":
		return "Memeriksa katalog pair aktif..."
	case "subscribe_pair":
		if sym != "" {
			return fmt.Sprintf("Menambahkan %s ke pantauan...", sym)
		}
		return "Menambahkan pair ke pantauan..."
	case "unsubscribe_pair":
		if sym != "" {
			return fmt.Sprintf("Menghapus %s dari pantauan...", sym)
		}
		return "Menghapus pair dari pantauan..."
	default:
		return "Menjalankan pemeriksaan data..."
	}
}

// ExecuteTool menjalankan satu tool call dengan userID yang selalu dikunci dari
// sesi autentikasi server.
func ExecuteTool(ctx context.Context, deps ToolDeps, userID, name, rawArgs string) ToolExecResult {
	switch name {
	case "get_signals":
		return execGetSignals(ctx, deps, userID, rawArgs)
	case "get_positions_and_journal":
		return execGetPositionsAndJournal(ctx, deps, userID, rawArgs)
	case "inspect_trade_candles":
		return execInspectTradeCandles(ctx, deps, rawArgs)
	case "get_watchlist_and_quotas":
		return execGetWatchlistAndQuotas(ctx, deps, userID)
	case "get_available_pairs":
		return execGetAvailablePairs(ctx, deps, rawArgs)
	case "subscribe_pair":
		return execSubscribePair(ctx, deps, userID, rawArgs)
	case "unsubscribe_pair":
		return execUnsubscribePair(ctx, deps, userID, rawArgs)
	default:
		return toolErr("UNKNOWN_TOOL", fmt.Sprintf("Tool %q tidak dikenal.", name), "Tool tidak tersedia")
	}
}

func resolveSymbolHint(ctx context.Context, deps ToolDeps, raw string) string {
	sym := strings.ToUpper(strings.TrimSpace(raw))
	if sym == "" {
		return ""
	}
	if deps.Pairs != nil {
		if resolved, err := deps.Pairs.ResolveSymbol(ctx, sym); err == nil && resolved != "" {
			return resolved
		}
	}
	switch sym {
	case "XAU", "GOLD":
		return "XAUUSD"
	case "NAS100", "US100", "NDX":
		return "USTEC"
	case "SPX500", "SPX":
		return "US500"
	case "EU":
		return "EURUSD"
	case "GU":
		return "GBPUSD"
	case "UJ":
		return "USDJPY"
	default:
		return sym
	}
}

func execGetSignals(ctx context.Context, deps ToolDeps, userID, rawArgs string) ToolExecResult {
	var in struct {
		Symbol    string `json:"symbol"`
		HTF       string `json:"htf"`
		SignalID  string `json:"signal_id"`
		OnlyRadar bool   `json:"only_radar"`
		Limit     int    `json:"limit"`
	}
	_ = json.Unmarshal([]byte(rawArgs), &in)
	limit := clampLimit(in.Limit, 10, 15)

	if sigID := strings.TrimSpace(in.SignalID); sigID != "" && deps.SignalByID != nil {
		sig, err := deps.SignalByID.FindSignal(ctx, sigID)
		if err != nil {
			return toolErr("SIGNAL_NOT_FOUND", "Sinyal dengan ID tersebut tidak ditemukan.", "Sinyal tidak ditemukan")
		}
		var us *model.UserSignal
		if foundUS, usErr := deps.SignalByID.FindUserSignal(ctx, userID, sigID); usErr == nil {
			us = foundUS
		}
		item := summarizeModelSignal(sig, us)
		return toolOK(map[string]any{"signals": []any{item}, "count": 1}, "1 sinyal ditemukan")
	}

	if deps.Signals == nil {
		return toolErr("UNAVAILABLE", "Layanan sinyal tidak tersedia.", "Gagal baca sinyal")
	}
	rows, err := deps.Signals.Signals(ctx, userID, 100)
	if err != nil {
		return toolErr("QUERY_FAILED", err.Error(), "Gagal baca sinyal")
	}

	targetSym := resolveSymbolHint(ctx, deps, in.Symbol)
	targetHTF := strings.ToUpper(strings.TrimSpace(in.HTF))

	filtered := make([]map[string]any, 0, limit)
	for i := range rows {
		r := rows[i]
		if targetSym != "" && !strings.EqualFold(r.Symbol, targetSym) {
			continue
		}
		if targetHTF != "" && !strings.EqualFold(r.Htf, targetHTF) {
			continue
		}
		if in.OnlyRadar && !model.IsWaitingSignalStatus(r.Status) {
			continue
		}
		filtered = append(filtered, summarizeDTOSignal(&r))
		if len(filtered) >= limit {
			break
		}
	}

	return toolOK(map[string]any{
		"signals": filtered,
		"count":   len(filtered),
	}, fmt.Sprintf("%d sinyal ditemukan", len(filtered)))
}

func summarizeDTOSignal(r *dto.SignalResponse) map[string]any {
	item := map[string]any{
		"signal_id":        r.SignalID,
		"symbol":           r.Symbol,
		"strategy_version": r.StrategyVersion,
		"htf":              r.Htf,
		"ltf":              r.Ltf,
		"feed_source":      r.FeedSource,
		"direction":        r.Direction,
		"status":           r.Status,
		"entry_price":      r.EntryPrice,
		"stop_loss":        r.StopLoss,
		"take_profit":      r.TakeProfit,
		"risk_reward":      r.RiskReward,
		"published_at_wib": formatWIB(r.PublishedAt),
		"published_at_ny":  formatNY(r.PublishedAt),
		"expires_at_wib":   formatWIB(r.ExpiresAt),
		"summary":          r.Summary,
		"next_action":      r.NextAction,
		"is_executed":      r.IsExecuted,
	}
	if r.RejectReason != "" {
		item["reject_reason"] = r.RejectReason
	}
	if r.PositionStatus != "" {
		item["position_status"] = r.PositionStatus
		item["outcome_r"] = r.OutcomeR
	}
	if r.EvalStatus != "" {
		item["simulation_label"] = formatSimulationLabel(r.EvalStatus, r.EvalRLower)
		item["eval_note"] = r.EvalNote
	}
	if r.HtfSweep != nil {
		item["htf_sweep"] = compactSweep(r.HtfSweep)
	}
	if r.Cisd != nil {
		item["cisd"] = compactCISD(r.Cisd)
	}
	return item
}

func summarizeModelSignal(sig *model.Signal, us *model.UserSignal) map[string]any {
	item := map[string]any{
		"signal_id":        sig.SignalID,
		"symbol":           sig.Symbol,
		"strategy_version": sig.StrategyVersion,
		"htf":              sig.Htf,
		"ltf":              sig.Ltf,
		"feed_source":      sig.FeedSource,
		"direction":        sig.Direction,
		"status":           sig.Status,
		"entry_price":      sig.EntryPrice,
		"stop_loss":        sig.StopLoss,
		"take_profit":      sig.TakeProfit,
		"risk_reward":      sig.RiskReward,
		"published_at_wib": formatWIB(sig.PublishedAt),
		"published_at_ny":  formatNY(sig.PublishedAt),
		"expires_at_wib":   formatWIB(sig.ExpiresAt),
		"summary":          sig.Summary,
		"next_action":      sig.NextAction,
	}
	if sig.HtfSweepJSON != "" {
		var sweep dto.SignalHtfSweep
		if json.Unmarshal([]byte(sig.HtfSweepJSON), &sweep) == nil {
			item["htf_sweep"] = compactSweep(&sweep)
		}
	}
	if sig.CisdJSON != "" {
		var cisd dto.SignalCISD
		if json.Unmarshal([]byte(sig.CisdJSON), &cisd) == nil {
			item["cisd"] = compactCISD(&cisd)
		}
	}
	if us != nil {
		item["user_status"] = us.Status
		if us.RejectReason != "" {
			item["reject_reason"] = us.RejectReason
		}
		if us.EvalStatus != "" {
			item["simulation_label"] = formatSimulationLabel(us.EvalStatus, us.EvalRLower)
			item["eval_note"] = us.EvalNote
		}
	}
	return item
}

func compactSweep(s *dto.SignalHtfSweep) map[string]any {
	out := map[string]any{
		"detected":          s.Detected,
		"sweep_in_progress": s.SweepInProgress,
		"direction":         s.Direction,
		"details":           s.Details,
		"stop_loss":         s.StopLoss,
	}
	if s.SweepTime != nil {
		out["sweep_time_wib"] = formatWIB(*s.SweepTime)
		out["sweep_time_ny"] = formatNY(*s.SweepTime)
	}
	if s.FVG != nil {
		out["fvg"] = map[string]any{
			"type":   s.FVG.Type,
			"top":    s.FVG.Top,
			"bottom": s.FVG.Bottom,
		}
	}
	if s.SweepCandle != nil {
		out["sweep_candle"] = compactCandle(s.SweepCandle)
	}
	if s.SweptCandle != nil {
		out["swept_candle"] = compactCandle(s.SweptCandle)
	}
	return out
}

func compactCISD(c *dto.SignalCISD) map[string]any {
	out := map[string]any{
		"detected":    c.Detected,
		"direction":   c.Direction,
		"cisd_level":  c.CISDLevel,
		"extreme_sl":  c.ExtremeSL,
		"entry_price": c.EntryPrice,
		"details":     c.Details,
	}
	if c.BreakTime != nil {
		out["break_time_wib"] = formatWIB(*c.BreakTime)
		out["break_time_ny"] = formatNY(*c.BreakTime)
	}
	if c.RefCandle != nil {
		out["ref_candle"] = compactCandle(c.RefCandle)
	}
	if c.BreakCandle != nil {
		out["break_candle"] = compactCandle(c.BreakCandle)
	}
	return out
}

func compactCandle(c *dto.SignalCandle) map[string]any {
	return map[string]any{
		"time_wib":  formatWIB(c.Timestamp),
		"time_ny":   formatNY(c.Timestamp),
		"open":      c.Open,
		"high":      c.High,
		"low":       c.Low,
		"close":     c.Close,
		"is_closed": c.IsClosed,
	}
}

func execGetPositionsAndJournal(ctx context.Context, deps ToolDeps, userID, rawArgs string) ToolExecResult {
	if deps.Journal == nil {
		return toolErr("UNAVAILABLE", "Repositori jurnal tidak tersedia.", "Jurnal tidak tersedia")
	}
	var in struct {
		Symbol string `json:"symbol"`
		Status string `json:"status"`
		Limit  int    `json:"limit"`
	}
	_ = json.Unmarshal([]byte(rawArgs), &in)
	limit := clampLimit(in.Limit, 10, 15)
	sym := resolveSymbolHint(ctx, deps, in.Symbol)
	status := strings.ToUpper(strings.TrimSpace(in.Status))

	filter := repository.JournalFilter{Symbol: sym}
	stats, _ := deps.Journal.Stats(ctx, userID, filter)

	out := map[string]any{
		"stats": map[string]any{
			"total_positions": stats.TotalPositions,
			"open_positions":  stats.OpenPositions,
			"closed_tp":       stats.ClosedTP,
			"closed_sl":       stats.ClosedSL,
			"closed_manual":   stats.ClosedManual,
			"wins":            stats.Wins,
			"losses":          stats.Losses,
			"total_r":         stats.TotalR,
			"total_profit":    stats.TotalProfit,
		},
	}

	totalCount := 0
	if status != "SKIPPED" {
		posFilter := filter
		if status != "" {
			posFilter.Status = status
		}
		positions, err := deps.Journal.ListPositions(ctx, userID, posFilter, limit)
		if err != nil {
			return toolErr("QUERY_FAILED", err.Error(), "Gagal baca posisi")
		}
		posOut := make([]map[string]any, 0, len(positions))
		for i := range positions {
			p := positions[i]
			item := map[string]any{
				"position_id":      p.ID,
				"source_signal_id": p.SourceSignalID,
				"symbol":           p.Symbol,
				"strategy_version": p.StrategyVersion,
				"direction":        p.Direction,
				"lot_size":         p.LotSize,
				"open_price":       p.OpenPrice,
				"stop_loss":        p.StopLoss,
				"take_profit":      p.TakeProfit,
				"exit_price":       p.ExitPrice,
				"profit_loss":      p.ProfitLoss,
				"outcome_r":        p.OutcomeR,
				"status":           p.Status,
				"is_dry_run":       p.IsDryRun,
				"opened_at_iso":    p.OpenedAt.UTC().Format(time.RFC3339),
				"opened_at_wib":    formatWIB(p.OpenedAt),
			}
			if p.ClosedAt != nil {
				item["closed_at_iso"] = p.ClosedAt.UTC().Format(time.RFC3339)
				item["closed_at_wib"] = formatWIB(*p.ClosedAt)
			}
			posOut = append(posOut, item)
		}
		out["positions"] = posOut
		totalCount += len(posOut)
	}

	if status == "" || status == "SKIPPED" {
		skipped, err := deps.Journal.ListSkipped(ctx, userID, filter, 0, limit)
		if err == nil {
			skipOut := make([]map[string]any, 0, len(skipped))
			for i := range skipped {
				u := skipped[i]
				item := map[string]any{
					"signal_id":        u.SignalID,
					"status_label":     formatUserSignalStatus(u.Status),
					"reject_reason":    u.RejectReason,
					"simulation_label": formatSimulationLabel(u.EvalStatus, u.EvalRLower),
					"note_context":     "Hasil simulasi hanya pantauan harga (bukan PnL akun).",
				}
				if u.EvalStatus != "" {
					item["eval_price"] = u.EvalPrice
					if u.EvalNote != "" {
						item["eval_note"] = u.EvalNote
					}
				}
				if u.PublishedAt != nil {
					item["published_at_wib"] = formatWIB(*u.PublishedAt)
				}
				if u.EvalEntryAt != nil {
					item["eval_entry_at_wib"] = formatWIB(*u.EvalEntryAt)
				}
				if u.EvalAt != nil {
					item["eval_at_wib"] = formatWIB(*u.EvalAt)
				}
				skipOut = append(skipOut, item)
			}
			out["skipped_simulations"] = skipOut
			totalCount += len(skipOut)
		}
	}

	return toolOK(out, fmt.Sprintf("%d catatan jurnal dibaca", totalCount))
}

func execInspectTradeCandles(ctx context.Context, deps ToolDeps, rawArgs string) ToolExecResult {
	if deps.Candles == nil {
		return toolOK(map[string]any{
			"available": false,
			"reason":    "Penyedia candle pasar tidak aktif.",
		}, "Feed candle tidak aktif")
	}

	var in struct {
		Symbol    string `json:"symbol"`
		Timeframe string `json:"timeframe"`
		FromISO   string `json:"from_iso"`
		ToISO     string `json:"to_iso"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &in); err != nil {
		return toolErr("BAD_ARGS", "Format argumen tidak valid.", "Argumen tidak sah")
	}

	sym := resolveSymbolHint(ctx, deps, in.Symbol)
	tf := strings.ToUpper(strings.TrimSpace(in.Timeframe))
	if sym == "" || market.MapResolution(tf) == "" {
		return toolErr("BAD_ARGS", "Simbol atau timeframe tidak valid.", "Parameter tidak sah")
	}

	feed := ""
	if deps.PairMeta != nil {
		if pair, err := deps.PairMeta.FindPairBySymbol(ctx, sym); err == nil && pair != nil {
			sym = pair.Symbol
			feed = pair.FeedSource
		}
	}

	candles, usedFeed, err := deps.Candles.Candles(ctx, sym, feed, tf, 180)
	if err != nil || len(candles) == 0 {
		reason := "Data candle belum tersedia dari feed pasar."
		if err != nil {
			reason = fmt.Sprintf("Gagal mengambil candle: %v", err)
		}
		return toolOK(map[string]any{
			"available": false,
			"symbol":    sym,
			"timeframe": tf,
			"reason":    reason,
		}, "Candle tidak tersedia")
	}

	var fromUnix, toUnix int64
	if t, err := parseFlexibleTime(in.FromISO); err == nil {
		fromUnix = t.Unix()
	}
	if t, err := parseFlexibleTime(in.ToISO); err == nil {
		toUnix = t.Unix()
	}

	filtered := make([]market.Candle, 0, len(candles))
	for _, c := range candles {
		if fromUnix > 0 && c.Time < fromUnix {
			continue
		}
		if toUnix > 0 && c.Time > toUnix {
			continue
		}
		filtered = append(filtered, c)
	}
	if len(filtered) == 0 && (fromUnix > 0 || toUnix > 0) {
		return toolOK(map[string]any{
			"available":     false,
			"symbol":        sym,
			"timeframe":     tf,
			"feed":          usedFeed,
			"oldest_candle": formatWIB(time.Unix(candles[0].Time, 0).UTC()),
			"newest_candle": formatWIB(time.Unix(candles[len(candles)-1].Time, 0).UTC()),
			"reason":        "Rentang waktu yang diminta sudah lebih lama dari jendela 180 candle terakhir yang tersedia di feed.",
		}, "Rentang candle di luar jendela")
	}

	maxOut := clampLimit(in.Limit, 40, 60)
	if len(filtered) > maxOut {
		filtered = filtered[len(filtered)-maxOut:]
	}

	rows := make([]map[string]any, 0, len(filtered))
	for _, c := range filtered {
		ts := time.Unix(c.Time, 0).UTC()
		rows = append(rows, map[string]any{
			"wib":   formatWIB(ts),
			"ny":    formatNY(ts),
			"open":  c.Open,
			"high":  c.High,
			"low":   c.Low,
			"close": c.Close,
		})
	}

	return toolOK(map[string]any{
		"available": true,
		"symbol":    sym,
		"timeframe": tf,
		"feed":      usedFeed,
		"count":     len(rows),
		"candles":   rows,
	}, fmt.Sprintf("%d candle %s %s dimuat", len(rows), sym, tf))
}

func execGetWatchlistAndQuotas(ctx context.Context, deps ToolDeps, userID string) ToolExecResult {
	now := time.Now().UTC()
	if deps.Clock != nil {
		now = deps.Clock()
	}

	var subs []dto.SubscriptionResponse
	if deps.Accounts != nil {
		if list, err := deps.Accounts.ListSubscriptions(ctx, userID); err == nil {
			subs = list
		}
	}
	var accounts []dto.AccountResponse
	if deps.Accounts != nil {
		if list, err := deps.Accounts.List(ctx, userID); err == nil {
			accounts = list
		}
	}

	tier := "free"
	if deps.Users != nil {
		if u, err := deps.Users.FindByID(ctx, userID); err == nil && u != nil {
			switch {
			case u.TrialExpiresAt != nil && now.Before(*u.TrialExpiresAt):
				tier = model.TierTrial
			case u.TierExpiresAt != nil && now.After(*u.TierExpiresAt):
				tier = "free"
			case u.Tier != "":
				tier = u.Tier
			}
		}
	}

	symbolLimit := 1
	switch tier {
	case "basic":
		symbolLimit = 5
	case "plus", "trial":
		symbolLimit = 20
	}
	if deps.Features != nil {
		raw := deps.Features.Feature(ctx, tier, model.FeatureMaxSymbols, strconv.Itoa(symbolLimit))
		if v, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && v > 0 {
			symbolLimit = v
		}
	}

	activeSymbols := make([]string, 0, len(subs))
	seen := make(map[string]bool)
	for _, s := range subs {
		if s.IsActive && !seen[s.Symbol] {
			seen[s.Symbol] = true
			activeSymbols = append(activeSymbols, s.Symbol)
		}
	}

	var daily *model.DailySummary
	if deps.Quota != nil {
		if d, err := deps.Quota.FindDailySummary(ctx, userID, timezone.DayKey(now)); err == nil {
			daily = d
		}
	}

	maxTrades := deps.MaxDailyTrades
	if maxTrades <= 0 {
		maxTrades = 2
	}
	dailyMap := map[string]any{
		"day_wib":      timezone.DayKey(now),
		"total_trades": 0,
		"max_trades":   maxTrades,
		"locked":       false,
	}
	if daily != nil {
		dailyMap["total_trades"] = daily.TotalTrades
		dailyMap["win_count"] = daily.WinCount
		dailyMap["loss_count"] = daily.LossCount
		dailyMap["locked"] = daily.LockedAt != nil
		if daily.LockReason != "" {
			dailyMap["lock_reason"] = daily.LockReason
		}
	}

	accOut := make([]map[string]any, 0, len(accounts))
	for _, a := range accounts {
		accOut = append(accOut, map[string]any{
			"label":       a.Label,
			"broker_type": a.BrokerType,
			"mode":        a.Mode,
			"is_active":   a.IsActive,
			"risk_pct":    a.RiskPercent,
		})
	}

	return toolOK(map[string]any{
		"tier":              tier,
		"watchlist":         activeSymbols,
		"watchlist_count":   len(activeSymbols),
		"watchlist_limit":   symbolLimit,
		"accounts":          accOut,
		"daily_trade_quota": dailyMap,
	}, fmt.Sprintf("%d/%d pair dipantau", len(activeSymbols), symbolLimit))
}

func execGetAvailablePairs(ctx context.Context, deps ToolDeps, rawArgs string) ToolExecResult {
	if deps.Pairs == nil {
		return toolErr("UNAVAILABLE", "Katalog pair tidak tersedia.", "Katalog tidak tersedia")
	}
	var in struct {
		Search string `json:"search"`
	}
	_ = json.Unmarshal([]byte(rawArgs), &in)
	needle := strings.ToUpper(strings.TrimSpace(in.Search))

	cat, err := deps.Pairs.Catalog(ctx)
	if err != nil {
		return toolErr("QUERY_FAILED", err.Error(), "Gagal baca katalog")
	}

	pairs := make([]map[string]string, 0)
	for _, c := range cat.Categories {
		for _, g := range c.Groups {
			for _, p := range g.Pairs {
				if needle != "" && !strings.Contains(strings.ToUpper(p.Symbol+" "+p.DisplayName), needle) {
					continue
				}
				catLabel := c.Label
				if g.Label != "" {
					catLabel = c.Label + " / " + g.Label
				}
				pairs = append(pairs, map[string]string{
					"symbol":       p.Symbol,
					"display_name": p.DisplayName,
					"category":     catLabel,
				})
			}
		}
	}

	return toolOK(map[string]any{
		"pairs": pairs,
		"count": len(pairs),
	}, fmt.Sprintf("%d pair aktif", len(pairs)))
}

func execSubscribePair(ctx context.Context, deps ToolDeps, userID, rawArgs string) ToolExecResult {
	if deps.Accounts == nil {
		return toolErr("UNAVAILABLE", "Layanan langganan tidak tersedia.", "Gagal menambah pair")
	}
	var in struct {
		Symbol string `json:"symbol"`
	}
	_ = json.Unmarshal([]byte(rawArgs), &in)
	sym := resolveSymbolHint(ctx, deps, in.Symbol)
	if sym == "" {
		return toolErr("BAD_ARGS", "Nama simbol wajib diisi.", "Simbol kosong")
	}

	// Cek apakah sudah aktif supaya bisa memberi pesan yang tepat (idempoten).
	existing, err := deps.Accounts.ListSubscriptions(ctx, userID)
	if err == nil {
		for _, s := range existing {
			if s.IsActive && strings.EqualFold(s.Symbol, sym) {
				return toolOK(map[string]any{
					"status":  "already_subscribed",
					"symbol":  s.Symbol,
					"message": fmt.Sprintf("%s sudah ada di daftar pantau Anda.", s.Symbol),
				}, fmt.Sprintf("%s sudah dipantau", s.Symbol))
			}
		}
	}

	sub, err := deps.Accounts.Subscribe(ctx, userID, dto.CreateSubscriptionRequest{Symbol: sym})
	if err != nil {
		switch {
		case errors.Is(err, apperror.ErrForbidden):
			return toolErr("LIMIT_OR_FORBIDDEN", err.Error(), "Kuota pantauan penuh")
		case errors.Is(err, apperror.ErrNotFound), errors.Is(err, apperror.ErrBadRequest):
			return toolErr("INVALID_SYMBOL", err.Error(), "Simbol tidak tersedia")
		default:
			return toolErr("SUBSCRIBE_FAILED", err.Error(), "Gagal menambah pantauan")
		}
	}

	res := toolOK(map[string]any{
		"status":  "subscribed",
		"symbol":  sub.Symbol,
		"message": fmt.Sprintf("%s berhasil ditambahkan ke daftar pantau.", sub.Symbol),
	}, fmt.Sprintf("%s ditambahkan ke pantauan", sub.Symbol))
	res.Action = &ActionAppliedEvent{
		Kind:   "subscription_added",
		Symbol: sub.Symbol,
	}
	return res
}

func execUnsubscribePair(ctx context.Context, deps ToolDeps, userID, rawArgs string) ToolExecResult {
	if deps.Accounts == nil {
		return toolErr("UNAVAILABLE", "Layanan langganan tidak tersedia.", "Gagal menghapus pair")
	}
	var in struct {
		Symbol string `json:"symbol"`
	}
	_ = json.Unmarshal([]byte(rawArgs), &in)
	sym := resolveSymbolHint(ctx, deps, in.Symbol)
	if sym == "" {
		return toolErr("BAD_ARGS", "Nama simbol wajib diisi.", "Simbol kosong")
	}

	existing, err := deps.Accounts.ListSubscriptions(ctx, userID)
	if err != nil {
		return toolErr("QUERY_FAILED", err.Error(), "Gagal memeriksa pantauan")
	}
	var target *dto.SubscriptionResponse
	for i := range existing {
		if existing[i].IsActive && strings.EqualFold(existing[i].Symbol, sym) {
			target = &existing[i]
			break
		}
	}
	if target == nil {
		return toolOK(map[string]any{
			"status":  "not_subscribed",
			"symbol":  sym,
			"message": fmt.Sprintf("%s memang belum ada di daftar pantau Anda.", sym),
		}, fmt.Sprintf("%s belum dipantau", sym))
	}

	if err := deps.Accounts.Unsubscribe(ctx, userID, target.ID); err != nil {
		return toolErr("UNSUBSCRIBE_FAILED", err.Error(), "Gagal menghapus pantauan")
	}

	res := toolOK(map[string]any{
		"status":  "unsubscribed",
		"symbol":  target.Symbol,
		"message": fmt.Sprintf("%s berhasil dihapus dari daftar pantau.", target.Symbol),
	}, fmt.Sprintf("%s dihapus dari pantauan", target.Symbol))
	res.Action = &ActionAppliedEvent{
		Kind:   "subscription_removed",
		Symbol: target.Symbol,
	}
	return res
}

func toolOK(data any, summary string) ToolExecResult {
	b, _ := json.Marshal(data)
	return ToolExecResult{
		OutputJSON: string(b),
		OK:         true,
		Summary:    summary,
	}
}

func toolErr(code, message, summary string) ToolExecResult {
	b, _ := json.Marshal(map[string]any{
		"error":   code,
		"message": message,
	})
	return ToolExecResult{
		OutputJSON: string(b),
		OK:         false,
		Summary:    summary,
	}
}

func clampLimit(v, def, max int) int {
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func formatWIB(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(timezone.WIB).Format("02 Jan 2006 15:04 WIB")
}

func formatNY(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(timezone.NY).Format("02 Jan 2006 15:04 NY")
}

func formatSimulationLabel(status string, rLower float64) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "":
		return "Belum dievaluasi"
	case "TP":
		if rLower != 0 {
			return fmt.Sprintf("Target TP tercapai (simulasi %+.2fR)", rLower)
		}
		return "Target TP tercapai (simulasi)"
	case "SL":
		if rLower != 0 {
			return fmt.Sprintf("Stop Loss tersentuh (simulasi %+.2fR)", rLower)
		}
		return "Stop Loss tersentuh (simulasi -1.00R)"
	case "NO_ENTRY":
		return "Harga tidak menyentuh titik entry"
	case "RUNNING":
		return "Simulasi sedang berjalan"
	case "PENDING":
		return "Menunggu pemantauan harga"
	case "UNKNOWN":
		return "Data candle sudah tidak tersedia"
	default:
		return "Belum dievaluasi"
	}
}

func formatUserSignalStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "REJECTED":
		return "Ditolak aturan akun/risiko"
	case "SKIPPED":
		return "Dilewati"
	case "EXPIRED":
		return "Kedaluwarsa sebelum tersentuh"
	case "FAILED":
		return "Gagal dikirim ke broker"
	case "EXECUTED":
		return "Tereksekusi"
	case "RECEIVED", "PENDING":
		return "Menunggu eksekusi"
	default:
		if status == "" {
			return "Tidak diketahui"
		}
		return status
	}
}

func parseFlexibleTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("empty")
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("waktu tidak dikenali: %q", raw)
}
