package middleware

import (
	"log/slog"
	"strings"

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

// sentinelDomain mencantumkan teks bawaan error domain agar tidak bocor sebagai
// prefix mentah ke layar pengguna (mis. "service temporarily unavailable: ...").
var sentinelDomain = []string{
	apperror.ErrNotFound.Error(),
	apperror.ErrAlreadyExists.Error(),
	apperror.ErrConflict.Error(),
	apperror.ErrUnauthorized.Error(),
	apperror.ErrForbidden.Error(),
	apperror.ErrBadRequest.Error(),
	apperror.ErrInternalServer.Error(),
	apperror.ErrServiceUnavailable.Error(),
	apperror.ErrTooManyRequests.Error(),
}

// pesanBawaanKode adalah pesan ramah pengguna saat error hanya berupa sentinel
// tanpa keterangan tambahan.
func pesanBawaanKode(code string) string {
	switch code {
	case "NOT_FOUND":
		return "Data tidak ditemukan."
	case "ALREADY_EXISTS":
		return "Data sudah terdaftar."
	case "CONFLICT":
		return "Permintaan tidak dapat diproses saat ini karena bentrok dengan status sebelumnya."
	case "UNAUTHORIZED":
		return "Sesi tidak valid atau telah berakhir. Silakan masuk kembali."
	case "FORBIDDEN":
		return "Anda tidak memiliki akses untuk tindakan ini."
	case "BAD_REQUEST":
		return "Data yang dikirim tidak valid. Periksa kembali isian Anda."
	case "TOO_MANY_REQUESTS":
		return "Terlalu banyak permintaan. Tunggu sebentar lalu coba lagi."
	case "SERVICE_UNAVAILABLE":
		return "Layanan sedang tidak tersedia sementara. Silakan coba lagi beberapa saat."
	default:
		return "Terjadi kesalahan internal. Silakan coba lagi."
	}
}

// bersihkanPesan menghapus prefix sentinel domain dan detail teknis berlebih
// agar yang diterima klien hanyalah pesan yang bisa dipahami pengguna.
func bersihkanPesan(raw, code string) string {
	if code == "INTERNAL_SERVER_ERROR" {
		return pesanBawaanKode(code)
	}

	pesan := strings.TrimSpace(raw)
	for _, s := range sentinelDomain {
		if pesan == s {
			return pesanBawaanKode(code)
		}
		if strings.HasPrefix(pesan, s+": ") {
			pesan = strings.TrimSpace(strings.TrimPrefix(pesan, s+": "))
			break
		}
		if strings.HasPrefix(pesan, s+"\n") {
			pesan = strings.TrimSpace(strings.TrimPrefix(pesan, s+"\n"))
			break
		}
	}

	// Untuk errors.Join(ErrServiceUnavailable, errors.New("pesan"), errUpstream),
	// baris pertama setelah sentinel adalah pesan aman untuk pengguna, sedangkan
	// baris berikutnya bisa berisi detail jaringan/upstream.
	if code == "SERVICE_UNAVAILABLE" {
		if idx := strings.Index(pesan, "\n"); idx >= 0 {
			pesan = strings.TrimSpace(pesan[:idx])
		}
		pesan = strings.TrimSpace(strings.TrimRight(pesan, ":"))
	} else if idx := strings.LastIndex(pesan, "\n"); idx >= 0 {
		pesan = strings.TrimSpace(pesan[idx+1:])
	}

	if code == "BAD_REQUEST" {
		pesan = httpx.PesanValidasi(pesan)
	}
	if pesan == "" {
		return pesanBawaanKode(code)
	}
	return pesan
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
		message := bersihkanPesan(err.Error(), code)
		httpx.Error(c, status, code, message, nil)
		c.Abort()
	}
}
