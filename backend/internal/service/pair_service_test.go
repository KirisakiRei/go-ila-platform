package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"ila/backend/internal/audit"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// --- Tiruan katalog pair ----------------------------------------------------

type fakePairRepo struct {
	seq        int
	categories []*model.PairCategory
	groups     []*model.PairGroup
	pairs      []*model.TradingPair
	// subscribers memetakan simbol ke jumlah langganan aktif.
	subscribers map[string]int64
}

func newFakePairRepo() *fakePairRepo {
	return &fakePairRepo{subscribers: map[string]int64{}}
}

func (f *fakePairRepo) nextID(prefix string) string {
	f.seq++
	return fmt.Sprintf("%s-%d", prefix, f.seq)
}

func (f *fakePairRepo) CreateCategory(_ context.Context, c *model.PairCategory) error {
	c.ID = f.nextID("cat")
	f.categories = append(f.categories, c)
	return nil
}

func (f *fakePairRepo) FindCategoryByID(_ context.Context, id string) (*model.PairCategory, error) {
	for _, c := range f.categories {
		if c.ID == id {
			copy := *c
			return &copy, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakePairRepo) FindCategoryByKey(_ context.Context, key string) (*model.PairCategory, error) {
	for _, c := range f.categories {
		if strings.EqualFold(c.Key, key) {
			copy := *c
			return &copy, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakePairRepo) ListCategories(_ context.Context, onlyEnabled bool) ([]model.PairCategory, error) {
	var out []model.PairCategory
	for _, c := range f.categories {
		if onlyEnabled && !c.IsEnabled {
			continue
		}
		out = append(out, *c)
	}
	return out, nil
}

func (f *fakePairRepo) UpdateCategory(_ context.Context, c *model.PairCategory) error {
	for i, existing := range f.categories {
		if existing.ID == c.ID {
			copy := *c
			f.categories[i] = &copy
		}
	}
	return nil
}

func (f *fakePairRepo) DeleteCategory(_ context.Context, id string) error {
	for i, c := range f.categories {
		if c.ID == id {
			f.categories = append(f.categories[:i], f.categories[i+1:]...)
			return nil
		}
	}
	return apperror.ErrNotFound
}

func (f *fakePairRepo) CreateGroup(_ context.Context, g *model.PairGroup) error {
	g.ID = f.nextID("grp")
	f.groups = append(f.groups, g)
	return nil
}

func (f *fakePairRepo) FindGroupByID(_ context.Context, id string) (*model.PairGroup, error) {
	for _, g := range f.groups {
		if g.ID == id {
			copy := *g
			return &copy, nil
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakePairRepo) ListGroups(_ context.Context, categoryID string, onlyEnabled bool) ([]model.PairGroup, error) {
	var out []model.PairGroup
	for _, g := range f.groups {
		if categoryID != "" && g.CategoryID != categoryID {
			continue
		}
		if onlyEnabled && !g.IsEnabled {
			continue
		}
		out = append(out, *g)
	}
	return out, nil
}

func (f *fakePairRepo) UpdateGroup(_ context.Context, g *model.PairGroup) error {
	for i, existing := range f.groups {
		if existing.ID == g.ID {
			copy := *g
			f.groups[i] = &copy
		}
	}
	return nil
}

func (f *fakePairRepo) DeleteGroup(_ context.Context, id string) error {
	for i, g := range f.groups {
		if g.ID == id {
			f.groups = append(f.groups[:i], f.groups[i+1:]...)
			return nil
		}
	}
	return apperror.ErrNotFound
}

func (f *fakePairRepo) CreatePair(_ context.Context, p *model.TradingPair) error {
	p.ID = f.nextID("pair")
	f.pairs = append(f.pairs, p)
	return nil
}

func (f *fakePairRepo) FindPairByID(_ context.Context, id string) (*model.TradingPair, error) {
	for _, p := range f.pairs {
		if p.ID == id {
			copy := *p
			return &copy, nil
		}
	}
	return nil, apperror.ErrNotFound
}

// FindPairBySymbol meniru repository asli: cocok pada simbol maupun alias.
func (f *fakePairRepo) FindPairBySymbol(_ context.Context, symbol string) (*model.TradingPair, error) {
	needle := strings.ToUpper(strings.TrimSpace(symbol))
	for _, p := range f.pairs {
		if strings.EqualFold(p.Symbol, needle) {
			copy := *p
			return &copy, nil
		}
		for _, alias := range strings.Split(p.Aliases, ",") {
			if strings.EqualFold(strings.TrimSpace(alias), needle) {
				copy := *p
				return &copy, nil
			}
		}
	}
	return nil, apperror.ErrNotFound
}

func (f *fakePairRepo) ListPairs(_ context.Context, filter repository.PairFilter) ([]model.TradingPair, error) {
	var out []model.TradingPair
	for _, p := range f.pairs {
		if filter.CategoryID != "" && p.CategoryID != filter.CategoryID {
			continue
		}
		if filter.GroupID != "" && (p.GroupID == nil || *p.GroupID != filter.GroupID) {
			continue
		}
		if filter.OnlyEnabled && !p.IsEnabled {
			continue
		}
		out = append(out, *p)
	}
	return out, nil
}

func (f *fakePairRepo) UpdatePair(_ context.Context, p *model.TradingPair) error {
	for i, existing := range f.pairs {
		if existing.ID == p.ID {
			copy := *p
			f.pairs[i] = &copy
		}
	}
	return nil
}

func (f *fakePairRepo) DeletePair(_ context.Context, id string) error {
	for i, p := range f.pairs {
		if p.ID == id {
			f.pairs = append(f.pairs[:i], f.pairs[i+1:]...)
			return nil
		}
	}
	return apperror.ErrNotFound
}

func (f *fakePairRepo) CountActiveSubscriptionsBySymbol(_ context.Context, symbol string) (int64, error) {
	return f.subscribers[symbol], nil
}

// pairAuditRecorder menangkap baris audit yang ditulis layanan admin pair.
type pairAuditRecorder struct {
	entries []*model.AuditLog
}

func (r *pairAuditRecorder) Record(_ context.Context, entry *model.AuditLog) error {
	r.entries = append(r.entries, entry)
	return nil
}

// seedPairFixture membuat kategori forex (dengan grup usd) dan indices.
func seedPairFixture(t *testing.T, repo *fakePairRepo) (forex, indices *model.PairCategory, usd *model.PairGroup) {
	t.Helper()
	ctx := context.Background()

	forex = &model.PairCategory{Key: "forex", Label: "Forex", IsEnabled: true}
	indices = &model.PairCategory{Key: "indices", Label: "Indices", IsEnabled: true}
	for _, c := range []*model.PairCategory{forex, indices} {
		if err := repo.CreateCategory(ctx, c); err != nil {
			t.Fatalf("seed kategori: %v", err)
		}
	}

	usd = &model.PairGroup{CategoryID: forex.ID, Key: "usd", Label: "USD Quote", IsEnabled: true}
	if err := repo.CreateGroup(ctx, usd); err != nil {
		t.Fatalf("seed grup: %v", err)
	}

	gid := usd.ID
	for _, p := range []*model.TradingPair{
		{Symbol: "EURUSD", CategoryID: forex.ID, GroupID: &gid, IsEnabled: true},
		{Symbol: "GBPUSD", CategoryID: forex.ID, GroupID: &gid, IsEnabled: true},
		{Symbol: "USTEC", CategoryID: indices.ID, Aliases: "NAS100,US100", IsEnabled: true},
		// Tanpa grup: harus tampil di grup sintetis "umum".
		{Symbol: "XAUUSD", CategoryID: forex.ID, Aliases: "GOLD", IsEnabled: true},
	} {
		if err := repo.CreatePair(ctx, p); err != nil {
			t.Fatalf("seed pair: %v", err)
		}
	}
	return forex, indices, usd
}

// --- Admin: kategori/grup/pair ---------------------------------------------

func TestPairAdmin_KunciKategoriHarusUnik(t *testing.T) {
	repo := newFakePairRepo()
	svc := NewPairAdminService(repo, &pairAuditRecorder{})
	ctx := context.Background()

	if _, err := svc.CreateCategory(ctx, "admin-1", dto.CreatePairCategoryRequest{Key: "Forex", Label: "Forex"}); err != nil {
		t.Fatalf("kategori pertama harus berhasil: %v", err)
	}
	_, err := svc.CreateCategory(ctx, "admin-1", dto.CreatePairCategoryRequest{Key: "forex", Label: "Forex Lagi"})
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("kunci kembar harus conflict, dapat %v", err)
	}
}

func TestPairAdmin_SimbolKembarTermasukAliasDitolak(t *testing.T) {
	repo := newFakePairRepo()
	forex, indices, _ := seedPairFixture(t, repo)
	svc := NewPairAdminService(repo, &pairAuditRecorder{})
	ctx := context.Background()

	// "NAS100" adalah alias USTEC; tidak boleh jadi pair sendiri.
	_, err := svc.CreatePair(ctx, "admin-1", dto.CreatePairRequest{Symbol: "nas100", CategoryID: indices.ID})
	if !errors.Is(err, apperror.ErrConflict) {
		t.Fatalf("alias kembar harus conflict, dapat %v", err)
	}

	view, err := svc.CreatePair(ctx, "admin-1", dto.CreatePairRequest{
		Symbol: " usdjpy ", CategoryID: forex.ID, Aliases: "jpy, JPY,,usd/jpy",
	})
	if err != nil {
		t.Fatalf("pair baru harus berhasil: %v", err)
	}
	if view.Symbol != "USDJPY" {
		t.Errorf("simbol dinormalisasi jadi %q, ingin USDJPY", view.Symbol)
	}
	if view.Aliases != "JPY,USD/JPY" {
		t.Errorf("alias = %q, ingin JPY,USD/JPY (unik, huruf besar, terurut)", view.Aliases)
	}
}

func TestPairAdmin_HapusDiblokirBilaMasihDipakai(t *testing.T) {
	repo := newFakePairRepo()
	forex, _, usd := seedPairFixture(t, repo)
	svc := NewPairAdminService(repo, &pairAuditRecorder{})
	ctx := context.Background()

	// Kategori masih punya pair.
	if err := svc.DeleteCategory(ctx, "admin-1", forex.ID); !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("hapus kategori berisi pair harus conflict, dapat %v", err)
	}
	// Grup masih punya pair.
	if err := svc.DeleteGroup(ctx, "admin-1", usd.ID); !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("hapus grup berisi pair harus conflict, dapat %v", err)
	}

	// Pair dengan pelanggan aktif tidak boleh dihapus.
	repo.subscribers["EURUSD"] = 3
	eur, err := repo.FindPairBySymbol(ctx, "EURUSD")
	if err != nil {
		t.Fatalf("cari EURUSD: %v", err)
	}
	if err := svc.DeletePair(ctx, "admin-1", eur.ID); !errors.Is(err, apperror.ErrConflict) {
		t.Errorf("hapus pair berlangganan harus conflict, dapat %v", err)
	}

	// Setelah pelanggan habis, penghapusan berjalan.
	repo.subscribers["EURUSD"] = 0
	if err := svc.DeletePair(ctx, "admin-1", eur.ID); err != nil {
		t.Errorf("hapus pair tanpa pelanggan harus berhasil: %v", err)
	}
}

func TestPairAdmin_MutasiTercatatDiAudit(t *testing.T) {
	repo := newFakePairRepo()
	recorder := &pairAuditRecorder{}
	svc := NewPairAdminService(repo, recorder)
	ctx := context.Background()

	cat, err := svc.CreateCategory(ctx, "admin-9", dto.CreatePairCategoryRequest{Key: "crypto", Label: "Crypto"})
	if err != nil {
		t.Fatalf("buat kategori: %v", err)
	}
	if _, err := svc.CreatePair(ctx, "admin-9", dto.CreatePairRequest{Symbol: "BTCUSD", CategoryID: cat.ID}); err != nil {
		t.Fatalf("buat pair: %v", err)
	}

	if len(recorder.entries) != 2 {
		t.Fatalf("baris audit = %d, ingin 2", len(recorder.entries))
	}
	want := []audit.Action{audit.PairCategoryChange, audit.PairChange}
	for i, entry := range recorder.entries {
		if entry.Action != string(want[i]) {
			t.Errorf("audit[%d].Action = %q, ingin %q", i, entry.Action, want[i])
		}
		if !audit.Known(audit.Action(entry.Action)) {
			t.Errorf("aksi %q tidak ada di katalog audit", entry.Action)
		}
		if entry.ActorID != "admin-9" {
			t.Errorf("audit[%d].ActorID = %q, ingin admin-9", i, entry.ActorID)
		}
		if !strings.Contains(entry.Metadata, `"operation":"create"`) {
			t.Errorf("audit[%d] metadata tanpa operasi create: %s", i, entry.Metadata)
		}
	}
}

func TestPairAdmin_PairTidakBolehMasukKategoriNonaktif(t *testing.T) {
	repo := newFakePairRepo()
	svc := NewPairAdminService(repo, &pairAuditRecorder{})
	ctx := context.Background()

	off := false
	cat, err := svc.CreateCategory(ctx, "admin-1", dto.CreatePairCategoryRequest{Key: "crypto", Label: "Crypto", IsEnabled: &off})
	if err != nil {
		t.Fatalf("buat kategori: %v", err)
	}
	_, err = svc.CreatePair(ctx, "admin-1", dto.CreatePairRequest{Symbol: "BTCUSD", CategoryID: cat.ID})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("pair di kategori nonaktif harus bad request, dapat %v", err)
	}
}

// Korelasi SMT boleh lebih dari satu, dinormalisasi, dan dipetakan ke simbol
// kanonik lewat alias katalog.
func TestPairAdmin_KorelasiSMTDinormalisasi(t *testing.T) {
	repo := newFakePairRepo()
	forex, _, _ := seedPairFixture(t, repo)
	svc := NewPairAdminService(repo, &pairAuditRecorder{})
	ctx := context.Background()

	view, err := svc.CreatePair(ctx, "admin-1", dto.CreatePairRequest{
		Symbol:            "AUDJPY",
		CategoryID:        forex.ID,
		CorrelatedSymbols: " gold, nas100 ,,GOLD,US100 ",
	})
	if err != nil {
		t.Fatalf("buat pair berkorelasi: %v", err)
	}
	// GOLD → XAUUSD, NAS100/US100 → USTEC; duplikat dibuang, urut abjad.
	if view.CorrelatedSymbols != "USTEC,XAUUSD" {
		t.Fatalf("korelasi = %q, ingin USTEC,XAUUSD", view.CorrelatedSymbols)
	}
}

func TestPairAdmin_KorelasiDiriSendiriDitolak(t *testing.T) {
	repo := newFakePairRepo()
	forex, _, _ := seedPairFixture(t, repo)
	svc := NewPairAdminService(repo, &pairAuditRecorder{})
	ctx := context.Background()

	// Saat membuat pair: simbol sama dengan dirinya sendiri.
	_, err := svc.CreatePair(ctx, "admin-1", dto.CreatePairRequest{
		Symbol: "AUDCAD", CategoryID: forex.ID, CorrelatedSymbols: "audcad",
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("korelasi diri sendiri saat create harus bad request, dapat %v", err)
	}

	// Saat mengubah: alias yang menunjuk ke simbol itu sendiri juga ditolak.
	eur, err := repo.FindPairBySymbol(ctx, "EURUSD")
	if err != nil {
		t.Fatalf("cari EURUSD: %v", err)
	}
	self := "EURUSD"
	if err := svc.UpdatePair(ctx, "admin-1", eur.ID, dto.UpdatePairRequest{CorrelatedSymbols: &self}); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("korelasi diri sendiri saat update harus bad request, dapat %v", err)
	}
}

func TestPairAdmin_KorelasiTakDikenalDitolak(t *testing.T) {
	repo := newFakePairRepo()
	forex, _, _ := seedPairFixture(t, repo)
	svc := NewPairAdminService(repo, &pairAuditRecorder{})
	ctx := context.Background()

	_, err := svc.CreatePair(ctx, "admin-1", dto.CreatePairRequest{
		Symbol: "AUDCAD", CategoryID: forex.ID, CorrelatedSymbols: "NGAWUR",
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("korelasi di luar katalog harus bad request, dapat %v", err)
	}
}

func TestPairAdmin_KorelasiDapatDikosongkan(t *testing.T) {
	repo := newFakePairRepo()
	forex, _, _ := seedPairFixture(t, repo)
	svc := NewPairAdminService(repo, &pairAuditRecorder{})
	ctx := context.Background()

	view, err := svc.CreatePair(ctx, "admin-1", dto.CreatePairRequest{
		Symbol: "AUDCAD", CategoryID: forex.ID, CorrelatedSymbols: "EURUSD",
	})
	if err != nil {
		t.Fatalf("buat pair: %v", err)
	}

	empty := ""
	if err := svc.UpdatePair(ctx, "admin-1", view.ID, dto.UpdatePairRequest{CorrelatedSymbols: &empty}); err != nil {
		t.Fatalf("kosongkan korelasi: %v", err)
	}

	pair, err := repo.FindPairByID(ctx, view.ID)
	if err != nil {
		t.Fatalf("baca pair: %v", err)
	}
	if pair.CorrelatedSymbols != "" {
		t.Fatalf("korelasi = %q, ingin kosong", pair.CorrelatedSymbols)
	}
}

// --- Katalog publik ---------------------------------------------------------

func TestPairCatalog_HanyaYangAktifTampil(t *testing.T) {
	repo := newFakePairRepo()
	_, indices, usd := seedPairFixture(t, repo)
	svc := NewPairCatalogService(repo)
	ctx := context.Background()

	symbols, err := svc.EnabledSymbols(ctx)
	if err != nil {
		t.Fatalf("EnabledSymbols: %v", err)
	}
	if got := strings.Join(symbols, ","); got != "EURUSD,GBPUSD,USTEC,XAUUSD" {
		t.Fatalf("simbol aktif = %s", got)
	}

	// Pair dinonaktifkan: hilang dari katalog.
	for _, p := range repo.pairs {
		if p.Symbol == "GBPUSD" {
			p.IsEnabled = false
		}
	}
	// Grup dinonaktifkan: seluruh pair di dalamnya ikut tersembunyi.
	usd.IsEnabled = false
	_ = repo.UpdateGroup(ctx, usd)
	indices.IsEnabled = false
	_ = repo.UpdateCategory(ctx, indices)

	catalog, err := svc.Catalog(ctx)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	got := append([]string(nil), catalog.Symbols...)
	sort.Strings(got)
	// EURUSD tersembunyi karena grup usd mati, GBPUSD karena pair mati,
	// USTEC karena kategori mati. Tersisa XAUUSD (tanpa grup).
	if strings.Join(got, ",") != "XAUUSD" {
		t.Fatalf("simbol katalog = %v, ingin hanya XAUUSD", got)
	}
}

func TestPairCatalog_PairTanpaGrupMasukUmum(t *testing.T) {
	repo := newFakePairRepo()
	seedPairFixture(t, repo)
	svc := NewPairCatalogService(repo)

	catalog, err := svc.Catalog(context.Background())
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	for _, category := range catalog.Categories {
		if category.Key != "forex" {
			continue
		}
		for _, group := range category.Groups {
			if group.Key != "umum" {
				continue
			}
			if len(group.Pairs) != 1 || group.Pairs[0].Symbol != "XAUUSD" {
				t.Fatalf("grup umum = %+v, ingin XAUUSD saja", group.Pairs)
			}
			return
		}
	}
	t.Fatal("grup sintetis umum tidak ditemukan di kategori forex")
}

// Katalog publik memuat daftar korelasi SMT agar frontend bisa menampilkan
// instrumen pembanding pada kartu pair.
func TestPairCatalog_MemuatKorelasiSMT(t *testing.T) {
	repo := newFakePairRepo()
	seedPairFixture(t, repo)
	for i := range repo.pairs {
		if repo.pairs[i].Symbol == "XAUUSD" {
			repo.pairs[i].CorrelatedSymbols = "EURUSD,USTEC"
		}
	}
	svc := NewPairCatalogService(repo)

	catalog, err := svc.Catalog(context.Background())
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	for _, category := range catalog.Categories {
		for _, group := range category.Groups {
			for _, pair := range group.Pairs {
				if pair.Symbol != "XAUUSD" {
					continue
				}
				if got := strings.Join(pair.CorrelatedSymbols, ","); got != "EURUSD,USTEC" {
					t.Fatalf("korelasi XAUUSD = %q, ingin EURUSD,USTEC", got)
				}
				return
			}
		}
	}
	t.Fatal("XAUUSD tidak ditemukan di katalog")
}

// Instrumen pembanding yang tidak tampil (nonaktif) tidak boleh disebut di
// katalog publik, karena engine juga tidak meminta feed-nya.
func TestPairCatalog_KorelasiNonaktifTidakDitampilkan(t *testing.T) {
	repo := newFakePairRepo()
	seedPairFixture(t, repo)
	for i := range repo.pairs {
		switch repo.pairs[i].Symbol {
		case "XAUUSD":
			repo.pairs[i].CorrelatedSymbols = "EURUSD,GBPUSD"
		case "EURUSD":
			repo.pairs[i].IsEnabled = false
		}
	}
	svc := NewPairCatalogService(repo)

	catalog, err := svc.Catalog(context.Background())
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	for _, category := range catalog.Categories {
		for _, group := range category.Groups {
			for _, pair := range group.Pairs {
				if pair.Symbol != "XAUUSD" {
					continue
				}
				if got := strings.Join(pair.CorrelatedSymbols, ","); got != "GBPUSD" {
					t.Fatalf("korelasi XAUUSD = %q, ingin hanya GBPUSD", got)
				}
				return
			}
		}
	}
	t.Fatal("XAUUSD tidak ditemukan di katalog")
}

func TestPairCatalog_ResolveAliasKeKanonik(t *testing.T) {
	repo := newFakePairRepo()
	seedPairFixture(t, repo)
	svc := NewPairCatalogService(repo)
	ctx := context.Background()

	for input, want := range map[string]string{
		"nas100": "USTEC", "US100": "USTEC", "gold": "XAUUSD", " eurusd ": "EURUSD",
	} {
		got, err := svc.ResolveSymbol(ctx, input)
		if err != nil {
			t.Fatalf("ResolveSymbol(%q): %v", input, err)
		}
		if got != want {
			t.Errorf("ResolveSymbol(%q) = %q, ingin %q", input, got, want)
		}
	}

	if _, err := svc.ResolveSymbol(ctx, "FOOBAR"); !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("simbol tak dikenal harus bad request, dapat %v", err)
	}

	// Simbol yang dinonaktifkan admin ditolak walau terdaftar.
	for _, p := range repo.pairs {
		if p.Symbol == "GBPUSD" {
			p.IsEnabled = false
		}
	}
	if _, err := svc.ResolveSymbol(ctx, "GBPUSD"); !errors.Is(err, apperror.ErrBadRequest) {
		t.Errorf("simbol nonaktif harus bad request, dapat %v", err)
	}
}

// --- Pilihan pair per akun --------------------------------------------------

// pairFeatureReader mengganti nilai fitur paket pada tes.
type pairFeatureReader map[string]string

func (f pairFeatureReader) Feature(_ context.Context, tier, key, fallback string) string {
	if v, ok := f[tier+"."+key]; ok {
		return v
	}
	return fallback
}

type accountPairFixture struct {
	svc      AccountService
	accounts *fakeAccountRepo
	subs     *fakeSubRepo
	pairs    *fakePairRepo
}

func newAccountPairFixture(t *testing.T, tier string, features TierFeatureReader) accountPairFixture {
	t.Helper()

	user := newTestUser(tier)
	users := newFakeUserRepo()
	users.users[user.Email] = user
	accounts := &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}}
	creds := &fakeCredRepo{creds: map[string]*model.Credential{}}
	subs := &fakeSubRepo{}
	pairs := newFakePairRepo()
	seedPairFixture(t, pairs)

	svc := NewAccountService(AccountServiceDeps{
		Users:             users,
		Accounts:          accounts,
		Credentials:       creds,
		Subscriptions:     subs,
		Audit:             &fakeAuditRepo{},
		Sealer:            testKeyring(t),
		Strategies:        &fakeStrategies{ila3: "off"},
		FingerprintSecret: fingerprintTestSecret,
		Features:          features,
		Catalog:           NewPairCatalogService(pairs),
	})
	return accountPairFixture{svc: svc, accounts: accounts, subs: subs, pairs: pairs}
}

