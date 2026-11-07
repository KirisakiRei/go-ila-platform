package audit

import (
	"encoding/json"
	"strings"
)

// sensitiveKeys adalah awalan nama kolom yang tidak boleh masuk before/after.
// Pencocokan memakai "mengandung", karena kunci bisa berbentuk
// password_hash, broker_token, atau api_secret.
var sensitiveKeys = []string{
	"password", "passwd", "secret", "token", "credential", "cipher",
	"apikey", "api_key", "private", "nonce", "wrapped", "recovery",
	"fingerprint", "ip_hash", "authorization", "cookie", "session",
}

// IsSensitiveKey menandai kunci yang harus disaring dari keadaan audit.
func IsSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, needle := range sensitiveKeys {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

// Scrub menyaring nilai rahasia dari peta keadaan.
// Hasilnya selalu dapat diserialisasi; kegagalan serialisasi menghasilkan
// teks kosong agar audit tidak pernah memuat data mentah.
func Scrub(state map[string]any) string {
	if len(state) == 0 {
		return ""
	}
	clean := make(map[string]any, len(state))
	for key, value := range state {
		if IsSensitiveKey(key) {
			clean[key] = "[disaring]"
			continue
		}
		clean[key] = value
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return ""
	}
	return string(raw)
}

// ContainsSensitive memeriksa teks JSON yang sudah tersimpan. Dipakai tes dan
// tampilan admin untuk memastikan tidak ada rahasia yang lolos.
func ContainsSensitive(payload string) bool {
	if payload == "" {
		return false
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		return false
	}
	for key, value := range decoded {
		if IsSensitiveKey(key) && value != "[disaring]" {
			return true
		}
	}
	return false
}
