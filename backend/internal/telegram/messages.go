package telegram

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
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

// SetupMessageParams memuat parameter lengkap notifikasi setup valid.
type SetupMessageParams struct {
	Symbol          string
	Direction       string
	StrategyVersion string
	Htf             string
	Ltf             string
	EntryPrice      float64
	StopLoss        float64
	TakeProfit      float64
	RiskReward      float64
	RiskPercent     float64
	RiskAmount      float64
	ApprovalLink    string
}

// FormatSetupDetailedMessage menyusun pesan "setup valid" lengkap untuk Tahap 3 (Ready to Entry).
func FormatSetupDetailedMessage(p SetupMessageParams) Message {
	var b strings.Builder

	strategy := p.StrategyVersion
	if strategy == "" {
		strategy = "ILA"
	}
	fmt.Fprintf(&b, "🟢 SETUP VALID — %s\n", p.Symbol)
	fmt.Fprintf(&b, "Arah: %s  |  Strategi: %s\n", strings.ToUpper(p.Direction), strategy)

	if p.Htf != "" && p.Ltf != "" {
		if p.StrategyVersion == "ila-3.0" {
			mtf := getILA3MTF(p.Htf, p.Ltf)
			fmt.Fprintf(&b, "Timeframe: %s ➔ %s ➔ %s\n", p.Htf, mtf, p.Ltf)
		} else {
			fmt.Fprintf(&b, "Timeframe: %s ➔ %s\n", p.Htf, p.Ltf)
		}
	}

	b.WriteString("\nTahap: 3/4 — Setup Sempurna (Ready to Entry)\n")
	b.WriteString("Kondisi: Konfirmasi struktur lengkap. Siap masuk pada open candle berikutnya!\n\n")

	b.WriteString("💎 Rincian Eksekusi:\n")
	fmt.Fprintf(&b, "• Entry Price: %s\n", formatPrice(p.EntryPrice))
	fmt.Fprintf(&b, "• Stop Loss (SL): %s\n", formatPrice(p.StopLoss))
	fmt.Fprintf(&b, "• Take Profit (TP): %s\n", formatPrice(p.TakeProfit))
	fmt.Fprintf(&b, "• Risk to Reward: 1:%s\n", formatFrom(p.RiskReward))

	if p.RiskPercent > 0 || p.RiskAmount > 0 {
		fmt.Fprintf(&b, "• Risiko Akun: %s%% ≈ %s\n", formatFrom(p.RiskPercent), formatMoney(p.RiskAmount))
	}
	b.WriteString("\n")

	b.WriteString("⚠️ Pintu konfirmasi terbuka maksimal 5 menit:\n")
	if p.ApprovalLink != "" {
		fmt.Fprintf(&b, "[ ⚡ EKSEKUSI ORDER SEKARANG ](%s)\n", p.ApprovalLink)
	} else {
		b.WriteString("Buka dashboard untuk mengonfirmasi order.\n")
	}

	return Message{Text: b.String()}
}

// FormatSetupMessage menyusun pesan "setup valid" untuk mode manual (kompatibilitas mundur).
// Angka memakai gaya Indonesia: titik ribuan, koma desimal.
func FormatSetupMessage(symbol, side, strategy string, entry, sl, tp, rr, riskPercent, riskAmount float64, link string) Message {
	return FormatSetupDetailedMessage(SetupMessageParams{
		Symbol:          symbol,
		Direction:       side,
		StrategyVersion: strategy,
		EntryPrice:      entry,
		StopLoss:        sl,
		TakeProfit:      tp,
		RiskReward:      rr,
		RiskPercent:     riskPercent,
		RiskAmount:      riskAmount,
		ApprovalLink:    link,
	})
}

// RadarHTFParams adalah masukan untuk notifikasi radar tahap 1 (Cek HTF).
type RadarHTFParams struct {
	Symbol          string
	Direction       string // BUY, SELL, atau NEUTRAL
	StrategyVersion string // e.g. "ILA 2.0"
	Htf             string // e.g. "H4"
	Ltf             string // e.g. "M15"
	TargetDOL       float64
	StopLoss        float64
	Details         string
	DashboardLink   string
}

