package runner

import (
	"context"
	"sync"
	"time"
)

// PairSource adalah sumber daftar simbol aktif; biasanya repository.
type PairSource interface {
	EnabledPairSymbols(ctx context.Context) ([]string, error)
}

// CachedPairGate menyaring simbol aktif dengan cache singkat.
//
// Tujuan cache: ComputeRequirements dipanggil pada setiap siklus sinkronisasi,
// sehingga query katalog tidak boleh terjadi pada setiap panggilan. Umur cache
// dibuat pendek agar perubahan admin cepat berlaku, tapi cukup panjang untuk
// menghindari beban berlebih.
type CachedPairGate struct {
	source PairSource
	ttl    time.Duration
	now    func() time.Time

	mu        sync.Mutex
	cached    map[string]bool
	expiresAt time.Time
}

// NewCachedPairGate membuat gate dengan umur cache ttl.
// Nilai ttl nol jatuh ke satu menit agar perubahan admin tetap cepat terlihat.
func NewCachedPairGate(source PairSource, ttl time.Duration) *CachedPairGate {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &CachedPairGate{
		source: source,
		ttl:    ttl,
		now:    func() time.Time { return time.Now() },
	}
}

// EnabledSymbols mengembalikan himpunan simbol aktif, memakai cache bila masih
// berlaku. Saat cache kedaluwarsa tetapi sumber gagal, cache lama dipakai agar
// tidak terjadi lonjakan unsubscribe karena gangguan sesaat.
func (g *CachedPairGate) EnabledSymbols(ctx context.Context) (map[string]bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.now()
	if g.cached != nil && now.Before(g.expiresAt) {
		return g.cached, nil
	}

	symbols, err := g.source.EnabledPairSymbols(ctx)
	if err != nil {
		if g.cached != nil {
			return g.cached, nil
		}
		return nil, err
	}

	fresh := make(map[string]bool, len(symbols))
	for _, s := range symbols {
		fresh[s] = true
	}
	g.cached = fresh
	g.expiresAt = now.Add(g.ttl)
	return fresh, nil
}
