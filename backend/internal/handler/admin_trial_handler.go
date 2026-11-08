package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/authz"
	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// AdminTrialHandler adalah pintu HTTP kelola uji coba.
type AdminTrialHandler struct {
	trials service.AdminTrialService
}

// NewAdminTrialHandler merangkai handler uji coba.
func NewAdminTrialHandler(trials service.AdminTrialService) *AdminTrialHandler {
	return &AdminTrialHandler{trials: trials}
}

func (h *AdminTrialHandler) actor(c *gin.Context) service.AdminActor {
	return service.AdminActor{
		ID:        middleware.CurrentUserID(c),
		RequestID: c.GetString(middleware.ContextRequestID),
	}
}

// Register memasang route uji coba.
func (h *AdminTrialHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/trials", guard(http.MethodGet, "/trials", authz.UserView), h.List)
	g.POST("/trials/:id/extend", guard(http.MethodPost, "/trials/:id/extend", authz.UserTier), h.Extend)
	g.POST("/trials/:id/end", guard(http.MethodPost, "/trials/:id/end", authz.UserTier), h.End)
}

// List menangani GET /api/v1/admin/trials.
func (h *AdminTrialHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := h.trials.Trials(c.Request.Context(), c.Query("status"), limit)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Daftar uji coba", items)
}

// Extend menangani POST /api/v1/admin/trials/:id/extend.
func (h *AdminTrialHandler) Extend(c *gin.Context) {
	var req dto.AdminTrialExtendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	item, err := h.trials.Extend(c.Request.Context(), h.actor(c), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Uji coba diperpanjang", item)
}

// End menangani POST /api/v1/admin/trials/:id/end.
func (h *AdminTrialHandler) End(c *gin.Context) {
	var req dto.UserActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.trials.End(c.Request.Context(), h.actor(c), c.Param("id"), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Uji coba dihentikan", nil)
}
