package market

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type fakeStreamProvider struct {
	connections atomic.Int32
	pushC       chan []Candle
}

func (f *fakeStreamProvider) StreamCandles(ctx context.Context, _, _, _ string, onBars func([]Candle)) error {
	f.connections.Add(1)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case bars, ok := <-f.pushC:
			if !ok {
				return nil
			}
			onBars(bars)
		}
	}
}

func TestStreamHubMultiplexDanAutoSleepSaatIdle(t *testing.T) {
	prov := &fakeStreamProvider{pushC: make(chan []Candle, 4)}
	hub := NewStreamHub(prov, StreamHubOptions{
		MaxStreams:      2,
		MaxSubscribers:  10,
		IdleGrace:       40 * time.Millisecond,
		MinEmitInterval: 10 * time.Millisecond,
	})
	defer hub.Close()

	// Dua penonton pada pasangan simbol+timeframe yang sama hanya membuka 1 stream upstream.
	sub1, feed1, unsub1, err := hub.Subscribe("xauusd", "forexcom", "M15")
	if err != nil {
		t.Fatalf("Subscribe 1 gagal: %v", err)
	}
	sub2, feed2, unsub2, err := hub.Subscribe("XAUUSD", "FOREXCOM", "15M")
	if err != nil {
		t.Fatalf("Subscribe 2 gagal: %v", err)
	}
	if feed1 != "FOREXCOM" || feed2 != "FOREXCOM" {
		t.Fatalf("feed tidak konsisten: %q / %q", feed1, feed2)
	}
	if hub.ActiveStreamCount() != 1 {
		t.Fatalf("ActiveStreamCount = %d, ingin 1 (multiplexed)", hub.ActiveStreamCount())
	}

	wantBar := Candle{Time: 1700000000, Open: 2600, High: 2605, Low: 2599, Close: 2604}
	prov.pushC <- []Candle{wantBar}

	for idx, ch := range []<-chan Candle{sub1, sub2} {
		select {
		case got := <-ch:
			if got.Close != wantBar.Close || got.Time != wantBar.Time {
				t.Fatalf("penonton %d menerima bar salah: %+v", idx+1, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("penonton %d tidak menerima siaran candle", idx+1)
		}
	}

	if gotConn := prov.connections.Load(); gotConn != 1 {
		t.Fatalf("koneksi upstream = %d, ingin tepat 1", gotConn)
	}

	// Saat seluruh penonton berhenti, stream otomatis diputus setelah masa tenggang.
	unsub1()
	unsub2()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if hub.ActiveStreamCount() == 0 {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("stream idle seharusnya sudah ditutup, masih aktif %d", hub.ActiveStreamCount())
}

func TestStreamHubBatasKapasitasMelindungiServer(t *testing.T) {
	prov := &fakeStreamProvider{pushC: make(chan []Candle, 1)}
	hub := NewStreamHub(prov, StreamHubOptions{
		MaxStreams:      1,
		MaxSubscribers:  4,
		IdleGrace:       time.Second,
		MinEmitInterval: 10 * time.Millisecond,
	})
	defer hub.Close()

	_, _, unsub1, err := hub.Subscribe("EURUSD", "FOREXCOM", "H1")
	if err != nil {
		t.Fatalf("stream pertama gagal: %v", err)
	}
	defer unsub1()

	if _, _, _, err := hub.Subscribe("GBPUSD", "FOREXCOM", "H1"); err != ErrStreamCapacity {
		t.Fatalf("stream kedua di atas batas seharusnya ErrStreamCapacity, dapat %v", err)
	}
}
