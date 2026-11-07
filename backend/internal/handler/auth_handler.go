package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/dto"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/middleware"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/httpx"
)

// AuthHandler adalah pintu HTTP untuk pendaftaran, masuk, dan profil akun.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler merangkai handler autentikasi.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
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

// Login menangani POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		c.Error(err)
		return
	}

	httpx.OK(c, "Berhasil masuk", result)
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
