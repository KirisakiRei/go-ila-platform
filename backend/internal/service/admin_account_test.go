package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
	"ila/backend/internal/repository"
	"ila/backend/pkg/apperror"
)

// fakeAdminAccountStore menyimpan akun broker di memori untuk pengujian.
type fakeAdminAccountStore struct {
	rows     map[string]*repository.AdminAccountRow
	balances map[string]repository.AccountBalanceRow
	labels   map[string]string
	deleted  map[string]bool
}

func newFakeAdminAccountStore() *fakeAdminAccountStore {
	return &fakeAdminAccountStore{
		rows:     map[string]*repository.AdminAccountRow{},
		balances: map[string]repository.AccountBalanceRow{},
		labels:   map[string]string{},
		deleted:  map[string]bool{},
	}
}

func (f *fakeAdminAccountStore) put(id, userID, label string) {
	f.rows[id] = &repository.AdminAccountRow{ID: id, UserID: userID, Label: label, BrokerType: "matchtrader"}
}

func (f *fakeAdminAccountStore) ListForAdmin(_ context.Context, userID string) ([]repository.AdminAccountRow, error) {
	out := make([]repository.AdminAccountRow, 0, len(f.rows))
	for id, row := range f.rows {
		if row.UserID != userID || f.deleted[id] {
			continue
		}
		out = append(out, *row)
	}
	return out, nil
}

func (f *fakeAdminAccountStore) FindForAdmin(_ context.Context, accountID string) (*repository.AdminAccountRow, error) {
	row, ok := f.rows[accountID]
	if !ok || f.deleted[accountID] {
		return nil, apperror.ErrNotFound
	}
	copy := *row
	if label, ok := f.labels[accountID]; ok {
		copy.Label = label
	}
	return &copy, nil
}

func (f *fakeAdminAccountStore) BalancesForUser(_ context.Context, userID string) (map[string]repository.AccountBalanceRow, error) {
	out := map[string]repository.AccountBalanceRow{}
	for _, row := range f.rows {
		if row.UserID != userID {
			continue
		}
		if balance, ok := f.balances[row.ID]; ok {
			out[row.ID] = balance
		}
	}
	return out, nil
}

func (f *fakeAdminAccountStore) SetFrozen(context.Context, string, bool, *time.Time, string) error {
	return nil
}

func (f *fakeAdminAccountStore) SetLabel(_ context.Context, accountID, label string) error {
	f.labels[accountID] = label
	return nil
}

func (f *fakeAdminAccountStore) DeleteAccount(_ context.Context, accountID string) error {
	if _, ok := f.rows[accountID]; !ok {
		return apperror.ErrNotFound
	}
	f.deleted[accountID] = true
	return nil
}

func (f *fakeAdminAccountStore) DeleteAllForUser(context.Context, string) error { return nil }

// newAccountFixture menyiapkan layanan pelanggan dengan akun broker.
func newAccountFixture(t *testing.T) (AdminUserService, *fakeAdminAccountStore, *fakeAdminAudit) {
	t.Helper()
	store := newFakeUserStore()
	store.put("u1", "u1@contoh.id", "active", "plus", "user")

	accounts := newFakeAdminAccountStore()
	accounts.put("acc-1", "u1", "Akun Utama")
	rec := &fakeAdminAudit{}

	svc := NewAdminUserService(store, AdminUserExtras{Accounts: accounts}, rec, nil)
	return svc, accounts, rec
}

func TestAdminBalances_MemuatTelemetriSaldo(t *testing.T) {
	svc, accounts, rec := newAccountFixture(t)
	syncedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	accounts.balances["acc-1"] = repository.AccountBalanceRow{
		BrokerAccountID: "acc-1", Balance: 1500, Equity: 1510, Currency: "USD", LastSyncedAt: &syncedAt,
	}

	items, err := svc.Balances(context.Background(), AdminActor{ID: "admin-1"}, "u1")
	if err != nil {
		t.Fatalf("Balances: %v", err)
	}
	if len(items) != 1 || items[0].Balance != 1500 || items[0].Currency != "USD" {
		t.Fatalf("saldo = %+v", items)
	}
	// Membuka saldo adalah baca sensitif: wajib tercatat.
	if len(rec.entries) == 0 {
		t.Fatal("pembacaan saldo harus tercatat audit")
	}
}

func TestAdminBalances_PelangganTidakAda(t *testing.T) {
	svc, _, _ := newAccountFixture(t)
	if _, err := svc.Balances(context.Background(), AdminActor{ID: "admin-1"}, "tidak-ada"); err == nil {
		t.Fatal("pelanggan tak dikenal harus gagal")
	}
}

func TestAdminUpdateAccountLabel_HanyaLabel(t *testing.T) {
	svc, accounts, rec := newAccountFixture(t)

	if err := svc.UpdateAccountLabel(context.Background(), AdminActor{ID: "admin-1"}, "acc-1",
		dto.AdminAccountLabelRequest{Label: "  Akun Baru  "}); err != nil {
		t.Fatalf("UpdateAccountLabel: %v", err)
	}
	if got := accounts.labels["acc-1"]; got != "Akun Baru" {
		t.Fatalf("label tersimpan = %q, ingin \"Akun Baru\"", got)
	}
	if len(rec.entries) == 0 || rec.entries[0].Action != "account.update" {
		t.Fatalf("audit = %+v, ingin account.update", rec.entries)
	}
}

func TestAdminUpdateAccountLabel_MenolakLabelTerlaluPendek(t *testing.T) {
	svc, _, _ := newAccountFixture(t)
	err := svc.UpdateAccountLabel(context.Background(), AdminActor{ID: "admin-1"}, "acc-1",
		dto.AdminAccountLabelRequest{Label: "a"})
	if err == nil {
		t.Fatal("label satu karakter harus ditolak")
	}
}

func TestAdminDeleteAccount_WajibAlasan(t *testing.T) {
	svc, accounts, _ := newAccountFixture(t)
	ctx := context.Background()

	err := svc.DeleteAccount(ctx, AdminActor{ID: "admin-1"}, "acc-1", dto.UserActionRequest{Reason: "pendek"})
	if !errors.Is(err, apperror.ErrBadRequest) {
		t.Fatalf("alasan terlalu pendek harus bad request, dapat %v", err)
	}
	if accounts.deleted["acc-1"] {
		t.Fatal("akun tidak boleh terhapus saat alasan ditolak")
	}

	err = svc.DeleteAccount(ctx, AdminActor{ID: "admin-1"}, "acc-1", dto.UserActionRequest{
		Reason: "pelanggan meminta akun dihapus permanen", Category: model.ReasonOther,
	})
	if err != nil {
		t.Fatalf("hapus akun: %v", err)
	}
	if !accounts.deleted["acc-1"] {
		t.Fatal("akun seharusnya terhapus")
	}
}
