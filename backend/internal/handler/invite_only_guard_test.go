package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/internal/session"
	"ila/backend/pkg/apperror"
)

// gateTetap adalah AccessGate dengan jawaban tetap.
type gateTetap bool

func (g gateTetap) InviteOnly(context.Context) bool { return bool(g) }

// authServiceTiruan mencatat apakah Register benar-benar dipanggil.
type authServiceTiruan struct {
	dipanggil bool
}

func (a *authServiceTiruan) Register(context.Context, dto.RegisterRequest) (*dto.UserResponse, error) {
	a.dipanggil = true
	return &dto.UserResponse{}, nil
}

func (a *authServiceTiruan) Login(context.Context, dto.LoginRequest) (*dto.LoginResponse, error) {
	return nil, nil
}

func (a *authServiceTiruan) Profile(context.Context, string) (*dto.UserResponse, error) {
	return nil, nil
}

func (a *authServiceTiruan) ChangePassword(context.Context, string, dto.ChangePasswordRequest) error {
	return nil
}

func (a *authServiceTiruan) LogoutAll(context.Context, string) error { return nil }

func (a *authServiceTiruan) SetIdleWarning(context.Context, string, bool) error { return nil }

func (a *authServiceTiruan) VerifyTwoFactor(context.Context, dto.VerifyTwoFactorRequest, service.LoginContext) (*dto.LoginResponse, error) {
	return nil, nil
}

// newAuthRouter memasang rute register pada engine gin kecil.
func newAuthRouter(gate AccessGate) (*gin.Engine, *authServiceTiruan) {
	gin.SetMode(gin.TestMode)
	svc := &authServiceTiruan{}
	h := NewAuthHandler(svc, nil, nil, session.NewConfig("ila_session", "ila_csrf", false, ""), time.Hour)
	h.SetAccessGate(gate)

	r := gin.New()
	// Tanpa ErrorHandler, c.Error tidak diterjemahkan menjadi status HTTP.
	r.Use(middleware.ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.POST("/api/v1/auth/register", h.Register)
	return r, svc
}

func postRegisterJSON(r http.Handler) *httptest.ResponseRecorder {
	body := `{"name":"Uji Coba","email":"uji@contoh.id","password":"rahasia-panjang-123","accepted_terms":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestRegisterDitolakSaatModeUndangan mengunci kontrak utama: selama invite
// only, pendaftaran mandiri ditolak 403 dan layanan tidak pernah dipanggil.
func TestRegisterDitolakSaatModeUndangan(t *testing.T) {
	r, svc := newAuthRouter(gateTetap(true))
	w := postRegisterJSON(r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d; ingin 403 (body: %s)", w.Code, w.Body.String())
	}
	if svc.dipanggil {
		t.Fatal("layanan Register dipanggil padahal mode undangan menutup pendaftaran")
	}
	if !strings.Contains(w.Body.String(), "undangan") {
		t.Fatalf("balasan tidak menjelaskan jalur undangan: %s", w.Body.String())
	}
}

// TestRegisterBerjalanSaatGerbangTerbuka: membuka flag harus mengembalikan
// alur lama tanpa mengubah handler.
func TestRegisterBerjalanSaatGerbangTerbuka(t *testing.T) {
	r, svc := newAuthRouter(gateTetap(false))
	postRegisterJSON(r)

	if !svc.dipanggil {
		t.Fatal("layanan Register tidak dipanggil padahal gerbang terbuka")
	}
}

// TestErrInviteOnlyMemetakan403 memastikan error gerbang menghasilkan 403,
// bukan 500, di seluruh pintu yang memakainya.
func TestErrInviteOnlyMemetakan403(t *testing.T) {
	status, code := apperror.MapHTTPStatus(service.ErrInviteOnly)
	if status != http.StatusForbidden || code != "FORBIDDEN" {
		t.Fatalf("peta = %d/%s; ingin 403/FORBIDDEN", status, code)
	}
}
