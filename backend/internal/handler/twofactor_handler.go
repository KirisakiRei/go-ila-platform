package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// timeLayout adalah format waktu yang dipakai seluruh balasan API.
const timeLayout = time.RFC3339

// TwoFactorHandler melayani pengaturan 2FA untuk pintu user maupun admin.
// Alurnya sama; yang berbeda hanya akun pemilik sesi.
type TwoFactorHandler struct {
	twofa service.TwoFactorService
	// accountOf mengembalikan alamat surel akun untuk label di aplikasi
	// authenticator. Opsional: bila nil, ID akun dipakai.
	accountOf func(c *gin.Context, userID string) string
}

// NewTwoFactorHandler merangkai handler 2FA.
func NewTwoFactorHandler(twofa service.TwoFactorService, accountOf func(c *gin.Context, userID string) string) *TwoFactorHandler {
	return &TwoFactorHandler{twofa: twofa, accountOf: accountOf}
}

// formatOptionalTime mengubah waktu opsional menjadi teks RFC3339.
func formatOptionalTime(at *time.Time) string {
	if at == nil {
		return ""
	}
	return at.Format(timeLayout)
}

// Status menangani GET /2fa — keadaan 2FA akun ini.
func (h *TwoFactorHandler) Status(c *gin.Context) {
	userID := middleware.CurrentUserID(c)

	status, err := h.twofa.Status(c.Request.Context(), userID)
	if err != nil {
		c.Error(err)
		return
	}

	httpx.OK(c, "Status 2FA", gin.H{
		"enabled":            status.Enabled,
		"activated_at":       formatOptionalTime(status.ActivatedAt),
		"last_used_at":       formatOptionalTime(status.LastUsedAt),
		"recovery_remaining": status.RecoveryRemaining,
	})
}

// Setup menangani POST /2fa/setup — membuat secret baru yang belum aktif.
func (h *TwoFactorHandler) Setup(c *gin.Context) {
	userID := middleware.CurrentUserID(c)

	account := userID
	if h.accountOf != nil {
		if label := h.accountOf(c, userID); label != "" {
			account = label
		}
	}

	setup, err := h.twofa.Setup(c.Request.Context(), userID, account)
	if err != nil {
		c.Error(err)
		return
	}

	// Secret dikirim sekali di sini agar bisa dipindai; tidak pernah lagi.
	httpx.OK(c, "Secret 2FA baru", gin.H{
		"secret":         setup.Secret,
		"otpauth_url":    setup.ProvisioningURL,
		"expires_at":     setup.ExpiresAt.Format(timeLayout),
		"digits":         6,
		"period_seconds": 30,
	})
}

// Activate menangani POST /2fa/activate — verifikasi kode lalu aktifkan.
func (h *TwoFactorHandler) Activate(c *gin.Context) {
	var req dto.ActivateTwoFactorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	codes, err := h.twofa.Activate(c.Request.Context(), middleware.CurrentUserID(c), req.Code)
	if err != nil {
		c.Error(err)
		return
	}

	// Kode pemulihan hanya ditampilkan sekali; tidak ada cara melihat ulang.
	httpx.OK(c, "2FA aktif. Simpan kode pemulihan berikut sekarang.", gin.H{
		"recovery_codes": codes,
		"note":           "Kode pemulihan hanya ditampilkan sekali. Simpan di tempat aman.",
	})
}

// Disable menangani POST /2fa/disable — matikan setelah kode terbukti.
func (h *TwoFactorHandler) Disable(c *gin.Context) {
	var req dto.CodeOnlyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	if err := h.twofa.Disable(c.Request.Context(), middleware.CurrentUserID(c), req.Code); err != nil {
		c.Error(err)
		return
	}

	httpx.OK(c, "2FA dinonaktifkan", gin.H{"enabled": false})
}

// Regenerate menangani POST /2fa/recovery-codes — set kode pemulihan baru.
func (h *TwoFactorHandler) Regenerate(c *gin.Context) {
	var req dto.CodeOnlyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	codes, err := h.twofa.RegenerateRecovery(c.Request.Context(), middleware.CurrentUserID(c), req.Code)
	if err != nil {
		c.Error(err)
		return
	}

	httpx.OK(c, "Kode pemulihan baru. Kode lama tidak berlaku lagi.", gin.H{
		"recovery_codes": codes,
		"note":           "Kode pemulihan hanya ditampilkan sekali. Simpan di tempat aman.",
	})
}
