package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// BugHandler adalah pintu HTTP laporan bug: pengguna dan triage admin.
type BugHandler struct {
	svc   service.BugService
	admin service.AdminBugService
}

// NewBugHandler merangkai handler laporan bug.
func NewBugHandler(svc service.BugService, admin service.AdminBugService) *BugHandler {
	return &BugHandler{svc: svc, admin: admin}
}

// RegisterUser memasang route untuk pengguna yang sudah masuk.
//
// Lampiran TIDAK dipasang di sini: middleware Auth menolak sesi admin, padahal
// admin perlu membuka lampiran untuk men-triage. Route lampiran dipasang
// terpisah lewat RegisterAttachment dengan pengenal scope.
func (h *BugHandler) RegisterUser(g *gin.RouterGroup) {
	g.POST("/bug-reports", h.Create)
	g.GET("/bug-reports", h.ListMine)
	g.GET("/bug-reports/categories", h.Kategori)
}

// RegisterAttachment memasang route lampiran pada grup yang menerima sesi
// pengguna MAUPUN admin (dua cookie berbeda, dua middleware).
func (h *BugHandler) RegisterAttachment(g *gin.RouterGroup) {
	g.GET("/bug-reports/:id/attachments/:aid", h.Attachment)
}

// Register memasang route triage admin.
func (h *BugHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/bug-reports", guard(http.MethodGet, "/bug-reports", authz.UserView), h.AdminList)
	g.PATCH("/bug-reports/:id", guard(http.MethodPatch, "/bug-reports/:id", authz.UserEdit), h.AdminUpdate)
}

// Create menangani POST /api/v1/bug-reports (multipart).
//
// Badan permintaan dibatasi MaxBytesReader supaya unggahan besar tidak pernah
// masuk ke memori proses (kontainer backend dibatasi 160 MB).
func (h *BugHandler) Create(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.BugMaxLampiranByte+64*1024)

	if err := c.Request.ParseMultipartForm(service.BugMaxLampiranByte + 32*1024); err != nil {
		if isTooLarge(err) {
			c.Error(service.ErrLampiranTerlaluBesar)
			return
		}
		c.Error(httpx.WrapValidation("formulir laporan tidak terbaca"))
		return
	}

	in := service.BugInput{
		Category:  c.PostForm("category"),
		Title:     c.PostForm("title"),
		Body:      c.PostForm("body"),
		PageURL:   c.PostForm("page_url"),
		UserAgent: c.Request.UserAgent(),
	}

	fileHeader, err := c.FormFile("attachment")
	switch {
	case err == nil && fileHeader != nil:
		berkas, openErr := fileHeader.Open()
		if openErr != nil {
			c.Error(httpx.WrapValidation("lampiran tidak terbaca"))
			return
		}
		defer func() { _ = berkas.Close() }()
		in.Attachment = &service.BugUpload{FileName: fileHeader.Filename, Reader: berkas}
	case err != nil && isTooLarge(err):
		c.Error(service.ErrLampiranTerlaluBesar)
		return
	}

	detail, err := h.svc.Create(c.Request.Context(), middleware.CurrentUserID(c), in)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Laporan terkirim. Terima kasih.", detail)
}

// ListMine menangani GET /api/v1/bug-reports.
func (h *BugHandler) ListMine(c *gin.Context) {
	items, err := h.svc.ListMine(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Laporan Anda", items)
}

// Kategori menangani GET /api/v1/bug-reports/categories.
func (h *BugHandler) Kategori(c *gin.Context) {
	httpx.OK(c, "Kategori laporan", service.KategoriLaporan())
}

// Attachment menangani GET /api/v1/bug-reports/:id/attachments/:aid.
//
// Header sengaja dipasang agar berkas TIDAK PERNAH dirender peramban: lampiran
// pengguna adalah masukan tidak tepercaya, dan penyajian inline akan menjadi
// jalur XSS ke sesi admin yang membukanya. Kepemilikan diperiksa service:
// pemilik laporan atau admin, selain itu 404 (bukan 403, agar keberadaan
// laporan orang lain tidak bisa ditebak).
func (h *BugHandler) Attachment(c *gin.Context) {
	actor := service.BugActor{
		UserID: middleware.CurrentUserID(c),
		// Nilai ini hanya ada bila permintaan lolos middleware admin.
		Admin: middleware.CurrentAdminScope(c),
	}

	file, err := h.svc.OpenAttachment(c.Request.Context(), actor, c.Param("id"), c.Param("aid"))
	if err != nil {
		c.Error(err)
		return
	}

	c.Header("Content-Disposition", service.ContentDisposition(file.FileName))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Type", file.MimeType)
	c.Header("Cache-Control", "private, no-store")
	c.File(file.Path)
}

// AdminList menangani GET /api/v1/admin/bug-reports.
func (h *BugHandler) AdminList(c *gin.Context) {
	items, err := h.admin.List(c.Request.Context(), c.Query("status"), c.Query("category"))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar laporan bug", items)
}

// BugTriageRequest adalah payload perubahan status laporan.
type BugTriageRequest struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note" binding:"omitempty,max=2000"`
}

// AdminUpdate menangani PATCH /api/v1/admin/bug-reports/:id.
func (h *BugHandler) AdminUpdate(c *gin.Context) {
	var req BugTriageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	actor := service.AdminActor{ID: middleware.CurrentUserID(c), RequestID: c.GetString(middleware.ContextRequestID)}
	if err := h.admin.UpdateStatus(c.Request.Context(), actor, c.Param("id"), req.Status, req.Note); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Status laporan diperbarui", nil)
}

// isTooLarge mengenali galat batas ukuran dari net/http.
func isTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}
	pesan := strings.ToLower(err.Error())
	return strings.Contains(pesan, "too large") || strings.Contains(pesan, "request body too large")
}
