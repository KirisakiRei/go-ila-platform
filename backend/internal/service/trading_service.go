package service

import (
	"context"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
)

// FreeTierDelay adalah jeda tampil sinyal untuk tier Free (plans/14 §3.1).
// Memotong jumlah sinyal akan membuat user tidak bisa menilai produk, jadi
// yang dibatasi adalah kecepatan, bukan isinya.
const FreeTierDelay = 15 * time.Minute

// Batas jumlah baris yang dikembalikan daftar dashboard.
const (
	DefaultSignalLimit   = 50
	DefaultPositionLimit = 100
)

// SignalLister membaca sinyal yang relevan bagi user.
type SignalLister interface {
	ListForUser(ctx context.Context, userID string, since time.Time, limit int) ([]model.Signal, error)
}

// PositionLister membaca posisi user.
type PositionLister interface {
	ListByUser(ctx context.Context, userID string, limit int) ([]model.Position, error)
	FindOpenByUser(ctx context.Context, userID string) ([]model.Position, error)
}

// UserSignalReader membaca status sinyal per user.
type UserSignalReader interface {
	FindUserSignal(ctx context.Context, userID, signalID string) (*model.UserSignal, error)
}

// TradingService menyajikan sinyal dan posisi untuk dashboard.
type TradingService interface {
	Signals(ctx context.Context, userID string, limit int) ([]dto.SignalResponse, error)
	Positions(ctx context.Context, userID string, limit int) ([]dto.PositionResponse, error)
}

type tradingService struct {
	users      UserRepository
	signals    SignalLister
	positions  PositionLister
	userSignal UserSignalReader
	clock      func() time.Time
}

// NewTradingService membuat service tampilan trading.
func NewTradingService(
	users UserRepository,
	signals SignalLister,
	positions PositionLister,
	userSignal UserSignalReader,
) TradingService {
	return &tradingService{
		users:      users,
		signals:    signals,
		positions:  positions,
		userSignal: userSignal,
		clock:      func() time.Time { return time.Now().UTC() },
	}
}

// Signals mengembalikan sinyal untuk user, sudah memperhitungkan delay tier.
func (s *tradingService) Signals(ctx context.Context, userID string, limit int) ([]dto.SignalResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit > DefaultSignalLimit {
		limit = DefaultSignalLimit
	}

	now := s.clock()
	tier := effectiveTier(user, now)

	// Ambil jendela lebih lebar agar sinyal yang masih ditunda tetap terhitung
	// pada retensi riwayat tier tersebut.
	since := now.Add(-signalRetention(tier))

	rows, err := s.signals.ListForUser(ctx, userID, since, limit)
	if err != nil {
		return nil, err
	}

	out := make([]dto.SignalResponse, 0, len(rows))
	for i := range rows {
		signal := &rows[i]

		// Sinyal Free baru terlihat setelah jeda; yang TTL-nya habis sebelum
		// jeda selesai tidak ditampilkan sama sekali.
		delayed := false
		if tier == "free" {
			visibleAt := signal.PublishedAt.Add(FreeTierDelay)
			if now.Before(visibleAt) {
				continue
			}
			if !now.Before(signal.ExpiresAt) {
				continue
			}
			delayed = true
		}

		view := dto.SignalResponse{
			SignalID:        signal.SignalID,
			Symbol:          signal.Symbol,
			StrategyVersion: signal.StrategyVersion,
			EngineVersion:   signal.EngineVersion,
			Direction:       signal.Direction,
			EntryPrice:      signal.EntryPrice,
			StopLoss:        signal.StopLoss,
			TakeProfit:      signal.TakeProfit,
			RiskReward:      signal.RiskReward,
			Status:          signal.Status,
			Summary:         signal.Summary,
			NextAction:      signal.NextAction,
			PublishedAt:     signal.PublishedAt,
			ExpiresAt:       signal.ExpiresAt,
			Delayed:         delayed,
		}

		if us, usErr := s.userSignal.FindUserSignal(ctx, userID, signal.SignalID); usErr == nil && us != nil {
			view.RejectReason = us.RejectReason
			// Konfirmasi hanya bermakna bila user belum memutuskan apa pun.
			if us.Status == model.SignalStatusReadyToEntry {
				view.Status = model.SignalStatusReadyToEntry
				view.CanConfirm = signal.Status == model.SignalStatusReadyToEntry &&
					now.Before(signal.ExpiresAt)
			}
		}

		out = append(out, view)
	}
	return out, nil
}

// Positions mengembalikan posisi user, terbaru lebih dulu.
func (s *tradingService) Positions(ctx context.Context, userID string, limit int) ([]dto.PositionResponse, error) {
	if limit <= 0 || limit > DefaultPositionLimit {
		limit = DefaultPositionLimit
	}

	rows, err := s.positions.ListByUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	out := make([]dto.PositionResponse, 0, len(rows))
	for _, pos := range rows {
		out = append(out, dto.PositionResponse{
			ID:              pos.ID,
			Symbol:          pos.Symbol,
			StrategyVersion: pos.StrategyVersion,
			Direction:       pos.Direction,
			TicketNumber:    pos.TicketNumber,
			LotSize:         pos.LotSize,
			OpenPrice:       pos.OpenPrice,
			StopLoss:        pos.StopLoss,
			TakeProfit:      pos.TakeProfit,
			ExitPrice:       pos.ExitPrice,
			ProfitLoss:      pos.ProfitLoss,
			OutcomeR:        pos.OutcomeR,
			RiskPercent:     pos.RiskPercent,
			Status:          pos.Status,
			IsDryRun:        pos.IsDryRun,
			OpenedAt:        pos.OpenedAt,
			ClosedAt:        pos.ClosedAt,
		})
	}
	return out, nil
}

// signalRetention mengembalikan rentang riwayat sinyal per tier.
func signalRetention(tier string) time.Duration {
	switch tier {
	case "plus":
		return 365 * 24 * time.Hour
	case "basic":
		return 90 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}
