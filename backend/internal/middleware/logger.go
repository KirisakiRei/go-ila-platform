package middleware

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// maxRequestIDLen selaras dengan panjang kolom audit_log.request_id varchar(64).
const maxRequestIDLen = 64

// RequestID memberi setiap permintaan identitas untuk pelacakan log.
// Nilai kiriman klien hanya diterima bila panjang dan karakternya aman, supaya
// tidak pernah menggagalkan penulisan audit_log akibat melebihi varchar(64)
// atau menyuntikkan karakter kontrol ke header/log.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if !validRequestID(id) {
			id = newRequestID()
		}
		c.Set(ContextRequestID, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > maxRequestIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		ch := id[i]
		alnum := (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
		if !alnum && ch != '-' && ch != '_' && ch != '.' && ch != ':' {
			return false
		}
	}
	return true
}

// Logger mencatat satu baris ringkas per permintaan.
// Header Authorization dan Cookie sengaja tidak pernah disertakan.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		attrs := []any{
			"evt", "http.selesai",
			"method", c.Request.Method,
			"path", safePath(c.Request.URL.Path),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"trace", c.GetString(ContextRequestID),
		}
		if userID, ok := c.Get(ContextUserID); ok {
			attrs = append(attrs, "user_id", userID)
		}

		switch {
		case c.Writer.Status() >= 500:
			logger.Error("permintaan gagal", attrs...)
		case c.Writer.Status() >= 400:
			// 4xx adalah kesalahan pemakaian, bukan kerusakan sistem.
			logger.Warn("permintaan ditolak", attrs...)
		default:
			logger.Info("permintaan selesai", attrs...)
		}
	}
}

// safePath menyamarkan token pada path sebelum dicatat.
//
// Token deep-link konfirmasi entry berada di path (/api/v1/approve/<token>).
// Tanpa penyamaran, siapa pun yang bisa membaca log dapat memakai tautan itu.
func safePath(path string) string {
	const penanda = "approve/"

	idx := strings.Index(path, penanda)
	if idx < 0 {
		return path
	}

	prefix := path[:idx+len(penanda)]
	rest := path[idx+len(penanda):]

	// Sisa token boleh memuat sub-path seperti /confirm; hanya segmen pertama
	// yang merupakan token dan harus disamarkan.
	segment := rest
	suffix := ""
	if slash := strings.Index(rest, "/"); slash >= 0 {
		segment = rest[:slash]
		suffix = rest[slash:]
	}
	if segment == "" {
		return path
	}

	return prefix + "<disembunyikan>" + suffix
}
