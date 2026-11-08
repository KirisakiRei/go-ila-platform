package middleware

import (
	"sync"
	"time"
)

// Batas percobaan login sesuai plans/12 §4.1: 5 gagal per 15 menit.
const (
	loginMaxFailures = 5
	loginWindow      = 15 * time.Minute
)

// Batas pendaftaran: 5 akun per IP per jam, dan 1 akun per alamat surel.
// Angka ini menahan pembuatan akun massal dari satu jaringan tanpa menahan
// pengguna yang salah ketik lalu mendaftar ulang.
const (
	registerMaxPerIP   = 5
	registerWindow     = time.Hour
	registerEmailLimit = 1
)

// LoginLimiter menghitung kegagalan per kunci (email + IP) di memori.
//
// Disimpan di proses karena hanya ada satu backend-api; bila nanti berjalan
// lebih dari satu replika, hitungan ini perlu dipindah ke Redis.
type LoginLimiter struct {
	mu      sync.Mutex
	entries map[string]*loginEntry
	now     func() time.Time
}

type loginEntry struct {
	failures int
	resetAt  time.Time
}

// RegisterLimiter membatasi pendaftaran akun.
//
// Dua sumbu dihitung terpisah: per IP menahan pendaftaran massal dari satu
// jaringan, per surel menahan pengulangan ke alamat yang sama. Reset saat
// pendaftaran berhasil agar pengguna sah tidak terhalang percobaan sebelumnya.
type RegisterLimiter struct {
	mu     sync.Mutex
	byIP   map[string][]time.Time
	byMail map[string][]time.Time
	now    func() time.Time
}

// NewRegisterLimiter membuat pembatas pendaftaran.
func NewRegisterLimiter() *RegisterLimiter {
	return &RegisterLimiter{
		byIP:   make(map[string][]time.Time),
		byMail: make(map[string][]time.Time),
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// Allow menandai apakah pendaftaran dari pasangan IP dan surel diizinkan.
// Nilai kedua adalah sisa detik sampai percobaan berikutnya boleh dicoba.
func (l *RegisterLimiter) Allow(ip, email string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if count, retry := pruneWindow(l.byIP[ip], now); count >= registerMaxPerIP {
		return false, retry
	}
	if count, retry := pruneWindow(l.byMail[email], now); count >= registerEmailLimit {
		return false, retry
	}
	return true, 0
}

// Record mencatat satu pendaftaran yang berhasil.
func (l *RegisterLimiter) Record(ip, email string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.byIP[ip] = append(pruneSlice(l.byIP[ip], now), now)
	l.byMail[email] = append(pruneSlice(l.byMail[email], now), now)
}

// pruneWindow memangkas catatan lama lalu mengembalikan jumlah dan sisa detik.
func pruneWindow(times []time.Time, now time.Time) (int, int) {
	kept := pruneSlice(times, now)
	if len(kept) == 0 {
		return 0, 0
	}
	retry := int(kept[0].Add(registerWindow).Sub(now).Seconds())
	if retry < 1 {
		retry = 1
	}
	return len(kept), retry
}

// pruneSlice membuang waktu yang sudah lewat jendela.
func pruneSlice(times []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-registerWindow)
	kept := times[:0]
	for _, at := range times {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	return kept
}

// NewLoginLimiter membuat pembatas percobaan login.
func NewLoginLimiter() *LoginLimiter {
	return &LoginLimiter{
		entries: make(map[string]*loginEntry),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// Blocked menandai kunci yang masih terkunci.
func (l *LoginLimiter) Blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok {
		return false
	}
	if l.now().After(entry.resetAt) {
		delete(l.entries, key)
		return false
	}
	return entry.failures >= loginMaxFailures
}

// Fail menambah hitungan kegagalan dan membuka jendela baru bila perlu.
func (l *LoginLimiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	entry, ok := l.entries[key]
	if !ok || now.After(entry.resetAt) {
		l.entries[key] = &loginEntry{failures: 1, resetAt: now.Add(loginWindow)}
		return
	}
	entry.failures++
}

// Reset menghapus hitungan setelah login berhasil.
func (l *LoginLimiter) Reset(key string) {
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}

// RetryAfter mengembalikan sisa waktu kunci, nol bila tidak terkunci.
func (l *LoginLimiter) RetryAfter(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[key]
	if !ok {
		return 0
	}
	if remaining := entry.resetAt.Sub(l.now()); remaining > 0 {
		return remaining
	}
	return 0
}
