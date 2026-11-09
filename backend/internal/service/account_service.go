package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/ctraderapi"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/cryptox"
)

// Batas risiko per trade yang berlaku untuk semua tier (plans/14 §6).
const (
	MinRiskPercent = 0.10
	// MaxRiskPercent adalah pagar per-trade. Nilai di atas 1% tetap diizinkan
	// tetapi wajib dikonfirmasi user di UI; 5% adalah batas paling agresif yang
	// masih masuk akal untuk akun retail.
	MaxRiskPercent = 5.00
)

// Batas jumlah simbol yang dipantau per user, menjaga beban engine.
const MaxSubscriptionsPerUser = 8

// Sealer mengenkripsi kredensial sebelum disimpan.
// Layer service hanya melihat antarmuka ini; kunci tidak pernah menyentuhnya.
type Sealer interface {
	Seal(plaintext, associatedData []byte) (cryptox.Envelope, error)
	ActiveVersion() int
}

// AccountStrategyLister membaca saklar metode per akun milik satu pengguna.
// Dipersempit agar AccountService tidak bergantung pada seluruh kontrak
// repository saklar.
type AccountStrategyLister interface {
	ListByUser(ctx context.Context, userID string) ([]model.AccountPairStrategy, error)
}

// AccountService mengelola akun broker user dan langganan simbol.
//
// Batas tier ditegakkan di sini (plans/14 §5.1), bukan di Runner, karena
// pemeriksaan terjadi saat user menambah akun.
type AccountService interface {
	List(ctx context.Context, userID string) ([]dto.AccountResponse, error)
	Create(ctx context.Context, userID string, req dto.CreateAccountRequest) (*dto.AccountResponse, error)
	Update(ctx context.Context, userID, accountID string, req dto.UpdateAccountRequest) (*dto.AccountResponse, error)
	Delete(ctx context.Context, userID, accountID string) error

	ListSubscriptions(ctx context.Context, userID string) ([]dto.SubscriptionResponse, error)
	Subscribe(ctx context.Context, userID string, req dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error)
	Unsubscribe(ctx context.Context, userID, subscriptionID string) error
	Strategies(ctx context.Context, userID string) ([]dto.StrategyResponse, error)

	// cTrader OAuth
	GetCTraderAuthURL(ctx context.Context) (dto.CTraderAuthURLResponse, error)
	ConnectCTrader(ctx context.Context, userID string, req dto.CTraderCallbackRequest) (*dto.CTraderCallbackResponse, error)

	// Checklist pair per akun (katalog dikelola admin).
	AccountPairs(ctx context.Context, userID, accountID string) (*dto.AccountPairSelectionResponse, error)
	UpdateAccountPairs(ctx context.Context, userID, accountID string, req dto.UpdateAccountPairsRequest) (*dto.AccountPairSelectionResponse, error)
}

type accountService struct {
	users      UserRepository
	accounts   BrokerAccountRepository
	creds      CredentialRepository
	subs       SubscriptionRepository
	states     BrokerAccountStateLister
	audit      AuditRepository
	sealer     Sealer
	strategies StrategyCatalog
	revoker    CredentialRevoker
	// legal memastikan disclaimer risiko disetujui sebelum akun broker nyata
	// dihubungkan (plans/09 §2.2).
	legal AccountLinkGuard
	// features membaca batas akun dan simbol per tier dari katalog paket.
	features TierFeatureReader
	// secret mengunci sidik jari token agar tidak bisa dibalik dari database.
	secret  string
	clock   func() time.Time
	ctrader CTraderOAuthClient
	// catalog menyajikan katalog pair aktif dan memvalidasi simbol pilihan user.
	catalog PairCatalogService
	// seeder menyiapkan saklar metode bawaan saat pair baru dipantau.
	seeder StrategySwitchService
	// switches membaca saklar metode per akun untuk laporan status strategi.
	switches AccountStrategyLister
}

// CTraderOAuthClient mendefinisikan kontrak komunikasi ke REST API cTrader Spotware.
type CTraderOAuthClient interface {
	BuildAuthURL(scope string) string
	ExchangeToken(ctx context.Context, code string) (*ctraderapi.TokenResponse, error)
	GetAccounts(ctx context.Context, accessToken string) ([]ctraderapi.AccountInfo, error)
}

// AccountLinkGuard memeriksa syarat sebelum akun broker dihubungkan.
type AccountLinkGuard interface {
	RequireAccountLink(ctx context.Context, userID string) error
}

// ErrDuplicateCredential menandai token yang sudah dipakai akun broker lain.
//
// Satu token adalah satu akun broker. Bila token yang sama didaftarkan dua
// kali, Runner mengeksekusi sinyal yang sama dua kali ke akun yang sama,
// sehingga risiko user menjadi dua kali lipat (plans/12 §3.4).
var ErrDuplicateCredential = errors.New("token ini sudah dipakai akun broker lain")

// CredentialRevoker meminta Runner memutus sesi broker setelah kredensial
// dihapus, agar token lama tidak tetap hidup di memori (plans/12 par 3.4).
type CredentialRevoker interface {
	RevokeAccount(ctx context.Context, accountID string) error
}

// TierFeatureReader membaca batas paket dari database.
// Batas tier sengaja tidak ditulis di kode: admin harus bisa mengubahnya
// tanpa deploy (plans/19 C2).
type TierFeatureReader interface {
	Feature(ctx context.Context, tier, key, fallback string) string
}

