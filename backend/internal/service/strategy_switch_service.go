package service

import (
	"context"
	"fmt"
	"strings"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// StrategySwitchService mengatur saklar metode analisis per akun dan per pair.
//
// Saklar ini yang menentukan metode mana yang benar-benar dieksekusi, sehingga
// daftar pair pantauan tetap bersifat global sementara tiap akun bebas memilih
// metodenya sendiri.
type StrategySwitchService interface {
	// Matrix membuat matriks akun × metode untuk satu pair.
	Matrix(ctx context.Context, userID, symbol string) (*dto.StrategySwitchMatrix, error)
	// Set mengubah saklar satu akun untuk satu pair.
	Set(ctx context.Context, userID, accountID, symbol string, req dto.UpdateStrategySwitchRequest) (*dto.StrategySwitchMatrix, error)
	// EnsureDefaults menyiapkan saklar bawaan saat pair baru dipantau.
	EnsureDefaults(ctx context.Context, userID, symbol string) error
}

// switchAccountReader adalah kebutuhan minimum saklar metode terhadap daftar
// akun. Dependensi dipersempit agar modul ini tidak menyeret seluruh kontrak
// repository hanya untuk membaca akun milik satu user.
type switchAccountReader interface {
	ListByUser(ctx context.Context, userID string) ([]model.BrokerAccount, error)
	FindByID(ctx context.Context, id string) (*model.BrokerAccount, error)
}

// switchEntitlements membaca hak metode pengguna. Dipersempit dari AccountService
// agar layanan ini bisa diuji tanpa merangkai seluruh akun.
type switchEntitlements interface {
	Strategies(ctx context.Context, userID string) ([]dto.StrategyResponse, error)
}

type strategySwitchService struct {
	accounts   switchAccountReader
	strategies repository.AccountPairStrategyRepository
	catalog    PairCatalogService
	features   switchEntitlements
}

// NewStrategySwitchService merangkai layanan saklar metode.
func NewStrategySwitchService(
	accounts switchAccountReader,
	strategies repository.AccountPairStrategyRepository,
	catalog PairCatalogService,
	features switchEntitlements,
) StrategySwitchService {
	return &strategySwitchService{
		accounts:   accounts,
		strategies: strategies,
		catalog:    catalog,
		features:   features,
	}
}

// metodeSah membatasi nama metode yang dikenal engine.
func metodeSah(strategy string) bool {
	return strategy == "ila-2.0" || strategy == "ila-3.0"
}

// Matrix menyusun tampilan akun × metode untuk satu simbol.
func (s *strategySwitchService) Matrix(ctx context.Context, userID, symbol string) (*dto.StrategySwitchMatrix, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: simbol wajib diisi", apperror.ErrBadRequest)
	}

	accounts, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.strategies.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	allowed := s.hakMetode(ctx, userID)

	// Indeks saklar: akun|simbol|metode -> aktif.
	aktif := make(map[string]bool, len(rows))
	for _, row := range rows {
		key := row.BrokerAccountID + "|" + strings.ToUpper(row.Symbol) + "|" + row.Strategy
		aktif[key] = row.IsEnabled
	}

	out := &dto.StrategySwitchMatrix{Symbol: symbol, Akun: make([]dto.StrategySwitchAccount, 0, len(accounts))}
	for _, account := range accounts {
		entry := dto.StrategySwitchAccount{
			AccountID:  account.ID,
			Label:      account.Label,
			BrokerType: account.BrokerType,
			IsActive:   account.IsActive,
			IsDryRun:   account.IsDryRun,
			Strategies: make(map[string]bool, 2),
			Allowed:    allowed,
		}
		for _, strategy := range []string{"ila-2.0", "ila-3.0"} {
			key := account.ID + "|" + symbol + "|" + strategy
			// Bila belum pernah diatur, metode dianggap menyala untuk ILA 2.0.
			// ILA 3.0 mengikuti apakah saklarnya pernah dibuat.
			if value, ok := aktif[key]; ok {
				entry.Strategies[strategy] = value
			} else {
				entry.Strategies[strategy] = strategy == "ila-2.0"
			}
		}
		out.Akun = append(out.Akun, entry)
	}

	return out, nil
}

