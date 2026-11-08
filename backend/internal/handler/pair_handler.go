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

// PairHandler adalah pintu HTTP katalog pair untuk user.
type PairHandler struct {
	catalog service.PairCatalogService
	account service.AccountService
}

// NewPairHandler merangkai handler katalog pair.
func NewPairHandler(catalog service.PairCatalogService, account service.AccountService) *PairHandler {
	return &PairHandler{catalog: catalog, account: account}
}

// Register memasang route user untuk katalog pair dan checklist per akun.
func (h *PairHandler) Register(me *gin.RouterGroup) {
	me.GET("/pairs", h.Catalog)
	me.GET("/accounts/:id/pairs", h.AccountPairs)
	me.PUT("/accounts/:id/pairs", h.UpdateAccountPairs)
}

// Catalog menangani GET /api/v1/pairs.
func (h *PairHandler) Catalog(c *gin.Context) {
	result, err := h.catalog.Catalog(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Katalog pair", result)
}

// AccountPairs menangani GET /api/v1/accounts/:id/pairs.
func (h *PairHandler) AccountPairs(c *gin.Context) {
	ctx := c.Request.Context()
	result, err := h.account.AccountPairs(ctx, middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Checklist pair akun", result)
}

// UpdateAccountPairs menangani PUT /api/v1/accounts/:id/pairs.
func (h *PairHandler) UpdateAccountPairs(c *gin.Context) {
	var req dto.UpdateAccountPairsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	ctx := c.Request.Context()
	result, err := h.account.UpdateAccountPairs(ctx, middleware.CurrentUserID(c), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Pilihan pair akun disimpan", result)
}

// AdminPairHandler adalah pintu HTTP pengelolaan katalog pair oleh admin.
type AdminPairHandler struct {
	pairs service.PairAdminService
}

// NewAdminPairHandler merangkai handler admin katalog pair.
func NewAdminPairHandler(pairs service.PairAdminService) *AdminPairHandler {
	return &AdminPairHandler{pairs: pairs}
}

// Register memasang route admin katalog pair dengan permission masing-masing.
func (h *AdminPairHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/pairs", guard(http.MethodGet, "/pairs", authz.SubscriptionEdit), h.Catalog)

	g.POST("/pair-categories", guard(http.MethodPost, "/pair-categories", authz.SubscriptionEdit), h.CreateCategory)
	g.PATCH("/pair-categories/:id", guard(http.MethodPatch, "/pair-categories/:id", authz.SubscriptionEdit), h.UpdateCategory)
	g.DELETE("/pair-categories/:id", guard(http.MethodDelete, "/pair-categories/:id", authz.SubscriptionEdit), h.DeleteCategory)

	g.POST("/pair-groups", guard(http.MethodPost, "/pair-groups", authz.SubscriptionEdit), h.CreateGroup)
	g.PATCH("/pair-groups/:id", guard(http.MethodPatch, "/pair-groups/:id", authz.SubscriptionEdit), h.UpdateGroup)
	g.DELETE("/pair-groups/:id", guard(http.MethodDelete, "/pair-groups/:id", authz.SubscriptionEdit), h.DeleteGroup)

	g.POST("/pairs", guard(http.MethodPost, "/pairs", authz.SubscriptionEdit), h.CreatePair)
	g.PATCH("/pairs/:id", guard(http.MethodPatch, "/pairs/:id", authz.SubscriptionEdit), h.UpdatePair)
	g.DELETE("/pairs/:id", guard(http.MethodDelete, "/pairs/:id", authz.SubscriptionEdit), h.DeletePair)
}

// Catalog menangani GET /api/v1/admin/pairs.
func (h *AdminPairHandler) Catalog(c *gin.Context) {
	result, err := h.pairs.Catalog(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Katalog pair (admin)", result)
}

// CreateCategory menangani POST /api/v1/admin/pair-categories.
func (h *AdminPairHandler) CreateCategory(c *gin.Context) {
	var req dto.CreatePairCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.pairs.CreateCategory(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Kategori pair dibuat", result)
}

// UpdateCategory menangani PATCH /api/v1/admin/pair-categories/:id.
func (h *AdminPairHandler) UpdateCategory(c *gin.Context) {
	var req dto.UpdatePairCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.pairs.UpdateCategory(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kategori pair diperbarui", nil)
}

// DeleteCategory menangani DELETE /api/v1/admin/pair-categories/:id.
func (h *AdminPairHandler) DeleteCategory(c *gin.Context) {
	if err := h.pairs.DeleteCategory(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Kategori pair dihapus", nil)
}

// CreateGroup menangani POST /api/v1/admin/pair-groups.
func (h *AdminPairHandler) CreateGroup(c *gin.Context) {
	var req dto.CreatePairGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.pairs.CreateGroup(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Grup pair dibuat", result)
}

// UpdateGroup menangani PATCH /api/v1/admin/pair-groups/:id.
func (h *AdminPairHandler) UpdateGroup(c *gin.Context) {
	var req dto.UpdatePairGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.pairs.UpdateGroup(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Grup pair diperbarui", nil)
}

// DeleteGroup menangani DELETE /api/v1/admin/pair-groups/:id.
func (h *AdminPairHandler) DeleteGroup(c *gin.Context) {
	if err := h.pairs.DeleteGroup(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Grup pair dihapus", nil)
}

// CreatePair menangani POST /api/v1/admin/pairs.
func (h *AdminPairHandler) CreatePair(c *gin.Context) {
	var req dto.CreatePairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	result, err := h.pairs.CreatePair(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Pair dibuat", result)
}

// UpdatePair menangani PATCH /api/v1/admin/pairs/:id.
func (h *AdminPairHandler) UpdatePair(c *gin.Context) {
	var req dto.UpdatePairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.pairs.UpdatePair(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Pair diperbarui", nil)
}

// DeletePair menangani DELETE /api/v1/admin/pairs/:id.
func (h *AdminPairHandler) DeletePair(c *gin.Context) {
	if err := h.pairs.DeletePair(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Pair dihapus", nil)
}
