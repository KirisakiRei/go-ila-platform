package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// PairAdminService mengelola katalog pair. Hanya admin yang boleh menyentuhnya,
// karena menonaktifkan pair mengubah risiko seluruh platform.
type PairAdminService interface {
	Catalog(ctx context.Context) (dto.AdminPairCatalogResponse, error)

	CreateCategory(ctx context.Context, actorID string, req dto.CreatePairCategoryRequest) (*dto.AdminPairCategoryView, error)
	UpdateCategory(ctx context.Context, actorID, id string, req dto.UpdatePairCategoryRequest) error
	DeleteCategory(ctx context.Context, actorID, id string) error

	CreateGroup(ctx context.Context, actorID string, req dto.CreatePairGroupRequest) (*dto.AdminPairGroupView, error)
	UpdateGroup(ctx context.Context, actorID, id string, req dto.UpdatePairGroupRequest) error
	DeleteGroup(ctx context.Context, actorID, id string) error

	CreatePair(ctx context.Context, actorID string, req dto.CreatePairRequest) (*dto.AdminPairView, error)
	UpdatePair(ctx context.Context, actorID, id string, req dto.UpdatePairRequest) error
	DeletePair(ctx context.Context, actorID, id string) error
}

type pairAdminService struct {
	pairs repository.PairRepository
	audit AuditRecorder
	clock func() time.Time
}

// NewPairAdminService membuat layanan admin katalog pair.
func NewPairAdminService(pairs repository.PairRepository, audit AuditRecorder) PairAdminService {
	return &pairAdminService{
		pairs: pairs,
		audit: audit,
		clock: func() time.Time { return time.Now().UTC() },
	}
}

// --- Katalog admin ----------------------------------------------------------

func (s *pairAdminService) Catalog(ctx context.Context) (dto.AdminPairCatalogResponse, error) {
	categories, err := s.pairs.ListCategories(ctx, false)
	if err != nil {
		return dto.AdminPairCatalogResponse{}, err
	}
	groups, err := s.pairs.ListGroups(ctx, "", false)
	if err != nil {
		return dto.AdminPairCatalogResponse{}, err
	}
	pairs, err := s.pairs.ListPairs(ctx, repository.PairFilter{})
	if err != nil {
		return dto.AdminPairCatalogResponse{}, err
	}

	categoryByID := make(map[string]model.PairCategory, len(categories))
	groupByID := make(map[string]model.PairGroup, len(groups))
	for _, c := range categories {
		categoryByID[c.ID] = c
	}
	for _, g := range groups {
		groupByID[g.ID] = g
	}

	views := make([]dto.AdminPairCategoryView, 0, len(categories))
	viewIndex := make(map[string]int, len(categories))
	for _, c := range categories {
		views = append(views, dto.AdminPairCategoryView{
			ID: c.ID, Key: c.Key, Label: c.Label, SortOrder: c.SortOrder, IsEnabled: c.IsEnabled,
			Groups: []dto.AdminPairGroupView{},
			Pairs:  []dto.AdminPairView{},
		})
		viewIndex[c.ID] = len(views) - 1
	}

	for _, g := range groups {
		idx, ok := viewIndex[g.CategoryID]
		if !ok {
			continue
		}
		views[idx].Groups = append(views[idx].Groups, dto.AdminPairGroupView{
			ID: g.ID, Key: g.Key, Label: g.Label, SortOrder: g.SortOrder, IsEnabled: g.IsEnabled,
		})
	}

	for _, p := range pairs {
		idx, ok := viewIndex[p.CategoryID]
		if !ok {
			continue
		}
		view := toAdminPairView(p, categoryByID[p.CategoryID], groupByID)
		count, err := s.pairs.CountActiveSubscriptionsBySymbol(ctx, p.Symbol)
		if err == nil {
			view.Subscribers = count
		}
		views[idx].Pairs = append(views[idx].Pairs, view)
	}

	return dto.AdminPairCatalogResponse{Categories: views}, nil
}

