package middleware_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authstate"
	"ila/backend/internal/middleware"
	"ila/backend/pkg/utils"
)

/*
Route yang dijaga permission tidak boleh membalas 200 tanpa isi.

Bug yang dijaga di sini terjadi di produksi: GET /admin/roles dan
/admin/permissions membalas status 200 dengan body 0 byte. Panel admin
menerjemahkannya sebagai "Balasan server tidak dapat dibaca", dan halaman RBAC
tidak bisa dibuka sama sekali. Route yang tidak memakai guard (mis. /2fa,
/plans) tetap normal, sehingga penyebabnya ada pada rangkaian penjaga
permission — bukan pada handler-nya.
*/
func TestRouteDenganGuardMenulisBalasan(t *testing.T) {
	gin.SetMode(gin.TestMode)

	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, err := issuer.IssueWithClaims(utils.Claims{
		UserID: "a1", Role: "admin", Scope: utils.ScopeAdmin, Session: 1,
	})
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))

	admin := r.Group("/api/v1/admin")
	admin.Use(middleware.AdminScope(issuer, "ila_admin_session", stubReader{}))
	admin.Use(middleware.RateLimitByUser(middleware.NewRateLimiter(0, 0)))
	admin.Use(middleware.RequireCSRF("ila_admin_csrf"))

	// Persis seperti guard di router: catat lalu pasang penjaga.
	guard := func(method, path, permission string) gin.HandlerFunc {
		middleware.RegisterAdminRoute(method, path, permission)
		return middleware.RequirePermission(stubChecker{allowed: true}, permission)
	}

	admin.GET("/roles", guard(http.MethodGet, "/api/v1/admin/roles", "role.view"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Daftar role", "data": []string{}})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/roles", nil)
	req.AddCookie(&http.Cookie{Name: "ila_admin_session", Value: token})
	req.AddCookie(&http.Cookie{Name: "ila_admin_csrf", Value: "token-csrf"})
	req.Header.Set("X-CSRF-Token", "token-csrf")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; ingin 200 (body: %q)", w.Code, w.Body.String())
	}
	if w.Body.Len() == 0 {
		t.Fatal("body kosong: route yang dijaga guard harus tetap mengirim balasan JSON")
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Error("Content-Type kosong pada balasan yang berisi")
	}
}

// stubReader meloloskan sesi admin yang masih aktif.
type stubReader struct{}

func (stubReader) FindSessionUser(context.Context, string) (*authstate.User, error) {
	return &authstate.User{ID: "a1", Role: "admin", Status: authstate.StatusActive, SessionVersion: 1}, nil
}

// stubChecker mengizinkan seluruh permission.
type stubChecker struct{ allowed bool }

func (s stubChecker) Has(context.Context, string, string) (bool, error) {
	return s.allowed, nil
}
