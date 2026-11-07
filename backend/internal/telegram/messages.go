package telegram

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// Istilah internal yang dilarang muncul di teks untuk user (plans/12 §7.1).
// "ila" sengaja tidak masuk daftar: itu nama produk yang justru wajib tampil.
var forbiddenTerms = []string{
	"fvg", "fair value gap", "cisd", "change in state of delivery",
	"sweep", "sweept", "imaginary liquidity", "draw on liquidity", "dol",
	"body break body", "candle 4", "candle 5", "htf fvg", "internal liquidity",
	"external range liquidity", "erl", "irl", "displacement", "mitigation",
}

var (
	patternOnce sync.Once
	pattern     *regexp.Regexp
)

// forbiddenPattern menyusun satu regex dengan batas kata, supaya "erl" tidak
// cocok di dalam kata lain dan "dol" tidak menangkap "dollar".
func forbiddenPattern() *regexp.Regexp {
	patternOnce.Do(func() {
		parts := make([]string, 0, len(forbiddenTerms))
		for _, term := range forbiddenTerms {
			parts = append(parts, regexp.QuoteMeta(term))
		}
		pattern = regexp.MustCompile(`(?i)\b(?:` + strings.Join(parts, "|") + `)\b`)
	})
	return pattern
}

// ContainsForbiddenTerms memeriksa apakah teks membocorkan istilah strategi.
func ContainsForbiddenTerms(text string) bool {
	return forbiddenPattern().MatchString(text)
}

// Message adalah isi notifikasi siap kirim.
type Message struct {
	Text string
	// Key mencegah pesan yang sama terkirim dua kali.
	Key string
}

// FormatSetupMessage menyusun pesan "setup valid" untuk mode manual.
// Angka memakai gaya Indonesia: titik ribuan, koma desimal.
func FormatSetupMessage(symbol, side, strategy string, entry, sl, tp, rr, riskPercent, riskAmount float64, link string) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "🟢 SETUP VALID — %s\n", symbol)
	fmt.Fprintf(&b, "Arah: %s  |  %s\n", side, strategy)
	fmt.Fprintf(&b, "Entry: %s\n", formatPrice(entry))
	fmt.Fprintf(&b, "SL: %s  |  TP: %s  (RR 1:%s)\n", formatPrice(sl), formatPrice(tp), formatFrom(rr))
	fmt.Fprintf(&b, "Risiko: %s%% ≈ %s\n\n", formatFrom(riskPercent), formatMoney(riskAmount))
	b.WriteString("Konfirmasi maksimal 5 menit:\n")

	if link != "" {
		fmt.Fprintf(&b, "[ Konfirmasi Entry ](%s)\n", link)
	} else {
		b.WriteString("Buka dashboard untuk mengonfirmasi.\n")
	}

	return Message{Text: b.String()}
}

// FormatHeldMessage menyusun pesan untuk sinyal yang tertahan aturan harian.
// Sinyal tetap dikirim walau tidak dieksekusi (plans/05 §3).
func FormatHeldMessage(symbol, side, strategy string, rr float64, reason string) Message {
	text := fmt.Sprintf(
		"⏸ SETUP TERLIHAT — %s\nArah: %s  |  %s\nRR 1:%s\n\nTidak dieksekusi: %s\nSinyal tetap ditampilkan untuk pemantauan.\n",
		symbol, side, strategy, formatFrom(rr), reason,
	)
	return Message{Text: text}
}

// FormatFinishedMessage memberi tahu bahwa setup berakhir sebelum entry.
func FormatFinishedMessage(symbol, reason string) Message {
	return Message{Text: fmt.Sprintf(
		"⏹ SETUP SELESAI — %s\n%s\nTidak ada aksi yang perlu dilakukan.\n",
		symbol, reason,
	)}
}

// FormatClosedMessage merangkum posisi yang sudah ditutup.
func FormatClosedMessage(symbol, result string, entry, exit, pnl, outcomeR float64, dailyNote string) Message {
	text := fmt.Sprintf(
		"🎯 POSISI DITUTUP — %s\nHasil: %s (%sR)\nEntry %s → Exit %s\nPnL: %s\n\n%s\n",
		symbol, result, formatFrom(outcomeR), formatPrice(entry), formatPrice(exit),
		formatMoney(pnl), dailyNote,
	)
	return Message{Text: text}
}

// Sanitize mengganti teks yang membocorkan istilah internal dengan versi aman.
// Ringkasan tetap disampaikan; hanya istilahnya yang disederhanakan.
func Sanitize(text string) string {
	if !ContainsForbiddenTerms(text) {
		return text
	}
	return "Setup sesuai rencana trading. Detail level tersedia di dashboard."
}

func formatPrice(v float64) string {
	return formatThousands(v, decimalsFor(v))
}

func formatMoney(v float64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	return sign + "$" + formatThousands(v, 2)
}

func formatFrom(v float64) string {
	return formatThousands(v, 2)
}

func decimalsFor(v float64) int {
	if v >= 10 {
		return 2
	}
	return 5
}

// formatThousands menulis angka dengan titik ribuan dan koma desimal.
func formatThousands(v float64, decimals int) string {
	raw := fmt.Sprintf("%.*f", decimals, v)
	parts := strings.SplitN(raw, ".", 2)

	head := parts[0]
	var b strings.Builder
	for i, ch := range head {
		if i > 0 && (len(head)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}

	if len(parts) == 2 {
		return b.String() + "," + parts[1]
	}
	return b.String()
}
