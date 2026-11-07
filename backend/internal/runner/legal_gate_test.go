package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/runner"
)

// newServiceWithLegal membuat Runner dengan penanda persetujuan dokumen
// yang dikendalikan tes.
func newServiceWithLegal(store *fakeStore, pub *fakePublisher, blocked func(_, _ string) bool) *runner.Service {
	registry := broker.NewRegistry()
	registry.Register("paper", func(accountID, brokerType string) (broker.Client, error) {
		return broker.NewPaperClient(accountID, brokerType), nil
	})

	return runner.NewService(store, registry, pub, discardLogger(), runner.Options{
		Policy:          runner.DefaultDailyPolicy(),
		SlippageGuard:   true,
		AutoExecute:     true,
		MarketOpenCheck: false,
		Settings:        ila3General(),
		LegalGate: func(_ context.Context, userID string) bool {
			return blocked("", userID)
		},
	})
}

// Dokumen wajib yang belum disetujui menahan entry baru.
func TestLegalPendingBlocksEntry(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newServiceWithLegal(store, pub, func(_, userID string) bool {
		return userID == "u1"
	})
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 0 {
		t.Fatal("dokumen belum disetujui harus memblokir entry")
	}

	us, _ := store.FindUserSignal(context.Background(), "u1", "sig-XAUUSD")
	if us.RejectReason != string(runner.RejectLegalPending) {
		t.Fatalf("alasan = %q, ingin legal pending", us.RejectReason)
	}
}

// Setelah dokumen disetujui, entry berjalan seperti biasa.
func TestLegalAcceptedAllowsEntry(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newServiceWithLegal(store, pub, func(_, _ string) bool { return false })
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	positions, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(positions) != 1 {
		t.Fatal("persetujuan lengkap harus mengizinkan entry")
	}
}

// Dokumen wajib tidak menutup posisi yang sudah berjalan.
func TestLegalPendingDoesNotClosePositions(t *testing.T) {
	store := newFakeStore()
	pub := &fakePublisher{}
	addUser(store, "u1", "plus", "auto", true)

	svc := newServiceWithLegal(store, pub, func(_, _ string) bool { return false })
	svc.HandleSignal(context.Background(), readySignal("XAUUSD", time.Hour))

	before, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(before) != 1 {
		t.Fatal("posisi awal harus terbuka")
	}

	// Dokumen baru terbit: hanya entry berikutnya yang ditahan.
	blockedSvc := newServiceWithLegal(store, pub, func(_, _ string) bool { return true })
	blockedSvc.HandleSignal(context.Background(), readySignal("GBPUSD", time.Hour))

	after, _ := store.ListOpenPositionsByUser(context.Background(), "u1")
	if len(after) != 1 {
		t.Fatalf("posisi berjalan tidak boleh bertambah atau tertutup, dapat %d", len(after))
	}
}
