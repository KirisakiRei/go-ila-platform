package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// PairCatalogService menyajikan katalog pair yang boleh dipilih user.
//
// Sumber kebenaran adalah tabel yang dikelola admin; user tidak pernah
// mengetik simbol sendiri. Ini mencegah typo dan menjaga mesin tetap
// menerima simbol yang benar-benar dikenal.
type PairCatalogService interface {
	// Catalog mengembalikan katalog aktif yang dikelompokkan per kategori/grup.
	Catalog(ctx context.Context) (dto.PairCatalogResponse, error)
	// EnabledSymbols mengembalikan daftar datar simbol aktif.
	EnabledSymbols(ctx context.Context) ([]string, error)
	// ResolveSymbol memetakan simbol atau alias ke simbol kanonik aktif.
	ResolveSymbol(ctx context.Context, symbol string) (string, error)
}

type pairCatalogService struct {
	pairs repository.PairRepository
}

// NewPairCatalogService membuat layanan katalog pair publik.
func NewPairCatalogService(pairs repository.PairRepository) PairCatalogService {
	return &pairCatalogService{pairs: pairs}
}

func (s *pairCatalogService) Catalog(ctx context.Context) (dto.PairCatalogResponse, error) {
	categories, err := s.pairs.ListCategories(ctx, true)
	if err != nil {
		return dto.PairCatalogResponse{}, err
	}
	groups, err := s.pairs.ListGroups(ctx, "", true)
	if err != nil {
		return dto.PairCatalogResponse{}, err
	}
	pairs, err := s.pairs.ListPairs(ctx, repository.PairFilter{OnlyEnabled: true})
	if err != nil {
		return dto.PairCatalogResponse{}, err
	}

	return buildPairCatalog(categories, groups, pairs), nil
}

func (s *pairCatalogService) EnabledSymbols(ctx context.Context) ([]string, error) {
	pairs, err := s.pairs.ListPairs(ctx, repository.PairFilter{OnlyEnabled: true})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, p.Symbol)
	}
	sort.Strings(out)
	return out, nil
}

func (s *pairCatalogService) ResolveSymbol(ctx context.Context, symbol string) (string, error) {
	needle := strings.ToUpper(strings.TrimSpace(symbol))
	if needle == "" {
		return "", fmt.Errorf("%w: simbol kosong", apperror.ErrBadRequest)
	}
	pair, err := s.pairs.FindPairBySymbol(ctx, needle)
	if err != nil {
		return "", fmt.Errorf("%w: simbol %s tidak terdaftar di katalog", apperror.ErrBadRequest, needle)
	}
	if !pair.IsEnabled {
		return "", fmt.Errorf("%w: simbol %s sedang dinonaktifkan admin", apperror.ErrBadRequest, pair.Symbol)
	}
	return pair.Symbol, nil
}

// buildPairCatalog menyusun struktur kategori → grup → pair.
//
// Pair yang belum punya sub-kategori diletakkan pada grup sintetis "umum" agar
// tidak hilang dari tampilan.
func buildPairCatalog(categories []model.PairCategory, groups []model.PairGroup, pairs []model.TradingPair) dto.PairCatalogResponse {
	enabledCategory := make(map[string]bool, len(categories))
	order := make([]string, 0, len(categories))
	byID := make(map[string]model.PairCategory, len(categories))
	for _, c := range categories {
		enabledCategory[c.ID] = c.IsEnabled
		byID[c.ID] = c
		order = append(order, c.ID)
	}

	groupEnabled := make(map[string]bool, len(groups))
	groupOrder := make(map[string][]model.PairGroup, len(categories))
	for _, g := range groups {
		groupEnabled[g.ID] = g.IsEnabled
		groupOrder[g.CategoryID] = append(groupOrder[g.CategoryID], g)
	}

	type bucket struct {
		groups map[string][]dto.PairItem
		order  []string
	}
	byCategory := make(map[string]*bucket, len(categories))

	// Simbol yang benar-benar tampil untuk user. Dipakai menyaring daftar
	// korelasi agar kartu pair tidak menyebut instrumen yang sedang
	// dinonaktifkan admin (instrumen itu juga tidak diminta ke engine).
	visible := make(map[string]bool, len(pairs))
	for _, p := range pairs {
		if !p.IsEnabled || !enabledCategory[p.CategoryID] {
			continue
		}
		if p.GroupID != nil && !groupEnabled[*p.GroupID] {
			continue
		}
		visible[p.Symbol] = true
	}

	for _, p := range pairs {
		if !p.IsEnabled || !enabledCategory[p.CategoryID] {
			continue
		}
		groupKey := "_ungrouped"
		if p.GroupID != nil {
			if !groupEnabled[*p.GroupID] {
				continue
			}
			groupKey = *p.GroupID
		}
		b, ok := byCategory[p.CategoryID]
		if !ok {
			b = &bucket{groups: make(map[string][]dto.PairItem)}
			byCategory[p.CategoryID] = b
		}
		if _, seen := b.groups[groupKey]; !seen {
			b.order = append(b.order, groupKey)
		}
		b.groups[groupKey] = append(b.groups[groupKey], dto.PairItem{
			Symbol:            p.Symbol,
			DisplayName:       p.DisplayName,
			Description:       p.Description,
			Aliases:           p.Aliases,
			CorrelatedSymbols: filterVisible(splitCSV(p.CorrelatedSymbols), visible),
		})
	}

	out := dto.PairCatalogResponse{Categories: make([]dto.PairCategoryView, 0, len(categories))}
	symbols := make([]string, 0, len(pairs))

	for _, catID := range order {
		b := byCategory[catID]
		if b == nil {
			continue
		}

		catView := dto.PairCategoryView{
			Key:    byID[catID].Key,
			Label:  byID[catID].Label,
			Groups: make([]dto.PairGroupView, 0, len(b.order)),
		}

		// Grup sesuai urutan admin, lalu grup sintetis "_ungrouped" di akhir.
		for _, g := range groupOrder[catID] {
			items, ok := b.groups[g.ID]
			if !ok {
				continue
			}
			catView.Groups = append(catView.Groups, dto.PairGroupView{Key: g.Key, Label: g.Label, Pairs: items})
		}
		if items, ok := b.groups["_ungrouped"]; ok {
			catView.Groups = append(catView.Groups, dto.PairGroupView{Key: "umum", Label: "Umum", Pairs: items})
		}

		for _, grp := range catView.Groups {
			for _, p := range grp.Pairs {
				symbols = append(symbols, p.Symbol)
			}
		}
		out.Categories = append(out.Categories, catView)
	}

	sort.Strings(symbols)
	out.Symbols = symbols
	return out
}

// filterVisible menyisakan simbol yang masih tampil di katalog.
func filterVisible(symbols []string, visible map[string]bool) []string {
	if len(symbols) == 0 {
		return nil
	}
	out := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		if visible[symbol] {
			out = append(out, symbol)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// splitCSV memecah daftar dipisah koma menjadi irisan bersih tanpa entri kosong.
func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.ToUpper(strings.TrimSpace(part))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
