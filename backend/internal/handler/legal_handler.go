package handler

import (
	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// LegalHandler adalah pintu HTTP dokumen legal dan persetujuannya.
// Endpoint publik (List/BySlug) dan endpoint bersesi (Me/Accept) berbagi
// handler yang sama agar bentuk balasannya konsisten.
type LegalHandler struct {
	legal service.LegalService
}

// NewLegalHandler merangkai handler legal.
func NewLegalHandler(legal service.LegalService) *LegalHandler {
	return &LegalHandler{legal: legal}
}

// List menangani GET /api/v1/legal — daftar dokumen yang berlaku.
func (h *LegalHandler) List(c *gin.Context) {
	result, err := h.legal.Published(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar dokumen legal", result)
}

// BySlug menangani GET /api/v1/legal/:slug — isi versi yang berlaku.
func (h *LegalHandler) BySlug(c *gin.Context) {
	result, err := h.legal.BySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Dokumen legal", result)
}

// Me menangani GET /api/v1/me/legal — status persetujuan akun ini.
func (h *LegalHandler) Me(c *gin.Context) {
	result, err := h.legal.ForUser(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Status persetujuan legal", result)
}

// Accept menangani POST /api/v1/me/legal/:slug/accept.
func (h *LegalHandler) Accept(c *gin.Context) {
	var req dto.AcceptLegalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Badan kosong diperlakukan sebagai "versi terbaru".
		req = dto.AcceptLegalRequest{}
	}

	result, err := h.legal.Accept(
		c.Request.Context(),
		middleware.CurrentUserID(c),
		c.Param("slug"),
		req,
		service.AcceptanceMeta{UserAgent: c.Request.UserAgent()},
	)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Persetujuan tercatat", result)
}

// AdminList menangani GET /api/v1/admin/legal — seluruh versi termasuk draf.
func (h *LegalHandler) AdminList(c *gin.Context) {
	result, err := h.legal.ListAll(c.Request.Context(), c.Query("slug"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Seluruh versi dokumen legal", result)
}

// AdminCreate menangani POST /api/v1/admin/legal — membuat draf versi baru.
func (h *LegalHandler) AdminCreate(c *gin.Context) {
	var req dto.LegalDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.legal.CreateDraft(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Draf dokumen dibuat", result)
}

// AdminUpdate menangani PATCH /api/v1/admin/legal/:id — hanya untuk draf.
func (h *LegalHandler) AdminUpdate(c *gin.Context) {
	var req dto.LegalDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	result, err := h.legal.UpdateDraft(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Draf diperbarui", result)
}

// AdminPublish menangani POST /api/v1/admin/legal/:id/publish.
func (h *LegalHandler) AdminPublish(c *gin.Context) {
	var req dto.PublishLegalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = dto.PublishLegalRequest{}
	}

	result, err := h.legal.Publish(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Dokumen diterbitkan", result)
}
