package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newTestContext(t *testing.T, method string, headers map[string]string, cookies []*http.Cookie) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, "/api/v1/me", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = req
	return ctx, rec
}

func TestSetSessionMenulisCookieHttpOnly(t *testing.T) {
	cfg := NewConfig("ila_session", "ila_csrf", true, "")
	ctx, rec := newTestContext(t, http.MethodPost, nil, nil)

	cfg.SetSession(ctx, "token-abc", time.Now().Add(time.Hour))

	cookie := findCookie(t, rec.Result().Cookies(), "ila_session")
	if !cookie.HttpOnly {
		t.Fatal("cookie sesi wajib HttpOnly agar tidak terbaca JavaScript")
	}
	if !cookie.Secure {
		t.Fatal("cookie sesi wajib Secure")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("SameSite = %v, ingin Strict", cookie.SameSite)
	}
	if cookie.Value != "token-abc" {
		t.Fatalf("nilai cookie = %q", cookie.Value)
	}
}

func TestCSRFValidMenolakHeaderKosongAtauBeda(t *testing.T) {
	cases := []struct {
		name    string
		cookie  string
		header  string
		want    bool
		comment string
	}{
		{"tanpa cookie", "", "abc", false, "cookie hilang berarti bukan request sah"},
		{"tanpa header", "abc", "", false, "header hilang adalah ciri serangan CSRF"},
		{"berbeda", "abc", "abd", false, "nilai tidak cocok"},
		{"cocok", "abc", "abc", true, "double-submit terpenuhi"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newTestContext(t, http.MethodPost, map[string]string{}, nil)
			if tc.header != "" {
				ctx.Request.Header.Set(CookieHeader, tc.header)
			}
			if tc.cookie != "" {
				ctx.Request.AddCookie(&http.Cookie{Name: "ila_csrf", Value: tc.cookie})
			}

			if got := CSRFValid(ctx, "ila_csrf"); got != tc.want {
				t.Fatalf("CSRFValid = %v, ingin %v (%s)", got, tc.want, tc.comment)
			}
		})
	}
}

func TestRotateCSRFMenghasilkanNilaiBerbeda(t *testing.T) {
	cfg := NewConfig("ila_session", "ila_csrf", false, "")
	ctx1, rec1 := newTestContext(t, http.MethodGet, nil, nil)
	ctx2, rec2 := newTestContext(t, http.MethodGet, nil, nil)

	first, err := cfg.RotateCSRF(ctx1, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("RotateCSRF: %v", err)
	}
	second, err := cfg.RotateCSRF(ctx2, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("RotateCSRF: %v", err)
	}

	if first == second {
		t.Fatal("token CSRF tidak boleh berulang")
	}

	// Cookie CSRF harus terbaca JavaScript agar bisa dikirim sebagai header.
	if cookie := findCookie(t, rec1.Result().Cookies(), "ila_csrf"); cookie.HttpOnly {
		t.Fatal("cookie CSRF tidak boleh HttpOnly")
	}
	if cookie := findCookie(t, rec2.Result().Cookies(), "ila_csrf"); cookie.Value != second {
		t.Fatalf("nilai cookie = %q, ingin %q", cookie.Value, second)
	}
}

func TestClearMenghapusKeduaCookie(t *testing.T) {
	cfg := NewConfig("ila_session", "ila_csrf", false, "")
	ctx, rec := newTestContext(t, http.MethodPost, nil, nil)

	cfg.Clear(ctx)

	for _, name := range []string{"ila_session", "ila_csrf"} {
		cookie := findCookie(t, rec.Result().Cookies(), name)
		if cookie.MaxAge >= 0 {
			t.Fatalf("cookie %s tidak dihapus (MaxAge=%d)", name, cookie.MaxAge)
		}
	}
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s tidak ditemukan", name)
	return nil
}
