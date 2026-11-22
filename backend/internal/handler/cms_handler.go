package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// CMSHandler adalah pintu HTTP konten: FAQ, pengumuman, dan tautan media sosial.
type CMSHandler struct {
	cms    service.CMSService
	social service.SocialLinksService
}

// NewCMSHandler merangkai handler konten.
func NewCMSHandler(cms service.CMSService) *CMSHandler {
	return &CMSHandler{cms: cms}
}

// WithSocialLinks memasang pengelola tautan media sosial publik.
func (h *CMSHandler) WithSocialLinks(social service.SocialLinksService) *CMSHandler {
	h.social = social
	return h
}

// Register memasang route konten admin.
func (h *CMSHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/faq", guard(http.MethodGet, "/faq", authz.CMSView), h.ListFaq)
	g.POST("/faq", guard(http.MethodPost, "/faq", authz.CMSEdit), h.CreateFaq)
	g.PATCH("/faq/:id", guard(http.MethodPatch, "/faq/:id", authz.CMSEdit), h.UpdateFaq)
	g.DELETE("/faq/:id", guard(http.MethodDelete, "/faq/:id", authz.CMSEdit), h.DeleteFaq)
	g.GET("/announcements", guard(http.MethodGet, "/announcements", authz.CMSView), h.ListAnnouncements)
	g.POST("/announcements", guard(http.MethodPost, "/announcements", authz.CMSPublish), h.CreateAnnouncement)
	g.PATCH("/announcements/:id", guard(http.MethodPatch, "/announcements/:id", authz.CMSPublish), h.UpdateAnnouncement)
	g.DELETE("/announcements/:id", guard(http.MethodDelete, "/announcements/:id", authz.CMSPublish), h.DeleteAnnouncement)
	if h.social != nil {
		g.GET("/social-links", guard(http.MethodGet, "/social-links", authz.CMSView), h.GetSocialLinks)
		g.PUT("/social-links", guard(http.MethodPut, "/social-links", authz.CMSEdit), h.UpdateSocialLinks)
	}
}

// PublicRegister memasang route konten publik.
func (h *CMSHandler) PublicRegister(g *gin.RouterGroup) {
	g.GET("/faq", h.PublicFaq)
	g.GET("/announcements", h.PublicAnnouncements)
}

// PublicFaq menangani GET /api/v1/faq.
func (h *CMSHandler) PublicFaq(c *gin.Context) {
	h.replyList(c, false)
}

// PublicAnnouncements menangani GET /api/v1/announcements.
func (h *CMSHandler) PublicAnnouncements(c *gin.Context) {
	items, err := h.cms.Announcements(c.Request.Context(), false)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Pengumuman", items)
}

// ListFaq menangani GET /api/v1/admin/faq.
func (h *CMSHandler) ListFaq(c *gin.Context) {
	h.replyList(c, true)
}

func (h *CMSHandler) replyList(c *gin.Context, admin bool) {
	items, err := h.cms.FAQs(c.Request.Context(), admin)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar FAQ", items)
}

// CreateFaq menangani POST /api/v1/admin/faq.
func (h *CMSHandler) CreateFaq(c *gin.Context) {
	h.saveFaq(c, "")
}

// UpdateFaq menangani PATCH /api/v1/admin/faq/:id.
func (h *CMSHandler) UpdateFaq(c *gin.Context) {
	h.saveFaq(c, c.Param("id"))
}

func (h *CMSHandler) saveFaq(c *gin.Context, id string) {
	var req dto.FaqRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	item, err := h.cms.SaveFaq(c.Request.Context(), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "FAQ disimpan", item)
}

// DeleteFaq menangani DELETE /api/v1/admin/faq/:id.
func (h *CMSHandler) DeleteFaq(c *gin.Context) {
	if err := h.cms.DeleteFaq(c.Request.Context(), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "FAQ dihapus", nil)
}

// ListAnnouncements menangani GET /api/v1/admin/announcements.
func (h *CMSHandler) ListAnnouncements(c *gin.Context) {
	items, err := h.cms.Announcements(c.Request.Context(), true)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar pengumuman", items)
}

// CreateAnnouncement menangani POST /api/v1/admin/announcements.
func (h *CMSHandler) CreateAnnouncement(c *gin.Context) {
	h.saveAnnouncement(c, "")
}

// UpdateAnnouncement menangani PATCH /api/v1/admin/announcements/:id.
func (h *CMSHandler) UpdateAnnouncement(c *gin.Context) {
	h.saveAnnouncement(c, c.Param("id"))
}

func (h *CMSHandler) saveAnnouncement(c *gin.Context, id string) {
	var req dto.AnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	item, err := h.cms.SaveAnnouncement(c.Request.Context(), actorOf(c), id, req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Pengumuman disimpan", item)
}

// DeleteAnnouncement menangani DELETE /api/v1/admin/announcements/:id.
func (h *CMSHandler) DeleteAnnouncement(c *gin.Context) {
	if err := h.cms.DeleteAnnouncement(c.Request.Context(), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Pengumuman dihapus", nil)
}

// GetSocialLinks menangani GET /api/v1/admin/social-links.
func (h *CMSHandler) GetSocialLinks(c *gin.Context) {
	httpx.OK(c, "Tautan media sosial", h.social.Get(c.Request.Context()))
}

// UpdateSocialLinks menangani PUT /api/v1/admin/social-links.
func (h *CMSHandler) UpdateSocialLinks(c *gin.Context) {
	var req dto.SocialLinks
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	updated, err := h.social.Update(c.Request.Context(), actorOf(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Tautan media sosial diperbarui", updated)
}
