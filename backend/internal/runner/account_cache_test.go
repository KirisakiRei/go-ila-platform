package runner

import (
	"testing"
	"time"

	"ila/backend/internal/broker"
)

// TestAccountSnapshotCacheMemakaiUlangDalamTTL memastikan dua pembacaan akun
// dalam satu eksekusi hanya menempuh satu perjalanan ke broker.
func TestAccountSnapshotCacheMemakaiUlangDalamTTL(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := newAccountSnapshotCache(func() time.Time { return now })

	cache.put("acct-1", broker.AccountSnapshot{AccountID: "acct-1", Equity: 1000})

	now = now.Add(AccountSnapshotTTL / 2)
	snapshot, ok := cache.get("acct-1")
	if !ok {
		t.Fatal("snapshot dalam TTL harus tersedia")
	}
	if snapshot.Equity != 1000 {
		t.Fatalf("Equity = %v, ingin 1000", snapshot.Equity)
	}
}

// TestAccountSnapshotCacheKedaluwarsa memastikan equity lama tidak dipakai
// melewati TTL: deposit/penarikan harus terbaca pada eksekusi berikutnya.
func TestAccountSnapshotCacheKedaluwarsa(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := newAccountSnapshotCache(func() time.Time { return now })

	cache.put("acct-1", broker.AccountSnapshot{AccountID: "acct-1", Equity: 1000})

	now = now.Add(AccountSnapshotTTL)
	if _, ok := cache.get("acct-1"); ok {
		t.Fatal("snapshot pada tepat TTL harus dianggap kedaluwarsa")
	}
}

// TestAccountSnapshotCacheTerpisahPerAkun menjaga agar akun berbeda tidak
// saling memakai equity.
func TestAccountSnapshotCacheTerpisahPerAkun(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := newAccountSnapshotCache(func() time.Time { return now })

	cache.put("acct-1", broker.AccountSnapshot{AccountID: "acct-1", Equity: 1000})
	cache.put("acct-2", broker.AccountSnapshot{AccountID: "acct-2", Equity: 5000})

	satu, ok := cache.get("acct-1")
	if !ok || satu.Equity != 1000 {
		t.Fatalf("acct-1 Equity = %v, ingin 1000", satu.Equity)
	}
	dua, ok := cache.get("acct-2")
	if !ok || dua.Equity != 5000 {
		t.Fatalf("acct-2 Equity = %v, ingin 5000", dua.Equity)
	}
}

// TestAccountSnapshotCacheForget memastikan pencabutan kredensial membuang
// data akun sehingga tidak dipakai untuk keputusan berikutnya.
func TestAccountSnapshotCacheForget(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache := newAccountSnapshotCache(func() time.Time { return now })

	cache.put("acct-1", broker.AccountSnapshot{AccountID: "acct-1", Equity: 1000})
	cache.forget("acct-1")

	if _, ok := cache.get("acct-1"); ok {
		t.Fatal("snapshot yang dilupakan tidak boleh tersedia")
	}
}

// TestFanOutConcurrencyTerbatas menjaga agar batas konkurensi tetap wajar:
// terlalu besar membuka puluhan koneksi broker, terlalu kecil mematikan
// manfaat paralelisasi.
func TestFanOutConcurrencyTerbatas(t *testing.T) {
	if FanOutConcurrency < 2 {
		t.Fatalf("FanOutConcurrency = %d, ingin minimal 2 agar paralel", FanOutConcurrency)
	}
	if FanOutConcurrency > 32 {
		t.Fatalf("FanOutConcurrency = %d, terlalu besar untuk VPS kecil", FanOutConcurrency)
	}
}

// TestAccountSnapshotTTLAman menjaga kompromi kesegaran data.
func TestAccountSnapshotTTLAman(t *testing.T) {
	if AccountSnapshotTTL <= 0 {
		t.Fatal("AccountSnapshotTTL harus positif")
	}
	if AccountSnapshotTTL > 5*time.Second {
		t.Fatalf("AccountSnapshotTTL = %v, terlalu lama untuk data equity", AccountSnapshotTTL)
	}
}