// FormatRadarHTFMessage menyusun pesan Tahap 1: Pemantauan Struktur Timeframe Besar (HTF).
func FormatRadarHTFMessage(p RadarHTFParams) Message {
	var b strings.Builder

	strategy := p.StrategyVersion
	if strategy == "" {
		strategy = "ILA"
	}
	fmt.Fprintf(&b, "⏳ RADAR %s — %s\n", strings.ToUpper(strategy), p.Symbol)

	arah := strings.ToUpper(p.Direction)
	if arah == "" || arah == "NEUTRAL" {
		b.WriteString("Arah Pantauan: Memantau Reaksi Harga\n")
	} else {
		fmt.Fprintf(&b, "Arah Pantauan: %s (Potensi Reversal)\n", arah)
	}

	htf := p.Htf
	if htf == "" {
		htf = "HTF"
	}
	ltf := p.Ltf
	if ltf == "" {
		ltf = "LTF"
	}
	if p.StrategyVersion == "ila-3.0" {
		mtf := getILA3MTF(htf, ltf)
		fmt.Fprintf(&b, "Timeframe: %s (HTF) ➔ %s (MTF) ➔ %s (LTF)\n\n", htf, mtf, ltf)
	} else {
		fmt.Fprintf(&b, "Timeframe: %s (HTF) ➔ %s (LTF)\n\n", htf, ltf)
	}

	fmt.Fprintf(&b, "Tahap: 1/4 — Pemantauan Struktur Timeframe Besar (%s)\n", htf)
	fmt.Fprintf(&b, "Status: Menunggu Penutupan Candle %s\n\n", htf)

	b.WriteString("📊 Kondisi Market Saat Ini:\n")
	details := Sanitize(p.Details)
	if details == "" {
		fmt.Fprintf(&b, "• %s: Reaksi harga di area likuiditas penting sedang berlangsung.\n", htf)
	} else {
		fmt.Fprintf(&b, "• %s: %s\n", htf, details)
	}
	fmt.Fprintf(&b, "• Syarat Validasi: Candle %s harus resmi closed untuk konfirmasi arah.\n", htf)
	fmt.Fprintf(&b, "• %s: Belum aktif dipindai (standby menunggu konfirmasi %s).\n\n", ltf, htf)

	if p.TargetDOL > 0 || p.StopLoss > 0 {
		b.WriteString("🎯 Rencana Setup Sementara:\n")
		if p.TargetDOL > 0 {
			fmt.Fprintf(&b, "• Target Likuiditas: %s\n", formatPrice(p.TargetDOL))
		}
		if p.StopLoss > 0 {
			fmt.Fprintf(&b, "• Batas Area Aman: %s\n", formatPrice(p.StopLoss))
		}
		b.WriteString("\n")
	}

	if p.DashboardLink != "" {
		fmt.Fprintf(&b, "[ 🔍 Pantau Candle %s di Dashboard ](%s)\n", htf, p.DashboardLink)
	} else {
		b.WriteString("Pantau perkembangan langsung di dashboard.\n")
	}

	return Message{Text: b.String()}
}

// RadarLTFParams adalah masukan untuk notifikasi radar tahap 2 (Cek LTF).
type RadarLTFParams struct {
	Symbol          string
	Direction       string // BUY atau SELL
	StrategyVersion string
	Htf             string
	Ltf             string
	EntryPrice      float64
	StopLoss        float64
	TakeProfit      float64
	RiskReward      float64
	CISDLevel       float64
	Details         string
	DashboardLink   string
}

// FormatRadarLTFMessage menyusun pesan Tahap 2: Menunggu Konfirmasi Timeframe Kecil (LTF).
func FormatRadarLTFMessage(p RadarLTFParams) Message {
	var b strings.Builder

	strategy := p.StrategyVersion
	if strategy == "" {
		strategy = "ILA"
	}
	fmt.Fprintf(&b, "⏳ RADAR %s — %s\n", strings.ToUpper(strategy), p.Symbol)

	arah := strings.ToUpper(p.Direction)
	if arah == "" || arah == "NEUTRAL" {
		arah = "PEMBALIKAN ARAH"
	}
	fmt.Fprintf(&b, "Arah Setup: %s\n", arah)

	htf := p.Htf
	if htf == "" {
		htf = "HTF"
	}
	ltf := p.Ltf
	if ltf == "" {
		ltf = "LTF"
	}
	if p.StrategyVersion == "ila-3.0" {
		mtf := getILA3MTF(htf, ltf)
		fmt.Fprintf(&b, "Timeframe: %s ➔ %s ➔ %s\n\n", htf, mtf, ltf)
	} else {
		fmt.Fprintf(&b, "Timeframe: %s ➔ %s\n\n", htf, ltf)
	}

	fmt.Fprintf(&b, "Tahap: 2/4 — Menunggu Konfirmasi Timeframe Kecil (%s)\n", ltf)
	fmt.Fprintf(&b, "Status: %s Valid! Menunggu Trigger Entry di %s\n\n", htf, ltf)

	b.WriteString("📊 Analisis Multi-Timeframe:\n")
	fmt.Fprintf(&b, "• %s [TERKONFIRMASI]: Struktur timeframe besar valid dengan bias %s.\n", htf, arah)
	details := Sanitize(p.Details)
	if details != "" {
		fmt.Fprintf(&b, "• %s [DALAM PROSES]: %s\n", ltf, details)
	} else if p.CISDLevel > 0 {
		fmt.Fprintf(&b, "• %s [DALAM PROSES]: Perubahan arah struktur di level %s. Menunggu candle %s closed menembus level tersebut.\n", ltf, formatPrice(p.CISDLevel), ltf)
	} else {
		fmt.Fprintf(&b, "• %s [DALAM PROSES]: Memantau konfirmasi penembusan struktur momentum.\n", ltf)
	}
	b.WriteString("\n")

	if p.EntryPrice > 0 || p.StopLoss > 0 || p.TakeProfit > 0 {
		b.WriteString("🎯 Proyeksi Parameter:\n")
		if p.EntryPrice > 0 {
			fmt.Fprintf(&b, "• Estimasi Entry: ~%s\n", formatPrice(p.EntryPrice))
		}
		if p.StopLoss > 0 {
			fmt.Fprintf(&b, "• Stop Loss (SL): %s\n", formatPrice(p.StopLoss))
		}
		if p.TakeProfit > 0 {
			fmt.Fprintf(&b, "• Take Profit (TP): %s\n", formatPrice(p.TakeProfit))
		}
		if p.RiskReward > 0 {
			fmt.Fprintf(&b, "• Estimasi RR: 1:%s\n", formatFrom(p.RiskReward))
		}
		b.WriteString("\n")
	}

	if p.DashboardLink != "" {
		fmt.Fprintf(&b, "[ 🔍 Pantau Radar %s di Dashboard ](%s)\n", ltf, p.DashboardLink)
	} else {
		b.WriteString("Pantau perkembangan langsung di dashboard.\n")
	}

	return Message{Text: b.String()}
}