func (f accountPairFixture) addAccount(t *testing.T, token string, risk float64) *dto.AccountResponse {
	t.Helper()
	res, err := f.svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: token, RiskPercent: risk,
	})
	if err != nil {
		t.Fatalf("buat akun: %v", err)
	}
	return res
}

func TestAccountPairs_SimpanDanBacaPerAkun(t *testing.T) {
	fx := newAccountPairFixture(t, model.TierPlus, nil)
	ctx := context.Background()
	first := fx.addAccount(t, sampleToken+"-a", 0.5)
	second := fx.addAccount(t, sampleToken+"-b", 0.5)

	// Alias dikanonikalkan; duplikat dibuang.
	res, err := fx.svc.UpdateAccountPairs(ctx, userIDTest, first.ID, dto.UpdateAccountPairsRequest{
		Symbols: []string{"gold", "XAUUSD", "nas100"},
	})
	if err != nil {
		t.Fatalf("UpdateAccountPairs: %v", err)
	}
	sort.Strings(res.Selected)
	if strings.Join(res.Selected, ",") != "USTEC,XAUUSD" {
		t.Fatalf("selected = %v, ingin USTEC,XAUUSD", res.Selected)
	}

	// Akun kedua tidak ikut berubah.
	other, err := fx.svc.AccountPairs(ctx, userIDTest, second.ID)
	if err != nil {
		t.Fatalf("AccountPairs: %v", err)
	}
	if len(other.Selected) != 0 {
		t.Fatalf("akun kedua selected = %v, ingin kosong", other.Selected)
	}
}

