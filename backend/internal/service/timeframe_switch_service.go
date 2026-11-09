package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// TimeframeSwitchService mengatur pasangan timeframe mana yang dipantau dan mana
// yang dieksekusi otomatis, per akun dan per pair.
//
// Pasangan timeframe itu sendiri milik aturan trading ILA (pakem engine). Service
// ini hanya menyaring: ia tidak pernah mengarang pasangan yang belum pernah
// terbukti menghasilkan sinyal.
type TimeframeSwitchService interface {
	// Matrix menyusun seluruh saklar timeframe untuk satu pair.
	Matrix(ctx context.Context, userID, symbol string) (*dto.TimeframeSwitchMatrix, error)
	// Set mengubah saklar satu akun untuk satu metode pada satu pair.
	Set(ctx context.Context, userID, accountID, symbol string, req dto.UpdateTimeframeSwitchRequest) (*dto.TimeframeSwitchMatrix, error)
}

// timeframeSignalReader membaca pasangan timeframe yang pernah terlihat.
type timeframeSignalReader interface {
	ListKnownTimeframes(ctx context.Context) (map[string][]repository.TimeframePairRef, error)
}

// timeframeEntitlements membaca hak tier dan tier efektif pengguna.
type timeframeEntitlements interface {
	Strategies(ctx context.Context, userID string) ([]dto.StrategyResponse, error)
}

type timeframeSwitchService struct {
	accounts    switchAccountReader
	strategies  repository.AccountPairStrategyRepository
	timeframes  repository.AccountPairTimeframeRepository
	signals     timeframeSignalReader
	features    timeframeEntitlements
	tierOf      func(ctx context.Context, userID string) (string, error)
	tierFeature TierFeatureReader
}

// NewTimeframeSwitchService merangkai layanan saklar timeframe.
func NewTimeframeSwitchService(
	accounts switchAccountReader,
	strategies repository.AccountPairStrategyRepository,
	timeframes repository.AccountPairTimeframeRepository,
	signals timeframeSignalReader,
	features timeframeEntitlements,
	tierOf func(ctx context.Context, userID string) (string, error),
	tierFeature TierFeatureReader,
) TimeframeSwitchService {
	return &timeframeSwitchService{
		accounts:    accounts,
		strategies:  strategies,
		timeframes:  timeframes,
		signals:     signals,
		features:    features,
		tierOf:      tierOf,
		tierFeature: tierFeature,
	}
}

// Matrix menyusun tampilan saklar timeframe untuk satu pair.
func (s *timeframeSwitchService) Matrix(ctx context.Context, userID, symbol string) (*dto.TimeframeSwitchMatrix, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: simbol wajib diisi", apperror.ErrBadRequest)
	}

	accounts, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	tersedia, err := s.pasanganTersedia(ctx)
	if err != nil {
		return nil, err
	}

	existing, err := s.timeframes.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Saring lewat metode yang dinyalakan akun: saklar timeframe hanya bermakna
	// untuk metode yang benar-benar dijalankan.
	methoden, err := s.metodePerAkun(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Indeks saklar: akun|metode|htf|ltf -> baris.
	indeks := make(map[string]model.AccountPairTimeframe, len(existing))
	for _, row := range existing {
		if !strings.EqualFold(row.Symbol, symbol) {
			continue
		}
		key := row.BrokerAccountID + "|" + row.Strategy + "|" + row.Htf + "|" + row.Ltf
		indeks[key] = row
	}

	autoAllowed := s.autoDiizinkan(ctx, userID)

	out := &dto.TimeframeSwitchMatrix{
		Symbol:      symbol,
		Tersedia:    tersedia,
		AutoAllowed: autoAllowed,
		Groups:      make([]dto.StrategyTimeframeGroup, 0, len(methoden)),
	}

	for _, strategy := range methoden {
		daftar := tersedia[strategy]
		group := dto.StrategyTimeframeGroup{
			Strategy: strategy,
			Akun:     make([]dto.TimeframeSwitchAccount, 0, len(accounts)),
		}
		for _, account := range accounts {
			item := dto.TimeframeSwitchAccount{
				AccountID: account.ID,
				Label:     account.Label,
				Pairs:     make([]dto.TimeframeSwitchItem, 0, len(daftar)),
			}
			for _, pair := range daftar {
				key := account.ID + "|" + strategy + "|" + pair.Htf + "|" + pair.Ltf
				saklar := dto.TimeframeSwitchItem{Htf: pair.Htf, Ltf: pair.Ltf}
				if row, ok := indeks[key]; ok {
					saklar.Watch = row.IsWatch
					saklar.Auto = row.IsAuto
					if !autoAllowed {
						// Tier tanpa auto_execute tidak boleh melihat auto menyala;
						// nilainya mungkin tersisa dari tier sebelumnya.
						saklar.Auto = false
					}
				}
				item.Pairs = append(item.Pairs, saklar)
			}
			group.Akun = append(group.Akun, item)
		}
		out.Groups = append(out.Groups, group)
	}

	return out, nil
}

