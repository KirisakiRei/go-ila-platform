package telegram

import (
	"fmt"
	"strings"
	"time"
)

// BarisRiwayat adalah satu posisi pada daftar riwayat.
type BarisRiwayat struct {
	PositionID string
	Symbol     string
	Direction  string
	OutcomeR   float64
	ProfitLoss float64
	ClosedAt   time.Time
}

// RiwayatParams memuat satu halaman riwayat posisi.
type RiwayatParams struct {
	Baris        []BarisRiwayat
	Halaman      int
	TotalHalaman int
	TotalPosisi  int
}

// FormatRiwayatMessage menyusun satu halaman riwayat posisi.
//
// Lima posisi per halaman: cukup untuk melihat pola, tidak sampai harus
// menggulir jauh di layar ponsel.
func FormatRiwayatMessage(p RiwayatParams) Message {
	var b strings.Builder

	if p.TotalPosisi == 0 {
		b.WriteString("📋 Riwayat Posisi\n\nBelum ada posisi yang tercatat.\n")
		b.WriteString("Setiap entry yang diambil akan tampil di sini.\n")
		return Message{Text: b.String(), Buttons: tombolRiwayat(0, 0)}
	}

	fmt.Fprintf(&b, "📋 Riwayat Posisi\nHalaman %d dari %d · %d posisi\n", p.Halaman, p.TotalHalaman, p.TotalPosisi)

	hariTerakhir := ""
	for _, row := range p.Baris {
		hari := row.ClosedAt.In(time.FixedZone("WIB", 7*3600)).Format("02 Jan")
		if hari != hariTerakhir {
			fmt.Fprintf(&b, "\n%s\n", hari)
			hariTerakhir = hari
		}
		jam := row.ClosedAt.In(time.FixedZone("WIB", 7*3600)).Format("15:04")
		fmt.Fprintf(&b, "%s %s %s · %s · %s\n",
			simbolHasil(row.OutcomeR),
			arahTampil(row.Direction),
			row.Symbol,
			hasilR(row.OutcomeR),
			jam,
		)
	}

	return Message{Text: b.String(), Buttons: tombolRiwayat(p.Halaman, p.TotalHalaman)}
}

// tombolRiwayat menyusun navigasi halaman.
//
// Tombol yang tidak mungkin ditekan dikirim sebagai tombol tanpa aksi, supaya
// tata letaknya tidak bergeser antar halaman.
func tombolRiwayat(halaman, total int) [][]InlineButton {
	if total <= 1 {
		return [][]InlineButton{{{Text: LabelMenu, CallbackData: CallbackMenu}}}
	}

	sebelum := InlineButton{Text: "‹ Sebelumnya"}
	if halaman > 1 {
		sebelum.CallbackData = fmt.Sprintf("%s%d", CallbackRiwayat, (halaman-2)*RiwayatPerHalaman)
	}

	berikut := InlineButton{Text: "Berikutnya ›"}
	if halaman < total {
		berikut.CallbackData = fmt.Sprintf("%s%d", CallbackRiwayat, halaman*RiwayatPerHalaman)
	}

	return [][]InlineButton{
		{sebelum, {Text: fmt.Sprintf("%d/%d", halaman, total), CallbackData: CallbackMenu}, berikut},
		{{Text: LabelMenu, CallbackData: CallbackMenu}},
	}
}

// RiwayatPerHalaman adalah jumlah posisi pada satu halaman riwayat.
const RiwayatPerHalaman = 5

// HariParams memuat ringkasan satu hari dan bulan berjalan.
type HariParams struct {
	Hari          string
	TradeTerpakai int
	TradeMaks     int
	Win           int
	Loss          int
	ProfitLoss    float64
	PosisiJalan   int
	BulanTotal    int
	BulanWinRate  float64
	BulanNet      float64
	DashboardLink string
}

// FormatHariMessage menyusun ringkasan hari ini.
func FormatHariMessage(p HariParams) Message {
	var b strings.Builder

	b.WriteString("📈 Ringkasan Hari Ini")
	if p.Hari != "" {
		b.WriteString(" · " + p.Hari)
	}
	b.WriteString("\n\n")
	b.WriteString(daftar([][2]string{
		{"Trade", kuotaTampil(p.TradeTerpakai, p.TradeMaks)},
		{"Menang", itoaOpsional(p.Win)},
		{"Kalah", itoaOpsional(p.Loss)},
		{"Untung/Rugi", formatProfitHarian(p.ProfitLoss)},
		{"Posisi jalan", itoaOpsional(p.PosisiJalan)},
	}))
	if p.BulanTotal > 0 {
		b.WriteString("\nBulan ini\n")
		b.WriteString(daftar([][2]string{
			{"Total posisi", fmt.Sprintf("%d", p.BulanTotal)},
			{"Rasio menang", fmt.Sprintf("%s%%", formatFrom(p.BulanWinRate))},
			{"Untung/Rugi", formatMoney(p.BulanNet)},
		}))
	}

	var tombol [][]InlineButton
	tombol = append(tombol, []InlineButton{
		{Text: LabelRiwayat, CallbackData: CallbackRiwayat + "0"},
		{Text: LabelMenu, CallbackData: CallbackMenu},
	})
	if p.DashboardLink != "" {
		tombol = append(tombol, []InlineButton{{Text: "Buka di dashboard", URL: p.DashboardLink}})
	}

	return Message{Text: b.String(), Buttons: tombol}
}

