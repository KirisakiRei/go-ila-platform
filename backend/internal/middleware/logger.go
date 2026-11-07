package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestID memberi setiap permintaan identitas untuk pelacakan log.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// Logger mencatat satu baris ringkas per permintaan.
// Header Authorization dan Cookie sengaja tidak pernah disertakan.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", c.GetString("request_id"),
		}
		if userID, ok := c.Get(ContextUserID); ok {
			attrs = append(attrs, "user_id", userID)
		}

		switch {
		case c.Writer.Status() >= 500:
			logger.Error("permintaan selesai", attrs...)
		case c.Writer.Status() >= 400:
			logger.Warn("permintaan selesai", attrs...)
		default:
			logger.Info("permintaan selesai", attrs...)
		}
	}
}
