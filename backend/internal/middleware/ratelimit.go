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
