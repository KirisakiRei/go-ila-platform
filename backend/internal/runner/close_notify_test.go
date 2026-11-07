package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
)

// Posisi yang ditutup broker wajib menghasilkan notifikasi ke user.
// Tanpa ini user tidak pernah tahu hasil trading-nya (plans/19 C1).
func TestTutupPosisiMengirimNotifikasi(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	pub := &fakePublisher{}
	svc := newService(store, pub)
	svc.SetClock(func() time.Time { return now })

	position := &model.Position{
		Base:         model.Base{ID: "pos-1"},
		UserID:       "u1",
		Symbol:       "XAUUSD",
		Status:       model.PositionStatusOpen,
		TicketNumber: "T-1",
		OpenPrice:    2000,
		StopLoss:     1990,
		LotSize:      1,
		OpenedAt:     now.Add(-time.Hour),
	}

	svc.ApplyClose(context.Background(), position, model.PositionStatusClosedSL, 1990, -100)

	pub.mu.Lock()
	defer pub.mu.Unlock()

	if len(pub.closed) != 1 {
		t.Fatalf("notifikasi tutup = %d, ingin 1", len(pub.closed))
	}
	if pub.closed[0].positionID != "pos-1" {
		t.Fatalf("posisi salah: %s", pub.closed[0].positionID)
	}
	if pub.closed[0].daily == nil {
		t.Fatal("catatan kuota harian wajib ikut dikirim")
	}
}

// Menang pada trade pertama harus mengunci hari dan menandai daily.Locked.
func TestTutupMenangMengunciHari(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	pub := &fakePublisher{}
	svc := newService(store, pub)
	svc.SetClock(func() time.Time { return now })

	position := &model.Position{
		Base:         model.Base{ID: "pos-2"},
		UserID:       "u2",
		Symbol:       "EURUSD",
		Status:       model.PositionStatusOpen,
		TicketNumber: "T-2",
		OpenPrice:    1.1,
		StopLoss:     1.09,
		LotSize:      1,
		OpenedAt:     now.Add(-time.Hour),
	}

	svc.ApplyClose(context.Background(), position, model.PositionStatusClosedTP, 1.12, 200)

	pub.mu.Lock()
	defer pub.mu.Unlock()

	if len(pub.closed) != 1 {
		t.Fatalf("notifikasi tutup = %d, ingin 1", len(pub.closed))
	}
	if !pub.closed[0].daily.Locked {
		t.Fatal("kemenangan pertama seharusnya mengunci hari")
	}
}