func toAdminPairView(p model.TradingPair, category model.PairCategory, groups map[string]model.PairGroup) dto.AdminPairView {
	view := dto.AdminPairView{
		ID:                p.ID,
		Symbol:            p.Symbol,
		DisplayName:       p.DisplayName,
		CategoryID:        p.CategoryID,
		CategoryKey:       category.Key,
		Aliases:           p.Aliases,
		CorrelatedSymbols: p.CorrelatedSymbols,
		SortOrder:         p.SortOrder,
		IsEnabled:         p.IsEnabled,
		Description:       p.Description,
		ValuePerLot:       p.ValuePerLot,
		ContractSize:      p.ContractSize,
		MinLot:            p.MinLot,
		MaxLot:            p.MaxLot,
		LotStep:           p.LotStep,
		FeedSource:        p.FeedSource,
	}
	if p.GroupID != nil {
		view.GroupID = *p.GroupID
		if g, ok := groups[*p.GroupID]; ok {
			view.GroupKey = g.Key
		}
	}
	return view
}

// --- Kategori ---------------------------------------------------------------

func (s *pairAdminService) CreateCategory(ctx context.Context, actorID string, req dto.CreatePairCategoryRequest) (*dto.AdminPairCategoryView, error) {
	key := normalizeKey(req.Key)
	if key == "" {
		return nil, fmt.Errorf("%w: kunci kategori tidak sah", apperror.ErrBadRequest)
	}
	if _, err := s.pairs.FindCategoryByKey(ctx, key); err == nil {
		return nil, fmt.Errorf("%w: kategori %s sudah ada", apperror.ErrConflict, key)
	} else if !errors.Is(err, apperror.ErrNotFound) {
		return nil, err
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}

	category := &model.PairCategory{
		Key:       key,
		Label:     strings.TrimSpace(req.Label),
		SortOrder: req.SortOrder,
		IsEnabled: enabled,
	}
	if err := s.pairs.CreateCategory(ctx, category); err != nil {
		return nil, err
	}
	s.record(ctx, actorID, audit.PairCategoryChange, "create", category.ID, map[string]any{"key": key})
	return &dto.AdminPairCategoryView{
		ID: category.ID, Key: category.Key, Label: category.Label,
		SortOrder: category.SortOrder, IsEnabled: category.IsEnabled,
		Groups: []dto.AdminPairGroupView{}, Pairs: []dto.AdminPairView{},
	}, nil
}

func (s *pairAdminService) UpdateCategory(ctx context.Context, actorID, id string, req dto.UpdatePairCategoryRequest) error {
	category, err := s.pairs.FindCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	if req.Label != nil {
		category.Label = strings.TrimSpace(*req.Label)
	}
	if req.SortOrder != nil {
		category.SortOrder = *req.SortOrder
	}
	if req.IsEnabled != nil {
		// Menonaktifkan kategori menyembunyikan seluruh pair di dalamnya.
		// Posisi terbuka tidak disentuh; hanya sinyal baru yang berhenti.
		category.IsEnabled = *req.IsEnabled
	}
	if err := s.pairs.UpdateCategory(ctx, category); err != nil {
		return err
	}
	s.record(ctx, actorID, audit.PairCategoryChange, "update", category.ID, map[string]any{"enabled": category.IsEnabled})
	return nil
}

func (s *pairAdminService) DeleteCategory(ctx context.Context, actorID, id string) error {
	category, err := s.pairs.FindCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	pairs, err := s.pairs.ListPairs(ctx, repository.PairFilter{CategoryID: id})
	if err != nil {
		return err
	}
	if len(pairs) > 0 {
		return fmt.Errorf("%w: kategori %s masih memiliki %d pair; hapus atau pindahkan dulu", apperror.ErrConflict, category.Key, len(pairs))
	}
	if err := s.pairs.DeleteCategory(ctx, id); err != nil {
		return err
	}
	s.record(ctx, actorID, audit.PairCategoryChange, "delete", id, map[string]any{"key": category.Key})
	return nil
}

// --- Grup -------------------------------------------------------------------

func (s *pairAdminService) CreateGroup(ctx context.Context, actorID string, req dto.CreatePairGroupRequest) (*dto.AdminPairGroupView, error) {
	category, err := s.pairs.FindCategoryByID(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("%w: kategori tidak ditemukan", apperror.ErrBadRequest)
	}
	key := normalizeKey(req.Key)
	if key == "" {
		return nil, fmt.Errorf("%w: kunci grup tidak sah", apperror.ErrBadRequest)
	}

	existing, err := s.pairs.ListGroups(ctx, category.ID, false)
	if err != nil {
		return nil, err
	}
	for _, g := range existing {
		if strings.EqualFold(g.Key, key) {
			return nil, fmt.Errorf("%w: grup %s sudah ada di kategori ini", apperror.ErrConflict, key)
		}
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}
	group := &model.PairGroup{
		CategoryID: category.ID,
		Key:        key,
		Label:      strings.TrimSpace(req.Label),
		SortOrder:  req.SortOrder,
		IsEnabled:  enabled,
	}
	if err := s.pairs.CreateGroup(ctx, group); err != nil {
		return nil, err
	}
	s.record(ctx, actorID, audit.PairGroupChange, "create", group.ID, map[string]any{"key": key, "category": category.Key})
	return &dto.AdminPairGroupView{
		ID: group.ID, Key: group.Key, Label: group.Label,
		SortOrder: group.SortOrder, IsEnabled: group.IsEnabled,
	}, nil
}

