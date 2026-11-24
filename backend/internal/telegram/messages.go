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
	// Buttons memuat tombol tautan inline keyboard di bawah pesan.
	Buttons [][]InlineButton
}

// ---------------------------------------------------------------------------
// Kerangka pesan
// ---------------------------------------------------------------------------

// Tahap siklus setup yang ditampilkan di setiap notifikasi.
//
// Nomor tahap dipakai user untuk tahu sedang di mana prosesnya, jadi nilainya
// harus sama dengan urutan cerita di dashboard: struktur besar, konfirmasi
// arah, tunggu trigger, siap masuk, posisi jalan, posisi selesai.
const (
	tahapStruktur   = "1/6"
	tahapKonfirmasi = "2/6"
	tahapTrigger    = "3/6"
	tahapSiap       = "4/6"
	tahapPosisi     = "5/6"
	tahapSelesai    = "6/6"
)

// judulPesan menyusun baris pertama: emoji, pair, pasangan timeframe, lalu
// keadaan sekarang. Dua baris ini yang dibaca user duluan di notifikasi.
func judulPesan(emoji, symbol, htf, ltf, keadaan string) string {
	pasangan := strings.TrimSpace(htf)
	if ltf != "" {
		if pasangan != "" {
			pasangan += " ➔ " + ltf
		} else {
			pasangan = ltf
		}
	}
	if pasangan == "" {
		return fmt.Sprintf("%s %s\n%s\n", emoji, symbol, keadaan)
	}
	return fmt.Sprintf("%s %s · %s\n%s\n", emoji, symbol, pasangan, keadaan)
}

// makeBox merender baris label-nilai sebagai kotak monospace.
//
// Tag <pre> dipakai, bukan backtick: client mengirim tanpa parse_mode
// sehingga backtick tampil mentah di Telegram.
//
// Isi nilai di-escape karena sebagian berasal dari engine (mis. keterangan
// alasan) dan bisa memuat karakter khusus HTML yang merusak pesan.
func makeBox(rows [][2]string) string {
	lebar := 0
	for _, row := range rows {
		if row[1] == "" {
			continue
		}
		if len(row[0]) > lebar {
			lebar = len(row[0])
		}
	}
	if lebar == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("<pre>")
	for _, row := range rows {
		if row[1] == "" {
			continue
		}
		fmt.Fprintf(&b, "│ %-*s │ %s\n", lebar, row[0], escapeHTML(row[1]))
	}
	b.WriteString("</pre>")
	return b.String()
}

// daftar merender baris label-nilai dengan gaya daftar pohon.
//
// Sengaja tanpa <pre>: blok monospace membuat Telegram menampilkan tombol
// salin, yang tidak berguna untuk angka posisi. Kolom nilai jadi tidak sejajar,
// dan itu memang disengaja.
func daftar(rows [][2]string) string {
	var b strings.Builder
	for _, row := range rows {
		if row[1] == "" {
			continue
		}
		fmt.Fprintf(&b, "└ %s: %s\n", row[0], escapeHTML(row[1]))
	}
	return b.String()
}

// escapeHTML mengamankan teks bebas sebelum dikirim dengan parse_mode HTML.
func escapeHTML(text string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(text)
}

// arahTampil menyeragamkan arah posisi untuk ditampilkan.
func arahTampil(direction string) string {
	arah := strings.ToUpper(strings.TrimSpace(direction))
	switch arah {
	case "":
		return "NETRAL"
	case "BULLISH":
		return "BUY"
	case "BEARISH":
		return "SELL"
	default:
		return arah
	}
}

// risikoTampil menyusun baris risiko, kosong bila angkanya belum ada.
func risikoTampil(persen, nominal float64, lot float64) string {
	bagian := make([]string, 0, 2)
	if persen > 0 || nominal > 0 {
		bagian = append(bagian, fmt.Sprintf("%s%% (%s)", formatFrom(persen), formatMoney(nominal)))
	}
	if lot > 0 {
		bagian = append(bagian, formatThousands(lot, 2)+" lot")
	}
	return strings.Join(bagian, " · ")
}

