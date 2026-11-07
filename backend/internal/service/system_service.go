package service

import (
	"context"
	"time"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/model"
)

// WIB adalah zona waktu basis aturan harian (plans/05 §3.1).
// Sesi pasar tetap mengikuti UTC/NY; hanya hari akuntansi yang WIB.
var wib = time.FixedZone("WIB", 7*60*60)

// Batas kuota harian yang ditampilkan bila tidak ada konfigurasi lain.
const defaultMaxDailyTrades = 2

// EngineStateReader membaca status koneksi engine yang ditulis Runner.
type EngineStateReader interface {
	ListEngines(ctx context.Context) ([]model.EngineState, error)
}

// DailySummaryReader membaca kuota harian per user.
type DailySummaryReader interface {
	FindDailySummary(ctx context.Context, userID, day string) (*model.DailySummary, error)
}

// SystemService menyajikan status sistem untuk dashboard.
type SystemService interface {
	Status(ctx context.Context, userID string) (*dto.SystemStatus, error)
}

type systemService struct {
	accounts  BrokerAccountRepository
	states    BrokerAccountStateLister
	engines   EngineStateReader
	daily     DailySummaryReader
	positions PositionLister
	maxTrades int
	clock     func() time.Time
}

// SystemServiceDeps mengumpulkan dependensi service status sistem.
type SystemServiceDeps struct {
	Accounts  BrokerAccountRepository
	States    BrokerAccountStateLister
	Engines   EngineStateReader
	Daily     DailySummaryReader
	Positions PositionLister
	MaxTrades int
	Clock     func() time.Time
}

// NewSystemService membuat service status sistem.
func NewSystemService(deps SystemServiceDeps) SystemService {
	clock := deps.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	maxTrades := deps.MaxTrades
	if maxTrades <= 0 {
		maxTrades = defaultMaxDailyTrades
	}
	return &systemService{
		accounts:  deps.Accounts,
		states:    deps.States,
		engines:   deps.Engines,
		daily:     deps.Daily,
		positions: deps.Positions,
		maxTrades: maxTrades,
		clock:     clock,
	}
}

// Status merangkum pasar, engine, broker, dan kuota harian user.
func (s *systemService) Status(ctx context.Context, userID string) (*dto.SystemStatus, error) {
	now := s.clock()

	engines, err := s.engines.ListEngines(ctx)
	if err != nil {
		return nil, err
	}

	status := &dto.SystemStatus{
		ServerTime:    now,
		DailyTimezone: "Asia/Jakarta",
		Market:        dto.MarketStatus{State: "UNKNOWN", Label: "Tidak diketahui"},
		Engines:       make([]dto.EngineStatus, 0, len(engines)),
	}

	for _, engine := range engines {
		status.Engines = append(status.Engines, dto.EngineStatus{
			Strategy:      engine.Strategy,
			Connected:     engine.Connected,
			EngineVersion: engine.EngineVersion,
			LastError:     engine.LastError,
			LastSeenAt:    engine.LastSeenAt,
		})

		// Status pasar dilaporkan engine mana pun yang aktif; ILA 2.0 diprioritaskan.
		if engine.MarketState != "" && (status.Market.State == "UNKNOWN" || engine.Strategy == "ila-2.0") {
			status.Market = dto.MarketStatus{
				State:     engine.MarketState,
				Label:     marketLabel(engine.MarketState),
				UpdatedAt: engine.MarkedAt,
			}
		}
	}

	brokers, err := s.brokerStatus(ctx, userID)
	if err != nil {
		return nil, err
	}
	status.Brokers = brokers

	daily, err := s.dailyStatus(ctx, userID, now)
	if err != nil {
		return nil, err
	}
	status.Daily = daily

	return status, nil
}

// brokerStatus menghitung akun yang terhubung dari telemetry terakhir.
func (s *systemService) brokerStatus(ctx context.Context, userID string) (dto.BrokerStatus, error) {
	accounts, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return dto.BrokerStatus{}, err
	}

	out := dto.BrokerStatus{Total: len(accounts)}
	if len(accounts) == 0 {
		return out, nil
	}

	states, err := s.states.ListByUser(ctx, userID)
	if err != nil {
		return dto.BrokerStatus{}, err
	}

	connected := make(map[string]bool, len(states))
	for _, state := range states {
		connected[state.BrokerAccountID] = state.Connected
		if state.LastError != "" {
			out.AnyError = true
		}
	}

	for _, account := range accounts {
		if connected[account.ID] {
			out.Connected++
		}
	}
	return out, nil
}

// dailyStatus menyusun kuota hari ini dalam WIB.
func (s *systemService) dailyStatus(ctx context.Context, userID string, now time.Time) (dto.DailyResponse, error) {
	day := now.In(wib).Format("2006-01-02")

	summary, err := s.daily.FindDailySummary(ctx, userID, day)
	if err != nil {
		return dto.DailyResponse{}, err
	}

	open, err := s.positions.FindOpenByUser(ctx, userID)
	if err != nil {
		return dto.DailyResponse{}, err
	}

	out := dto.DailyResponse{
		Day:         day,
		MaxTrades:   s.maxTrades,
		OpenCount:   len(open),
		TotalTrades: summary.TotalTrades,
		WinCount:    summary.WinCount,
		LossCount:   summary.LossCount,
	}
	if summary.LockedAt != nil {
		out.Locked = true
		out.LockReason = summary.LockReason
		out.LockedAt = summary.LockedAt
	}
	return out, nil
}

// marketLabel memberi keterangan bahasa manusia atas status pasar.
func marketLabel(state string) string {
	switch state {
	case "OPEN":
		return "Pasar buka"
	case "CLOSED":
		return "Pasar tutup"
	case "ROLLOVER":
		return "Jeda rollover"
	default:
		return "Tidak diketahui"
	}
}
