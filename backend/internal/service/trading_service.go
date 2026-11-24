package service

import (
	"context"
	"strings"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// FreeTierDelay adalah jeda tampil sinyal untuk tier Free (plans/14 §3.1).
// Memotong jumlah sinyal akan membuat user tidak bisa menilai produk, jadi
// yang dibatasi adalah kecepatan, bukan isinya.
const FreeTierDelay = 15 * time.Minute

// Batas jumlah baris yang dikembalikan daftar dashboard.
const (
	DefaultSignalLimit   = 50
	DefaultPositionLimit = 100

	// DefaultRadarLimit membatasi jumlah baris radar yang dikirim. Radar tidak
	// memakai DefaultSignalLimit karena jumlahnya sebanding dengan jumlah slot
	// yang pernah berjalan, bukan dengan riwayat sinyal.
	DefaultRadarLimit = 200
)

// SignalLister membaca sinyal yang relevan bagi user.
type SignalLister interface {
	ListForUser(ctx context.Context, userID string, since time.Time, limit int) ([]model.Signal, error)
	// ListRadarForUser membaca baris radar terpisah dari riwayat supaya keduanya
	// tidak saling memakan jatah limit.
	ListRadarForUser(ctx context.Context, userID string, batas int) ([]model.Signal, error)
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

// batchUserSignalReader adalah perluasan opsional untuk membaca banyak sinyal
// sekaligus dalam satu query SQL (menghindari N+1 pada daftar sinyal).
type batchUserSignalReader interface {
	FindUserSignals(ctx context.Context, userID string, signalIDs []string) (map[string]model.UserSignal, error)
}

// TradingService menyajikan sinyal dan posisi untuk dashboard.
type TradingService interface {
	Signals(ctx context.Context, userID string, limit int) ([]dto.SignalResponse, error)
	Positions(ctx context.Context, userID string, limit int) ([]dto.PositionResponse, error)
	// ExportPositions menyiapkan riwayat posisi sebagai CSV.
	ExportPositions(ctx context.Context, userID string, from, to time.Time) (*PositionExport, error)
	// Journal menyusun jurnal eksekusi: posisi, ringkasan, dan peluang terlewat.
	Journal(ctx context.Context, userID string, query JournalQuery) (*dto.JournalResponse, error)
}

type tradingService struct {
	users      UserRepository
	signals    SignalLister
	positions  PositionLister
	userSignal UserSignalReader
	clock      func() time.Time
	// journal dan signalDetail disuntikkan lewat WithJournal agar pemanggil
	// lama (dan pengujian yang sudah ada) tetap sah tanpa perubahan.
	journal      JournalRepositoryReader
	signalDetail SignalDetailReader
}

// NewTradingService membuat service tampilan trading.
func NewTradingService(
	users UserRepository,
	signals SignalLister,
	positions PositionLister,
	userSignal UserSignalReader,
) *TradingServiceBuilder {
	return &TradingServiceBuilder{tradingService{
		users:      users,
		signals:    signals,
		positions:  positions,
		userSignal: userSignal,
		clock:      func() time.Time { return time.Now().UTC() },
	}}
}

// TradingServiceBuilder merangkai layanan tampilan trading.
//
// Mengembalikan struct (bukan interface) supaya pemanggil bisa memasang
// tambahan seperti jurnal sebelum layanan dipakai. Tanpa ini, jurnal harus
// menjadi parameter wajib konstruktor dan setiap pemanggil lama ikut berubah.
type TradingServiceBuilder struct {
	tradingService
}

// WithJournal memasang pembaca jurnal beserta pembaca rincian sinyal.
func (b *TradingServiceBuilder) WithJournal(journal JournalRepositoryReader, detail SignalDetailReader) *TradingServiceBuilder {
	b.journal = journal
	b.signalDetail = detail
	return b
}

// Build menyerahkan layanan sebagai antarmuka yang dipakai handler.
func (b *TradingServiceBuilder) Build() TradingService {
	return &b.tradingService
}

// Signals mengembalikan sinyal untuk user, sudah memperhitungkan delay tier.
func (s *tradingService) Signals(ctx context.Context, userID string, limit int) ([]dto.SignalResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = DefaultSignalLimit
	}
	if limit > MaxSignalLimit {
		limit = MaxSignalLimit
	}

	now := s.clock()
	tier := effectiveTier(user, now)

	// Ambil jendela lebih lebar agar sinyal yang masih ditunda tetap terhitung
	// pada retensi riwayat tier tersebut, dan duplikat penerbitan ulang tidak
	// menghabiskan jatah limit setup unik.
	since := now.Add(-signalRetention(tier))
	fetchLimit := limit * 3
	if fetchLimit > 300 {
		fetchLimit = 300
	}

	rows, err := s.signals.ListForUser(ctx, userID, since, fetchLimit)
	if err != nil {
		return nil, err
	}

	// Baris radar diambil terpisah dan ditaruh paling depan. Penggabungan ini
	// menjaga urutan yang sudah dikenal frontend: radar dulu, lalu riwayat
	// terbaru, sambil menyaring bila ada signal_id yang bertumpuk.
	radar, err := s.signals.ListRadarForUser(ctx, userID, DefaultRadarLimit)
	if err != nil {
		return nil, err
	}
	seenSignalID := make(map[string]struct{}, len(radar)+len(rows))
	gabungan := make([]model.Signal, 0, len(radar)+len(rows))
	for i := range radar {
		id := radar[i].SignalID
		if id != "" {
			if _, dup := seenSignalID[id]; dup {
				continue
			}
			seenSignalID[id] = struct{}{}
		}
		gabungan = append(gabungan, radar[i])
	}
	for i := range rows {
		id := rows[i].SignalID
		if id != "" {
			if _, dup := seenSignalID[id]; dup {
				continue
			}
			seenSignalID[id] = struct{}{}
		}
		gabungan = append(gabungan, rows[i])
	}
	rows = gabungan

	var batchSignals map[string]model.UserSignal
	if batcher, ok := s.userSignal.(batchUserSignalReader); ok && len(rows) > 0 {
		ids := make([]string, 0, len(rows))
		for i := range rows {
			if rows[i].SignalID != "" {
				ids = append(ids, rows[i].SignalID)
			}
		}
		if loaded, batchErr := batcher.FindUserSignals(ctx, userID, ids); batchErr == nil {
			batchSignals = loaded
		}
	}

	var posBySignalID map[string]model.Position
	var posByID map[string]model.Position
	if s.positions != nil && len(rows) > 0 {
		if userPositions, posErr := s.positions.ListByUser(ctx, userID, DefaultPositionLimit*2); posErr == nil && len(userPositions) > 0 {
			posBySignalID = make(map[string]model.Position, len(userPositions))
			posByID = make(map[string]model.Position, len(userPositions))
			for i := range userPositions {
				p := userPositions[i]
				if p.ID != "" {
					posByID[p.ID] = p
				}
				if p.SourceSignalID != "" {
					if _, exists := posBySignalID[p.SourceSignalID]; !exists {
						posBySignalID[p.SourceSignalID] = p
					}
				}
			}
		}
	}

	out := make([]dto.SignalResponse, 0, len(rows))
	for i := range rows {
		signal := &rows[i]

		// Sinyal Free baru terlihat setelah jeda; yang TTL-nya habis sebelum
		// jeda selesai tidak ditampilkan sama sekali.
		// Pengecualian: status pemantauan radar (WAIT_*) dikecualikan dari jeda
		// agar dashboard radar tetap bergerak real-time untuk seluruh pengguna.
		delayed := false
		if tier == "free" && !model.IsWaitingSignalStatus(signal.Status) {
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
			Htf:             signal.Htf,
			Ltf:             signal.Ltf,
			FeedSource:      signal.FeedSource,
			Direction:       signal.Direction,
			EntryPrice:      signal.EntryPrice,
			StopLoss:        signal.StopLoss,
			TakeProfit:      signal.TakeProfit,
			RiskReward:      signal.RiskReward,
			Status:          signal.Status,
			Summary:         signal.Summary,
			NextAction:      signal.NextAction,
			PublishedAt:     waktuTerjadi(signal),
			UpdatedAt:       signal.UpdatedAt,
			ExpiresAt:       signal.ExpiresAt,
			Delayed:         delayed,
		}

		view.HtfSweep = parseHtfSweep(signal.HtfSweepJSON)
		view.Cisd = parseCISD(signal.CisdJSON)

		var us *model.UserSignal
		if batchSignals != nil {
			if item, found := batchSignals[signal.SignalID]; found {
				us = &item
			}
		} else if s.userSignal != nil {
			if single, usErr := s.userSignal.FindUserSignal(ctx, userID, signal.SignalID); usErr == nil {
				us = single
			}
		}
		if us != nil {
			view.RejectReason = us.RejectReason
			// Konfirmasi hanya bermakna bila user belum memutuskan apa pun.
			if us.Status == model.SignalStatusReadyToEntry {
				view.Status = model.SignalStatusReadyToEntry
				view.CanConfirm = signal.Status == model.SignalStatusReadyToEntry &&
					now.Before(signal.ExpiresAt)
			} else if us.Status == model.SignalStatusOpportunity && now.Before(signal.ExpiresAt) {
				view.Status = model.SignalStatusOpportunity
			} else if us.Status == model.SignalStatusRejected {
				// Pagar otomatis (kuota harian / posisi terbuka) tetap membuka
				// tombol eksekusi manual selama sinyal belum kedaluwarsa.
				view.CanConfirm = signal.Status == model.SignalStatusReadyToEntry &&
					runner.OverrideKuotaManual(runner.RejectReason(us.RejectReason)) &&
					now.Before(signal.ExpiresAt)
			} else if us.Status == model.SignalStatusConsumed {
				view.Status = model.SignalStatusConsumed
			}
		}

		var matchedPos *model.Position
		if posBySignalID != nil {
			if p, ok := posBySignalID[signal.SignalID]; ok {
				matchedPos = &p
			}
		}
		if matchedPos == nil && us != nil && us.PositionID != "" && posByID != nil {
			if p, ok := posByID[us.PositionID]; ok {
				matchedPos = &p
			}
		}

		if !isRadarSignalRow(signal) {
			applySignalEvaluation(&view, signal, us)
			applySignalPosition(&view, matchedPos)
		}

		out = append(out, view)
	}
	return dedupeSignalViews(out, now, limit), nil
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
			BrokerAccountID: pos.BrokerAccountID,
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
			OpenedAtPair:    dto.NewTimePair(pos.OpenedAt),
			ClosedAt:        pos.ClosedAt,
			ClosedAtPair:    dto.NewTimePairPtr(pos.ClosedAt),
			PnLKnown:        pnlKnown(&pos),
		})
	}
	return out, nil
}

// waktuTerjadi memilih waktu yang mewakili "kapan ini terjadi" untuk sebuah sinyal.
//
// Sinyal biasa diterbitkan sekali, sehingga `published_at` sudah benar. Baris
// radar berbeda: identitasnya tetap per slot, jadi `published_at` membeku sejak
// slot itu lahir sementara isinya terus diperbarui. Memakai `published_at` di
// sana membuat dashboard menampilkan tanggal yang jauh lampau (pernah sampai
// selisih lebih dari tiga hari) padahal radar baru saja bergerak.
func waktuTerjadi(signal *model.Signal) time.Time {
	if strings.HasPrefix(signal.SignalID, "status-") && signal.UpdatedAt.After(signal.PublishedAt) {
		return signal.UpdatedAt
	}
	return signal.PublishedAt
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

// pnlKnown menandai apakah hasil posisi benar-benar terbaca dari broker.
// Posisi yang sudah ditutup tanpa exit price berarti broker tidak melaporkan
// hasilnya; angka 0 pada kondisi itu bukan keuntungan maupun kerugian.
func pnlKnown(pos *model.Position) bool {
	if pos.Status == model.PositionStatusOpen {
		return false
	}
	return pos.ExitPrice != 0 || pos.ProfitLoss != 0
}