// AccountServiceDeps mengumpulkan dependensi agar konstruktor tetap ringkas.
type AccountServiceDeps struct {
	Users         UserRepository
	Accounts      BrokerAccountRepository
	Credentials   CredentialRepository
	Subscriptions SubscriptionRepository
	States        BrokerAccountStateLister
	Audit         AuditRepository
	Sealer        Sealer
	Strategies    StrategyCatalog
	Revoker       CredentialRevoker
	Legal         AccountLinkGuard
	// Features membaca batas akun dan simbol per tier. Bila nil, dipakai
	// batas bawaan.
	Features TierFeatureReader
	// FingerprintSecret mengunci sidik jari token. Disarankan sama dengan
	// kunci internal; kosong membuat pemeriksaan duplikat dimatikan.
	FingerprintSecret string
	Clock             func() time.Time
	CTrader           CTraderOAuthClient
	// Catalog dipakai memvalidasi pilihan pair per akun. Bila nil, seluruh
	// simbol yang lolos normalisasi diterima (kompatibel dengan pemasangan lama).
	Catalog PairCatalogService
	// SwitchSeeder menyiapkan saklar metode bawaan saat pair baru dipantau,
	// sehingga tiap akun langsung punya pilihan metode yang bisa diubah.
	SwitchSeeder StrategySwitchService
	// Switches membaca saklar metode per akun untuk laporan status strategi.
	Switches AccountStrategyLister
}

// StrategyCatalog melaporkan ketersediaan strategi bagi user.
type StrategyCatalog interface {
	StrategyStatus(strategy string) string
	StrategySelectable(strategy string) bool
}

// NewAccountService membuat service akun broker.
func NewAccountService(deps AccountServiceDeps) AccountService {
	clock := deps.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &accountService{
		users:      deps.Users,
		accounts:   deps.Accounts,
		creds:      deps.Credentials,
		subs:       deps.Subscriptions,
		states:     deps.States,
		audit:      deps.Audit,
		sealer:     deps.Sealer,
		strategies: deps.Strategies,
		revoker:    deps.Revoker,
		legal:      deps.Legal,
		features:   deps.Features,
		secret:     deps.FingerprintSecret,
		clock:      clock,
		ctrader:    deps.CTrader,
		catalog:    deps.Catalog,
		seeder:     deps.SwitchSeeder,
		switches:   deps.Switches,
	}
}

// SetStrategySwitchSeeder memasang penyiap saklar metode setelah service saklar
// jadi. Dipisah karena service saklar membaca hak tier lewat AccountService,
// sehingga keduanya saling butuh dan tidak bisa dirangkai dalam satu literal.
func SetStrategySwitchSeeder(target AccountService, seeder StrategySwitchService) {
	if svc, ok := target.(*accountService); ok {
		svc.seeder = seeder
	}
}

// SetAccountStrategyLister memasang pembaca saklar metode untuk laporan status
// strategi. Dipisah dengan alasan yang sama seperti SetStrategySwitchSeeder.
func SetAccountStrategyLister(target AccountService, lister AccountStrategyLister) {
	if svc, ok := target.(*accountService); ok {
		svc.switches = lister
	}
}

// List mengembalikan akun broker user beserta telemetry terakhir.
func (s *accountService) List(ctx context.Context, userID string) ([]dto.AccountResponse, error) {
	accounts, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	states := make(map[string]model.BrokerAccountState)
	if s.states != nil {
		rows, stateErr := s.states.ListByUser(ctx, userID)
		if stateErr != nil {
			return nil, stateErr
		}
		for _, row := range rows {
			states[row.BrokerAccountID] = row
		}
	}

	out := make([]dto.AccountResponse, 0, len(accounts))
	for i := range accounts {
		out = append(out, s.toResponse(ctx, &accounts[i], states[accounts[i].ID]))
	}
	return out, nil
}

