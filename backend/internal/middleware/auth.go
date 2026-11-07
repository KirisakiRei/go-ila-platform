package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/session"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
)

// Auth memverifikasi token sesi dari cookie HttpOnly lalu menyimpan identitas
// akun di context. Token tidak pernah dibaca dari header Authorization agar
// tidak ada jalur alternatif yang bisa bocor ke JavaScript.
func Auth(issuer service.TokenIssuer, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := session.CookieValue(c, cookieName)
		if token == "" {
			c.Error(apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := issuer.Verify(token)
		if err != nil {
			c.Error(apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

// RequireCSRF menolak permintaan pengubah keadaan yang tidak membawa header
// CSRF yang cocok dengan cookie-nya.
func RequireCSRF(cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		if !session.CSRFValid(c, cookieName) {
			c.Error(apperror.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole membatasi endpoint pada peran tertentu.
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role := c.GetString(ContextRole)
		if _, ok := allowed[role]; !ok {
			c.Error(apperror.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// CurrentUserID membaca ID akun yang sudah diverifikasi dari context.
func CurrentUserID(c *gin.Context) string {
	return c.GetString(ContextUserID)
}
