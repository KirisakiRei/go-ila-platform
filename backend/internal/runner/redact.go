package runner

import (
	"strings"

	"github.com/KirisakiRei/go-ila-platform/backend/pkg/redact"
)

// RedactSecrets adalah pintu pendek ke pkg/redact untuk paket runner.
func RedactSecrets(text string) string {
	return redact.Secrets(text)
}

// errText mengubah error menjadi teks yang aman disimpan.
func errText(err error) string {
	return redact.Error(err)
}

// truncate memotong teks pada batas kolom.
func truncate(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}
