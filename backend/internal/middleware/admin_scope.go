package middleware

import "github.com/gin-gonic/gin"

// TandaiSesiAdmin mencatat bahwa permintaan lolos middleware sesi admin.
//
// Dipakai grup yang menerima dua jenis sesi sekaligus: sesi pengguna (cookie
// user) dan sesi admin (cookie admin). Tanpa penanda ini, handler tidak bisa
// membedakan "pemilik laporan" dari "admin yang sedang triage", padahal
// keduanya sah membuka lampiran yang sama.
//
// Penanda ini TIDAK memberi wewenang apa pun; ia hanya melaporkan jalur mana
// yang dilewati permintaan. Wewenang tetap dipegang RequirePermission.
func TandaiSesiAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ContextAdminScope, true)
		c.Next()
	}
}

// CurrentAdminScope melaporkan apakah permintaan datang lewat sesi admin.
func CurrentAdminScope(c *gin.Context) bool {
	return c.GetBool(ContextAdminScope)
}
