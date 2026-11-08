package service

import (
	"bytes"
	"context"
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
		Users:         users,
		Accounts:      accounts,
		Credentials:   creds,
		Subscriptions: &fakeSubRepo{},
		Audit:         &fakeAuditRepo{},
		Sealer:        testKeyring(t),
		Strategies:    &fakeStrategies{ila3: "off"},
		Revoker:       spy,
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
		Users:         users,
		Accounts:      &fakeAccountRepo{accounts: map[string]*model.BrokerAccount{}},
		Credentials:   creds,
		Subscriptions: &fakeSubRepo{},
		Audit:         &fakeAuditRepo{},
		Sealer:        testKeyring(t),
		Strategies:    &fakeStrategies{ila3: "off"},
		Revoker:       spy,
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
