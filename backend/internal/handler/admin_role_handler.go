package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// AdminRoleHandler adalah pintu HTTP untuk pengelolaan role dan kewenangan.
type AdminRoleHandler struct {
	roles service.RoleService
}

// NewAdminRoleHandler merangkai handler RBAC.
func NewAdminRoleHandler(roles service.RoleService) *AdminRoleHandler {
	return &AdminRoleHandler{roles: roles}
}

// List menangani GET /api/v1/admin/roles.
func (h *AdminRoleHandler) List(c *gin.Context) {
	result, err := h.roles.ListRoles(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar role", result)
}

// Detail menangani GET /api/v1/admin/roles/:id.
func (h *AdminRoleHandler) Detail(c *gin.Context) {
	result, err := h.roles.GetRole(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Detail role", result)
}

// Create menangani POST /api/v1/admin/roles.
func (h *AdminRoleHandler) Create(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.roles.CreateRole(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Role dibuat", result)
}

// Update menangani PATCH /api/v1/admin/roles/:id.
func (h *AdminRoleHandler) Update(c *gin.Context) {
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.roles.UpdateRole(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Permission role diperbarui", result)
}

// Delete menangani DELETE /api/v1/admin/roles/:id.
func (h *AdminRoleHandler) Delete(c *gin.Context) {
	if err := h.roles.DeleteRole(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Role dihapus", nil)
}

// Assign menangani PATCH /api/v1/admin/users/:id/role.
func (h *AdminRoleHandler) Assign(c *gin.Context) {
	var req dto.AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	if err := h.roles.AssignRole(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Role akun diperbarui", nil)
}

// CreateAdmin menangani POST /api/v1/admin/admins.
//
// Akun pengelola dibuat dari tab User pada halaman RBAC: cukup username, kata
// sandi, dan role. Tanpa verifikasi email dan tanpa alur trial/onboarding.
func (h *AdminRoleHandler) CreateAdmin(c *gin.Context) {
	var req dto.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.roles.CreateAdmin(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Akun pengelola dibuat", result)
}

// DeleteAdmin menangani DELETE /api/v1/admin/admins/:id.
//
// Alasan boleh dikirim lewat body JSON {"reason": "..."} atau query
// ?reason=..., agar panel dan perkakas CLI sama-sama nyaman.
func (h *AdminRoleHandler) DeleteAdmin(c *gin.Context) {
	var req dto.DeleteAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = c.Query("reason")
	}
	if len(strings.TrimSpace(req.Reason)) < 10 {
		c.Error(httpx.WrapValidation("alasan penghapusan wajib diisi minimal 10 karakter"))
		return
	}

	if err := h.roles.DeleteAdmin(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req.Reason); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Akun pengelola dihapus", nil)
}

// Permissions menangani GET /api/v1/admin/permissions.
func (h *AdminRoleHandler) Permissions(c *gin.Context) {
	result, err := h.roles.ListPermissions(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Katalog permission", result)
}