// tombolRadar membungkus satu tombol tautan opsional.
func tombolRadar(label, link string) [][]InlineButton {
	if link == "" {
		return nil
	}
	return [][]InlineButton{{{Text: label, URL: link}}}
}

// ---------------------------------------------------------------------------
// Tahap 4 — Setup siap dieksekusi
// ---------------------------------------------------------------------------

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
	LotSize         float64
	ApprovalLink    string
	// ApproveToken mengubah tombol menjadi aksi langsung di Telegram.
	ApproveToken string
}

// FormatSetupDetailedMessage menyusun pesan Tahap 4: setup lengkap dan siap masuk.
func FormatSetupDetailedMessage(p SetupMessageParams) Message {
	var b strings.Builder

	strategi := p.StrategyVersion
	if strategi == "" {
		strategi = "ILA"
	}
	b.WriteString(judulPesan("🟢", p.Symbol, p.Htf, p.Ltf, "Tahap "+tahapSiap+" · Siap masuk"))
	b.WriteString("\n")

	b.WriteString(makeBox([][2]string{
		{"Arah", arahTampil(p.Direction)},
		{"Metode", metodeTampil(strategi)},
		{"Entry", formatPrice(p.EntryPrice)},
		{"Stop", formatPrice(p.StopLoss)},
		{"Target", formatPrice(p.TakeProfit)},
		{"RR", "1:" + formatFrom(p.RiskReward)},
		{"Risiko", risikoTampil(p.RiskPercent, p.RiskAmount, p.LotSize)},
	}))

	b.WriteString("\nSetup memiliki batas waktu entry. Tinjau level dan risiko sebelum mengonfirmasi.\n")

	// Tombol aksi memakai callback agar keputusan selesai di Telegram, tanpa
	// membuka browser saat harga bergerak. Bila token gagal dibuat, tombol
	// kembali menjadi tautan dashboard supaya user masih punya jalan.
	if p.ApproveToken != "" {
		return Message{Text: b.String(), Buttons: [][]InlineButton{{
			{Text: "⚡ Eksekusi", CallbackData: CallbackEksekusi + p.ApproveToken, Style: "success"},
			{Text: "Lewati", CallbackData: CallbackLewati + p.ApproveToken},
		}}}
	}

	label := "🔍 Detail Setup"
	if p.ApprovalLink != "" {
		label = "⚡ Eksekusi Manual"
	}
	return Message{Text: b.String(), Buttons: tombolRadar(label, p.ApprovalLink)}
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

// ---------------------------------------------------------------------------
// Tahap 1 & 2 — Struktur timeframe besar
// ---------------------------------------------------------------------------

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
	Siklus          string // e.g. "Candle H4 · 20:00 NY"
	DashboardLink   string
}

// FormatRadarHTFMessage menyusun pesan Tahap 1: memantau struktur timeframe besar.
func FormatRadarHTFMessage(p RadarHTFParams) Message {
	htf := p.Htf
	if htf == "" {
		htf = "HTF"
	}

	bias := arahTampil(p.Direction)
	if bias == "" {
		bias = "NETRAL"
	}

	var b strings.Builder
	b.WriteString(judulPesan("⏳", p.Symbol, p.Htf, p.Ltf, "Tahap "+tahapStruktur+" · Baca arah "+htf))
	b.WriteString("\n")

	b.WriteString(makeBox([][2]string{
		{"Strategi", metodeTampil(p.StrategyVersion)},
		{"Tahap", tahapStruktur + " · Baca " + htf},
		{"Waktu", p.Siklus},
		{"Bias", bias},
		{"Target", formatHargaOpsional(p.TargetDOL)},
		{"Batas", formatHargaOpsional(p.StopLoss)},
	}))

	b.WriteString("\nMemantau " + htf + " untuk menentukan arah berikutnya.\n")
	fmt.Fprintf(&b, "Tunggu candle %s ditutup untuk mengonfirmasi arah.\n", htf)

	if detail := Sanitize(p.Details); detail != "" {
		fmt.Fprintf(&b, "\n%s\n", detail)
	}

	return Message{Text: b.String(), Buttons: tombolRadar("🔍 Buka Radar "+htf, p.DashboardLink)}
}

