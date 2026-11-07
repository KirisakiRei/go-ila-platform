package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/middleware"
)

// AdminGuard membuat penjaga permission sekaligus mencatat deklarasinya.
// Didefinisikan sebagai tipe agar router dan handler memakai bentuk yang sama.
type AdminGuard func(method, path, permission string) gin.HandlerFunc

// Register memasang seluruh route manajemen pelanggan.
// Setiap route mendeklarasikan permission miliknya tepat di barisnya.
func (h *AdminUserHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	// Daftar dan profil: tingkat 1, tanpa saldo dan tanpa kredensial.
	g.GET("/users", guard(http.MethodGet, "/users", authz.UserView), h.List)
	g.GET("/users/:id", guard(http.MethodGet, "/users/:id", authz.UserView), h.Detail)
	g.GET("/users/:id/accounts", guard(http.MethodGet, "/users/:id/accounts", authz.AccountView), h.Accounts)
	g.GET("/users/:id/legal", guard(http.MethodGet, "/users/:id/legal", authz.UserLegal), h.Legal)
	g.GET("/users/:id/notes", guard(http.MethodGet, "/users/:id/notes", authz.UserView), h.Notes)
	g.POST("/users/:id/notes", guard(http.MethodPost, "/users/:id/notes", authz.UserEdit), h.AddNote)

	// Riwayat posisi: baca sensitif, dicatat setiap kali dibuka.
	g.GET("/users/:id/positions", guard(http.MethodGet, "/users/:id/positions", authz.TradeView), h.Positions)

	g.POST("/users/:id/suspend", guard(http.MethodPost, "/users/:id/suspend", authz.UserSuspend), h.Suspend)
	g.POST("/users/:id/unsuspend", guard(http.MethodPost, "/users/:id/unsuspend", authz.UserSuspend), h.Unsuspend)
	g.POST("/users/:id/ban", guard(http.MethodPost, "/users/:id/ban", authz.UserBan), h.Ban)
	g.POST("/users/:id/unban", guard(http.MethodPost, "/users/:id/unban", authz.UserBan), h.Unban)
	g.POST("/users/:id/delete", guard(http.MethodPost, "/users/:id/delete", authz.UserDelete), h.Delete)
	g.PATCH("/users/:id/tier", guard(http.MethodPatch, "/users/:id/tier", authz.UserTier), h.ChangeTier)

	// Akun broker: hanya pembekuan. Menutup posisi atau mengambil alih
	// kredensial bukan kewenangan admin.
	g.POST("/accounts/:id/freeze", guard(http.MethodPost, "/accounts/:id/freeze", authz.AccountFreeze), h.FreezeAccount)
	g.POST("/accounts/:id/unfreeze", guard(http.MethodPost, "/accounts/:id/unfreeze", authz.AccountFreeze), h.UnfreezeAccount)

	// Ekspor daftar pelanggan: dibatasi 10.000 baris dan selalu diaudit.
	g.GET("/users-export", guard(http.MethodGet, "/users-export", authz.UserExport), h.Export)
}

var _ = middleware.RequirePermission
