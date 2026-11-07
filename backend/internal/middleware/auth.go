package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authstate"
	"ila/backend/internal/service"
	"ila/backend/internal/session"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

// ctxKeySessionUser menyimpan akun yang sudah divalidasi agar handler tidak
// perlu membaca DB lagi.
const ctxKeySessionUser = "session_user"

// Auth memverifikasi token sesi dari cookie HttpOnly lalu menyimpan identitas
// akun di context. Token tidak pernah dibaca dari header Authorization agar
// tidak ada jalur alternatif yang bisa bocor ke JavaScript.
//
// Bila reader diberikan, setiap permintaan juga mencocokkan session_version
// dengan DB dan memastikan akun masih boleh memakai platform. Inilah yang
// membuat ban dan ganti password berlaku pada request berikutnya (plans/03 §5).
func Auth(issuer service.TokenIssuer, cookieName string, reader authstate.Reader) gin.HandlerFunc {
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

		if claims.Scope == utils.ScopeAdmin {
			// Sesi admin tidak boleh melewati pintu user (plans/03 §2).
			c.Error(apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		if reader != nil && !validateSession(c, reader, claims) {
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextRole, claims.Role)
		c.Set(ContextSessionID, claims.SessionID)
		c.Next()
	}
}

// validateSession mencocokkan token dengan kondisi akun terkini.
func validateSession(c *gin.Context, reader authstate.Reader, claims utils.TokenClaims) bool {
	account, err := reader.FindSessionUser(context.WithoutCancel(c.Request.Context()), claims.UserID)
	if err != nil || account == nil {
		c.Error(apperror.ErrUnauthorized)
		c.Abort()
		return false
	}

	if account.SessionVersion != claims.Session {
		// Sesi dicabut: password berganti, ban, atau role berubah.
		c.Error(apperror.ErrUnauthorized)
		c.Abort()
		return false
	}

	if !authstate.CanUsePlatform(account.Status) {
		c.Error(apperror.ErrForbidden)
		c.Abort()
		return false
	}

	c.Set(ctxKeySessionUser, account)
	return true
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

// CurrentSessionID membaca ID sesi perangkat ini dari context.
func CurrentSessionID(c *gin.Context) string {
	return c.GetString(ContextSessionID)
}

// SessionAccount mengembalikan akun hasil validasi middleware, bila ada.
func SessionAccount(c *gin.Context) (*authstate.User, bool) {
	value, ok := c.Get(ctxKeySessionUser)
	if !ok {
		return nil, false
	}
	account, ok := value.(*authstate.User)
	return account, ok
}
