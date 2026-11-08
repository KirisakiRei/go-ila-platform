package alert

import (
	"context"
	"testing"
	"time"
)

// fakeSender mencatat pengiriman tanpa jaringan.
type fakeSender struct{ sent []string }

func (f *fakeSender) Send(_ context.Context, _, text string) error {
	f.sent = append(f.sent, text)
	return nil
}

func TestAntiBanjirBatasSatuPerCooldown(t *testing.T) {
	sender := &fakeSender{}
	n := New(sender, "chat-1", testLogger())
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	n.now = func() time.Time { return now }

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		n.Fire(ctx, EngineDisconnected, "engine mati")
	}
	if len(sender.sent) != 1 {
		t.Fatalf("pesan terkirim = %d; ingin 1 karena cooldown", len(sender.sent))
	}

	now = now.Add(Cooldown + time.Second)
	n.Fire(ctx, EngineDisconnected, "engine mati lagi")
	if len(sender.sent) != 2 {
		t.Fatalf("pesan terkirim = %d; ingin 2 setelah cooldown", len(sender.sent))
	}
}

func TestKabarPulihHanyaSekali(t *testing.T) {
	sender := &fakeSender{}
	n := New(sender, "chat-1", testLogger())
	ctx := context.Background()

	n.Fire(ctx, BrokerErrors, "3 akun error")
	n.Recovered(ctx, BrokerErrors, "semua akun tersambung")
	n.Recovered(ctx, BrokerErrors, "semua akun tersambung")
	if len(sender.sent) != 2 {
		t.Fatalf("pesan terkirim = %d; ingin peringatan + satu kabar pulih", len(sender.sent))
	}
}

func TestChatKosongTidakMengirim(t *testing.T) {
	sender := &fakeSender{}
	n := New(sender, "", testLogger())
	if n.Fire(context.Background(), EngineDisconnected, "engine mati") {
		t.Fatal("tanpa chat admin, tidak boleh ada pengiriman")
	}
	if len(sender.sent) != 0 {
		t.Fatalf("pesan terkirim = %d; ingin 0", len(sender.sent))
	}
}

func TestPemicuBerbedaTidakSalingMenahan(t *testing.T) {
	sender := &fakeSender{}
	n := New(sender, "chat-1", testLogger())
	ctx := context.Background()

	n.Fire(ctx, EngineDisconnected, "a")
	n.Fire(ctx, DiskAlmostFull, "b")
	if len(sender.sent) != 2 {
		t.Fatalf("pesan terkirim = %d; ingin 2 pemicu berbeda", len(sender.sent))
	}
}