// Set menulis saklar satu akun untuk satu metode.
func (s *timeframeSwitchService) Set(ctx context.Context, userID, accountID, symbol string, req dto.UpdateTimeframeSwitchRequest) (*dto.TimeframeSwitchMatrix, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: simbol wajib diisi", apperror.ErrBadRequest)
	}
	strategy := strings.TrimSpace(req.Strategy)
	if strategy != "ila-2.0" && strategy != "ila-3.0" {
		return nil, fmt.Errorf("%w: metode %s tidak dikenal", apperror.ErrBadRequest, strategy)
	}

	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil || account == nil || account.UserID != userID {
		return nil, apperror.ErrNotFound
	}

	methoden, err := s.metodePerAkun(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !berisi(methoden, strategy) {
		return nil, fmt.Errorf("%w: metode %s tidak aktif untuk akun ini", apperror.ErrForbidden, strategy)
	}

	autoAllowed := s.autoDiizinkan(ctx, userID)

	rows := make([]repository.TimeframeSwitch, 0, len(req.Pairs))
	for _, pair := range req.Pairs {
		htf := strings.ToUpper(strings.TrimSpace(pair.Htf))
		ltf := strings.ToUpper(strings.TrimSpace(pair.Ltf))
		if htf == "" || ltf == "" {
			return nil, fmt.Errorf("%w: pasangan timeframe tidak lengkap", apperror.ErrBadRequest)
		}

		auto := pair.Auto
		if auto && !autoAllowed {
			return nil, fmt.Errorf("%w: eksekusi otomatis tidak tersedia pada tier Anda", apperror.ErrForbidden)
		}

		// Auto tanpa pantau tidak sah: sinyalnya tidak akan pernah diminta ke
		// engine, sehingga auto tidak akan pernah berjalan. Lebih jujur
		// menolak daripada menerima pengaturan yang mustahil.
		if auto && !pair.Watch {
			return nil, fmt.Errorf("%w: pasangan %s/%s harus dipantau sebelum diizinkan eksekusi otomatis",
				apperror.ErrBadRequest, htf, ltf)
		}

		rows = append(rows, repository.TimeframeSwitch{
			Strategy: strategy,
			Htf:      htf,
			Ltf:      ltf,
			Watch:    pair.Watch,
			Auto:     auto,
		})
	}

	if err := s.timeframes.Replace(ctx, accountID, userID, symbol, strategy, rows); err != nil {
		return nil, err
	}

	return s.Matrix(ctx, userID, symbol)
}

// DefaultTimeframeCatalog menyediakan katalog baku pasangan timeframe untuk setiap strategi.
// Ini memastikan UI saklar timeframe langsung menampilkan daftar pasangan resmi
// tanpa menunggu sinyal pertama masuk dari engine.
var DefaultTimeframeCatalog = map[string][]dto.TimeframePair{
	"ila-2.0": {
		{Htf: "W", Ltf: "H4", Label: "W · H4"},
		{Htf: "D1", Ltf: "H1", Label: "D1 · H1"},
		{Htf: "H4", Ltf: "M15", Label: "H4 · M15"},
		{Htf: "H1", Ltf: "M5", Label: "H1 · M5"},
	},
	"ila-3.0": {
		{Htf: "4H", Ltf: "5M", Label: "4H · 5M"},
		{Htf: "D", Ltf: "15M", Label: "D · 15M"},
		{Htf: "1H", Ltf: "5M", Label: "1H · 5M"},
		{Htf: "W", Ltf: "1H", Label: "W · 1H"},
	},
}

