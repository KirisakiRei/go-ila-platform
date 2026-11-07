package middleware_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/middleware"
)

// limiterRouter memasang pembatas laju pada endpoint permisif.
func limiterRouter(limit int, window time.Duration) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.Use(middleware.RateLimitByIP(middleware.NewRateLimiter(limit, window)))
	r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func hit(r *gin.Engine) int {
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// Permintaan melebihi batas harus ditolak 429, bukan dilayani terus.
func TestRateLimitMenolakSetelahBatas(t *testing.T) {
	r := limiterRouter(3, time.Minute)

	for i := 0; i < 3; i++ {
		if code := hit(r); code != http.StatusOK {
			t.Fatalf("permintaan ke-%d = %d, ingin 200", i+1, code)
		}
	}
	if code := hit(r); code != http.StatusTooManyRequests {
		t.Fatalf("permintaan ke-4 = %d, ingin 429", code)
	}
}

// Jendela yang lewat mengembalikan kuota.
func TestRateLimitPulihSetelahJendela(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := middleware.NewRateLimiter(1, 50*time.Millisecond)

	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.Use(middleware.RateLimitByIP(limiter))
	r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })

	if code := hit(r); code != http.StatusOK {
		t.Fatalf("pertama = %d, ingin 200", code)
	}
	if code := hit(r); code != http.StatusTooManyRequests {
		t.Fatalf("kedua = %d, ingin 429", code)
	}

	time.Sleep(60 * time.Millisecond)

	if code := hit(r); code != http.StatusOK {
		t.Fatalf("setelah jendela = %d, ingin 200", code)
	}
}