// FormatRadarSweepMessage menyusun pesan Tahap 2: area HTF sudah tersentuh harga.
//
// Dipisah dari Tahap 1 karena keadaannya berbeda: arah sudah mengarah, tinggal
// menunggu candle HTF tutup. Menyatukannya membuat user tidak bisa membedakan
// "belum ada reaksi" dan "reaksi sudah terjadi".
func FormatRadarSweepMessage(p RadarHTFParams) Message {
	htf := p.Htf
	if htf == "" {
		htf = "HTF"
	}

	var b strings.Builder
	b.WriteString(judulPesan("👀", p.Symbol, p.Htf, p.Ltf, "Tahap "+tahapKonfirmasi+" · Area "+htf+" tersentuh"))
	b.WriteString("\n")

	b.WriteString(makeBox([][2]string{
		{"Strategi", metodeTampil(p.StrategyVersion)},
		{"Tahap", tahapKonfirmasi + " · Tunggu close " + htf},
		{"Waktu", p.Siklus},
		{"Bias", arahTampil(p.Direction)},
		{"Target", formatHargaOpsional(p.TargetDOL)},
		{"Batas", formatHargaOpsional(p.StopLoss)},
	}))

	fmt.Fprintf(&b, "\nHarga telah menyentuh area %s.\n", htf)
	fmt.Fprintf(&b, "Tunggu candle %s ditutup untuk mengonfirmasi arah %s.\n", htf, arahTampil(p.Direction))

	return Message{Text: b.String(), Buttons: tombolRadar("🔍 Buka Radar "+htf, p.DashboardLink)}
}

// ---------------------------------------------------------------------------
// Tahap 3 — Menunggu trigger timeframe kecil
// ---------------------------------------------------------------------------

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
	Siklus          string
	DashboardLink   string
}

// FormatRadarLTFMessage menyusun pesan Tahap 3: menunggu trigger di timeframe kecil.
func FormatRadarLTFMessage(p RadarLTFParams) Message {
	ltf := p.Ltf
	if ltf == "" {
		ltf = "LTF"
	}

	var b strings.Builder
	b.WriteString(judulPesan("⏳", p.Symbol, p.Htf, p.Ltf, "Tahap "+tahapTrigger+" · Cari entry "+ltf))
	b.WriteString("\n")

	b.WriteString(makeBox([][2]string{
		{"Strategi", metodeTampil(p.StrategyVersion)},
		{"Tahap", tahapTrigger + " · Entry " + ltf},
		{"Waktu", p.Siklus},
		{"Bias", arahTampil(p.Direction)},
		{"Target", formatHargaOpsional(p.TakeProfit)},
		{"Batas", formatHargaOpsional(p.StopLoss)},
		{"Pantau", formatHargaOpsional(p.CISDLevel)},
	}))

	if p.CISDLevel > 0 {
		fmt.Fprintf(&b, "\nTunggu candle %s ditutup melewati %s untuk mengonfirmasi pemicu entry.\n",
			ltf, formatPrice(p.CISDLevel))
	} else {
		// Tingkat acuan belum terbentuk. User tetap diberi tahu apa yang
		// ditunggu dan di harga mana batasnya, bukan kalimat kosong.
		fmt.Fprintf(&b, "\nLevel acuan pada %s belum terbentuk.\n", ltf)
		fmt.Fprintf(&b, "Tunggu candle %s ditutup untuk menentukan pemicu entry.\n", ltf)
		if p.StopLoss > 0 {
			fmt.Fprintf(&b, "Batas risiko yang tersedia berada di %s.\n", formatPrice(p.StopLoss))
		}
	}

	return Message{Text: b.String(), Buttons: tombolRadar("🔍 Pantau "+ltf, p.DashboardLink)}
}

// ---------------------------------------------------------------------------
// Penahanan eksekusi & pembatalan setup
// ---------------------------------------------------------------------------

// AlasanPenahanan mengelompokkan alasan tertahannya eksekusi.
//
// Nilainya sengaja berupa string biasa, bukan tipe dari paket runner, supaya
// paket telegram tidak bergantung pada aturan bisnis Runner.
type AlasanPenahanan string