func TestAccountPairs_SemuaAtauTidakSama(t *testing.T) {
	fx := newAccountPairFixture(t, model.TierPlus, nil)
	ctx := context.Background()
	account := fx.addAccount(t, sampleToken+"-a", 0.5)

	if _, err := fx.svc.UpdateAccountPairs(ctx, userIDTest, account.ID, dto.UpdateAccountPairsRequest{
		Symbols: []string{"EURUSD"},
	}); err != nil {
		t.Fatalf("pilihan awal: %v", err)
	}

	// Satu simbol tidak sah menggagalkan seluruh perubahan.
	_, err := fx.svc.UpdateAccountPairs(ctx, userIDTest, account.ID, dto.UpdateAccountPairsRequest{
		Symbols: []string{"GBPUSD", "NGAWUR"},
	})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("simbol tak sah harus bad request, dapat %v", err)
	}

	res, err := fx.svc.AccountPairs(ctx, userIDTest, account.ID)
	if err != nil {
		t.Fatalf("AccountPairs: %v", err)
	}
	if strings.Join(res.Selected, ",") != "EURUSD" {
		t.Fatalf("pilihan berubah setelah gagal: %v", res.Selected)
	}
}

func TestAccountPairs_BatasSimbolPerTier(t *testing.T) {
	features := pairFeatureReader{model.TierBasic + "." + model.FeatureMaxSymbols: "2"}
	fx := newAccountPairFixture(t, model.TierBasic, features)
	ctx := context.Background()
	account := fx.addAccount(t, sampleToken+"-a", 0.5)

	if _, err := fx.svc.UpdateAccountPairs(ctx, userIDTest, account.ID, dto.UpdateAccountPairsRequest{
		Symbols: []string{"EURUSD", "GBPUSD"},
	}); err != nil {
		t.Fatalf("dua simbol harus lolos: %v", err)
	}
	_, err := fx.svc.UpdateAccountPairs(ctx, userIDTest, account.ID, dto.UpdateAccountPairsRequest{
		Symbols: []string{"EURUSD", "GBPUSD", "XAUUSD"},
	})
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("tiga simbol harus forbidden, dapat %v", err)
	}
}

