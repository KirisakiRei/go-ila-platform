package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/pkg/cryptox"
)

// --- Tiruan penyimpanan -----------------------------------------------------

type fakeAccountRepo struct {
	accounts map[string]*model.BrokerAccount
	seq      int
}

func (f *fakeAccountRepo) Create(_ context.Context, account *model.BrokerAccount) error {
	f.seq++
	account.ID = "acc-" + string(rune('0'+f.seq))
	f.accounts[account.ID] = account
	return nil
}
func (f *fakeAccountRepo) FindByID(_ context.Context, id string) (*model.BrokerAccount, error) {
	a, ok := f.accounts[id]
	if !ok {
		return nil, errors.New("tidak ditemukan")
	}
	copy := *a
	return &copy, nil
}
func (f *fakeAccountRepo) ListByUser(_ context.Context, userID string) ([]model.BrokerAccount, error) {
	var out []model.BrokerAccount
	for _, a := range f.accounts {
		if a.UserID == userID {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (f *fakeAccountRepo) CountActiveByUser(_ context.Context, userID string) (int64, error) {
	var n int64
	for _, a := range f.accounts {
		if a.UserID == userID && a.IsActive {
			n++
		}
	}
	return n, nil
}
func (f *fakeAccountRepo) Update(_ context.Context, account *model.BrokerAccount) error {
	f.accounts[account.ID] = account
	return nil
}

type fakeCredRepo struct {
	creds map[string]*model.Credential
}

func (f *fakeCredRepo) Upsert(_ context.Context, c *model.Credential) error {
	f.creds[c.BrokerAccountID] = c
	return nil
}
func (f *fakeCredRepo) FindByAccount(_ context.Context, id string) (*model.Credential, error) {
	c, ok := f.creds[id]
	if !ok {
		return nil, errors.New("tidak ditemukan")
	}
	return c, nil
}
func (f *fakeCredRepo) DeleteByAccount(_ context.Context, id string) error {
	delete(f.creds, id)
	return nil
}

type fakeSubRepo struct {
	subs []*model.Subscription
}

func (f *fakeSubRepo) Create(_ context.Context, sub *model.Subscription) error {
	sub.ID = "sub-" + sub.Symbol
	f.subs = append(f.subs, sub)
	return nil
}
func (f *fakeSubRepo) ListActiveByUser(_ context.Context, userID string) ([]model.Subscription, error) {
	var out []model.Subscription
	for _, s := range f.subs {
		if s.UserID == userID && s.IsActive {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (f *fakeSubRepo) ListByUser(_ context.Context, userID string) ([]model.Subscription, error) {
	var out []model.Subscription
	for _, s := range f.subs {
		if s.UserID == userID {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (f *fakeSubRepo) Deactivate(_ context.Context, id, userID string) error {
	for _, s := range f.subs {
		if s.ID == id && s.UserID == userID {
			s.IsActive = false
			return nil
		}
	}
	return errors.New("tidak ditemukan")
}

type fakeStrategies struct{ ila3 string }

func (f *fakeStrategies) StrategyStatus(strategy string) string {
	if strategy == "ila-3.0" {
		return f.ila3
	}
	return "active"
}
func (f *fakeStrategies) StrategySelectable(strategy string) bool {
	if strategy == "ila-3.0" {
		return f.ila3 == "beta" || f.ila3 == "general"
	}
	return true
}

// --- Perkakas ---------------------------------------------------------------

// userIDTest adalah identitas user tetap untuk seluruh tes di berkas ini.
const userIDTest = "user-tes-1"

// newTestUser membuat user dengan ID tetap agar seluruh tes memakai identitas sama.
func newTestUser(tier string) *model.User {
	verified := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &model.User{
		Base:   model.Base{ID: userIDTest},
		Email:  "u1@contoh.id",
		Tier:   tier,
		Status: "active",
		// Email terverifikasi agar fokus tes ini pada aturan tier akun.
		EmailVerifiedAt: &verified,
	}
}

func testKeyring(t *testing.T) *cryptox.Keyring {
	t.Helper()
	// 32 byte base64.
	key := "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDA="
	ring, err := cryptox.NewKeyring(1, key, nil)
	if err != nil {
		t.Fatalf("keyring: %v", err)
	}
	return ring
}

func newTestAccountService(t *testing.T, user *model.User, strat *fakeStrategies) (AccountService, *fakeAccountRepo, *fakeCredRepo) {
	t.Helper()

	users := newFakeUserRepo()
	users.users[user.Email] = user
	accounts := &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}}
	creds := &fakeCredRepo{creds: map[string]*model.Credential{}}
	subs := &fakeSubRepo{}

	svc := NewAccountService(AccountServiceDeps{
		Users:         users,
		Accounts:      accounts,
		Credentials:   creds,
		Subscriptions: subs,
		Audit:         &fakeAuditRepo{},
		Sealer:        testKeyring(t),
		Strategies:    strat,
		// Sidik jari token wajib agar pemeriksaan duplikat berjalan.
		FingerprintSecret: fingerprintTestSecret,
	})
	return svc, accounts, creds
}

// fingerprintTestSecret dipakai tes untuk mengenali token kembar.
const fingerprintTestSecret = "rahasia-sidik-jari-uji"

const sampleToken = "eyJhbGciOiJIUzI1NiJ9.sample.token.value.for.testing"

// --- Tes --------------------------------------------------------------------

func TestFreeTierTidakBolehPunyaAkun(t *testing.T) {
	user := newTestUser("free")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	_, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	})
	if err == nil {
		t.Fatal("tier Free seharusnya ditolak")
	}
}

func TestBasicMaksimalSatuAkun(t *testing.T) {
	user := newTestUser("basic")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	if _, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	}); err != nil {
		t.Fatalf("akun pertama harus berhasil: %v", err)
	}

	_, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	})
	if err == nil {
		t.Fatal("akun kedua pada tier Basic seharusnya ditolak")
	}
}

func TestPlusMaksimalDuaAkun(t *testing.T) {
	user := newTestUser("plus")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "general"})

	for i := 0; i < 2; i++ {
		// Dua akun nyata memakai token berbeda: satu token = satu akun.
		if _, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
			BrokerType: "matchtrader", Token: sampleToken + "-akun-" + string(rune('1'+i)),
		}); err != nil {
			t.Fatalf("akun ke-%d gagal: %v", i+1, err)
		}
	}

	if _, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken + "-akun-3",
	}); err == nil {
		t.Fatal("akun ketiga seharusnya ditolak")
	}
}

