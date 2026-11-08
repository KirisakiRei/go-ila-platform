package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// tokenFingerprint adalah sidik jari token broker untuk mendeteksi dua akun
// yang memakai kredensial sama.
//
// Disimpan sebagai HMAC-SHA256 dengan kunci rahasia milik server, sehingga
// isi database tidak bisa dipakai membalik token. Tujuannya bukan keamanan
// penyimpanan (itu tugas enkripsi), melainkan mengenali duplikatnya.
func tokenFingerprint(secret, token string) string {
	sum := sha256.Sum256([]byte(secret + "\x00" + strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
