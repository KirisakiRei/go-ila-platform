package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/cryptox"
)

// Batas risiko per trade yang berlaku untuk semua tier (plans/14 §6).
const (
	MinRiskPercent = 0.10
	MaxRiskPercent = 1.00
)

// Batas jumlah simbol yang dipantau per user, menjaga beban engine.
const MaxSubscriptionsPerUser = 8

// Sealer mengenkripsi kredensial sebelum disimpan.
// Layer service hanya melihat antarmuka ini; kunci tidak pernah menyentuhnya.
type Sealer interface {
	Seal(plaintext, associatedData []byte) (cryptox.Envelope, error)
	ActiveVersion() int
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
	secret string
	clock  func() time.Time
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

	riskPercent, err := clampRisk(req.RiskPercent)
	if err != nil {
		return nil, err
	}

	account := &model.BrokerAccount{
		UserID:      userID,
		BrokerType:  brokerType,
		Label:       strings.TrimSpace(req.Label),
		Mode:        mode,
		RiskMode:    "percent",
		RiskPercent: riskPercent,
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

	return ptr(s.toResponse(ctx, account, model.BrokerAccountState{})), nil
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
		riskPercent, riskErr := clampRisk(*req.RiskPercent)
		if riskErr != nil {
			return nil, riskErr
		}
		account.RiskPercent = riskPercent
	}
	if req.IsActive != nil {
		account.IsActive = *req.IsActive
	}
	if req.IsDryRun != nil {
		account.IsDryRun = *req.IsDryRun
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

	strategy := req.Strategy
	if strategy == "" {
		strategy = "ila-2.0"
	}
	if !knownStrategy(strategy) {
		return nil, fmt.Errorf("%w: strategi tidak dikenal", apperror.ErrBadRequest)
	}
	if strategy == "ila-3.0" {
		if tier != "plus" {
			return nil, fmt.Errorf("%w: ILA 3.0 tersedia untuk tier Plus", apperror.ErrForbidden)
		}
		if !s.strategies.StrategySelectable(strategy) {
			return nil, fmt.Errorf("%w: ILA 3.0 belum tersedia", apperror.ErrForbidden)
		}
	}

	existing, findErr := s.subs.ListByUser(ctx, userID)
	if findErr != nil {
		return nil, findErr
	}
	activeCount := 0
	for _, row := range existing {
		if row.IsActive && row.Symbol == symbol && row.Strategy == strategy {
			return ptr(toSubscriptionResponse(row)), nil
		}
		if row.IsActive {
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
		Strategy: strategy,
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

	s.recordStrategyAudit(ctx, userID, strategy, "subscription.enable", symbol)

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

	active, err := s.subs.ListActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	selected := make(map[string]bool, len(active))
	for _, row := range active {
		selected[row.Strategy] = true
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
		RiskPercent:       account.RiskPercent,
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
		return fmt.Errorf("%w: cTrader belum tersedia", apperror.ErrBadRequest)
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
		Strategy: sub.Strategy,
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
