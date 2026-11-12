package runner

import (
	"context"
	"sync"
	"time"

	"ila/backend/internal/broker"
	"ila/backend/internal/model"
)

// AccountSnapshotTTL adalah umur snapshot akun yang boleh dipakai ulang.
//
// Satu eksekusi membutuhkan equity untuk dua hal: memastikan akun sehat, lalu
// menghitung ukuran lot. Tanpa cache, keduanya menempuh perjalanan jaringan
// terpisah ke broker - ongkos yang langsung menambah keterlambatan entry.
//
// Dua detik dipilih karena kompromi teraman:
//   - Cukup panjang untuk menghapus satu perjalanan penuh dalam satu eksekusi.
//   - Cukup pendek sehingga deposit atau penarikan dana yang terjadi tepat
//     sebelum order tetap terbaca. Equity yang bergerak karena harga berubah
//     dalam dua detik tidak mengubah ukuran lot secara berarti.
const AccountSnapshotTTL = 2 * time.Second

// accountSnapshotCache menyimpan snapshot akun sebentar agar satu eksekusi
// tidak membaca akun dua kali.
//
// Aman dipakai beberapa goroutine: fan-out memproses banyak akun bersamaan dan
// akun yang sama bisa saja dievaluasi dari sinyal berbeda.
type accountSnapshotCache struct {
	mu      sync.Mutex
	entries map[string]accountSnapshotEntry
	now     func() time.Time
}

type accountSnapshotEntry struct {
	snapshot broker.AccountSnapshot
	expires  time.Time
}

func newAccountSnapshotCache(now func() time.Time) *accountSnapshotCache {
	if now == nil {
		now = time.Now
	}
	return &accountSnapshotCache{
		entries: make(map[string]accountSnapshotEntry),
		now:     now,
	}
}

// get mengembalikan snapshot yang masih berlaku untuk akun tersebut.
func (c *accountSnapshotCache) get(accountID string) (broker.AccountSnapshot, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[accountID]
	if !ok {
		return broker.AccountSnapshot{}, false
	}
	if !c.now().Before(entry.expires) {
		delete(c.entries, accountID)
		return broker.AccountSnapshot{}, false
	}
	return entry.snapshot, true
}

// put menyimpan snapshot selama AccountSnapshotTTL.
func (c *accountSnapshotCache) put(accountID string, snapshot broker.AccountSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Buang entri yang sudah kedaluwarsa agar peta tidak tumbuh selamanya
	// ketika akun dihapus atau berganti kredensial.
	now := c.now()
	for id, entry := range c.entries {
		if !now.Before(entry.expires) {
			delete(c.entries, id)
		}
	}
	c.entries[accountID] = accountSnapshotEntry{
		snapshot: snapshot,
		expires:  now.Add(AccountSnapshotTTL),
	}
}

// forget membuang snapshot satu akun. Dipakai saat kredensial diganti atau
// akun dicabut, supaya data lama tidak dipakai untuk keputusan berikutnya.
func (c *accountSnapshotCache) forget(accountID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, accountID)
}

// accountSnapshot membaca kondisi akun, memakai cache bila masih segar.
//
// Kegagalan tidak di-cache: percobaan berikutnya harus benar-benar menghubungi
// broker, bukan mengulang kegagalan yang mungkin sudah pulih.
func (s *Service) accountSnapshot(ctx context.Context, account *model.BrokerAccount) (broker.AccountSnapshot, error) {
	if s.accounts != nil {
		if snapshot, ok := s.accounts.get(account.ID); ok {
			return snapshot, nil
		}
	}

	client, err := s.client(ctx, account)
	if err != nil {
		return broker.AccountSnapshot{}, err
	}
	snapshot, err := client.Account(ctx)
	if err != nil {
		return broker.AccountSnapshot{}, err
	}

	if s.accounts != nil {
		s.accounts.put(account.ID, snapshot)
	}
	return snapshot, nil
}
