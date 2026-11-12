package middleware

import (
	"crypto/rand"
	"encoding/hex"
)

// ContextUserID adalah kunci context Gin untuk ID akun yang sudah terautentikasi.
const ContextUserID = "user_id"

// ContextRole adalah kunci context Gin untuk peran akun.
const ContextRole = "user_role"

// ContextAdminScope menandai bahwa permintaan lolos middleware sesi ADMIN.
// Dipakai route yang menerima kedua jenis sesi (mis. unduh lampiran
// laporan bug) untuk membedakan wewenang admin dari wewenang pemilik.
const ContextAdminScope = "admin_scope"

// ContextSessionID adalah kunci context Gin untuk ID sesi perangkat ini.
const ContextSessionID = "session_id"

// ContextRequestID adalah kunci context Gin untuk nomor permintaan.
// Nilai ini disalin ke baris audit agar satu permintaan bisa ditelusuri.
const ContextRequestID = "request_id"

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Kegagalan sumber acak tidak boleh menghentikan permintaan;
		// ID tetap harus ada agar log bisa dilacak.
		return "req-unknown"
	}
	return hex.EncodeToString(b)
}
