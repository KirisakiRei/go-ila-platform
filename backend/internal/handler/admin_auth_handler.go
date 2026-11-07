package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/internal/session"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/httpx"
)

// AdminAuthHandler adalah pintu HTTP masuk pintu admin.
type AdminAuthHandler struct {
	auth     service.AdminAuthService
	limiter  LoginLimiter
	cookies  session.Config
	csrfName string
	expiry   time.Duration
}

// NewAdminAuthHandler merangkai handler autentikasi admin.
func NewAdminAuthHandler(
	auth service.AdminAuthService,
	limiter LoginLimiter,
	cookies session.Config,
	csrfName string,
	expiry time.Duration,
) *AdminAuthHandler {
	return &AdminAuthHandler{
		auth:     auth,
		limiter:  limiter,
		cookies:  cookies,
		csrfName: csrfName,
		expiry:   expiry,
	}
}

// Login menangani POST /api/v1/admin/auth/login.
//
// Batas percobaan dihitung per IP + email, sama seperti pintu user: penyerang
// tidak bisa mengunci akun admin dari jaringan lain (plans/12 §4.1).
func (h *AdminAuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	key := "admin|" + c.ClientIP() + "|" + req.Email
	if h.limiter != nil && h.limiter.Blocked(key) {
		c.Error(apperror.ErrTooManyRequests)
		return
	}

	result, err := h.auth.Login(c.Request.Context(), req, service.LoginContext{
		UserAgent: c.Request.UserAgent(),
	})
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
	h.cookies.SetSession(c, result.Token, expiresAt)
	if _, err := h.cookies.RotateCSRF(c, expiresAt); err != nil {
		c.Error(err)
		return
	}

	result.Token = ""
	httpx.OK(c, "Berhasil masuk", result)
}

// Logout menangani POST /api/v1/admin/auth/logout.
func (h *AdminAuthHandler) Logout(c *gin.Context) {
	h.cookies.Clear(c)
	httpx.OK(c, "Berhasil keluar", gin.H{"logged_out": true})
}

// Profile menangani GET /api/v1/admin/auth/me.
func (h *AdminAuthHandler) Profile(c *gin.Context) {
	result, err := h.auth.Profile(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Profil admin", result)
}

// CSRF menerbitkan ulang nilai CSRF untuk pintu admin.
func (h *AdminAuthHandler) CSRF(c *gin.Context) {
	value, err := h.cookies.RotateCSRF(c, time.Now().UTC().Add(h.expiry))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Token CSRF", gin.H{"csrf_token": value})
}