// Create mendaftarkan akun baru, menegakkan batas tier, lalu menyimpan
// kredensial terenkripsi. Token asli tidak pernah disimpan atau dicatat.
func (s *accountService) Create(ctx context.Context, userID string, req dto.CreateAccountRequest) (*dto.AccountResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Alamat email wajib terbukti sebelum akun broker boleh dihubungkan.
	if user.EmailVerifiedAt == nil {
		return nil, ErrEmailNotVerified
	}

	// Disclaimer risiko wajib disetujui sebelum akun nyata dihubungkan:
	// sebelum ada akun, user belum menanggung risiko finansial apa pun.
	if s.legal != nil {
		if err := s.legal.RequireAccountLink(ctx, userID); err != nil {
			return nil, err
		}
	}

	tier := effectiveTier(user, s.clock())
	limit := s.tierAccountLimit(ctx, tier)
	// Selama trial hanya satu akun, berapa pun paket aslinya.
	if InTrialPeriod(user, s.clock()) && limit > model.MaxAccountsDuringTrial {
		limit = model.MaxAccountsDuringTrial
	}
	if limit == 0 {
		return nil, fmt.Errorf("%w: tier %s tidak dapat menghubungkan akun broker", apperror.ErrForbidden, tier)
	}

	active, err := s.accounts.CountActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if int(active) >= limit {
		return nil, fmt.Errorf("%w: tier %s maksimal %d akun", apperror.ErrForbidden, tier, limit)
	}

	brokerType := strings.ToLower(strings.TrimSpace(req.BrokerType))
	if err := validateCredential(brokerType, req); err != nil {
		return nil, err
	}

	// Token yang sudah dipakai akun lain (termasuk akun simulasi) ditolak.
	if token := strings.TrimSpace(req.Token); token != "" {
		taken, err := s.tokenTaken(ctx, userID, token, "")
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrDuplicateCredential
		}
	}

	mode := req.Mode
	if mode == "" {
		mode = "manual"
	}
	if mode == "auto" && tier != "plus" {
		return nil, fmt.Errorf("%w: eksekusi otomatis hanya untuk tier Plus", apperror.ErrForbidden)
	}

	riskPercent, err := s.validateRiskForTier(ctx, tier, req.RiskPercent)
	if err != nil {
		return nil, err
	}

	riskMode, lotSize, err := s.resolveRiskSettings(req.RiskMode, req.LotSize, riskPercent, "")
	if err != nil {
		return nil, err
	}

	// Total risiko seluruh akun aktif tidak boleh melewati batas tier.
	existing, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := s.validateTotalRisk(ctx, tier, existing, riskPercent, ""); err != nil {
		return nil, err
	}

	account := &model.BrokerAccount{
		UserID:      userID,
		BrokerType:  brokerType,
		Label:       strings.TrimSpace(req.Label),
		Mode:        mode,
		RiskMode:    riskMode,
		RiskPercent: riskPercent,
		LotSize:     lotSize,
		IsActive:    true,
		IsDryRun:    req.IsDryRun,
	}

	if account.Label == "" {
		account.Label = strings.ToUpper(brokerType)
	}

	if err := s.accounts.Create(ctx, account); err != nil {
		return nil, err
	}

	if err := s.storeCredential(ctx, userID, account.ID, brokerType, req); err != nil {
		// Akun tanpa kredensial sah tidak boleh tertinggal setengah jadi.
		account.IsActive = false
		account.LastError = "kredensial gagal disimpan"
		_ = s.accounts.Update(ctx, account)
		return nil, err
	}

	s.recordAudit(ctx, userID, account, "account.create", "")

	// Pair yang sudah dipantau sebelum akun ini ada belum punya saklar untuknya.
	// Siapkan sekarang agar metode langsung bisa dipilih di halaman Sinyal.
	s.seedSaklarUntukLangganan(ctx, userID)

	return ptr(s.toResponse(ctx, account, model.BrokerAccountState{})), nil
}

// seedSaklarUntukLangganan menyiapkan saklar bawaan untuk semua pair yang sudah
// dipantau user. Dipanggil setelah akun baru lahir, karena saklar bermakna per
// akun sehingga pair lama belum punya baris untuk akun yang baru ditambahkan.
func (s *accountService) seedSaklarUntukLangganan(ctx context.Context, userID string) {
	if s.seeder == nil {
		return
	}
	subs, err := s.subs.ListActiveByUser(ctx, userID)
	if err != nil {
		return
	}
	seen := make(map[string]bool, len(subs))
	for _, sub := range subs {
		if sub.Symbol == "" || seen[sub.Symbol] {
			continue
		}
		seen[sub.Symbol] = true
		_ = s.seeder.EnsureDefaults(ctx, userID, sub.Symbol)
	}
}

// Update mengubah pengaturan akun; kredensial tidak ikut berubah.
func (s *accountService) Update(ctx context.Context, userID, accountID string, req dto.UpdateAccountRequest) (*dto.AccountResponse, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, apperror.ErrNotFound
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	tier := effectiveTier(user, s.clock())

	if req.Mode != nil {
		if *req.Mode == "auto" && tier != "plus" {
			return nil, fmt.Errorf("%w: eksekusi otomatis hanya untuk tier Plus", apperror.ErrForbidden)
		}
		account.Mode = *req.Mode
	}
	if req.Label != nil {
		account.Label = strings.TrimSpace(*req.Label)
	}
	if req.RiskPercent != nil {
		riskPercent, riskErr := s.validateRiskForTier(ctx, tier, *req.RiskPercent)
		if riskErr != nil {
			return nil, riskErr
		}
		if riskPercent < account.RiskPercent {
			// Menurunkan risiko tidak pernah melewati batas total.
			account.RiskPercent = riskPercent
		} else {
			siblings, listErr := s.accounts.ListByUser(ctx, userID)
			if listErr != nil {
				return nil, listErr
			}
			if totalErr := s.validateTotalRisk(ctx, tier, siblings, riskPercent, account.ID); totalErr != nil {
				return nil, totalErr
			}
			account.RiskPercent = riskPercent
		}
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}
	if req.IsDryRun != nil {
		account.IsDryRun = *req.IsDryRun
	}

	// Mode risiko dan ukuran lot dapat diubah terpisah dari persentase. Bila
	// hanya salah satu dikirim, nilai lama dipakai sebagai pasangan.
	if req.RiskMode != nil || req.LotSize != nil {
		modeDiminta := account.RiskMode
		if req.RiskMode != nil {
			modeDiminta = *req.RiskMode
		}
		lotDiminta := account.LotSize
		if req.LotSize != nil {
			lotDiminta = *req.LotSize
		}
		riskMode, lotSize, riskErr := s.resolveRiskSettings(modeDiminta, lotDiminta, account.RiskPercent, account.ID)
		if riskErr != nil {
			return nil, riskErr
		}
		account.RiskMode = riskMode
		account.LotSize = lotSize
	}

	// Rotasi kredensial: token baru ditulis lalu sesi lama di Runner diputus,
	// sehingga login berikutnya memakai token terbaru (plans/12 §3.4).
	credentialRotated := false
	if req.Token != nil && strings.TrimSpace(*req.Token) != "" {
		rotate := dto.CreateAccountRequest{
			BrokerType: account.BrokerType,
			Token:      *req.Token,
		}
		if req.BaseURL != nil {
			rotate.BaseURL = *req.BaseURL
		}
		if err := validateCredential(account.BrokerType, rotate); err != nil {
			return nil, err
		}
		taken, err := s.tokenTaken(ctx, userID, rotate.Token, account.ID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, ErrDuplicateCredential
		}
		if err := s.storeCredential(ctx, userID, account.ID, account.BrokerType, rotate); err != nil {
			return nil, err
		}
		credentialRotated = true
	}

	if err := s.accounts.Update(ctx, account); err != nil {
		return nil, err
	}

	if credentialRotated && s.revoker != nil {
		// Kegagalan tidak membatalkan penyimpanan: Runner akan gagal login
		// dan mencoba ulang, bukan memakai sesi mati selamanya.
		revokeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = s.revoker.RevokeAccount(revokeCtx, account.ID)
	}

	s.recordAudit(ctx, userID, account, "account.update", "")

	return ptr(s.toResponse(ctx, account, model.BrokerAccountState{})), nil
}

