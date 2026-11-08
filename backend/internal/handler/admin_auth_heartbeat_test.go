package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/handler"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/internal/session"
)

// stubAdminAuth adalah AdminAuthService yang tidak dipakai heartbeat.
type stubAdminAuth struct{}

func (stubAdminAuth) Login(context.Context, dto.LoginRequest, service.LoginContext) (*dto.AdminLoginResponse, error) {
	return nil, nil
}

func (stubAdminAuth) VerifyTwoFactor(context.Context, dto.VerifyTwoFactorRequest, service.LoginContext) (*dto.AdminLoginResponse, error) {
	return nil, nil
}

func (stubAdminAuth) Profile(context.Context, string) (*dto.AdminProfileResponse, error) {
	return nil, nil
}

func newHeartbeatRouter(cfg session.Config) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Tanpa ErrorHandler, c.Error tidak diterjemahkan menjadi status HTTP.
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	h := handler.NewAdminAuthHandler(stubAdminAuth{}, nil, cfg, "ila_admin_csrf", 2*time.Hour, 10*time.Minute)
	r.POST("/heartbeat", h.Heartbeat)
	return r
}

// Heartbeat tanpa sesi harus ditolak: ia memperpanjang umur cookie.
func TestHeartbeatTanpaSesiDitolak(t *testing.T) {
	cfg := session.NewConfig("ila_admin_session", "ila_admin_csrf", false, "")
	r := newHeartbeatRouter(cfg)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/heartbeat", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d; ingin 401", w.Code)
	}
}

// Heartbeat dengan sesi sah menulis ulang cookie dan mengembalikan batas diam.
func TestHeartbeatMemperpanjangCookie(t *testing.T) {
	cfg := session.NewConfig("ila_admin_session", "ila_admin_csrf", false, "")
	r := newHeartbeatRouter(cfg)

	req := httptest.NewRequest(http.MethodPost, "/heartbeat", nil)
	req.AddCookie(&http.Cookie{Name: cfg.SessionName, Value: "token-lama"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; ingin 200 (%s)", w.Code, w.Body.String())
	}

	var found *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == cfg.SessionName {
			found = cookie
		}
	}
	if found == nil {
		t.Fatal("cookie sesi tidak ditulis ulang")
	}
	if found.Value != "token-lama" {
		t.Fatalf("token tidak boleh berubah, dapat %q", found.Value)
	}
	if !found.HttpOnly {
		t.Fatal("cookie sesi harus HttpOnly")
	}
	if time.Until(found.Expires) < 90*time.Minute {
		t.Fatalf("umur cookie terlalu pendek: %v", time.Until(found.Expires))
	}

	var body struct {
		Data struct {
			IdleMinutes float64 `json:"idle_minutes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("balasan bukan JSON: %v", err)
	}
	if body.Data.IdleMinutes != 10 {
		t.Fatalf("idle_minutes = %v; ingin 10", body.Data.IdleMinutes)
	}
}