// FormatPengaturanMessage menjelaskan apa yang bisa dan tidak bisa diatur
// dari Telegram.
//
// Kalimatnya sengaja menjelaskan alasannya, bukan sekadar melarang. Trader
// perlu tahu bahwa urusan akun tetap aman di tempat yang semestinya.
func FormatPengaturanMessage(dashboardLink string) Message {
	var b strings.Builder

	b.WriteString("⚙️ PENGATURAN\n\n")
	b.WriteString("Sebagian besar pengaturan tetap dikelola di website, karena menyangkut akun dan uang Anda.\n\n")
	b.WriteString("Yang bisa Anda atur di website\n")
	b.WriteString(daftar([][2]string{
		{"Akun broker", "kredensial dan koneksi"},
		{"Risiko", "per transaksi dan ukuran lot"},
		{"Pantauan", "pair dan timeframe"},
		{"Mode", "otomatis atau manual"},
		{"Langganan", "notifikasi dan paket"},
	}))
	b.WriteString("\nLewat Telegram Anda cukup memantau dan memutuskan entry.\n")

	var tombol [][]InlineButton
	if dashboardLink != "" {
		tombol = append(tombol, []InlineButton{{Text: "Buka Pengaturan", URL: dashboardLink}})
	}
	tombol = append(tombol, []InlineButton{{Text: LabelMenu, CallbackData: CallbackMenu}})
	return Message{Text: b.String(), Buttons: tombol}
}

// FormatMenuMessage menyusun halaman menu utama.
//
// Susunannya mengikuti kebiasaan bot yang sudah dikenal trader: identitas
// singkat di atas, lalu tombol besar satu per baris supaya mudah ditekan
// dengan jempol.
func FormatMenuMessage(nama, paket string) Message {
	var b strings.Builder

	b.WriteString("🤖 BOT Trading ILA\n\n")
	if nama != "" {
		b.WriteString("Halo, " + escapeHTML(nama) + " 👋\n")
	} else {
		b.WriteString("Halo 👋\n")
	}
	b.WriteString("Mau lihat apa hari ini?\n")

	if paket != "" {
		b.WriteString("\n")
		b.WriteString(daftar([][2]string{{"Paket", paket}}))
	}

	return Message{Text: b.String(), Buttons: tombolMenu()}
}

// tombolMenu menyusun tombol halaman menu, satu per baris.
func tombolMenu() [][]InlineButton {
	return [][]InlineButton{
		{{Text: LabelPantau, CallbackData: CallbackPantau}},
		{{Text: LabelRiwayat, CallbackData: CallbackRiwayat + "0"}},
		{{Text: LabelHariIni, CallbackData: CallbackHariIni}},
		{{Text: LabelPengaturan, CallbackData: CallbackPengaturan}},
		{{Text: "❓ Panduan Singkat", CallbackData: CallbackBantuan}},
	}
}

// FormatBantuanMessage menjelaskan cara pakai bot dengan singkat.
func FormatBantuanMessage() Message {
	var b strings.Builder

	b.WriteString("❓ PANDUAN SINGKAT\n\n")
	b.WriteString("Bot ini mengabari setup ILA dan menjalankan entry tanpa Anda harus membuka website.\n\n")
	b.WriteString("Cara pakai\n")
	b.WriteString(daftar([][2]string{
		{"Notifikasi", "pesan masuk saat setup siap dieksekusi"},
		{"Eksekusi", "tekan tombol di pesan untuk entry atau lewati"},
		{"Pantau", "posisi berjalan dan setup yang menunggu"},
		{"Riwayat", "posisi yang sudah tutup beserta hasilnya"},
		{"Pengaturan", "akun, risiko, dan mode tetap di website"},
	}))
	b.WriteString("\nTombol menu di bawah layar bisa disembunyikan lewat tanda panah, dan muncul lagi setiap kali Anda membuka menu.\n")

	return Message{Text: b.String(), Buttons: [][]InlineButton{
		{{Text: LabelMenu, CallbackData: CallbackMenu}},
	}}
}

// itoaOpsional menampilkan angka atau tanda hubung bila nol.
func itoaOpsional(v int) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d", v)
}
