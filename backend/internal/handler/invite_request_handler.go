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

// InviteRequestHandler menangani form pengajuan akses publik dan antrean tinjauan admin.
type InviteRequestHandler struct {
	publik service.InviteRequestService
	admin  service.AdminInviteRequestService
}

// NewInviteRequestHandler merangkai handler permohonan undangan.
func NewInviteRequestHandler(publik service.InviteRequestService, admin service.AdminInviteRequestService) *InviteRequestHandler {
	return &InviteRequestHandler{publik: publik, admin: admin}
}

// Register memasang route admin dengan permission setara pengelolaan undangan.
func (h *InviteRequestHandler) Register(g *gin.RouterGroup, guard AdminGuard) {
	g.GET("/invite-requests", guard(http.MethodGet, "/invite-requests", authz.UserView), h.List)
	g.POST("/invite-requests/:id/approve", guard(http.MethodPost, "/invite-requests/:id/approve", authz.UserEdit), h.Approve)
	g.POST("/invite-requests/:id/reject", guard(http.MethodPost, "/invite-requests/:id/reject", authz.UserEdit), h.Reject)
}

func (h *InviteRequestHandler) actor(c *gin.Context) service.AdminActor {
	return service.AdminActor{ID: middleware.CurrentUserID(c), RequestID: c.GetString(middleware.ContextRequestID)}
}

// Submit menangani POST /api/v1/invite-requests (publik, tanpa sesi).
func (h *InviteRequestHandler) Submit(c *gin.Context) {
	var req dto.SubmitInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.publik.Submit(c.Request.Context(), req); err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Permohonan akses diterima. Tim kami akan meninjau dan mengirimkan tautan undangan.", nil)
}

// List menangani GET /api/v1/admin/invite-requests.
func (h *InviteRequestHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	res, meta, err := h.admin.List(c.Request.Context(), c.Query("status"), page, perPage)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OKWithMeta(c, "Daftar permohonan undangan", res, meta)
}

// Approve menangani POST /api/v1/admin/invite-requests/:id/approve.
func (h *InviteRequestHandler) Approve(c *gin.Context) {
	var req dto.ApproveInviteRequestBody
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(httpx.WrapValidation(err.Error()))
			return
		}
	}
	res, err := h.admin.Approve(c.Request.Context(), h.actor(c), c.Param("id"), req)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Permohonan disetujui dan tautan undangan dibuat.", res)
}

// Reject menangani POST /api/v1/admin/invite-requests/:id/reject.
func (h *InviteRequestHandler) Reject(c *gin.Context) {
	var req dto.RejectInviteRequestBody
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(httpx.WrapValidation(err.Error()))
			return
		}
	}
	if err := h.admin.Reject(c.Request.Context(), h.actor(c), c.Param("id"), req.Reason); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Permohonan ditolak.", nil)
}
