package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authstate"
	"ila/backend/pkg/utils"
)

// stubChecker menjawab pertanyaan permission dari daftar tetap.
type stubChecker struct {
	allowed map[string]bool
	err     error
}

func (s stubChecker) Has(_ context.Context, _, permission string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.allowed[permission], nil
}

// stubReader membaca satu akun tetap.
type stubReader struct {
	user *authstate.User
	err  error
}

func (s stubReader) FindSessionUser(context.Context, string) (*authstate.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

func newAdminRouter(issuer *utils.JWTIssuer, reader authstate.Reader, checker PermissionChecker, permission string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.GET("/api/v1/admin/probe", AdminScope(issuer, "ila_admin_session", reader), RequirePermission(checker, permission), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

// Penolakan tanpa sesi harus 401, bukan 403.
func TestAdminTanpaSesi401(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	r := newAdminRouter(issuer, stubReader{user: activeAdmin()}, stubChecker{allowed: map[string]bool{}}, "role.view")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/probe", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; ingin 401", w.Code)
	}
}

// Token user tidak boleh melewati pintu admin.
func TestSesiUserDitolakDiPintuAdmin(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, err := issuer.IssueWithClaims(utils.Claims{UserID: "u1", Role: "user", Scope: utils.ScopeUser, Session: 1})
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	r := newAdminRouter(issuer, stubReader{user: activeAdmin()}, stubChecker{allowed: map[string]bool{}}, "role.view")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/probe", nil)
	req.AddCookie(&http.Cookie{Name: "ila_admin_session", Value: token})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; ingin 403", w.Code)
	}
}

// Sesi admin tanpa permission yang diminta harus 403.
func TestAdminTanpaPermission403(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, err := issuer.IssueWithClaims(utils.Claims{UserID: "a1", Role: "admin", Scope: utils.ScopeAdmin, Session: 1})
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	r := newAdminRouter(issuer, stubReader{user: activeAdmin()}, stubChecker{allowed: map[string]bool{}}, "role.view")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/probe", nil)
	req.AddCookie(&http.Cookie{Name: "ila_admin_session", Value: token})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; ingin 403", w.Code)
	}
	// Pesan tidak boleh menyebut permission mana yang kurang.
	if body := w.Body.String(); contains(body, "role.view") {
		t.Fatalf("balasan membocorkan nama permission: %s", body)
	}
}

// Sesi admin dengan permission yang diminta boleh lewat.
func TestAdminDenganPermissionLolos(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, err := issuer.IssueWithClaims(utils.Claims{UserID: "a1", Role: "admin", Scope: utils.ScopeAdmin, Session: 1})
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	r := newAdminRouter(issuer, stubReader{user: activeAdmin()}, stubChecker{allowed: map[string]bool{"role.view": true}}, "role.view")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/probe", nil)
	req.AddCookie(&http.Cookie{Name: "ila_admin_session", Value: token})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; ingin 200 (body: %s)", w.Code, w.Body.String())
	}
}

// Session version yang tidak cocok langsung memutus akses admin.
func TestAdminSesiDicabut401(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, err := issuer.IssueWithClaims(utils.Claims{UserID: "a1", Role: "admin", Scope: utils.ScopeAdmin, Session: 1})
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	user := activeAdmin()
	user.SessionVersion = 2
	r := newAdminRouter(issuer, stubReader{user: user}, stubChecker{allowed: map[string]bool{"role.view": true}}, "role.view")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/probe", nil)
	req.AddCookie(&http.Cookie{Name: "ila_admin_session", Value: token})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; ingin 401", w.Code)
	}
}

// Admin yang di-ban tidak boleh memakai pintu admin meski tokennya belum habis.
func TestAdminDiBanDitolak(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, err := issuer.IssueWithClaims(utils.Claims{UserID: "a1", Role: "admin", Scope: utils.ScopeAdmin, Session: 1})
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	user := activeAdmin()
	user.Status = authstate.StatusBanned
	r := newAdminRouter(issuer, stubReader{user: user}, stubChecker{allowed: map[string]bool{"role.view": true}}, "role.view")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/probe", nil)
	req.AddCookie(&http.Cookie{Name: "ila_admin_session", Value: token})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; ingin 403", w.Code)
	}
}

// Route admin tanpa deklarasi permission harus terdeteksi.
func TestUndeclaredAdminRoutes(t *testing.T) {
	adminRoutes.mu.Lock()
	adminRoutes.items = make(map[string]string)
	adminRoutes.mu.Unlock()

	routes := []gin.RouteInfo{
		{Method: http.MethodGet, Path: "/api/v1/admin/roles"},
		{Method: http.MethodPost, Path: "/api/v1/admin/roles"},
		{Method: http.MethodGet, Path: "/api/v1/me"},
	}

	missing := UndeclaredAdminRoutes(routes)
	if len(missing) != 2 {
		t.Fatalf("missing = %v; ingin 2 route", missing)
	}

	RegisterAdminRoute(http.MethodGet, "/api/v1/admin/roles", "role.view")
	RegisterAdminRoute(http.MethodPost, "/api/v1/admin/roles", "role.edit")

	if missing := UndeclaredAdminRoutes(routes); len(missing) != 0 {
		t.Fatalf("setelah deklarasi masih ada yang bolong: %v", missing)
	}

	adminRoutes.mu.Lock()
	adminRoutes.items = make(map[string]string)
	adminRoutes.mu.Unlock()
}

// Route login admin memang tidak butuh permission dan tidak boleh dilaporkan.
func TestAuthMeTidakDianggapBolong(t *testing.T) {
	adminRoutes.mu.Lock()
	adminRoutes.items = make(map[string]string)
	adminRoutes.mu.Unlock()

	missing := UndeclaredAdminRoutes([]gin.RouteInfo{
		{Method: http.MethodGet, Path: "/api/v1/admin/auth/me"},
		{Method: http.MethodPost, Path: "/api/v1/admin/auth/login"},
	})
	if len(missing) != 0 {
		t.Fatalf("missing = %v; ingin kosong", missing)
	}

	adminRoutes.mu.Lock()
	adminRoutes.items = make(map[string]string)
	adminRoutes.mu.Unlock()
}

// Kegagalan pembacaan permission bukan 403: harus diteruskan sebagai error.
func TestErrorCheckerDiteruskan(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, err := issuer.IssueWithClaims(utils.Claims{UserID: "a1", Role: "admin", Scope: utils.ScopeAdmin, Session: 1})
	if err != nil {
		t.Fatalf("Issue error: %v", err)
	}

	sentinel := errors.New("database mati")
	r := newAdminRouter(issuer, stubReader{user: activeAdmin()}, stubChecker{err: sentinel}, "role.view")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/probe", nil)
	req.AddCookie(&http.Cookie{Name: "ila_admin_session", Value: token})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d; ingin 500", w.Code)
	}
}

func activeAdmin() *authstate.User {
	return &authstate.User{ID: "a1", Role: "admin", Status: authstate.StatusActive, SessionVersion: 1}
}

func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
