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
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatalf("posisi tidak tercatat: %d", len(positions))
	}
	if !positions[0].IsDryRun {
		t.Fatal("posisi dari user trial wajib ditandai simulasi")
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
