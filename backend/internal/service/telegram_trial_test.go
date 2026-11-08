package service

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
)

// --- Tiruan ----------------------------------------------------------------

type fakeOnboardingStore struct {
	rows map[string]*model.OnboardingToken
}

func newFakeOnboardingStore() *fakeOnboardingStore {
	return &fakeOnboardingStore{rows: map[string]*model.OnboardingToken{}}
}

func (f *fakeOnboardingStore) Create(_ context.Context, token *model.OnboardingToken) error {
	token.ID = "tok-1"
	f.rows[token.TokenHash] = token
	return nil
}
func (f *fakeOnboardingStore) FindByHash(_ context.Context, hash string) (*model.OnboardingToken, error) {
	row, ok := f.rows[hash]
	if !ok {
		return nil, errTestNotFound
	}
	return row, nil
}
func (f *fakeOnboardingStore) MarkUsed(_ context.Context, id string, at time.Time) error {
	for _, row := range f.rows {
		if row.ID == id {
			row.UsedAt = &at
		}
	}
	return nil
}

var errTestNotFound = errString("tidak ditemukan")

type errString string

func (e errString) Error() string { return string(e) }

// --- Tes Telegram -----------------------------------------------------------

func TestTelegramCreateLinkMenyimpanHash(t *testing.T) {
	users := newFakeUserRepo()
	keyring := newTestUser("basic")
	users.users[keyring.Email] = keyring

	tokens := newFakeOnboardingStore()
	svc := NewTelegramService(users, tokens, &fakeAuditRepo{})

	link, err := svc.CreateLink(context.Background(), userIDTest, "@ILABot")
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	if link.URL == "" {
		t.Fatal("tautan kosong")
	}
	if len(tokens.rows) != 1 {
		t.Fatalf("token tersimpan = %d, ingin 1", len(tokens.rows))
	}

	// Yang disimpan hanya hash, bukan token mentah.
	for hash := range tokens.rows {
		if len(hash) != 64 {
			t.Fatalf("hash tidak wajar: %q", hash)
		}
		if containsText(link.URL, hash) {
			t.Fatal("hash tidak boleh muncul di tautan")
		}
	}
}

func TestTelegramStartMenghubungkanChat(t *testing.T) {
	users := newFakeUserRepo()
	user := newTestUser("basic")
	users.users[user.Email] = user

	tokens := newFakeOnboardingStore()
	svc := NewTelegramService(users, tokens, &fakeAuditRepo{})

	link, err := svc.CreateLink(context.Background(), userIDTest, "@ILABot")
	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	raw := lastQueryValue(link.URL)
	result, err := svc.HandleStart(context.Background(), raw, "12345")
	if err != nil {
		t.Fatalf("handle start: %v", err)
	}
	if result.ChatID != "12345" {
		t.Fatalf("chat id=%q", result.ChatID)
	}

	stored := users.users[user.Email]
	if stored.TelegramChatID == nil || *stored.TelegramChatID != "12345" {
		t.Fatal("chat id tidak tersimpan di user")
	}
}

func TestTelegramTokenSekaliPakai(t *testing.T) {
	users := newFakeUserRepo()
	user := newTestUser("basic")
	users.users[user.Email] = user

	tokens := newFakeOnboardingStore()
	svc := NewTelegramService(users, tokens, &fakeAuditRepo{})

	link, _ := svc.CreateLink(context.Background(), userIDTest, "@ILABot")
	raw := lastQueryValue(link.URL)

	if _, err := svc.HandleStart(context.Background(), raw, "12345"); err != nil {
		t.Fatalf("pemakaian pertama: %v", err)
	}
	if _, err := svc.HandleStart(context.Background(), raw, "99999"); err == nil {
		t.Fatal("token yang sudah dipakai seharusnya ditolak")
	}
}

func TestTelegramTokenKedaluwarsa(t *testing.T) {
	users := newFakeUserRepo()
	user := newTestUser("basic")
	users.users[user.Email] = user

	tokens := newFakeOnboardingStore()
	svc := NewTelegramService(users, tokens, &fakeAuditRepo{})
	// Waktu dimajukan melewati masa berlaku.
	svc.(*telegramService).clock = func() time.Time { return time.Now().UTC().Add(10 * time.Minute) }

	link, _ := svc.CreateLink(context.Background(), userIDTest, "@ILABot")
	raw := lastQueryValue(link.URL)

	svc.(*telegramService).clock = func() time.Time { return time.Now().UTC().Add(20 * time.Minute) }
	if _, err := svc.HandleStart(context.Background(), raw, "12345"); err == nil {
		t.Fatal("token kedaluwarsa seharusnya ditolak")
	}
}

func TestTelegramDisconnect(t *testing.T) {
	users := newFakeUserRepo()
	user := newTestUser("basic")
	chatID := "12345"
	user.TelegramChatID = &chatID
	users.users[user.Email] = user

	svc := NewTelegramService(users, newFakeOnboardingStore(), &fakeAuditRepo{})

	if err := svc.Disconnect(context.Background(), userIDTest); err != nil {
		t.Fatalf("disconnect: %v", err)
	}
	if users.users[user.Email].TelegramChatID != nil {
		t.Fatal("chat id seharusnya terhapus")
	}
}

// --- Tes Trial --------------------------------------------------------------

func TestTrialSekaliPerAkun(t *testing.T) {
	users := newFakeUserRepo()
	user := newTestUser("free")
	users.users[user.Email] = user

	svc := NewTrialService(users, &fakeAuditRepo{})

	result, err := svc.Start(context.Background(), userIDTest)
	if err != nil {
		t.Fatalf("start trial: %v", err)
	}
	if !result.Active || !result.DryRunOnly {
		t.Fatalf("trial harus aktif dan hanya simulasi: %+v", *result)
	}
	if result.DaysRemaining != 7 {
		t.Errorf("sisa hari = %d, ingin 7", result.DaysRemaining)
	}

	if _, err := svc.Start(context.Background(), userIDTest); err == nil {
		t.Fatal("trial kedua seharusnya ditolak")
	}
}

func TestTrialSetaraBasic(t *testing.T) {
	users := newFakeUserRepo()
	user := newTestUser("free")
	users.users[user.Email] = user

	svc := NewTrialService(users, &fakeAuditRepo{})
	if _, err := svc.Start(context.Background(), userIDTest); err != nil {
		t.Fatalf("start trial: %v", err)
	}

	updated := users.users[user.Email]
	if tier := effectiveTier(updated, time.Now().UTC()); tier != "basic" {
		t.Fatalf("tier saat trial = %q, ingin basic", tier)
	}
	if result := toTrialResponse(updated, time.Now().UTC()); result.AccountLimit != 1 {
		t.Fatalf("batas akun trial = %d, ingin 1", result.AccountLimit)
	}
}

// --- Bantuan ----------------------------------------------------------------

func containsText(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// lastQueryValue mengambil nilai ?start=<token> dari tautan t.me.
func lastQueryValue(link string) string {
	idx := indexOf(link, "start=")
	if idx < 0 {
		return ""
	}
	return link[idx+len("start="):]
}
