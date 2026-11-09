package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"ila/backend/internal/service"
	"ila/backend/pkg/httpx"
)

// PublicSignalHandler menyajikan ringkasan arah sinyal untuk halaman publik.
//
// Endpoint ini terbuka tanpa sesi, sehingga hanya boleh memuat arah dan waktu.
// Parameter yang diterima pun dibatasi: yang tidak dikenal katalog diabaikan
// oleh layanan, bukan diteruskan ke kueri.
type PublicSignalHandler struct {
	preview service.PublicSignalService
}

// NewPublicSignalHandler merangkai handler ringkasan sinyal publik.
func NewPublicSignalHandler(preview service.PublicSignalService) *PublicSignalHandler {
	return &PublicSignalHandler{preview: preview}
}

// Preview menangani GET /api/v1/signals/preview.
//
// Simbol dikirim sebagai daftar dipisah koma pada `symbols`. Nilai kosong
// berarti memakai instrumen bawaan, sehingga halaman depan tidak perlu tahu
// simbol apa yang sedang dipantau.
func (h *PublicSignalHandler) Preview(c *gin.Context) {
	symbols := parseSymbolList(c.Query("symbols"))

	result, err := h.preview.Preview(c.Request.Context(), symbols)
	if err != nil {
		c.Error(err)
		return
	}

	// Ringkasan ini identik untuk semua pengunjung dalam satu menit, jadi boleh
	// disimpan sebentar oleh peramban maupun CDN.
	c.Header("Cache-Control", "public, max-age=60")
	httpx.OK(c, "Ringkasan pantauan pasar", result)
}

// parseSymbolList memecah daftar simbol dan membuang entri kosong.
func parseSymbolList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	bagian := strings.Split(raw, ",")
	out := make([]string, 0, len(bagian))
	for _, item := range bagian {
		if nama := strings.TrimSpace(item); nama != "" {
			out = append(out, nama)
		}
	}
	return out
}
