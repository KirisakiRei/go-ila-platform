package ingest

import (
	"strings"
	"unicode"
)

// NormalizeTimeframe membakukan penulisan timeframe dari engine.
//
// Engine belum menetapkan kontrak untuk field htf/ltf, jadi fungsi ini dibuat
// toleran: "M15", "15m", "m15", " 15M " semuanya menjadi "M15". Tujuannya agar
// data dari engine versi mana pun tetap bisa dibandingkan dengan pilihan user
// tanpa menebak formatnya lebih dulu.
//
// Nilai yang tidak dikenali dikembalikan apa adanya dalam huruf besar, sehingga
// timeframe baru dari engine tetap tersimpan walau belum kita kenal. String
// kosong berarti "belum dikirim engine" dan sengaja tidak diisi nilai karangan.
func NormalizeTimeframe(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	// Hanya huruf dan angka: buang pemisah seperti "H4-M15" yang mungkin
	// terkirim sebagai satu nilai.
	cleaned := make([]rune, 0, len(trimmed))
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cleaned = append(cleaned, unicode.ToUpper(r))
		}
	}
	return string(cleaned)
}

// TimeframeLabel mengubah nilai baku menjadi label yang enak dibaca manusia,
// mis. "M15" menjadi "M15" dan "W1" menjadi "W1". Nilai kosong menjadi ""
// supaya pemanggil bisa memilih tidak menampilkan apa pun.
func TimeframeLabel(value string) string {
	return strings.TrimSpace(value)
}