// TestKredensialTerenkripsi memastikan token tidak pernah tersimpan sebagai
// plaintext di blob yang ditulis ke database.
func TestKredensialTerenkripsi(t *testing.T) {
	user := newTestUser("basic")
	svc, _, creds := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	if _, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	var blob *model.Credential
	for _, c := range creds.creds {
		blob = c
	}
	if blob == nil {
		t.Fatal("kredensial tidak tersimpan")
	}

	if containsBytes(blob.Ciphertext, []byte(sampleToken)) {
		t.Fatal("token muncul sebagai plaintext di ciphertext")
	}

	ring := testKeyring(t)
	plain, err := ring.Open(cryptox.Envelope{
		UserKeyWrapped: blob.UserKeyWrapped,
		UserKeyNonce:   blob.UserKeyNonce,
		Ciphertext:     blob.Ciphertext,
		Nonce:          blob.Nonce,
		KeyVersion:     blob.KeyVersion,
	}, []byte(blob.BrokerAccountID))
	if err != nil {
		t.Fatalf("membuka kredensial: %v", err)
	}
	if !containsBytes(plain, []byte(sampleToken)) {
		t.Fatal("token tidak kembali setelah dekripsi")
	}
}

// TestKredensialTerikatAkun memastikan blob tidak bisa dipakai akun lain.
func TestKredensialTerikatAkun(t *testing.T) {
	user := newTestUser("basic")
	svc, _, creds := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	if _, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	var blob *model.Credential
	for _, c := range creds.creds {
		blob = c
	}

	ring := testKeyring(t)
	_, err := ring.Open(cryptox.Envelope{
		UserKeyWrapped: blob.UserKeyWrapped,
		UserKeyNonce:   blob.UserKeyNonce,
		Ciphertext:     blob.Ciphertext,
		Nonce:          blob.Nonce,
		KeyVersion:     blob.KeyVersion,
	}, []byte("akun-lain"))
	if err == nil {
		t.Fatal("blob seharusnya gagal dibuka dengan akun lain")
	}
}

func TestModeAutoHanyaPlus(t *testing.T) {
	user := newTestUser("basic")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	_, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken, Mode: "auto",
	})
	if err == nil {
		t.Fatal("mode auto pada tier Basic seharusnya ditolak")
	}
}

func TestRisikoDiLuarPagarDitolak(t *testing.T) {
	user := newTestUser("plus")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "general"})

	for _, risk := range []float64{0.05, 1.5} {
		_, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
			BrokerType: "matchtrader", Token: sampleToken, RiskPercent: risk,
		})
		if err == nil {
			t.Fatalf("risiko %.2f seharusnya ditolak", risk)
		}
	}
}

