package handler

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"

	"github.com/gin-gonic/gin"
)

type ManualJournalHandler struct{ svc *service.ManualJournalService }

func NewManualJournalHandler(svc *service.ManualJournalService) *ManualJournalHandler {
	return &ManualJournalHandler{svc: svc}
}
func (h *ManualJournalHandler) Register(g *gin.RouterGroup) {
	r := g.Group("/manual-journal")
	r.GET("/accounts", h.ListAccounts)
	r.POST("/accounts", h.CreateAccount)
	r.GET("/accounts/:id", h.GetAccount)
	r.PUT("/accounts/:id", h.UpdateAccount)
	r.DELETE("/accounts/:id", h.ArchiveAccount)
	r.GET("/capital", h.ListCapital)
	r.POST("/capital", h.CreateCapital)
	r.GET("/capital/:id", h.GetCapital)
	r.PUT("/capital/:id", h.UpdateCapital)
	r.DELETE("/capital/:id", h.DeleteCapital)
	r.GET("/entries", h.ListEntries)
	r.POST("/entries", h.CreateEntry)
	r.GET("/entries/:id", h.GetEntry)
	r.PUT("/entries/:id", h.UpdateEntry)
	r.DELETE("/entries/:id", h.DeleteEntry)
	r.GET("/stats", h.Stats)
	r.GET("/insights", h.Insights)
	r.GET("/entries/:id/screenshots", h.ListScreenshots)
	r.POST("/entries/:id/screenshots", h.UploadScreenshot)
	r.GET("/entries/:id/screenshots/:sid", h.Screenshot)
	r.DELETE("/entries/:id/screenshots/:sid", h.DeleteScreenshot)
	r.POST("/import/preview", h.PreviewCSV)
	r.POST("/import/confirm", h.ConfirmCSV)
	r.GET("/export", h.ExportCSV)
}
func manualJournalBind(c *gin.Context, v any, limit int64) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	if err := c.ShouldBindJSON(v); err != nil {
		c.Error(httpx.WrapValidation("invalid or oversized JSON body"))
		return false
	}
	return true
}
func manualJournalQuery(c *gin.Context) (dto.ManualJournalFilter, bool) {
	var f dto.ManualJournalFilter
	if err := c.ShouldBindQuery(&f); err != nil {
		c.Error(httpx.WrapValidation("invalid query parameters"))
		return f, false
	}
	return f, true
}
func manualJournalReply(c *gin.Context, data any, err error, created bool) {
	if err != nil {
		c.Error(err)
		return
	}
	if created {
		httpx.Created(c, "Manual journal saved", data)
	} else {
		httpx.OK(c, "Manual journal", data)
	}
}
func manualJournalList(c *gin.Context, data any, meta dto.ManualJournalMeta, err error) {
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OKWithMeta(c, "Manual journal", data, meta)
}
func (h *ManualJournalHandler) ListAccounts(c *gin.Context) {
	f, ok := manualJournalQuery(c)
	if !ok {
		return
	}
	v, m, e := h.svc.ListAccounts(c.Request.Context(), middleware.CurrentUserID(c), f)
	manualJournalList(c, v, m, e)
}
func (h *ManualJournalHandler) GetAccount(c *gin.Context) {
	v, e := h.svc.GetAccount(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	manualJournalReply(c, v, e, false)
}
func (h *ManualJournalHandler) saveAccount(c *gin.Context, id string) {
	var in dto.ManualJournalAccountInput
	if !manualJournalBind(c, &in, 128<<10) {
		return
	}
	v, e := h.svc.SaveAccount(c.Request.Context(), middleware.CurrentUserID(c), id, in)
	manualJournalReply(c, v, e, id == "")
}
func (h *ManualJournalHandler) CreateAccount(c *gin.Context) { h.saveAccount(c, "") }
func (h *ManualJournalHandler) UpdateAccount(c *gin.Context) { h.saveAccount(c, c.Param("id")) }
func (h *ManualJournalHandler) ArchiveAccount(c *gin.Context) {
	manualJournalReply(c, nil, h.svc.ArchiveAccount(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")), false)
}
func (h *ManualJournalHandler) ListCapital(c *gin.Context) {
	f, ok := manualJournalQuery(c)
	if !ok {
		return
	}
	v, m, e := h.svc.ListCapital(c.Request.Context(), middleware.CurrentUserID(c), f)
	manualJournalList(c, v, m, e)
}
func (h *ManualJournalHandler) GetCapital(c *gin.Context) {
	v, e := h.svc.GetCapital(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	manualJournalReply(c, v, e, false)
}
func (h *ManualJournalHandler) saveCapital(c *gin.Context, id string) {
	var in dto.ManualJournalCapitalInput
	if !manualJournalBind(c, &in, 128<<10) {
		return
	}
	v, e := h.svc.SaveCapital(c.Request.Context(), middleware.CurrentUserID(c), id, in)
	manualJournalReply(c, v, e, id == "")
}
func (h *ManualJournalHandler) CreateCapital(c *gin.Context) { h.saveCapital(c, "") }
func (h *ManualJournalHandler) UpdateCapital(c *gin.Context) { h.saveCapital(c, c.Param("id")) }
func (h *ManualJournalHandler) DeleteCapital(c *gin.Context) {
	manualJournalReply(c, nil, h.svc.DeleteCapital(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")), false)
}
func (h *ManualJournalHandler) ListEntries(c *gin.Context) {
	f, ok := manualJournalQuery(c)
	if !ok {
		return
	}
	v, m, e := h.svc.ListEntries(c.Request.Context(), middleware.CurrentUserID(c), f)
	manualJournalList(c, v, m, e)
}
func (h *ManualJournalHandler) GetEntry(c *gin.Context) {
	v, e := h.svc.GetEntry(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	manualJournalReply(c, v, e, false)
}
func (h *ManualJournalHandler) saveEntry(c *gin.Context, id string) {
	var in dto.ManualJournalEntryInput
	if !manualJournalBind(c, &in, 128<<10) {
		return
	}
	v, e := h.svc.SaveEntry(c.Request.Context(), middleware.CurrentUserID(c), id, in)
	manualJournalReply(c, v, e, id == "")
}
func (h *ManualJournalHandler) CreateEntry(c *gin.Context) { h.saveEntry(c, "") }
func (h *ManualJournalHandler) UpdateEntry(c *gin.Context) { h.saveEntry(c, c.Param("id")) }
func (h *ManualJournalHandler) DeleteEntry(c *gin.Context) {
	manualJournalReply(c, nil, h.svc.DeleteEntry(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")), false)
}
func (h *ManualJournalHandler) Stats(c *gin.Context) {
	f, ok := manualJournalQuery(c)
	if !ok {
		return
	}
	v, e := h.svc.Stats(c.Request.Context(), middleware.CurrentUserID(c), f)
	manualJournalReply(c, v, e, false)
}
func (h *ManualJournalHandler) Insights(c *gin.Context) {
	f, ok := manualJournalQuery(c)
	if !ok {
		return
	}
	v, e := h.svc.Insights(c.Request.Context(), middleware.CurrentUserID(c), f)
	manualJournalReply(c, v, e, false)
}
func (h *ManualJournalHandler) ListScreenshots(c *gin.Context) {
	v, e := h.svc.ListScreenshots(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	manualJournalReply(c, v, e, false)
}
func (h *ManualJournalHandler) UploadScreenshot(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.ManualJournalMaxUpload+64*1024)
	if err := c.Request.ParseMultipartForm(service.ManualJournalMaxUpload + 32*1024); err != nil {
		c.Error(httpx.WrapValidation("invalid multipart or screenshot exceeds 5 MiB"))
		return
	}
	if c.Request.MultipartForm != nil {
		defer func() {
			if err := c.Request.MultipartForm.RemoveAll(); err != nil {
				slog.Warn("manual journal multipart cleanup", "error", err)
			}
		}()
	}
	header, err := c.FormFile("file")
	if err != nil {
		c.Error(httpx.WrapValidation("file required"))
		return
	}
	f, err := header.Open()
	if err != nil {
		c.Error(err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Warn("manual journal upload close", "error", err)
		}
	}()
	v, e := h.svc.UploadScreenshot(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), header.Filename, f)
	manualJournalReply(c, v, e, true)
}
func (h *ManualJournalHandler) Screenshot(c *gin.Context) {
	f, err := h.svc.OpenScreenshot(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), c.Param("sid"))
	if err != nil {
		c.Error(err)
		return
	}
	defer func() {
		if err := f.File.Close(); err != nil {
			slog.Warn("manual journal screenshot close", "error", err)
		}
	}()
	c.Header("Content-Type", f.MIME)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Name}))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(c.Writer, c.Request, f.Name, time.Time{}, f.File)
}
func (h *ManualJournalHandler) DeleteScreenshot(c *gin.Context) {
	manualJournalReply(c, nil, h.svc.DeleteScreenshot(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"), c.Param("sid")), false)
}
func (h *ManualJournalHandler) PreviewCSV(c *gin.Context) {
	var in dto.ManualJournalCSVInput
	// JSON escaping can expand each CSV byte to six bytes; service still caps decoded CSV at 1 MiB.
	if !manualJournalBind(c, &in, 6*service.ManualJournalMaxCSV+1024) {
		return
	}
	v, e := h.svc.PreviewCSV(c.Request.Context(), middleware.CurrentUserID(c), in.CSV)
	manualJournalReply(c, v, e, false)
}
func (h *ManualJournalHandler) ConfirmCSV(c *gin.Context) {
	var in dto.ManualJournalConfirmInput
	if !manualJournalBind(c, &in, 6*service.ManualJournalMaxCSV+1024) {
		return
	}
	n, e := h.svc.ConfirmCSV(c.Request.Context(), middleware.CurrentUserID(c), in.CSV, in.Token)
	manualJournalReply(c, gin.H{"count": n}, e, false)
}
func (h *ManualJournalHandler) ExportCSV(c *gin.Context) {
	f, ok := manualJournalQuery(c)
	if !ok {
		return
	}
	b, e := h.svc.ExportCSV(c.Request.Context(), middleware.CurrentUserID(c), f)
	if e != nil {
		c.Error(e)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="manual-journal.csv"`)
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Length", fmt.Sprint(len(b)))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", b)
}
