package runner_test

import (
	"context"
	"testing"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/runner"
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

// Hasil yang tidak dilaporkan broker tidak boleh dihitung sebagai kalah,
// karena akan menghabiskan kuota tanpa trade yang benar-benar rugi.
func TestHasilTidakDiketahuiTidakMenghabiskanKuota(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	pub := &fakePublisher{}
	svc := newService(store, pub)
	svc.SetClock(func() time.Time { return now })

	position := &model.Position{
		Base:         model.Base{ID: "pos-3"},
		UserID:       "u3",
		Symbol:       "GBPUSD",
		Status:       model.PositionStatusOpen,
		TicketNumber: "T-3",
		OpenPrice:    1.25,
		StopLoss:     1.24,
		LotSize:      1,
		OpenedAt:     now.Add(-time.Hour),
	}

	// Exit price dan profit nol: broker tidak melaporkan hasil.
	svc.ApplyClose(context.Background(), position, model.PositionStatusClosedManual, 0, 0)

	summary, err := store.GetDailySummary(context.Background(), "u3", runner.DayKey(now))
	if err != nil {
		t.Fatalf("membaca ringkasan galat: %v", err)
	}
	if summary.TotalTrades != 0 || summary.LossCount != 0 {
		t.Fatalf("kuota terpakai padahal hasil tak diketahui: %+v", summary)
	}
}

// Hasil yang diketahui tetap dihitung seperti biasa.
func TestHasilDiketahuiTetapMencatatKuota(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	pub := &fakePublisher{}
	svc := newService(store, pub)
	svc.SetClock(func() time.Time { return now })

	position := &model.Position{
		Base:         model.Base{ID: "pos-4"},
		UserID:       "u4",
		Symbol:       "GBPUSD",
		Status:       model.PositionStatusOpen,
		TicketNumber: "T-4",
		OpenPrice:    1.25,
		StopLoss:     1.24,
		LotSize:      1,
		OpenedAt:     now.Add(-time.Hour),
	}

	// Alur nyata: trade dihitung saat eksekusi, hasil dicatat saat tutup.
	if err := store.IncrementDailyTrade(context.Background(), "u4", runner.DayKey(now)); err != nil {
		t.Fatalf("menaikkan kuota galat: %v", err)
	}
	svc.ApplyClose(context.Background(), position, model.PositionStatusClosedSL, 1.24, -100)

	summary, err := store.GetDailySummary(context.Background(), "u4", runner.DayKey(now))
	if err != nil {
		t.Fatalf("membaca ringkasan galat: %v", err)
	}
	if summary.TotalTrades != 1 || summary.LossCount != 1 {
		t.Fatalf("kuota tidak tercatat: %+v", summary)
	}
}
