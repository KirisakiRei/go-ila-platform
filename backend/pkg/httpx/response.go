package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"ila/backend/pkg/apperror"
)

// Response adalah bentuk tunggal semua balasan API.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Error   string `json:"error,omitempty"`
	Details any    `json:"details,omitempty"`
}

// OK mengirim balasan sukses 200.
func OK(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{Success: true, Message: message, Data: data})
}

// OKWithMeta mengirim balasan sukses 200 beserta metadata (dipakai untuk pagination).
func OKWithMeta(c *gin.Context, message string, data, meta any) {
	c.JSON(http.StatusOK, Response{Success: true, Message: message, Data: data, Meta: meta})
}

// Created mengirim balasan sukses 201.
func Created(c *gin.Context, message string, data any) {
	c.JSON(http.StatusCreated, Response{Success: true, Message: message, Data: data})
}

// NoContent mengirim balasan sukses tanpa isi.
func NoContent(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{Success: true, Message: message})
}

// Error mengirim balasan gagal dengan kode error eksplisit.
func Error(c *gin.Context, httpStatus int, errCode string, message string, details any) {
	c.JSON(httpStatus, Response{Success: false, Message: message, Error: errCode, Details: details})
}

// Fail memetakan domain error ke status HTTP lalu mengirimkannya.
// Pesan mentah err tidak diteruskan ke klien untuk error internal, agar detail
// implementasi tidak bocor. Detail asli tetap dikembalikan agar bisa dicatat log.
func Fail(c *gin.Context, err error) error {
	status, code := apperror.MapHTTPStatus(err)

	message := err.Error()
	if code == "INTERNAL_SERVER_ERROR" {
		message = "Terjadi kesalahan internal. Silakan coba lagi."
	}

	Error(c, status, code, message, nil)

	if code == "INTERNAL_SERVER_ERROR" {
		return err
	}
	return nil
}

// IsInternal menandai apakah error termasuk kegagalan internal.
func IsInternal(err error) bool {
	status, _ := apperror.MapHTTPStatus(err)
	return status >= http.StatusInternalServerError
}

// WrapValidation membungkus satu pesan validasi menjadi error BAD_REQUEST.
func WrapValidation(message string) error {
	return errors.Join(apperror.ErrBadRequest, errors.New(message))
}