func normPair(htf, ltf string) string {
	clean := func(s string) string {
		s = strings.ToUpper(strings.TrimSpace(s))
		switch s {
		case "H4", "4H":
			return "4H"
		case "H1", "1H":
			return "1H"
		case "M15", "15M":
			return "15M"
		case "M5", "5M":
			return "5M"
		case "D1", "1D", "D":
			return "D"
		case "W1", "1W", "W":
			return "W"
		default:
			return s
		}
	}
	return clean(htf) + "|" + clean(ltf)
}

// pasanganTersedia mengembalikan pasangan timeframe baku yang digabung dengan
// pasangan dari sinyal historis yang pernah masuk.
func (s *timeframeSwitchService) pasanganTersedia(ctx context.Context) (map[string][]dto.TimeframePair, error) {
	out := make(map[string][]dto.TimeframePair, len(DefaultTimeframeCatalog))
	seen := make(map[string]map[string]struct{}, len(DefaultTimeframeCatalog))

	for strat, defaultPairs := range DefaultTimeframeCatalog {
		list := make([]dto.TimeframePair, len(defaultPairs))
		copy(list, defaultPairs)
		out[strat] = list

		seen[strat] = make(map[string]struct{}, len(defaultPairs))
		for _, p := range defaultPairs {
			seen[strat][normPair(p.Htf, p.Ltf)] = struct{}{}
		}
	}

	if s.signals == nil {
		return out, nil
	}

	known, err := s.signals.ListKnownTimeframes(ctx)
	if err != nil {
		return nil, err
	}

	for strategy, pairs := range known {
		if _, ok := seen[strategy]; !ok {
			seen[strategy] = make(map[string]struct{})
		}
		for _, pair := range pairs {
			key := normPair(pair.Htf, pair.Ltf)
			if _, dup := seen[strategy][key]; !dup {
				seen[strategy][key] = struct{}{}
				out[strategy] = append(out[strategy], dto.TimeframePair{
					Htf:   pair.Htf,
					Ltf:   pair.Ltf,
					Label: pair.Htf + " · " + pair.Ltf,
				})
			}
		}
	}

	return out, nil
}

// metodePerAkun mengembalikan metode yang menyala pada minimal satu akun.
func (s *timeframeSwitchService) metodePerAkun(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.strategies.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	menyala := make(map[string]bool, 2)
	for _, row := range rows {
		if row.IsEnabled {
			menyala[row.Strategy] = true
		}
	}

	out := make([]string, 0, 2)
	for _, strategy := range []string{"ila-2.0", "ila-3.0"} {
		if menyala[strategy] {
			out = append(out, strategy)
		}
	}
	return out, nil
}

// autoDiizinkan menandai apakah tier pengguna boleh mengeksekusi otomatis.
//
// Nilainya dibaca dari katalog paket (auto_execute), sumber yang sama dengan
// pagar eksekusi di Runner. Bila tier tidak terbaca, fitur ditutup agar tidak
// terbuka untuk yang tidak berhak.
func (s *timeframeSwitchService) autoDiizinkan(ctx context.Context, userID string) bool {
	if s.tierOf == nil || s.tierFeature == nil {
		return false
	}
	tier, err := s.tierOf(ctx, userID)
	if err != nil || tier == "" {
		return false
	}
	value, err := strconv.ParseBool(strings.TrimSpace(
		s.tierFeature.Feature(ctx, tier, model.FeatureAutoExecute, "false")))
	if err != nil {
		return false
	}
	return value
}

func berisi(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