func TestRisikoDefaultSetengahPersen(t *testing.T) {
	user := newTestUser("basic")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	result, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if result.RiskPercent != 0.5 {
		t.Errorf("risk default=%v, want 0.5", result.RiskPercent)
	}
}

func TestTokenKosongDitolak(t *testing.T) {
	user := newTestUser("basic")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	if _, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: "  ",
	}); err == nil {
		t.Fatal("token kosong seharusnya ditolak")
	}
}

func TestSubscribeILA3PerluPlus(t *testing.T) {
	user := newTestUser("basic")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "general"})

	_, err := svc.Subscribe(context.Background(), userIDTest, dto.CreateSubscriptionRequest{
		Symbol: "XAUUSD", Strategy: "ila-3.0",
	})
	if err == nil {
		t.Fatal("ILA 3.0 pada tier Basic seharusnya ditolak")
	}
}

func TestSubscribeILA3SaatComingSoon(t *testing.T) {
	user := newTestUser("plus")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "coming_soon"})

	_, err := svc.Subscribe(context.Background(), userIDTest, dto.CreateSubscriptionRequest{
		Symbol: "XAUUSD", Strategy: "ila-3.0",
	})
	if err == nil {
		t.Fatal("ILA 3.0 yang belum dirilis seharusnya ditolak")
	}
}

func TestSubscribeILA2Idempoten(t *testing.T) {
	user := newTestUser("free")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	first, err := svc.Subscribe(context.Background(), userIDTest, dto.CreateSubscriptionRequest{Symbol: "xau/usd"})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if first.Symbol != "XAUUSD" {
		t.Errorf("simbol=%q, want XAUUSD", first.Symbol)
	}

	second, err := svc.Subscribe(context.Background(), userIDTest, dto.CreateSubscriptionRequest{Symbol: "XAUUSD"})
	if err != nil {
		t.Fatalf("subscribe ulang: %v", err)
	}
	if second.ID != first.ID {
		t.Error("melanggan ulang seharusnya mengembalikan baris yang sama")
	}
}

func TestBatasSubscription(t *testing.T) {
	user := newTestUser("plus")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	symbols := []string{"XAUUSD", "EURUSD", "GBPUSD", "USDJPY", "AUDUSD", "USDCAD", "USDCHF", "NZDUSD", "USTEC"}
	for i, sym := range symbols {
		_, err := svc.Subscribe(context.Background(), userIDTest, dto.CreateSubscriptionRequest{Symbol: sym})
		if i < MaxSubscriptionsPerUser && err != nil {
			t.Fatalf("simbol ke-%d gagal: %v", i+1, err)
		}
		if i >= MaxSubscriptionsPerUser && err == nil {
			t.Fatalf("simbol ke-%d seharusnya ditolak (batas %d)", i+1, MaxSubscriptionsPerUser)
		}
	}
}

func TestStrategiesMenampilkanStatus(t *testing.T) {
	user := newTestUser("basic")
	svc, _, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "coming_soon"})

	list, err := svc.Strategies(context.Background(), userIDTest)
	if err != nil {
		t.Fatalf("strategies: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("jumlah strategi=%d, want 2", len(list))
	}

	var ila3 *dto.StrategyResponse
	for i := range list {
		if list[i].Version == "ila-3.0" {
			ila3 = &list[i]
		}
	}
	if ila3 == nil {
		t.Fatal("ILA 3.0 tidak ada dalam daftar")
	}
	if ila3.Status != "coming_soon" || ila3.Available || ila3.Allowed {
		t.Errorf("ILA 3.0 untuk Basic seharusnya coming_soon & tidak tersedia: %+v", *ila3)
	}
}

func TestHapusAkunMenghapusKredensial(t *testing.T) {
	user := newTestUser("basic")
	svc, _, creds := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	created, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(creds.creds) != 1 {
		t.Fatal("kredensial seharusnya tersimpan")
	}

	if err := svc.Delete(context.Background(), userIDTest, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(creds.creds) != 0 {
		t.Fatal("kredensial seharusnya terhapus permanen")
	}
}

func TestTidakBisaMengaksesAkunUserLain(t *testing.T) {
	user := newTestUser("basic")
	svc, accounts, _ := newTestAccountService(t, user, &fakeStrategies{ila3: "off"})

	created, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = accounts

	if err := svc.Delete(context.Background(), "penyusup", created.ID); err == nil {
		t.Fatal("user lain seharusnya tidak bisa menghapus akun ini")
	}
}

func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return true
		}
	}
	return false
}