func TestAccountPairs_AkunMilikUserLainDitolak(t *testing.T) {
	fx := newAccountPairFixture(t, model.TierPlus, nil)
	ctx := context.Background()
	account := fx.addAccount(t, sampleToken+"-a", 0.5)

	_, err := fx.svc.UpdateAccountPairs(ctx, "user-lain", account.ID, dto.UpdateAccountPairsRequest{
		Symbols: []string{"EURUSD"},
	})
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("akun user lain harus not found, dapat %v", err)
	}
	if _, err := fx.svc.AccountPairs(ctx, "user-lain", account.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("baca akun user lain harus not found, dapat %v", err)
	}
}

func TestSubscribe_MenolakSimbolDiLuarKatalog(t *testing.T) {
	fx := newAccountPairFixture(t, model.TierPlus, nil)
	ctx := context.Background()

	if _, err := fx.svc.Subscribe(ctx, userIDTest, dto.CreateSubscriptionRequest{Symbol: "NGAWUR"}); !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("simbol di luar katalog harus bad request, dapat %v", err)
	}

	res, err := fx.svc.Subscribe(ctx, userIDTest, dto.CreateSubscriptionRequest{Symbol: "nas100"})
	if err != nil {
		t.Fatalf("alias katalog harus lolos: %v", err)
	}
	if res.Symbol != "USTEC" {
		t.Errorf("simbol tersimpan %q, ingin USTEC", res.Symbol)
	}
}

