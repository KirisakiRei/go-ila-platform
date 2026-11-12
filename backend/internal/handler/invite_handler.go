package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// InviteHandler adalah pintu HTTP undangan: panel admin dan pemeriksaan publik.
type InviteHandler struct {
	admin  service.AdminInviteService
	publik service.InviteService
}

// NewInviteHandler merangkai handler undangan.
func NewInviteHandler(admin service.AdminInviteService, publik service.InviteService) *InviteHandler {
	return &InviteHandler{admin: admin, publik: publik}
}

// Register memasang route admin. Setiap route menyatakan permission-nya di sini.
//
// Undangan dijaga user.edit karena membuat undangan berarti memberi akses masuk
// ke platform; wewenangnya setara membuat akun.
func (h *InviteHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/invites", guard(http.MethodGet, "/invites", authz.UserView), h.List)
	g.POST("/invites", guard(http.MethodPost, "/invites", authz.UserEdit), h.Create)
	g.POST("/invites/:id/revoke", guard(http.MethodPost, "/invites/:id/revoke", authz.UserEdit), h.Revoke)
}

// actor mengambil pelaku dari sesi admin.
func (h *InviteHandler) actor(c *gin.Context) service.AdminActor {
	return service.AdminActor{ID: middleware.CurrentUserID(c), RequestID: c.GetString(middleware.ContextRequestID)}
}

// List menangani GET /api/v1/admin/invites.
func (h *InviteHandler) List(c *gin.Context) {
	items, err := h.admin.List(c.Request.Context(), c.Query("status"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar undangan", items)
}

// Create menangani POST /api/v1/admin/invites.
//
// Balasan memuat kode mentah satu kali; setelah ini kode tidak bisa dibaca lagi.
func (h *InviteHandler) Create(c *gin.Context) {
	var req dto.CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	created, err := h.admin.Create(c.Request.Context(), h.actor(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Undangan dibuat. Salin tautannya sekarang, tidak bisa dilihat lagi.", created)
}

// RevokeRequest adalah payload pencabutan undangan.
type RevokeRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=255"`
}

// Revoke menangani POST /api/v1/admin/invites/:id/revoke.
func (h *InviteHandler) Revoke(c *gin.Context) {
	var req RevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.admin.Revoke(c.Request.Context(), h.actor(c), c.Param("id"), req.Reason); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Undangan dicabut", nil)
}

// Periksa menangani GET /api/v1/invites/:code (publik).
//
// Tidak memakai kuota. Halaman daftar memanggil ini untuk memutuskan apakah
// formulir ditampilkan; keputusan sebenarnya tetap diambil saat submit.
func (h *InviteHandler) Periksa(c *gin.Context) {
	info, err := h.publik.Periksa(c.Request.Context(), c.Param("code"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Undangan berlaku", info)
}