const (
	PenahananKuota  AlasanPenahanan = "kuota"
	PenahananPosisi AlasanPenahanan = "posisi"
	PenahananAkun   AlasanPenahanan = "akun"
	PenahananPasar  AlasanPenahanan = "pasar"
	PenahananSistem AlasanPenahanan = "sistem"
	PenahananHarga  AlasanPenahanan = "harga"
	PenahananBatal  AlasanPenahanan = "batal"
)

// HeldParams adalah masukan notifikasi setup yang tertahan atau batal.
type HeldParams struct {
	Symbol          string
	Direction       string
	StrategyVersion string
	Htf             string
	Ltf             string
	EntryPrice      float64
	StopLoss        float64
	TakeProfit      float64
	RiskReward      float64
	Alasan          AlasanPenahanan
	// Keterangan adalah kalimat dari Runner yang menjelaskan keadaannya.
	Keterangan string
	// KuotaTerpakai dan KuotaMaks diisi bila alasan tertahan karena aturan harian.
	KuotaTerpakai int
	KuotaMaks     int
	// BatasTunggu diisi bila setup menunggu harga kembali ke level entry.
	BatasTunggu string
	Link        string
}

// FormatHeldMessage menyusun pesan untuk setup yang MASIH berlaku tetapi
// eksekusinya ditahan. Judul dan penutup mengikuti alasan penahanannya supaya
// user tahu apakah masih ada kesempatan atau tidak.
func FormatHeldMessage(p HeldParams) Message {
	if p.Alasan == PenahananBatal || p.Alasan == PenahananHarga {
		return FormatCancelledMessage(p)
	}

	emoji, keadaan, penutup, label, _ := penahananTeks(p.Alasan)

	var b strings.Builder
	b.WriteString(judulPesan(emoji, p.Symbol, p.Htf, p.Ltf, keadaan))
	b.WriteString("\n")

	b.WriteString(makeBox([][2]string{
		{"Arah", arahTampil(p.Direction)},
		{"Metode", metodeTampil(p.StrategyVersion)},
		{"Entry", formatPrice(p.EntryPrice)},
		{"Stop", formatPrice(p.StopLoss)},
		{"Target", formatPrice(p.TakeProfit)},
		{"RR", rrTampil(p.RiskReward)},
		{"Kuota", kuotaTampil(p.KuotaTerpakai, p.KuotaMaks)},
		{"Sebab", p.Keterangan},
	}))

	fmt.Fprintf(&b, "\n%s\n", penutup)

	return Message{Text: b.String(), Buttons: tombolRadar(label, p.Link)}
}

// FormatCancelledMessage menyusun pesan untuk setup yang sudah tidak berlaku.
//
// Dipakai dua keadaan: penahanan karena harga sudah lewat (masih menunggu
// kembali sampai batas waktu) dan pembatalan penuh.
func FormatCancelledMessage(p HeldParams) Message {
	var b strings.Builder

	if p.Alasan == PenahananHarga {
		b.WriteString(judulPesan("⏳", p.Symbol, p.Htf, p.Ltf, "Tahap "+tahapSiap+" · Menunggu harga kembali"))
		b.WriteString("\n")
		b.WriteString(makeBox([][2]string{
			{"Arah", arahTampil(p.Direction)},
			{"Entry", formatPrice(p.EntryPrice)},
			{"Stop", formatPrice(p.StopLoss)},
			{"Target", formatPrice(p.TakeProfit)},
			{"RR", rrTampil(p.RiskReward)},
			{"Batas tunggu", p.BatasTunggu},
		}))
		fmt.Fprintf(&b, "\nHarga udah lewat dari entry %s.\n", formatPrice(p.EntryPrice))
		if p.BatasTunggu != "" {
			fmt.Fprintf(&b, "Nunggu harga balik sebelum %s. Kalau gak balik, setup batal otomatis.\n", p.BatasTunggu)
		} else {
			b.WriteString("Nunggu harga balik ke entry. Kalau gak balik, setup batal otomatis.\n")
		}
		return Message{Text: b.String(), Buttons: tombolRadar("📊 Pantau Harga", p.Link)}
	}

	b.WriteString(judulPesan("⌛", p.Symbol, p.Htf, p.Ltf, "Setup batal"))
	b.WriteString("\n")
	b.WriteString(makeBox([][2]string{
		{"Arah", arahTampil(p.Direction)},
		{"Metode", metodeTampil(p.StrategyVersion)},
		{"Sebab", p.Keterangan},
	}))

	b.WriteString("\nSetup ini sudah gak berlaku. Gak ada posisi yang dibuka.\n")
	b.WriteString("Sekarang nunggu setup berikutnya.\n")

	return Message{Text: b.String(), Buttons: tombolRadar("🔍 Buka Radar", p.Link)}
}

