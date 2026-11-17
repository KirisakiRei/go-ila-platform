package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSafePathMenyamarkanTokenApprove(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "token saja",
			input: "/api/v1/approve/RAHASIA123",
			want:  "/api/v1/approve/<disembunyikan>",
		},
		{
			name:  "dengan sub-path confirm",
			input: "/api/v1/approve/RAHASIA123/confirm",
			want:  "/api/v1/approve/<disembunyikan>/confirm",
		},
		{
			name:  "dengan sub-path decline",
			input: "/api/v1/approve/RAHASIA123/decline",
			want:  "/api/v1/approve/<disembunyikan>/decline",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := safePath(tc.input); got != tc.want {
				t.Fatalf("safePath(%q) = %q, ingin %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSafePathTidakMengubahPathBiasa(t *testing.T) {
	for _, path := range []string{
		"/api/v1/me",
		"/api/v1/accounts",
		"/api/v1/subscriptions/abc",
		"/health",
		"/approve/",
	} {
		if got := safePath(path); got != path {
			t.Fatalf("safePath(%q) berubah menjadi %q", path, got)
		}
	}
}

func TestRequestIDMenolakHeaderTerlaluPanjangAtauKotor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/cek", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(ContextRequestID))
	})

	// Header sah dipertahankan agar log Caddy dan backend-api tetap tersambung.
	reqSah := httptest.NewRequest(http.MethodGet, "/cek", nil)
	reqSah.Header.Set("X-Request-ID", "trace-sah_123:abc.1")
	wSah := httptest.NewRecorder()
	r.ServeHTTP(wSah, reqSah)
	if wSah.Body.String() != "trace-sah_123:abc.1" {
		t.Fatalf("X-Request-ID sah berubah menjadi %q", wSah.Body.String())
	}

	// Header > 64 karakter atau memuat karakter kontrol diganti ID baru <= 64 karakter.
	for _, buruk := range []string{strings.Repeat("a", 80), "jahat\r\nSet-Cookie: x=1", "spasi tidak sah"} {
		req := httptest.NewRequest(http.MethodGet, "/cek", nil)
		req.Header.Set("X-Request-ID", buruk)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		got := w.Body.String()
		if got == "" || got == buruk || len(got) > maxRequestIDLen {
			t.Fatalf("X-Request-ID buruk %q tidak diganti dengan aman: %q", buruk, got)
		}
	}
}
