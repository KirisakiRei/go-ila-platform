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

// AdminSystemHandler adalah pintu HTTP pengaturan sistem dan operasional.
type AdminSystemHandler struct {
	flags  service.AdminSystemService
	ops    service.AdminSystemOps
	public service.PublicStatusService
}

// NewAdminSystemHandler merangkai handler sistem.
func NewAdminSystemHandler(flags service.AdminSystemService, ops service.AdminSystemOps, public service.PublicStatusService) *AdminSystemHandler {
	return &AdminSystemHandler{flags: flags, ops: ops, public: public}
}

// Register memasang route pengaturan dan operasional.
// Ubah flag dan ubah mode dijaga superadmin lewat authz.SuperadminOnly.
func (h *AdminSystemHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/system", guard(http.MethodGet, "/system", authz.SystemView), h.Overview)
	g.GET("/system/engine", guard(http.MethodGet, "/system/engine", authz.SystemView), h.Engine)
	g.GET("/system/runner", guard(http.MethodGet, "/system/runner", authz.SystemView), h.Runner)
	g.GET("/system/brokers", guard(http.MethodGet, "/system/brokers", authz.SystemView), h.Brokers)
	g.GET("/system/settings", guard(http.MethodGet, "/system/settings", authz.SystemSettings), h.Settings)
	g.PUT("/system/settings/:key", guard(http.MethodPut, "/system/settings/:key", authz.SystemSettings), h.UpdateSetting)
	g.DELETE("/system/settings/:key", guard(http.MethodDelete, "/system/settings/:key", authz.SystemSettings), h.ResetSetting)

	g.PUT("/system/mode", guard(http.MethodPut, "/system/mode", authz.SystemMaintenance), h.ChangeMode)

	g.GET("/users/:id/support-notes", guard(http.MethodGet, "/users/:id/support-notes", authz.UserView), h.SupportNotes)
	g.POST("/users/:id/support-notes", guard(http.MethodPost, "/users/:id/support-notes", authz.UserEdit), h.AddSupportNote)
}

// Overview menangani GET /api/v1/admin/system.
func (h *AdminSystemHandler) Overview(c *gin.Context) {
	result, err := h.ops.Overview(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kondisi sistem", result)
}

// Engine menangani GET /api/v1/admin/system/engine.
func (h *AdminSystemHandler) Engine(c *gin.Context) {
	result, err := h.ops.EngineDetail(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kondisi engine", result)
}

// Runner menangani GET /api/v1/admin/system/runner.
func (h *AdminSystemHandler) Runner(c *gin.Context) {
	result, err := h.ops.RunnerDetail(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kondisi runner", result)
}

// Brokers menangani GET /api/v1/admin/system/brokers.
func (h *AdminSystemHandler) Brokers(c *gin.Context) {
	result, err := h.ops.BrokerDetail(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kesehatan akun broker", result)
}

// Settings menangani GET /api/v1/admin/system/settings.
func (h *AdminSystemHandler) Settings(c *gin.Context) {
	result, err := h.flags.Settings(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar flag sistem", result)
}

// UpdateSetting menangani PUT /api/v1/admin/system/settings/:key.
func (h *AdminSystemHandler) UpdateSetting(c *gin.Context) {
	var req dto.SystemSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.flags.UpdateSetting(c.Request.Context(), actorOf(c), c.Param("key"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Flag sistem diperbarui", result)
}

// ResetSetting menangani DELETE /api/v1/admin/system/settings/:key.
func (h *AdminSystemHandler) ResetSetting(c *gin.Context) {
	var req dto.SystemSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.flags.ResetSetting(c.Request.Context(), actorOf(c), c.Param("key"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Flag dikembalikan ke nilai bawaan", nil)
}

// ChangeMode menangani PUT /api/v1/admin/system/mode.
func (h *AdminSystemHandler) ChangeMode(c *gin.Context) {
	var req dto.MaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.ops.ChangeMode(c.Request.Context(), actorOf(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Mode platform diperbarui", result)
}

// SupportNotes menangani GET /api/v1/admin/users/:id/support-notes.
func (h *AdminSystemHandler) SupportNotes(c *gin.Context) {
	result, err := h.ops.SupportNotes(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Jurnal dukungan", result)
}

// AddSupportNote menangani POST /api/v1/admin/users/:id/support-notes.
func (h *AdminSystemHandler) AddSupportNote(c *gin.Context) {
	var req dto.SupportNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.ops.AddSupportNote(c.Request.Context(), actorOf(c), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Catatan dukungan disimpan", result)
}

// PublicStatusHandler menyajikan mode platform tanpa sesi.
type PublicStatusHandler struct {
	public service.PublicStatusService
}

// NewPublicStatusHandler merangkai handler status publik.
func NewPublicStatusHandler(public service.PublicStatusService) *PublicStatusHandler {
	return &PublicStatusHandler{public: public}
}

// Status menangani GET /api/v1/status.
func (h *PublicStatusHandler) Status(c *gin.Context) {
	result, err := h.public.Status(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Status layanan", result)
}

// actorOf membentuk pelaku admin dari konteks permintaan.
func actorOf(c *gin.Context) service.AdminActor {
	return service.AdminActor{
		ID:        middleware.CurrentUserID(c),
		RequestID: c.GetString(middleware.ContextRequestID),
	}
}
