// Package session menyimpan token sesi di cookie HttpOnly dan melindungi
// permintaan tulis dengan skema double-submit CSRF.
package session

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CookieHeader adalah header yang membawa token CSRF dari klien.
const CookieHeader = "X-CSRF-Token"

// Config menampung nama cookie dan atribut keamanannya.
type Config struct {
	SessionName string
	CSRFName    string
	Secure      bool
	Domain      string
	// SameSite menentukan apakah cookie ikut pada permintaan lintas situs.
	SameSite http.SameSite
}

// NewConfig menyusun konfigurasi cookie dengan bawaan yang aman.
func NewConfig(sessionName, csrfName string, secure bool, domain string) Config {
	return Config{
		SessionName: sessionName,
		CSRFName:    csrfName,
		Secure:      secure,
		Domain:      domain,
		SameSite:    http.SameSiteStrictMode,
	}
}

// WithSessionName mengembalikan salinan dengan nama cookie sesi lain,
// dipakai pintu admin agar sesinya terpisah dari sesi user.
func (c Config) WithSessionName(name string) Config {
	c.SessionName = name
	return c
}

// WithCSRFName mengembalikan salinan dengan nama cookie CSRF lain.
//
// Pemisahan ini penting: bila admin dan user berbagi satu cookie CSRF, login
// di satu pintu menimpa nilai pintu lain dan permintaan berikutnya ditolak 403
// secara acak.
func (c Config) WithCSRFName(name string) Config {
	c.CSRFName = name
	return c
}

// SetSession menulis cookie sesi. HttpOnly membuatnya tidak terbaca JavaScript.
func (c Config) SetSession(ctx *gin.Context, token string, expiresAt time.Time) {
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     c.SessionName,
		Value:    token,
		Path:     "/",
		Domain:   c.Domain,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: c.SameSite,
	})
}

// RotateCSRF menerbitkan nilai CSRF baru; token lama tidak berlaku lagi.
func (c Config) RotateCSRF(ctx *gin.Context, expiresAt time.Time) (string, error) {
	value, err := NewToken()
	if err != nil {
		return "", err
	}

	// Cookie CSRF sengaja tidak HttpOnly: frontend harus membacanya untuk
	// mengirim header yang cocok (double-submit).
	http.SetCookie(ctx.Writer, &http.Cookie{
		Name:     c.CSRFName,
		Value:    value,
		Path:     "/",
		Domain:   c.Domain,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: false,
		Secure:   c.Secure,
		SameSite: c.SameSite,
	})
	return value, nil
}

// Clear menghapus kedua cookie, dipakai saat keluar atau sesi ditolak.
func (c Config) Clear(ctx *gin.Context) {
	for _, name := range []string{c.SessionName, c.CSRFName} {
		http.SetCookie(ctx.Writer, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Domain:   c.Domain,
			MaxAge:   -1,
			HttpOnly: name == c.SessionName,
			Secure:   c.Secure,
			SameSite: c.SameSite,
		})
	}
}

// CookieValue membaca nilai cookie; kosong berarti tidak ada.
func CookieValue(ctx *gin.Context, name string) string {
	value, err := ctx.Cookie(name)
	if err != nil {
		return ""
	}
	return value
}

// CSRFValid membandingkan header dengan cookie secara constant-time.
// Permintaan dianggap sah hanya bila keduanya ada dan identik.
func CSRFValid(ctx *gin.Context, cookieName string) bool {
	cookieValue := CookieValue(ctx, cookieName)
	headerValue := ctx.GetHeader(CookieHeader)
	if cookieValue == "" || headerValue == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookieValue), []byte(headerValue)) == 1
}

// NewToken membuat nilai acak 32 byte berformat base64url.
func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