func (s *pairAdminService) UpdateGroup(ctx context.Context, actorID, id string, req dto.UpdatePairGroupRequest) error {
	group, err := s.pairs.FindGroupByID(ctx, id)
	if err != nil {
		return err
	}
	if req.Label != nil {
		group.Label = strings.TrimSpace(*req.Label)
	}
	if req.SortOrder != nil {
		group.SortOrder = *req.SortOrder
	}
	if req.IsEnabled != nil {
		group.IsEnabled = *req.IsEnabled
	}
	if err := s.pairs.UpdateGroup(ctx, group); err != nil {
		return err
	}
	s.record(ctx, actorID, audit.PairGroupChange, "update", group.ID, map[string]any{"enabled": group.IsEnabled})
	return nil
}

func (s *pairAdminService) DeleteGroup(ctx context.Context, actorID, id string) error {
	group, err := s.pairs.FindGroupByID(ctx, id)
	if err != nil {
		return err
	}
	pairs, err := s.pairs.ListPairs(ctx, repository.PairFilter{GroupID: id})
	if err != nil {
		return err
	}
	if len(pairs) > 0 {
		return fmt.Errorf("%w: grup %s masih memiliki %d pair; pindahkan dulu", apperror.ErrConflict, group.Key, len(pairs))
	}
	if err := s.pairs.DeleteGroup(ctx, id); err != nil {
		return err
	}
	s.record(ctx, actorID, audit.PairGroupChange, "delete", id, map[string]any{"key": group.Key})
	return nil
}

// --- Pair -------------------------------------------------------------------

func (s *pairAdminService) CreatePair(ctx context.Context, actorID string, req dto.CreatePairRequest) (*dto.AdminPairView, error) {
	symbol := normalizeSymbol(req.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: simbol tidak sah", apperror.ErrBadRequest)
	}
	category, err := s.pairs.FindCategoryByID(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("%w: kategori tidak ditemukan", apperror.ErrBadRequest)
	}
	if !category.IsEnabled {
		return nil, fmt.Errorf("%w: kategori %s sedang nonaktif", apperror.ErrBadRequest, category.Key)
	}

	var groupID *string
	if strings.TrimSpace(req.GroupID) != "" {
		group, err := s.pairs.FindGroupByID(ctx, req.GroupID)
		if err != nil || group.CategoryID != category.ID {
			return nil, fmt.Errorf("%w: grup tidak ditemukan pada kategori ini", apperror.ErrBadRequest)
		}
		gid := group.ID
		groupID = &gid
	}

	if _, err := s.pairs.FindPairBySymbol(ctx, symbol); err == nil {
		return nil, fmt.Errorf("%w: simbol %s sudah terdaftar (termasuk sebagai alias)", apperror.ErrConflict, symbol)
	} else if !errors.Is(err, apperror.ErrNotFound) {
		return nil, err
	}

	correlated, err := s.resolveCorrelations(ctx, symbol, req.CorrelatedSymbols)
	if err != nil {
		return nil, err
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}
	pair := &model.TradingPair{
		Symbol:            symbol,
		DisplayName:       strings.TrimSpace(req.DisplayName),
		CategoryID:        category.ID,
		GroupID:           groupID,
		Aliases:           normalizeAliases(req.Aliases),
		CorrelatedSymbols: correlated,
		SortOrder:         req.SortOrder,
		IsEnabled:         enabled,
		Description:       strings.TrimSpace(req.Description),
		FeedSource:        normalizeFeedSource(req.FeedSource),
	}
	pair.ValuePerLot, pair.ContractSize, pair.MinLot, pair.MaxLot, pair.LotStep = normalizeContract(
		req.ValuePerLot, req.ContractSize, req.MinLot, req.MaxLot, req.LotStep,
	)
	if err := s.pairs.CreatePair(ctx, pair); err != nil {
		return nil, err
	}
	s.record(ctx, actorID, audit.PairChange, "create", pair.ID, map[string]any{"symbol": symbol, "correlated": pair.CorrelatedSymbols})

	view := toAdminPairView(*pair, *category, map[string]model.PairGroup{})
	return &view, nil
}

