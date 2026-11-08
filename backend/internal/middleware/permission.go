package middleware

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authstate"
	"ila/backend/internal/session"
	"ila/backend/pkg/apperror"
	"ila/backend/pkg/utils"
)

// adminRole adalah nilai role kasar yang wajib dimiliki sesi admin.
const adminRole = "admin"

// PermissionChecker memeriksa kewenangan akun.
type PermissionChecker interface {
	Has(ctx context.Context, userID, permission string) (bool, error)
}

// AdminScope memastikan permintaan datang dari sesi admin yang masih sah.
//
// Verifikasi tidak berhenti pada tanda tangan token: status akun dan
// session_version ikut dicocokkan ke database, sehingga ban, ganti password,
// atau perubahan role berlaku pada permintaan berikutnya (plans/03 §5).
func AdminScope(issuer interface {
	Verify(token string) (utils.TokenClaims, error)
}, cookieName string, reader authstate.Reader) gin.HandlerFunc {
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
		if claims.Scope != utils.ScopeAdmin {
			c.Error(apperror.ErrForbidden)
			c.Abort()
			return
		}

		if reader != nil {
			account, err := reader.FindSessionUser(context.WithoutCancel(c.Request.Context()), claims.UserID)
			if err != nil || account == nil {
				c.Error(apperror.ErrUnauthorized)
				c.Abort()
				return
			}
			if account.SessionVersion != claims.Session {
				c.Error(apperror.ErrUnauthorized)
				c.Abort()
				return
			}
			if account.Status != authstate.StatusActive {
				c.Error(apperror.ErrForbidden)
				c.Abort()
				return
			}
			// Role kasar admin wajib: mencabut status admin memutus akses.
			if account.Role != adminRole {
				c.Error(apperror.ErrForbidden)
				c.Abort()
				return
			}
			c.Set(ContextUserID, account.ID)
			c.Set(ContextRole, account.Role)
			c.Set(ContextSessionID, claims.SessionID)
			c.Next()
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextRole, claims.Role)
		c.Set(ContextSessionID, claims.SessionID)
		c.Next()
	}
}

// RequirePermission menolak permintaan yang tidak memiliki permission tertentu.
// Pesan penolakan tidak menyebut permission yang kurang agar penyerang tidak
// bisa memetakan sistem (plans/04 §5.3).
func RequirePermission(checker PermissionChecker, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := CurrentUserID(c)
		if userID == "" {
			c.Error(apperror.ErrUnauthorized)
			c.Abort()
			return
		}

		ok, err := checker.Has(c.Request.Context(), userID, permission)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		if !ok {
			c.Error(apperror.ErrForbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// routeRegistry mencatat route admin yang sudah mendeklarasikan permission.
type routeRegistry struct {
	mu    sync.Mutex
	items map[string]string
}

var adminRoutes = &routeRegistry{items: make(map[string]string)}

// adminPublicRoutes adalah route yang memang tidak menuntut permission karena
// belum ada sesi admin (login) atau hanya menyentuh sesi sendiri (me, logout).
var adminPublicRoutes = map[string]struct{}{
	http.MethodPost + " /api/v1/admin/auth/login":      {},
	http.MethodPost + " /api/v1/admin/auth/verify-2fa": {},
	http.MethodPost + " /api/v1/admin/auth/logout":     {},
	http.MethodPost + " /api/v1/admin/auth/reset-2fa":  {},
	http.MethodGet + " /api/v1/admin/auth/me":          {},
	http.MethodGet + " /api/v1/admin/auth/csrf":        {},
}

// RegisterAdminRoute mencatat pasangan metode+path dengan permission-nya.
func RegisterAdminRoute(method, path, permission string) {
	adminRoutes.mu.Lock()
	adminRoutes.items[method+" "+path] = permission
	adminRoutes.mu.Unlock()
}

// UndeclaredAdminRoutes memeriksa apakah ada route /admin/* tanpa permission.
// Mengembalikan daftar route yang melanggar; kosong berarti aman.
func UndeclaredAdminRoutes(routes []gin.RouteInfo) []string {
	adminRoutes.mu.Lock()
	defer adminRoutes.mu.Unlock()

	var missing []string
	for _, info := range routes {
		if !isAdminPath(info.Path) {
			continue
		}
		key := info.Method + " " + info.Path
		if _, public := adminPublicRoutes[key]; public {
			continue
		}
		if adminRoutes.items[key] == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

// ResetAdminRoutesForTest mengosongkan catatan deklarasi. Dipakai pengujian
// agar satu berkas tes tidak mewarisi deklarasi tes lain.
func ResetAdminRoutesForTest() {
	adminRoutes.mu.Lock()
	adminRoutes.items = make(map[string]string)
	adminRoutes.mu.Unlock()
}

func isAdminPath(path string) bool {
	const prefix = "/api/v1/admin"
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}