// penahananTeks memilih emoji, judul, penutup, dan tombol per alasan.
//
// penutup wajib menyebut apakah setup masih berlaku. Kalimat yang sama untuk
// semua alasan membuat user mengira setup yang sudah mati masih dipantau.
func penahananTeks(alasan AlasanPenahanan) (emoji, keadaan, penutup, label string, tombol string) {
	switch alasan {
	case PenahananKuota:
		return "🔔", "Tahap " + tahapSiap + " · Batas auto-entry",
			"Kuota auto-entry hari ini udah habis. Kalau mau tetap masuk, eksekusi manual masih bisa.",
			"⚡ Eksekusi Manual", "eksekusi"
	case PenahananPosisi:
		return "🔔", "Tahap " + tahapSiap + " · Ditahan posisi jalan",
			"Masih ada posisi terbuka di akun ini. Auto-entry ditahan, tapi kalau mau tetap masuk, eksekusi manual masih bisa.",
			"⚡ Eksekusi Manual", "eksekusi"
	case PenahananAkun:
		return "⚠️", "Tahap " + tahapSiap + " · Akun ada masalah",
			"Setup tetap dipantau setelah urusan akun selesai. Buka pengaturan buat beresin.",
			"🔧 Buka Pengaturan", "pengaturan"
	case PenahananPasar:
		return "🌙", "Tahap " + tahapSiap + " · Pasar tutup",
			"Pasar lagi tutup. Setup diproses otomatis begitu pasar buka.",
			"", ""
	case PenahananSistem:
		return "🛠", "Tahap " + tahapSiap + " · Ditahan sistem",
			"Entry baru ditahan sementara sama sistem. Setup tetap dipantau.",
			"", ""
	default:
		return "⌛", "Setup batal",
			"Setup sudah gak berlaku.",
			"", ""
	}
}

// ---------------------------------------------------------------------------
// Tahap 5 & 6 — Posisi berjalan dan posisi selesai
// ---------------------------------------------------------------------------

