package middleware

// Pengujian pesan galat yang sampai ke pengguna.
//
// Kasus nyata: pendaftaran ditolak dan layar pengguna menampilkan
// "invalid request payload key: 'RegisterRequest.Username' Error:Field
// validation for 'Username' failed on the 'alphanum' tag". Dua hal salah di
// situ: istilah internal "invalid request payload" ikut terkirim, dan pesan
// validator mentah tidak dapat ditindaklanjuti pengguna.

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"ila/backend/pkg/httpx"
)

// routerGalat memasang ErrorHandler dengan satu endpoint yang mengembalikan err.
func routerGalat(err error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	r.GET("/uji", func(c *gin.Context) {
		c.Error(err)
	})
	return r
}

// pesanBalasan mengambil field message dari balasan JSON.
func pesanBalasan(t *testing.T, body []byte) string {
	t.Helper()
	var res struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("balasan bukan JSON: %v (%s)", err, body)
	}
	return res.Message
}

// Galat validasi tidak boleh membawa istilah internal maupun nama tag validator.
func TestErrorHandlerGalatValidasiBersih(t *testing.T) {
	err := httpx.WrapValidation(
		"Key: 'RegisterRequest.Username' Error:Field validation for 'Username' failed on the 'alphanum' tag")

	w := httptest.NewRecorder()
	routerGalat(err).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uji", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("harapan 400, dapat %d", w.Code)
	}
	pesan := pesanBalasan(t, w.Body.Bytes())

	for _, terlarang := range []string{
		"invalid request payload",
		"Field validation",
		"alphanum",
		"Key:",
		"RegisterRequest",
	} {
		if strings.Contains(pesan, terlarang) {
			t.Errorf("balasan masih memuat %q: %q", terlarang, pesan)
		}
	}
	if !strings.Contains(pesan, "username") {
		t.Errorf("balasan harus menyebut kolom yang salah, dapat %q", pesan)
	}
}

// Pesan validasi yang ditulis manusia tetap utuh, tanpa prefix internal.
func TestErrorHandlerPesanManusiaTetapUtuh(t *testing.T) {
	err := httpx.WrapValidation("alasan penghapusan wajib diisi minimal 10 karakter")

	w := httptest.NewRecorder()
	routerGalat(err).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uji", nil))

	pesan := pesanBalasan(t, w.Body.Bytes())
	if pesan != "alasan penghapusan wajib diisi minimal 10 karakter" {
		t.Fatalf("pesan berubah: %q", pesan)
	}
}

// Galat internal tidak membocorkan detail apa pun.
func TestErrorHandlerGalatInternalDisamarkan(t *testing.T) {
	err := httpx.WrapValidation("x") // placeholder agar tidak nil
	_ = err

	w := httptest.NewRecorder()
	routerGalat(io.ErrUnexpectedEOF).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/uji", nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("harapan 500, dapat %d", w.Code)
	}
	pesan := pesanBalasan(t, w.Body.Bytes())
	if strings.Contains(pesan, "unexpected") {
		t.Fatalf("detail internal bocor: %q", pesan)
	}
}