func (s *pairAdminService) UpdatePair(ctx context.Context, actorID, id string, req dto.UpdatePairRequest) error {
	pair, err := s.pairs.FindPairByID(ctx, id)
	if err != nil {
		return err
	}

	if req.DisplayName != nil {
		pair.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	if req.CategoryID != nil && *req.CategoryID != "" && *req.CategoryID != pair.CategoryID {
		category, err := s.pairs.FindCategoryByID(ctx, *req.CategoryID)
		if err != nil {
			return fmt.Errorf("%w: kategori tidak ditemukan", apperror.ErrBadRequest)
		}
		pair.CategoryID = category.ID
		// Pindah kategori selalu mengosongkan grup agar konsisten.
		pair.GroupID = nil
	}
	if req.GroupID != nil {
		trimmed := strings.TrimSpace(*req.GroupID)
		if trimmed == "" {
			pair.GroupID = nil
		} else {
			group, err := s.pairs.FindGroupByID(ctx, trimmed)
			if err != nil || group.CategoryID != pair.CategoryID {
				return fmt.Errorf("%w: grup tidak ditemukan pada kategori ini", apperror.ErrBadRequest)
			}
			gid := group.ID
			pair.GroupID = &gid
		}
	}
	if req.Aliases != nil {
		pair.Aliases = normalizeAliases(*req.Aliases)
	}
	if req.CorrelatedSymbols != nil {
		correlated, err := s.resolveCorrelations(ctx, pair.Symbol, *req.CorrelatedSymbols)
		if err != nil {
			return err
		}
		pair.CorrelatedSymbols = correlated
	}
	if req.SortOrder != nil {
		pair.SortOrder = *req.SortOrder
	}
	if req.Description != nil {
		pair.Description = strings.TrimSpace(*req.Description)
	}
	if req.IsEnabled != nil {
		// Menonaktifkan pair hanya menghentikan sinyal baru. Posisi yang sudah
		// terbuka tetap dikelola Runner sampai ditutup.
		pair.IsEnabled = *req.IsEnabled
	}

	// Metadata kontrak boleh disunting sebagian: field yang tidak dikirim
	// dipertahankan, lalu dinormalkan bersama aturan konsistensi.
	valuePerLot := pair.ValuePerLot
	contractSize := pair.ContractSize
	minLot := pair.MinLot
	maxLot := pair.MaxLot
	lotStep := pair.LotStep
	if req.ValuePerLot != nil {
		valuePerLot = *req.ValuePerLot
	}
	if req.ContractSize != nil {
		contractSize = *req.ContractSize
	}
	if req.MinLot != nil {
		minLot = *req.MinLot
	}
	if req.MaxLot != nil {
		maxLot = *req.MaxLot
	}
	if req.LotStep != nil {
		lotStep = *req.LotStep
	}
	pair.ValuePerLot, pair.ContractSize, pair.MinLot, pair.MaxLot, pair.LotStep = normalizeContract(
		valuePerLot, contractSize, minLot, maxLot, lotStep,
	)
	if req.FeedSource != nil {
		pair.FeedSource = normalizeFeedSource(*req.FeedSource)
	}

	if err := s.pairs.UpdatePair(ctx, pair); err != nil {
		return err
	}
	s.record(ctx, actorID, audit.PairChange, "update", pair.ID, map[string]any{"enabled": pair.IsEnabled, "correlated": pair.CorrelatedSymbols})
	return nil
}

func (s *pairAdminService) DeletePair(ctx context.Context, actorID, id string) error {
	pair, err := s.pairs.FindPairByID(ctx, id)
	if err != nil {
		return err
	}
	count, err := s.pairs.CountActiveSubscriptionsBySymbol(ctx, pair.Symbol)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: simbol %s masih dipakai %d langganan aktif; nonaktifkan saja", apperror.ErrConflict, pair.Symbol, count)
	}
	if err := s.pairs.DeletePair(ctx, id); err != nil {
		return err
	}
	s.record(ctx, actorID, audit.PairChange, "delete", id, map[string]any{"symbol": pair.Symbol})
	return nil
}

// --- Pembantu ---------------------------------------------------------------

