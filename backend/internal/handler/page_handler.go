package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// PageHandler adalah pintu HTTP isi halaman publik.
//
// Kontennya dikelola lewat panel admin dan disimpan di basis data, sehingga
// memeriksa ejaan atau mengganti kalimat tidak memerlukan deploy ulang.
type PageHandler struct {
	pages service.PageService
}

// NewPageHandler merangkai handler halaman.
func NewPageHandler(pages service.PageService) *PageHandler {
	return &PageHandler{pages: pages}
}

// PublicRegister memasang route baca halaman publik.
func (h *PageHandler) PublicRegister(g *gin.RouterGroup) {
	g.GET("/pages/:slug", h.Public)
}

// Register memasang route pengelolaan halaman di panel admin.
func (h *PageHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/pages", guard(http.MethodGet, "/pages", authz.CMSView), h.List)
	g.GET("/pages/:slug", guard(http.MethodGet, "/pages/:slug", authz.CMSView), h.Admin)
	g.PUT("/pages/:slug", guard(http.MethodPut, "/pages/:slug", authz.CMSEdit), h.Replace)
}

// Public menangani GET /api/v1/pages/:slug.
func (h *PageHandler) Public(c *gin.Context) {
	page, err := h.pages.Public(c.Request.Context(), c.Param("slug"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Isi halaman", page)
}

// List menangani GET /api/v1/admin/pages.
func (h *PageHandler) List(c *gin.Context) {
	pages, err := h.pages.All(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar halaman", pages)
}

// Admin menangani GET /api/v1/admin/pages/:slug.
func (h *PageHandler) Admin(c *gin.Context) {
	page, err := h.pages.Admin(c.Request.Context(), c.Param("slug"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Isi halaman", page)
}

// Replace menangani PUT /api/v1/admin/pages/:slug.
func (h *PageHandler) Replace(c *gin.Context) {
	var req dto.PageUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	page, err := h.pages.Replace(c.Request.Context(), actorOf(c), c.Param("slug"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Halaman disimpan", page)
}