// --- Batas risiko per akun --------------------------------------------------

func TestRiskGuard_TotalRisikoSemuaAkunDibatasi(t *testing.T) {
	features := pairFeatureReader{model.TierPlus + "." + model.FeatureMaxTotalRisk: "1"}
	fx := newAccountPairFixture(t, model.TierPlus, features)
	ctx := context.Background()

	first := fx.addAccount(t, sampleToken+"-a", 0.6)

	// 0.6 + 0.6 = 1.2 melewati batas total 1.0.
	_, err := fx.svc.Create(ctx, userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken + "-b", RiskPercent: 0.6,
	})
	if !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("total risiko berlebih harus forbidden, dapat %v", err)
	}

	// 0.6 + 0.4 = 1.0 tepat di batas, boleh.
	second := fx.addAccount(t, sampleToken+"-b", 0.4)

	// Menaikkan akun kedua ke 0.5 menembus batas.
	up := 0.5
	if _, err := fx.svc.Update(ctx, userIDTest, second.ID, dto.UpdateAccountRequest{RiskPercent: &up}); !errors.Is(err, apperror.ErrForbidden) {
		t.Fatalf("kenaikan risiko melewati total harus forbidden, dapat %v", err)
	}

	// Menurunkan risiko akun pertama selalu boleh.
	down := 0.3
	if _, err := fx.svc.Update(ctx, userIDTest, first.ID, dto.UpdateAccountRequest{RiskPercent: &down}); err != nil {
		t.Fatalf("penurunan risiko harus lolos: %v", err)
	}
}

func TestRiskGuard_BatasPerTradeTetapDitegakkan(t *testing.T) {
	fx := newAccountPairFixture(t, model.TierPlus, nil)

	for _, risk := range []float64{0.05, 1.2} {
		_, err := fx.svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
			BrokerType: "matchtrader", Token: sampleToken, RiskPercent: risk,
		})
		if !errors.Is(err, apperror.ErrBadRequest) {
			t.Errorf("risiko %.2f harus bad request, dapat %v", risk, err)
		}
	}
}

func TestRiskGuard_BatasBawaanBasicSatuPersen(t *testing.T) {
	if got := defaultMaxTotalRisk(model.TierBasic); got != 1.0 {
		t.Errorf("default basic = %v, ingin 1.0", got)
	}
	if got := defaultMaxTotalRisk(model.TierPlus); got != 2.0 {
		t.Errorf("default plus = %v, ingin 2.0", got)
	}
	if got := defaultMaxTotalRisk(model.TierFree); got != 0 {
		t.Errorf("default free = %v, ingin 0", got)
	}
}