// Delete menghapus akun beserta kredensialnya secara permanen.
// Posisi yang masih hidup di broker tidak disentuh (plans/14 §5.3).
func (s *accountService) Delete(ctx context.Context, userID, accountID string) error {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return err
	}
	if account.UserID != userID {
		return apperror.ErrNotFound
	}

	if err := s.creds.DeleteByAccount(ctx, accountID); err != nil {
		return err
	}

	// Kredensial sudah hilang dari database; sesi di memori Runner juga harus
	// berakhir. Kegagalan di sini tidak membatalkan pencabutan.
	if s.revoker != nil {
		revokeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = s.revoker.RevokeAccount(revokeCtx, accountID)
	}

	account.IsActive = false
	account.IsFrozen = true
	if err := s.accounts.Update(ctx, account); err != nil {
		return err
	}

	s.recordAudit(ctx, userID, account, "account.disconnect", "")
	return nil
}

// ListSubscriptions mengembalikan seluruh langganan simbol user.
func (s *accountService) ListSubscriptions(ctx context.Context, userID string) ([]dto.SubscriptionResponse, error) {
	rows, err := s.subs.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make([]dto.SubscriptionResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toSubscriptionResponse(row))
	}
	return out, nil
}

// Subscribe mengaktifkan simbol pada satu strategi.
func (s *accountService) Subscribe(ctx context.Context, userID string, req dto.CreateSubscriptionRequest) (*dto.SubscriptionResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	tier := effectiveTier(user, s.clock())

	symbol := normalizeSymbol(req.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: simbol tidak sah", apperror.ErrBadRequest)
	}
	// Bila katalog terpasang, simbol wajib terdaftar dan aktif. Ini mencegah
	// user memilih pair yang tidak dikenal engine.
	resolved, resolveErr := s.validateSymbolAgainstCatalog(ctx, symbol)
	if resolveErr != nil {
		return nil, resolveErr
	}
	symbol = resolved

	// Metode tidak lagi ditentukan saat memantau pair. Satu pair dipantau
	// seluruh metode; yang memilih metode adalah saklar per akun. Field
	// req.Strategy tetap diterima (klien lama) tetapi hanya dipakai untuk
	// memvalidasi hak, tidak disimpan.
	if req.Strategy != "" && !knownStrategy(req.Strategy) {
		return nil, fmt.Errorf("%w: strategi tidak dikenal", apperror.ErrBadRequest)
	}
	if req.Strategy == "ila-3.0" {
		if tier != "plus" {
			return nil, fmt.Errorf("%w: ILA 3.0 tersedia untuk tier Plus", apperror.ErrForbidden)
		}
		if !s.strategies.StrategySelectable("ila-3.0") {
			return nil, fmt.Errorf("%w: ILA 3.0 belum tersedia", apperror.ErrForbidden)
		}
	}

	existing, findErr := s.subs.ListByUser(ctx, userID)
	if findErr != nil {
		return nil, findErr
	}
	activeCount := 0
	seenSymbol := make(map[string]bool)
	for _, row := range existing {
		if !row.IsActive {
			continue
		}
		// Satu pair cukup dipantau sekali. Menambah pair yang sudah dipantau
		// mengembalikan baris lama, bukan menggandakan risiko.
		if row.Symbol == symbol {
			return ptr(toSubscriptionResponse(row)), nil
		}
		if !seenSymbol[row.Symbol] {
			seenSymbol[row.Symbol] = true
			activeCount++
		}
	}
	// Batas simbol mengikuti tier dan bisa diubah admin tanpa deploy.
	limit := s.tierSymbolLimit(ctx, tier)
	if activeCount >= limit {
		return nil, fmt.Errorf("%w: tier %s maksimal %d simbol dipantau", apperror.ErrForbidden, tier, limit)
	}

	sub := &model.Subscription{
		UserID:   userID,
		Symbol:   symbol,
		IsActive: true,
	}
	if req.BrokerAccountID != "" {
		account, accErr := s.accounts.FindByID(ctx, req.BrokerAccountID)
		if accErr != nil || account.UserID != userID {
			return nil, apperror.ErrNotFound
		}
		sub.BrokerAccountID = &req.BrokerAccountID
	}

	if err := s.subs.Create(ctx, sub); err != nil {
		return nil, err
	}

	// Audit mencatat bahwa pair mulai dipantau. Metodenya tidak lagi dipilih di
	// sini, jadi dicatat sebagai ILA 2.0 (metode dasar) agar riwayat tetap terbaca.
	s.recordStrategyAudit(ctx, userID, "ila-2.0", "subscription.enable", symbol)

	// Siapkan saklar metode bawaan: ILA 2.0 menyala, ILA 3.0 hanya bila
	// pengguna berhak. Kegagalan di sini tidak membatalkan langganan karena
	// saklar bisa disusun belakangan saat dibuka di halaman Sinyal.
	if s.seeder != nil {
		_ = s.seeder.EnsureDefaults(ctx, userID, symbol)
	}

	return ptr(toSubscriptionResponse(*sub)), nil
}