// normalizeKey membersihkan kunci kategori/grup menjadi huruf kecil.
// normalizeContract merapikan metadata kontrak pair.
//
// Nilai tidak masuk akal dikembalikan sebagai nol agar pemanggil memakai
// bawaan kategori. Aturan konsistensi: nilai tidak negatif, step positif,
// dan min <= max.
func normalizeContract(valuePerLot, contractSize, minLot, maxLot, lotStep float64) (float64, float64, float64, float64, float64) {
	if valuePerLot < 0 {
		valuePerLot = 0
	}
	if contractSize < 0 {
		contractSize = 0
	}
	if lotStep <= 0 {
		lotStep = 0.01
	}
	if minLot <= 0 {
		minLot = 0.01
	}
	if maxLot <= 0 {
		maxLot = 10
	}
	if minLot > maxLot {
		minLot, maxLot = maxLot, minLot
	}
	return valuePerLot, contractSize, minLot, maxLot, lotStep
}

// normalizeFeedSource menyeragamkan nama feed broker menjadi huruf besar.
// Nilai kosong berarti simbol memakai pemetaan bawaan engine.
func normalizeFeedSource(raw string) string {
	feed := strings.ToUpper(strings.TrimSpace(raw))
	if len(feed) > 30 {
		feed = feed[:30]
	}
	return feed
}

func normalizeKey(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "-")
	if len(s) < 2 || len(s) > 30 {
		return ""
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return ""
		}
	}
	return s
}

// normalizeAliases merapikan daftar alias dipisah koma.
func normalizeAliases(raw string) string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		alias := strings.ToUpper(strings.TrimSpace(part))
		if alias == "" || seen[alias] {
			continue
		}
		seen[alias] = true
		out = append(out, alias)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// resolveCorrelations merapikan dan memvalidasi daftar simbol pembanding SMT.
//
// Setiap entri dipetakan ke simbol kanonik lewat katalog, sehingga alias yang
// diketik admin (mis. "GOLD") tersimpan sebagai simbol yang dikenal engine.
// Korelasi diri sendiri ditolak karena tidak menghasilkan perbandingan apa pun.
func (s *pairAdminService) resolveCorrelations(ctx context.Context, ownerSymbol, raw string) (string, error) {
	owner := strings.ToUpper(strings.TrimSpace(ownerSymbol))
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))

	for _, part := range parts {
		input := strings.ToUpper(strings.TrimSpace(part))
		if input == "" {
			continue
		}
		if input == owner {
			return "", fmt.Errorf("%w: simbol %s tidak boleh menjadi korelasi dirinya sendiri", apperror.ErrBadRequest, owner)
		}

		found, err := s.pairs.FindPairBySymbol(ctx, input)
		if err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				return "", fmt.Errorf("%w: simbol korelasi %s belum terdaftar di katalog", apperror.ErrBadRequest, input)
			}
			return "", err
		}
		canonical := strings.ToUpper(found.Symbol)
		if canonical == owner {
			return "", fmt.Errorf("%w: simbol %s tidak boleh menjadi korelasi dirinya sendiri", apperror.ErrBadRequest, owner)
		}
		if seen[canonical] {
			continue
		}
		seen[canonical] = true
		out = append(out, canonical)
	}

	sort.Strings(out)
	return strings.Join(out, ","), nil
}

func (s *pairAdminService) record(ctx context.Context, actorID string, action audit.Action, operation, entityID string, meta map[string]any) {
	if s.audit == nil {
		return
	}
	builder := strings.Builder{}
	builder.WriteString("{")
	first := true
	if meta == nil {
		meta = make(map[string]any, 1)
	}
	meta["operation"] = operation
	keys := make([]string, 0, len(meta))
	for k := range meta {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !first {
			builder.WriteString(",")
		}
		first = false
		builder.WriteString(strconv.Quote(k))
		builder.WriteString(":")
		switch v := meta[k].(type) {
		case bool:
			builder.WriteString(strconv.FormatBool(v))
		default:
			builder.WriteString(strconv.Quote(fmt.Sprint(v)))
		}
	}
	builder.WriteString("}")

	entry := &model.AuditLog{
		UserID:     actorID,
		ActorID:    actorID,
		ActorType:  "admin",
		Event:      string(action),
		Entity:     "trading_pair",
		EntityID:   entityID,
		Action:     string(action),
		TargetType: "trading_pair",
		TargetID:   entityID,
		Metadata:   builder.String(),
	}
	_ = s.audit.Record(ctx, entry)
}
