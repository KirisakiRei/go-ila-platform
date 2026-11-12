package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authstate"
	"ila/backend/internal/session"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

// SesiPenggunaAtauAdmin menerima permintaan yang membawa cookie sesi pengguna
// ATAU cookie sesi admin.
//
// Dipakai satu tempat saja: unduhan lampiran laporan bug, yang sah dibuka oleh
// pemilik laporan (sesi pengguna) maupun admin yang men-triage (sesi admin).
// Middleware Auth biasa menolak sesi admin dan AdminScope menolak sesi pengguna,
// sehingga tanpa ini tidak ada satu grup pun yang melayani keduanya.
//
// Urutannya: sesi pengguna dicoba lebih dulu karena itulah pemakaian normal.
// Saat jalur admin yang berhasil, SesiPenggunaAtauAdmin memasang penanda
// ContextAdminScope supaya handler tahu wewenang tambahan itu ada.
//
// Keputusan wewenang TIDAK diambil di sini. Middleware ini hanya membuktikan
// identitas; service tetap memeriksa apakah akun itu pemilik laporan atau admin.
func SesiPenggunaAtauAdmin(
	userIssuer interface {
		Verify(token string) (utils.TokenClaims, error)
	},
	adminIssuer interface {
		Verify(token string) (utils.TokenClaims, error)
	},
	userCookie, adminCookie string,
	reader authstate.Reader,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cokelat := session.CookieValue(c, userCookie); cokelat != "" {
			if claims, err := userIssuer.Verify(cokelat); err == nil && claims.Scope != utils.ScopeAdmin {
				if reader != nil && !validasiSesi(c, reader, claims) {
					return
				}
				c.Set(ContextUserID, claims.UserID)
				c.Set(ContextSessionID, claims.SessionID)
				c.Next()
				return
			}
		}

		tokenAdmin := session.CookieValue(c, adminCookie)
		if tokenAdmin == "" {
			c.Error(apperror.ErrUnauthorized)
			c.Abort()
			return
		}
		claims, err := adminIssuer.Verify(tokenAdmin)
		if err != nil || claims.Scope != utils.ScopeAdmin {
			c.Error(apperror.ErrUnauthorized)
			c.Abort()
			return
		}
		if reader != nil && !validasiSesi(c, reader, claims) {
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextSessionID, claims.SessionID)
		c.Set(ContextAdminScope, true)
		c.Next()
	}
}

// validasiSesi mencocokkan session_version dan status akun ke database.
// Mengembalikan false (dan sudah menulis galat) bila sesi tidak lagi sah.
func validasiSesi(c *gin.Context, reader authstate.Reader, claims utils.TokenClaims) bool {
	account, err := reader.FindSessionUser(context.WithoutCancel(c.Request.Context()), claims.UserID)
	if err != nil || account == nil {
		c.Error(apperror.ErrUnauthorized)
		c.Abort()
		return false
	}
	if account.SessionVersion != claims.Session {
		c.Error(apperror.ErrUnauthorized)
		c.Abort()
		return false
	}
	if account.Status != authstate.StatusActive {
		c.Error(apperror.ErrForbidden)
		c.Abort()
		return false
	}
	return true
}
