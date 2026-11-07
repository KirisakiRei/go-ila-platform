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

type fakeReader struct {
	account *authstate.User
	err     error
}

func (f fakeReader) FindSessionUser(context.Context, string) (*authstate.User, error) {
	return f.account, f.err
}

// newRouter memasang middleware Auth lalu endpoint yang selalu 200.
func newRouter(issuer *utils.JWTIssuer, reader authstate.Reader, token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.GET("/probe", middleware.Auth(issuer, "ila_session", reader), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func request(r *gin.Engine, token string) int {
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "ila_session", Value: token})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestSesiSahDiterima(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, _ := issuer.IssueWithClaims(utils.Claims{UserID: "u1", Role: "user", Scope: utils.ScopeUser, Session: 3})

	r := newRouter(issuer, fakeReader{account: &authstate.User{ID: "u1", Role: "user", Status: "active", SessionVersion: 3}}, token)
	if code := request(r, token); code != http.StatusOK {
		t.Fatalf("status = %d, ingin 200", code)
	}
}

// session_version yang berbeda berarti sesi sudah dicabut.
func TestSesiDicabutDitolak(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, _ := issuer.IssueWithClaims(utils.Claims{UserID: "u1", Role: "user", Scope: utils.ScopeUser, Session: 2})

	r := newRouter(issuer, fakeReader{account: &authstate.User{ID: "u1", Role: "user", Status: "active", SessionVersion: 3}}, token)
	if code := request(r, token); code != http.StatusUnauthorized {
		t.Fatalf("status = %d, ingin 401", code)
	}
}

// Akun ter-ban tidak boleh melewati middleware.
func TestAkunTerBanDitolak(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, _ := issuer.IssueWithClaims(utils.Claims{UserID: "u1", Role: "user", Scope: utils.ScopeUser, Session: 1})

	r := newRouter(issuer, fakeReader{account: &authstate.User{ID: "u1", Role: "user", Status: "banned", SessionVersion: 1}}, token)
	if code := request(r, token); code != http.StatusForbidden {
		t.Fatalf("status = %d, ingin 403", code)
	}
}

// Suspended masih boleh melihat data; eksekusi diblokir Runner.
func TestAkunDitangguhkanMasihBolehMasuk(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, _ := issuer.IssueWithClaims(utils.Claims{UserID: "u1", Role: "user", Scope: utils.ScopeUser, Session: 1})

	r := newRouter(issuer, fakeReader{account: &authstate.User{ID: "u1", Role: "user", Status: "suspended", SessionVersion: 1}}, token)
	if code := request(r, token); code != http.StatusOK {
		t.Fatalf("status = %d, ingin 200", code)
	}
}

// Token berscope admin tidak boleh dipakai di pintu user.
func TestScopeAdminDitolakDiPintuUser(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	token, _, _ := issuer.IssueWithClaims(utils.Claims{UserID: "a1", Role: "admin", Scope: utils.ScopeAdmin, Session: 1})

	r := newRouter(issuer, fakeReader{account: &authstate.User{ID: "a1", Role: "admin", Status: "active", SessionVersion: 1}}, token)
	if code := request(r, token); code != http.StatusUnauthorized {
		t.Fatalf("status = %d, ingin 401", code)
	}
}

// Tanpa cookie sama sekali harus ditolak.
func TestTanpaSesiDitolak(t *testing.T) {
	issuer := utils.NewJWTIssuer("rahasia-panjang-untuk-uji-1234567890", time.Hour)
	r := newRouter(issuer, nil, "")
	if code := request(r, ""); code != http.StatusUnauthorized {
		t.Fatalf("status = %d, ingin 401", code)
	}
}
