package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// EmailAccountHandler adalah pintu HTTP alur akun: lupa sandi, verifikasi,
// onboarding, dan banner langganan.
type EmailAccountHandler struct {
	flows service.EmailAccountService
}

// NewEmailAccountHandler merangkai handler alur akun.
func NewEmailAccountHandler(flows service.EmailAccountService) *EmailAccountHandler {
	return &EmailAccountHandler{flows: flows}
}

// RegisterPublic memasang route tanpa sesi.
func (h *EmailAccountHandler) RegisterPublic(g *gin.RouterGroup, limiter gin.HandlerFunc) {
	// Pesan sukses tetap sama apa pun hasilnya: alamat surel tidak boleh
	// bisa dipetakan dari perbedaan balasan (plans/17 §3.2).
	g.POST("/forgot-password", limiter, h.ForgotPassword)
	g.POST("/reset-password", limiter, h.ResetPassword)
	g.POST("/verify-email", limiter, h.VerifyEmail)
}

// RegisterUser memasang route yang butuh sesi.
func (h *EmailAccountHandler) RegisterUser(g *gin.RouterGroup) {
	g.POST("/me/verify-email/resend", h.ResendVerification)
	g.GET("/me/onboarding", h.Onboarding)
	g.PUT("/me/onboarding", h.SaveOnboarding)
	g.GET("/me/subscription-banner", h.Banner)
}

// ForgotPassword menangani POST /api/v1/auth/forgot-password.
func (h *EmailAccountHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.flows.ForgotPassword(c.Request.Context(), req.Email, c.ClientIP()); err != nil {
		// Kegagalan internal tetap dibalas sama agar tidak membocorkan status.
		httpx.OK(c, "Bila alamat terdaftar, tautan reset sudah dikirim", nil)
		return
	}
	httpx.OK(c, "Bila alamat terdaftar, tautan reset sudah dikirim", nil)
}

// ResetPassword menangani POST /api/v1/auth/reset-password.
func (h *EmailAccountHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.flows.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kata sandi diperbarui. Semua perangkat telah dikeluarkan.", nil)
}

// VerifyEmail menangani POST /api/v1/auth/verify-email.
func (h *EmailAccountHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.flows.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Email berhasil diverifikasi", nil)
}

// ResendVerification menangani POST /api/v1/me/verify-email/resend.
func (h *EmailAccountHandler) ResendVerification(c *gin.Context) {
	if err := h.flows.ResendVerification(c.Request.Context(), middleware.CurrentUserID(c)); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Tautan verifikasi baru sudah dikirim", nil)
}

// Onboarding menangani GET /api/v1/me/onboarding.
func (h *EmailAccountHandler) Onboarding(c *gin.Context) {
	result, err := h.flows.Onboarding(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Status onboarding", result)
}

// SaveOnboarding menangani PUT /api/v1/me/onboarding.
func (h *EmailAccountHandler) SaveOnboarding(c *gin.Context) {
	var req dto.OnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.flows.SaveOnboarding(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Onboarding disimpan", result)
}

// Banner menangani GET /api/v1/me/subscription-banner.
func (h *EmailAccountHandler) Banner(c *gin.Context) {
	result, err := h.flows.Banner(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Status langganan", result)
}

// pastikan import http dipakai untuk dokumentasi rute.
var _ = http.MethodPost
