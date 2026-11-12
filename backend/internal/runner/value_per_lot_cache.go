package runner

import (
	"context"
	"sync"
	"time"

	"ila/backend/internal/model"
	"ila/backend/internal/repository"
)

// PairValueSource menyediakan metadata kontrak pair dari katalog admin.
type PairValueSource interface {
	ListPairs(ctx context.Context, filter repository.PairFilter) ([]model.TradingPair, error)
}

// CachedValuePerLot menyajikan nilai per lot per simbol dengan cache singkat.
//
// Sumber utamanya katalog pair yang diisi admin (value_per_lot, contract_size).
// Bila admin belum mengisi, nilai dihitung dari contract_size; bila itu pun
// kosong dipakai DefaultValuePerLot. Urutan ini penting: nilai yang diisi admin
// harus menang, karena hanya dia yang tahu kontrak broker yang dipakai.
type CachedValuePerLot struct {
	source PairValueSource
	ttl    time.Duration
	now    func() time.Time

	mu        sync.Mutex
	cached    map[string]float64
	expiresAt time.Time

	// memuat menahan pemanggil lain selama satu pemuatan berjalan. Dipisah
	// dari mu agar kueri basis data tidak berjalan sambil memegang mu: kalau
	// tidak, satu kueri lambat menghentikan seluruh penutupan posisi lain.
	memuat sync.Mutex
}

// NewCachedValuePerLot membuat pembaca nilai per lot dengan umur cache ttl.
func NewCachedValuePerLot(source PairValueSource, ttl time.Duration) *CachedValuePerLot {
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &CachedValuePerLot{
		source: source,
		ttl:    ttl,
		now:    func() time.Time { return time.Now() },
	}
}

// ValuePerLot mengembalikan nilai uang untuk pergerakan harga 1.0 pada 1 lot.
//
// Tidak pernah mengembalikan nol: R harus tetap terhitung walaupun katalog
// sedang tidak terbaca. Saat gagal, nilai bawaan per jenis simbol dipakai.
func (c *CachedValuePerLot) ValuePerLot(symbol string) float64 {
	if nilai, ok := c.dariCache(symbol); ok {
		return nilai
	}

	c.muatUlang()

	if nilai, ok := c.dariCache(symbol); ok {
		return nilai
	}
	return DefaultValuePerLot(symbol)
}

// dariCache membaca nilai tanpa memuat ulang katalog.
func (c *CachedValuePerLot) dariCache(symbol string) (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached == nil || !c.now().Before(c.expiresAt) {
		return 0, false
	}
	nilai, ok := c.cached[symbol]
	if !ok || nilai <= 0 {
		return 0, false
	}
	return nilai, true
}

// muatUlang membaca katalog sekali untuk banyak pemanggil.
//
// Pemanggil yang datang saat pemuatan berjalan menunggu di `memuat`, lalu
// memeriksa cache lagi: muatan yang baru selesai hampir pasti sudah memuat
// simbol yang dicarinya, sehingga kueri tidak diulang.
func (c *CachedValuePerLot) muatUlang() {
	c.memuat.Lock()
	defer c.memuat.Unlock()

	c.mu.Lock()
	masihBerlaku := c.cached != nil && c.now().Before(c.expiresAt)
	c.mu.Unlock()
	if masihBerlaku {
		return
	}

	// Konteks tanpa tenggat wajar di sini: pemanggil adalah penutupan posisi
	// yang tidak punya tenggat sendiri, dan muatannya satu kueri kecil yang
	// sudah dibatasi cache satu menit.
	peta, err := c.load(context.Background())
	if err != nil {
		// Katalog gagal dibaca: cache lama dibiarkan (dariCache akan menolaknya
		// bila sudah kedaluwarsa, lalu nilai bawaan dipakai).
		return
	}

	c.mu.Lock()
	c.cached = peta
	c.expiresAt = c.now().Add(c.ttl)
	c.mu.Unlock()
}

// load membaca katalog lalu menghitung nilai per lot per simbol.
func (c *CachedValuePerLot) load(ctx context.Context) (map[string]float64, error) {
	rows, err := c.source.ListPairs(ctx, repository.PairFilter{})
	if err != nil {
		return nil, err
	}

	peta := make(map[string]float64, len(rows))
	for i := range rows {
		pair := &rows[i]
		switch {
		case pair.ValuePerLot > 0:
			peta[pair.Symbol] = pair.ValuePerLot
		case pair.ContractSize > 0:
			// Tanpa value_per_lot, nilai 1.0 pergerakan sama dengan ukuran
			// kontrak (1 lot = contract_size unit).
			peta[pair.Symbol] = pair.ContractSize
		}
	}
	return peta, nil
}
