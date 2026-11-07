package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/runner"
)

// Akun yang di-ban tidak boleh dieksekusi, walaupun sesinya masih hidup dan
// akun brokernya aktif. Ini celah C3 di plans/19.
func TestBannedUserNotExecuted(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)
	store.users["u1"].Status = "banned"

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, err := store.ListOpenPositionsByUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("membaca posisi galat: %v", err)
	}
	if len(positions) != 0 {
		t.Fatalf("user ter-ban membuka %d posisi, ingin 0", len(positions))
	}

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.signals) == 0 {
		t.Fatal("penolakan harus dicatat untuk user")
	}
	last := pub.signals[len(pub.signals)-1]
	if last.Status != "REJECTED" {
		t.Fatalf("status = %q, ingin REJECTED", last.Status)
	}
	if last.Reason != string(runner.RejectAccountBanned) {
		t.Fatalf("alasan = %q, ingin %q", last.Reason, runner.RejectAccountBanned)
	}
}

// Suspended tetap boleh melihat data tetapi tidak boleh entry baru.
func TestSuspendedUserNotExecuted(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	past := time.Now().Add(-time.Hour)
	store.users["u1"].SuspendedAt = &past

	svc := newService(store, pub)
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatalf("user ditangguhkan membuka %d posisi, ingin 0", len(positions))
	}
}