// PositionOpenedParams adalah masukan untuk notifikasi order yang sudah live di pasar (Tahap 4).
type PositionOpenedParams struct {
	Symbol       string
	Direction    string
	TicketNumber string
	BrokerType   string
	LotSize      float64
	OpenPrice    float64
	StopLoss     float64
	TakeProfit   float64
	OutcomeR     float64
	OpenedAt     time.Time
	PosisiLink   string
}

// FormatPositionOpenedMessage menyusun pesan Tahap 4: Posisi Terpasang & Berjalan di Pasar.
func FormatPositionOpenedMessage(p PositionOpenedParams) Message {
	var b strings.Builder

	fmt.Fprintf(&b, "🚀 POSISI TERPASANG — %s\n", p.Symbol)
	ticket := p.TicketNumber
	if ticket == "" {
		ticket = "PROSES"
	}
	broker := p.BrokerType
	if broker == "" {
		broker = "Broker"
	}
	fmt.Fprintf(&b, "Order: %s #%s  |  Broker: %s\n", strings.ToUpper(p.Direction), ticket, broker)
	b.WriteString("Status: POSISI AKTIF (OPEN)\n\n")

	b.WriteString("Tahap: 4/4 — Posisi Berjalan di Pasar\n")
	waktu := p.OpenedAt
	if waktu.IsZero() {
		waktu = time.Now()
	}
	// Tampilkan waktu WIB (UTC+7)
	wib := waktu.In(time.FixedZone("WIB", 7*3600))
	fmt.Fprintf(&b, "Waktu Masuk: %s WIB\n\n", wib.Format("02 Jan 2006, 15:04:05"))

	b.WriteString("📋 Parameter Tiket Aktif:\n")
	if p.LotSize > 0 {
		fmt.Fprintf(&b, "• Volume / Lot: %s Lot\n", formatThousands(p.LotSize, 2))
	}
	fmt.Fprintf(&b, "• Open Price: %s\n", formatPrice(p.OpenPrice))
	if p.StopLoss > 0 {
		fmt.Fprintf(&b, "• Stop Loss Terpasang: %s\n", formatPrice(p.StopLoss))
	}
	if p.TakeProfit > 0 {
		fmt.Fprintf(&b, "• Take Profit Terpasang: %s\n", formatPrice(p.TakeProfit))
	}
	if p.OutcomeR > 0 {
		fmt.Fprintf(&b, "• Potensi Outcome: +%sR\n", formatFrom(p.OutcomeR))
	}
	b.WriteString("\n")

	b.WriteString("Catatan Risiko:\n")
	b.WriteString("Order dilindungi pengaman trailing / target otomatis. Jangan geser Stop Loss melewati batas risiko terukur.\n\n")

	if p.PosisiLink != "" {
		fmt.Fprintf(&b, "[ 📊 Pantau Live PnL di Dashboard ](%s)\n", p.PosisiLink)
	} else {
		b.WriteString("Pantau posisi langsung di dashboard.\n")
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

// FormatOpportunityMessage menyusun pesan untuk setup yang TIDAK dieksekusi
// otomatis karena pasangan timeframe-nya hanya dipantau.
//
// Bedanya dengan FormatHeldMessage: ini bukan penolakan, melainkan pilihan user.
// Karena itu pesannya menyebut pasangan timeframe-nya dan memberi jalan untuk
// memutuskan, bukan sekadar memberi tahu bahwa setup tidak dijalankan.
func FormatOpportunityMessage(symbol, side, strategy, htf, ltf string, entry, sl, tp, rr float64, link string) Message {
	var b strings.Builder

	b.WriteString("🔔 PELUANG ENTRY — ")
	b.WriteString(symbol)
	b.WriteString("\n")

	pasangan := htf
	if ltf != "" {
		if strategy == "ila-3.0" {
			mtf := getILA3MTF(htf, ltf)
			pasangan = fmt.Sprintf("%s ➔ %s ➔ %s", htf, mtf, ltf)
		} else {
			pasangan = htf + "-" + ltf
		}
	}
	if pasangan != "" {
		fmt.Fprintf(&b, "Timeframe: %s\n", pasangan)
	}

	fmt.Fprintf(&b, "Arah: %s  |  %s\n", side, strategy)
	fmt.Fprintf(&b, "Entry: %s\n", formatPrice(entry))
	fmt.Fprintf(&b, "SL: %s  |  TP: %s  (RR 1:%s)\n\n", formatPrice(sl), formatPrice(tp), formatFrom(rr))

	b.WriteString("Eksekusi otomatis tidak aktif untuk pasangan ini pada akun Anda.\n")
	if link != "" {
		fmt.Fprintf(&b, "[ Eksekusi Manual ](%s)\n", link)
	} else {
		b.WriteString("Buka dashboard untuk memutuskan.\n")
	}

	return Message{Text: b.String()}
}

// FormatFinishedMessage memberi tahu bahwa setup berakhir sebelum entry.
func FormatFinishedMessage(symbol, reason string) Message {
	return Message{Text: fmt.Sprintf(
		"⏹ SETUP SELESAI — %s\n%s\nTidak ada aksi yang perlu dilakukan.\n",
		symbol, reason,
	)}
}

// FormatRadarStepMessage menyusun notifikasi kemajuan radar setup tanpa membocorkan istilah internal.
func FormatRadarStepMessage(symbol, strategy, htf, ltf, stepName, link string) Message {
	var b strings.Builder
	fmt.Fprintf(&b, "⏳ RADAR %s — %s\n", strings.ToUpper(strategy), symbol)
	if htf != "" && ltf != "" {
		fmt.Fprintf(&b, "Timeframe: %s ➔ %s\n", htf, ltf)
	}
	fmt.Fprintf(&b, "Tahap: %s\n", stepName)
	b.WriteString("Menunggu konfirmasi penutupan harga.\n\n")
	if link != "" {
		fmt.Fprintf(&b, "[ Buka Radar Dashboard ](%s)\n", link)
	} else {
		b.WriteString("Pantau perkembangan langsung di dashboard.\n")
	}
	return Message{Text: b.String()}
}

// FormatClosedMessage merangkum posisi yang sudah ditutup.
func FormatClosedMessage(symbol, result string, entry, exit, pnl, outcomeR float64, dailyNote string, link ...string) Message {
	var b strings.Builder
	fmt.Fprintf(&b, "🎯 POSISI DITUTUP — %s\nHasil: %s (%sR)\nEntry %s → Exit %s\nPnL: %s\n\n%s\n",
		symbol, result, formatFrom(outcomeR), formatPrice(entry), formatPrice(exit),
		formatMoney(pnl), dailyNote,
	)
	if len(link) > 0 && link[0] != "" {
		fmt.Fprintf(&b, "\n[ 📊 Riwayat Posisi di Dashboard ](%s)\n", link[0])
	}
	return Message{Text: b.String()}
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

// getILA3MTF menentukan timeframe medium (MTF) dari ILA 3.0 berdasarkan HTF dan LTF.
func getILA3MTF(htf, ltf string) string {
	h := strings.ToUpper(strings.TrimSpace(htf))
	l := strings.ToUpper(strings.TrimSpace(ltf))
	if (h == "4H" || h == "H4") && (l == "5M" || l == "M5") {
		return "1H"
	}
	if (h == "D" || h == "D1" || h == "1D") && (l == "15M" || l == "M15") {
		return "4H"
	}
	if (h == "1H" || h == "H1") && (l == "5M" || l == "M5") {
		return "15M"
	}
	if (h == "W" || h == "W1" || h == "1W") && (l == "1H" || l == "H1") {
		return "4H"
	}
	return "MTF"
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
