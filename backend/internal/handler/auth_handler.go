package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/middleware"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/session"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/httpx"
)

// AuthHandler adalah pintu HTTP untuk pendaftaran, masuk, dan profil akun.
type AuthHandler struct {
	authService service.AuthService
	limiter     LoginLimiter
	cookies     session.Config
	jwtExpiry   time.Duration
}

// NewAuthHandler merangkai handler autentikasi.
func NewAuthHandler(
	authService service.AuthService,
	limiter LoginLimiter,
	cookies session.Config,
	jwtExpiry time.Duration,
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		limiter:     limiter,
		cookies:     cookies,
		jwtExpiry:   jwtExpiry,
	}
}

// Register menangani POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}

	httpx.Created(c, "Akun berhasil dibuat", result)
}

// Login menangani POST /api/v1/auth/login dengan batas percobaan.
//
// Batas dihitung per email + IP: penyerang tidak bisa mengunci akun orang lain
// hanya dengan menebak dari jaringannya sendiri (plans/12 par 4.1).
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	key := c.ClientIP() + "|" + req.Email
	if h.limiter != nil && h.limiter.Blocked(key) {
		c.Error(apperror.ErrTooManyRequests)
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		if h.limiter != nil {
			h.limiter.Fail(key)
		}
		c.Error(err)
		return
	}

	if h.limiter != nil {
		h.limiter.Reset(key)
	}

	expiresAt, err := time.Parse(time.RFC3339, result.ExpiresAt)
	if err != nil {
		c.Error(err)
		return
	}
	if err := h.issueCookies(c, result.Token, expiresAt); err != nil {
		c.Error(err)
		return
	}

	// Token tidak ikut dikirim: satu-satunya tempat menyimpannya adalah cookie.
	result.Token = ""
	httpx.OK(c, "Berhasil masuk", result)
}

// Logout menangani POST /api/v1/auth/logout dengan menghapus kedua cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	h.cookies.Clear(c)
	httpx.OK(c, "Berhasil keluar", gin.H{"logged_out": true})
}

// CSRF menangani GET /api/v1/auth/csrf untuk menerbitkan token bagi klien
// yang membuka aplikasi dengan sesi yang masih berlaku.
func (h *AuthHandler) CSRF(c *gin.Context) {
	value, err := h.cookies.RotateCSRF(c, time.Now().UTC().Add(h.jwtExpiry))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Token CSRF", gin.H{"csrf_token": value})
}

// Profile menangani GET /api/v1/me.
func (h *AuthHandler) Profile(c *gin.Context) {
	result, err := h.authService.Profile(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		if err == apperror.ErrNotFound {
			c.Error(apperror.ErrUnauthorized)
			return
		}
		c.Error(err)
		return
	}

	httpx.OK(c, "Profil akun", result)
}

func (h *AuthHandler) issueCookies(c *gin.Context, token string, expiresAt time.Time) error {
	h.cookies.SetSession(c, token, expiresAt)
	_, err := h.cookies.RotateCSRF(c, expiresAt)
	return err
}
