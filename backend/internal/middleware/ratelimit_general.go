package middleware

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/pkg/apperror"
)

// Batas default pembatas laju sesuai plans/21 D4.
const (
	defaultRateLimit  = 120
	defaultRateWindow = time.Minute
)

// RateLimiter membatasi jumlah permintaan per kunci dalam satu jendela waktu.
// Disimpan di memori karena backend-api berjalan sebagai satu proses.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	limit   int
	window  time.Duration
	now     func() time.Time
}

type rateBucket struct {
	count   int
	resetAt time.Time
}

// NewRateLimiter membuat pembatas dengan batas dan jendela tertentu.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = defaultRateLimit
	}
	if window <= 0 {
		window = defaultRateWindow
	}
	return &RateLimiter{
		buckets: make(map[string]*rateBucket),
		limit:   limit,
		window:  window,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// allow mencatat satu permintaan dan mengembalikan sisa waktu tunggu.
func (l *RateLimiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	bucket, ok := l.buckets[key]
	if !ok || now.After(bucket.resetAt) {
		l.buckets[key] = &rateBucket{count: 1, resetAt: now.Add(l.window)}
		return true, 0
	}

	bucket.count++
	if bucket.count > l.limit {
		return false, bucket.resetAt.Sub(now)
	}
	return true, 0
}

// sweep membuang entri kedaluwarsa agar peta tidak tumbuh tanpa batas.
func (l *RateLimiter) sweep() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for key, bucket := range l.buckets {
		if now.After(bucket.resetAt) {
			delete(l.buckets, key)
		}
	}
}

// Middleware membatasi permintaan per kunci yang dihasilkan keyFunc.
// Nilai "retry after" dikirim lewat header agar klien tahu kapan mencoba lagi.
func (l *RateLimiter) Middleware(keyFunc func(*gin.Context) string) gin.HandlerFunc {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			l.sweep()
		}
	}()

	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			c.Next()
			return
		}

		ok, retryAfter := l.allow(key)
		if !ok {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			c.Error(apperror.ErrTooManyRequests)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitByUser membatasi per akun; jatuh ke IP bila belum terautentikasi.
func RateLimitByUser(l *RateLimiter) gin.HandlerFunc {
	return l.Middleware(func(c *gin.Context) string {
		if userID := c.GetString(ContextUserID); userID != "" {
			return "u:" + userID
		}
		return "ip:" + c.ClientIP()
	})
}

// RateLimitByIP membatasi per alamat IP, dipakai endpoint publik.
func RateLimitByIP(l *RateLimiter) gin.HandlerFunc {
	return l.Middleware(func(c *gin.Context) string {
		return "ip:" + c.ClientIP()
	})
}

// RejectTooMany menandai balasan 429 beserta sisa detik.
func RejectTooMany(c *gin.Context, retryAfter time.Duration) {
	c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
	c.Error(apperror.ErrTooManyRequests)
	c.Abort()
}
