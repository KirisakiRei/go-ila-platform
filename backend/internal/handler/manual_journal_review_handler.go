package handler

import (
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type ManualJournalReviewHandler struct {
	svc *service.ManualJournalReviewService
}

func NewManualJournalReviewHandler(svc *service.ManualJournalReviewService) *ManualJournalReviewHandler {
	return &ManualJournalReviewHandler{svc: svc}
}

// Register requires the same authenticated group and error middleware as ManualJournalHandler.
func (h *ManualJournalReviewHandler) Register(g *gin.RouterGroup) {
	r := g.Group("/manual-journal")
	r.GET("/review", h.Get)
	r.GET("/review/settings", h.Settings)
	r.PUT("/review/settings", h.SaveSettings)
}
func (h *ManualJournalReviewHandler) Get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), middleware.CurrentUserID(c), c.Query("account_id"))
	manualJournalReply(c, v, err, false)
}
func (h *ManualJournalReviewHandler) Settings(c *gin.Context) {
	v, err := h.svc.Settings(c.Request.Context(), middleware.CurrentUserID(c), c.Query("account_id"))
	manualJournalReply(c, v, err, false)
}
func (h *ManualJournalReviewHandler) SaveSettings(c *gin.Context) {
	var in service.ManualJournalReviewSettingsInput
	if !manualJournalBind(c, &in, 1024) {
		return
	}
	v, err := h.svc.SaveSettings(c.Request.Context(), middleware.CurrentUserID(c), c.Query("account_id"), in)
	manualJournalReply(c, v, err, false)
}
