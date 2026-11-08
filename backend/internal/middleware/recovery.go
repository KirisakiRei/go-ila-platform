package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"ila/backend/pkg/apperror"
	"ila/backend/pkg/httpx"
)

// Recovery menangkap panic agar proses tetap hidup dan balasan tetap seragam.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic tertangkap",
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"panic", rec,
				)
				httpx.Error(c, 500, "INTERNAL_SERVER_ERROR", "Terjadi kesalahan internal. Silakan coba lagi.", nil)
				c.Abort()
			}
		}()
		c.Next()
	}
}

// ErrorHandler memusatkan penerjemahan error domain ke balasan HTTP.
// Handler cukup mengembalikan error lewat c.Error, tidak perlu menulis status sendiri.
func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err
		if httpx.IsInternal(err) {
			logger.Error("kesalahan internal",
				"evt", "http.internal",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"error", err.Error(),
			)
		}

		if c.Writer.Written() {
			return
		}

		status, code := apperror.MapHTTPStatus(err)
		message := err.Error()
		if code == "INTERNAL_SERVER_ERROR" {
			message = "Terjadi kesalahan internal. Silakan coba lagi."
		}
		httpx.Error(c, status, code, message, nil)
		c.Abort()
	}
}
