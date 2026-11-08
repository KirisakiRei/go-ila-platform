package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
)

// Trial tidak boleh mengeksekusi order nyata, walau akunnya bukan dry-run.
// Ini pagar uang yang paling penting (plans/14 par 5.5.2).
func TestTrialDipaksaSimulasi(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", false)

	expiresAt := time.Now().UTC().Add(48 * time.Hour)
	store.users["u1"].TrialExpiresAt = &expiresAt

	svc := newServiceWithSettings(store, pub, runner.NewSettings(runner.ILA3Off))

	account := store.accounts["acct-u1"]
	if !runner.MustDryRunForTest(svc, store.users["u1"], account) {
		t.Fatal("user trial wajib disimulasikan walau akunnya bukan dry-run")
	}

	// Setelah trial berakhir dan akunnya bukan dry-run, pagar simulasi lepas.
	past := time.Now().UTC().Add(-time.Hour)
	store.users["u1"].TrialExpiresAt = &past
	if runner.MustDryRunForTest(svc, store.users["u1"], account) {
		t.Fatal("setelah trial berakhir, akun nyata tidak lagi dipaksa simulasi")
	}
}

// Trial berjalan Basic, jadi mode auto tidak boleh dipakai.
func TestTrialTidakDapatModeAuto(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	expiresAt := time.Now().UTC().Add(48 * time.Hour)
	store.users["u1"].TrialExpiresAt = &expiresAt

	svc := newServiceWithSettings(store, pub, runner.NewSettings(runner.ILA3Off))
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatalf("trial dengan mode auto tidak boleh eksekusi, dapat %d posisi", len(positions))
	}
}

func TestInTrial(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour)
	past := time.Now().UTC().Add(-time.Hour)
	now := time.Now().UTC()

	if !runner.InTrial(&model.User{TrialExpiresAt: &future}, now) {
		t.Error("trial yang belum berakhir harus terdeteksi")
	}
	if runner.InTrial(&model.User{TrialExpiresAt: &past}, now) {
		t.Error("trial yang sudah lewat tidak boleh terdeteksi")
	}
	if runner.InTrial(&model.User{}, now) {
		t.Error("user tanpa trial tidak boleh dianggap trial")
	}
	if runner.InTrial(nil, now) {
		t.Error("nil user tidak boleh panik atau dianggap trial")
	}
}
