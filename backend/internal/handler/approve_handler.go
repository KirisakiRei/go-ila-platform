package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/KirisakiRei/go-ila-platform/backend/internal/middleware"
	"github.com/KirisakiRei/go-ila-platform/backend/internal/service"
	"github.com/KirisakiRei/go-ila-platform/backend/pkg/httpx"
)

// ApproveHandler adalah pintu HTTP untuk halaman konfirmasi entry.
//
// Semua endpoint membutuhkan sesi login: deep-link tanpa sesi hanya
// menampilkan ajakan masuk, bukan kuasa eksekusi (plans/13 §4.3).
type ApproveHandler struct {
	approveService service.ApproveService
}

// NewApproveHandler merangkai handler konfirmasi.
func NewApproveHandler(approveService service.ApproveService) *ApproveHandler {
	return &ApproveHandler{approveService: approveService}
}

// Preview menangani GET /api/v1/approve/:token.
func (h *ApproveHandler) Preview(c *gin.Context) {
	preview, err := h.approveService.Resolve(
		c.Request.Context(),
		middleware.CurrentUserID(c),
		c.Param("token"),
	)
	if err != nil {
		c.Error(err)
		return
	}

	httpx.OK(c, "Detail sinyal", preview)
}

// Confirm menangani POST /api/v1/approve/:token/confirm.
func (h *ApproveHandler) Confirm(c *gin.Context) {
	result, err := h.approveService.Confirm(
		c.Request.Context(),
		middleware.CurrentUserID(c),
		c.Param("token"),
	)
	if err != nil {
		c.Error(err)
		return
	}

	httpx.OK(c, result.Message, result)
}

// Decline menangani POST /api/v1/approve/:token/decline.
func (h *ApproveHandler) Decline(c *gin.Context) {
	result, err := h.approveService.Decline(
		c.Request.Context(),
		middleware.CurrentUserID(c),
		c.Param("token"),
	)
	if err != nil {
		c.Error(err)
		return
	}

	httpx.OK(c, result.Message, result)
}
