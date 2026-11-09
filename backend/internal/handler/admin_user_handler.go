package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// AdminUserHandler adalah pintu HTTP manajemen pelanggan.
type AdminUserHandler struct {
	users service.AdminUserService
}

// NewAdminUserHandler merangkai handler pelanggan.
func NewAdminUserHandler(users service.AdminUserService) *AdminUserHandler {
	return &AdminUserHandler{users: users}
}

// actor mengambil pelaku beserta nomor permintaan untuk baris audit.
func (h *AdminUserHandler) actor(c *gin.Context) service.AdminActor {
	return service.AdminActor{
		ID:        middleware.CurrentUserID(c),
		RequestID: c.GetString(middleware.ContextRequestID),
	}
}

// List menangani GET /api/v1/admin/users.
func (h *AdminUserHandler) List(c *gin.Context) {
	var q dto.AdminUserFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	q.HasTrial = c.Query("trial") == "yes"
	q.OnlyAdmins = c.Query("role") == "admin"
	q.RoleID = c.Query("role_id")

	items, meta, err := h.users.List(c.Request.Context(), h.actor(c), q)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OKWithMeta(c, "Daftar pelanggan", items, meta)
}

// Create menangani POST /api/v1/admin/users.
func (h *AdminUserHandler) Create(c *gin.Context) {
	var req dto.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	res, err := h.users.Create(c.Request.Context(), h.actor(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Pengguna berhasil dibuat", res)
}

// Detail menangani GET /api/v1/admin/users/:id.
func (h *AdminUserHandler) Detail(c *gin.Context) {
	result, err := h.users.Detail(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Detail pelanggan", result)
}

// Accounts menangani GET /api/v1/admin/users/:id/accounts.
func (h *AdminUserHandler) Accounts(c *gin.Context) {
	result, err := h.users.Accounts(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Akun broker pelanggan", result)
}

// Balances menangani GET /api/v1/admin/users/:id/account-balances.
// Baca sensitif: pembacaannya dicatat audit oleh service.
func (h *AdminUserHandler) Balances(c *gin.Context) {
	result, err := h.users.Balances(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Saldo akun pelanggan", result)
}

// Positions menangani GET /api/v1/admin/users/:id/positions.
// Setiap pembacaan tercatat sebagai baca sensitif.
func (h *AdminUserHandler) Positions(c *gin.Context) {
	result, err := h.users.Positions(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Riwayat posisi pelanggan", result)
}

// ExportPositions menangani GET /api/v1/admin/users/:id/positions-export.
// Berkas langsung diunduh; pembacaan dicatat audit oleh service.
func (h *AdminUserHandler) ExportPositions(c *gin.Context) {
	result, err := h.users.ExportPositions(c.Request.Context(), h.actor(c), c.Param("id"), c.Query("from"), c.Query("to"))
	if err != nil {
		c.Error(err)
		return
	}
	writeCSV(c, result.Filename, result.Content)
}

// UpdateAccountLabel menangani PATCH /api/v1/admin/accounts/:id/label.
func (h *AdminUserHandler) UpdateAccountLabel(c *gin.Context) {
	var req dto.AdminAccountLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.users.UpdateAccountLabel(c.Request.Context(), h.actor(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Label akun diperbarui", nil)
}

// DeleteAccount menangani DELETE /api/v1/admin/accounts/:id.
func (h *AdminUserHandler) DeleteAccount(c *gin.Context) {
	var req dto.UserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.users.DeleteAccount(c.Request.Context(), h.actor(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Akun pelanggan dihapus", nil)
}

// Legal menangani GET /api/v1/admin/users/:id/legal.
func (h *AdminUserHandler) Legal(c *gin.Context) {
	result, err := h.users.Legal(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Persetujuan dokumen pelanggan", result)
}

// Notes menangani GET /api/v1/admin/users/:id/notes.
func (h *AdminUserHandler) Notes(c *gin.Context) {
	result, err := h.users.Notes(c.Request.Context(), h.actor(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Catatan internal", result)
}

// AddNote menangani POST /api/v1/admin/users/:id/notes.
func (h *AdminUserHandler) AddNote(c *gin.Context) {
	var req dto.UserNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.users.AddNote(c.Request.Context(), h.actor(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Catatan disimpan", nil)
}

// Suspend menangani POST /api/v1/admin/users/:id/suspend.
func (h *AdminUserHandler) Suspend(c *gin.Context) {
	h.action(c, func(ctx *gin.Context, actor service.AdminActor, req dto.UserActionRequest) error {
		return h.users.Suspend(ctx.Request.Context(), actor, ctx.Param("id"), req)
	}, "Akun ditangguhkan")
}

// Unsuspend menangani POST /api/v1/admin/users/:id/unsuspend.
func (h *AdminUserHandler) Unsuspend(c *gin.Context) {
	h.action(c, func(ctx *gin.Context, actor service.AdminActor, req dto.UserActionRequest) error {
		return h.users.Unsuspend(ctx.Request.Context(), actor, ctx.Param("id"), req)
	}, "Penangguhan dibuka")
}

// Ban menangani POST /api/v1/admin/users/:id/ban.
func (h *AdminUserHandler) Ban(c *gin.Context) {
	h.action(c, func(ctx *gin.Context, actor service.AdminActor, req dto.UserActionRequest) error {
		return h.users.Ban(ctx.Request.Context(), actor, ctx.Param("id"), req)
	}, "Akun diblokir")
}

// Unban menangani POST /api/v1/admin/users/:id/unban.
func (h *AdminUserHandler) Unban(c *gin.Context) {
	h.action(c, func(ctx *gin.Context, actor service.AdminActor, req dto.UserActionRequest) error {
		return h.users.Unban(ctx.Request.Context(), actor, ctx.Param("id"), req)
	}, "Blokir dibuka")
}

// Delete menangani POST /api/v1/admin/users/:id/delete.
func (h *AdminUserHandler) Delete(c *gin.Context) {
	h.action(c, func(ctx *gin.Context, actor service.AdminActor, req dto.UserActionRequest) error {
		return h.users.Delete(ctx.Request.Context(), actor, ctx.Param("id"), req)
	}, "Akun dihapus")
}

// ChangeTier menangani PATCH /api/v1/admin/users/:id/tier.
func (h *AdminUserHandler) ChangeTier(c *gin.Context) {
	var req dto.UserTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.users.ChangeTier(c.Request.Context(), h.actor(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Tier diperbarui", nil)
}

// FreezeAccount menangani POST /api/v1/admin/accounts/:id/freeze.
func (h *AdminUserHandler) FreezeAccount(c *gin.Context) {
	var req dto.AccountFreezeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.users.FreezeAccount(c.Request.Context(), h.actor(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Akun broker dibekukan", nil)
}

// UnfreezeAccount menangani POST /api/v1/admin/accounts/:id/unfreeze.
func (h *AdminUserHandler) UnfreezeAccount(c *gin.Context) {
	var req dto.AccountFreezeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Alasan opsional saat membuka pembekuan.
		req = dto.AccountFreezeRequest{}
	}
	if err := h.users.UnfreezeAccount(c.Request.Context(), h.actor(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Pembekuan dibuka", nil)
}

// Export menangani GET /api/v1/admin/users-export.
func (h *AdminUserHandler) Export(c *gin.Context) {
	var q dto.AdminUserFilter
	if err := c.ShouldBindQuery(&q); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	q.HasTrial = c.Query("trial") == "yes"
	q.OnlyAdmins = c.Query("role") == "admin"
	q.RoleID = c.Query("role_id")

	body, filename, err := h.users.ExportUsers(c.Request.Context(), h.actor(c), q)
	if err != nil {
		c.Error(err)
		return
	}
	// BOM sudah ikut di dalam isi berkas agar Excel membaca UTF-8 dengan benar.
	writeCSV(c, filename, body)
}

// writeCSV mengirim berkas CSV sebagai unduhan.
// BOM sudah ikut di dalam isi berkas agar Excel membaca UTF-8 dengan benar.
func writeCSV(c *gin.Context, filename string, body []byte) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/csv; charset=utf-8", body)
}

func (h *AdminUserHandler) action(c *gin.Context, run func(*gin.Context, service.AdminActor, dto.UserActionRequest) error, message string) {
	var req dto.UserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := run(c, h.actor(c), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, message, nil)
}