// PositionOpenedParams adalah masukan untuk notifikasi order yang sudah live di pasar.
type PositionOpenedParams struct {
	Symbol       string
	Direction    string
	Htf          string
	Ltf          string
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

// FormatPositionOpenedMessage menyusun pesan Tahap 5: posisi berjalan di pasar.
func FormatPositionOpenedMessage(p PositionOpenedParams) Message {
	var b strings.Builder

	b.WriteString(judulPesan("🚀", p.Symbol, p.Htf, p.Ltf, "Tahap "+tahapPosisi+" · Posisi jalan"))
	b.WriteString("\n")

	order := arahTampil(p.Direction)
	if p.TicketNumber != "" && p.TicketNumber != "PROSES" {
		order += " #" + p.TicketNumber
	}

	masuk := ""
	if !p.OpenedAt.IsZero() {
		wib := p.OpenedAt.In(time.FixedZone("WIB", 7*3600))
		masuk = wib.Format("02 Jan 2006, 15:04") + " WIB"
	}

	b.WriteString(makeBox([][2]string{
		{"Order", order},
		{"Broker", p.BrokerType},
		{"Lot", lotTampil(p.LotSize)},
		{"Entry", formatPrice(p.OpenPrice)},
		{"Stop", formatPrice(p.StopLoss)},
		{"Target", formatPrice(p.TakeProfit)},
		{"RR", rrTampil(p.OutcomeR)},
		{"Masuk", masuk},
	}))

	b.WriteString("\nPosisi telah dibuka. Perkembangan hasil akan dikirim setelah posisi ditutup.\n")

	return Message{Text: b.String(), Buttons: tombolRadar("📊 Lihat Posisi", p.PosisiLink)}
}

// ClosedParams adalah masukan notifikasi posisi yang sudah ditutup.
type ClosedParams struct {
	Symbol      string
	Direction   string
	Htf         string
	Ltf         string
	Result      string
	EntryPrice  float64
	ExitPrice   float64
	ProfitLoss  float64
	OutcomeR    float64
	Duration    string
	DailyNote   string
	RiwayatLink string
}

// FormatClosedMessage merangkum posisi yang sudah ditutup.
func FormatClosedMessage(p ClosedParams) Message {
	var b strings.Builder

	b.WriteString(judulPesan("🏁", p.Symbol, p.Htf, p.Ltf, "Tahap "+tahapSelesai+" · "+p.Result))
	b.WriteString("\n")

	b.WriteString(makeBox([][2]string{
		{"Arah", arahTampil(p.Direction)},
		{"Entry", formatPrice(p.EntryPrice)},
		{"Exit", formatPrice(p.ExitPrice)},
		{"Hasil", hasilR(p.OutcomeR)},
		{"PnL", formatMoney(p.ProfitLoss)},
		{"Durasi", p.Duration},
	}))

	if p.DailyNote != "" {
		fmt.Fprintf(&b, "\n%s\n", p.DailyNote)
	}

	return Message{Text: b.String(), Buttons: tombolRadar("📊 Riwayat Posisi", p.RiwayatLink)}
}

// ---------------------------------------------------------------------------
// Peluang entry & helper tampilan
// ---------------------------------------------------------------------------

// FormatOpportunityMessage menyusun pesan untuk setup yang TIDAK dieksekusi
// otomatis karena pasangan timeframe-nya hanya dipantau.
//
// Bedanya dengan FormatHeldMessage: ini bukan penolakan, melainkan pilihan user.
// Karena itu pesannya menyebut pasangan timeframe-nya dan memberi jalan untuk
// memutuskan, bukan sekadar memberi tahu bahwa setup tidak dijalankan.
func FormatOpportunityMessage(symbol, side, strategy, htf, ltf string, entry, sl, tp, rr float64, link string) Message {
	var b strings.Builder

	b.WriteString(judulPesan("🔔", symbol, htf, ltf, "Tahap "+tahapSiap+" · Siap, ditahan pengaturan"))
	b.WriteString("\n")

	b.WriteString(makeBox([][2]string{
		{"Arah", arahTampil(side)},
		{"Metode", metodeTampil(strategy)},
		{"Entry", formatPrice(entry)},
		{"Stop", formatPrice(sl)},
		{"Target", formatPrice(tp)},
		{"RR", rrTampil(rr)},
	}))

	b.WriteString("\nEksekusi otomatis tidak aktif untuk pasangan timeframe ini.\n")
	b.WriteString("Setup masih berlaku dan bisa dieksekusi manual.\n")

	label := "⚡ Eksekusi Manual"
	if link == "" {
		label = ""
	}
	return Message{Text: b.String(), Buttons: tombolRadar(label, link)}
}

// FormatFinishedMessage memberi tahu bahwa setup berakhir sebelum entry.
func FormatFinishedMessage(symbol, reason string) Message {
	return FormatCancelledMessage(HeldParams{
		Symbol:     symbol,
		Alasan:     PenahananBatal,
		Keterangan: reason,
	})
}

// FormatRadarStepMessage menyusun notifikasi kemajuan radar setup tanpa membocorkan istilah internal.
func FormatRadarStepMessage(symbol, strategy, htf, ltf, stepName, link string) Message {
	var b strings.Builder
	b.WriteString(judulPesan("⏳", symbol, htf, ltf, "Tahap "+tahapStruktur+" · "+stepName))
	b.WriteString("\nMenunggu konfirmasi penutupan harga.\n")
	if strategy != "" {
		fmt.Fprintf(&b, "Metode: %s\n", metodeTampil(strategy))
	}
	return Message{Text: b.String(), Buttons: tombolRadar("🔍 Buka Radar", link)}
}

// Sanitize mengganti teks yang membocorkan istilah internal dengan versi aman.
// Ringkasan tetap disampaikan; hanya istilahnya yang disederhanakan.
// penggantiIstilah menerjemahkan istilah internal ke bahasa yang boleh dibaca
// user. Kalimatnya dipertahankan supaya informasi level dan arah tidak hilang;
// sebelumnya seluruh kalimat diganti sehingga pesan jadi tanpa isi.
var penggantiIstilah = map[string]string{
	"fvg":                         "area",
	"fair value gap":              "area",
	"htf fvg":                     "area besar",
	"internal liquidity":          "area internal",
	"external range liquidity":    "area terluar",
	"erl":                         "area terluar",
	"irl":                         "area internal",
	"imaginary liquidity":         "area likuiditas",
	"draw on liquidity":           "target",
	"dol":                         "target",
	"cisd":                        "konfirmasi arah",
	"change in state of delivery": "konfirmasi arah",
	"sweep":                       "sentuh",
	"sweept":                      "disentuh",
	"body break body":             "penembusan harga",
	"candle 4":                    "candle acuan",
	"candle 5":                    "candle entry",
	"displacement":                "dorongan harga",
	"mitigation":                  "uji area",
}

// Sanitize mengganti istilah internal dengan padanan yang boleh dibaca user.
// Kalimat dan angkanya tetap utuh; hanya istilahnya yang disederhanakan.
func Sanitize(text string) string {
	if !ContainsForbiddenTerms(text) {
		return text
	}

	hasil := text
	for istilah, ganti := range penggantiIstilah {
		hasil = forbiddenTermPattern(istilah).ReplaceAllString(hasil, ganti)
	}

	// Penggantian bisa memunculkan istilah terlarang lain lewat kata pengganti
	// yang kebetulan cocok. Bila masih bocor, kalimatnya diganti utuh.
	if ContainsForbiddenTerms(hasil) {
		return "Setup sesuai rencana trading. Detail level tersedia di dashboard."
	}
	return hasil
}

// forbiddenTermPattern menyusun pola satu istilah dengan batas kata, supaya
// "erl" tidak cocok di dalam kata lain.
func forbiddenTermPattern(term string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(term) + `\b`)
}

