package service

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"ila/backend/internal/dto"
	"ila/backend/internal/model"
)

// revokerSpy mencatat permintaan pemutusan sesi broker yang dikirim ke Runner.
type revokerSpy struct {
	ids []string
}

func (r *revokerSpy) RevokeAccount(_ context.Context, accountID string) error {
	r.ids = append(r.ids, accountID)
	return nil
}

// Token baru harus menulis ulang blob terenkripsi dan memutus sesi lama di
// Runner, agar login berikutnya memakai token terbaru (plans/12 §3.4).
func TestRotasiTokenMemutusSesiLama(t *testing.T) {
	user := newTestUser("basic")

	users := newFakeUserRepo()
	users.users[user.Email] = user
	accounts := &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}}
	creds := &fakeCredRepo{creds: map[string]*model.Credential{}}
	spy := &revokerSpy{}

	svc := NewAccountService(AccountServiceDeps{
		Users:             users,
		Accounts:          accounts,
		Credentials:       creds,
		Subscriptions:     &fakeSubRepo{},
		Audit:             &fakeAuditRepo{},
		Sealer:            testKeyring(t),
		Strategies:        &fakeStrategies{ila3: "off"},
		Revoker:           spy,
		FingerprintSecret: fingerprintTestSecret,
	})

	created, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := creds.creds[created.ID].Ciphertext

	const newToken = "eyJhbGciOiJIUzI1NiJ9.token.baru.yang.panjang.sekali"
	token := newToken
	if _, err := svc.Update(context.Background(), userIDTest, created.ID, dto.UpdateAccountRequest{
		Token: &token,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	if bytes.Equal(creds.creds[created.ID].Ciphertext, before) {
		t.Fatal("blob kredensial seharusnya diperbarui")
	}
	if len(spy.ids) != 1 || spy.ids[0] != created.ID {
		t.Fatalf("sesi lama seharusnya diputus, tercatat: %v", spy.ids)
	}
}

// Update tanpa token tidak boleh menyentuh kredensial.
func TestUpdateTanpaTokenTidakMenyentuhKredensial(t *testing.T) {
	user := newTestUser("basic")

	users := newFakeUserRepo()
	users.users[user.Email] = user
	creds := &fakeCredRepo{creds: map[string]*model.Credential{}}
	spy := &revokerSpy{}

	svc := NewAccountService(AccountServiceDeps{
		Users:             users,
		Accounts:          &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}},
		Credentials:       creds,
		Subscriptions:     &fakeSubRepo{},
		Audit:             &fakeAuditRepo{},
		Sealer:            testKeyring(t),
		Strategies:        &fakeStrategies{ila3: "off"},
		Revoker:           spy,
		FingerprintSecret: fingerprintTestSecret,
	})

	created, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	before := creds.creds[created.ID].Ciphertext

	label := "Akun Utama"
	if _, err := svc.Update(context.Background(), userIDTest, created.ID, dto.UpdateAccountRequest{
		Label: &label,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}

	if !bytes.Equal(creds.creds[created.ID].Ciphertext, before) {
		t.Fatal("kredensial tidak boleh berubah tanpa token baru")
	}
	if len(spy.ids) != 0 {
		t.Fatal("sesi tidak perlu diputus bila token tidak berubah")
	}
}

// Dua akun dengan token sama berarti dua eksekusi ke satu akun broker.
// Pendaftaran harus ditolak, termasuk saat token baru disamakan.
func TestTokenSamaDitolak(t *testing.T) {
	user := newTestUser("plus")

	users := newFakeUserRepo()
	users.users[user.Email] = user

	svc := NewAccountService(AccountServiceDeps{
		Users:             users,
		Accounts:          &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}},
		Credentials:       &fakeCredRepo{creds: map[string]*model.Credential{}},
		Subscriptions:     &fakeSubRepo{},
		Audit:             &fakeAuditRepo{},
		Sealer:            testKeyring(t),
		Strategies:        &fakeStrategies{ila3: "general"},
		FingerprintSecret: fingerprintTestSecret,
	})

	first, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	})
	if err != nil {
		t.Fatalf("akun pertama: %v", err)
	}

	if _, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken,
	}); !errors.Is(err, ErrDuplicateCredential) {
		t.Fatalf("token kembar seharusnya ditolak, dapat: %v", err)
	}

	second, err := svc.Create(context.Background(), userIDTest, dto.CreateAccountRequest{
		BrokerType: "matchtrader", Token: sampleToken + "-lain",
	})
	if err != nil {
		t.Fatalf("akun kedua dengan token lain: %v", err)
	}

	// Menyamakan token akun kedua dengan akun pertama juga harus ditolak.
	if _, err := svc.Update(context.Background(), userIDTest, second.ID, dto.UpdateAccountRequest{
		Token: &first.Label,
	}); err == nil {
		t.Fatal("token harus diisi, bukan label")
	}

	same := sampleToken
	if _, err := svc.Update(context.Background(), userIDTest, second.ID, dto.UpdateAccountRequest{
		Token: &same,
	}); !errors.Is(err, ErrDuplicateCredential) {
		t.Fatalf("rotasi ke token kembar seharusnya ditolak, dapat: %v", err)
	}
}