// hakMetode melaporkan metode mana yang berhak dipakai pengguna. UI memakai peta
// ini untuk menyembunyikan metode yang tier-nya tidak berhak, sehingga ILA 3.0
// tidak bocor ke pengguna Basic.
func (s *strategySwitchService) hakMetode(ctx context.Context, userID string) map[string]bool {
	hasil := map[string]bool{"ila-2.0": true, "ila-3.0": false}
	if s.features == nil {
		return hasil
	}
	list, err := s.features.Strategies(ctx, userID)
	if err != nil {
		// Hak akses tidak terbaca: tutup metode lanjutan, jangan sampai bocor.
		return hasil
	}
	for _, item := range list {
		if !metodeSah(item.Version) {
			continue
		}
		hasil[item.Version] = item.Allowed && item.Available
	}
	return hasil
}

// Set menulis saklar satu akun untuk satu simbol.
func (s *strategySwitchService) Set(ctx context.Context, userID, accountID, symbol string, req dto.UpdateStrategySwitchRequest) (*dto.StrategySwitchMatrix, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return nil, fmt.Errorf("%w: simbol wajib diisi", apperror.ErrBadRequest)
	}

	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, apperror.ErrNotFound
	}

	diminta := map[string]bool{}
	for strategy, enabled := range req.Strategies {
		if !metodeSah(strategy) {
			return nil, fmt.Errorf("%w: metode %s tidak dikenal", apperror.ErrBadRequest, strategy)
		}
		diminta[strategy] = enabled
	}
	if len(diminta) == 0 {
		return nil, fmt.Errorf("%w: tidak ada saklar yang dikirim", apperror.ErrBadRequest)
	}

	// ILA 3.0 tetap dijaga hak akses walau saklarnya dimatikan.
	if _, ada := diminta["ila-3.0"]; ada {
		if err := s.pastikanHakIla3(ctx, userID); err != nil {
			return nil, err
		}
	}

	if err := s.strategies.Replace(ctx, account.ID, userID, symbol, diminta); err != nil {
		return nil, err
	}

	return s.Matrix(ctx, userID, symbol)
}

// pastikanHakIla3 menolak perubahan saklar ILA 3.0 oleh tier yang tidak berhak.
func (s *strategySwitchService) pastikanHakIla3(ctx context.Context, userID string) error {
	if s.features == nil {
		return nil
	}
	list, err := s.features.Strategies(ctx, userID)
	if err != nil {
		return err
	}
	for _, item := range list {
		if item.Version == "ila-3.0" {
			if !item.Allowed {
				return fmt.Errorf("%w: ILA 3.0 tersedia untuk tier Plus", apperror.ErrForbidden)
			}
			if !item.Available {
				return fmt.Errorf("%w: ILA 3.0 belum tersedia", apperror.ErrForbidden)
			}
			return nil
		}
	}
	return fmt.Errorf("%w: ILA 3.0 tidak tersedia", apperror.ErrForbidden)
}

// EnsureDefaults membuat saklar bawaan: ILA 2.0 menyala, ILA 3.0 hanya bila
// pengguna berhak.
func (s *strategySwitchService) EnsureDefaults(ctx context.Context, userID, symbol string) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	accounts, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return err
	}

	ila3Boleh := false
	if s.features != nil {
		if list, listErr := s.features.Strategies(ctx, userID); listErr == nil {
			for _, item := range list {
				if item.Version == "ila-3.0" && item.Allowed && item.Available {
					ila3Boleh = true
				}
			}
		}
	}

	rows := make([]model.AccountPairStrategy, 0, len(accounts)*2)
	for _, account := range accounts {
		rows = append(rows, model.AccountPairStrategy{
			BrokerAccountID: account.ID,
			UserID:          userID,
			Symbol:          symbol,
			Strategy:        "ila-2.0",
			IsEnabled:       true,
		})
		if ila3Boleh {
			rows = append(rows, model.AccountPairStrategy{
				BrokerAccountID: account.ID,
				UserID:          userID,
				Symbol:          symbol,
				Strategy:        "ila-3.0",
				IsEnabled:       true,
			})
		}
	}

	return s.strategies.Ensure(ctx, rows)
}