// metodeTampil menyeragamkan nama metode, kosong bila tidak ada yang dikirim.
func metodeTampil(strategy string) string {
	s := strings.TrimSpace(strategy)
	switch strings.ToLower(s) {
	case "ila-2.0", "ila 2.0":
		return "ILA 2.0"
	case "ila-3.0", "ila 3.0":
		return "ILA 3.0"
	}
	if s == "" || s == "ILA" {
		return ""
	}
	return s
}

// formatHargaOpsional mengosongkan harga nol supaya baris box dilewati.
func formatHargaOpsional(v float64) string {
	if v <= 0 {
		return ""
	}
	return formatPrice(v)
}

// rrTampil menyusun rasio risiko-imbalan, kosong bila belum ada.
func rrTampil(v float64) string {
	if v <= 0 {
		return ""
	}
	return "1:" + formatFrom(v)
}

// hasilR menampilkan hasil trade sebagai kelipatan risiko, dengan tanda.
func hasilR(v float64) string {
	return fmt.Sprintf("%sR", formatThousands(v, 2))
}

// lotTampil mengosongkan ukuran lot nol.
func lotTampil(v float64) string {
	if v <= 0 {
		return ""
	}
	return formatThousands(v, 2) + " lot"
}

// kuotaTampil menyusun baris kuota harian, kosong bila datanya tidak ada.
func kuotaTampil(terpakai, maks int) string {
	if maks <= 0 {
		return ""
	}
	if terpakai < 0 {
		terpakai = 0
	}
	return fmt.Sprintf("%d dari %d trade otomatis", terpakai, maks)
}

// ---------------------------------------------------------------------------
// Format angka gaya Indonesia
// ---------------------------------------------------------------------------

func formatPrice(v float64) string {
	return formatThousands(v, decimalsFor(v))
}

