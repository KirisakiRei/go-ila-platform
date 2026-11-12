package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/internal/session"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/httpx"
)

// AuthHandler adalah pintu HTTP untuk pendaftaran, masuk, dan profil akun.
type AuthHandler struct {
	authService service.AuthService
	limiter     LoginLimiter
	// registerLimiter opsional; nil berarti hanya batas per IP middleware.
	registerLimiter RegisterLimiter
	// accessGate memutuskan apakah pendaftaran mandiri masih terbuka.
	// Nil berarti terbuka — hanya untuk pemanggil lama dan pengujian.
	accessGate AccessGate
	// invites dipakai saat pintu mandiri tertutup: satu-satunya jalan masuk
	// adalah tautan undangan. Nil berarti jalur undangan tidak tersedia.
	invites   InviteRegistrar
	cookies   session.Config
	jwtExpiry time.Duration
}

// InviteRegistrar memakai satu undangan lalu membuat akun.
type InviteRegistrar interface {
	Klaim(ctx context.Context, code string, req dto.RegisterRequest) (*dto.UserResponse, error)
}

// AccessGate melaporkan apakah pendaftaran mandiri masih dibuka.
//
// Dipisahkan dari settings.Service agar handler tidak bergantung pada
// pembacaan basis data dan tetap mudah diuji.
type AccessGate interface {
	// InviteOnly bernilai true bila akun baru hanya boleh lahir dari undangan admin.
	InviteOnly(ctx context.Context) bool
}

// ErrPendaftaranTertutup adalah penolakan pendaftaran saat mode undangan.
var ErrPendaftaranTertutup = errors.Join(
	apperror.ErrForbidden,
	errors.New("Pendaftaran hanya lewat undangan. Hubungi admin untuk akun."),
)

// NewAuthHandler merangkai handler autentikasi.
func NewAuthHandler(
	authService service.AuthService,
	limiter LoginLimiter,
	registerLimiter RegisterLimiter,
	cookies session.Config,
	jwtExpiry time.Duration,
) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		limiter:         limiter,
		registerLimiter: registerLimiter,
		cookies:         cookies,
		jwtExpiry:       jwtExpiry,
	}
}

// SetAccessGate memasang penjaga pendaftaran. Dipanggil saat perakitan router
// agar konstruktor lama tetap sah.
func (h *AuthHandler) SetAccessGate(gate AccessGate) {
	h.accessGate = gate
}

// SetInviteRegistrar memasang jalur pendaftaran lewat undangan.
func (h *AuthHandler) SetInviteRegistrar(reg InviteRegistrar) {
	h.invites = reg
}

// gateTertutup melaporkan apakah pendaftaran mandiri sedang ditutup.
// Tanpa gerbang (pemanggil lama), pintu dianggap terbuka.
func (h *AuthHandler) gateTertutup(ctx context.Context) bool {
	if h.accessGate == nil {
		return false
	}
	return h.accessGate.InviteOnly(ctx)
}

// Register menangani POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	// Selama gate tertutup, satu-satunya jalan masuk adalah tautan undangan.
	// Limiter per IP/email sengaja TIDAK dipakai di jalur ini: anggota komunitas
	// sering berbagi satu IP, dan batas sesungguhnya adalah kuota + masa berlaku
	// pada undangan itu sendiri.
	if h.gateTertutup(c.Request.Context()) {
		code := strings.TrimSpace(req.InviteCode)
		if code == "" {
			c.Error(ErrPendaftaranTertutup)
			return
		}
		if h.invites == nil {
			c.Error(ErrPendaftaranTertutup)
			return
		}
		result, err := h.invites.Klaim(c.Request.Context(), code, req)
		if err != nil {
			c.Error(err)
			return
		}
		httpx.Created(c, "Akun berhasil dibuat lewat undangan", result)
		return
	}

	// Pendaftaran massal dari satu jaringan membebani database dan mengotori
	// data trial. Batasnya diperiksa sebelum akun dibuat.
	if h.registerLimiter != nil {
		email := strings.ToLower(strings.TrimSpace(req.Email))
		if ok, retry := h.registerLimiter.Allow(c.ClientIP(), email); !ok {
			c.Header("Retry-After", strconv.Itoa(retry))
			c.Error(errors.Join(apperror.ErrTooManyRequests,
				errors.New("Terlalu banyak pendaftaran dari jaringan ini. Coba lagi nanti.")))
			return
		}
	}

	result, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}

	if h.registerLimiter != nil {
		h.registerLimiter.Record(c.ClientIP(), strings.ToLower(strings.TrimSpace(req.Email)))
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

	key := c.ClientIP() + "|" + strings.ToLower(req.Identifier)
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

// VerifyTwoFactor menangani POST /api/v1/auth/verify-2fa.
// Cookie sesi baru ditulis setelah kode terbukti sah.
func (h *AuthHandler) VerifyTwoFactor(c *gin.Context) {
	var req dto.VerifyTwoFactorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.authService.VerifyTwoFactor(c.Request.Context(), req, service.LoginContext{
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		c.Error(err)
		return
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

// SetIdleWarning menangani PATCH /api/v1/me/idle-warning.
func (h *AuthHandler) SetIdleWarning(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.authService.SetIdleWarning(c.Request.Context(), middleware.CurrentUserID(c), *req.Enabled); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Preferensi peringatan diam diperbarui", gin.H{"idle_warning": *req.Enabled})
}

// ChangePassword menangani POST /api/v1/me/password.
// Semua sesi dicabut, jadi cookie perangkat ini ikut dibersihkan.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	if err := h.authService.ChangePassword(c.Request.Context(), middleware.CurrentUserID(c), req); err != nil {
		c.Error(err)
		return
	}

	h.cookies.Clear(c)
	httpx.OK(c, "Password diperbarui. Silakan masuk lagi.", gin.H{"re_login_required": true})
}

// LogoutAll menangani POST /api/v1/me/logout-all.
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	if err := h.authService.LogoutAll(c.Request.Context(), middleware.CurrentUserID(c)); err != nil {
		c.Error(err)
		return
	}

	h.cookies.Clear(c)
	httpx.OK(c, "Berhasil keluar dari semua perangkat", gin.H{"logged_out": true})
}

func (h *AuthHandler) issueCookies(c *gin.Context, token string, expiresAt time.Time) error {
	h.cookies.SetSession(c, token, expiresAt)
	_, err := h.cookies.RotateCSRF(c, expiresAt)
	return err
}
