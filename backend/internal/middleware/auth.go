package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/apperror"
)

// Auth memverifikasi token Bearer lalu menyimpan identitas akun di context.
func Auth(issuer service.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.Error(apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
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