// decimalsFor memilih jumlah desimal: pair harga tinggi cukup dua angka,
// pair harga rendah butuh lima supaya selisihnya terbaca.
func decimalsFor(v float64) int {
	if v < 0 {
		v = -v
	}
	if v >= 10 {
		return 2
	}
	return 5
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

// formatProfitHarian mengosongkan nominal nol karena tabel ringkasan harian
// tidak menyimpan nilai uang: nol berarti belum tercatat, bukan impas.
func formatProfitHarian(v float64) string {
	if v == 0 {
		return "—"
	}
	return formatMoney(v)
}

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
		b.WriteString(",")
		b.WriteString(parts[1])
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Hasil tekanan tombol
// ---------------------------------------------------------------------------

// FormatEksekusiBerhasil menyusun pesan pengganti setelah entry dikonfirmasi.
//
// Pesannya menggantikan pesan setup yang lama, jadi tombolnya otomatis hilang
// dan tidak ada lagi tekanan yang bisa mengulang order.
func FormatEksekusiBerhasil(p EksekusiParams) Message {
	var b strings.Builder

	b.WriteString(judulPesan("✅", p.Symbol, p.Htf, p.Ltf, "Posisi terpasang"))
	b.WriteString("\n")
	b.WriteString(makeBox([][2]string{
		{"Arah", arahTampil(p.Direction)},
		{"Order", nomorOrder(p.Ticket)},
		{"Entry", formatPrice(p.EntryPrice)},
		{"Stop", formatPrice(p.StopLoss)},
		{"Target", formatPrice(p.TakeProfit)},
		{"Lot", lotTampil(p.LotSize)},
	}))
	b.WriteString("\nOrder sudah masuk ke broker. Bot akan mengabari saat posisi ditutup.\n")

	return Message{Text: b.String()}
}

// EksekusiParams memuat rincian posisi yang baru dibuka.
type EksekusiParams struct {
	Symbol     string
	Direction  string
	Htf        string
	Ltf        string
	Ticket     string
	EntryPrice float64
	StopLoss   float64
	TakeProfit float64
	LotSize    float64
}

// FormatEksekusiGagal menyusun pesan pengganti saat entry tidak jadi dijalankan.
//
// Alasan ditulis apa adanya dari Runner, karena trader perlu tahu apakah setup
// masih berlaku atau sudah lewat.
func FormatEksekusiGagal(symbol, htf, ltf, alasan string) Message {
	var b strings.Builder

	b.WriteString(judulPesan("⛔", symbol, htf, ltf, "Entry tidak dijalankan"))
	b.WriteString("\n")
	if alasan != "" {
		b.WriteString(Sanitize(alasan))
		b.WriteString("\n")
	}
	b.WriteString("\nTidak ada order yang terkirim.\n")

	return Message{Text: b.String()}
}

// FormatDilewatiMessage menyusun pesan pengganti setelah setup dilewati.
func FormatDilewatiMessage(symbol, htf, ltf string) Message {
	var b strings.Builder

	b.WriteString(judulPesan("🚫", symbol, htf, ltf, "Setup dilewati"))
	b.WriteString("\nSetup ini tidak akan dieksekusi. Kuota trading Anda tidak terpakai.\n")

	return Message{Text: b.String()}
}

// FormatSudahTidakBerlaku menandai pesan lama yang tombolnya harus dibuang.
//
// Dipakai saat setup berakhir sebelum sempat dieksekusi, supaya pesan lamanya
// tidak lagi menawarkan tombol yang tidak ada gunanya.
func FormatSudahTidakBerlaku(symbol, htf, ltf, sebab string) Message {
	var b strings.Builder

	b.WriteString(judulPesan("⌛", symbol, htf, ltf, "Sudah tidak berlaku"))
	b.WriteString("\n")
	if sebab != "" {
		b.WriteString(Sanitize(sebab))
		b.WriteString("\n")
	}

	return Message{Text: b.String()}
}

// FormatKedaluwarsaTautan dipakai saat tombol ditekan tetapi tautannya sudah mati.
func FormatKedaluwarsaTautan(symbol string) Message {
	return Message{Text: fmt.Sprintf(
		"⌛ %s\n\nTautan ini sudah tidak berlaku. Setup mungkin sudah lewat, sudah dieksekusi, atau sudah dibatalkan.\n",
		symbol,
	)}
}

// nomorOrder menampilkan nomor tiket, atau keterangan bila belum ada.
func nomorOrder(ticket string) string {
	if ticket == "" {
		return "menunggu broker"
	}
	return "#" + ticket
}