// Unsubscribe menonaktifkan langganan tanpa menghapus riwayat.
func (s *accountService) Unsubscribe(ctx context.Context, userID, subscriptionID string) error {
	if err := s.subs.Deactivate(ctx, subscriptionID, userID); err != nil {
		return err
	}
	s.recordStrategyAudit(ctx, userID, "", "subscription.disable", subscriptionID)
	return nil
}

// Strategies melaporkan status setiap strategi bagi user.
func (s *accountService) Strategies(ctx context.Context, userID string) ([]dto.StrategyResponse, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	tier := effectiveTier(user, s.clock())

	// `selected` kini berarti: ada minimal satu akun yang menyalakan metode ini.
	// Sebelumnya diturunkan dari kolom langganan, yang sudah dihapus.
	selected := make(map[string]bool, 2)
	if s.switches != nil {
		if rows, listErr := s.switches.ListByUser(ctx, userID); listErr == nil {
			for _, row := range rows {
				if row.IsEnabled {
					selected[row.Strategy] = true
				}
			}
		}
	}

	definitions := []struct {
		version string
		name    string
		desc    string
	}{
		{"ila-2.0", "ILA 2.0", "Strategi stabil untuk semua tier."},
		{"ila-3.0", "ILA 3.0", "Strategi lanjutan khusus tier Plus."},
	}

	out := make([]dto.StrategyResponse, 0, len(definitions))
	for _, def := range definitions {
		allowed := strategyAllowedForTier(tier, def.version)
		available := s.strategies.StrategySelectable(def.version)

		out = append(out, dto.StrategyResponse{
			Version:     def.version,
			Name:        def.name,
			Description: def.desc,
			Status:      s.strategies.StrategyStatus(def.version),
			Available:   available,
			Allowed:     allowed,
			Selected:    selected[def.version],
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Bantuan internal
// ---------------------------------------------------------------------------

func (s *accountService) tokenTaken(ctx context.Context, userID, token, exceptAccountID string) (bool, error) {
	if s.secret == "" {
		// Tanpa kunci sidik jari, duplikat tidak bisa dibuktikan. Menolak lebih
		// aman daripada membiarkan dua akun mengeksekusi sinyal yang sama.
		return false, fmt.Errorf("%w: pemeriksaan token belum dikonfigurasi", apperror.ErrForbidden)
	}

	wanted := tokenFingerprint(s.secret, token)
	accounts, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, account := range accounts {
		if account.ID == exceptAccountID {
			continue
		}
		blob, findErr := s.creds.FindByAccount(ctx, account.ID)
		if findErr != nil {
			if errors.Is(findErr, apperror.ErrNotFound) {
				continue
			}
			return false, findErr
		}
		if blob.TokenFingerprint == wanted && wanted != "" {
			return true, nil
		}
	}
	return false, nil
}

func (s *accountService) storeCredential(ctx context.Context, userID, accountID, brokerType string, req dto.CreateAccountRequest) error {
	if s.sealer == nil {
		return errors.New("enkripsi kredensial belum disiapkan")
	}

	creds := broker.Credentials{
		Platform:  brokerType,
		AccountID: accountID,
		Token:     strings.TrimSpace(req.Token),
	}
	if req.BaseURL != "" {
		creds.Extra = map[string]string{"base_url": strings.TrimSpace(req.BaseURL)}
	}

	plaintext, err := encodeCredentials(creds)
	if err != nil {
		return err
	}

	envelope, err := s.sealer.Seal(plaintext, []byte(accountID))
	if err != nil {
		return fmt.Errorf("mengenkripsi kredensial: %w", err)
	}

	// Sidik jari disimpan terpisah dari ciphertext agar duplikat bisa dikenali
	// tanpa membuka enkripsi setiap akun.
	fingerprint := ""
	if s.secret != "" && creds.Token != "" {
		fingerprint = tokenFingerprint(s.secret, creds.Token)
	}

	return s.creds.Upsert(ctx, &model.Credential{
		BrokerAccountID:  accountID,
		UserID:           userID,
		UserKeyWrapped:   envelope.UserKeyWrapped,
		UserKeyNonce:     envelope.UserKeyNonce,
		Ciphertext:       envelope.Ciphertext,
		Nonce:            envelope.Nonce,
		KeyVersion:       envelope.KeyVersion,
		TokenFingerprint: fingerprint,
		RotatedAt:        s.clock(),
	})
}

func (s *accountService) toResponse(ctx context.Context, account *model.BrokerAccount, state model.BrokerAccountState) dto.AccountResponse {
	hasCredential := false
	if _, err := s.creds.FindByAccount(ctx, account.ID); err == nil {
		hasCredential = true
	}

	return dto.AccountResponse{
		ID:                account.ID,
		BrokerType:        account.BrokerType,
		Label:             account.Label,
		ExternalAccountID: account.ExternalAccountID,
		Mode:              account.Mode,
		RiskMode:          account.RiskMode,
		RiskPercent:       account.RiskPercent,
		LotSize:           account.LotSize,
		IsActive:          account.IsActive,
		IsFrozen:          account.IsFrozen,
		IsDryRun:          account.IsDryRun,
		HasCredential:     hasCredential,
		LastVerifiedAt:    account.LastVerifiedAt,
		LastError:         account.LastError,
		Connected:         state.Connected,
		Balance:           state.Balance,
		Equity:            state.Equity,
		FreeMargin:        state.FreeMargin,
		LastSyncedAt:      state.LastSyncedAt,
	}
}

func (s *accountService) recordAudit(ctx context.Context, userID string, account *model.BrokerAccount, event, note string) {
	entry := &model.AuditLog{
		UserID:   userID,
		Event:    event,
		Entity:   "broker_account",
		EntityID: account.ID,
		Metadata: fmt.Sprintf(`{"broker_type":%q,"dry_run":%t,"note":%q}`, account.BrokerType, account.IsDryRun, note),
	}
	_ = s.audit.Record(ctx, entry)
}

func (s *accountService) recordStrategyAudit(ctx context.Context, userID, strategy, event, target string) {
	entry := &model.AuditLog{
		UserID:   userID,
		Event:    event,
		Entity:   "subscription",
		EntityID: target,
		Metadata: fmt.Sprintf(`{"strategy":%q}`, strategy),
	}
	_ = s.audit.Record(ctx, entry)
}

// validateCredential memastikan bentuk kredensial sesuai platform.
func validateCredential(brokerType string, req dto.CreateAccountRequest) error {
	switch brokerType {
	case "matchtrader":
		token := strings.TrimSpace(req.Token)
		if token == "" {
			return fmt.Errorf("%w: token Match-Trader wajib diisi", apperror.ErrBadRequest)
		}
		// Token disimpan apa adanya, tetapi bentuknya diperiksa agar salah tempel
		// langsung ketahuan sebelum sampai Runner.
		if len(token) < 20 {
			return fmt.Errorf("%w: token Match-Trader tampak tidak lengkap", apperror.ErrBadRequest)
		}
	case "ctrader":
		token := strings.TrimSpace(req.Token)
		if token == "" {
			return fmt.Errorf("%w: token cTrader wajib diisi", apperror.ErrBadRequest)
		}
		if len(token) < 10 {
			return fmt.Errorf("%w: token cTrader tampak tidak lengkap", apperror.ErrBadRequest)
		}
	default:
		return fmt.Errorf("%w: platform tidak dikenal", apperror.ErrBadRequest)
	}
	return nil
}

// clampRisk menerapkan pagar risiko global (plans/14 §6).
func clampRisk(value float64) (float64, error) {
	if value == 0 {
		return 0.5, nil
	}
	if value < MinRiskPercent || value > MaxRiskPercent {
		return 0, fmt.Errorf("%w: risiko per trade harus %.2f%% sampai %.2f%%",
			apperror.ErrBadRequest, MinRiskPercent, MaxRiskPercent)
	}
	return value, nil
}

// tierAccountLimit membaca batas akun broker dari katalog paket.
// Nilai kode hanya cadangan bila baris fitur tidak ada.
func (s *accountService) tierAccountLimit(ctx context.Context, tier string) int {
	fallback := defaultAccountLimit(tier)
	if s.features == nil {
		return fallback
	}
	raw := s.features.Feature(ctx, tier, model.FeatureAccountLimit, strconv.Itoa(fallback))
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

// tierSymbolLimit membaca batas simbol yang dipantau per tier.
func (s *accountService) tierSymbolLimit(ctx context.Context, tier string) int {
	if s.features == nil {
		return MaxSubscriptionsPerUser
	}
	raw := s.features.Feature(ctx, tier, model.FeatureMaxSymbols, strconv.Itoa(MaxSubscriptionsPerUser))
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return MaxSubscriptionsPerUser
	}
	return value
}

// defaultAccountLimit adalah cadangan bila katalog paket tidak terbaca.
func defaultAccountLimit(tier string) int {
	switch tier {
	case model.TierPlus:
		return 2
	case model.TierBasic:
		return 1
	default:
		return 0
	}
}

// effectiveTier menghitung tier yang berlaku: trial berjalan setara Basic.
func effectiveTier(user *model.User, now time.Time) string {
	if user.TrialExpiresAt != nil && now.Before(*user.TrialExpiresAt) {
		return model.TierTrial
	}
	if user.TierExpiresAt != nil && now.After(*user.TierExpiresAt) {
		return "free"
	}
	if user.Tier == "" {
		return "free"
	}
	return user.Tier
}

func strategyAllowedForTier(tier, strategy string) bool {
	if strategy == "ila-3.0" {
		return tier == "plus"
	}
	return true
}

func knownStrategy(strategy string) bool {
	return strategy == "ila-2.0" || strategy == "ila-3.0"
}

// normalizeSymbol membersihkan dan membakukan nama simbol.
func normalizeSymbol(symbol string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	s = strings.ReplaceAll(s, "/", "")
	if len(s) < 3 || len(s) > 20 {
		return ""
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return s
}

func toSubscriptionResponse(sub model.Subscription) dto.SubscriptionResponse {
	out := dto.SubscriptionResponse{
		ID:       sub.ID,
		Symbol:   sub.Symbol,
		IsActive: sub.IsActive,
	}
	if sub.BrokerAccountID != nil {
		out.BrokerAccountID = *sub.BrokerAccountID
	}
	return out
}

func ptr[T any](v T) *T {
	return &v
}

func (s *accountService) GetCTraderAuthURL(ctx context.Context) (dto.CTraderAuthURLResponse, error) {
	if s.ctrader == nil {
		return dto.CTraderAuthURLResponse{}, fmt.Errorf("%w: integrasi cTrader belum dikonfigurasi", apperror.ErrBadRequest)
	}
	url := s.ctrader.BuildAuthURL("trading")
	return dto.CTraderAuthURLResponse{AuthURL: url}, nil
}

func (s *accountService) ConnectCTrader(ctx context.Context, userID string, req dto.CTraderCallbackRequest) (*dto.CTraderCallbackResponse, error) {
	if s.ctrader == nil {
		return nil, fmt.Errorf("%w: integrasi cTrader belum dikonfigurasi", apperror.ErrBadRequest)
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.EmailVerifiedAt == nil {
		return nil, ErrEmailNotVerified
	}

	if s.legal != nil {
		if err := s.legal.RequireAccountLink(ctx, userID); err != nil {
			return nil, err
		}
	}

	tier := effectiveTier(user, s.clock())
	limit := s.tierAccountLimit(ctx, tier)
	if InTrialPeriod(user, s.clock()) && limit > model.MaxAccountsDuringTrial {
		limit = model.MaxAccountsDuringTrial
	}
	if limit == 0 {
		return nil, fmt.Errorf("%w: tier %s tidak dapat menghubungkan akun broker", apperror.ErrForbidden, tier)
	}

	active, err := s.accounts.CountActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if int(active) >= limit {
		return nil, fmt.Errorf("%w: tier %s maksimal %d akun (sudah terhubung %d)", apperror.ErrForbidden, tier, limit, active)
	}

	tokenResp, err := s.ctrader.ExchangeToken(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("%w: pertukaran token cTrader gagal: %v", apperror.ErrBadRequest, err)
	}

	accounts, err := s.ctrader.GetAccounts(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: mengambil daftar akun cTrader gagal: %v", apperror.ErrBadRequest, err)
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("%w: tidak ada akun cTrader yang ditemukan di profil ini", apperror.ErrNotFound)
	}

	mode := req.Mode
	if mode == "" {
		mode = "manual"
	}
	if mode == "auto" && tier != "plus" {
		return nil, fmt.Errorf("%w: eksekusi otomatis hanya untuk tier Plus", apperror.ErrForbidden)
	}

	riskPercent, err := s.validateRiskForTier(ctx, tier, req.RiskPercent)
	if err != nil {
		return nil, err
	}

	riskMode, lotSize, err := s.resolveRiskSettings(req.RiskMode, req.LotSize, riskPercent, "")
	if err != nil {
		return nil, err
	}

	// Ambil akun yang sudah ada untuk mendeteksi duplikat
	existingAccounts, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	existingMap := make(map[string]*model.BrokerAccount)
	for i := range existingAccounts {
		if existingAccounts[i].BrokerType == "ctrader" {
			existingMap[existingAccounts[i].ExternalAccountID] = &existingAccounts[i]
		}
	}

	var connectedResponses []dto.AccountResponse

	for _, acc := range accounts {
		extID := strconv.FormatInt(acc.AccountID, 10)
		loginStr := strconv.FormatInt(acc.TraderLogin, 10)

		typeStr := "Live"
		if !acc.Live {
			typeStr = "Demo"
		}
		label := fmt.Sprintf("cTrader %s (%s)", loginStr, typeStr)
		if acc.BrokerTitle != "" {
			label = fmt.Sprintf("%s %s (%s)", acc.BrokerTitle, loginStr, typeStr)
		}

		if existing, ok := existingMap[extID]; ok {
			creds := broker.Credentials{
				Platform:  "ctrader",
				AccountID: existing.ID,
				Login:     loginStr,
				Token:     tokenResp.AccessToken,
				Extra: map[string]string{
					"ctid_trader_account_id": extID,
					"refresh_token":          tokenResp.RefreshToken,
					"live":                   strconv.FormatBool(acc.Live),
				},
			}
			plaintext, encErr := encodeCredentials(creds)
			if encErr != nil {
				return nil, encErr
			}
			envelope, sealErr := s.sealer.Seal(plaintext, []byte(existing.ID))
			if sealErr != nil {
				return nil, sealErr
			}
			_ = s.creds.Upsert(ctx, &model.Credential{
				BrokerAccountID: existing.ID,
				UserID:          userID,
				UserKeyWrapped:  envelope.UserKeyWrapped,
				UserKeyNonce:    envelope.UserKeyNonce,
				Ciphertext:      envelope.Ciphertext,
				Nonce:           envelope.Nonce,
				KeyVersion:      envelope.KeyVersion,
				RotatedAt:       s.clock(),
			})
			existing.IsActive = true
			existing.LastError = ""
			_ = s.accounts.Update(ctx, existing)
			connectedResponses = append(connectedResponses, s.toResponse(ctx, existing, model.BrokerAccountState{}))
			continue
		}

		if int(active) >= limit {
			break
		}

		// Akumulasi risiko dicek per akun yang akan ditambahkan, sehingga
		// menghubungkan banyak akun sekaligus tidak melewati batas total.
		currentAccounts, listErr := s.accounts.ListByUser(ctx, userID)
		if listErr != nil {
			return nil, listErr
		}
		if totalErr := s.validateTotalRisk(ctx, tier, currentAccounts, riskPercent, ""); totalErr != nil {
			return nil, totalErr
		}

		newAcc := &model.BrokerAccount{
			UserID:            userID,
			BrokerType:        "ctrader",
			Label:             label,
			ExternalAccountID: extID,
			Mode:              mode,
			RiskMode:          riskMode,
			RiskPercent:       riskPercent,
			LotSize:           lotSize,
			IsActive:          true,
			IsDryRun:          !acc.Live,
		}

		if err := s.accounts.Create(ctx, newAcc); err != nil {
			return nil, err
		}

		creds := broker.Credentials{
			Platform:  "ctrader",
			AccountID: newAcc.ID,
			Login:     loginStr,
			Token:     tokenResp.AccessToken,
			Extra: map[string]string{
				"ctid_trader_account_id": extID,
				"refresh_token":          tokenResp.RefreshToken,
				"live":                   strconv.FormatBool(acc.Live),
			},
		}
		plaintext, encErr := encodeCredentials(creds)
		if encErr != nil {
			newAcc.IsActive = false
			newAcc.LastError = "kredensial gagal disandikan"
			_ = s.accounts.Update(ctx, newAcc)
			return nil, encErr
		}
		envelope, sealErr := s.sealer.Seal(plaintext, []byte(newAcc.ID))
		if sealErr != nil {
			newAcc.IsActive = false
			newAcc.LastError = "kredensial gagal dienkripsi"
			_ = s.accounts.Update(ctx, newAcc)
			return nil, sealErr
		}

		if err := s.creds.Upsert(ctx, &model.Credential{
			BrokerAccountID: newAcc.ID,
			UserID:          userID,
			UserKeyWrapped:  envelope.UserKeyWrapped,
			UserKeyNonce:    envelope.UserKeyNonce,
			Ciphertext:      envelope.Ciphertext,
			Nonce:           envelope.Nonce,
			KeyVersion:      envelope.KeyVersion,
			RotatedAt:       s.clock(),
		}); err != nil {
			newAcc.IsActive = false
			newAcc.LastError = "kredensial gagal disimpan"
			_ = s.accounts.Update(ctx, newAcc)
			return nil, err
		}

		active++
		s.recordAudit(ctx, userID, newAcc, "account.create", "")
		connectedResponses = append(connectedResponses, s.toResponse(ctx, newAcc, model.BrokerAccountState{}))
	}

	if len(connectedResponses) > 0 {
		// Akun cTrader juga butuh saklar untuk pair yang sudah dipantau.
		s.seedSaklarUntukLangganan(ctx, userID)
	}

	return &dto.CTraderCallbackResponse{
		TotalConnected: len(connectedResponses),
		Accounts:       connectedResponses,
	}, nil
}

// catalogSnapshot mengambil katalog pair aktif. Bila layanan katalog tidak
// terpasang (pemasangan lama atau pengujian tanpa katalog), hasilnya kosong dan
// validasi simbol jatuh ke perilaku lama.
func (s *accountService) catalogSnapshot(ctx context.Context) (dto.PairCatalogResponse, error) {
	if s.catalog == nil {
		return dto.PairCatalogResponse{}, nil
	}
	return s.catalog.Catalog(ctx)
}

// validateSymbolAgainstCatalog memastikan simbol benar-benar ada di katalog dan
// aktif. Simbol yang tidak dikenal ditolak agar user tidak memilih pair asal.
func (s *accountService) validateSymbolAgainstCatalog(ctx context.Context, raw string) (string, error) {
	symbol := normalizeSymbol(raw)
	if symbol == "" {
		return "", fmt.Errorf("%w: simbol tidak sah", apperror.ErrBadRequest)
	}
	if s.catalog == nil {
		return symbol, nil
	}
	canonical, err := s.catalog.ResolveSymbol(ctx, symbol)
	if err != nil {
		return "", err
	}
	return canonical, nil
}

// AccountPairs menampilkan checklist pair satu akun beserta katalog aktif.
func (s *accountService) AccountPairs(ctx context.Context, userID, accountID string) (*dto.AccountPairSelectionResponse, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, apperror.ErrNotFound
	}

	subs, err := s.subs.ListActiveByAccount(ctx, userID, accountID)
	if err != nil {
		return nil, err
	}
	selected := make([]string, 0, len(subs))
	seen := make(map[string]bool, len(subs))
	for _, sub := range subs {
		if seen[sub.Symbol] {
			continue
		}
		seen[sub.Symbol] = true
		selected = append(selected, sub.Symbol)
	}

	catalog, err := s.catalogSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	return &dto.AccountPairSelectionResponse{
		AccountID:   account.ID,
		RiskPercent: account.RiskPercent,
		Selected:    selected,
		Catalog:     catalog,
	}, nil
}

// UpdateAccountPairs mengganti seluruh pilihan pair satu akun.
//
// Seluruh simbol divalidasi lebih dulu terhadap katalog; bila satu saja tidak
// sah, tidak ada perubahan yang disimpan (all-or-nothing).
func (s *accountService) UpdateAccountPairs(ctx context.Context, userID, accountID string, req dto.UpdateAccountPairsRequest) (*dto.AccountPairSelectionResponse, error) {
	account, err := s.accounts.FindByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, apperror.ErrNotFound
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	tier := effectiveTier(user, s.clock())
	limit := s.tierSymbolLimit(ctx, tier)

	canonical := make([]string, 0, len(req.Symbols))
	seen := make(map[string]bool, len(req.Symbols))
	for _, raw := range req.Symbols {
		symbol, err := s.validateSymbolAgainstCatalog(ctx, raw)
		if err != nil {
			return nil, err
		}
		if seen[symbol] {
			continue
		}
		seen[symbol] = true
		canonical = append(canonical, symbol)
	}

	if len(canonical) > limit {
		return nil, fmt.Errorf("%w: tier %s maksimal %d simbol per akun", apperror.ErrForbidden, tier, limit)
	}

	if err := s.subs.ReplaceAccountPairs(ctx, userID, accountID, canonical); err != nil {
		return nil, err
	}

	s.recordStrategyAudit(ctx, userID, "", "subscription.account.update", accountID)
	return s.AccountPairs(ctx, userID, accountID)
}
