// Package redact menyaring rahasia dari teks yang akan dicatat atau disimpan.
//
// Pertahanan berlapis: paket asal (mis. telegram) sudah menyaring tokennya
// sendiri, tetapi kolom seperti last_error juga bisa terisi error dari sumber
// lain. Saringan ini memastikan tidak ada bentuk rahasia umum yang lolos ke
// log, database, atau balasan API.
package redact

import "regexp"

const penanda = "<disembunyikan>"

var (
	// Token bot Telegram: <angka>:<base64url panjang>.
	tokenBotTelegram = regexp.MustCompile(`\d{6,}:[A-Za-z0-9_-]{30,}`)

	// JWT tiga bagian, seperti access token broker Match-Trader.
	tokenJWT = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}`)

	// Nilai header Authorization yang ikut tercetak.
	headerBearer = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._-]{8,}`)

	// Parameter sensitif pada query string atau JSON.
	parameterRahasia = regexp.MustCompile(
		`(?i)("?(?:token|password|secret|api[_-]?key|access[_-]?token)"?\s*[:=]\s*"?)[^"&,\s}]+`,
	)
)

// Secrets mengganti setiap pola rahasia dengan penanda tetap.
func Secrets(text string) string {
	if text == "" {
		return ""
	}

	out := tokenJWT.ReplaceAllString(text, penanda)
	out = tokenBotTelegram.ReplaceAllString(out, penanda)
	out = headerBearer.ReplaceAllString(out, "Bearer "+penanda)
	out = parameterRahasia.ReplaceAllString(out, "${1}"+penanda)
	return out
}

// Error mengubah error menjadi teks aman; error nil menjadi string kosong.
func Error(err error) string {
	if err == nil {
		return ""
	}
	return Secrets(err.Error())
}
