package handler

import (
	"github.com/gin-gonic/gin"

	"ila/backend/internal/dto"
	"ila/backend/internal/middleware"
	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// WebPushHandler melayani endpoint pengelolaan Web Push pengguna.
type WebPushHandler struct {
	push service.WebPushService
}

// NewWebPushHandler merangkai handler Web Push.
func NewWebPushHandler(push service.WebPushService) *WebPushHandler {
	return &WebPushHandler{push: push}
}

// Register mendaftarkan rute Web Push di bawah grup pengguna terautentikasi.
func (h *WebPushHandler) Register(rg *gin.RouterGroup) {
	rg.GET("/me/push/status", h.Status)
	rg.POST("/me/push/subscriptions", h.Subscribe)
	rg.PATCH("/me/push/subscriptions/:id", h.UpdatePreferences)
	rg.DELETE("/me/push/subscriptions/:id", h.Delete)
	rg.POST("/me/push/unsubscribe", h.Unsubscribe)
	rg.POST("/me/push/test", h.SendTest)
}

// Status menangani GET /api/v1/me/push/status.
func (h *WebPushHandler) Status(c *gin.Context) {
	var query dto.PushPaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	res, meta, err := h.push.Status(c.Request.Context(), middleware.CurrentUserID(c), query)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OKWithMeta(c, "Status notifikasi Web Push", res, meta)
}

// Subscribe menangani POST /api/v1/me/push/subscriptions.
func (h *WebPushHandler) Subscribe(c *gin.Context) {
	var req dto.PushSubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	item, err := h.push.Subscribe(
		c.Request.Context(),
		middleware.CurrentUserID(c),
		c.GetHeader("User-Agent"),
		req,
	)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.Created(c, "Perangkat berhasil didaftarkan untuk notifikasi push", item)
}

// UpdatePreferences menangani PATCH /api/v1/me/push/subscriptions/:id.
func (h *WebPushHandler) UpdatePreferences(c *gin.Context) {
	var req dto.PushUpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}

	item, err := h.push.UpdatePreferences(
		c.Request.Context(),
		middleware.CurrentUserID(c),
		c.Param("id"),
		req,
	)
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Preferensi notifikasi perangkat disimpan", item)
}

// Delete menangani DELETE /api/v1/me/push/subscriptions/:id.
func (h *WebPushHandler) Delete(c *gin.Context) {
	if err := h.push.Delete(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Perangkat dihapus dari daftar notifikasi push", gin.H{"deleted": true})
}

// Unsubscribe menangani POST /api/v1/me/push/unsubscribe.
func (h *WebPushHandler) Unsubscribe(c *gin.Context) {
	var req dto.PushUnsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(httpx.WrapValidation(err.Error()))
		return
	}
	if err := h.push.Unsubscribe(c.Request.Context(), middleware.CurrentUserID(c), req); err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Notifikasi push perangkat ini dinonaktifkan", gin.H{"unsubscribed": true})
}

// SendTest menangani POST /api/v1/me/push/test.
func (h *WebPushHandler) SendTest(c *gin.Context) {
	res, err := h.push.SendTest(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	httpx.OK(c, "Notifikasi uji berhasil dikirim", res)
}
